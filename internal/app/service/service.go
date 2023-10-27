package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/dictyBase/apihelpers/aphgrpc"
	"github.com/dictyBase/go-genproto/dictybaseapis/api/jsonapi"
	"github.com/dictyBase/go-genproto/dictybaseapis/content"
	"github.com/dictyBase/modware-content/internal/message"
	"github.com/dictyBase/modware-content/internal/repository"
	"github.com/go-playground/validator/v10"
	"github.com/golang/protobuf/ptypes/empty"
)

var noncharReg = regexp.MustCompile("[^a-z0-9]+")

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
	Publisher  message.Publisher         `validate:"required"`
	Options    []aphgrpc.Option          `validate:"required"`
	Group      string                    `validate:"required"`
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

func (s *ContentService) Healthz(
	ctx context.Context,
	r *jsonapi.HealthzIdRequest,
) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}

func (s *ContentService) GetContentBySlug(
	ctx context.Context,
	r *content.ContentRequest,
) (*content.Content, error) {
	return &content.Content{}, nil
}

func (s *ContentService) GetContent(
	ctx context.Context,
	r *content.ContentIdRequest,
) (*content.Content, error) {
	return &content.Content{}, nil
}

func (s *ContentService) StoreContent(
	ctx context.Context,
	req *content.StoreContentRequest,
) (*content.Content, error) {
	return &content.Content{}, nil
}

func (s *ContentService) UpdateContent(
	ctx context.Context,
	req *content.UpdateContentRequest,
) (*content.Content, error) {
	return &content.Content{}, nil

}

func (s *ContentService) DeleteContent(
	ctx context.Context,
	req *content.ContentIdRequest,
) (*empty.Empty, error) {
	return &content.Content{}, nil
}

// -- Functions that builds up the various parts of the final content resource
// objects.
func (s *ContentService) buildResourceData(
	ctx context.Context,
	idn int64,
	attr *content.ContentAttributes,
) *content.ContentData {
	return &content.ContentData{
		Attributes: attr,
		Id:         idn,
		Type:       s.GetResourceName(),
		Links: &jsonapi.Links{
			Self: s.GenResourceSelfLink(ctx, idn),
		},
	}
}

func (s *ContentService) buildResource(
	ctx context.Context,
	idn int64,
	attr *content.ContentAttributes,
) *content.Content {
	return &content.Content{
		Data: s.buildResourceData(ctx, idn, attr),
		Links: &jsonapi.Links{
			Self: s.GenResourceSelfLink(ctx, idn),
		},
	}
}

func slug(s string) string {
	return strings.Trim(
		noncharReg.ReplaceAllString(strings.ToLower(s), "-"),
		"-",
	)
}
