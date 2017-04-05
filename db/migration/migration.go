// Copyright (C) 2015 NTT Innovation Institute, Inc.
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

package migration

import (
	"fmt"

	"database/sql"
	"os"
	"path"

	"github.com/cloudwan/gohan/log"
	"github.com/cloudwan/gohan/util"
	"github.com/cloudwan/goose"
)

var logger = log.NewLogger()

func LoadConfig(configFile string) (err error) {
	config := util.GetConfig()

	err = config.ReadConfig(configFile)

	if err != nil {
		fmt.Printf("error: failed to load config: %s\n", err)
		return
	}

	err = os.Chdir(path.Dir(configFile))
	if err != nil {
		fmt.Printf("error: chdir() failed: %s\n", err)
		return
	}

	return
}

func readGooseConfig() (dbType, dbConnection, migrationsPath string) {
	config := util.GetConfig()
	dbType = config.GetString("database/type", "sqlite3")
	dbConnection = config.GetString("database/connection", "")
	migrationsPath = config.GetString("database/migrations", "etc/db/migrations")
	return
}

func Init() error {
	logger.Info("migration: init")

	dbType, dbConnection, migrationsPath := readGooseConfig()

	if err := goose.SetDialect(dbType); err != nil {
		return fmt.Errorf("migration: failed to set goose dialect: %s", err)
	}

	db, err := sql.Open(dbType, dbConnection)

	if err != nil {
		return fmt.Errorf("migration: failed to open db: %s", err)
	}

	v, err := goose.EnsureDBVersion(db)
	if err != nil {
		return fmt.Errorf("migration: failed to ensure db version: %s", err)
	}

	logger.Info("migration path: %q, version: %d", migrationsPath, v)

	if err = goose.LoadMigrationPlugins(migrationsPath); err != nil {
		return fmt.Errorf("migration: failed to load migration plugins: %s", err)
	}

	return goose.Status(db, migrationsPath)
}

func Help() {
	fmt.Println("missing subcommand: help, up, up-by-one, up-to, create, create-next, down, down-to, redo, status, version")
}

func Run(subCmd string, args []string) {
	dbType, dbConnection, migrationsPath := readGooseConfig()

	if err := goose.SetDialect(dbType); err != nil {
		fmt.Printf("error: failed to set goose dialect: %s\n", err)
		os.Exit(1)
	}

	db, err := sql.Open(dbType, dbConnection)

	if err != nil {
		fmt.Printf("error: failed to open db: %s\n", err)
		os.Exit(1)
	}

	if err = goose.LoadMigrationPlugins(migrationsPath); err != nil {
		logger.Error("migration: failed to load migration plugins: %s", err)
		return
	}

	if err != nil {
		logger.Error("migration: failed to load runtime migrations: %s", err.Error())
		return
	}

	if err = goose.Run(subCmd, db, migrationsPath, args...); err != nil {
		fmt.Printf("migration: failed to run: %s\n", err)
		os.Exit(1)
	}
}

var modifiedSchemas = map[string]bool{}

func MarkSchemaAsModified(schemaID string) {
	modifiedSchemas[schemaID] = true
}

func GetModifiedSchemas() []string {
	schemas := make([]string, len(modifiedSchemas))
	for schema, _ := range modifiedSchemas {
		schemas = append(schemas, schema)
	}
	return schemas
}
