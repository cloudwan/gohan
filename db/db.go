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
	"math/rand"
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

// ITransaction is a common interface for transaction
type ITransaction interface {
	Commit() error
	Close() error
	Closed() bool
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

func tryToCommit(fn func(ITransaction) error) func(ITransaction) error {
	return func(tx ITransaction) error {
		if err := fn(tx); err != nil {
			return err
		}
		return tx.Commit()
	}
}

func tryWithinTx(
	beginStrategy func() (ITransaction, error),
	fn func(ITransaction) error,
) error {
	tx, err := beginStrategy()
	if err != nil {
		log.Warning("failed to begin scoped transaction: %s", err)
		return err
	}

	defer func() {
		if tx != nil && !tx.Closed() {
			if err := tx.Close(); err != nil {
				log.Warning(
					"close scoped database transaction failed with error: %s",
					err,
				)
			}
		}
	}()

	if err = tryToCommit(fn)(tx); err != nil {
		log.Debug("scoped database transaction failed with error: %s", err)
	}
	return err
}

func WithinTemplate(
	retries int,
	retryStrategy func() time.Duration,
	beginStrategy func() (ITransaction, error),
	fn func(ITransaction) error,
) error {
	var err error
	for attempt := 0; attempt <= retries; attempt++ {
		if err = tryWithinTx(beginStrategy, fn); err == nil || !IsDeadlock(err) {
			return err
		}
		retryInterval := GetRetryInterval(retryStrategy())
		log.Warning(
			"scoped transaction deadlocked, retrying %d / %d, after %dms",
			attempt,
			retries,
			retryInterval.Nanoseconds()/int64(time.Millisecond),
		)
		if retryInterval > 0 {
			time.Sleep(retryInterval)
		}
	}
	log.Warning(
		"scoped transaction still deadlocked after %d retries; gave up",
		retries,
	)
	return err
}

func GetRetryInterval(retryInterval time.Duration) time.Duration {
	if retryInterval > 0 {
		// Add random duration between [0, interval] to decrease collision chance
		return retryInterval + time.Duration(rand.Intn(int(retryInterval.Nanoseconds())))
	}
	return 0
}

func withinDatabase(
	db DB,
	fn func(transaction.Transaction) error,
	beginStrategy func() (ITransaction, error),
) error {
	return WithinTemplate(
		db.Options().RetryTxCount,
		func() time.Duration {
			return db.Options().RetryTxInterval
		},
		beginStrategy,
		func(tx ITransaction) error {
			return fn(tx.(transaction.Transaction))
		},
	)
}

// Within executes a scoped transaction on a database
func Within(db DB, fn func(transaction.Transaction) error) error {
	return withinDatabase(db, fn, func() (ITransaction, error) {
		return db.Begin()
	})
}

// WithinTx executes a scoped transaction with options on a database
func WithinTx(ctx context.Context, db DB, options *transaction.TxOptions, fn func(transaction.Transaction) error) error {
	return withinDatabase(db, fn, func() (ITransaction, error) {
		return db.BeginTx(ctx, options)
	})
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
			return nil
		}); errorOutputTx != nil {
			return errorOutputTx
		}
		return nil
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

type InitDBParams struct {
	DropOnCreate, Cascade, AutoMigrate, AllowEmpty bool
}

func DefaultTestInitDBParams() InitDBParams {
	return InitDBParams{
		DropOnCreate: true, // always drop DB during tests
		Cascade:      false,
		AutoMigrate:  false, // do not migrate during tests
		AllowEmpty:   true,  // allow tests to run without schemas
	}
}

// InitDBConnWithSchemas initializes database connection using schemas stored in Manager
func InitDBConnWithSchemas(aDb DB, initDBParams InitDBParams) error {
	var err error
	schemaManager := schema.GetManager()
	schemas := schemaManager.OrderedSchemas()
	if len(schemas) == 0 && !initDBParams.AllowEmpty {
		return errNoSchemasInManager
	}
	if initDBParams.DropOnCreate {
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
		err = aDb.RegisterTable(s, initDBParams.Cascade, initDBParams.AutoMigrate)
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
func InitDBWithSchemas(dbType, dbConnection string, initDBParams InitDBParams) error {
	aDb, err := ConnectDB(dbType, dbConnection, DefaultMaxOpenConn, options.Default())
	if err != nil {
		return err
	}
	defer aDb.Close()
	return InitDBConnWithSchemas(aDb, initDBParams)
}
