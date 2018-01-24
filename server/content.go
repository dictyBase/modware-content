package server

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	dat "gopkg.in/mgutz/dat.v1"

	"gopkg.in/mgutz/dat.v1/sqlx-runner"

	"github.com/dictyBase/apihelpers/aphgrpc"
	"github.com/dictyBase/go-genproto/dictybaseapis/api/jsonapi"
	"github.com/dictyBase/go-genproto/dictybaseapis/content"
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
	ContentId int64          `db:"content_id"`
	Name      dat.NullString `db:"name"`
	Slug      dat.NullString `db:"slug"`
	CreatedBy dat.NullInt64  `db:"created_by"`
	UpdatedBy dat.NullInt64  `db:"updated_by"`
	CreatedAt dat.NullTime   `db:"created_at"`
	UpdatedAt dat.NullTime   `db:"updated_at"`
	Namespace dat.NullString `db:"namespace"`
	Content   string         `db:"content"`
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
	Content     string         `db:"content"`
}

type ContentService struct {
	*aphgrpc.Service
}

func NewContentService(dbh *runner.DB, pathPrefix string) *ContentService {
	return &ContentService{
		&aphgrpc.Service{
			Resource:   "contents",
			Dbh:        dbh,
			PathPrefix: pathPrefix,
		},
	}
}

func (s *ContentService) GetContent(ctx context.Context, r content.ContentRequest) (*content.Content, error) {
	s.SetBaseURL(ctx)
	ct, err := s.getResourceBySlug(r.Slug)
	if err != nil {
		return &content.Content{}, aphgrpc.HandleError(ctx, err)
	}
	return ct, nil
}

func (s *ContentService) GetContentById(ctx context.Context, r content.ContentIdRequest) (*content.Content, error) {
	s.SetBaseURL(ctx)
	ct, err := s.getResource(r.Id)
	if err != nil {
		return &content.Content{}, aphgrpc.HandleError(ctx, err)
	}
	return ct, nil

}

func (s *ContentService) StoreContent(ctx context.Context, r content.StoreContentRequest) (*content.Content, error) {
	if err := r.Data.Attributes.Validate(); err != nil {
		grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseInsert)
		return &content.Content{}, status.Error(codes.InvalidArgument, err.Error())
	}
	var namespaceId int64
	tx, _ := s.Dbh.Begin()
	defer tx.AutoRollback()
	err := tx.InsertInto(namespaceDbTable).
		Columns("name").Values(r.Data.Attributes.Namespace).
		Returning("namespace_id").QueryScalar(&namespaceId)
	if err != nil {
		grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseInsert)
		return &content.Content{}, status.Error(codes.Internal, err.Error())
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
		grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseInsert)
		return &content.Content{}, status.Error(codes.Internal, err.Error())
	}

	tx.Commit()
	s.SetBaseURL(ctx)
	grpc.SetTrailer(ctx, metadata.Pairs("method", "POST"))
	attr := s.dbCoreToResourceAttributes(dbct)
	attr.CreatedAt = aphgrpc.NullToTime(at)
	attr.UpdatedAt = attr.CreatedAt
	attr.Namespace = r.Data.Attributes.Namespace
	return s.buildResource(ctId, attr), nil
}

func (s *ContentService) UpdateContent(ctx context.Context, r content.UpdateContentRequest) (*content.Content, error) {
	if err := s.existsResource(r.Id); err != nil {
		return &content.Content{}, aphgrpc.HandleError(ctx, err)
	}
	if err := r.Data.Attributes.Validate(); err != nil {
		grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseUpdate)
		return &content.Content{}, status.Error(codes.InvalidArgument, err.Error())
	}
	dbct := &dbContentCore{}
	tx, _ := s.Dbh.Begin()
	tx.AutoRollback()
	err := tx.Update(contentDbTable).
		SetMap(
			map[string]interface{}{
				"updated_by": r.Data.Attributes.UpdatedBy,
				"content":    r.Data.Attributes.Content,
			},
		).
		Where(prKeyCol+" = $1", r.Id).
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
	s.SetBaseURL(ctx)
	attr := s.dbCoreToResourceAttributes(dbct)
	attr.Namespace = namespace
	return s.buildResource(dbct.ContentId, attr), nil
}

func (s *ContentService) DeleteContent(ctx context.Context, r content.ContentIdRequest) (*empty.Empty, error) {
	if err := s.existsResource(r.Id); err != nil {
		return &empty.Empty{}, aphgrpc.HandleError(ctx, err)
	}
	tx, _ := s.Dbh.Begin()
	tx.AutoRollback()
	_, err := tx.DeleteFrom(contentDbTable).Where(prKeyCol+" = $1", r.Id).Exec()
	if err != nil {
		grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseDelete)
		return &empty.Empty{}, status.Error(codes.Internal, err.Error())
	}
	tx.Commit()
	return &empty.Empty{}, nil
}

func (s *ContentService) existsResource(id int64) error {
	_, err := s.Dbh.Select(
		fmt.Sprintf("%s.%s", contentDbTable, prKeyCol),
	).From(
		contentDbTable,
	).Where(
		fmt.Sprintf("%s.%s = $1", contentDbTable, prKeyCol),
		id,
	).Exec()
	return err
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
	return s.buildResource(id, s.dbToResourceAttributes(dct)), nil
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
	return s.buildResource(dct.ContentId, s.dbToResourceAttributes(dct)), nil
}

// -- Functions that builds up the various parts of the final user resource objects
func (s *ContentService) buildResourceData(id int64, attr *content.ContentAttributes) *content.ContentData {
	return &content.ContentData{
		Attributes: attr,
		Id:         id,
		Links: &jsonapi.Links{
			Self: s.GenResourceSelfLink(id),
		},
		Type: s.GetResourceName(),
	}
}

func (s *ContentService) buildResource(id int64, attr *content.ContentAttributes) *content.Content {
	return &content.Content{
		Data: s.buildResourceData(id, attr),
		Links: &jsonapi.Links{
			Self: s.GenResourceSelfLink(id),
		},
	}
}

// Functions that generates resource objects or parts of it from database mapped objects
func (s *ContentService) dbToResourceAttributes(dct *dbContent) *content.ContentAttributes {
	return &content.ContentAttributes{
		Name:      aphgrpc.NullToString(dct.Name),
		Slug:      aphgrpc.NullToString(dct.Slug),
		CreatedBy: aphgrpc.NullToInt64(dct.CreatedBy),
		UpdatedBy: aphgrpc.NullToInt64(dct.UpdatedBy),
		Namespace: aphgrpc.NullToString(dct.Namespace),
		CreatedAt: aphgrpc.NullToTime(dct.CreatedAt),
		UpdatedAt: aphgrpc.NullToTime(dct.UpdatedAt),
		Content:   dct.Content,
	}
}

func (s *ContentService) dbCoreToResourceAttributes(dct *dbContentCore) *content.ContentAttributes {
	return &content.ContentAttributes{
		Name:      aphgrpc.NullToString(dct.Name),
		Slug:      aphgrpc.NullToString(dct.Slug),
		CreatedBy: aphgrpc.NullToInt64(dct.CreatedBy),
		UpdatedBy: aphgrpc.NullToInt64(dct.UpdatedBy),
		CreatedAt: aphgrpc.NullToTime(dct.CreatedAt),
		UpdatedAt: aphgrpc.NullToTime(dct.UpdatedAt),
		Content:   dct.Content,
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
		Content:   attr.Content,
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
		Content:   attr.Content,
	}
}

func (s *ContentService) createAttrTodbContentCore(attr *content.NewContentAttributes) *dbContentCore {
	return &dbContentCore{
		Name:      dat.NullStringFrom(attr.Name),
		CreatedBy: dat.NullInt64From(attr.CreatedBy),
		UpdatedBy: dat.NullInt64From(attr.CreatedBy),
		Content:   attr.Content,
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
