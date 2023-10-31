package arangodb

import (
	"encoding/json"
	"fmt"
	"testing"

	manager "github.com/dictyBase/arangomanager"
	"github.com/dictyBase/arangomanager/testarango"
	"github.com/dictyBase/go-genproto/dictybaseapis/content"
	"github.com/dictyBase/modware-content/internal/model"
	"github.com/dictyBase/modware-content/internal/repository"
	"github.com/stretchr/testify/require"
)

type ContentJSON struct {
	Paragraph string `json:"paragraph"`
	Text      string `json:"text"`
}

func NewStoreContent(name, namespace string) *content.NewContentAttributes {
	cdata, _ := json.Marshal(ContentJSON{
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

func setUp(t *testing.T) (*require.Assertions, repository.ContentRepository) {
	t.Helper()
	tra, err := testarango.NewTestArangoFromEnv(true)
	if err != nil {
		t.Fatalf("unable to construct new TestArango instance %s", err)
	}
	assert := require.New(t)
	repo, err := NewContentRepo(
		&manager.ConnectParams{
			User:     tra.User,
			Pass:     tra.Pass,
			Database: tra.Database,
			Host:     tra.Host,
			Port:     tra.Port,
			Istls:    false,
		}, manager.RandomString(16, 19),
	)
	assert.NoErrorf(
		err,
		"expect no error connecting to annotation repository, received %s",
		err,
	)

	return assert, repo
}

func tearDown(repo repository.ContentRepository) {
	_ = repo.Dbh().Drop()
}

func TestAddContent(t *testing.T) {
	t.Parallel()
	assert, repo := setUp(t)
	defer tearDown(repo)
	nct, err := repo.AddContent(NewStoreContent("catalog", "dsc"))
	assert.NoErrorf(err, "expect no error from creating content %s", err)
	assert.Equal(nct.Name, "catalog", "name should match")
	assert.Equal(nct.Namespace, "dsc", "namespace should match")
	assert.Equal(nct.Slug, "catalog-dsc", "slug should match")
}
