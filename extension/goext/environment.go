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

// IEnvironment is the only scope of Gohan available for a go extensions;
// other packages must not be imported nor used
type IEnvironment interface {
	// modules
	Core() ICore
	Logger() ILogger
	Schemas() ISchemas
	Sync() ISync
	Database() IDatabase
	HTTP() IHTTP
	Auth() IAuth

	// state
	Reset()
}

// ResourceBase is the base class for all resources
type ResourceBase struct {
	Environment IEnvironment
	Logger      ILogger
	Schema      ISchema
}

// NullString represents a nullable string
type NullString struct {
	Value string
	Valid bool
}

// NullBool represents a nullable bool
type NullBool struct {
	Value bool
	Valid bool
}

// NullInt represents a nullable int
type NullInt struct {
	Value int
	Valid bool
}

// NullFloat represents a nullable float
type NullFloat struct {
	Value float64
	Valid bool
}

// MarshalJSON marshals a nullable string to byte buffer
func (ns NullString) MarshalJSON() ([]byte, error) {
	if ns.Valid {
		return json.Marshal(ns.Value)
	}
	return json.Marshal(false)
}

// UnmarshalJSON unmarshals a byte buffer to a nullable string
func (ns *NullString) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		var valid bool
		if err := json.Unmarshal(b, &valid); err != nil {
			return err
		}
		ns.Valid = valid
		return nil
	}
	ns.Value = s
	ns.Valid = true
	return nil
}

// MarshalJSON marshals a nullable bool to byte buffer
func (nb NullBool) MarshalJSON() ([]byte, error) {
	if nb.Valid {
		return json.Marshal(nb.Value)
	}
	return json.Marshal(false)
}

// UnmarshalJSON unmarshals a byte buffer to a nullable bool
func (nb *NullBool) UnmarshalJSON(b []byte) error {
	var val bool
	if err := json.Unmarshal(b, &val); err != nil {
		var valid bool
		if err := json.Unmarshal(b, &valid); err != nil {
			return err
		}
		nb.Valid = valid
		return nil
	}
	nb.Value = val
	nb.Valid = true
	return nil
}

// MarshalJSON marshals a nullable int to byte buffer
func (ni NullInt) MarshalJSON() ([]byte, error) {
	if ni.Valid {
		return json.Marshal(ni.Value)
	}
	return json.Marshal(false)
}

// UnmarshalJSON unmarshals a byte buffer to a nullable int
func (ni *NullInt) UnmarshalJSON(b []byte) error {
	var i int
	if err := json.Unmarshal(b, &i); err != nil {
		var valid bool
		if err := json.Unmarshal(b, &valid); err != nil {
			return err
		}
		ni.Valid = valid
		return nil
	}
	ni.Value = i
	ni.Valid = true
	return nil
}

// MarshalJSON marshals a nullable float to byte buffer
func (nf NullFloat) MarshalJSON() ([]byte, error) {
	if nf.Valid {
		return json.Marshal(nf.Value)
	}
	return json.Marshal(false)
}

// UnmarshalJSON unmarshals a byte buffer to a nullable float
func (nf *NullFloat) UnmarshalJSON(b []byte) error {
	var f float64
	if err := json.Unmarshal(b, &f); err != nil {
		var valid bool
		if err := json.Unmarshal(b, &valid); err != nil {
			return err
		}
		nf.Valid = valid
		return nil
	}
	nf.Value = f
	nf.Valid = true
	return nil
}

// MakeNullString allocates a new nullable string and sets its value
func MakeNullString(value string) NullString {
	return NullString{
		Value: value,
		Valid: true,
	}
}

// MakeNullBool allocates a new nullable bool and sets its value
func MakeNullBool(value bool) NullBool {
	return NullBool{
		Value: value,
		Valid: true,
	}
}

// MakeNullInt allocates a new nullable int and sets its value
func MakeNullInt(value int) NullInt {
	return NullInt{
		Value: value,
		Valid: true,
	}
}

// MakeNullFloat allocates a new nullable float and sets its value
func MakeNullFloat(value float64) NullFloat {
	return NullFloat{
		Value: value,
		Valid: true,
	}
}
