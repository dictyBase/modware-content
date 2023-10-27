package arangodb

import (
	"fmt"

	driver "github.com/arangodb/go-driver"
	manager "github.com/dictyBase/arangomanager"
	"github.com/dictyBase/modware-content/internal/model"
	"github.com/dictyBase/modware-content/internal/repository"
)

type arangorepository struct {
	sess     *manager.Session
	database *manager.Database
	content  driver.Collection
}

func NewContentRepo(
	connP *manager.ConnectParams,
	collection string,
) (repository.ContentRepository, error) {
	arp := &arangorepository{}
	sess, dbs, err := manager.NewSessionDb(connP)
	if err != nil {
		return arp, fmt.Errorf("error in getting new session %s", err)
	}
	arp.sess = sess
	arp.database = dbs
	schemaOptions := &driver.CollectionSchemaOptions{}
	schemaOptions.LoadRule(model.Schema())
	contentCollection, err := dbs.FindOrCreateCollection(
		collection,
		&driver.CreateCollectionOptions{Schema: schemaOptions},
	)
	if err != nil {
		return arp, fmt.Errorf(
			"error in finding or creating collection %s",
			err,
		)
	}
	arp.content = contentCollection

	return arp, nil
}
