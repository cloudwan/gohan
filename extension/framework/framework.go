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
	"os"
	"path/filepath"
	"regexp"

	"github.com/cloudwan/gohan/extension/framework/buflog"
	"github.com/cloudwan/gohan/extension/framework/runner"
	gohan_log "github.com/cloudwan/gohan/log"
	"github.com/cloudwan/gohan/util"
	"github.com/codegangsta/cli"
	logging "github.com/op/go-logging"
)

var log = logging.MustGetLogger("extest")

// TestExtensions runs extension tests when invoked from Gohan CLI
func TestExtensions(c *cli.Context) {
	buflog.SetUpDefaultLogging()

	var config *util.Config
	configFilePath := c.String("config-file")

	if configFilePath != "" && !c.Bool("verbose") {
		config = util.GetConfig()
		err := config.ReadConfig(configFilePath)
		if err != nil {
			log.Error(fmt.Sprintf("Failed to read config from path %s: %v", configFilePath, err))
			os.Exit(1)
		}

		err = gohan_log.SetUpLogging(config)
		if err != nil {
			log.Error(fmt.Sprintf("Failed to set up logging: %v", err))
			os.Exit(1)
		}
	}

	testFiles := getTestFiles(c.Args())

	//logging from config is a limited printAllLogs option
	returnCode := RunTests(testFiles, c.Bool("verbose") || config != nil)
	os.Exit(returnCode)
}

// RunTests runs extension tests for CLI
func RunTests(testFiles []string, printAllLogs bool) (returnCode int) {
	errors := map[string]map[string]error{}
	for _, testFile := range testFiles {
		testRunner := runner.NewTestRunner(testFile, printAllLogs)
		errors[testFile] = testRunner.Run()
		if err, ok := errors[testFile][runner.GeneralError]; ok {
			log.Error(fmt.Sprintf("\t ERROR (%s): %v", testFile, err))
		}
	}

	summary := makeSummary(errors)
	printSummary(summary, printAllLogs)

	for _, err := range summary {
		if err != nil {
			return 1
		}
	}
	return 0
}

func makeSummary(errors map[string]map[string]error) (summary map[string]error) {
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
	allPassed := true

	if !printAllLogs {
		buflog.Buf().Activate()
		defer func() {
			if !allPassed {
				buflog.Buf().PrintLogs()
			}
			buflog.Buf().Deactivate()
		}()
	}

	log.Info("Run %d test files.", len(summary))
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
