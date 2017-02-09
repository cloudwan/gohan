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

package runner

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"

	"github.com/robertkrimen/otto"
	"github.com/robertkrimen/otto/ast"
	"github.com/robertkrimen/otto/parser"

	l "github.com/cloudwan/gohan/log"
)

const (
	// GeneralError denotes runner error not related to tests failures
	GeneralError = ""
)

// TestRunner abstracts running extension tests from a single file
type TestRunner struct {
	testFileName string
	printAllLogs bool
	testFilter   *regexp.Regexp

	setUp    bool
	tearDown bool
}

// TestRunnerErrors map[testFunction]error
type TestRunnerErrors map[string]error

type metaError struct {
	error
}

var setUpPattern = regexp.MustCompile("^setUp$")
var tearDownPattern = regexp.MustCompile("^tearDown$")
var testPattern = regexp.MustCompile("^test.*")

// NewTestRunner creates a new test runner for a given test file
func NewTestRunner(testFileName string, printAllLogs bool, testFilter string) *TestRunner {
	return &TestRunner{
		testFileName: testFileName,
		printAllLogs: printAllLogs,
		testFilter:   regexp.MustCompile(testFilter),
	}
}

// Run performs extension tests from the file specified at runner's creation
func (runner *TestRunner) Run() TestRunnerErrors {
	src, err := ioutil.ReadFile(runner.testFileName)
	if err != nil {
		return generalError(fmt.Errorf("Failed to read file '%s': %s", runner.testFileName, err.Error()))
	}

	program, err := parser.ParseFile(nil, runner.testFileName, src, 0)
	if err != nil {
		return generalError(fmt.Errorf("Failed to parse file '%s': %s", runner.testFileName, err.Error()))
	}
	tests := []string{}
	for _, declaration := range program.DeclarationList {
		if functionDeclaration, ok := declaration.(*ast.FunctionDeclaration); ok {
			name := functionDeclaration.Function.Name.Name
			switch {
			case setUpPattern.MatchString(name):
				runner.setUp = true
			case tearDownPattern.MatchString(name):
				runner.tearDown = true
			case testPattern.MatchString(name) && runner.testFilter.MatchString(name):
				tests = append(tests, name)
			}
		}
	}

	env := NewEnvironment(runner.testFileName, src)

	errors := TestRunnerErrors{}
	for _, test := range tests {
		errors[test] = runner.runTest(test, env)

		if !runner.printAllLogs {
			w := l.BufWritter{}
			if errors[test] != nil {
				w.Dump(os.Stderr)
			}
			w.Reset()
		}

		if _, ok := errors[test].(metaError); ok {
			return generalError(errors[test])
		}
	}

	return errors
}

func generalError(err error) TestRunnerErrors {
	return TestRunnerErrors{
		GeneralError: err,
	}
}

func (runner *TestRunner) runTest(testName string, env *Environment) (err error) {
	defer func() {
		runner.printTestResult(testName, err)
	}()

	err = env.InitializeEnvironment()
	if err != nil {
		return metaError{err}
	}
	defer env.ClearEnvironment()

	if runner.setUp {
		_, err = env.VM.Call("setUp", nil)
		if err != nil {
			return
		}
	}

	defer func() {
		if failed := recover(); failed != nil {
			if _, ok := failed.(error); ok {
				err = failed.(error)
			} else {
				err = fmt.Errorf("%v", failed)
			}
		}
	}()

	if runner.tearDown {
		defer func() {
			_, tearDownError := env.VM.Call("tearDown", nil)
			if tearDownError != nil && err == nil {
				err = tearDownError
			}
		}()
	}

	_, err = env.VM.Call(testName, nil)
	if err != nil {
		if ottoError, ok := err.(*otto.Error); ok {
			err = fmt.Errorf(ottoError.String())
		}
	}
	mockError := env.CheckAllMockCallsMade()
	if err == nil {
		err = mockError
	}
	return
}

func (runner *TestRunner) printTestResult(testName string, testErr error) {
	if testErr != nil {
		log.Error(fmt.Sprintf("\t FAIL (%s:%s): %s",
			runner.testFileName, testName, testErr.Error()))
	} else {
		log.Notice("\t PASS (%s:%s)",
			runner.testFileName, testName)
	}
}
