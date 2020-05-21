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

	"github.com/cloudwan/gohan/db/migration"
	db_options "github.com/cloudwan/gohan/db/options"
	db_sql "github.com/cloudwan/gohan/db/sql"
	"github.com/cloudwan/gohan/schema"
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
