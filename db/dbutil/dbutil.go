// Copyright (C) 2018 NTT Innovation Institute, Inc.
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

package dbutil

import (
	"context"
	"errors"
	"strings"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/initializer"
	"github.com/cloudwan/gohan/db/options"
	"github.com/cloudwan/gohan/db/sql"
	"github.com/cloudwan/gohan/db/transaction"
	l "github.com/cloudwan/gohan/log"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/util"
)

var (
	log                   = l.NewLogger()
	errNoSchemasInManager = errors.New("No schemas in Manager. Did you remember to load them?")
)

//ConnectDB is builder function of DB
func ConnectDB(dbType, conn string, maxOpenConn int, opt options.Options) (db.DB, error) {
	db := sql.NewDB(opt)
	err := db.Connect(dbType, conn, maxOpenConn)
	if err != nil {
		return nil, err
	}
	return db, nil
}

//CopyDBResources copies resources from input database to output database
func CopyDBResources(input *initializer.Initializer, output db.DB, overrideExisting bool) error {
	schemaManager := schema.GetManager()
	schemas := schemaManager.OrderedSchemas()

	if len(schemas) == 0 {
		return errNoSchemasInManager
	}

	ctx := context.TODO() //pass from outside?

	return db.WithinTx(output, func(outputTx transaction.Transaction) error {
		for _, schema := range schemas {
			if schema.IsAbstract() {
				continue
			}
			log.Info("Populating resources for schema %s", schema.ID)
			resources, _, err := input.List(schema)
			if err != nil {
				return err
			}

			for _, resource := range resources {
				log.Info("Creating resource %s", resource.ID())
				destResource, _ := outputTx.Fetch(ctx, schema, transaction.IDFilter(resource.ID()), nil)
				if destResource == nil {
					resource.PopulateDefaults()
					_, err := outputTx.Create(ctx, resource)
					if err != nil {
						return err
					}
				} else if overrideExisting {
					err := outputTx.Update(ctx, resource)
					if err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
}

// CreateFromConfig creates db connection from config
func CreateFromConfig(config *util.Config) (db.DB, error) {
	dbType := config.GetString("database/type", "sqlite3")
	dbConnection := config.GetString("database/connection", "")
	maxConn := config.GetInt("database/max_open_conn", db.DefaultMaxOpenConn)
	dbOptions := options.Read(config)

	dbConn := sql.NewDB(dbOptions)
	err := dbConn.Connect(dbType, dbConnection, maxConn)
	if err != nil {
		return nil, err
	}
	return dbConn, nil
}

// InitDBConnWithSchemas initializes database connection using schemas stored in Manager
func InitDBConnWithSchemas(aDb db.DB, initDBParams db.InitDBParams) error {
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
func InitDBWithSchemas(dbType, dbConnection string, initDBParams db.InitDBParams) error {
	aDb, err := ConnectDB(dbType, dbConnection, db.DefaultMaxOpenConn, options.Default())
	if err != nil {
		return err
	}
	defer aDb.Close()
	return InitDBConnWithSchemas(aDb, initDBParams)
}

func ClearTable(ctx context.Context, tx transaction.Transaction, s *schema.Schema) error {
	if s.IsAbstract() {
		return nil
	}
	for _, schema := range schema.GetManager().Schemas() {
		if schema.ParentSchema == s {
			err := ClearTable(ctx, tx, schema)
			if err != nil {
				return err
			}
		} else {
			for _, property := range schema.Properties {
				if property.Relation == s.Singular {
					err := ClearTable(ctx, tx, schema)
					if err != nil {
						return err
					}
				}
			}
		}
	}
	resources, _, err := tx.List(ctx, s, nil, nil, nil)
	if err != nil {
		return err
	}
	for _, resource := range resources {
		err = tx.Delete(ctx, s, resource.ID())
		if err != nil {
			return err
		}
	}
	return nil
}
