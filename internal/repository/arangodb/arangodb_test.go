package arangodb

import (
	"testing"

	manager "github.com/dictyBase/arangomanager"
	"github.com/dictyBase/arangomanager/testarango"
	"github.com/dictyBase/modware-content/internal/repository"
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
