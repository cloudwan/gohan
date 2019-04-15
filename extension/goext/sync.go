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

import (
	"context"
	"time"
)

// Event is an event from Sync
type Event struct {
	Action   string
	Key      string
	ClientId string
	Data     map[string]interface{}
	Revision int64
}

// Node is a node from Sync
type Node struct {
	Key      string
	Value    string
	Revision int64
	Children []*Node
}

// ISync is an interface to sync in Gohan
type ISync interface {
	// Fetch fetches a path from sync
	Fetch(ctx context.Context, path string) (*Node, error)
	// Delete deletes a path from sync
	Delete(ctx context.Context, path string, prefix bool) error
	// Watch watches a single path in sync
	Watch(ctx context.Context, path string, timeout time.Duration, revision int64) ([]*Event, error)
	// Update updates a path with given json
	Update(ctx context.Context, path string, json string) error
}

type ErrCompacted struct {
	error
	// CompactRevision is the minimum revision a watcher may receive
	CompactRevision int64
}

func NewErrCompacted(err error, revision int64) ErrCompacted {
	return ErrCompacted{err, revision}
}

// RevisionCurrent is current sync revision
const RevisionCurrent int64 = -1
