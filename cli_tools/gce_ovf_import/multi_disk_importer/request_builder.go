//  Copyright 2021 Google Inc. All Rights Reserved.
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

package multidiskimporter

import (
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/image/importer"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	daisyovfutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/daisy_utils"
	ovfdomain "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/domain"
)

type requestBuilder struct {
	workflowDir   string
	sourceFactory importer.SourceFactory
}

// buildRequests constructs a list of requests to import data disks based on the user's
// invocation parameters and the files referenced in the OVF descriptor.
func (r *requestBuilder) buildRequests(params *ovfdomain.OVFImportParams, fileURIs []string) (requests []importer.ImageImportRequest, err error) {
	for i, dataDiskURI := range fileURIs {
		var source importer.Source
		if source, err = r.sourceFactory.Init(dataDiskURI, ""); err != nil {
			return nil, err
		}
		dataDiskPrefix := getDisksPrefixName(params)
		diskName := daisyovfutils.GenerateDataDiskName(dataDiskPrefix, i)
		request := importer.ImageImportRequest{
			ExecutionID:           diskName,
			CloudLogsDisabled:     params.CloudLogsDisabled,
			ComputeEndpoint:       params.Ce,
			ComputeServiceAccount: params.ComputeServiceAccount,
			WorkflowDir:           r.workflowDir,
			DaisyLogLinePrefix:    fmt.Sprintf("disk-%d", i+1),
			GcsLogsDisabled:       params.GcsLogsDisabled,
			Network:               params.Network,
			NoExternalIP:          params.NoExternalIP,
			Oauth:                 params.Oauth,
			Project:               *params.Project,
			ScratchBucketGcsPath:  path.JoinURL(params.ScratchBucketGcsPath, diskName),
			Source:                source,
			StdoutLogsDisabled:    params.StdoutLogsDisabled,
			Subnet:                params.Subnet,
			Timeout:               params.Deadline.Sub(time.Now()),
			Tool:                  params.GetTool(),
			UefiCompatible:        params.UefiCompatible,
			Zone:                  params.Zone,
			OS:                    params.OsID,
			DataDisk:              true,
			DiskName:              diskName,
		}
		requests = append(requests, request)
	}
	return requests, nil
}

func getDisksPrefixName(params *ovfdomain.OVFImportParams) string {
	if params.IsInstanceImport() {
		return params.InstanceNames
	}
	return params.MachineImageName
}
