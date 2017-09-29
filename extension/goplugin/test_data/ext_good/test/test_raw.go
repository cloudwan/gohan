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

package test

import "github.com/cloudwan/gohan/extension/goext"

// Subobject is a test resource
type Subobject struct {
	Subproperty string `json:"subproperty,omitempty"`
}

// Test is a test resource
type Test struct {
	ID          string            `db:"id" json:"id"`
	Description string            `db:"description" json:"description"`
	Name        goext.MaybeString `db:"name" json:"name,omitempty"`
	Subobject   *Subobject        `db:"subobject" json:"subobject,omitempty"`
	TestSuiteID goext.MaybeString `db:"test_suite_id" json:"test_suite_id"`
}

// TestSuite is a test suite resource
type TestSuite struct {
	ID string `db:"id" json:"id"`
}
