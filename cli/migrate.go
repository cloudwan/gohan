// Copyright (C) 2017 NTT Innovation Institute, Inc.
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
	"bytes"
	context_pkg "context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/dbutil"
	"github.com/cloudwan/gohan/db/migration"
	db_options "github.com/cloudwan/gohan/db/options"
	db_sql "github.com/cloudwan/gohan/db/sql"
	"github.com/cloudwan/gohan/extension"
	"github.com/cloudwan/gohan/extension/goplugin"
	"github.com/cloudwan/gohan/extension/otto"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/server"
	"github.com/cloudwan/gohan/server/middleware"
	"github.com/cloudwan/gohan/sync"
	sync_util "github.com/cloudwan/gohan/sync/util"
	"github.com/cloudwan/gohan/util"
	"github.com/urfave/cli"
)

const (
	// events
	eventPostMigration = "post-migration"

	// settings
	syncMigrationsPath   = "/gohan/cluster/migrations"
	postMigrationEnvName = "post-migration-env"

	// flags
	FlagEmitPostMigrationEvent    = "emit-post-migration-event"
	FlagPostMigrationEventTimeout = "post-migration-event-timeout"
	FlagForcedSchemas             = "forced-schemas"
	FlagLockWithETCD              = "lock-with-etcd"
	FlagSyncETCDEvent             = "sync-etcd-event"
)

func withinLockedMigration(fn func(context_pkg.Context, *cli.Context)) func(*cli.Context) {
	return func(context *cli.Context) {
		workingDir, _ := os.Getwd()
		defer func() {
			_ = os.Chdir(workingDir)
		}()

		configFile := context.String("config-file")

		if migration.LoadConfig(configFile) != nil {
			return
		}

		ctx := context_pkg.Background()

		if !context.Bool(FlagLockWithETCD) {
			fn(ctx, context)
			return
		}

		config := util.GetConfig()
		sync, err := sync_util.CreateFromConfig(config)

		if err != nil {
			log.Fatal(err)
		}

		if sync == nil {
			log.Fatal("sync is nil")
		}

		_, err = sync.Lock(ctx, syncMigrationsPath, true)

		if err != nil {
			log.Fatal(err)
		}

		defer sync.Unlock(ctx, syncMigrationsPath)

		fn(ctx, context)
	}
}

func selectModifiedSchemas(forcedSchemas string) []string {
	if len(forcedSchemas) > 0 {
		return strings.Split(forcedSchemas, ",")
	}
	return migration.GetModifiedSchemas()
}

func emitPostMigrateEvent(ctx context_pkg.Context, forcedSchemas string, syncETCDEvent bool, postMigrationEventTimeout time.Duration) {
	config := util.GetConfig()

	log.Info("Emit post-migrate event")

	modifiedSchemas := selectModifiedSchemas(forcedSchemas)

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

	if err := publishPostMigrateEvent(ctx, postMigrationEnvName, modifiedSchemas, syncETCDEvent, postMigrationEventTimeout); err != nil {
		log.Fatal("Publish post-migrate event failed: %s", err)
	}

	schema.ClearManager()

	log.Info("Published post-migrate event: %s", strings.Join(modifiedSchemas, ", "))
}

func actionMigrate(subcmd string) func(context *cli.Context) {
	return withinLockedMigration(func(ctx context_pkg.Context, context *cli.Context) {
		if err := migration.Run(subcmd, context.Args()); err != nil {
			log.Fatalf("Migrate run failed: %s", err)
		}
	})
}

func actionMigrateWithPostMigrationEvent(subcmd string) func(context *cli.Context) {
	return withinLockedMigration(func(ctx context_pkg.Context, context *cli.Context) {
		if err := migration.Run(subcmd, context.Args()); err != nil {
			log.Fatalf("Migrate run failed: %s", err)
		}
		if !context.Bool(FlagEmitPostMigrationEvent) {
			return
		}
		emitPostMigrateEvent(ctx, context.String(FlagForcedSchemas), context.Bool(FlagSyncETCDEvent), context.Duration(FlagPostMigrationEventTimeout))
	})
}

func actionMigrateHelp() func(context *cli.Context) {
	return func(context *cli.Context) {
		migration.Help()
	}
}

func actionMigrateCreateInitialMigration() func(context *cli.Context) {
	return func(c *cli.Context) {
		schemaFile := c.String("schema")
		cascade := c.Bool("cascade")
		manager := schema.GetManager()
		configFile := c.String("config-file")
		if configFile != "" {
			config := util.GetConfig()
			config.ReadConfig(configFile)
			schemaFiles := config.GetStringList("schemas", nil)
			if schemaFiles == nil {
				log.Fatal("No schema specified in configuration")
				return
			}
			if err := manager.LoadSchemasFromFiles(schemaFiles...); err != nil {
				log.Fatal(err)
				return
			}
		}
		if schemaFile != "" {
			manager.LoadSchemasFromFiles(schemaFile)
		}
		name := c.String("name")
		now := time.Now()
		version := fmt.Sprintf("%s_%s.sql", now.Format("20060102150405"), name)
		path := filepath.Join(c.String("path"), version)
		var sqlString = bytes.NewBuffer(make([]byte, 0, 100))
		fmt.Printf("Generating goose migration file to %s ...\n", path)
		sqlDB := db_sql.NewDB(db_options.Default())
		schemas := manager.OrderedSchemas()
		sqlString.WriteString("\n")
		sqlString.WriteString("-- +goose Up\n")
		sqlString.WriteString("-- SQL in section 'Up' is executed when this migration is applied\n")
		for _, s := range schemas {
			if s.IsAbstract() {
				continue
			}
			if s.Metadata["type"] == "metaschema" {
				continue
			}
			createSQL, indices := sqlDB.GenTableDef(s, cascade)
			sqlString.WriteString(createSQL + "\n")
			for _, indexSQL := range indices {
				sqlString.WriteString(indexSQL + "\n")
			}
		}
		sqlString.WriteString("\n")
		sqlString.WriteString("-- +goose Down\n")
		sqlString.WriteString("-- SQL section 'Down' is executed when this migration is rolled back\n")
		for _, s := range schemas {
			if s.IsAbstract() {
				continue
			}
			if s.Metadata["type"] == "metaschema" {
				continue
			}
			sqlString.WriteString(fmt.Sprintf("drop table %s;", s.GetDbTableName()))
			sqlString.WriteString("\n\n")
		}
		err := ioutil.WriteFile(path, sqlString.Bytes(), os.ModePerm)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func publishPostMigrateEvent(ctx context_pkg.Context, envName string, modifiedSchemas []string, syncETCDEvent bool,
	eventTimeout time.Duration) error {

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

	publishPostMigrateEventWithServices(ctx, envName, modifiedSchemas, syncETCDEvent, eventTimeout, db, manager, envManager,
		sync, ident)

	return nil
}

func publishPostMigrateEventWithServices(ctx context_pkg.Context, envName string, modifiedSchemas []string, syncETCDEvent bool,
	eventTimeout time.Duration, db db.DB, manager *schema.Manager, envManager *extension.Manager, sync sync.Sync,
	ident middleware.IdentityService) {

	deadline := time.Now().Add(eventTimeout)

	for _, s := range manager.Schemas() {
		if !util.ContainsString(modifiedSchemas, s.ID) {
			continue
		}

		pluralURL := s.GetPluralURL()

		if _, ok := envManager.GetEnvironment(s.ID); !ok {
			now := time.Now()
			left := deadline.Sub(now)
			if now.After(deadline) {
				log.Fatalf("Timeout after '%s' secs while publishing event to schemas", eventTimeout.Seconds())
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
		eventContext["context"] = ctx
		eventContext["trace_id"] = util.NewTraceID()

		if err := env.HandleEvent(eventPostMigration, eventContext); err != nil {
			log.Fatalf("Failed to handle event '%s': %s", eventPostMigration, err)
		}

		if err := migration.UnmarkSchema(s.ID); err != nil {
			log.Fatalf("Failed to remove '%s' from pending post migration schemas: %s", s.ID, err)
		}
	}

	clearDeadPendingSchemas(modifiedSchemas, manager)

	if syncETCDEvent {
		if _, err := server.NewSyncWriter(sync, db).Sync(ctx); err != nil {
			log.Fatalf("Failed to synchronize post-migration events, err: %s", err)
		}
	}
}

func clearDeadPendingSchemas(modifiedSchemas []string, manager *schema.Manager) {
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
