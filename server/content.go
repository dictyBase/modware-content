package server

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/dictyBase/modware-content/message"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	dat "gopkg.in/mgutz/dat.v2/dat"
	runner "gopkg.in/mgutz/dat.v2/sqlx-runner"

	"github.com/dictyBase/apihelpers/aphgrpc"
	"github.com/dictyBase/go-genproto/dictybaseapis/api/jsonapi"
	"github.com/dictyBase/go-genproto/dictybaseapis/content"
	"github.com/dictyBase/go-genproto/dictybaseapis/pubsub"
	"github.com/golang/protobuf/ptypes/empty"
)

const (
	contentDbTable   = "content"
	namespaceDbTable = "namespace"
	prKeyCol         = "content_id"
)

var contentCols = []string{
	"content_id",
	"name",
	"slug",
	"created_by",
	"updated_by",
	"created_at",
	"updated_at",
	"content",
	"namespace_id",
}

var noncharReg = regexp.MustCompile("[^a-z0-9]+")

type dbContent struct {
	ContentId   int64          `db:"content_id"`
	Name        dat.NullString `db:"name"`
	Slug        dat.NullString `db:"slug"`
	CreatedBy   dat.NullInt64  `db:"created_by"`
	UpdatedBy   dat.NullInt64  `db:"updated_by"`
	CreatedAt   dat.NullTime   `db:"created_at"`
	UpdatedAt   dat.NullTime   `db:"updated_at"`
	Namespace   dat.NullString `db:"namespace"`
	NamespaceId int64          `db:"namespace_id"`
	Content     dat.JSON       `db:"content"`
}

type dbContentCore struct {
	ContentId   int64          `db:"content_id"`
	Name        dat.NullString `db:"name"`
	Slug        dat.NullString `db:"slug"`
	CreatedBy   dat.NullInt64  `db:"created_by"`
	UpdatedBy   dat.NullInt64  `db:"updated_by"`
	CreatedAt   dat.NullTime   `db:"created_at"`
	UpdatedAt   dat.NullTime   `db:"updated_at"`
	NamespaceId int64          `db:"namespace_id"`
	Content     dat.JSON       `db:"content"`
}

type ContentService struct {
	*aphgrpc.Service
	request message.Request
}

func defaultOptions() *aphgrpc.ServiceOptions {
	return &aphgrpc.ServiceOptions{
		PathPrefix: "contents",
		Resource:   "contents",
		Topics: map[string]string{
			"userExists": "UserService.Exist",
		},
	}
}

func NewContentService(dbh *runner.DB, req message.Request, options ...aphgrpc.Option) *ContentService {
	so := defaultOptions()
	for _, optfn := range options {
		optfn(so)
	}
	srv := &aphgrpc.Service{Dbh: dbh}
	aphgrpc.AssignFieldsToStructs(so, srv)
	return &ContentService{
		Service: srv,
		request: req,
	}
}

func (s *ContentService) Healthz(ctx context.Context, r *jsonapi.HealthzIdRequest) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}

func (s *ContentService) GetContentBySlug(ctx context.Context, r *content.ContentRequest) (*content.Content, error) {
	ct, err := s.getResourceBySlug(r.Slug)
	if err != nil {
		return &content.Content{}, aphgrpc.HandleError(ctx, err)
	}
	return ct, nil
}

func (s *ContentService) GetContent(ctx context.Context, r *content.ContentIdRequest) (*content.Content, error) {
	ct, err := s.getResource(r.Id)
	if err != nil {
		return &content.Content{}, aphgrpc.HandleError(ctx, err)
	}
	return ct, nil

}

func (s *ContentService) StoreContent(ctx context.Context, r *content.StoreContentRequest) (*content.Content, error) {
	emptyCt := new(content.Content)
	if err := r.Data.Attributes.Validate(); err != nil {
		grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseInsert)
		return emptyCt, status.Error(codes.InvalidArgument, err.Error())
	}
	// Check for presence of user
	// by messaging through user service
	reply, err := s.request.UserRequestWithContext(
		context.Background(),
		s.Topics["userExists"],
		&pubsub.IdRequest{Id: r.Data.Attributes.CreatedBy},
	)
	if err != nil {
		return emptyCt, aphgrpc.HandleGenericError(ctx, err)
	}
	if reply.Status != nil {
		return emptyCt, aphgrpc.HandleMessagingError(ctx, reply.Status)
	}
	if !reply.Exist {
		return emptyCt, aphgrpc.HandleNotFoundError(
			ctx,
			fmt.Errorf("user id %d not found", r.Data.Attributes.CreatedBy),
		)
	}
	tx, _ := s.Dbh.Begin()
	defer tx.AutoRollback()
	// Check if namespace exists
	var namespaceId int64
	err = tx.Select("namespace_id").From(namespaceDbTable).
		Where("name = $1", r.Data.Attributes.Namespace).QueryScalar(&namespaceId)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			err := tx.InsertInto(namespaceDbTable).
				Columns("name").Values(r.Data.Attributes.Namespace).
				Returning("namespace_id").QueryScalar(&namespaceId)
			if err != nil {
				return emptyCt, aphgrpc.HandleInsertError(ctx, err)
			}
		} else {
			return emptyCt, aphgrpc.HandleUpdateError(ctx, err)
		}
	}

	var ctId int64
	var at dat.NullTime
	dbct := s.createAttrTodbContentCore(r.Data.Attributes)
	dbct.NamespaceId = namespaceId
	ctcolumns := aphgrpc.GetDefinedTags(dbct, "db")
	err = tx.InsertInto(contentDbTable).
		Columns(ctcolumns...).
		Record(dbct).
		Returning(prKeyCol, "created_at").
		QueryScalar(&ctId, &at)
	if err != nil {
		return &content.Content{}, aphgrpc.HandleInsertError(ctx, err)
	}

	tx.Commit()
	grpc.SetTrailer(ctx, metadata.Pairs("method", "POST"))
	attr := s.dbCoreToResourceAttributes(dbct)
	attr.CreatedAt = aphgrpc.NullToTime(at)
	attr.UpdatedAt = attr.CreatedAt
	attr.Namespace = r.Data.Attributes.Namespace
	return s.buildResource(context.TODO(), ctId, attr), nil
}

func (s *ContentService) UpdateContent(ctx context.Context, r *content.UpdateContentRequest) (*content.Content, error) {
	result, err := s.existsResource(r.Id)
	if err != nil {
		return &content.Content{}, aphgrpc.HandleError(ctx, err)
	}
	if !result {
		grpc.SetTrailer(ctx, aphgrpc.ErrNotFound)
		return &content.Content{}, status.Error(codes.NotFound, fmt.Sprintf("id %d not found", r.Id))
	}
	if err := r.Data.Attributes.Validate(); err != nil {
		grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseUpdate)
		return &content.Content{}, status.Error(codes.InvalidArgument, err.Error())
	}
	dbct := &dbContentCore{}
	tx, _ := s.Dbh.Begin()
	defer tx.AutoRollback()
	err = tx.Update(contentDbTable).
		Set("updated_by", r.Data.Attributes.UpdatedBy).
		Set("content", r.Data.Attributes.Content).
		Where(prKeyCol+"= $1", r.Id).
		Returning(contentCols...).
		QueryStruct(dbct)
	if err != nil {
		grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseUpdate)
		return &content.Content{}, status.Error(codes.Internal, err.Error())
	}
	var namespace string
	err = tx.Select("name").From(namespaceDbTable).
		Where("namespace_id = $1", dbct.NamespaceId).QueryScalar(&namespace)
	if err != nil {
		grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseUpdate)
		return &content.Content{}, status.Error(codes.Internal, err.Error())
	}
	tx.Commit()
	attr := s.dbCoreToResourceAttributes(dbct)
	attr.Namespace = namespace
	return s.buildResource(context.TODO(), dbct.ContentId, attr), nil
}

func (s *ContentService) DeleteContent(ctx context.Context, r *content.ContentIdRequest) (*empty.Empty, error) {
	result, err := s.existsResource(r.Id)
	if err != nil {
		return &empty.Empty{}, aphgrpc.HandleError(ctx, err)
	}
	if !result {
		grpc.SetTrailer(ctx, aphgrpc.ErrNotFound)
		return &empty.Empty{}, status.Error(codes.NotFound, fmt.Sprintf("id %d not found", r.Id))
	}
	tx, _ := s.Dbh.Begin()
	defer tx.AutoRollback()
	_, err = tx.DeleteFrom(contentDbTable).Where(prKeyCol+" = $1", r.Id).Exec()
	if err != nil {
		grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseDelete)
		return &empty.Empty{}, status.Error(codes.Internal, err.Error())
	}
	tx.Commit()
	return &empty.Empty{}, nil
}

func (s *ContentService) existsResource(id int64) (bool, error) {
	r, err := s.Dbh.Select(
		fmt.Sprintf("%s.%s", contentDbTable, prKeyCol),
	).From(
		contentDbTable,
	).Where(
		fmt.Sprintf("%s.%s = $1", contentDbTable, prKeyCol),
		id,
	).Exec()
	if err != nil {
		return false, err
	}
	if r.RowsAffected != 1 {
		return false, nil
	}
	return true, nil
}

// -- Functions that queries the storage and generates resource object
func (s *ContentService) getResource(id int64) (*content.Content, error) {
	dct := &dbContent{}
	err := s.Dbh.Select(
		fmt.Sprintf("%s.*", contentDbTable),
		fmt.Sprintf("%s.name namespace", namespaceDbTable),
	).From(
		fmt.Sprintf(
			"%s JOIN %s ON %s.namespace_id = %s.namespace_id",
			contentDbTable, namespaceDbTable, contentDbTable, namespaceDbTable,
		),
	).Where(
		fmt.Sprintf("%s.%s = $1", contentDbTable, prKeyCol),
		id,
	).QueryStruct(dct)
	if err != nil {
		return &content.Content{}, err
	}
	return s.buildResource(context.TODO(), id, s.dbToResourceAttributes(dct)), nil
}

func (s *ContentService) getResourceBySlug(slug string) (*content.Content, error) {
	dct := &dbContent{}
	err := s.Dbh.Select(
		fmt.Sprintf("%s.*", contentDbTable),
		fmt.Sprintf("%s.name namespace", namespaceDbTable),
	).From(
		fmt.Sprintf(
			"%s JOIN %s ON %s.namespace_id = %s.namespace_id",
			contentDbTable, namespaceDbTable, contentDbTable, namespaceDbTable,
		),
	).Where(
		fmt.Sprintf("%s.slug = $1", contentDbTable),
		slug,
	).QueryStruct(dct)
	if err != nil {
		return &content.Content{}, err
	}
	return s.buildResource(context.TODO(), dct.ContentId, s.dbToResourceAttributes(dct)), nil
}

// -- Functions that builds up the various parts of the final content resource objects
func (s *ContentService) buildResourceData(ctx context.Context, id int64, attr *content.ContentAttributes) *content.ContentData {
	return &content.ContentData{
		Attributes: attr,
		Id:         id,
		Type:       s.GetResourceName(),
		Links: &jsonapi.Links{
			Self: s.GenResourceSelfLink(ctx, id),
		},
	}
}

func (s *ContentService) buildResource(ctx context.Context, id int64, attr *content.ContentAttributes) *content.Content {
	return &content.Content{
		Data: s.buildResourceData(ctx, id, attr),
		Links: &jsonapi.Links{
			Self: s.GenResourceSelfLink(ctx, id),
		},
	}
}

// Functions that generates resource objects or parts of it from database mapped objects
func (s *ContentService) dbToResourceAttributes(dct *dbContent) *content.ContentAttributes {
	ct, _ := dct.Content.Interpolate()
	return &content.ContentAttributes{
		Name:      aphgrpc.NullToString(dct.Name),
		Slug:      aphgrpc.NullToString(dct.Slug),
		CreatedBy: aphgrpc.NullToInt64(dct.CreatedBy),
		UpdatedBy: aphgrpc.NullToInt64(dct.UpdatedBy),
		Namespace: aphgrpc.NullToString(dct.Namespace),
		CreatedAt: aphgrpc.NullToTime(dct.CreatedAt),
		UpdatedAt: aphgrpc.NullToTime(dct.UpdatedAt),
		Content:   ct,
	}
}

func (s *ContentService) dbCoreToResourceAttributes(dct *dbContentCore) *content.ContentAttributes {
	ct, _ := dct.Content.Interpolate()
	return &content.ContentAttributes{
		Name:      aphgrpc.NullToString(dct.Name),
		Slug:      aphgrpc.NullToString(dct.Slug),
		CreatedBy: aphgrpc.NullToInt64(dct.CreatedBy),
		UpdatedBy: aphgrpc.NullToInt64(dct.UpdatedBy),
		CreatedAt: aphgrpc.NullToTime(dct.CreatedAt),
		UpdatedAt: aphgrpc.NullToTime(dct.UpdatedAt),
		Content:   ct,
	}
}

// Functions that generates database mapped objects from resource objects
func (s *ContentService) attrTodbContent(attr *content.ContentAttributes) *dbContent {
	return &dbContent{
		Name:      dat.NullStringFrom(attr.Name),
		Slug:      dat.NullStringFrom(attr.Slug),
		CreatedBy: dat.NullInt64From(attr.CreatedBy),
		UpdatedBy: dat.NullInt64From(attr.UpdatedBy),
		CreatedAt: dat.NullTimeFrom(aphgrpc.ProtoTimeStamp(attr.CreatedAt)),
		UpdatedAt: dat.NullTimeFrom(aphgrpc.ProtoTimeStamp(attr.UpdatedAt)),
		Namespace: dat.NullStringFrom(attr.Namespace),
		Content:   dat.JSONFromString(attr.Content),
	}
}

func (s *ContentService) attrTodbContentCore(attr *content.ContentAttributes) *dbContentCore {
	return &dbContentCore{
		Name:      dat.NullStringFrom(attr.Name),
		Slug:      dat.NullStringFrom(attr.Slug),
		CreatedBy: dat.NullInt64From(attr.CreatedBy),
		UpdatedBy: dat.NullInt64From(attr.UpdatedBy),
		CreatedAt: dat.NullTimeFrom(aphgrpc.ProtoTimeStamp(attr.CreatedAt)),
		UpdatedAt: dat.NullTimeFrom(aphgrpc.ProtoTimeStamp(attr.UpdatedAt)),
		Content:   dat.JSONFromString(attr.Content),
	}
}

func (s *ContentService) createAttrTodbContentCore(attr *content.NewContentAttributes) *dbContentCore {
	return &dbContentCore{
		Name:      dat.NullStringFrom(attr.Name),
		CreatedBy: dat.NullInt64From(attr.CreatedBy),
		UpdatedBy: dat.NullInt64From(attr.CreatedBy),
		Content:   dat.JSONFromString(attr.Content),
		Slug: dat.NullStringFrom(
			slug(
				fmt.Sprintf("%s %s", attr.Namespace, attr.Name),
			),
		),
	}
}

func slug(s string) string {
	return strings.Trim(noncharReg.ReplaceAllString(strings.ToLower(s), "-"), "-")
}
