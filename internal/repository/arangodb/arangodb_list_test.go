package arangodb

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/dictyBase/modware-content/internal/model"
	"github.com/dictyBase/modware-content/internal/repository"
	"github.com/dictyBase/modware-content/internal/testutils"
	"github.com/stretchr/testify/require"
)

func TestListContents(t *testing.T) {
	t.Parallel()
	assert, repo := setUp(t)
	defer tearDown(repo)

	// Create test data
	createTestContents(assert, repo, 14)
	createCustomTestContents(assert, repo, 14, "gallery", "genome")

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

}

func createCustomTestContents(
	assert *require.Assertions,
	repo repository.ContentRepository,
	count int,
	name, namespace string,
) {
	for i := 0; i < count; i++ {
		time.Sleep(1 * time.Millisecond)
		_, err := repo.AddContent(
			testutils.NewStoreContent(fmt.Sprintf("%s-%d", name, i), namespace),
		)
		assert.NoErrorf(
			err,
			"expect no error from creating %s %s content %s",
			name, namespace, err,
		)
	}
}

func validateContentList(
	assert *require.Assertions,
	clist []*model.ContentDoc,
	name, namespace string,
) {
	nrxp := regexp.MustCompile(fmt.Sprintf(`%s-\d+`, name))
	for idx, cnt := range clist {
		assert.Equal(
			cnt.Namespace,
			namespace,
			"should match the namespace",
		)
		assert.Regexp(nrxp, cnt.Name, "should match names")
		assert.Equal(
			cnt.CreatedBy,
			"content@content.org",
			"should match the created_by",
		)
		if idx != 0 {
			assert.True(
				clist[idx-1].CreatedOn.After(clist[idx].CreatedOn),
				"previous record should be created after the current one",
			)
		}
	}
}
