// Code generated by MockGen. DO NOT EDIT.
// Source: ../sync.go

// Package mock_sync is a generated GoMock package.
package mock_sync

import (
	context "context"
	reflect "reflect"

	sync "github.com/cloudwan/gohan/sync"
	gomock "github.com/golang/mock/gomock"
)

// MockSync is a mock of Sync interface
type MockSync struct {
	ctrl     *gomock.Controller
	recorder *MockSyncMockRecorder
}

// MockSyncMockRecorder is the mock recorder for MockSync
type MockSyncMockRecorder struct {
	mock *MockSync
}

// NewMockSync creates a new mock instance
func NewMockSync(ctrl *gomock.Controller) *MockSync {
	mock := &MockSync{ctrl: ctrl}
	mock.recorder = &MockSyncMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockSync) EXPECT() *MockSyncMockRecorder {
	return m.recorder
}

// HasLock mocks base method
func (m *MockSync) HasLock(path string) bool {
	ret := m.ctrl.Call(m, "HasLock", path)
	ret0, _ := ret[0].(bool)
	return ret0
}

// HasLock indicates an expected call of HasLock
func (mr *MockSyncMockRecorder) HasLock(path interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HasLock", reflect.TypeOf((*MockSync)(nil).HasLock), path)
}

// Lock mocks base method
func (m *MockSync) Lock(ctx context.Context, path string, block bool) (chan struct{}, error) {
	ret := m.ctrl.Call(m, "Lock", ctx, path, block)
	ret0, _ := ret[0].(chan struct{})
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Lock indicates an expected call of Lock
func (mr *MockSyncMockRecorder) Lock(ctx, path, block interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Lock", reflect.TypeOf((*MockSync)(nil).Lock), ctx, path, block)
}

// Unlock mocks base method
func (m *MockSync) Unlock(path string) error {
	ret := m.ctrl.Call(m, "Unlock", path)
	ret0, _ := ret[0].(error)
	return ret0
}

// Unlock indicates an expected call of Unlock
func (mr *MockSyncMockRecorder) Unlock(path interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Unlock", reflect.TypeOf((*MockSync)(nil).Unlock), path)
}

// Fetch mocks base method
func (m *MockSync) Fetch(path string) (*sync.Node, error) {
	ret := m.ctrl.Call(m, "Fetch", path)
	ret0, _ := ret[0].(*sync.Node)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Fetch indicates an expected call of Fetch
func (mr *MockSyncMockRecorder) Fetch(path interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Fetch", reflect.TypeOf((*MockSync)(nil).Fetch), path)
}

// Update mocks base method
func (m *MockSync) Update(path, json string) error {
	ret := m.ctrl.Call(m, "Update", path, json)
	ret0, _ := ret[0].(error)
	return ret0
}

// Update indicates an expected call of Update
func (mr *MockSyncMockRecorder) Update(path, json interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockSync)(nil).Update), path, json)
}

// Delete mocks base method
func (m *MockSync) Delete(path string, prefix bool) error {
	ret := m.ctrl.Call(m, "Delete", path, prefix)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete
func (mr *MockSyncMockRecorder) Delete(path, prefix interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockSync)(nil).Delete), path, prefix)
}

// Watch mocks base method
func (m *MockSync) Watch(path string, responseChan chan *sync.Event, stopChan chan bool, revision int64) error {
	ret := m.ctrl.Call(m, "Watch", path, responseChan, stopChan, revision)
	ret0, _ := ret[0].(error)
	return ret0
}

// Watch indicates an expected call of Watch
func (mr *MockSyncMockRecorder) Watch(path, responseChan, stopChan, revision interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Watch", reflect.TypeOf((*MockSync)(nil).Watch), path, responseChan, stopChan, revision)
}

// WatchContext mocks base method
func (m *MockSync) WatchContext(ctx context.Context, path string, revision int64) <-chan *sync.Event {
	ret := m.ctrl.Call(m, "WatchContext", ctx, path, revision)
	ret0, _ := ret[0].(<-chan *sync.Event)
	return ret0
}

// WatchContext indicates an expected call of WatchContext
func (mr *MockSyncMockRecorder) WatchContext(ctx, path, revision interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchContext", reflect.TypeOf((*MockSync)(nil).WatchContext), ctx, path, revision)
}

// GetProcessID mocks base method
func (m *MockSync) GetProcessID() string {
	ret := m.ctrl.Call(m, "GetProcessID")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetProcessID indicates an expected call of GetProcessID
func (mr *MockSyncMockRecorder) GetProcessID() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetProcessID", reflect.TypeOf((*MockSync)(nil).GetProcessID))
}

// Close mocks base method
func (m *MockSync) Close() {
	m.ctrl.Call(m, "Close")
}

// Close indicates an expected call of Close
func (mr *MockSyncMockRecorder) Close() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockSync)(nil).Close))
}
