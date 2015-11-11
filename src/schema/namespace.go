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

package schema

// Namespace describes a group of schemas that form a common endpoint
type Namespace struct {
	ID              string
	Parent          string
	ParentNamespace *Namespace
	Prefix          string
}

// Version ...
type Version struct {
	Status string `json:"status"`
	ID     string `json:"id"`
	Links  []Link `json:"links"`
}

// NamespaceResource ...
type NamespaceResource struct {
	Links      []Link `json:"links"`
	Name       string `json:"name"`
	Collection string `json:"collection"`
}

// Link ...
type Link struct {
	Href string `json:"href"`
	Rel  string `json:"rel"`
}

// NewNamespace is a constructor for a namespace
func NewNamespace(raw interface{}) (*Namespace, error) {
	typeData := raw.(map[string](interface{}))
	namespace := &Namespace{}
	namespace.ID, _ = typeData["id"].(string)
	namespace.Prefix, _ = typeData["prefix"].(string)
	namespace.Parent, _ = typeData["parent"].(string)
	return namespace, nil
}

// SetParentNamespace sets a parent of a namespace to the provided one
func (namespace *Namespace) SetParentNamespace(parent *Namespace) {
	namespace.ParentNamespace = parent
}

// GetFullPrefix returns a full prefix of a namespace
func (namespace *Namespace) GetFullPrefix() string {
	if namespace.Parent == "" {
		return "/" + namespace.Prefix
	}

	return namespace.ParentNamespace.GetFullPrefix() + "/" + namespace.Prefix
}

// IsTopLevel checks whether namespace is a top-level namespace
func (namespace *Namespace) IsTopLevel() bool {
	return namespace.Parent == ""
}
