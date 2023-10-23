package model

import (
	"time"

	driver "github.com/arangodb/go-driver"
)

type ContentDoc struct {
	driver.DocumentMeta
	Name      string    `json:"name"       validate:"required"`
	Slug      string    `json:"slug"       validate:"required"`
	Namespace string    `json:"namespace"  validate:"required"`
	CreatedBy string    `json:"created_by" validate:"email"`
	UpdatedBy string    `json:"updated_by" validate:"required,email"`
	Content   string    `json:"content"    validate:"required"`
	CreatedOn time.Time `json:"created_on"`
	UpdatedOn time.Time `json:"updated_on"`
}
