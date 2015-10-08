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
)

const noSchemasInManagerError = "No schemas in Manager. Did you remember to load them?"

//DB is a common interface for handing db
type DB interface {
	Connect(string, string) error
	Begin() (transaction.Transaction, error)
	RegisterTable(*schema.Schema, bool) error
	DropTable(*schema.Schema) error
}

//ConnectDB is builder function of DB
func ConnectDB(dbType, conn string) (DB, error) {
	var db DB
	if dbType == "json" || dbType == "yaml" {
		db = file.NewDB()
	} else {
		db = sql.NewDB()
	}
	err := db.Connect(dbType, conn)
	if err != nil {
		return nil, err
	}
	return db, nil
}

//CopyDBResources copies resources from input database to output database
func CopyDBResources(input, output DB) error {
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
		log.Info("Populating resources for schema %s", s.ID)
		resources, _, err := itx.List(s, nil, nil)
		if err != nil {
			return err
		}

		for _, resource := range resources {
			log.Info("Creating resource %s", resource.ID())
			destResource, _ := otx.Fetch(s, resource.ID(), nil)
			if destResource == nil {
				err := otx.Create(resource)
				if err != nil {
					return err
				}
			} else {
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

//InitDBWithSchemas initializes database using schemas stored in Manager
func InitDBWithSchemas(dbType, dbConnection string, dropOnCreate, cascade bool) error {
	aDb, err := ConnectDB(dbType, dbConnection)
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
			log.Debug("Dropping table '%s'", s.Plural)
			err = aDb.DropTable(s)
			if err != nil {
				log.Fatal("Error during deleting table:", err.Error())
			}
		}
	}
	for _, s := range schemas {
		log.Debug("Registering schema %s", s.ID)
		err = aDb.RegisterTable(s, cascade)
		if err != nil {
			message := "Error during registering table: %s"
			if strings.Contains(err.Error(), "already exists") {
				log.Warning(message, err.Error())
			} else {
				log.Fatal(message, err.Error())
			}
		}
	}
	return nil
}
