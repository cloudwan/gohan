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
	"regexp"
	"testing"

	gohan_db "github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/options"
	"github.com/cloudwan/gohan/extension"
	"github.com/cloudwan/gohan/extension/goext"
	"github.com/cloudwan/gohan/extension/goplugin"
	gohan_logger "github.com/cloudwan/gohan/log"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/sync/noop"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/twinj/uuid"
)

const (
	dbBaseFileName = "test.db"
)

var log = gohan_logger.NewLogger()

// TestRunner is a test runner for go extensions
type TestRunner struct {
	fileNames      []string
	verboseLogs    bool
	fileNameFilter *regexp.Regexp
	workerCount    int
}

// NewTestRunner allocates a new TestRunner
func NewTestRunner(fileNames []string, printAllLogs bool, testFilter string, workers int) *TestRunner {
	return &TestRunner{
		fileNames:      fileNames,
		verboseLogs:    printAllLogs,
		fileNameFilter: regexp.MustCompile(testFilter),
		workerCount:    workers,
	}
}

// Run runs go test runner
func (testRunner *TestRunner) Run() error {
	// configure reporter
	gomega.RegisterFailHandler(ginkgo.Fail)
	t := &testing.T{}
	reporter := NewReporter()

	// run suites
	var err error
	for _, fileName := range testRunner.fileNames {
		if !testRunner.fileNameFilter.MatchString(fileName) {
			continue
		}
		if err = testRunner.runSingle(t, reporter, fileName); err != nil {
			break
		}
	}

	// display report
	reporter.Report()

	if err != nil {
		return err
	}

	if !reporter.AllSuitesSucceed() {
		return fmt.Errorf("tests are failing")
	}

	return nil
}

func readSchemas(p *plugin.Plugin) ([]string, error) {
	fnRaw, err := p.Lookup("Schemas")

	if err != nil {
		return nil, fmt.Errorf("missing 'Schemas' export: %s", err)
	}

	fn, ok := fnRaw.(func() []string)

	if !ok {
		return nil, fmt.Errorf("invalid signature of 'Schemas' export")
	}

	return fn(), nil
}

func readBinaries(p *plugin.Plugin) ([]string, error) {
	fnRaw, err := p.Lookup("Binaries") // optional

	if err != nil {
		return []string{}, nil
	}

	fn, ok := fnRaw.(func() []string)

	if !ok {
		return nil, fmt.Errorf("invalid signature of 'Binaries' export")
	}

	return fn(), nil
}

func readTest(p *plugin.Plugin) (func(goext.MockIEnvironment), error) {
	fnRaw, err := p.Lookup("Test")

	if err != nil {
		return nil, fmt.Errorf("missing 'Test' export: %s", err)
	}

	testFn, ok := fnRaw.(func(goext.MockIEnvironment))

	if !ok {
		return nil, fmt.Errorf("invalid signature of 'Test' export")
	}

	return testFn, nil
}

func (testRunner *TestRunner) runSingle(t ginkgo.GinkgoTestingT, reporter *Reporter, fileName string) error {
	log.Notice("Running Go extensions test: %s", fileName)

	// inform reporter about test suite
	reporter.Prepare(fileName)

	// load plugin
	p, err := plugin.Open(fileName)

	if err != nil {
		return fmt.Errorf("failed to open plugin: %s", err)
	}

	// read schemas
	schemas, err := readSchemas(p)

	if err != nil {
		return fmt.Errorf("failed to read schemas from: %s", err)
	}

	// get state
	path := filepath.Dir(fileName)
	manager := schema.GetManager()

	// load schemas
	for _, schemaPath := range schemas {
		if err = manager.LoadSchemaFromFile(path + "/" + schemaPath); err != nil {
			return fmt.Errorf("failed to load schema: %s", err)
		}
	}

	// get binaries
	binaries, err := readBinaries(p)

	if err != nil {
		return fmt.Errorf("failed to read binaries: %s", err)
	}

	// create env
	beforeStartHook := func(env *goplugin.Environment) error {
		// db
		dbFileName := dbBaseFileName + "_" + uuid.NewV4().String()
		dbConnString := fmt.Sprintf("file:%s?mode=memory&cache=shared", dbFileName)
		db, err := gohan_db.ConnectDB("sqlite3", dbConnString, gohan_db.DefaultMaxOpenConn, options.Default())

		if err != nil {
			return fmt.Errorf("failed to connect db: %s", err)
		}

		if err = gohan_db.InitDBConnWithSchemas(db, true, false, false); err != nil {
			return fmt.Errorf("failed to init db: %s", err)
		}

		env.SetDatabase(db)

		// sync
		sync := noop.NewSync()
		env.SetSync(sync)

		return nil
	}

	// create env
	envName := "Go test environment"

	env := goplugin.NewEnvironment(envName, beforeStartHook, nil)
	mockEnv := goplugin.NewMockIEnvironment(env, ginkgo.GinkgoT())

	// register
	extensionManager := extension.GetManager()
	for schemaID := range manager.Schemas() {
		if err := extensionManager.RegisterEnvironment(schemaID, mockEnv); err != nil {
			return fmt.Errorf("failed to register environment: %s", err)
		}
	}

	// load binaries
	for _, binary := range binaries {
		if err := env.Load(path + "/" + binary); err != nil {
			return fmt.Errorf("failed to load binary: %s", err)
		}
	}

	// load extensions
	if err := env.LoadExtensionsForPath(manager.Extensions, manager.TimeLimit, manager.TimeLimits, ""); err != nil {
		return fmt.Errorf("failed to load schemas extensions: %s", err)
	}

	// get test
	test, err := readTest(p)

	if err != nil {
		return fmt.Errorf("failed to read test: %s", err)
	}

	// prepare test
	mockEnv.Reset() // TODO remove
	test(mockEnv)
	mockEnv.Reset()

	// run test
	ginkgo.RunSpecsWithCustomReporters(t, fileName, []ginkgo.Reporter{reporter})
	ginkgo.Reset()

	// stop env
	env.Stop()

	// unregister
	for schemaID := range manager.Schemas() {
		if err := extensionManager.UnRegisterEnvironment(schemaID); err != nil {
			return fmt.Errorf("failed to unregister schema %s", err)
		}
	}

	// clear state
	manager.ClearExtensions()
	schema.ClearManager()

	log.Notice("Go extension test finished: %s", fileName)

	return nil
}
