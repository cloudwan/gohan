package etcdv3

import (
	"fmt"
	"strings"
	"testing"
	"time"

	gohan_sync "github.com/cloudwan/gohan/sync"
	etcd "github.com/coreos/etcd/clientv3"
	"golang.org/x/net/context"
)

var endpoints = []string{"localhost:2379"}

func TestNewSyncTimeout(t *testing.T) {
	done := make(chan struct{})
	go func() {
		_, err := NewSync([]string{"invalid:1000"}, time.Millisecond*100)
		if err == nil {
			t.Fatalf("nil returned for error")
		}
		close(done)
	}()
	select {
	case <-time.NewTimer(time.Millisecond * 200).C:
		t.Fatalf("timeout didn't work")
	case <-done:
	}
}

func TestNonEmptyUpdate(t *testing.T) {
	sync := newSync(t)
	sync.etcdClient.Delete(context.Background(), "/", etcd.WithPrefix())

	path := "/path/to/somewhere"
	data := "blabla"
	err := sync.Update(path, data)
	if err != nil {
		t.Fatalf("unexpected error")
	}

	node, err := sync.Fetch(path)
	if err != nil {
		t.Fatalf("unexpected error")
	}
	if node.Key != path || node.Value != data || len(node.Children) != 0 {
		t.Fatalf("unexpected node: %+v", node)
	}

	err = sync.Delete(path, false)
	if err != nil {
		t.Fatalf("unexpected error")
	}

	node, err = sync.Fetch(path)
	if err == nil || !strings.Contains(err.Error(), fmt.Sprintf("Key not found (%s)", path)) {
		t.Fatalf("unexpected non error")
	}
}

func TestEmptyUpdate(t *testing.T) {
	sync := newSync(t)
	sync.etcdClient.Delete(context.Background(), "/", etcd.WithPrefix())

	path := "/path/to/somewhere"
	data := ""
	err := sync.Update(path, data)
	if err != nil {
		t.Fatalf("unexpected error")
	}

	// not found because v3 doesn't support directories
	_, err = sync.Fetch(path)
	if err == nil {
		t.Fatalf("unexpected error")
	}
}

func TestRecursiveUpdate(t *testing.T) {
	syn := newSync(t)
	syn.etcdClient.Delete(context.Background(), "/", etcd.WithPrefix())
	mustUpdate := func(key, value string) {
		if err := syn.Update(key, value); err != nil {
			t.Fatalf("unexpected error %s", err.Error())
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

	checkNode(node, "/a", "", 1, t)
	ab := node.Children[0]
	checkNode(ab, "/a/b", "", 2, t)
	abc := ab.Children[0]
	checkNode(abc, "/a/b/c", "", 1, t)
	abcd := abc.Children[0]
	checkNode(abcd, "/a/b/c/d", "", 3, t)
	checkNode(abcd.Children[0], "/a/b/c/d/1", "test1", 0, t)
	checkNode(abcd.Children[1], "/a/b/c/d/2", "test2", 0, t)
	checkNode(abcd.Children[2], "/a/b/c/d/3", "test3", 0, t)

	abe := ab.Children[1]
	checkNode(abe, "/a/b/e", "", 1, t)
	abed := abe.Children[0]
	checkNode(abed, "/a/b/e/d", "", 3, t)
	checkNode(abed.Children[0], "/a/b/e/d/1", "test4", 0, t)
	checkNode(abed.Children[1], "/a/b/e/d/2", "test5", 0, t)
	checkNode(abed.Children[2], "/a/b/e/d/3", "test6", 0, t)

	node, err = syn.Fetch("/a/b/c")
	if err != nil {
		t.Fatalf("unexpected error %s", err.Error())
	}
	checkNode(abc, "/a/b/c", "", 1, t)
}

func TestLockUnblocking(t *testing.T) {
	sync0 := newSync(t)
	sync1 := newSync(t)
	sync0.etcdClient.Delete(context.Background(), "/", etcd.WithPrefix())

	ctx := context.Background()

	path := "/path/lock"
	_, err := sync0.Lock(ctx, path, false)
	if err != nil {
		t.Fatalf("unexpected error")
	}
	_, err = sync1.Lock(ctx, path, false)
	if err == nil {
		t.Fatalf("unexpected non error")
	}

	if sync0.HasLock(path) != true {
		t.Fatalf("unexpected false")
	}
	if sync1.HasLock(path) != false {
		t.Fatalf("unexpected true")
	}

	err = sync0.Unlock(path)
	if err != nil {
		t.Fatalf("unexpected error")
	}
	_, err = sync1.Lock(ctx, path, false)
	if err != nil {
		t.Fatalf("unexpected  error")
	}

	if sync0.HasLock(path) != false {
		t.Fatalf("unexpected true")
	}
	if sync1.HasLock(path) != true {
		t.Fatalf("unexpected false")
	}
}

func TestLockBlocking(t *testing.T) {
	ctx := context.Background()

	sync0 := newSync(t)
	sync1 := newSync(t)
	sync0.etcdClient.Delete(ctx, "/", etcd.WithPrefix())

	path := "/path/lock"
	_, err := sync0.Lock(ctx, path, true)
	if err != nil {
		t.Fatalf("unexpected error")
	}
	locked1 := make(chan struct{})
	go func() {
		_, err := sync1.Lock(ctx, path, true)
		if err != nil {
			t.Fatalf("unexpected error")
		}
		close(locked1)
	}()

	time.Sleep(time.Millisecond * 100)
	select {
	case <-locked1:
		t.Fatalf("blocking failed")
	default:
	}

	if sync0.HasLock(path) != true {
		t.Fatalf("unexpected false")
	}
	if sync1.HasLock(path) != false {
		t.Fatalf("unexpected true")
	}

	err = sync0.Unlock(path)
	if err != nil {
		t.Fatalf("unexpected error")
	}
	time.Sleep(time.Millisecond * 200)
	<-locked1

	if sync0.HasLock(path) != false {
		t.Fatalf("unexpected true")
	}
	if sync1.HasLock(path) != true {
		t.Fatalf("unexpected false")
	}
}

func TestWatch(t *testing.T) {
	sync := newSync(t)
	sync.etcdClient.Delete(context.Background(), "/", etcd.WithPrefix())

	path := "/path/to/watch/without/revision"
	responseChan := make(chan *gohan_sync.Event)
	stopChan := make(chan bool)

	sync.etcdClient.Put(context.Background(), path+"/existing", `{"existing": true}`)

	go func() {
		err := sync.Watch(path, responseChan, stopChan, gohan_sync.RevisionCurrent)
		if err != nil {
			t.Fatalf("failed to watch")
		}
	}()

	resp := <-responseChan
	if resp.Action != "get" || resp.Key != path+"/existing" || resp.Data["existing"].(bool) != true {
		t.Fatalf("mismatch response: %+v", resp)
	}

	sync.etcdClient.Put(context.Background(), path+"/new", `{"existing": false}`)
	resp = <-responseChan
	if resp.Action != "set" || resp.Key != path+"/new" || resp.Data["existing"].(bool) != false {
		t.Fatalf("mismatch response: %+v", resp)
	}

	sync.etcdClient.Delete(context.Background(), path+"/existing")
	resp = <-responseChan
	if resp.Action != "delete" || resp.Key != path+"/existing" || len(resp.Data) != 0 {
		t.Fatalf("mismatch response: %+v", resp)
	}
}

func TestWatchWithRevision(t *testing.T) {
	sync := newSync(t)
	sync.etcdClient.Delete(context.Background(), "/", etcd.WithPrefix())

	path := "/path/to/watch/with/revision"
	responseChan := make(chan *gohan_sync.Event)
	stopChan := make(chan bool)

	putResponse, err := sync.etcdClient.Put(context.Background(), path+"/existing", `{"existing": true}`)
	if err != nil {
		t.Fatalf("failed to put key: %s", err)
	}
	startRev := putResponse.Header.Revision

	putResponse, err = sync.etcdClient.Put(context.Background(), path+"/new", `{"existing": false}`)
	if err != nil {
		t.Fatalf("failed to update key: %s", err)
	}
	secondRevision := putResponse.Header.Revision

	go func() {
		err := sync.Watch(path, responseChan, stopChan, startRev+1)
		if err != nil {
			t.Fatalf("failed to watch")
		}
	}()

	resp := <-responseChan
	if resp.Key != path+"/new" || resp.Data["existing"].(bool) != false || resp.Revision != secondRevision {
		t.Fatalf("mismatch response: %+v, expecting /new, existing==false, revision==%d", resp, secondRevision)
	}

	putResponse, err = sync.etcdClient.Put(context.Background(), path+"/third", `{"existing": false}`)
	if err != nil {
		t.Fatalf("failed to update key: %s", err)
	}
	thirdRevision := putResponse.Header.Revision

	resp = <-responseChan
	if resp.Key != path+"/third" || resp.Data["existing"].(bool) != false || resp.Revision != thirdRevision {
		t.Fatalf("mismatch response: %+v, expecting /third, existing==false, revision==%d", resp, thirdRevision)
	}

}

func TestFetchMultipleNodes(t *testing.T) {
	sync := newSync(t)
	sync.etcdClient.Delete(context.Background(), "/", etcd.WithPrefix())

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
		t.Fatalf("unexpected error %s", err.Error())
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

func TestUpdateWithoutPath(t *testing.T) {
	sync := newSync(t)
	err := sync.Update("post-migration", "test")
	if err != nil {
		t.Fatalf("unexpected error")
	}

	nodes, err := sync.Fetch("post-migration")
	if err != nil {
		t.Fatalf("unexpected error %s", err.Error())
	}
	checkNode(nodes, "post-migration", "test", 0, t)
}

func checkNode(node *gohan_sync.Node, key, value string, children int, t *testing.T) {
	if node.Key != key {
		t.Fatalf("expected key %s has %s", key, node.Key)
	}
	if node.Value != value {
		t.Fatalf("expected value %s has %s", value, node.Value)
	}
	if len(node.Children) != children {
		t.Fatalf("expected to has %d children has %d", children, len(node.Children))
	}
}

func TestNotIncludedPaths(t *testing.T) {
	sync := newSync(t)
	sync.etcdClient.Delete(context.Background(), "/", etcd.WithPrefix())

	err := sync.Update("/path/to/somewhere", "test")
	if err != nil {
		t.Fatalf("unexpected error")
	}
	err = sync.Update("/pathnottobeincluded", "should not appear")
	if err != nil {
		t.Fatalf("unexpected error")
	}

	nodes, err := sync.Fetch("/path")
	if err != nil {
		t.Fatalf("unexpected error %s", err.Error())
	}

	checkNode(nodes, "/path", "", 1, t)
	pathTo := nodes.Children[0]
	checkNode(pathTo, "/path/to", "", 1, t)
	checkNode(pathTo.Children[0], "/path/to/somewhere", "test", 0, t)

	_, err = sync.Fetch("/path/not")
	if err == nil || !strings.Contains(err.Error(), fmt.Sprintf("Key not found (%s)", "/path/not")) {
		t.Fatalf("unexpected error %s", err.Error())
	}

	err = sync.Update("/path/to/notbeincluded", "should not appear")
	if err != nil {
		t.Fatalf("unexpected error")
	}
	_, err = sync.Fetch("/path/to/not")
	if err == nil || !strings.Contains(err.Error(), fmt.Sprintf("Key not found (%s)", "/path/to/not")) {
		t.Fatalf("unexpected error %s", err.Error())
	}
}

func TestSubstr(t *testing.T) {
	expectToEqual := func(a, b string) {
		if a != b {
			t.Fatalf("expected %s to equal %s", a, b)
		}
	}
	expectToEqual(substrN("/a/b/c/d", "/", 1), "/a")
	expectToEqual(substrN("/a/b/c/d", "/", 2), "/a/b")
	expectToEqual(substrN("/a/b/c/d", "/", 3), "/a/b/c")
	expectToEqual(substrN("/a/b/c/d", "/", 4), "/a/b/c/d")
	expectToEqual(substrN("/a/b/c/d", "/", 5), "/a/b/c/d")
}

func newSync(t *testing.T) *Sync {
	sync, err := NewSync(endpoints, time.Millisecond*100)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	return sync
}
