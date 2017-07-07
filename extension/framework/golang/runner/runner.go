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

package runner

import (
	"fmt"
	"path/filepath"
	"plugin"
	"testing"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/extension/goext"
	"github.com/cloudwan/gohan/extension/golang"
	l "github.com/cloudwan/gohan/log"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/server/middleware"
	"github.com/cloudwan/gohan/sync/noop"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var log = l.NewLogger()

type GoTestRunner struct {
	tests []string
}

func NewGoTestRunner(tests []string) *GoTestRunner {
	return &GoTestRunner{
		tests: tests,
	}
}

func (self *GoTestRunner) Run() error {
	log.Notice("Running Go extensions tests")

	t := &testing.T{}

	for _, pluginFileName := range self.tests {
		log.Notice("Loading test: %s", pluginFileName)

		test, err := plugin.Open(pluginFileName)

		if err != nil {
			return err
		}

		dataStore, err := newDBConnection(memoryDbConn("test.db"))
		if err != nil {
			return err
		}

		golangEnv := golang.NewEnvironment("test"+pluginFileName, dataStore, &middleware.FakeIdentity{}, noop.NewSync())
		extEnv := golangEnv.ExtEnvironment()
		pluginPath := filepath.Dir(pluginFileName)

		// Schemas
		schemasFnRaw, err := test.Lookup("Schemas")

		mgr := schema.GetManager()

		if err != nil {
			return fmt.Errorf("Golang extension test does not export Schemas: %s", err)
		}

		schemasFn, ok := schemasFnRaw.(func() []string)

		if !ok {
			log.Error("Invalid signature of Schemas function in golang extension test: %s", pluginFileName)
			return err
		}

		schemas := schemasFn()

		for _, schemaPath := range schemas {
			if err = mgr.LoadSchemaFromFile(pluginPath + "/" + schemaPath); err != nil {
				return fmt.Errorf("Failed to load schema: %s", err)
			}
		}

		// Binary
		binaryFnRaw, err := test.Lookup("Binary")

		if err != nil {
			return err
		}

		binaryFn, ok := binaryFnRaw.(func() string)

		if !ok {
			log.Error("Invalid signature of Binary function in golang extension test: %s", pluginFileName)
			return err
		}

		binary := binaryFn()
		golangEnv.Load(binary, "")

		// DB
		err = db.InitDBWithSchemas("sqlite3", memoryDbConn("test.db"), true, false, false)

		if err != nil {
			schema.ClearManager()
			return fmt.Errorf("Failed to init DB: %s", err)
		}

		//Setup test suite
		testFnRaw, err := test.Lookup("Test")

		if err != nil {
			return err
		}

		testFn, ok := testFnRaw.(func(*goext.Environment))

		if !ok {
			log.Error("Invalid signature of Test function in golang extension test: %s", pluginFileName)
			return err
		}
		
		// Run test
		testFn(&extEnv)
	}

	RegisterFailHandler(Fail)
	RunSpecs(t, "Go Extensions Test Suite")

	return nil
}

func memoryDbConn(name string) string {
	return fmt.Sprintf("file:%s?mode=memory&cache=shared", name)
}

func newDBConnection(dbfilename string) (db.DB, error) {
	connection, err := db.ConnectDB("sqlite3", dbfilename, db.DefaultMaxOpenConn)
	if err != nil {
		return nil, err
	}
	return connection, nil
}
