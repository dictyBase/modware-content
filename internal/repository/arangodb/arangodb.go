package arangodb

import (
	"fmt"

	driver "github.com/arangodb/go-driver"
	manager "github.com/dictyBase/arangomanager"
	"github.com/dictyBase/go-genproto/dictybaseapis/content"
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
	if err := schemaOptions.LoadRule(model.Schema()); err != nil {
		return arp, fmt.Errorf("error in loading schema %s", err)
	}
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
	_, _, err = dbs.EnsurePersistentIndex(
		collection,
		[]string{"slug"},
		&driver.EnsurePersistentIndexOptions{
			Unique:       true,
			InBackground: true,
			Name:         "collection_slug_idx",
		},
	)
	if err != nil {
		return arp, fmt.Errorf(
			"error in creating unique index for slug field %s",
			err,
		)
	}
	_, _, err = dbs.EnsurePersistentIndex(
		collection,
		[]string{"namespace"},
		&driver.EnsurePersistentIndexOptions{
			InBackground: true,
			Name:         "content_namespace_idx",
		},
	)
	if err != nil {
		return arp, fmt.Errorf(
			"error in creating index for name field %s",
			err,
		)
	}

	return arp, nil
}

func (arp *arangorepository) GetContentBySlug(
	id string,
) (*model.ContentDoc, error) {
	panic("not implemented") // TODO: Implement
}

func (arp *arangorepository) GetContent(id string) (*model.ContentDoc, error) {
	panic("not implemented") // TODO: Implement
}

func (arp *arangorepository) AddContent(
	cnt *content.NewContentAttributes,
) (*model.ContentDoc, error) {
	panic("not implemented") // TODO: Implement
}

func (arp *arangorepository) EditContent(
	cnt *content.ExistingContentAttributes,
) (*model.ContentDoc, error) {
	panic("not implemented") // TODO: Implement
}

func (arp *arangorepository) DeleteContent(id string) error {
	panic("not implemented") // TODO: Implement
}
