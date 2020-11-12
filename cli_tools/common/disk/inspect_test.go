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

package disk

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
	"github.com/GoogleCloudPlatform/compute-image-tools/proto/go/pb"
)

func TestBootInspector_Inspect_PassesVarsWhenInvokingWorkflow(t *testing.T) {
	for caseNumber, tt := range []struct {
		inspectOS bool
		reference string
	}{
		{inspectOS: true, reference: "uri/for/pd"},
		{inspectOS: false, reference: "uri/for/pd"},
	} {
		caseName := fmt.Sprintf("%d inspectOS=%v, reference=%v", caseNumber, tt.inspectOS, tt.reference)
		t.Run(caseName, func(t *testing.T) {
			expected := &pb.InspectionResults{
				UefiBootable: true,
			}

			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			worker := mocks.NewMockDaisyWorker(mockCtrl)
			worker.EXPECT().RunAndReadSerialValue("inspect_pb", map[string]string{
				"pd_uri":        tt.reference,
				"is_inspect_os": strconv.FormatBool(tt.inspectOS),
			}).Return(encodeToBase64(expected), nil)
			inspector := bootInspector{worker: worker}

			actual, err := inspector.Inspect(tt.reference, tt.inspectOS)
			assert.NoError(t, err)
			assert.Equal(t, InspectionResult{UEFIBootable: true}, actual)
		})
	}
}

func TestBootInspector_Inspect_WorkerAndTransitErrors(t *testing.T) {
	for _, tt := range []struct {
		caseName             string
		base64FromInspection string
		errorFromInspection  error
		expectResults        InspectionResult
		expectErrorToContain string
	}{
		{
			caseName:             "worker fails to run",
			errorFromInspection:  errors.New("failure-from-daisy"),
			expectResults:        InspectionResult{},
			expectErrorToContain: "failure-from-daisy",
		}, {
			caseName:             "worker returns invalid base64",
			base64FromInspection: "garbage",
			expectResults:        InspectionResult{},
			expectErrorToContain: "base64",
		}, {
			caseName:             "worker returns invalid proto bytes",
			base64FromInspection: base64.StdEncoding.EncodeToString([]byte("garbage")),
			expectResults:        InspectionResult{},
			expectErrorToContain: "cannot parse",
		},
	} {
		t.Run(tt.caseName, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			worker := mocks.NewMockDaisyWorker(mockCtrl)
			worker.EXPECT().RunAndReadSerialValue("inspect_pb", map[string]string{
				"pd_uri":        "reference",
				"is_inspect_os": "true",
			}).Return(tt.base64FromInspection, tt.errorFromInspection)
			inspector := bootInspector{worker: worker}
			actual, err := inspector.Inspect("reference", true)
			if err == nil {
				t.Fatal("err must be non-nil")
			}
			assert.Contains(t, err.Error(), tt.expectErrorToContain)
			assert.Equal(t, tt.expectResults, actual)
		})
	}
}

func TestBootInspector_Inspect_InvalidWorkerResponses(t *testing.T) {
	for _, tt := range []struct {
		caseName               string
		responseFromInspection *pb.InspectionResults
		expectResults          InspectionResult
		expectErrorToContain   string
	}{
		{
			caseName: "Fail when OsCount is zero and OsRelease non-nil",
			responseFromInspection: &pb.InspectionResults{
				OsCount:   0,
				OsRelease: &pb.OsRelease{},
			},
			expectResults:        InspectionResult{},
			expectErrorToContain: "Worker should not return OsRelease when NumOsFound != 1",
		},
		{
			caseName: "Fail when OsCount > 1 and OsRelease non-nil",
			responseFromInspection: &pb.InspectionResults{
				OsCount:   2,
				OsRelease: &pb.OsRelease{},
			},
			expectResults:        InspectionResult{},
			expectErrorToContain: "Worker should not return OsRelease when NumOsFound != 1",
		},
		{
			caseName: "Fail when CliFormatted is populated",
			responseFromInspection: &pb.InspectionResults{
				OsCount: 1,
				OsRelease: &pb.OsRelease{
					Architecture: pb.Architecture_X64,
					MajorVersion: "18",
					MinorVersion: "04",
					DistroId:     pb.Distro_UBUNTU,
					CliFormatted: "ubuntu-1804",
				},
			},
			expectResults: InspectionResult{
				Architecture: "x64",
				Major:        "18",
				Minor:        "04",
			},
			expectErrorToContain: "Worker should not return CliFormatted",
		}, {
			caseName: "Fail when Distro name is populated",
			responseFromInspection: &pb.InspectionResults{
				OsCount: 1,
				OsRelease: &pb.OsRelease{
					Architecture: pb.Architecture_X64,
					MajorVersion: "10",
					DistroId:     pb.Distro_UBUNTU,
					Distro:       "ubuntu",
				},
			},
			expectResults: InspectionResult{
				Architecture: "x64",
				Distro:       "ubuntu",
				Major:        "10",
			},
			expectErrorToContain: "Worker should not return Distro name",
		}, {
			caseName: "Fail when missing MajorVersion",
			responseFromInspection: &pb.InspectionResults{
				OsCount: 1,
				OsRelease: &pb.OsRelease{
					Architecture: pb.Architecture_X64,
					DistroId:     pb.Distro_UBUNTU,
				},
			},
			expectResults: InspectionResult{
				Architecture: "x64",
			},
			expectErrorToContain: "Missing MajorVersion",
		}, {
			caseName: "Fail when missing Architecture",
			responseFromInspection: &pb.InspectionResults{
				OsCount: 1,
				OsRelease: &pb.OsRelease{
					DistroId:     pb.Distro_UBUNTU,
					MajorVersion: "10",
				},
			},
			expectResults: InspectionResult{
				Major: "10",
			},
			expectErrorToContain: "Missing Architecture",
		}, {
			caseName: "Fail when missing DistroId",
			responseFromInspection: &pb.InspectionResults{
				OsCount: 1,
				OsRelease: &pb.OsRelease{
					Architecture: pb.Architecture_X64,
					MajorVersion: "10",
				},
			},
			expectResults: InspectionResult{
				Architecture: "x64",
				Major:        "10",
			},
			expectErrorToContain: "Missing DistroId",
		},
	} {
		t.Run(tt.caseName, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			worker := mocks.NewMockDaisyWorker(mockCtrl)
			worker.EXPECT().RunAndReadSerialValue("inspect_pb", map[string]string{
				"pd_uri":        "reference",
				"is_inspect_os": "true",
			}).Return(encodeToBase64(tt.responseFromInspection), nil)
			worker.EXPECT().TraceLogs().Return(nil)
			inspector := bootInspector{worker: worker}
			results, err := inspector.Inspect("reference", true)
			if err == nil {
				t.Fatal("err must be non-nil")
			}
			assert.Contains(t, err.Error(), tt.expectErrorToContain)
			assertLogsContainResults(t, inspector, tt.responseFromInspection)
			assert.Equal(t, tt.expectResults, results)
		})
	}
}

func TestBootInspector_IncludesRemoteAndWorkerLogs(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	worker := mocks.NewMockDaisyWorker(mockCtrl)
	worker.EXPECT().TraceLogs().Return([]string{"serial console1", "serial console2"})

	inspector := bootInspector{worker: worker}
	inspector.tracef("log %s %v", "A", false)
	inspector.tracef("log %s", "B")

	actual := inspector.TraceLogs()
	assert.Contains(t, actual, "serial console1")
	assert.Contains(t, actual, "serial console2")
	assert.Contains(t, actual, "log A false")
	assert.Contains(t, actual, "log B")
}

func TestBootInspector_ForwardsCancelToWorkflow(t *testing.T) {
	for _, tt := range []struct {
		name      string
		reason    string
		cancelled bool
	}{
		{"cancel success", "reason 1", true},
		{"cancel failed", "reason 2", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			worker := mocks.NewMockDaisyWorker(mockCtrl)
			worker.EXPECT().Cancel(tt.reason).Return(tt.cancelled)
			inspector := bootInspector{worker: worker}
			assert.Equal(t, tt.cancelled, inspector.Cancel(tt.reason))
		})
	}
}

func encodeToBase64(results *pb.InspectionResults) string {
	if results == nil {
		return ""
	}
	bytes, err := proto.Marshal(results)
	if err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(bytes)
}

func assertLogsContainResults(t *testing.T, inspector bootInspector, results *pb.InspectionResults) {
	var traceIncludesResults bool
	logs := inspector.TraceLogs()
	resultString := results.String()
	for _, log := range logs {
		if strings.Contains(log, resultString) {
			traceIncludesResults = true
			break
		}
	}
	if !traceIncludesResults {
		t.Errorf("Trace logs didn't include results.\n Logs:%#v\n Results: %v", logs, resultString)
	}
}
