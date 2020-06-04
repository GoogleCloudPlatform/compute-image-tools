// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain (interfaces: ObjectHandleInterface)

// Package mocks is a generated GoMock package.
package mocks

import (
	storage "cloud.google.com/go/storage"
	domain "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	gomock "github.com/golang/mock/gomock"
	io "io"
	reflect "reflect"
)

// MockObjectHandleInterface is a mock of ObjectHandleInterface interface
type MockObjectHandleInterface struct {
	ctrl     *gomock.Controller
	recorder *MockObjectHandleInterfaceMockRecorder
}

// MockObjectHandleInterfaceMockRecorder is the mock recorder for MockObjectHandleInterface
type MockObjectHandleInterfaceMockRecorder struct {
	mock *MockObjectHandleInterface
}

// NewMockObjectHandleInterface creates a new mock instance
func NewMockObjectHandleInterface(ctrl *gomock.Controller) *MockObjectHandleInterface {
	mock := &MockObjectHandleInterface{ctrl: ctrl}
	mock.recorder = &MockObjectHandleInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockObjectHandleInterface) EXPECT() *MockObjectHandleInterfaceMockRecorder {
	return m.recorder
}

// Delete mocks base method
func (m *MockObjectHandleInterface) Delete() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete")
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete
func (mr *MockObjectHandleInterfaceMockRecorder) Delete() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockObjectHandleInterface)(nil).Delete))
}

// GetObjectHandle mocks base method
func (m *MockObjectHandleInterface) GetObjectHandle() *storage.ObjectHandle {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetObjectHandle")
	ret0, _ := ret[0].(*storage.ObjectHandle)
	return ret0
}

// GetObjectHandle indicates an expected call of GetObjectHandle
func (mr *MockObjectHandleInterfaceMockRecorder) GetObjectHandle() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetObjectHandle", reflect.TypeOf((*MockObjectHandleInterface)(nil).GetObjectHandle))
}

// NewReader mocks base method
func (m *MockObjectHandleInterface) NewReader() (io.ReadCloser, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewReader")
	ret0, _ := ret[0].(io.ReadCloser)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewReader indicates an expected call of NewReader
func (mr *MockObjectHandleInterfaceMockRecorder) NewReader() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewReader", reflect.TypeOf((*MockObjectHandleInterface)(nil).NewReader))
}

// NewWriter mocks base method
func (m *MockObjectHandleInterface) NewWriter() io.WriteCloser {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewWriter")
	ret0, _ := ret[0].(io.WriteCloser)
	return ret0
}

// NewWriter indicates an expected call of NewWriter
func (mr *MockObjectHandleInterfaceMockRecorder) NewWriter() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewWriter", reflect.TypeOf((*MockObjectHandleInterface)(nil).NewWriter))
}

// ObjectName mocks base method
func (m *MockObjectHandleInterface) ObjectName() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ObjectName")
	ret0, _ := ret[0].(string)
	return ret0
}

// ObjectName indicates an expected call of ObjectName
func (mr *MockObjectHandleInterfaceMockRecorder) ObjectName() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ObjectName", reflect.TypeOf((*MockObjectHandleInterface)(nil).ObjectName))
}

// RunComposer mocks base method
func (m *MockObjectHandleInterface) RunComposer(arg0 ...domain.ObjectHandleInterface) (*storage.ObjectAttrs, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range arg0 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "RunComposer", varargs...)
	ret0, _ := ret[0].(*storage.ObjectAttrs)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// RunComposer indicates an expected call of RunComposer
func (mr *MockObjectHandleInterfaceMockRecorder) RunComposer(arg0 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RunComposer", reflect.TypeOf((*MockObjectHandleInterface)(nil).RunComposer), arg0...)
}

// RunCopier mocks base method
func (m *MockObjectHandleInterface) RunCopier(arg0 domain.ObjectHandleInterface) (*storage.ObjectAttrs, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RunCopier", arg0)
	ret0, _ := ret[0].(*storage.ObjectAttrs)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// RunCopier indicates an expected call of RunCopier
func (mr *MockObjectHandleInterfaceMockRecorder) RunCopier(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RunCopier", reflect.TypeOf((*MockObjectHandleInterface)(nil).RunCopier), arg0)
}
