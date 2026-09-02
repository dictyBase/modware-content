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
	createCustomTestContents(assert, repo, 14, "gallery", "genome")

	// Test initial list
	clist, err := repo.ListContents(0, 4, "")
	assert.NoError(err, "expect no error from listing content")
	assert.Len(clist, 5, "should have 5 content entries")
	validateContentList(assert, clist, "gallery", "genome")

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

func TestListContentsWithFilter(t *testing.T) {
	t.Parallel()
	assert, repo := setUp(t)
	defer tearDown(repo)

	createCustomTestContents(assert, repo, 6, "sword", "longbottom")
	createCustomTestContents(assert, repo, 8, "field", "drinkwater")

	testCases := []struct {
		name          string
		filter        string
		expectedCount int
		expectedName  string
		expectedNS    string
		expectError   bool
	}{
		{
			name:          "Filter by namespace",
			filter:        `FILTER cnt.namespace == "longbottom"`,
			expectedCount: 6,
			expectedName:  "sword",
			expectedNS:    "longbottom",
			expectError:   false,
		},
		{
			name:          "Filter by slug substring",
			filter:        `FILTER cnt.slug =~ "drink"`,
			expectedCount: 8,
			expectedName:  "field",
			expectedNS:    "drinkwater",
			expectError:   false,
		},
		{
			name:          "Combination filter with multiple results",
			filter:        `FILTER cnt.name =~ "sword-" AND cnt.namespace =~ "long"`,
			expectedCount: 6,
			expectedName:  "sword",
			expectedNS:    "longbottom",
			expectError:   false,
		},
		{
			name:        "Combination filter with no results",
			filter:      `FILTER cnt.name == "sword-3" AND cnt.namespace =~ "drink"`,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clist, err := repo.ListContents(0, 15, tc.filter)
			if tc.expectError {
				assert.Error(err, "expect error from listing content")
				assert.True(
					repository.IsContentListNotFound(err),
					"should be list not found type of error",
				)
			} else {
				assert.NoError(err, "expect no error from listing content")
				assert.Len(clist, tc.expectedCount, "should have expected number of content entries")
				validateContentList(assert, clist, tc.expectedName, tc.expectedNS)
			}
		})
	}
}

func TestListContentsWithFilterAndCursor(t *testing.T) {
	t.Parallel()
	assert, repo := setUp(t)
	defer tearDown(repo)
	createCustomTestContents(assert, repo, 10, "wand", "hogwarts")
	createCustomTestContents(assert, repo, 12, "potion", "hogsmeade")

	for _, tc := range testutils.CreateTestCases() {
		t.Run(tc.Name, func(t *testing.T) {
			runListContentsWithFilterAndCursorSubtest(assert, repo, tc)
		})
	}
}

func runListContentsWithFilterAndCursorSubtest(
	assert *require.Assertions,
	repo repository.ContentRepository,
	tc testutils.TestCaseWithFilterandCursor,
) {
	// Initial list
	clist, err := repo.ListContents(0, tc.InitialLimit, tc.Filter)
	assert.NoError(
		err,
		"expect no error from listing content with filter",
	)
	assert.Len(
		clist,
		tc.ExpectedInitialCount,
		"should have expected number of content entries",
	)
	validateContentList(assert, clist, tc.ExpectedName, tc.ExpectedNS)

	// Subsequent list with cursor
	clist2, err := repo.ListContents(
		clist[len(clist)-1].CreatedOn.UnixMilli(),
		tc.SubsequentLimit,
		tc.Filter,
	)
	assert.NoError(
		err,
		"expect no error from listing content with filter and cursor",
	)
	assert.Len(
		clist2,
		tc.ExpectedSubsequentCount,
		"should have expected number of content entries",
	)
	assert.Exactly(
		clist[len(clist)-1],
		clist2[0],
		"should be identical content object",
	)
	validateContentList(assert, clist2, tc.ExpectedName, tc.ExpectedNS)
}

func createCustomTestContents(
	assert *require.Assertions,
	repo repository.ContentRepository,
	count int,
	name, namespace string,
) {
	for i := range count {
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
