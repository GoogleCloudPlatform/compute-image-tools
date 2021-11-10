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

package import_test

import (
	"strings"
	"sync"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_vm_image_import/cli"
	"github.com/GoogleCloudPlatform/compute-image-tools/proto/go/pb"
	"github.com/GoogleCloudPlatform/compute-image-tools/proto/go/pbtesting"
)

func Test_IncludeInflationWorkerLogs_WhenInflationFails(t *testing.T) {
	t.Parallel()
	logger := newBufferedToolLogger()
	imageName := "i" + uuid.New().String()
	err := cli.Main([]string{
		"-image_name", imageName,
		"-client_id", "test",
		"-source_file", "gs://compute-image-tools-test-resources/vmdk-with-missing-footer.vmdk",
		"-project", project,
		"-zone", zone,
	}, logger, "../../../daisy_workflows")

	if err == nil {
		t.Fatal("Expected import to fail.")
	}
	assert.Contains(t, err.Error(), "ImportFailed: The image file is not decodable")
	// Look for a log line from daisy_workflows/image_import/import_image.sh
	assertTraceLogsContain(t, logger, "IMAGE_PATH: /daisy-scratch/vmdk-with-missing-footer.vmdk")
}

func Test_IncludeTranslationLogs_WhenTranslationFails(t *testing.T) {
	t.Parallel()
	logger := newBufferedToolLogger()
	imageName := "i" + uuid.New().String()
	err := cli.Main([]string{
		"-image_name", imageName,
		"-client_id", "test",
		"-source_image", "projects/compute-image-tools-test/global/images/debian-9-translate",
		"-os=sles-15",
		"-project", project,
		"-zone", zone,
	}, logger, "../../../daisy_workflows")
	if err == nil {
		t.Fatal("Expected import to fail.")
	}
	assert.Contains(t, err.Error(), "\"debian-9\" was detected on your disk, but \"sles-15\" was specified. Please verify and re-import")
	// Look for a log line from daisy_workflows/image_import/import_image.sh
	assertTraceLogsContain(t, logger, "/files/run-translate.sh")
	actualResults := logger.ReadOutputInfo().InspectionResults
	actualResults.ElapsedTimeMs = 0
	pbtesting.AssertEqual(t, &pb.InspectionResults{
		OsCount: 1,
		OsRelease: &pb.OsRelease{
			CliFormatted: "debian-9",
			Distro:       "debian",
			MajorVersion: "9",
			MinorVersion: "12",
			Architecture: pb.Architecture_X64,
			DistroId:     pb.Distro_DEBIAN,
		},
	}, actualResults)
}

func assertTraceLogsContain(t *testing.T, logger *bufferedLogger, substring string) {
	if len(logger.trace) < 1 {
		t.Errorf("No trace logs recorded. Expected to find %q", substring)
		return
	}
	var found bool
	for _, msg := range logger.trace {
		if strings.Contains(msg, substring) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Did not find %q in trace logs.", substring)
	}
}

func newBufferedToolLogger() *bufferedLogger {
	return &bufferedLogger{outputInfo: &pb.OutputInfo{}}
}

type bufferedLogger struct {
	user, debug, trace []string
	outputInfo         *pb.OutputInfo
	mu                 sync.Mutex
}

func (b *bufferedLogger) ReadOutputInfo() *pb.OutputInfo {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.outputInfo
}

func (b *bufferedLogger) User(message string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.user = append(b.user, message)
}

func (b *bufferedLogger) Debug(message string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.debug = append(b.debug, message)
}

func (b *bufferedLogger) Trace(message string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.trace = append(b.trace, message)
}

func (b *bufferedLogger) Metric(metric *pb.OutputInfo) {
	b.mu.Lock()
	defer b.mu.Unlock()
	proto.Merge(b.outputInfo, metric)
}

func (b *bufferedLogger) NewLogger(userPrefix string) logging.Logger {
	panic("not expected for this test")
}
