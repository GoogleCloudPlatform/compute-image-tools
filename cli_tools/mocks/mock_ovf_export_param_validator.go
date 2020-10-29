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
	reflect "reflect"
)

// MockOvfExportParamValidator is a mock of OvfExportParamValidator interface
type MockOvfExportParamValidator struct {
	ctrl     *gomock.Controller
	recorder *MockOvfExportParamValidatorMockRecorder
}

// MockOvfExportParamValidatorMockRecorder is the mock recorder for MockOvfExportParamValidator
type MockOvfExportParamValidatorMockRecorder struct {
	mock *MockOvfExportParamValidator
}

// NewMockOvfExportParamValidator creates a new mock instance
func NewMockOvfExportParamValidator(ctrl *gomock.Controller) *MockOvfExportParamValidator {
	mock := &MockOvfExportParamValidator{ctrl: ctrl}
	mock.recorder = &MockOvfExportParamValidatorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockOvfExportParamValidator) EXPECT() *MockOvfExportParamValidatorMockRecorder {
	return m.recorder
}

// ValidateAndParseParams mocks base method
func (m *MockOvfExportParamValidator) ValidateAndParseParams(arg0 *domain.OVFExportParams) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ValidateAndParseParams", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// ValidateAndParseParams indicates an expected call of ValidateAndParseParams
func (mr *MockOvfExportParamValidatorMockRecorder) ValidateAndParseParams(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ValidateAndParseParams", reflect.TypeOf((*MockOvfExportParamValidator)(nil).ValidateAndParseParams), arg0)
}
