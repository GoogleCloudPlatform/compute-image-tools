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
// Source: github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_vm_image_import/domain (interfaces: ComputeServiceInterface)

// Package mocks is a generated GoMock package.
package mocks

import (
	gomock "github.com/golang/mock/gomock"
	v1 "google.golang.org/api/compute/v1"
	reflect "reflect"
)

// MockComputeServiceInterface is a mock of ComputeServiceInterface interface
type MockComputeServiceInterface struct {
	ctrl     *gomock.Controller
	recorder *MockComputeServiceInterfaceMockRecorder
}

// MockComputeServiceInterfaceMockRecorder is the mock recorder for MockComputeServiceInterface
type MockComputeServiceInterfaceMockRecorder struct {
	mock *MockComputeServiceInterface
}

// NewMockComputeServiceInterface creates a new mock instance
func NewMockComputeServiceInterface(ctrl *gomock.Controller) *MockComputeServiceInterface {
	mock := &MockComputeServiceInterface{ctrl: ctrl}
	mock.recorder = &MockComputeServiceInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockComputeServiceInterface) EXPECT() *MockComputeServiceInterfaceMockRecorder {
	return m.recorder
}

// GetZones mocks base method
func (m *MockComputeServiceInterface) GetZones(arg0 string) ([]*v1.Zone, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetZones", arg0)
	ret0, _ := ret[0].([]*v1.Zone)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetZones indicates an expected call of GetZones
func (mr *MockComputeServiceInterfaceMockRecorder) GetZones(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetZones", reflect.TypeOf((*MockComputeServiceInterface)(nil).GetZones), arg0)
}
