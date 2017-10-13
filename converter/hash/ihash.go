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

package hash

// IHashable represent tree node can be hashed
type IHashable interface {
	// ToString should return string representation of a value of a node
	// return:
	//   string identifying a value of a node
	ToString() string

	// Compress should compress child of a node
	// args:
	//   1. IHashable - source (node to compress the other node to)
	//   2. IHashable - destination (node to compress)
	Compress(IHashable, IHashable)

	// GetChildren should return children of a node
	// return:
	//   array of children of a node
	GetChildren() []IHashable
}
