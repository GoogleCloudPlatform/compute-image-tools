//  Copyright 2019 Google Inc. All Rights Reserved.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/domain (interfaces: OvfDescriptorLoaderInterface)

// Package mocks is a generated GoMock package.
package mocks

import (
	ovf "github.com/GoogleCloudPlatform/compute-image-tools/third_party/govmomi/ovf"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockOvfDescriptorLoaderInterface is a mock of OvfDescriptorLoaderInterface interface
type MockOvfDescriptorLoaderInterface struct {
	ctrl     *gomock.Controller
	recorder *MockOvfDescriptorLoaderInterfaceMockRecorder
}

// MockOvfDescriptorLoaderInterfaceMockRecorder is the mock recorder for MockOvfDescriptorLoaderInterface
type MockOvfDescriptorLoaderInterfaceMockRecorder struct {
	mock *MockOvfDescriptorLoaderInterface
}

// NewMockOvfDescriptorLoaderInterface creates a new mock instance
func NewMockOvfDescriptorLoaderInterface(ctrl *gomock.Controller) *MockOvfDescriptorLoaderInterface {
	mock := &MockOvfDescriptorLoaderInterface{ctrl: ctrl}
	mock.recorder = &MockOvfDescriptorLoaderInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockOvfDescriptorLoaderInterface) EXPECT() *MockOvfDescriptorLoaderInterfaceMockRecorder {
	return m.recorder
}

// Load mocks base method
func (m *MockOvfDescriptorLoaderInterface) Load(arg0 string) (*ovf.Envelope, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Load", arg0)
	ret0, _ := ret[0].(*ovf.Envelope)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Load indicates an expected call of Load
func (mr *MockOvfDescriptorLoaderInterfaceMockRecorder) Load(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Load", reflect.TypeOf((*MockOvfDescriptorLoaderInterface)(nil).Load), arg0)
}
