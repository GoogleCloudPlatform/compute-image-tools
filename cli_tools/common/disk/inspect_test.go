package disk

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/daisycommon"
	"github.com/GoogleCloudPlatform/compute-image-tools/proto/go/pb"
)

func Test_NewInspector_SetsWorkflowVars(t *testing.T) {
	inspector, err := NewInspector(daisycommon.WorkflowAttributes{
		Project:           "project-id",
		Zone:              "zone-id",
		WorkflowDirectory: "../../../daisy_workflows",
	}, "network-id", "subnet-id")

	assert.NoError(t, err)
	realWorker := inspector.(*bootInspector).worker.(*defaultDaisyWorker)
	assert.Equal(t, realWorker.wf.Project, "project-id")
	assert.Equal(t, realWorker.wf.Zone, "zone-id")
	assert.Equal(t, realWorker.wf.Vars["network"].Value, "network-id")
	assert.Equal(t, realWorker.wf.Vars["subnet"].Value, "subnet-id")
}

func Test_bootInspector_PassesVarsWhenInvokingWorkflow(t *testing.T) {
	for caseNumber, tt := range []struct {
		inspectOS bool
		pdURI     string
	}{
		{inspectOS: true, pdURI: "uri/for/pd"},
		{inspectOS: false, pdURI: "uri/for/pd"},
	} {
		caseName := fmt.Sprintf("%d inspectOS=%v, pdURI=%v", caseNumber, tt.inspectOS, tt.pdURI)
		t.Run(caseName, func(t *testing.T) {
			expected := &pb.InspectionResults{
				UefiBootable: true,
			}
			inspector := bootInspector{
				worker: &mockDaisyWorker{
					runExpectedKey: "inspect_pb",
					runExpectedVars: map[string]string{
						"pd_uri":        tt.pdURI,
						"is_inspect_os": strconv.FormatBool(tt.inspectOS),
					},
					runReturnString: encode(expected),
					t:               t,
				},
			}

			actual, err := inspector.Inspect(tt.pdURI, tt.inspectOS)
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
			inspector := bootInspector{
				worker: &mockDaisyWorker{
					runReturnError:  tt.errorFromInspection,
					runReturnString: tt.base64FromInspection,
				},
			}
			actual, err := inspector.Inspect("pdURI", true)
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
			inspector := bootInspector{
				worker: &mockDaisyWorker{
					runReturnString: encode(tt.responseFromInspection),
				},
			}
			results, err := inspector.Inspect("pdURI", true)
			if err == nil {
				t.Fatal("err must be non-nil")
			}
			assert.Contains(t, err.Error(), tt.expectErrorToContain)
			assertLogsContainResults(t, inspector, tt.responseFromInspection)
			assert.Equal(t, tt.expectResults, results)
		})
	}
}

func encode(results *pb.InspectionResults) string {
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

func Test_bootInspector_IncludesRemoteAndWorkerLogs(t *testing.T) {
	workerLogs := []string{"serial console1", "serial console2"}
	inspector := bootInspector{
		worker: &mockDaisyWorker{traceLogsReturn: workerLogs},
	}

	inspector.tracef("log %s %v", "A", false)
	inspector.tracef("log %s", "B")

	assert.Contains(t, inspector.TraceLogs(), "serial console1")
	assert.Contains(t, inspector.TraceLogs(), "serial console2")
	assert.Contains(t, inspector.TraceLogs(), "log A false")
	assert.Contains(t, inspector.TraceLogs(), "log B")
}

func Test_bootInspector_ForwardsCancelToDaisyWorker(t *testing.T) {
	mockWorker := &mockDaisyWorker{
		cancelExpectedReason: "reason",
		t:                    t,
	}
	inspector := bootInspector{
		worker: mockWorker,
	}
	inspector.Cancel("reason")

}

func Test_bootInspector_ForwardsCancelToWorkflow(t *testing.T) {
	for _, tt := range []struct {
		name      string
		reason    string
		cancelled bool
	}{
		{"cancel success", "reason 1", true},
		{"cancel failed", "reason 2", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mockWorker := &mockDaisyWorker{
				cancelExpectedReason: tt.reason,
				cancelReturn:         tt.cancelled,
				t:                    t,
			}
			inspector := bootInspector{
				worker: mockWorker,
			}
			assert.Equal(t, tt.cancelled, inspector.Cancel(tt.reason))
		})
	}
}

type mockDaisyWorker struct {
	runExpectedKey       string
	runExpectedVars      map[string]string
	runReturnString      string
	runReturnError       error
	traceLogsReturn      []string
	cancelExpectedReason string
	cancelReturn         bool
	t                    *testing.T
}

func (m *mockDaisyWorker) runAndReadSerialValue(key string, vars map[string]string) (string, error) {
	if m.runExpectedKey != "" {
		assert.Equal(m.t, m.runExpectedKey, key)
	}
	if m.runExpectedVars != nil {
		assert.Equal(m.t, m.runExpectedVars, vars)
	}
	return m.runReturnString, m.runReturnError
}

func (m *mockDaisyWorker) cancel(reason string) bool {
	assert.Equal(m.t, m.cancelExpectedReason, reason)
	return m.cancelReturn
}

func (m *mockDaisyWorker) traceLogs() []string {
	return m.traceLogsReturn
}
