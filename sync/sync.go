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

import (
	"context"

	l "github.com/cloudwan/gohan/log"
)

// RevisionCurrent is current sync revision
const RevisionCurrent = -1

var log = l.NewLogger()

//Sync is a interface for sync servers
type Sync interface {
	HasLock(path string) bool
	Lock(path string, block bool) (notifyLost chan struct{}, err error)
	Unlock(path string) error
	Fetch(path string) (*Node, error)
	Update(path, json string) error
	Delete(path string, prefix bool) error
	// Watch monitors changes on path and emits Events to responseChan.
	// Close stopChan to cancel.
	// You can specify the revision to start watching,
	// give RevisionCurrent when you want to start from the current revision.
	// Returns an error when gets any error including connection failures.
	Watch(path string, responseChan chan *Event, stopChan chan bool, revision int64) error
	//WatchContext keep watch update under the path until context is canceled.
	WatchContext(ctx context.Context, path string, revision int64) <-chan *Event
	GetProcessID() string
	Close()
}

//Event is a struct for Watch response
type Event struct {
	Action   string
	Key      string
	Data     map[string]interface{}
	Revision int64
	// Err is used only by Sync.WatchContext()
	Err error
}

//Node is a struct for Fetch response
type Node struct {
	Key      string
	Value    string
	Revision int64
	Children []*Node
}
