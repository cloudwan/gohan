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
	"database/sql"
	"fmt"
	"os"
	"path"

	"github.com/pressly/goose"

	"github.com/cloudwan/gohan/log"
	"github.com/cloudwan/gohan/util"
)

var logger = log.NewLogger()

func LoadConfig(configFile string) (err error) {
	config := util.GetConfig()

	err = config.ReadConfig(configFile)

	if err != nil {
		fmt.Printf("error: failed to load config: %s\n", err.Error())
		return
	}

	err = os.Chdir(path.Dir(configFile))

	if err != nil {
		fmt.Printf("error: chdir() failed: %s\n", err.Error())
		return
	}

	return
}

type config struct {
	dbType string
	dbConnection string
	dbNoInit bool
	dbMigrations string
}

func readConfig() *config {
	c := util.GetConfig()
	return &config{
		dbType: c.GetString("database/type", "sqlite3"),
		dbConnection: c.GetString("database/connection", ""),
		dbNoInit: c.GetBool("database/no_init", false),
		dbMigrations: c.GetString("database/migrations", "etc/db/migrations"),
	}
}

func EnsureVersion() error {
	config := readConfig()

	if !config.dbNoInit {
		logger.Debug("migration: db version check skipped: no_init")
		return nil
	}

	if config.dbMigrations == "" {
		logger.Debug("migration: db version check skipped: no migrations path")
		return nil
	}

	if err := goose.SetDialect(config.dbType); err != nil {
		return fmt.Errorf("migration: failed to set goose dialect: %s", err)
	}
	db, err := sql.Open(config.dbType, config.dbConnection)
	if err != nil {
		return fmt.Errorf("migration: failed to open db: %s", err)
	}

	ms, err := goose.CollectMigrations(config.dbMigrations, 0, goose.MaxVersion)
	if err != nil {
		return fmt.Errorf("migration: failed to list migrations: %s", err)
	}
	if len(ms) == 0 {
		logger.Debug("migration: no migrations")
		return nil
	}

	m, err := ms.Last()
	if err != nil {
		return fmt.Errorf("migration: failed to get last migration: %s", err)
	}

	v, err := goose.EnsureDBVersion(db)
	if err != nil {
		return fmt.Errorf("migration: failed to ensure db version: %s", err)
	}
	logger.Info("migration path: %q, db version: %d", config.dbMigrations, v)

	if m.Version != v {
		return fmt.Errorf("migration: version mismatch db version=%d, last migration=%d", v, m.Version)
	}

	return nil
}

func Help() {
	fmt.Println("missing subcommand: help, up, up-by-one, up-to, create, down, down-to, redo, status, version")
}

func Run(subcmd string, args []string) {
	config := readConfig()

	if err := goose.SetDialect(config.dbType); err != nil {
		fmt.Printf("error: failed to set goose dialect: %s\n", err.Error())
		return
	}

	db, err := sql.Open(config.dbType, config.dbConnection)

	if err != nil {
		fmt.Printf("error: failed to open db: %s\n", err.Error())
		return
	}

	err = goose.Run(subcmd, db, config.dbMigrations, args...)
	if err != nil {
		fmt.Println(err)
	}
}
