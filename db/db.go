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

package db

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/cloudwan/gohan/db/file"
	"github.com/cloudwan/gohan/db/options"
	"github.com/cloudwan/gohan/db/sql"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/util"
)

//DefaultMaxOpenConn will applied for db object
const DefaultMaxOpenConn = 100

var errNoSchemasInManager = errors.New("No schemas in Manager. Did you remember to load them?")

//DB is a common interface for handing db
type DB interface {
	Connect(string, string, int) error
	Close()
	Begin() (transaction.Transaction, error)
	BeginTx(ctx context.Context, options *transaction.TxOptions) (transaction.Transaction, error)
	RegisterTable(s *schema.Schema, cascade, migrate bool) error
	DropTable(*schema.Schema) error

	// options
	Options() options.Options
}

// IsDeadlock checks if error is deadlock
func IsDeadlock(err error) bool {
	knownDatabaseErrorMessages := []string{
		"Deadlock found when trying to get lock; try restarting transaction", /* MySQL / MariaDB */
		"database is locked",                                                 /* SQLite */
	}

	for _, msg := range knownDatabaseErrorMessages {
		if strings.Contains(err.Error(), msg) {
			return true
		}
	}

	return false
}

func withinTxImpl(db DB, beginStrategy func(db DB) (transaction.Transaction, error), fn func(transaction.Transaction) error) error {
	var tx transaction.Transaction
	var err error

	defer func() {
		if !tx.Closed() {
			tx.Close()
		}
	}()

	for attempt := 0; attempt <= db.Options().RetryTxCount; attempt++ {
		tx, err = beginStrategy(db)

		if err != nil {
			log.Warning("failed to begin scoped transaction")
			return err
		}

		err = fn(tx)

		if !tx.Closed() {
			tx.Close()
		}

		if err == nil {
			return nil
		}

		log.Debug("scoped database transaction failed with error: %s", err)

		if !IsDeadlock(err) {
			return err
		}

		log.Warning("scoped transaction deadlocked, retrying %d / %d", attempt, db.Options().RetryTxCount)
		time.Sleep(db.Options().RetryTxInterval)
	}

	log.Warning("scoped transaction still deadlocked after %d retries; gave up", db.Options().RetryTxCount)
	return err
}

// Within executes a scoped transaction on a database
func Within(db DB, fn func(transaction.Transaction) error) error {
	return withinTxImpl(db,
		func(db DB) (transaction.Transaction, error) {
			return db.Begin()
		}, fn)
}

// WithinTx executes a scoped transaction with options on a database
func WithinTx(ctx context.Context, db DB, options *transaction.TxOptions, fn func(transaction.Transaction) error) error {
	return withinTxImpl(db,
		func(db DB) (transaction.Transaction, error) {
			return db.BeginTx(ctx, options)
		}, fn)
}

//ConnectDB is builder function of DB
func ConnectDB(dbType, conn string, maxOpenConn int, opt options.Options) (DB, error) {
	var db DB

	if dbType == "json" || dbType == "yaml" {
		db = file.NewDB(opt)
	} else {
		db = sql.NewDB(opt)
	}

	err := db.Connect(dbType, conn, maxOpenConn)
	if err != nil {
		return nil, err
	}
	return db, nil
}

//CopyDBResources copies resources from input database to output database
func CopyDBResources(input, output DB, overrideExisting bool) error {
	schemaManager := schema.GetManager()
	schemas := schemaManager.OrderedSchemas()

	if len(schemas) == 0 {
		return errNoSchemasInManager
	}

	if errInputTx := Within(input, func(inputTx transaction.Transaction) error {
		if errorOutputTx := Within(output, func(outputTx transaction.Transaction) error {
			for _, s := range schemas {
				if s.IsAbstract() {
					continue
				}
				log.Info("Populating resources for schema %s", s.ID)
				resources, _, err := inputTx.List(s, nil, nil, nil)
				if err != nil {
					return err
				}

				for _, resource := range resources {
					log.Info("Creating resource %s", resource.ID())
					destResource, _ := outputTx.Fetch(s, transaction.IDFilter(resource.ID()), nil)
					if destResource == nil {
						resource.PopulateDefaults()
						err := outputTx.Create(resource)
						if err != nil {
							return err
						}
					} else if overrideExisting {
						err := outputTx.Update(resource)
						if err != nil {
							return err
						}
					}
				}
			}
			return outputTx.Commit()
		}); errorOutputTx != nil {
			return errorOutputTx
		}
		return inputTx.Commit()
	}); errInputTx != nil {
		return errInputTx
	}

	return nil
}

// CreateFromConfig creates db connection from config
func CreateFromConfig(config *util.Config) (DB, error) {
	dbType := config.GetString("database/type", "sqlite3")
	dbConnection := config.GetString("database/connection", "")
	maxConn := config.GetInt("database/max_open_conn", DefaultMaxOpenConn)
	dbOptions := options.Read(config)

	var dbConn DB
	if dbType == "json" || dbType == "yaml" {
		dbConn = file.NewDB(dbOptions)
	} else {
		dbConn = sql.NewDB(dbOptions)
	}
	err := dbConn.Connect(dbType, dbConnection, maxConn)
	if err != nil {
		return nil, err
	}
	return dbConn, nil
}

// InitDBConnWithSchemas initializes database connection using schemas stored in Manager
func InitDBConnWithSchemas(aDb DB, dropOnCreate, cascade, migrate bool) error {
	var err error
	schemaManager := schema.GetManager()
	schemas := schemaManager.OrderedSchemas()
	if len(schemas) == 0 {
		return errNoSchemasInManager
	}
	if dropOnCreate {
		for i := len(schemas) - 1; i >= 0; i-- {
			s := schemas[i]
			if s.IsAbstract() {
				continue
			}
			log.Debug("Dropping table '%s':", s.Plural)
			err = aDb.DropTable(s)
			if err != nil {
				log.Warning("Error during deleting table '%s': %s", s.Plural, err.Error())
			}
		}
	}
	for _, s := range schemas {
		if s.IsAbstract() {
			continue
		}
		log.Debug("Registering schema %s", s.ID)
		err = aDb.RegisterTable(s, cascade, migrate)
		if err != nil {
			message := "Error during registering table %q: %s"
			if strings.Contains(err.Error(), "already exists") {
				log.Warning(message, s.GetDbTableName(), err)
			} else {
				log.Fatalf(message, s.GetDbTableName(), err)
			}
		}
	}
	return nil
}

// InitDBWithSchemas initializes database using schemas stored in Manager
func InitDBWithSchemas(dbType, dbConnection string, dropOnCreate, cascade, migrate bool) error {
	aDb, err := ConnectDB(dbType, dbConnection, DefaultMaxOpenConn, options.Default())
	if err != nil {
		return err
	}
	defer aDb.Close()
	return InitDBConnWithSchemas(aDb, dropOnCreate, cascade, migrate)
}
