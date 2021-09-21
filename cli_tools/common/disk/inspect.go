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
	"path"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/distro"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisyutils"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/proto/go/pb"
)

const (
	workflowFile = "image_import/inspection/boot-inspect.wf.json"
)

// Inspector finds partition and boot-related properties for a disk.
//
//go:generate go run github.com/golang/mock/mockgen -package diskmocks -source $GOFILE -destination mocks/mock_inspect.go
type Inspector interface {
	// Inspect finds partition and boot-related properties for a disk and
	// returns an InspectionResult. The reference is implementation specific.
	Inspect(reference string) (*pb.InspectionResults, error)
	Cancel(reason string) bool
}

// NewInspector creates an Inspector that can inspect GCP disks.
// A GCE instance runs the inspection; network and subnet are used
// for its network interface.
func NewInspector(env daisyutils.EnvironmentSettings, logger logging.Logger) (Inspector, error) {
	wf, err := daisy.NewFromFile(path.Join(env.WorkflowDirectory, workflowFile))
	if err != nil {
		return nil, err
	}

	if env.DaisyLogLinePrefix != "" {
		env.DaisyLogLinePrefix += "-"
	}
	env.DaisyLogLinePrefix += "inspect"
	return &bootInspector{daisyutils.NewDaisyWorker(wf, env, logger), logger}, nil
}

// bootInspector implements disk.Inspector using the Python boot-inspect package,
// executed on a worker VM using Daisy.
type bootInspector struct {
	worker daisyutils.DaisyWorker
	logger logging.Logger
}

func (i *bootInspector) Cancel(reason string) bool {
	i.logger.Debug(fmt.Sprintf("Canceling inspection with reason: %q", reason))
	return i.worker.Cancel(reason)
}

// Inspect finds partition and boot-related properties for a GCP persistent disk, and
// returns an InspectionResult. `reference` is a fully-qualified PD URI, such as
// "projects/project-name/zones/us-central1-a/disks/disk-name".
func (i *bootInspector) Inspect(reference string) (*pb.InspectionResults, error) {
	startTime := time.Now()
	results := &pb.InspectionResults{}

	// Run the inspection worker.
	vars := map[string]string{
		"pd_uri": reference,
	}
	encodedProto, err := i.worker.RunAndReadSerialValue("inspect_pb", vars)
	if err != nil {
		return i.assembleErrors(reference, results, pb.InspectionResults_RUNNING_WORKER, err, startTime)
	}

	// Decode the base64-encoded proto.
	bytes, err := base64.StdEncoding.DecodeString(encodedProto)
	if err == nil {
		err = proto.Unmarshal(bytes, results)
	}
	if err != nil {
		return i.assembleErrors(reference, results, pb.InspectionResults_DECODING_WORKER_RESPONSE, err, startTime)
	}
	i.logger.Debug(fmt.Sprintf("Detection results: %s", results.String()))

	// Validate the results.
	if err = i.validate(results); err != nil {
		return i.assembleErrors(reference, results, pb.InspectionResults_INTERPRETING_INSPECTION_RESULTS, err, startTime)
	}

	if err = i.populate(results); err != nil {
		return i.assembleErrors(reference, results, pb.InspectionResults_INTERPRETING_INSPECTION_RESULTS, err, startTime)
	}

	results.ElapsedTimeMs = time.Now().Sub(startTime).Milliseconds()
	i.logger.Metric(&pb.OutputInfo{InspectionResults: results})
	return results, nil
}

// assembleErrors sets the errorWhen field, and generates an error object.
func (i *bootInspector) assembleErrors(reference string, results *pb.InspectionResults,
	errorWhen pb.InspectionResults_ErrorWhen, err error, startTime time.Time) (*pb.InspectionResults, error) {
	results.ErrorWhen = errorWhen
	if err != nil {
		err = fmt.Errorf("failed to inspect %v: %w", reference, err)
	} else {
		err = fmt.Errorf("failed to inspect %v", reference)
	}
	results.ElapsedTimeMs = time.Now().Sub(startTime).Milliseconds()
	return results, err
}

// validate checks the fields from a pb.InspectionResults object for consistency, returning
// an error if an issue is found.
func (i *bootInspector) validate(results *pb.InspectionResults) error {
	// Only populate OsRelease when one OS is found.
	if results.OsCount != 1 {
		if results.OsRelease != nil {
			return fmt.Errorf(
				"Worker should not return OsRelease when NumOsFound != 1: NumOsFound=%d", results.OsCount)
		}
		return nil
	}

	if results.OsRelease == nil {
		return errors.New("Worker should return OsRelease when OsCount == 1")
	}

	if results.OsRelease.CliFormatted != "" {
		return errors.New("Worker should not return CliFormatted")
	}

	if results.OsRelease.Distro != "" {
		return errors.New("Worker should not return Distro name, only DistroId")
	}

	if results.OsRelease.MajorVersion == "" {
		return errors.New("Missing MajorVersion")
	}

	if results.OsRelease.Architecture == pb.Architecture_ARCHITECTURE_UNKNOWN {
		return errors.New("Missing Architecture")
	}

	if results.OsRelease.DistroId == pb.Distro_DISTRO_UNKNOWN {
		return errors.New("Missing DistroId")
	}

	return nil
}

// populate fills the fields in the pb.InspectionResults that are not returned by the worker.
// This is required since the worker is unaware of import-specific idioms, such as the formatting
// used by gcloud's --os argument.
func (i *bootInspector) populate(results *pb.InspectionResults) error {
	if results.ErrorWhen == pb.InspectionResults_NO_ERROR && results.OsCount == 1 {
		distroEnum, major, minor := results.OsRelease.DistroId,
			results.OsRelease.MajorVersion, results.OsRelease.MinorVersion

		distroName := strings.ReplaceAll(strings.ToLower(results.OsRelease.GetDistroId().String()), "_", "-")

		results.OsRelease.Distro = distroName
		version, err := distro.FromComponents(distroName, major, minor,
			results.OsRelease.Architecture.String())
		if err != nil {
			i.logger.Trace(
				fmt.Sprintf("Failed to interpret version distro=%q, major=%q, minor=%q: %v",
					distroEnum, major, minor, err))
		} else {
			results.OsRelease.CliFormatted = version.AsGcloudArg()
		}
	}
	return nil
}
