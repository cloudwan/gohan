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

package goext

import "encoding/json"

type MaybeState int

const (
	MaybeUndefined MaybeState = iota
	MaybeNull
	MaybeValue
)

// Maybe is a 3-value state keeper
//
// Add it to your custom structure as a anonymous field to make structure 3-value
// 3-value structures are marshalled into JSON value based on their state
// Structures in:
// - Undefined state won't appear in JSON at all
// - Null state will appear in JSON as null
// - NotNull state will appear in JSON as they would normally be marshalled
//
// Example:
// Given structures:
// type TestSuite struct {
// 	 goext.Maybe
//	 ID string `json:"id"`
// }
//
// type Test struct {
// 	 ID string `json:"id"`
//	 Name string  `json:"name"`
//	 TestSuite TestSuite `json:"test_suite"`
// }
// func MakeNullTestSuite() Test {
// 	 return TestSuite{
//   Maybe: goext.Maybe{MaybeState: goext.MaybeNull}
//	 }
// }
//
// json.Marshal(env.Util().ResourceToMap(&Test{ID: "some-id", Name:"My resource"}))
// Will produce {"id": "some-id", "name": "My resource"}
//
// json.Marshal(env.Util().ResourceToMap(&Test{ID: "some-id", Name:"My resource"}, TestSuite: MakeNullTestSuite()))
// Will produce {"id": "some-id", "name": "My resource", "test_suite": null}
//
// json.Marshal(env.Util().ResourceToMap(&Test{ID: "some-id", Name:"My resource"}, TestSuite: MakeTestSuite("test-suite-id")))
// Will produce {"id": "some-id", "name": "My resource", "test_suite": {"id": "test-suite-id"}}
//
//
// Structures embedding Maybe are sensitive to internal state changes.
// User must take extra care to change MaybeState field during every operation on any field.
// Example:
// nullTestSuite := MakeNullTestSuite()
// nullTestSuite.ID = "some-test-id"
//
// nullTestSuite.MaybeState is still MaybeNull, it will be marshalled into null even that it contain proper value
//

type Maybe struct {
	MaybeState MaybeState
}

// IsUndefined returns whether value is undefined
func (m Maybe) IsUndefined() bool {
	return m.MaybeState == MaybeUndefined
}

// IsNull returns whether value is null
func (m Maybe) IsNull() bool {
	return m.MaybeState == MaybeNull
}

// HasValue returns whether value is defined and not null
func (m Maybe) HasValue() bool {
	return m.MaybeState == MaybeValue
}

// MaybeString represents 3-valued string
type MaybeString struct {
	Maybe
	value string
}

// MaybeFloat represents 3-valued float
type MaybeFloat struct {
	Maybe
	value float64
}

// MaybeInt represents 3-valued int
type MaybeInt struct {
	Maybe
	value int
}

// MaybeBool represents 3-valued bool
type MaybeBool struct {
	Maybe
	value bool
}

func (ms MaybeString) Value() string {
	return ms.value
}

func (mf MaybeFloat) Value() float64 {
	return mf.value
}

func (mi MaybeInt) Value() int {
	return mi.value
}

func (mb MaybeBool) Value() bool {
	return mb.value
}

func (ms *MaybeString) UnmarshalJSON(b []byte) error {
	if b == nil {
		ms.MaybeState = MaybeNull
	} else if err := json.Unmarshal(b, &ms.value); err != nil {
		return err
	}
	return nil
}

func (ms MaybeString) MarshalJSON() ([]byte, error) {
	if ms.IsNull() || ms.IsUndefined() {
		return []byte("null"), nil
	}
	return json.Marshal(ms.value)
}

func (mi *MaybeInt) UnmarshalJSON(b []byte) error {
	if b == nil {
		mi.MaybeState = MaybeNull
	} else if err := json.Unmarshal(b, &mi.value); err != nil {
		return err
	}
	return nil
}

func (mi MaybeInt) MarshalJSON() ([]byte, error) {
	if mi.IsNull() || mi.IsUndefined() {
		return []byte("null"), nil
	}
	return json.Marshal(mi.value)
}

func (mb *MaybeBool) UnmarshalJSON(b []byte) error {
	if b == nil {
		mb.MaybeState = MaybeNull
	} else if err := json.Unmarshal(b, &mb.value); err != nil {
		return err
	}
	return nil
}

func (mb MaybeBool) MarshalJSON() ([]byte, error) {
	if mb.IsNull() || mb.IsUndefined() {
		return []byte("null"), nil
	}
	return json.Marshal(mb.value)
}

func (mf *MaybeFloat) UnmarshalJSON(b []byte) error {
	if b == nil {
		mf.MaybeState = MaybeNull
	} else if err := json.Unmarshal(b, &mf.value); err != nil {
		return err
	}
	return nil
}

func (mf MaybeFloat) MarshalJSON() ([]byte, error) {
	if mf.IsNull() || mf.IsUndefined() {
		return []byte("null"), nil
	}
	return json.Marshal(mf.value)
}

/*
   Equality rules:
   https://developer.mozilla.org/en-US/docs/Web/JavaScript/Equality_comparisons_and_sameness

   |-----------|-----------|-------|-------|
   | Operands  | Undefined | Null  | Value |
   |-----------|-----------|-------|-------|
   | Undefined | true      | true  | false |
   | Null      | true      | true  | false |
   | Value     | false     | false | A==B  |
   |-----------|-----------|-------|-------|

*/

// Equals returns whether two maybe values are equal
func (mf MaybeFloat) Equals(other MaybeFloat) bool {
	if mf.HasValue() && other.HasValue() {
		return mf.value == other.value
	}
	return !mf.HasValue() && !other.HasValue()
}

// Equals returns whether two maybe values are equal
func (this MaybeString) Equals(other MaybeString) bool {
	if this.HasValue() && other.HasValue() {
		return this.value == other.value
	}
	return !this.HasValue() && !other.HasValue()
}

// Equals returns whether two maybe values are equal
func (mb MaybeBool) Equals(other MaybeBool) bool {
	if mb.HasValue() && other.HasValue() {
		return mb.value == other.value
	}
	return !mb.HasValue() && !other.HasValue()
}

// Equals returns whether two maybe values are equal
func (this MaybeInt) Equals(other MaybeInt) bool {
	if this.HasValue() && other.HasValue() {
		return this.value == other.value
	}
	return !this.HasValue() && !other.HasValue()
}

// MakeNullString allocates a new null string
func MakeNullString() MaybeString {
	return MaybeString{
		Maybe: Maybe{MaybeState: MaybeNull},
	}
}

// MakeNullInt allocates a new null integer
func MakeNullInt() MaybeInt {
	return MaybeInt{
		Maybe: Maybe{MaybeState: MaybeNull},
	}
}

// MakeNullBool allocates a new null bool
func MakeNullBool() MaybeBool {
	return MaybeBool{
		Maybe: Maybe{MaybeState: MaybeNull},
	}
}

// MakeNullFloat allocates a new null float
func MakeNullFloat() MaybeFloat {
	return MaybeFloat{
		Maybe: Maybe{MaybeState: MaybeNull},
	}
}

// MakeString allocates a new MaybeString and sets its value
func MakeString(value string) MaybeString {
	return MaybeString{
		value: value,
		Maybe: Maybe{MaybeState: MaybeValue},
	}
}

// MakeInt allocates a new MaybeInt and sets its value
func MakeInt(value int) MaybeInt {
	return MaybeInt{
		value: value,
		Maybe: Maybe{MaybeState: MaybeValue},
	}
}

// MakeFloat allocates a new MaybeFloat and sets its value
func MakeFloat(value float64) MaybeFloat {
	return MaybeFloat{
		value: value,
		Maybe: Maybe{MaybeState: MaybeValue},
	}
}

// MakeBool allocates a new MaybeBool and sets its value
func MakeBool(value bool) MaybeBool {
	return MaybeBool{
		value: value,
		Maybe: Maybe{MaybeState: MaybeValue},
	}
}

// MakeUndefinedString allocates a new MaybeString with undefined value
func MakeUndefinedString() MaybeString {
	return MaybeString{
		Maybe: Maybe{MaybeState: MaybeUndefined},
	}
}

// MakeUndefinedInt allocates a new MaybeInt with undefined value
func MakeUndefinedInt() MaybeInt {
	return MaybeInt{
		Maybe: Maybe{MaybeState: MaybeUndefined},
	}
}

// MakeUndefinedFloat allocates a new MaybeFloat with undefined value
func MakeUndefinedFloat() MaybeFloat {
	return MaybeFloat{
		Maybe: Maybe{MaybeState: MaybeUndefined},
	}
}

// MakeUndefinedBool allocates a new MaybeBool with undefined value
func MakeUndefinedBool() MaybeBool {
	return MaybeBool{
		Maybe: Maybe{MaybeState: MaybeUndefined},
	}
}
