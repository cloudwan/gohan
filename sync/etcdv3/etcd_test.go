package etcdv3

import (
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
	ctx := context.Background()

	sync := newSync(t, ctx)
	sync.cleanup()

	path := "/path/to/somewhere"
	data := "blabla"
	sync.mustUpdate(path, data)
	sync.mustExist(path, data, 0)

	sync.mustDelete(path, false)
	sync.mustNotExist(path)
}

func TestEmptyUpdate(t *testing.T) {
	ctx := context.Background()

	sync := newSync(t, ctx)
	sync.cleanup()

	path := "/path/to/somewhere"
	data := ""
	sync.mustUpdate(path, data)

	// not found because v3 doesn't support directories
	sync.mustNotExist(path)
}

func TestRecursiveUpdate(t *testing.T) {
	ctx := context.Background()

	sync := newSync(t, ctx)
	sync.cleanup()

	sync.mustUpdate("/a/b/c/d/1", "test1")
	sync.mustUpdate("/a/b/c/d/2", "test2")
	sync.mustUpdate("/a/b/c/d/3", "test3")
	sync.mustUpdate("/a/b/e/d/1", "test4")
	sync.mustUpdate("/a/b/e/d/2", "test5")
	sync.mustUpdate("/a/b/e/d/3", "test6")

	node := sync.mustExist("/a", "", 1)
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

	sync.mustExist("/a/b/c", "", 1)
}

func TestLockUnblocking(t *testing.T) {
	ctx := context.Background()

	sync0 := newSync(t, ctx)
	sync1 := newSync(t, ctx)
	sync0.cleanup()

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

	err = sync0.Unlock(ctx, path)
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

	sync0 := newSync(t, ctx)
	sync1 := newSync(t, ctx)
	sync0.cleanup()

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

	err = sync0.Unlock(ctx, path)
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sync := newSync(t, ctx)
	sync.cleanup()

	path := "/path/to/watch/without/revision"
	sync.etcdClient.Put(ctx, path+"/existing", `{"existing": true}`)

	responseChan := sync.Watch(ctx, path, gohan_sync.RevisionCurrent)

	resp := <-responseChan
	if resp.Action != "get" || resp.Key != path+"/existing" || resp.Data["existing"].(bool) != true {
		t.Fatalf("mismatch response: %+v", resp)
	}

	sync.etcdClient.Put(ctx, path+"/new", `{"existing": false}`)
	resp = <-responseChan
	if resp.Action != "set" || resp.Key != path+"/new" || resp.Data["existing"].(bool) != false {
		t.Fatalf("mismatch response: %+v", resp)
	}

	sync.etcdClient.Delete(ctx, path+"/existing")
	resp = <-responseChan
	if resp.Action != "delete" || resp.Key != path+"/existing" || len(resp.Data) != 0 {
		t.Fatalf("mismatch response: %+v", resp)
	}
}

func TestWatchWithRevision(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sync := newSync(t, ctx)
	sync.cleanup()

	path := "/path/to/watch/with/revision"

	putResponse, err := sync.etcdClient.Put(ctx, path+"/existing", `{"existing": true}`)
	if err != nil {
		t.Fatalf("failed to put key: %s", err)
	}
	startRev := putResponse.Header.Revision

	putResponse, err = sync.etcdClient.Put(ctx, path+"/new", `{"existing": false}`)
	if err != nil {
		t.Fatalf("failed to update key: %s", err)
	}
	secondRevision := putResponse.Header.Revision

	responseChan := sync.Watch(ctx, path, startRev+1)

	resp := <-responseChan
	if resp.Key != path+"/new" || resp.Data["existing"].(bool) != false || resp.Revision != secondRevision {
		t.Fatalf("mismatch response: %+v, expecting /new, existing==false, revision==%d", resp, secondRevision)
	}

	putResponse, err = sync.etcdClient.Put(ctx, path+"/third", `{"existing": false}`)
	if err != nil {
		t.Fatalf("failed to update key: %s", err)
	}
	thirdRevision := putResponse.Header.Revision

	resp = <-responseChan
	if resp.Key != path+"/third" || resp.Data["existing"].(bool) != false || resp.Revision != thirdRevision {
		t.Fatalf("mismatch response: %+v, expecting /third, existing==false, revision==%d", resp, thirdRevision)
	}
}

func TestShouldStartWatchingAtSpecifiedRevision(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sync := newSync(t)
	sync.etcdClient.Delete(ctx, "/", etcd.WithPrefix())

	path := "/path/to/watch/with/starting/revision"

	putResponse, err := sync.etcdClient.Put(ctx, path, `{"version": "1"}`)
	if err != nil {
		t.Fatalf("failed to update key: %s", err)
	}
	firstRevision := putResponse.Header.Revision

	putResponse, err = sync.etcdClient.Put(ctx, path, `{"version": "2"}`)
	if err != nil {
		t.Fatalf("failed to update key: %s", err)
	}
	secondRevision := putResponse.Header.Revision

	responseChan := sync.Watch(ctx, path, firstRevision-1)

	resp := <-responseChan
	if resp.Key != path || resp.Data["version"].(string) != "1" || resp.Revision != firstRevision {
		t.Fatalf("mismatch response: %+v, expecting version==1, revision==%d", resp, firstRevision)
	}

	resp = <-responseChan
	if resp.Key != path || resp.Data["version"].(string) != "2" || resp.Revision != secondRevision {
		t.Fatalf("mismatch response: %+v, expecting version==2, revision==%d", resp, secondRevision)
	}
}

func TestFetchMultipleNodes(t *testing.T) {
	ctx := context.Background()

	sync := newSync(t, ctx)
	sync.cleanup()

	sync.mustUpdate("/path/to/somewhere", "test1")
	sync.mustUpdate("/path/to/elsewhere", "test2")
	sync.mustUpdate("/path/notto/elsewhere", "test3")

	nodes := sync.mustExist("/path", "", 2)

	checkNode(nodes.Children[0], "/path/notto", "", 1, t)
	checkNode(nodes.Children[0].Children[0], "/path/notto/elsewhere", "test3", 0, t)

	checkNode(nodes.Children[1], "/path/to", "", 2, t)
	checkNode(nodes.Children[1].Children[0], "/path/to/elsewhere", "test2", 0, t)
	checkNode(nodes.Children[1].Children[1], "/path/to/somewhere", "test1", 0, t)

}

func TestUpdateWithoutPath(t *testing.T) {
	ctx := context.Background()
	sync := newSync(t, ctx)

	sync.mustUpdate("post-migration", "test")
	sync.mustExist("post-migration", "test", 0)
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
	ctx := context.Background()

	sync := newSync(t, ctx)
	sync.cleanup()

	sync.mustUpdate("/path/to/somewhere", "test")
	sync.mustUpdate("/pathnottobeincluded", "should not appear")

	nodes := sync.mustExist("/path", "", 1)
	pathTo := nodes.Children[0]
	checkNode(pathTo, "/path/to", "", 1, t)
	checkNode(pathTo.Children[0], "/path/to/somewhere", "test", 0, t)

	sync.mustNotExist("/path/not")

	sync.mustUpdate("/path/to/notbeincluded", "should not appear")
	sync.mustNotExist("/path/to/not")
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

func TestCASShouldUpdateWhenRevisionDidNotChange(t *testing.T) {
	ctx := context.Background()

	sync := newSync(t, ctx)
	sync.cleanup()

	path := "/path/to/cas"
	data := "initial_data"
	sync.mustUpdate(path, data)

	currentRev := sync.getCurrentRevision(path)

	newData := "new_data"
	swapped := sync.compareAndSwap(path, newData, sync.ByRevision(currentRev))
	if !swapped {
		t.Fatalf("Value was not swapped")
	}

	sync.mustExist(path, newData, 0)
}

func TestCASShouldNotUpdateWhenRevisionChanged(t *testing.T) {
	ctx := context.Background()

	sync := newSync(t, ctx)
	sync.cleanup()

	path := "/path/to/cas"
	data := "initial_data"
	sync.mustUpdate(path, data)

	initialRev := sync.getCurrentRevision(path)

	updatedData := "updated_data"
	sync.mustUpdate(path, updatedData)

	newData := "new_data"
	swapped := sync.compareAndSwap(path, newData, sync.ByRevision(initialRev))

	if swapped {
		t.Fatalf("Value was unexpectedly swapped")
	}

	sync.mustExist(path, updatedData, 0)
}

func TestCASShouldUpdateWhenValueDidNotChange(t *testing.T) {
	ctx := context.Background()

	sync := newSync(t, ctx)
	sync.cleanup()

	path := "/path/to/cas"
	data := "initial_data"
	sync.mustUpdate(path, data)

	newData := "new_data"
	swapped := sync.compareAndSwap(path, newData, sync.ByValue(data))
	if !swapped {
		t.Fatalf("Value was not swapped")
	}

	sync.mustExist(path, newData, 0)
}

func TestCASShouldNotUpdateWhenValueChanged(t *testing.T) {
	ctx := context.Background()

	sync := newSync(t, ctx)
	sync.cleanup()

	path := "/path/to/cas"
	data := "initial_data"
	sync.mustUpdate(path, data)

	updatedData := "updated_data"
	sync.mustUpdate(path, updatedData)

	newData := "new_data"
	swapped := sync.compareAndSwap(path, newData, sync.ByValue(data))
	if swapped {
		t.Fatalf("Value was unexpectedly swapped")
	}

	sync.mustExist(path, updatedData, 0)
}

func TestCASShouldUpdateWhenValueAndRevisionDidNotChange(t *testing.T) {
	ctx := context.Background()

	sync := newSync(t, ctx)
	sync.cleanup()

	path := "/path/to/cas"
	data := "initial_data"
	sync.mustUpdate(path, data)

	initialRev := sync.getCurrentRevision(path)

	newData := "new_data"
	swapped := sync.compareAndSwap(path, newData, sync.ByValue(data), sync.ByRevision(initialRev))
	if !swapped {
		t.Fatalf("Value was not swapped")
	}

	sync.mustExist(path, newData, 0)
}

type testedSync struct {
	*Sync
	t   *testing.T
	ctx context.Context
}

func (sync *testedSync) cleanup() {
	sync.etcdClient.Delete(sync.ctx, "/", etcd.WithPrefix())
}

func (sync *testedSync) getCurrentRevision(key string) int64 {
	node, err := sync.Fetch(sync.ctx, key)
	if err != nil {
		sync.t.Fatalf("Fetch failed %s", err)
	}

	return node.Revision
}

func (sync *testedSync) mustUpdate(path, data string) {
	err := sync.Update(sync.ctx, path, data)
	if err != nil {
		sync.t.Fatalf("unexpected error on updating %s with %s: %s", path, data, err)
	}
}

func (sync *testedSync) mustFetch(path string) *gohan_sync.Node {
	node, err := sync.Fetch(sync.ctx, path)
	if err != nil {
		sync.t.Fatalf("unexpected error on fetching %s failed: %s", path, err)
	}

	return node
}

func (sync *testedSync) mustDelete(path string, prefix bool) {
	if err := sync.Delete(sync.ctx, path, prefix); err != nil {
		sync.t.Fatalf("unexpected error on deleting %s, prefix: %v: %s", path, prefix, err)
	}
}

func (sync *testedSync) compareAndSwap(path, data string, conds ...gohan_sync.CASCondition) bool {
	swapped, err := sync.CompareAndSwap(sync.ctx, path, data, conds...)
	if err != nil {
		sync.t.Fatalf("CAS failed: %s", err)
	}

	return swapped
}

func (sync *testedSync) mustNotExist(path string) {
	node, err := sync.Fetch(sync.ctx, path)
	if err == nil || err != KeyNotFound {
		sync.t.Fatalf("path %s should not exist, error: %s, value: %+v", path, err, node)
	}
}

func (sync *testedSync) mustExist(path, value string, children int) *gohan_sync.Node {
	node := sync.mustFetch(path)
	checkNode(node, path, value, children, sync.t)

	return node
}

func newSync(t *testing.T, ctx context.Context) *testedSync {
	sync, err := NewSync(endpoints, time.Millisecond*100)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	return &testedSync{
		sync, t, ctx,
	}
}
