// Code generated by mockery v2.43.2. DO NOT EDIT.

package mocks

import (
	mock "github.com/stretchr/testify/mock"
	v1 "k8s.io/client-go/kubernetes/typed/apps/v1"
)

// DeploymentsGetter is an autogenerated mock type for the DeploymentsGetter type
type DeploymentsGetter struct {
	mock.Mock
}

// Deployments provides a mock function with given fields: namespace
func (_m *DeploymentsGetter) Deployments(namespace string) v1.DeploymentInterface {
	ret := _m.Called(namespace)

	if len(ret) == 0 {
		panic("no return value specified for Deployments")
	}

	var r0 v1.DeploymentInterface
	if rf, ok := ret.Get(0).(func(string) v1.DeploymentInterface); ok {
		r0 = rf(namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(v1.DeploymentInterface)
		}
	}

	return r0
}

// NewDeploymentsGetter creates a new instance of DeploymentsGetter. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewDeploymentsGetter(t interface {
	mock.TestingT
	Cleanup(func())
}) *DeploymentsGetter {
	mock := &DeploymentsGetter{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
