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
	logPkg "github.com/cloudwan/gohan/log"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/server/middleware"
	"github.com/cloudwan/gohan/sync/noop"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var log = logPkg.NewLogger()

type GoTestRunner struct {
	pluginFileNames []string
}

func NewGoTestRunner(pluginFileNames []string) *GoTestRunner {
	return &GoTestRunner{
		pluginFileNames: pluginFileNames,
	}
}

type GoTestSuite struct {
	plugin  *plugin.Plugin
	db      db.DB
	goEnv   *golang.Environment
	extEnv  goext.Environment
	path    string
	manager *schema.Manager

	schemasFnRaw plugin.Symbol
	schemasFn    func() [] string
	schemas      []string

	binaryFnRaw plugin.Symbol
	binaryFn    func() string
	binary      string

	testFnRaw plugin.Symbol
	testFn    func(*goext.Environment)
}

func (goTestRunner *GoTestRunner) Run() error {
	log.Notice("Running Go extensions tests")

	// note: must hold refs to go tests so that they are available at the time of being run by ginkgo
	goTestSuites := []*GoTestSuite{}

	for _, pluginFileName := range goTestRunner.pluginFileNames {
		log.Notice("Loading test: %s", pluginFileName)

		var err error
		var ok bool
		goTestSuite := &GoTestSuite{}

		goTestSuite.plugin, err = plugin.Open(pluginFileName)

		if err != nil {
			return err
		}

		goTestSuite.db, err = newDBConnection(memoryDbConn("test.db"))
		if err != nil {
			return err
		}

		goTestSuite.goEnv = golang.NewEnvironment("test"+pluginFileName, goTestSuite.db, &middleware.FakeIdentity{}, noop.NewSync())
		goTestSuite.extEnv = goTestSuite.goEnv.ExtEnvironment()
		goTestSuite.path = filepath.Dir(pluginFileName)

		// Schemas
		goTestSuite.schemasFnRaw, err = goTestSuite.plugin.Lookup("Schemas")

		goTestSuite.manager = schema.GetManager()

		if err != nil {
			return fmt.Errorf("Golang extension test does not export Schemas: %s", err)
		}

		goTestSuite.schemasFn, ok = goTestSuite.schemasFnRaw.(func() []string)

		if !ok {
			log.Error("Invalid signature of Schemas function in golang extension test: %s", pluginFileName)
			return err
		}

		goTestSuite.schemas = goTestSuite.schemasFn()

		for _, schemaPath := range goTestSuite.schemas {
			if err = goTestSuite.manager.LoadSchemaFromFile(goTestSuite.path + "/" + schemaPath); err != nil {
				return fmt.Errorf("Failed to load schema: %s", err)
			}
		}

		// Binary
		goTestSuite.binaryFnRaw, err = goTestSuite.plugin.Lookup("Binary")

		if err != nil {
			return err
		}

		goTestSuite.binaryFn, ok = goTestSuite.binaryFnRaw.(func() string)

		if !ok {
			log.Error("Invalid signature of Binary function in golang extension test: %s", pluginFileName)
			return err
		}

		goTestSuite.binary = goTestSuite.binaryFn()
		goTestSuite.goEnv.Load(goTestSuite.binary, "")

		// DB
		err = db.InitDBWithSchemas("sqlite3", memoryDbConn("test.db"), true, false, false)

		if err != nil {
			schema.ClearManager()
			return fmt.Errorf("Failed to init DB: %s", err)
		}

		//Setup test suite
		goTestSuite.testFnRaw, err = goTestSuite.plugin.Lookup("Test")

		if err != nil {
			return err
		}

		goTestSuite.testFn, ok = goTestSuite.testFnRaw.(func(*goext.Environment))

		if !ok {
			log.Error("Invalid signature of Test function in golang extension test: %s", pluginFileName)
			return err
		}

		// Hold a reference to test
		goTestSuites = append(goTestSuites, goTestSuite)

		// Run test
		goTestSuite.testFn(&goTestSuite.extEnv)
	}

	RegisterFailHandler(Fail)

	t := &testing.T{}
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
