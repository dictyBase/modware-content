package arangodb

import (
	"encoding/json"
	"fmt"
	"regexp"
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

func TestListContents(t *testing.T) {
	t.Parallel()
	assert, repo := setUp(t)
	defer tearDown(repo)

	// Create test data
	createTestContents(assert, repo, 14)

	// Test initial list
	clist, err := repo.ListContents(0, 4, "")
	assert.NoError(err, "expect no error from listing content")
	assert.Len(clist, 5, "should have 5 content entries")
	validateContentList(assert, clist)

	// Test pagination
	clist2, err := repo.ListContents(
		clist[len(clist)-1].CreatedOn.UnixMilli(),
		5,
		"",
	)
	assert.NoError(err, "expect no error from listing content")
	assert.Len(clist2, 6, "should have 6 content entries")
	assert.Exactly(
		clist[len(clist)-1],
		clist2[0],
		"should be identical content object",
	)

	// Test final page
	clist3, err := repo.ListContents(
		clist2[len(clist2)-1].CreatedOn.UnixMilli(),
		7,
		"",
	)
	assert.NoError(err, "expect no error from listing content")
	assert.Len(clist3, 5, "should have 5 content entries")
	assert.Exactly(
		clist2[len(clist2)-1],
		clist3[0],
		"should be identical content object",
	)
}

func createTestContents(
	assert *require.Assertions,
	repo repository.ContentRepository,
	count int,
) {
	for i := 0; i < count; i++ {
		_, err := repo.AddContent(
			testutils.NewStoreContent(fmt.Sprintf("gallery-%d", i), "genome"),
		)
		assert.NoErrorf(
			err,
			"expect no error from creating gallery genome content %s",
			err,
		)
	}
}

func validateContentList(
	assert *require.Assertions,
	clist []*model.ContentDoc,
) {
	nrxp := regexp.MustCompile(`gallery-\d+`)
	for idx, cnt := range clist {
		assert.Equal(
			cnt.Namespace,
			"genome",
			"should match the genome namespace",
		)
		assert.Regexp(nrxp, cnt.Name, "should match gallery names")
		assert.Equal(
			cnt.CreatedBy,
			"content@content.org",
			"should match the created_by",
		)
		if idx != 0 {
			assert.True(
				clist[idx-1].CreatedOn.After(clist[idx].CreatedOn),
				"previous gallery record should be created after the current one",
			)
		}
	}
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
	assert.NotEqual(
		sct.UpdatedBy,
		sct.CreatedBy,
		"update and creation email should not match",
	)
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
	act, err := repo.GetContentBySlug(sct.Slug)
	assert.NoError(err, "expect no error for retrieval after update")
	assert.Equal(act.UpdatedBy, sct.UpdatedBy, "should match updated_by")
	assert.NotEqual(
		act.UpdatedBy,
		act.CreatedBy,
		"update and created email should not match",
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
	assert.Equal(sct.UpdatedBy, nct.UpdatedBy, "should match updated by")
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
