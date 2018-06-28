// Copyright (C) 2018 NTT Innovation Institute, Inc.
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

package etcd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/cloudwan/gohan/sync"
)

func TestFetch(t *testing.T) {
	sync := newSync()
	sync.etcdClient.Delete("/", true)
	path := "/path/to/somewhere"

	err := sync.Update(path, "test1")
	if err != nil {
		t.Fatalf("unexpected error")
	}

	nodes, err := sync.Fetch(path)
	if err != nil {
		t.Fatalf("unexpected error")
	}
	if nodes.Key != path {
		t.Fatalf("expected node to has key %s has %s instead", path, nodes.Key)
	}
	if len(nodes.Children) != 0 {
		t.Fatalf("expected node to not has children has %d", len(nodes.Children))
	}
	if nodes.Value != "test1" {
		t.Fatalf("expected node to has value test1 has %s instead", nodes.Value)
	}

	err = sync.Delete(path, false)
	if err != nil {
		t.Fatalf("unexpected error")
	}

	_, err = sync.Fetch(path)
	if err == nil || !strings.Contains(err.Error(), fmt.Sprintf("Key not found (%s)", path)) {
		t.Fatalf("unexpected non error")
	}
}

func TestFetchMultipleNodes(t *testing.T) {
	sync := newSync()
	sync.etcdClient.Delete("/", true)

	err := sync.Update("/path/to/somewhere", "test1")
	if err != nil {
		t.Fatalf("unexpected error")
	}
	err = sync.Update("/path/to/elsewhere", "test2")
	if err != nil {
		t.Fatalf("unexpected error")
	}
	err = sync.Update("/path/notto/elsewhere", "test3")
	if err != nil {
		t.Fatalf("unexpected error")
	}

	nodes, err := sync.Fetch("/path")
	if err != nil {
		t.Fatalf("unexpected error")
	}
	if nodes.Key != "/path" {
		t.Fatalf("expected node to has key /path has %s instead", nodes.Key)
	}
	if len(nodes.Children) != 2 {
		t.Fatalf("expected 2 node has %d", len(nodes.Children))
	}
	if nodes.Children[0].Key != "/path/notto" {
		t.Fatalf("expected node to has key /path/notto has %s instead", nodes.Children[0].Key)
	}
	if len(nodes.Children[0].Children) != 1 {
		t.Fatalf("expected 1 node has %d", len(nodes.Children[0].Children))
	}
	node0 := nodes.Children[0].Children[0]
	if node0.Value != "test3" || node0.Key != "/path/notto/elsewhere" || len(node0.Children) != 0 {
		t.Fatalf("incorrect value for node %+v", node0)
	}
	if nodes.Children[1].Key != "/path/to" {
		t.Fatalf("expected node to has key /path/to has %s instead", nodes.Children[1].Key)
	}
	if len(nodes.Children[1].Children) != 2 {
		t.Fatalf("expected 2 nodes has %d", len(nodes.Children[1].Children))
	}
	node0 = nodes.Children[1].Children[0]
	if node0.Value != "test2" || node0.Key != "/path/to/elsewhere" || len(node0.Children) != 0 {
		t.Fatalf("incorrect value for node %+v", node0)
	}
	node1 := nodes.Children[1].Children[1]
	if node1.Value != "test1" || node1.Key != "/path/to/somewhere" || len(node1.Children) != 0 {
		t.Fatalf("incorrect value for node %+v", node1)
	}
}

func TestRecursiveUpdate(t *testing.T) {
	syn := newSync()
	syn.etcdClient.Delete("/", true)
	mustUpdate := func(key, value string) {
		if err := syn.Update(key, value); err != nil {
			t.Fatalf("unexpected error %s", err.Error())
		}
	}
	checkNode := func(node *sync.Node, key, value string, children int) {
		if node.Key != key {
			t.Fatalf("expected key %s has %s", key, node.Key)
		}
		if node.Value != value {
			t.Fatalf("expected value %s has %s", value, node.Value)
		}
		if len(node.Children) != children {
			t.Fatalf("expected to has %d children has %s", children, len(node.Children))
		}
	}

	mustUpdate("/a/b/c/d/1", "test1")
	mustUpdate("/a/b/c/d/2", "test2")
	mustUpdate("/a/b/c/d/3", "test3")
	mustUpdate("/a/b/e/d/1", "test4")
	mustUpdate("/a/b/e/d/2", "test5")
	mustUpdate("/a/b/e/d/3", "test6")

	node, err := syn.Fetch("/a")
	if err != nil {
		t.Fatalf("unexpected error %s", err.Error())
	}

	checkNode(node, "/a", "", 1)
	ab := node.Children[0]
	checkNode(ab, "/a/b", "", 2)
	abc := ab.Children[0]
	checkNode(abc, "/a/b/c", "", 1)
	abcd := abc.Children[0]
	checkNode(abcd, "/a/b/c/d", "", 3)
	checkNode(abcd.Children[0], "/a/b/c/d/1", "test1", 0)
	checkNode(abcd.Children[1], "/a/b/c/d/2", "test2", 0)
	checkNode(abcd.Children[2], "/a/b/c/d/3", "test3", 0)

	abe := ab.Children[1]
	checkNode(abe, "/a/b/e", "", 1)
	abed := abe.Children[0]
	checkNode(abed, "/a/b/e/d", "", 3)
	checkNode(abed.Children[0], "/a/b/e/d/1", "test4", 0)
	checkNode(abed.Children[1], "/a/b/e/d/2", "test5", 0)
	checkNode(abed.Children[2], "/a/b/e/d/3", "test6", 0)

	node, err = syn.Fetch("/a/b/c")
	if err != nil {
		t.Fatalf("unexpected error %s", err.Error())
	}
	checkNode(abc, "/a/b/c", "", 1)
}

func newSync() *Sync {
	sync := NewSync([]string{"http://127.0.0.1:2379"})
	return sync
}
