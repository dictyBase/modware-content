package service

import (
	"context"
	"fmt"
	"strconv"

	"github.com/dictyBase/aphgrpc"
	"github.com/dictyBase/go-genproto/dictybaseapis/api/jsonapi"
	"github.com/dictyBase/go-genproto/dictybaseapis/content"
	"github.com/dictyBase/modware-content/internal/message"
	"github.com/dictyBase/modware-content/internal/model"
	"github.com/dictyBase/modware-content/internal/repository"
	"github.com/go-playground/validator/v10"
	"github.com/golang/protobuf/ptypes/empty"
)

func defaultOptions() *aphgrpc.ServiceOptions {
	return &aphgrpc.ServiceOptions{
		Resource: "contents",
	}
}

type ContentService struct {
	*aphgrpc.Service
	repo      repository.ContentRepository
	publisher message.Publisher
	group     string
	content.UnimplementedContentServiceServer
}

// ServiceParams are the attributes that are required for creating new ContentService.
type Params struct {
	Repository repository.ContentRepository `validate:"required"`
	Publisher  message.Publisher            `validate:"required"`
	Options    []aphgrpc.Option             `validate:"required"`
	Group      string                       `validate:"required"`
}

func NewContentService(srvP *Params) (*ContentService, error) {
	if err := validator.New().Struct(srvP); err != nil {
		return &ContentService{}, fmt.Errorf(
			"error in validating struct %s",
			err,
		)
	}
	so := defaultOptions()
	for _, optfn := range srvP.Options {
		optfn(so)
	}
	srv := &aphgrpc.Service{}
	aphgrpc.AssignFieldsToStructs(so, srv)

	return &ContentService{
		Service:   srv,
		repo:      srvP.Repository,
		publisher: srvP.Publisher,
		group:     srvP.Group,
	}, nil
}

func (srv *ContentService) Healthz(
	ctx context.Context,
	rdr *jsonapi.HealthzIdRequest,
) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}

func (srv *ContentService) GetContentBySlug(
	ctx context.Context,
	rdr *content.ContentRequest,
) (*content.Content, error) {
	ctnt := &content.Content{}
	if err := rdr.Validate(); err != nil {
		return ctnt, aphgrpc.HandleInvalidParamError(ctx, err)
	}
	mcont, err := srv.repo.GetContentBySlug(rdr.Slug)
	if err != nil {
		return ctnt, aphgrpc.HandleGetError(ctx, err)
	}
	if mcont.NotFound {
		return ctnt, aphgrpc.HandleNotFoundError(ctx, err)
	}
	cid, _ := strconv.ParseInt(mcont.Key, 10, 64)

	return srv.buildContent(cid, mcont), nil
}

func (srv *ContentService) GetContent(
	ctx context.Context,
	rdr *content.ContentIdRequest,
) (*content.Content, error) {
	ctnt := &content.Content{}
	if err := rdr.Validate(); err != nil {
		return ctnt, aphgrpc.HandleInvalidParamError(ctx, err)
	}
	mcont, err := srv.repo.GetContent(rdr.Id)
	if err != nil {
		return ctnt, aphgrpc.HandleGetError(ctx, err)
	}
	if mcont.NotFound {
		return ctnt, aphgrpc.HandleNotFoundError(ctx, err)
	}
	cid, _ := strconv.ParseInt(mcont.Key, 10, 64)

	return srv.buildContent(cid, mcont), nil
}

func (srv *ContentService) buildContent(
	cid int64,
	mcont *model.ContentDoc,
) *content.Content {
	return &content.Content{
		Data: &content.ContentData{
			Type: srv.GetResourceName(),
			Id:   cid,
			Attributes: &content.ContentAttributes{
				Name:      mcont.Name,
				Namespace: mcont.Namespace,
				Slug:      mcont.Slug,
				CreatedBy: mcont.CreatedBy,
				UpdatedBy: mcont.UpdatedBy,
				CreatedAt: aphgrpc.TimestampProto(mcont.CreatedOn),
				UpdatedAt: aphgrpc.TimestampProto(mcont.UpdatedOn),
				Content:   mcont.Content,
			},
		}}
}

func (srv *ContentService) StoreContent(
	ctx context.Context,
	req *content.StoreContentRequest,
) (*content.Content, error) {
	ctnt := &content.Content{}
	if err := req.Validate(); err != nil {
		return ctnt, aphgrpc.HandleInvalidParamError(ctx, err)
	}
	mcont, err := srv.repo.AddContent(req.Data.Attributes)
	if err != nil {
		return ctnt, aphgrpc.HandleGetError(ctx, err)
	}
	cid, _ := strconv.ParseInt(mcont.Key, 10, 64)

	return srv.buildContent(cid, mcont), nil
}

func (srv *ContentService) UpdateContent(
	ctx context.Context,
	req *content.UpdateContentRequest,
) (*content.Content, error) {
	return &content.Content{}, nil
}

func (srv *ContentService) DeleteContent(
	ctx context.Context,
	req *content.ContentIdRequest,
) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}
