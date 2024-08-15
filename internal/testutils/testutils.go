package testutils

import (
	"encoding/json"
	"fmt"

	"github.com/dictyBase/go-genproto/dictybaseapis/content"
	"github.com/dictyBase/modware-content/internal/model"
)

type TestCaseWithFilterandCursor struct {
	Name                    string
	Filter                  string
	InitialLimit            int64
	SubsequentLimit         int64
	ExpectedName            string
	ExpectedNS              string
	ExpectedInitialCount    int
	ExpectedSubsequentCount int
}

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

func CreateTestCases() []TestCaseWithFilterandCursor {
	return []TestCaseWithFilterandCursor{
		{
			Name:                    "Filter by namespace",
			Filter:                  `FILTER cnt.namespace == "hogwarts"`,
			InitialLimit:            5,
			SubsequentLimit:         5,
			ExpectedName:            "wand",
			ExpectedNS:              "hogwarts",
			ExpectedInitialCount:    6,
			ExpectedSubsequentCount: 5,
		},
		{
			Name:                    "Filter by slug substring",
			Filter:                  `FILTER cnt.slug =~ "potion"`,
			InitialLimit:            7,
			SubsequentLimit:         7,
			ExpectedName:            "potion",
			ExpectedNS:              "hogsmeade",
			ExpectedInitialCount:    8,
			ExpectedSubsequentCount: 5,
		},
		{
			Name:                    "Combination filter",
			Filter:                  `FILTER cnt.name =~ "wand-" AND cnt.namespace =~ "hog"`,
			InitialLimit:            6,
			SubsequentLimit:         6,
			ExpectedName:            "wand",
			ExpectedNS:              "hogwarts",
			ExpectedInitialCount:    7,
			ExpectedSubsequentCount: 4,
		},
	}
}
