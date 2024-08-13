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
	ListContents(
		cursor int64,
		limit int64,
		filter string,
	) ([]*model.ContentDoc, error)
	Dbh() *manager.Database
}

type ContentListNotFoundError struct{}

func (al *ContentListNotFoundError) Error() string {
	return "annotation list not found"
}

func IsContentListNotFound(err error) bool {
	if _, ok := err.(*ContentListNotFoundError); ok {
		return true
	}

	return false
}
