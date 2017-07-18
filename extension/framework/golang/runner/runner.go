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
	env     *golang.Environment
	path    string
	manager *schema.Manager

	schemasFnRaw plugin.Symbol
	schemasFn    func() []string
	schemas      []string

	binaryFnRaw plugin.Symbol
	binaryFn    func() string
	binary      string

	testFnRaw plugin.Symbol
	testFn    func(goext.IEnvironment)
}

func (goTestRunner *GoTestRunner) Run() error {
	log.Notice("Running Go extensions tests")

	// note: must hold refs to go tests so that they are available at the time of being run by ginkgo
	goTestSuites := []*GoTestSuite{}

	for _, pluginFileName := range goTestRunner.pluginFileNames {
		log.Notice("Loading test: %s", pluginFileName)

		var err error
		var ok, loaded bool
		goTestSuite := &GoTestSuite{}

		goTestSuite.plugin, err = plugin.Open(pluginFileName)

		if err != nil {
			return err
		}

		goTestSuite.db, err = newDBConnection(memoryDbConn("test.db"))
		if err != nil {
			return err
		}

		goTestSuite.env = golang.NewEnvironment("test"+pluginFileName, goTestSuite.db, &middleware.FakeIdentity{}, noop.NewSync())
		goTestSuite.path = filepath.Dir(pluginFileName)

		goTestSuite.manager = schema.GetManager()

		// Schemas
		goTestSuite.schemasFnRaw, err = goTestSuite.plugin.Lookup("Schemas")

		if err != nil {
			return fmt.Errorf("golang extension test does not export Schemas: %s", err)
		}

		goTestSuite.schemasFn, ok = goTestSuite.schemasFnRaw.(func() []string)

		if !ok {
			log.Error("invalid signature of Schemas function in golang extension test: %s", pluginFileName)
			return err
		}

		goTestSuite.schemas = goTestSuite.schemasFn()

		// Binary
		goTestSuite.binaryFnRaw, err = goTestSuite.plugin.Lookup("Binary")

		if err != nil {
			return err
		}

		goTestSuite.binaryFn, ok = goTestSuite.binaryFnRaw.(func() string)

		if !ok {
			log.Error("invalid signature of Binary function in golang extension test: %s", pluginFileName)
			return err
		}

		goTestSuite.binary = goTestSuite.binaryFn()

		loaded, err = goTestSuite.env.Load(goTestSuite.path + "/" + goTestSuite.binary, func() error {
			// initial schemas
			for _, schemaPath := range goTestSuite.schemas {
				if err = goTestSuite.manager.LoadSchemaFromFile(goTestSuite.path + "/" + schemaPath); err != nil {
					return fmt.Errorf("failed to load schema: %s", err)
				}
			}

			// DB
			err = db.InitDBWithSchemas("sqlite3", memoryDbConn("test.db"), true, true, false)

			if err != nil {
				return fmt.Errorf("failed to init DB: %s", err)
			}

			return nil
		})

		if err != nil {
			log.Error("failed to load golang extension test dependant plugin: %s; error: %s", pluginFileName, err)
			return err
		}

		if loaded {
			err = goTestSuite.env.Start()

			if err != nil {
				log.Error("failed to load start extension test dependant plugin: %s; error: %s", pluginFileName, err)
				return err
			}
		}

		// Setup test suite
		goTestSuite.testFnRaw, err = goTestSuite.plugin.Lookup("Test")

		if err != nil {
			return err
		}

		goTestSuite.testFn, ok = goTestSuite.testFnRaw.(func(goext.IEnvironment))

		if !ok {
			log.Error("invalid signature of Test function in golang extension test: %s", pluginFileName)
			return err
		}

		// Hold a reference to test
		goTestSuites = append(goTestSuites, goTestSuite)

		// Run test
		goTestSuite.testFn(goTestSuite.env)
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
