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
// Source: github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain (interfaces: ResourceLocationRetrieverInterface)

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockResourceLocationRetrieverInterface is a mock of ResourceLocationRetrieverInterface interface.
type MockResourceLocationRetrieverInterface struct {
	ctrl     *gomock.Controller
	recorder *MockResourceLocationRetrieverInterfaceMockRecorder
}

// MockResourceLocationRetrieverInterfaceMockRecorder is the mock recorder for MockResourceLocationRetrieverInterface.
type MockResourceLocationRetrieverInterfaceMockRecorder struct {
	mock *MockResourceLocationRetrieverInterface
}

// NewMockResourceLocationRetrieverInterface creates a new mock instance.
func NewMockResourceLocationRetrieverInterface(ctrl *gomock.Controller) *MockResourceLocationRetrieverInterface {
	mock := &MockResourceLocationRetrieverInterface{ctrl: ctrl}
	mock.recorder = &MockResourceLocationRetrieverInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockResourceLocationRetrieverInterface) EXPECT() *MockResourceLocationRetrieverInterfaceMockRecorder {
	return m.recorder
}

// GetLargestStorageLocation mocks base method.
func (m *MockResourceLocationRetrieverInterface) GetLargestStorageLocation(arg0 string) string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLargestStorageLocation", arg0)
	ret0, _ := ret[0].(string)
	return ret0
}

// GetLargestStorageLocation indicates an expected call of GetLargestStorageLocation.
func (mr *MockResourceLocationRetrieverInterfaceMockRecorder) GetLargestStorageLocation(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLargestStorageLocation", reflect.TypeOf((*MockResourceLocationRetrieverInterface)(nil).GetLargestStorageLocation), arg0)
}

// GetZone mocks base method.
func (m *MockResourceLocationRetrieverInterface) GetZone(arg0, arg1 string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetZone", arg0, arg1)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetZone indicates an expected call of GetZone.
func (mr *MockResourceLocationRetrieverInterfaceMockRecorder) GetZone(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetZone", reflect.TypeOf((*MockResourceLocationRetrieverInterface)(nil).GetZone), arg0, arg1)
}
