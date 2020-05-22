// Copyright (C) 2020 NTT Innovation Institute, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cli

import (
	context_pkg "context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/dbutil"
	"github.com/cloudwan/gohan/db/migration"
	"github.com/cloudwan/gohan/extension"
	"github.com/cloudwan/gohan/extension/goplugin"
	"github.com/cloudwan/gohan/extension/otto"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/server"
	"github.com/cloudwan/gohan/server/middleware"
	"github.com/cloudwan/gohan/sync"
	sync_util "github.com/cloudwan/gohan/sync/util"
	"github.com/cloudwan/gohan/util"
)

func emitPostMigrateEvent(ctx context_pkg.Context, forcedSchemas string, syncETCDEvent bool,
	postMigrationEventTimeout time.Duration) {

	e := &postMigrationEventEmitter{
		ctx:           ctx,
		forcedSchemas: forcedSchemas,
		syncETCDEvent: syncETCDEvent,
		eventTimeout:  postMigrationEventTimeout,
	}

	e.emit()
}

type postMigrationEventEmitter struct {
	ctx           context_pkg.Context
	forcedSchemas string
	syncETCDEvent bool
	eventTimeout  time.Duration
}

func (e *postMigrationEventEmitter) emit() {
	config := util.GetConfig()

	log.Info("Emit post-migrate event")

	modifiedSchemas := selectModifiedSchemas(e.forcedSchemas)

	if len(modifiedSchemas) == 0 {
		log.Info("No modified schemas, skipping post-migration event")
		return
	}

	log.Debug("Modified schemas: %s", strings.Join(modifiedSchemas, ", "))

	schemaFiles := config.GetStringList("schemas", nil)

	if schemaFiles == nil {
		log.Fatal("No schema specified in configuration")
	}

	manager := schema.GetManager()
	if err := manager.LoadSchemasFromFiles(schemaFiles...); err != nil {
		log.Fatal(err)
	}

	if err := e.publishPostMigrateEvent(postMigrationEnvName, modifiedSchemas); err != nil {
		log.Fatal("Publish post-migrate event failed: %s", err)
	}

	schema.ClearManager()

	log.Info("Published post-migrate event: %s", strings.Join(modifiedSchemas, ", "))
}

func (e *postMigrationEventEmitter) publishPostMigrateEvent(envName string, modifiedSchemas []string) error {
	config := util.GetConfig()

	rawDB, err := dbutil.CreateFromConfig(config)

	if err != nil {
		log.Fatal(err)
	}

	db := server.NewDbSyncWrapper(rawDB)

	sync, err := sync_util.CreateFromConfig(config)

	if err != nil {
		log.Fatal(err)
	}

	ident, err := middleware.CreateIdentityServiceFromConfig(config)

	if err != nil {
		log.Fatal(err)
	}

	envManager := extension.GetManager()
	manager := schema.GetManager()

	e.publishPostMigrateEventWithServices(envName, modifiedSchemas, db, manager, envManager, sync, ident)

	return nil
}

func (e *postMigrationEventEmitter) publishPostMigrateEventWithServices(envName string, modifiedSchemas []string, db db.DB,
	manager *schema.Manager, envManager *extension.Manager, sync sync.Sync, ident middleware.IdentityService) {

	deadline := time.Now().Add(e.eventTimeout)

	for _, s := range manager.Schemas() {
		if !util.ContainsString(modifiedSchemas, s.ID) {
			continue
		}

		pluralURL := s.GetPluralURL()

		if _, ok := envManager.GetEnvironment(s.ID); !ok {
			now := time.Now()
			left := deadline.Sub(now)
			if now.After(deadline) {
				log.Fatalf("Timeout after '%s' secs while publishing event to schemas", e.eventTimeout.Seconds())
			}

			envOtto := otto.NewEnvironment(envName, db, ident, sync)
			envOtto.SetEventTimeLimit(eventPostMigration, left)

			envGoplugin := goplugin.NewEnvironment(envName, nil, nil)
			envGoplugin.SetDatabase(db)
			envGoplugin.SetSync(sync)

			env := extension.NewEnvironment([]extension.Environment{envOtto, envGoplugin})

			log.Info("Loading environment for %s schema with URL: %s", s.ID, pluralURL)

			if err := env.LoadExtensionsForPath(manager.Extensions, manager.TimeLimit, manager.TimeLimits, pluralURL); err != nil {
				log.Fatal(fmt.Sprintf("[%s] %v", pluralURL, err))
			}

			envManager.RegisterEnvironment(s.ID, env)
		}

		env, _ := envManager.GetEnvironment(s.ID)

		eventContext := map[string]interface{}{}
		eventContext["schema"] = s
		eventContext["schema_id"] = s.ID
		eventContext["sync"] = sync
		eventContext["db"] = db
		eventContext["identity_service"] = ident
		eventContext["context"] = e.ctx
		eventContext["trace_id"] = util.NewTraceID()

		if err := env.HandleEvent(eventPostMigration, eventContext); err != nil {
			log.Fatalf("Failed to handle event '%s': %s", eventPostMigration, err)
		}

		if err := migration.UnmarkSchema(s.ID); err != nil {
			log.Fatalf("Failed to remove '%s' from pending post migration schemas: %s", s.ID, err)
		}
	}

	e.clearDeadPendingSchemas(modifiedSchemas, manager)

	if e.syncETCDEvent {
		if _, err := server.NewSyncWriter(sync, db).Sync(e.ctx); err != nil {
			log.Fatalf("Failed to synchronize post-migration events, err: %s", err)
		}
	}
}

func (e *postMigrationEventEmitter) clearDeadPendingSchemas(modifiedSchemas []string, manager *schema.Manager) {
	for _, schemaID := range modifiedSchemas {
		_, exists := manager.Schema(schemaID)
		if exists {
			continue
		}

		log.Warning("Found no longer existing schema '%s' in pending post-migration events, removing", schemaID)
		if err := migration.UnmarkSchema(schemaID); err != nil {
			log.Fatalf("Failed to remove no longer existing '%s' from pending post migration schemas: %s", schemaID, err)
		}
	}
}
