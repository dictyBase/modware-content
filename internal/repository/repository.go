package repository

import (
	"github.com/dictyBase/go-genproto/dictybaseapis/content"
	"github.com/dictyBase/modware-content/internal/model"
)

type ContentRepository interface {
	GetContentBySlug(slug string) (*model.ContentDoc, error)
	GetContent(id string) (*model.ContentDoc, error)
	AddContent(cnt *content.NewContentAttributes) (*model.ContentDoc, error)
	EditContent(
		cnt *content.ExistingContentAttributes,
	) (*model.ContentDoc, error)
	DeleteContent(id string) error
}
