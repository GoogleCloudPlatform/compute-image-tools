//  Copyright 2020 Google Inc. All Rights Reserved.
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

// Package mocks is a generated GoMock package.
package mocks

import (
	disk "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/disk"
	domain "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export/domain"
	gomock "github.com/golang/mock/gomock"
	v1 "google.golang.org/api/compute/v1"
	reflect "reflect"
)

// MockOvfDescriptorGenerator is a mock of OvfDescriptorGenerator interface
type MockOvfDescriptorGenerator struct {
	ctrl     *gomock.Controller
	recorder *MockOvfDescriptorGeneratorMockRecorder
}

// MockOvfDescriptorGeneratorMockRecorder is the mock recorder for MockOvfDescriptorGenerator
type MockOvfDescriptorGeneratorMockRecorder struct {
	mock *MockOvfDescriptorGenerator
}

// NewMockOvfDescriptorGenerator creates a new mock instance
func NewMockOvfDescriptorGenerator(ctrl *gomock.Controller) *MockOvfDescriptorGenerator {
	mock := &MockOvfDescriptorGenerator{ctrl: ctrl}
	mock.recorder = &MockOvfDescriptorGeneratorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockOvfDescriptorGenerator) EXPECT() *MockOvfDescriptorGeneratorMockRecorder {
	return m.recorder
}

// Cancel mocks base method
func (m *MockOvfDescriptorGenerator) Cancel(arg0 string) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Cancel", arg0)
	ret0, _ := ret[0].(bool)
	return ret0
}

// Cancel indicates an expected call of Cancel
func (mr *MockOvfDescriptorGeneratorMockRecorder) Cancel(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Cancel", reflect.TypeOf((*MockOvfDescriptorGenerator)(nil).Cancel), arg0)
}

// GenerateAndWriteOVFDescriptor mocks base method
func (m *MockOvfDescriptorGenerator) GenerateAndWriteOVFDescriptor(arg0 *v1.Instance, arg1 []*domain.ExportedDisk, arg2, arg3 string, arg4 *disk.InspectionResult) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GenerateAndWriteOVFDescriptor", arg0, arg1, arg2, arg3, arg4)
	ret0, _ := ret[0].(error)
	return ret0
}

// GenerateAndWriteOVFDescriptor indicates an expected call of GenerateAndWriteOVFDescriptor
func (mr *MockOvfDescriptorGeneratorMockRecorder) GenerateAndWriteOVFDescriptor(arg0, arg1, arg2, arg3, arg4 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GenerateAndWriteOVFDescriptor", reflect.TypeOf((*MockOvfDescriptorGenerator)(nil).GenerateAndWriteOVFDescriptor), arg0, arg1, arg2, arg3, arg4)
}
