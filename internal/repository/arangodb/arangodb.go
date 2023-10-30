package arangodb

import (
	"context"
	"errors"
	"fmt"
	"strconv"

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
	slug string,
) (*model.ContentDoc, error) {
	cntModel := &model.ContentDoc{}
	resp, err := arp.database.GetRow(
		ContentFindBySlug,
		map[string]interface{}{
			"@content_collection": arp.content.Name(),
			"slug":                slug,
		},
	)
	if err != nil {
		return cntModel, fmt.Errorf(
			"error in getting content by slug name %s",
			err,
		)
	}
	if resp.IsEmpty() {
		cntModel.NotFound = true

		return cntModel, nil
	}
	if err := resp.Read(cntModel); err != nil {
		return cntModel, fmt.Errorf(
			"error in reading response to struct %s",
			err,
		)
	}

	return cntModel, nil
}

func (arp *arangorepository) GetContent(cid int64) (*model.ContentDoc, error) {
	cntModel := &model.ContentDoc{}
	cntCollection, err := arp.database.Collection(arp.content.Name())
	if err != nil {
		return cntModel, fmt.Errorf("error in getting collection %s", err)
	}
	meta, err := cntCollection.ReadDocument(
		context.Background(),
		strconv.Itoa(int(cid)),
		cntModel,
	)
	if err != nil {
		errMsg := fmt.Sprintf("error in reading document %s", err)
		if driver.IsNotFoundGeneral(err) {
			errMsg = fmt.Sprintf("document with ID %d not found", cid)
		}

		return cntModel, errors.New(errMsg)
	}
	cntModel.ID = meta.ID
	cntModel.Key = meta.Key

	return cntModel, nil
}

func (arp *arangorepository) DeleteContent(cid int64) error {
	cntCollection, err := arp.database.Collection(arp.content.Name())
	if err != nil {
		return fmt.Errorf("error in getting collection %s", err)
	}
	_, err = cntCollection.RemoveDocument(context.Background(), strconv.Itoa(int(cid)))
	if err != nil {
		errMsg := fmt.Sprintf("error in reading document %s", err)
		if driver.IsNotFoundGeneral(err) {
			errMsg = fmt.Sprintf("document with ID %d not found", cid)
		}

		return errors.New(errMsg)
	}

	return nil
}

func (arp *arangorepository) AddContent(
	cattr *content.NewContentAttributes,
) (*model.ContentDoc, error) {
	cntModel := &model.ContentDoc{}
	res, err := arp.database.DoRun(
		ContentInsert,
		map[string]interface{}{
			"name":       cattr.Name,
			"namespace":  cattr.Namespace,
			"created_by": cattr.CreatedBy,
			"updated_by": cattr.CreatedBy,
			"content":    cattr.Content,
			"slug":       cattr.Slug,
		},
	)
	if err != nil {
		return cntModel, fmt.Errorf("error in creating new content %s", err)
	}
	if err := res.Read(cntModel); err != nil {
		return cntModel, fmt.Errorf(
			"error in reading the model to struct %s",
			err,
		)
	}

	return cntModel, nil
}

func (arp *arangorepository) EditContent(
	cid int64,
	cattr *content.ExistingContentAttributes,
) (*model.ContentDoc, error) {
	cntModel := &model.ContentDoc{}
	res, err := arp.database.DoRun(
		ContentUpdate,
		map[string]interface{}{
			"key":        cid,
			"updated_by": cattr.UpdatedBy,
			"content":    cattr.Content,
		},
	)
	if err != nil {
		return cntModel, fmt.Errorf("error in updating content %s", err)
	}
	if err := res.Read(cntModel); err != nil {
		return cntModel, fmt.Errorf(
			"error in reading the model to struct %s",
			err,
		)
	}

	return cntModel, nil
}
