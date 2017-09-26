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

package goplugin

import (
	"github.com/cloudwan/gohan/extension/goext"
	gohan_sync "github.com/cloudwan/gohan/sync"
	"time"
)

func convertEvent(event *gohan_sync.Event) *goext.Event {
	if event == nil {
		return nil
	}

	return &goext.Event{
		Action:   event.Action,
		Key:      event.Key,
		Data:     event.Data,
		Revision: event.Revision,
	}
}

func convertNode(node *gohan_sync.Node) *goext.Node {
	if node == nil {
		return nil
	}

	return &goext.Node{
		Key:      node.Key,
		Value:    node.Value,
		Revision: node.Revision,
		Children: convertNodes(node.Children),
	}
}

func convertNodes(nodes []*gohan_sync.Node) []*goext.Node {
	result := []*goext.Node{}

	for _, node := range nodes {
		result = append(result, convertNode(node))
	}

	return result
}

// Sync is an implementation of ISync
type Sync struct {
	raw gohan_sync.Sync
}

// Fetch fetches a path from sync
func (sync *Sync) Fetch(path string) (*goext.Node, error) {
	node, err := sync.raw.Fetch(path)

	if err != nil {
		return nil, err
	}

	return convertNode(node), nil
}

// Delete deletes a path from sync
func (sync *Sync) Delete(path string, prefix bool) error {
	return sync.raw.Delete(path, prefix)
}

// Watch watches a single path in sync
func (sync *Sync) Watch(path string, timeout time.Duration, revision int64) ([]*goext.Event, error) {
	eventChan := make(chan *gohan_sync.Event, 32)
	stopChan := make(chan bool, 1)
	defer close(stopChan)
	errorChan := make(chan error, 1)

	go func() {
		if err := sync.raw.Watch(path, eventChan, stopChan, revision); err != nil {
			errorChan <- err
		}
	}()

	// todo(przemyslaw-dobrowolski-cl): add support for timeouts
	select {
	case event := <-eventChan:
		return []*goext.Event{convertEvent(event)}, nil
	case <-time.After(timeout):
		return nil, nil
	case err := <-errorChan:
		return nil, err
	}
}

// NewSync allocates Sync
func NewSync(sync gohan_sync.Sync) *Sync {
	return &Sync{raw: sync}
}

// Clone allocates a clone of Sync; object may be nil
func (sync *Sync) Clone() *Sync {
	if sync == nil {
		return nil
	}
	return &Sync{
		raw: sync.raw,
	}
}
