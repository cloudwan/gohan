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

package framework

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"sync/atomic"

	"github.com/cloudwan/gohan/extension/framework/runner"
	l "github.com/cloudwan/gohan/log"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/util"
	"github.com/codegangsta/cli"
)

var (
	logWriter io.Writer = os.Stderr
	log                 = l.NewLoggerForModule("extest")
)

// TestExtensions runs extension tests when invoked from Gohan CLI
func TestExtensions(c *cli.Context) {
	l.SetUpBasicLogging(logWriter, l.DefaultFormat, "", l.DEBUG)

	var config *util.Config
	configFilePath := c.String("config-file")

	if configFilePath != "" && !c.Bool("verbose") {
		config = util.GetConfig()
		err := config.ReadConfig(configFilePath)
		if err != nil {
			log.Error(fmt.Sprintf("Failed to read config from path %s: %v", configFilePath, err))
			os.Exit(1)
		}

		err = l.SetUpLogging(config)
		if err != nil {
			log.Error(fmt.Sprintf("Failed to set up logging: %v", err))
			os.Exit(1)
		}
	}

	testFiles := getTestFiles(c.Args())

	//logging from config is a limited printAllLogs option
	returnCode := RunTests(testFiles, c.Bool("verbose") || config != nil, c.String("run-test"), c.Int("parallel"))
	os.Exit(returnCode)
}

// RunTests runs extension tests for CLI.
func RunTests(testFiles []string, printAllLogs bool, testFilter string, workers int) (returnCode int) {
	if !printAllLogs {
		l.SetUpBasicLogging(l.BufWritter{}, l.DefaultFormat, "", l.DEBUG)
	}

	if workers <= 0 {
		panic("Workers must be greater than 0")
	}

	if len(testFiles) < workers {
		workers = len(testFiles)
	}

	var (
		maxIdx         = int64(len(testFiles) - 1)
		idx      int64 = -1
		errors         = make(map[string]runner.TestRunnerErrors)
		errorsMu sync.Mutex
		wg       sync.WaitGroup
	)

	worker := func() {
		for {
			i := atomic.AddInt64(&idx, 1)
			if i > maxIdx {
				break
			}

			fileName := testFiles[i]
			testErr := runner.NewTestRunner(fileName, printAllLogs, testFilter).Run()

			errorsMu.Lock()
			errors[fileName] = testErr
			errorsMu.Unlock()

			if err, ok := testErr[runner.GeneralError]; ok {
				log.Error(fmt.Sprintf("\t ERROR (%s): %v", fileName, err))
			}
		}
		wg.Done()
	}

	// force goroutine local manager
	schema.SetManagerScope(schema.ScopeGLSSingleton)

	for workers > 0 {
		wg.Add(1)
		go worker()
		workers -= 1
	}
	wg.Wait()

	summary := makeSummary(errors)
	printSummary(summary, printAllLogs)

	for _, err := range summary {
		if err != nil {
			return 1
		}
	}
	return 0
}

func makeSummary(errors map[string]runner.TestRunnerErrors) (summary map[string]error) {
	summary = map[string]error{}
	for testFile, errors := range errors {
		if err, ok := errors[runner.GeneralError]; ok {
			summary[testFile] = err
			continue
		}

		failed := 0
		for _, err := range errors {
			if err != nil {
				failed++
			}
		}
		summary[testFile] = nil
		if failed > 0 {
			summary[testFile] = fmt.Errorf("%d/%d tests failed", failed, len(errors))
		}
	}
	return
}

func printSummary(summary map[string]error, printAllLogs bool) {
	l.SetUpBasicLogging(logWriter, l.DefaultFormat, "", l.DEBUG)

	log.Info("Run %d test files.", len(summary))

	allPassed := true
	for testFile, err := range summary {
		if err != nil {
			log.Error(fmt.Sprintf("\tFAIL\t%s: %s", testFile, err.Error()))
			allPassed = false
		} else if printAllLogs {
			log.Notice("\tOK\t%s", testFile)
		}
	}
	if allPassed {
		log.Notice("All tests have passed.")
	}
}

func getTestFiles(args cli.Args) []string {
	paths := args
	if len(paths) == 0 {
		paths = append(paths, ".")
	}

	pattern := regexp.MustCompile(`^test_.*\.js$`)
	seen := map[string]bool{}
	testFiles := []string{}
	for _, path := range paths {
		filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				log.Error(fmt.Sprintf("Failed to process '%s': %s", filePath, err.Error()))
				return nil
			}

			if info.IsDir() {
				return nil
			}

			if !pattern.MatchString(info.Name()) {
				return nil
			}

			filePath = filepath.Clean(filePath)

			if !seen[filePath] {
				testFiles = append(testFiles, filePath)
				seen[filePath] = true
			}

			return nil
		})
	}

	return testFiles
}
