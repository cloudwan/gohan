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
	"path/filepath"
	"regexp"

	"github.com/dop251/otto"
	"github.com/robertkrimen/otto/ast"
	"github.com/robertkrimen/otto/parser"
)

const (
	// GeneralError denotes runner error not related to tests failures
	GeneralError = ""
)

type metaError struct {
	error
}

// TestRunner abstracts running extension tests from a single file
type TestRunner struct {
	testFileName string
	setUp        bool
	tearDown     bool
}

var setUpPattern = regexp.MustCompile("^setUp$")
var tearDownPattern = regexp.MustCompile("^tearDown$")
var testPattern = regexp.MustCompile("^test.*")

// NewTestRunner creates a new test runner for a given test file
func NewTestRunner(testFileName string) *TestRunner {
	return &TestRunner{
		testFileName: testFileName,
	}
}

// Run performs extension tests from the file specified at runner's creation
func (runner *TestRunner) Run() map[string]error {
	src, err := ioutil.ReadFile(runner.testFileName)
	if err != nil {
		return map[string]error{
			GeneralError: fmt.Errorf("Failed to read file '%s': %s", runner.testFileName, err.Error()),
		}
	}

	program, err := parser.ParseFile(nil, runner.testFileName, src, 0)
	if err != nil {
		return map[string]error{
			GeneralError: fmt.Errorf("Failed to parse file '%s': %s", runner.testFileName, err.Error()),
		}
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
			case testPattern.MatchString(name):
				tests = append(tests, name)
			}
		}
	}

	env := NewEnvironment(runner.testFileName, src)

	directory, _ := os.Getwd()
	if err := os.Chdir(filepath.Dir(runner.testFileName)); err != nil {
		return map[string]error{
			GeneralError: fmt.Errorf("Failed to change directory to '%s': %s",
				filepath.Dir(runner.testFileName),
				err.Error()),
		}
	}
	defer os.Chdir(directory)

	errors := map[string]error{}
	for _, test := range tests {
		errors[test] = runner.runTest(test, env)
		if _, ok := errors[test].(metaError); ok {
			return map[string]error{
				GeneralError: errors[test],
			}
		}
	}

	return errors
}

func (runner *TestRunner) runTest(testName string, env *Environment) (err error) {
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
