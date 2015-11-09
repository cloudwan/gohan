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

package noop

import "github.com/cloudwan/gohan/sync"

//Sync is struct for noop
type Sync struct {
}

//NewSync creates noop sync instance
func NewSync() *Sync {
	return &Sync{}
}

//Update sync update sync
func (sync *Sync) Update(path, json string) error {
	return nil
}

//Delete sync update sync
func (sync *Sync) Delete(path, json string) error {
	return nil
}

//HasLock checks is current process has lock
func (sync *Sync) HasLock(path string) bool {
	return false
}

//Lock get lock for path
func (sync *Sync) Lock(path string, block bool) error {
	return nil
}

//Unlock unlocks paths
func (sync *Sync) Unlock(path string) error {
	return nil
}

//Watch keep watch update under the path
func (sync *Sync) Watch(path string, responseChan chan *sync.Event, stopChan chan bool) error {
	return nil
}
