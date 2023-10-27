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
	NotFound  bool
}

func Schema() []byte {
	return []byte(`{
		  "type": "object",
		  "properties": {
		    "name": {"type": "string"},
		    "namespace": {"type": "string"},
		    "slug": {"type": "string"},
		    "content": {"type": "string"},
		    "created_by": {"type": "string", "format": "email"},
		    "updated_by": {"type": "string", "format": "email"},
	 	    "created_on": {"type": "string", "format": "date-time"},
	 	    "updated_on": {"type": "string", "format": "date-time"}
		  },
		  "required": [
			"name", 
			"namespace", 
			"slug", 
			"content", 
			"created_by",
			"created_on"
		   ]
		}
	`)
}
