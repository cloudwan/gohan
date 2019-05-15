// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/cloudwan/gohan/server/middleware (interfaces: IdentityService)

// Package mocks is a generated GoMock package.
package mocks

import (
	schema "github.com/cloudwan/gohan/schema"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockIdentityService is a mock of IdentityService interface
type MockIdentityService struct {
	ctrl     *gomock.Controller
	recorder *MockIdentityServiceMockRecorder
}

// MockIdentityServiceMockRecorder is the mock recorder for MockIdentityService
type MockIdentityServiceMockRecorder struct {
	mock *MockIdentityService
}

// NewMockIdentityService creates a new mock instance
func NewMockIdentityService(ctrl *gomock.Controller) *MockIdentityService {
	mock := &MockIdentityService{ctrl: ctrl}
	mock.recorder = &MockIdentityServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockIdentityService) EXPECT() *MockIdentityServiceMockRecorder {
	return m.recorder
}

// GetServiceAuthorization mocks base method
func (m *MockIdentityService) GetServiceAuthorization() (schema.Authorization, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetServiceAuthorization")
	ret0, _ := ret[0].(schema.Authorization)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetServiceAuthorization indicates an expected call of GetServiceAuthorization
func (mr *MockIdentityServiceMockRecorder) GetServiceAuthorization() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetServiceAuthorization", reflect.TypeOf((*MockIdentityService)(nil).GetServiceAuthorization))
}

// GetServiceTokenID mocks base method
func (m *MockIdentityService) GetServiceTokenID() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetServiceTokenID")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetServiceTokenID indicates an expected call of GetServiceTokenID
func (mr *MockIdentityServiceMockRecorder) GetServiceTokenID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetServiceTokenID", reflect.TypeOf((*MockIdentityService)(nil).GetServiceTokenID))
}

// GetTenantID mocks base method
func (m *MockIdentityService) GetTenantID(arg0 string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTenantID", arg0)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTenantID indicates an expected call of GetTenantID
func (mr *MockIdentityServiceMockRecorder) GetTenantID(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTenantID", reflect.TypeOf((*MockIdentityService)(nil).GetTenantID), arg0)
}

// GetTenantName mocks base method
func (m *MockIdentityService) GetTenantName(arg0 string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTenantName", arg0)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTenantName indicates an expected call of GetTenantName
func (mr *MockIdentityServiceMockRecorder) GetTenantName(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTenantName", reflect.TypeOf((*MockIdentityService)(nil).GetTenantName), arg0)
}

// ValidateDomainID mocks base method
func (m *MockIdentityService) ValidateDomainID(arg0 string) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ValidateDomainID", arg0)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ValidateDomainID indicates an expected call of ValidateDomainID
func (mr *MockIdentityServiceMockRecorder) ValidateDomainID(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ValidateDomainID", reflect.TypeOf((*MockIdentityService)(nil).ValidateDomainID), arg0)
}

// ValidateTenantID mocks base method
func (m *MockIdentityService) ValidateTenantID(arg0 string) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ValidateTenantID", arg0)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ValidateTenantID indicates an expected call of ValidateTenantID
func (mr *MockIdentityServiceMockRecorder) ValidateTenantID(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ValidateTenantID", reflect.TypeOf((*MockIdentityService)(nil).ValidateTenantID), arg0)
}

// VerifyToken mocks base method
func (m *MockIdentityService) VerifyToken(arg0 string) (schema.Authorization, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "VerifyToken", arg0)
	ret0, _ := ret[0].(schema.Authorization)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// VerifyToken indicates an expected call of VerifyToken
func (mr *MockIdentityServiceMockRecorder) VerifyToken(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "VerifyToken", reflect.TypeOf((*MockIdentityService)(nil).VerifyToken), arg0)
}
