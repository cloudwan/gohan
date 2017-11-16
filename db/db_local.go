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

package db

import (
	"fmt"
	"os"

	"github.com/cloudwan/gohan/db/options"
	"github.com/twinj/uuid"
)

const (
	dbMaxOpenConn        = 100
	dbSQLiteBaseFileName = "test.db"
	dbDefaultMySQLDSN    = "root@/gohan_test"
)

func ConnectLocalSQLite3() (DB, error) {
	dbType := "sqlite3"
	dbFileName := dbSQLiteBaseFileName + "_" + uuid.NewV4().String()
	dbConn := fmt.Sprintf("file:%s?mode=memory&cache=shared", dbFileName)
	return Connect(dbType, dbConn, dbMaxOpenConn, options.Default())
}

func ConnectLocalMySQL() (DB, error) {
	dbConn := ""
	dbType := "mysql"
	if dsn, ok := os.LookupEnv("MYSQL_DSN"); ok {
		dbConn = dsn
	} else {
		dbConn = dbDefaultMySQLDSN
	}
	return Connect(dbType, dbConn, dbMaxOpenConn, options.Default())
}

func ConnectLocal() (DB, error) {
	if os.Getenv("MYSQL_TEST") == "true" {
		return ConnectLocalMySQL()
	}
	return ConnectLocalSQLite3()
}

func MustConnectLocalSQLite3() DB {
	db, err := ConnectLocalSQLite3()
	if err != nil {
		panic(fmt.Errorf("failed to connect to local SQLite3 DB: %s", err))
	}
	return db
}

func MustConnectLocalMySQL() DB {
	db, err := ConnectLocalMySQL()
	if err != nil {
		panic(fmt.Errorf("failed to connect to local MySQL DB: %s", err))
	}
	return db
}

func MustConnectLocal() DB {
	db, err := ConnectLocal()
	if err != nil {
		panic(fmt.Errorf("failed to connect to local DB: %s", err))
	}
	return db
}
