package arangodb

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"

	manager "github.com/dictyBase/arangomanager"
	"github.com/dictyBase/arangomanager/testarango"
	"github.com/dictyBase/go-genproto/dictybaseapis/content"
	"github.com/dictyBase/modware-content/internal/model"
	"github.com/dictyBase/modware-content/internal/repository"
	"github.com/dictyBase/modware-content/internal/testutils"
	"github.com/stretchr/testify/require"
)

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
	nct, err := repo.AddContent(testutils.NewStoreContent("catalog", "dsc"))
	assert.NoErrorf(err, "expect no error from creating content %s", err)
	assert.Equal(nct.Name, "catalog", "name should match")
	assert.Equal(nct.Namespace, "dsc", "namespace should match")
	assert.Equal(nct.Slug, "catalog-dsc", "slug should match")
	assert.Equal(
		nct.CreatedBy,
		"content@content.org",
		"should match created_by",
	)
	assert.True(
		nct.CreatedOn.Equal(nct.UpdatedOn),
		"created_on should match updated_on",
	)
	assert.True(
		nct.CreatedOn.Before(time.Now()),
		"should have created before the current time",
	)
	ctnt, err := testutils.ContentFromStore(nct.Content)
	assert.NoError(err, "should not have any error with json unmarshaling")
	assert.Equal(
		ctnt,
		&testutils.ContentJSON{Paragraph: "paragraph", Text: "text"},
		"should match the content",
	)
}

func TestGetContentBySlug(t *testing.T) {
	t.Parallel()
	assert, repo := setUp(t)
	defer tearDown(repo)
	nct, err := repo.AddContent(testutils.NewStoreContent("catalog", "dsc"))
	assert.NoErrorf(err, "expect no error from creating content %s", err)
	sct, err := repo.GetContentBySlug(nct.Slug)
	assert.NoErrorf(err, "expect no error from getting content by slug %s", err)
	testContentProperties(assert, sct, nct)
}

func TestGetContent(t *testing.T) {
	t.Parallel()
	assert, repo := setUp(t)
	defer tearDown(repo)
	nct, err := repo.AddContent(testutils.NewStoreContent("catalog", "dsc"))
	assert.NoErrorf(err, "expect no error from creating content %s", err)
	key, err := strconv.ParseInt(nct.Key, 10, 64)
	assert.NoErrorf(
		err,
		"expect no error from string to int64 conversion of key %s",
		err,
	)
	sct, err := repo.GetContent(key)
	assert.NoErrorf(err, "expect no error from getting content by slug %s", err)
	testContentProperties(assert, sct, nct)
}

func TestDeleteContent(t *testing.T) {
	t.Parallel()
	assert, repo := setUp(t)
	defer tearDown(repo)
	nct, err := repo.AddContent(testutils.NewStoreContent("catalog", "dsc"))
	assert.NoErrorf(err, "expect no error from creating content %s", err)
	key, err := strconv.ParseInt(nct.Key, 10, 64)
	assert.NoErrorf(
		err,
		"expect no error from string to int64 conversion of key %s",
		err,
	)
	err = repo.DeleteContent(key)
	assert.NoErrorf(
		err,
		"expect no error from deleting content by slug %s",
		err,
	)
	ecnt, err := repo.GetContent(key)
	assert.NoErrorf(err, "expect no error from getting content by slug %s", err)
	assert.True(ecnt.NotFound, "expect no record to be found")
}

func TestEditContent(t *testing.T) {
	t.Parallel()
	assert, repo := setUp(t)
	defer tearDown(repo)
	nct, err := repo.AddContent(testutils.NewStoreContent("catalog", "dsc"))
	assert.NoErrorf(err, "expect no error from creating content %s", err)
	key, err := strconv.ParseInt(nct.Key, 10, 64)
	assert.NoErrorf(
		err,
		"expect no error from string to int64 conversion of key %s",
		err,
	)
	cdata, _ := json.Marshal(&testutils.ContentJSON{
		Paragraph: "clompous",
		Text:      "jack",
	})
	sct, err := repo.EditContent(
		key,
		&content.ExistingContentAttributes{
			UpdatedBy: "packer@packer.com",
			Content:   string(cdata),
		},
	)
	assert.NoErrorf(err, "expect no error from updating content %s", err)
	assert.Equal(sct.UpdatedBy, "packer@packer.com", "should match updated by")
	assert.Equal([]byte(sct.Content), cdata, "should match updated content")
	assert.True(
		sct.UpdatedOn.After(sct.CreatedOn),
		"should have correct updated timestamp",
	)
	assert.Equal(sct.Name, nct.Name, "name should match")
	assert.Equal(sct.Namespace, nct.Namespace, "namespace should match")
	assert.Equal(sct.Slug, nct.Slug, "slug should match")
	assert.Equal(
		sct.CreatedBy,
		nct.CreatedBy,
		"should match created_by",
	)
}

func TestSchemaValidation(t *testing.T) {
	t.Parallel()
	assert, repo := setUp(t)
	defer tearDown(repo)
	_, err := repo.AddContent(testutils.NewStoreContent("catalog", "dsc"))
	assert.NoErrorf(err, "expect no error from creating content %s", err)
	_, err = repo.AddContent(testutils.NewStoreContent("catalog", "dsc"))
	assert.Error(err, "expect schema validation error for duplicate slug")
	ncnt := testutils.NewStoreContent("price", "dsc")
	ncnt.CreatedBy = "yadayadayada"
	_, err = repo.AddContent(ncnt)
	assert.Error(
		err,
		"expect schema validation error for created by field does not have an email address",
	)
}

func testContentProperties(
	assert *require.Assertions,
	sct, nct *model.ContentDoc,
) {
	assert.Equal(sct.Name, nct.Name, "name should match")
	assert.Equal(sct.Namespace, nct.Namespace, "namespace should match")
	assert.Equal(sct.Slug, nct.Slug, "slug should match")
	assert.Equal(
		sct.CreatedBy,
		nct.CreatedBy,
		"should match created_by",
	)
	assert.True(
		sct.CreatedOn.Equal(nct.CreatedOn),
		"created_on should match",
	)
	assert.True(
		sct.UpdatedOn.Equal(nct.UpdatedOn),
		"created_on should match",
	)
	assert.Equal(sct.Content, nct.Content, "should match raw conent")
}
