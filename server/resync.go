package server

import (
	"fmt"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/sync"
	"github.com/cloudwan/gohan/util"
)

func Resync(dbConn db.DB, sync sync.Sync) (err error) {

	syncDbConn := &DbSyncWrapper{DB: dbConn}
	schemaManager := schema.GetManager()

	tx, err := syncDbConn.Begin()
	if err != nil {
		return fmt.Errorf("Error when acquiring DB transaction: %s", err)
	}
	committed := false
	defer func() {
		if !committed {
			tx.Close()
		}
	}()

	tl := tx.(*transactionEventLogger)
	for _, schemaType := range schemaManager.OrderedSchemas() {
		if schemaType.IsAbstract() {
			log.Debug("Skip abstract schema %s", schemaType.ID)
			continue
		}

		if schemaType.Metadata["type"] == "metaschema" {
			log.Debug("Skip metaschema %s", schemaType.ID)
			continue
		}

		if schemaType.Metadata["nosync"] == true {
			log.Debug("Skip nosync schema %s", schemaType.ID)
			continue
		}

		log.Info("Re-emitting events for resource type %s", schemaType.ID)
		all, _, err := tl.List(schemaType, transaction.Filter{}, nil, nil)
		if err != nil {
			util.ExitFatal(fmt.Sprintf("Error when acquiring DB transaction: %s", err))
		}
		for _, resource := range all {
			tl.Resync(resource)
		}
		log.Info("Done re-emitting events for resource type %s", schemaType.ID)
	}

	err = tl.Commit()
	if err != nil {
		return fmt.Errorf("Error when committing DB transaction: %s", err)
	}
	committed = true

	syncWriter := NewSyncWriter(sync, dbConn)
	totallySynced := 0
	for {
		synced, err := syncWriter.Sync()
		if err != nil {
			return fmt.Errorf("Error when syncing events: %s", err)
		}
		if synced == 0 {
			break
		}
		totallySynced += synced
	}
	log.Info("Resync completed, synced %d resources", totallySynced)

	return
}
