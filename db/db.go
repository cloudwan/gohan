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
	"fmt"
	"strings"

	"github.com/cloudwan/gohan/db/file"
	"github.com/cloudwan/gohan/db/sql"
	"github.com/cloudwan/gohan/db/transaction"

	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/util"
)

//DefaultMaxOpenConn will applied for db object
const DefaultMaxOpenConn = 100

const noSchemasInManagerError = "No schemas in Manager. Did you remember to load them?"

//DB is a common interface for handing db
type DB interface {
	Connect(string, string, int) error
	Close()
	Begin() (transaction.Transaction, error)
	RegisterTable(s *schema.Schema, cascade, migrate bool) error
	DropTable(*schema.Schema) error
}

//ConnectDB is builder function of DB
func ConnectDB(dbType, conn string, maxOpenConn int) (DB, error) {
	var db DB
	if dbType == "json" || dbType == "yaml" {
		db = file.NewDB()
	} else {
		db = sql.NewDB()
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
		return fmt.Errorf(noSchemasInManagerError)
	}

	itx, err := input.Begin()
	if err != nil {
		return err
	}
	defer itx.Close()

	otx, err := output.Begin()
	if err != nil {
		return err
	}
	defer otx.Close()

	for _, s := range schemas {
		if s.IsAbstract() {
			continue
		}
		log.Info("Populating resources for schema %s", s.ID)
		resources, _, err := itx.List(s, nil, nil)
		if err != nil {
			return err
		}

		for _, resource := range resources {
			log.Info("Creating resource %s", resource.ID())
			destResource, _ := otx.Fetch(s, transaction.IDFilter(resource.ID()))
			if destResource == nil {
				resource.PopulateDefaults()
				err := otx.Create(resource)
				if err != nil {
					return err
				}
			} else if overrideExisting {
				err := otx.Update(resource)
				if err != nil {
					return err
				}
			}
		}
	}
	err = itx.Commit()
	if err != nil {
		return err
	}
	return otx.Commit()
}

func CreateFromConfig(config *util.Config) (DB, error) {
	dbType := config.GetString("database/type", "sqlite3")
	dbConnection := config.GetString("database/connection", "")
	maxConn := config.GetInt("database/max_open_conn", DefaultMaxOpenConn)
	var dbConn DB
	if dbType == "json" || dbType == "yaml" {
		dbConn = file.NewDB()
	} else {
		dbConn = sql.NewDB()
	}
	err := dbConn.Connect(dbType, dbConnection, maxConn)
	if err != nil {
		return nil, err
	}
	return dbConn, nil
}

//InitDBWithSchemas initializes database using schemas stored in Manager
func InitDBWithSchemas(dbType, dbConnection string, dropOnCreate, cascade, migrate bool) error {
	aDb, err := ConnectDB(dbType, dbConnection, DefaultMaxOpenConn)
	if err != nil {
		return err
	}
	schemaManager := schema.GetManager()
	schemas := schemaManager.OrderedSchemas()
	if len(schemas) == 0 {
		return fmt.Errorf(noSchemasInManagerError)
	}
	if dropOnCreate {
		for i := len(schemas) - 1; i >= 0; i-- {
			s := schemas[i]
			if s.IsAbstract() {
				continue
			}
			log.Debug("Dropping table '%s'", s.Plural)
			err = aDb.DropTable(s)
			if err != nil {
				log.Fatal("Error during deleting table:", err.Error())
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
	aDb.Close()
	return nil
}
