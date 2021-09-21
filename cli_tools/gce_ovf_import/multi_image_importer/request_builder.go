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

package multiimageimporter

import (
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/image/importer"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	ovfdomain "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/domain"
)

type requestBuilder struct {
	workflowDir   string
	sourceFactory importer.SourceFactory
}

// buildRequests constructs a list of ImageImportRequests based on the user's
// invocation parameters and the files referenced in the OVF descriptor.
func (r *requestBuilder) buildRequests(params *ovfdomain.OVFImportParams, fileURIs []string) (requests []importer.ImageImportRequest, err error) {
	for i, fileURI := range fileURIs {
		var source importer.Source
		if source, err = r.sourceFactory.Init(fileURI, ""); err != nil {
			return nil, err
		}
		imageName := fmt.Sprintf("ovf-%s-%d", params.BuildID, i+1)
		request := importer.ImageImportRequest{
			ExecutionID:           imageName,
			CloudLogsDisabled:     params.CloudLogsDisabled,
			ComputeEndpoint:       params.Ce,
			ComputeServiceAccount: params.ComputeServiceAccount,
			WorkflowDir:           r.workflowDir,
			DaisyLogLinePrefix:    fmt.Sprintf("disk-%d", i+1),
			GcsLogsDisabled:       params.GcsLogsDisabled,
			ImageName:             imageName,
			Network:               params.Network,
			NoExternalIP:          params.NoExternalIP,
			NoGuestEnvironment:    params.NoGuestEnvironment,
			Oauth:                 params.Oauth,
			Project:               *params.Project,
			ScratchBucketGcsPath:  path.JoinURL(params.ScratchBucketGcsPath, imageName),
			Source:                source,
			StdoutLogsDisabled:    params.StdoutLogsDisabled,
			Subnet:                params.Subnet,
			Timeout:               params.Deadline.Sub(time.Now()),
			Tool:                  params.GetTool(),
			UefiCompatible:        params.UefiCompatible,
			Zone:                  params.Zone,
		}
		bootable := i == 0
		if bootable {
			request.OS = params.OsID
			request.BYOL = params.BYOL
			importer.FixBYOLAndOSArguments(&request.OS, &request.BYOL)
		} else {
			request.DataDisk = true
		}
		requests = append(requests, request)
	}
	return requests, nil
}
