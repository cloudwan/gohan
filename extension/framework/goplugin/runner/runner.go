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

	"regexp"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/options"
	"github.com/cloudwan/gohan/extension"
	"github.com/cloudwan/gohan/extension/goext"
	"github.com/cloudwan/gohan/extension/goplugin"
	pkgLog "github.com/cloudwan/gohan/log"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/server/middleware"
	"github.com/cloudwan/gohan/sync/noop"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
)

const (
	testDBFile = "test.db"
)

var log = pkgLog.NewLogger()

// GoTestRunner is a test runner for go (plugin) extensions
type GoTestRunner struct {
	pluginFileNames []string
	printAllLogs    bool
	testFilter      *regexp.Regexp
	workers         int
}

func dbConnString(name string) string {
	return fmt.Sprintf("file:%s?mode=memory&cache=shared", name)
}

func dbConnect(connString string) (db.DB, error) {
	conn, err := db.ConnectDB("sqlite3", connString, db.DefaultMaxOpenConn, options.Default())
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// NewGoTestRunner allocates a new GoTestRunner
func NewGoTestRunner(pluginFileNames []string, printAllLogs bool, testFilter string, workers int) *GoTestRunner {
	return &GoTestRunner{
		pluginFileNames: pluginFileNames,
		printAllLogs:    printAllLogs,
		testFilter:      regexp.MustCompile(testFilter),
		workers:         workers,
	}
}

// GoTestSuite is a test suite state for a go (plugin) extension runner
type GoTestSuite struct {
	plugin  *plugin.Plugin
	db      db.DB
	env     *goplugin.Environment
	path    string
	manager *schema.Manager

	schemasFnRaw plugin.Symbol
	schemasFn    func() []string
	schemas      []string

	binariesFnRaw plugin.Symbol
	binariesFn    func() []string
	binaries      []string

	testFnRaw plugin.Symbol
	testFn    func(goext.IEnvironment)
}

// Run runs go (plugin) test runner
func (goTestRunner *GoTestRunner) Run() error {
	log.Notice("Running Go extensions tests")

	// note: must hold refs to go tests so that they are available at the time of being run by ginkgo
	goTestSuites := []*GoTestSuite{}

	for _, pluginFileName := range goTestRunner.pluginFileNames {
		if !goTestRunner.testFilter.MatchString(pluginFileName) {
			continue
		}

		log.Notice("Loading test: %s", pluginFileName)

		var err error
		var ok, loaded bool
		goTestSuite := &GoTestSuite{}

		goTestSuite.plugin, err = plugin.Open(pluginFileName)

		if err != nil {
			return err
		}

		goTestSuite.db, err = dbConnect(dbConnString(testDBFile))
		if err != nil {
			return err
		}

		goTestSuite.env = goplugin.NewEnvironment("test"+pluginFileName, goTestSuite.db, &middleware.FakeIdentity{}, noop.NewSync())
		manager := extension.GetManager()
		if err := manager.RegisterEnvironment(pluginFileName, goTestSuite.env); err != nil {
			return err
		}
		goTestSuite.path = filepath.Dir(pluginFileName)

		goTestSuite.manager = schema.GetManager()

		// Get schemas
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

		// Load schemas
		for _, schemaPath := range goTestSuite.schemas {
			if err = goTestSuite.manager.LoadSchemaFromFile(goTestSuite.path + "/" + schemaPath); err != nil {
				return fmt.Errorf("failed to load schema: %s", err)
			}
		}

		// Binaries
		goTestSuite.binariesFnRaw, err = goTestSuite.plugin.Lookup("Binaries")

		if err != nil {
			return err
		}

		goTestSuite.binariesFn, ok = goTestSuite.binariesFnRaw.(func() []string)

		if !ok {
			log.Error("invalid signature of Binary function in golang extension test: %s", pluginFileName)
			return err
		}

		goTestSuite.binaries = goTestSuite.binariesFn()

		for _, binary := range goTestSuite.binaries {
			loaded, err = goTestSuite.env.Load(goTestSuite.path+"/"+binary, func() error {
				// reset DB
				err = db.InitDBWithSchemas("sqlite3", dbConnString(testDBFile), true, false, false)

				if err != nil {
					return fmt.Errorf("failed to init DB: %s", err)
				}

				return nil
			})

			if err != nil {
				log.Error("failed to load golang extension test dependant plugin: %s; error: %s", pluginFileName, err)
				return err
			}
		}

		if loaded {
			err = goTestSuite.env.Start()

			if err != nil {
				log.Error("failed to start extension test dependant plugin: %s; error: %s", pluginFileName, err)
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

	gomega.RegisterFailHandler(ginkgo.Fail)

	t := &testing.T{}
	ginkgo.RunSpecs(t, "Go Extensions Test Suite")

	return nil
}
