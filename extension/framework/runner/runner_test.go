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

package runner_test

import (
	"github.com/cloudwan/gohan/extension/framework/runner"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Runner", func() {
	const (
		schemaIncludesVar = "SCHEMA_INCLUDES"
		schemasVar        = "SCHEMAS"
		pathVar           = "PATH"
	)

	var (
		testFile   string
		testFilter string
		errors     map[string]error
	)

	JustBeforeEach(func() {
		theRunner := runner.NewTestRunner(testFile, true, testFilter)
		errors = theRunner.Run()
	})

	AfterEach(func() {
		testFilter = ""
	})

	Describe("With incorrect files", func() {
		Context("When the file does not exist", func() {
			BeforeEach(func() {
				testFile = "./test_data/nonexising_file.js"
			})

			It("Should return the proper errors", func() {
				Expect(errors).To(HaveLen(1))
				Expect(errors).To(HaveKeyWithValue(runner.GeneralError, MatchError(ContainSubstring("no such file"))))
			})
		})

		Context("When the file contains invalid javascript", func() {
			BeforeEach(func() {
				testFile = "./test_data/compilation_error.js"
			})

			It("Should return the proper errors", func() {
				Expect(errors).To(HaveLen(1))
				Expect(errors).To(HaveKeyWithValue(runner.GeneralError, MatchError(ContainSubstring("parse file"))))
			})
		})

		Context("When the test schema is not specified", func() {
			BeforeEach(func() {
				testFile = "./test_data/schema_not_specified.js"
			})

			It("Should return the proper errors", func() {
				Expect(errors).To(HaveLen(1))
				Expect(errors).To(HaveKeyWithValue(runner.GeneralError, MatchError(ContainSubstring(schemasVar))))
			})
		})

		Context("When the test schema is not a string", func() {
			BeforeEach(func() {
				testFile = "./test_data/schema_not_a_string.js"
			})

			It("Should return the proper errors", func() {
				Expect(errors).To(HaveLen(1))
				Expect(errors).To(HaveKeyWithValue(runner.GeneralError, MatchError(ContainSubstring(schemasVar))))
			})
		})

		Context("When the test schema does not exist", func() {
			BeforeEach(func() {
				testFile = "./test_data/nonexisting_schema.js"
			})

			It("Should return the proper errors", func() {
				Expect(errors).To(HaveLen(1))
				Expect(errors).To(HaveKeyWithValue(runner.GeneralError, MatchError(ContainSubstring("no such file"))))
			})
		})

		Context("When the test schema include is not specified", func() {
			BeforeEach(func() {
				testFile = "./test_data/schema_include_not_specified.js"
			})

			It("Should return the proper errors", func() {
				Expect(errors).To(HaveLen(1))
				Expect(errors).To(HaveKeyWithValue(runner.GeneralError, MatchError(ContainSubstring(schemaIncludesVar))))
			})
		})

		Context("When the test schema include is not a string", func() {
			BeforeEach(func() {
				testFile = "./test_data/schema_include_not_a_string.js"
			})

			It("Should return the proper errors", func() {
				Expect(errors).To(HaveLen(1))
				Expect(errors).To(HaveKeyWithValue(runner.GeneralError, MatchError(ContainSubstring(schemaIncludesVar))))
			})
		})

		Context("When the test schema include does not exist", func() {
			BeforeEach(func() {
				testFile = "./test_data/nonexisting_schema_include.js"
			})

			It("Should return the proper errors", func() {
				Expect(errors).To(HaveLen(1))
				Expect(errors).To(HaveKeyWithValue(runner.GeneralError, MatchError(ContainSubstring("no such file"))))
			})
		})

		Context("When the test path is not specified", func() {
			BeforeEach(func() {
				testFile = "./test_data/path_not_specified.js"
			})

			It("Should return the proper errors", func() {
				Expect(errors).To(HaveLen(1))
				Expect(errors).To(HaveKeyWithValue(runner.GeneralError, MatchError(ContainSubstring(pathVar))))
			})
		})

		Context("When the test path is not a string", func() {
			BeforeEach(func() {
				testFile = "./test_data/path_not_a_string.js"
			})

			It("Should return the proper errors", func() {
				Expect(errors).To(HaveLen(1))
				Expect(errors).To(HaveKeyWithValue(runner.GeneralError, MatchError(ContainSubstring(pathVar))))
			})
		})
	})

	Describe("With correct files", func() {
		Context("When a runtime occurs", func() {
			BeforeEach(func() {
				testFile = "./test_data/runtime_error.js"
			})

			It("Should return the proper errors", func() {
				Expect(errors).To(HaveLen(1))
				Expect(errors).To(HaveKeyWithValue("testRuntimeError", MatchError("runtime error")))
			})
		})

		Context("When loading multiple schemas", func() {
			BeforeEach(func() {
				testFile = "./test_data/multiple_schemas.js"
			})

			It("Should return no errors", func() {
				Expect(errors).To(HaveLen(1))
				Expect(errors).To(HaveKeyWithValue("testBothSchemasLoaded", BeNil()))
			})
		})

		Context("When filling default values to create db data", func() {
			BeforeEach(func() {
				testFile = "./test_data/default_value_schema.js"
			})

			It("Should return no errors", func() {
				Expect(errors).To(HaveLen(1))
				Expect(errors).To(HaveKeyWithValue("testDBCreatePopulateDefault", BeNil()))
			})
		})

		Context("When loading extensions", func() {
			BeforeEach(func() {
				testFile = "./test_data/extension_loading.js"
			})

			It("Should return no errors", func() {
				Expect(errors).To(HaveLen(2))
				Expect(errors).To(HaveKeyWithValue("testExtension1Loaded", BeNil()))
				Expect(errors).To(HaveKeyWithValue("testExtension2NotLoaded", BeNil()))
			})
		})

		Context("When loading extensions from schema includes", func() {
			BeforeEach(func() {
				testFile = "./test_data/load_schema_includes.js"
			})

			It("Should return no errors", func() {
				Expect(errors).To(HaveLen(2))
				Expect(errors).To(HaveKeyWithValue("testExtension1Loaded", BeNil()))
				Expect(errors).To(HaveKeyWithValue("testExtension2NotLoaded", BeNil()))
			})
		})

		Context("When using filter", func() {
			BeforeEach(func() {
				testFile = "./test_data/extension_loading.js"
				testFilter = ".*1.*"
			})

			It("Should skip non-matching tests", func() {
				Expect(errors).To(HaveLen(1))
				Expect(errors).To(HaveKeyWithValue("testExtension1Loaded", BeNil()))
				Expect(errors).NotTo(HaveKey("testExtension2NotLoaded"))
			})
		})

		Context("When the test fails", func() {
			BeforeEach(func() {
				testFile = "./test_data/fail.js"
			})

			It("Should return the proper errors", func() {
				Expect(errors).To(HaveLen(2))
				Expect(errors).To(HaveKeyWithValue("testFail", MatchError("called testFail")))
				Expect(errors).To(HaveKeyWithValue("testFailNoMessage", HaveOccurred()))
			})
		})

		Context("When fail is invoked incorrectly", func() {
			BeforeEach(func() {
				testFile = "./test_data/fail_error.js"
			})

			It("Should return the proper errors", func() {
				Expect(errors).To(HaveLen(1))
				Expect(errors).To(HaveKeyWithValue("testFailError", MatchError(ContainSubstring("format string expected"))))
			})
		})

		Context("When using GohanTrigger", func() {
			BeforeEach(func() {
				testFile = "./test_data/gohan_trigger.js"
			})

			It("Should return no errors", func() {
				Expect(errors).To(HaveLen(1))
				Expect(errors).To(HaveKeyWithValue("testGohanTrigger", BeNil()))
			})
		})

		Context("When incorrectly using mock", func() {
			BeforeEach(func() {
				testFile = "./test_data/mock_validate.js"
			})

			It("Should return the proper errors", func() {
				Expect(errors).To(HaveLen(3))
				Expect(errors).To(HaveKeyWithValue(
					"testMockExpectNotSpecified", MatchError(ContainSubstring("Expect() should be specified for each call to"))))
				Expect(errors).To(HaveKeyWithValue(
					"testMockReturnNotSpecified", MatchError(ContainSubstring("Return() should be specified for each call to"))))
				Expect(errors).To(HaveKeyWithValue(
					"testMockReturnEmpty", MatchError(ContainSubstring("Return() should be called with exactly one argument"))))
			})
		})

		Context("When mock expected calls do not occur", func() {
			BeforeEach(func() {
				testFile = "./test_data/mock_calls_not_made.js"
			})

			It("Should work", func() {
				Expect(errors).To(HaveLen(2))
				Expect(errors).To(HaveKeyWithValue(
					"testFirstMockCallNotMade", MatchError("Expected call to gohan_http([POST]) with return value OK, but not made")))
				Expect(errors).To(HaveKeyWithValue(
					"testLastMockCallNotMade", MatchError("Expected call to gohan_http([GET]) with return value OK, but not made")))
			})
		})

		Context("When correctly using the Gohan HTTP mock", func() {
			BeforeEach(func() {
				testFile = "./test_data/gohan_http_mock.js"
			})

			It("Should work", func() {
				Expect(errors).To(HaveLen(5))
				Expect(errors).To(HaveKeyWithValue(
					"testUnexpectedCall", MatchError(ContainSubstring("Unexpected call"))))
				Expect(errors).To(HaveKeyWithValue("testSingleReturnSingleCall", BeNil()))
				Expect(errors).To(HaveKeyWithValue("testSingleReturnSingleCallDeepEqual", BeNil()))
				Expect(errors).To(HaveKeyWithValue(
					"testSingleReturnMultipleCalls", MatchError(ContainSubstring("Unexpected call"))))
				Expect(errors).To(HaveKeyWithValue(
					"testWrongArgumentsCall", MatchError(ContainSubstring("Wrong arguments"))))
				Expect(errors).To(HaveKeyWithValue(
					"testWrongArgumentsCall", MatchError(ContainSubstring("expected [POST, http://www.abc.com, map[a:a], map[]]"))))
			})
		})

		Context("When correctly using the Gohan DB Transaction mock", func() {
			BeforeEach(func() {
				testFile = "./test_data/gohan_db_transaction_mock.js"
			})

			It("Should work", func() {
				Expect(errors).To(HaveLen(4))
				Expect(errors).To(HaveKeyWithValue(
					"testUnexpectedCall", MatchError(ContainSubstring("Unexpected call"))))
				Expect(errors).To(HaveKeyWithValue("testSingleReturnSingleCall", BeNil()))
				Expect(errors).To(HaveKeyWithValue(
					"testSingleReturnMultipleCalls", MatchError(ContainSubstring("Unexpected call"))))
				Expect(errors).To(HaveKeyWithValue(
					"testWrongArgumentsCall", MatchError(ContainSubstring("Wrong arguments"))))
				Expect(errors).To(HaveKeyWithValue(
					"testWrongArgumentsCall", MatchError(ContainSubstring("expected []"))))
			})
		})

		Context("When correctly using the Gohan Config mock", func() {
			BeforeEach(func() {
				testFile = "./test_data/gohan_config_mock.js"
			})

			It("Should work", func() {
				Expect(errors).To(HaveLen(4))
				Expect(errors).To(HaveKeyWithValue(
					"testUnexpectedCall", MatchError(ContainSubstring("Unexpected call"))))
				Expect(errors).To(HaveKeyWithValue("testSingleReturnSingleCall", BeNil()))
				Expect(errors).To(HaveKeyWithValue(
					"testSingleReturnMultipleCalls", MatchError(ContainSubstring("Unexpected call"))))
				Expect(errors).To(HaveKeyWithValue(
					"testWrongArgumentsCall", MatchError(ContainSubstring("Wrong arguments"))))
				Expect(errors).To(HaveKeyWithValue(
					"testWrongArgumentsCall", MatchError(ContainSubstring("expected [database/type, sqlite3]"))))
			})
		})

		Context("When correctly using the Gohan Sync Fetch mock", func() {
			BeforeEach(func() {
				testFile = "./test_data/gohan_sync_fetch_mock.js"
			})

			It("Should work", func() {
				Expect(errors).To(HaveLen(3))
				Expect(errors).To(HaveKeyWithValue("testLastThrow", BeNil()))
				Expect(errors).To(HaveKeyWithValue("testWrongThrow", MatchError(ContainSubstring("Fail"))))
				Expect(errors).To(HaveKeyWithValue(
					"testUnexpectedCall", MatchError(ContainSubstring("Unexpected call"))))
			})
		})

		Context("When correctly using the Gohan Sync Fetch mock", func() {
			BeforeEach(func() {
				testFile = "./test_data/gohan_sync_fetch_mock.js"
			})

			It("Should work", func() {
				Expect(errors).To(HaveLen(3))
				Expect(errors).To(HaveKeyWithValue("testLastThrow", BeNil()))
				Expect(errors).To(HaveKeyWithValue("testWrongThrow", MatchError(ContainSubstring("Fail"))))
				Expect(errors).To(HaveKeyWithValue(
					"testUnexpectedCall", MatchError(ContainSubstring("Unexpected call"))))
			})
		})

		Context("When using Gohan builtins", func() {
			BeforeEach(func() {
				testFile = "./test_data/gohan_builtins.js"
			})

			It("Should return no errors", func() {
				Expect(errors).To(HaveLen(1))
				Expect(errors).To(HaveKeyWithValue("testGohanBuiltins", BeNil()))
			})
		})

		Context("When passing extension errors", func() {
			BeforeEach(func() {
				testFile = "./test_data/extension_error.js"
			})

			It("Should return no errors", func() {
				Expect(errors).To(HaveLen(1))
				Expect(errors).To(HaveKeyWithValue("testExtensionError", BeNil()))
			})
		})

		Context("When using mock transactions", func() {
			BeforeEach(func() {
				testFile = "./test_data/mock_transactions.js"
			})

			It("Should return no errors", func() {
				Expect(errors).To(HaveLen(2))
				Expect(errors).To(HaveKeyWithValue("testMockTransactions", BeNil()))
				Expect(errors).To(HaveKeyWithValue("testMockTransactionsInOneTrigger", BeNil()))
			})
		})

		Describe("Using setUp and tearDown", func() {
			Context("When setUp is completed correctly", func() {
				BeforeEach(func() {
					testFile = "./test_data/set_up.js"
				})

				It("Should return no errors", func() {
					Expect(errors).To(HaveLen(1))
					Expect(errors).To(HaveKeyWithValue("testSetUp", BeNil()))
				})
			})

			Context("When an error occurs in setUp", func() {
				BeforeEach(func() {
					testFile = "./test_data/set_up_error.js"
				})

				It("Should return the proper errors", func() {
					Expect(errors).To(HaveLen(1))
					Expect(errors).To(HaveKeyWithValue("testSetUpError", MatchError(ContainSubstring("setUp"))))
				})
			})

			Context("When tearDown is completed correctly", func() {
				BeforeEach(func() {
					testFile = "./test_data/tear_down.js"
				})

				It("Should return no errors", func() {
					Expect(errors).To(HaveLen(1))
					Expect(errors).To(HaveKeyWithValue("testTearDown", MatchError("OK")))
				})
			})

			Context("When an error occurs in tearDown", func() {
				BeforeEach(func() {
					testFile = "./test_data/tear_down_error.js"
				})

				It("Should return the proper errors", func() {
					Expect(errors).To(HaveLen(1))
					Expect(errors).To(HaveKeyWithValue("testTearDownError", MatchError(ContainSubstring("tearDown"))))
				})
			})

			Context("When an error occurs in tearDown and during test execution", func() {
				BeforeEach(func() {
					testFile = "./test_data/tear_down_error_after_error.js"
				})

				It("Should return the original error", func() {
					Expect(errors).To(HaveLen(1))
					Expect(errors).To(HaveKeyWithValue("testTearDownErrorAfterError", MatchError(ContainSubstring("original"))))
				})
			})
		})
	})
})
