package repository

import (
	manager "github.com/dictyBase/arangomanager"
	"github.com/dictyBase/go-genproto/dictybaseapis/content"
	"github.com/dictyBase/modware-content/internal/model"
)

type ContentRepository interface {
	GetContentBySlug(slug string) (*model.ContentDoc, error)
	GetContent(cid int64) (*model.ContentDoc, error)
	AddContent(cnt *content.NewContentAttributes) (*model.ContentDoc, error)
	EditContent(
		cid int64,
		cnt *content.ExistingContentAttributes,
	) (*model.ContentDoc, error)
	DeleteContent(cid int64) error
	Dbh() *manager.Database
}
