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

package sync

//Sync is a interface for sync servers
type Sync interface {
	HasLock(path string) bool
	Lock(path string, block bool) error
	Unlock(path string) error
	Fetch(path string) (*Node, error)
	Update(path, json string) error
	UpdateTTL(path, json string, ttlSecs uint64) error
	Delete(path string) error
	Watch(path string, responseChan chan *Event, stopChan chan bool) error
}

//Event is a struct for Watch response
type Event struct {
	Action string
	Data   map[string]interface{}
	Key    string
}

// Node is a struct for Fetch response
type Node struct {
	Value    string
	Key      string
	Children []*Node
}
