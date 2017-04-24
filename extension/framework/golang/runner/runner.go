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
	"github.com/cloudwan/gohan/extension/goext"
	"github.com/cloudwan/gohan/extension/golang"
	l "github.com/cloudwan/gohan/log"
	"github.com/cloudwan/gohan/schema"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"path/filepath"
	"plugin"
	"testing"
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

func (self *GoTestRunner) Run() (err error) {
	log.Notice("Running Go extensions tests")

	t := &testing.T{}

	for _, testPath := range self.tests {
		log.Notice("Loading test: %s", testPath)

		test, err := plugin.Open(testPath)

		if err != nil {
			return err
		}

		environment := golang.NewEnvironment("test", nil, nil, nil).ExtEnvironment()
		directory := filepath.Dir(testPath)

		//Load required schemas
		Schemas, err := test.Lookup("Schemas")
		if err == nil {
			f := Schemas.(func() []string)

			mgr := schema.GetManager()

			for _, schemaPath := range f() {
				mgr.LoadSchemaFromFile(directory + "/" + schemaPath)
			}
		}

		//Load plugin
		Binary, err := test.Lookup("Binary")
		if err == nil {
			f := Binary.(func() string)

			plug, err := plugin.Open(directory + "/" + f())

			if err != nil {
				return err
			}

			Init, err := plug.Lookup("Init")

			if err != nil {
				return err
			}

			err = Init.(func(*goext.Environment) error)(&environment)

			if err != nil {
				return err
			}
		} else {
			return err
		}

		//Setup test suite
		Test, err := test.Lookup("Test")
		if err != nil {
			return err
		}

		f := Test.(func(*goext.Environment))
		f(&environment)
	}

	RegisterFailHandler(Fail)
	RunSpecs(t, "Go Extensions Test Suite")

	return nil
}
