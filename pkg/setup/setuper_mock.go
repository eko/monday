// Code generated by MockGen. DO NOT EDIT.
// Source: pkg/setup/setuper.go
//
// Generated by this command:
//
//	mockgen -source=pkg/setup/setuper.go -destination=pkg/setup/setuper_mock.go -package=setup
//

// Package setup is a generated GoMock package.
package setup

import (
	reflect "reflect"

	config "github.com/eko/monday/pkg/config"
	gomock "go.uber.org/mock/gomock"
)

// MockSetuper is a mock of Setuper interface.
type MockSetuper struct {
	ctrl     *gomock.Controller
	recorder *MockSetuperMockRecorder
}

// MockSetuperMockRecorder is the mock recorder for MockSetuper.
type MockSetuperMockRecorder struct {
	mock *MockSetuper
}

// NewMockSetuper creates a new mock instance.
func NewMockSetuper(ctrl *gomock.Controller) *MockSetuper {
	mock := &MockSetuper{ctrl: ctrl}
	mock.recorder = &MockSetuperMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSetuper) EXPECT() *MockSetuperMockRecorder {
	return m.recorder
}

// Setup mocks base method.
func (m *MockSetuper) Setup(application *config.Application) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Setup", application)
}

// Setup indicates an expected call of Setup.
func (mr *MockSetuperMockRecorder) Setup(application any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Setup", reflect.TypeOf((*MockSetuper)(nil).Setup), application)
}

// SetupAll mocks base method.
func (m *MockSetuper) SetupAll() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetupAll")
}

// SetupAll indicates an expected call of SetupAll.
func (mr *MockSetuperMockRecorder) SetupAll() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetupAll", reflect.TypeOf((*MockSetuper)(nil).SetupAll))
}
