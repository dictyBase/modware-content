package testutils

import (
	"encoding/json"
	"fmt"

	"github.com/dictyBase/go-genproto/dictybaseapis/content"
	"github.com/dictyBase/modware-content/internal/model"
)

type ContentJSON struct {
	Paragraph string `json:"paragraph"`
	Text      string `json:"text"`
}

func NewStoreContent(name, namespace string) *content.NewContentAttributes {
	cdata, _ := json.Marshal(&ContentJSON{
		Paragraph: "paragraph",
		Text:      "text",
	})

	return &content.NewContentAttributes{
		Name:      name,
		Namespace: namespace,
		CreatedBy: "content@content.org",
		Content:   string(cdata),
		Slug:      model.Slugify(fmt.Sprintf("%s %s", name, namespace)),
	}
}

func ContentFromStore(jsctnt string) (*ContentJSON, error) {
	ctnt := &ContentJSON{}
	err := json.Unmarshal([]byte(jsctnt), ctnt)
	if err != nil {
		return ctnt, fmt.Errorf("error in unmarshing json %s", err)
	}

	return ctnt, nil
}
