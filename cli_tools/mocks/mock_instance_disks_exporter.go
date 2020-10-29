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
	domain "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export/domain"
	gomock "github.com/golang/mock/gomock"
	v1 "google.golang.org/api/compute/v1"
	reflect "reflect"
)

// MockInstanceDisksExporter is a mock of InstanceDisksExporter interface
type MockInstanceDisksExporter struct {
	ctrl     *gomock.Controller
	recorder *MockInstanceDisksExporterMockRecorder
}

// MockInstanceDisksExporterMockRecorder is the mock recorder for MockInstanceDisksExporter
type MockInstanceDisksExporterMockRecorder struct {
	mock *MockInstanceDisksExporter
}

// NewMockInstanceDisksExporter creates a new mock instance
func NewMockInstanceDisksExporter(ctrl *gomock.Controller) *MockInstanceDisksExporter {
	mock := &MockInstanceDisksExporter{ctrl: ctrl}
	mock.recorder = &MockInstanceDisksExporterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockInstanceDisksExporter) EXPECT() *MockInstanceDisksExporterMockRecorder {
	return m.recorder
}

// Cancel mocks base method
func (m *MockInstanceDisksExporter) Cancel(arg0 string) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Cancel", arg0)
	ret0, _ := ret[0].(bool)
	return ret0
}

// Cancel indicates an expected call of Cancel
func (mr *MockInstanceDisksExporterMockRecorder) Cancel(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Cancel", reflect.TypeOf((*MockInstanceDisksExporter)(nil).Cancel), arg0)
}

// Export mocks base method
func (m *MockInstanceDisksExporter) Export(arg0 *v1.Instance, arg1 *domain.OVFExportParams) ([]*domain.ExportedDisk, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Export", arg0, arg1)
	ret0, _ := ret[0].([]*domain.ExportedDisk)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Export indicates an expected call of Export
func (mr *MockInstanceDisksExporterMockRecorder) Export(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Export", reflect.TypeOf((*MockInstanceDisksExporter)(nil).Export), arg0, arg1)
}

// TraceLogs mocks base method
func (m *MockInstanceDisksExporter) TraceLogs() []string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TraceLogs")
	ret0, _ := ret[0].([]string)
	return ret0
}

// TraceLogs indicates an expected call of TraceLogs
func (mr *MockInstanceDisksExporterMockRecorder) TraceLogs() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TraceLogs", reflect.TypeOf((*MockInstanceDisksExporter)(nil).TraceLogs))
}
