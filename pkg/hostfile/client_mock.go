// Code generated by MockGen. DO NOT EDIT.
// Source: pkg/hostfile/client.go
//
// Generated by this command:
//
//	mockgen -source=pkg/hostfile/client.go -destination=pkg/hostfile/client_mock.go -package=hostfile
//

// Package hostfile is a generated GoMock package.
package hostfile

import (
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockHostfile is a mock of Hostfile interface.
type MockHostfile struct {
	ctrl     *gomock.Controller
	recorder *MockHostfileMockRecorder
}

// MockHostfileMockRecorder is the mock recorder for MockHostfile.
type MockHostfileMockRecorder struct {
	mock *MockHostfile
}

// NewMockHostfile creates a new mock instance.
func NewMockHostfile(ctrl *gomock.Controller) *MockHostfile {
	mock := &MockHostfile{ctrl: ctrl}
	mock.recorder = &MockHostfileMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockHostfile) EXPECT() *MockHostfileMockRecorder {
	return m.recorder
}

// AddHost mocks base method.
func (m *MockHostfile) AddHost(ip, hostname string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddHost", ip, hostname)
	ret0, _ := ret[0].(error)
	return ret0
}

// AddHost indicates an expected call of AddHost.
func (mr *MockHostfileMockRecorder) AddHost(ip, hostname any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddHost", reflect.TypeOf((*MockHostfile)(nil).AddHost), ip, hostname)
}

// RemoveHost mocks base method.
func (m *MockHostfile) RemoveHost(hostname string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RemoveHost", hostname)
	ret0, _ := ret[0].(error)
	return ret0
}

// RemoveHost indicates an expected call of RemoveHost.
func (mr *MockHostfileMockRecorder) RemoveHost(hostname any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveHost", reflect.TypeOf((*MockHostfile)(nil).RemoveHost), hostname)
}
