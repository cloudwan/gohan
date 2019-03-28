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

import (
	"context"

	"github.com/cloudwan/gohan/sync"
)

//Sync is struct for noop
type Sync struct {
}

//NewSync creates noop sync instance
func NewSync() *Sync {
	return &Sync{}
}

//GetProcessID returns processID
func (sync *Sync) GetProcessID() string {
	return ""
}

//Update sync update sync
func (sync *Sync) Update(ctx context.Context, path, json string) error {
	return nil
}

//Delete sync update sync
func (sync *Sync) Delete(ctx context.Context, path string, prefix bool) error {
	return nil
}

// Fetch sync
func (sync *Sync) Fetch(ctx context.Context, path string) (*sync.Node, error) {
	return nil, nil
}

//HasLock checks is current process has lock
func (sync *Sync) HasLock(path string) bool {
	return false
}

//Lock get lock for path
func (sync *Sync) Lock(ctx context.Context, path string, block bool) (chan struct{}, error) {
	return nil, nil
}

//Unlock unlocks paths
func (sync *Sync) Unlock(ctx context.Context, path string) error {
	return nil
}

//Watch keep watch update under the path until context is canceled
func (sync *Sync) Watch(ctx context.Context, path string, revision int64) <-chan *sync.Event {
	return nil
}

func (s *Sync) Compact(ctx context.Context, revision int64) error {
	return nil
}

func (sync *Sync) CompareAndSwap(ctx context.Context, path, data string, condition ...sync.CASCondition) (bool, error) {
	return true, nil
}

func (sync *Sync) ByValue(value string) sync.CASCondition {
	return nil
}

// Close sync
func (sync *Sync) Close() {

}
