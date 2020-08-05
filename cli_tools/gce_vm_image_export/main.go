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

// GCE VM image export tool
package main

import (
	"flag"
	"os"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging/service"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_vm_image_export/exporter"
)

var (
	clientID             = flag.String(exporter.ClientIDFlagKey, "", "Identifies the client of the exporter, e.g. `gcloud` or `pantheon`.")
	clientVersion        = flag.String("client_version", "", "Identifies the version of the client of the exporter")
	destinationURI       = flag.String(exporter.DestinationURIFlagKey, "", "The Google Cloud Storage URI destination for the exported virtual disk file. For example: gs://my-bucket/my-exported-image.vmdk.")
	sourceImage          = flag.String(exporter.SourceImageFlagKey, "", "Compute Engine image from which to export")
	format               = flag.String("format", "", "Specify the format to export to, such as vmdk, vhdx, vpc, or qcow2.")
	project              = flag.String("project", "", "Project to run in, overrides what is set in workflow.")
	network              = flag.String("network", "", "Name of the network in your project to use for the image export. The network must have access to Google Cloud Storage. If not specified, the network named default is used.")
	subnet               = flag.String("subnet", "", "Name of the subnetwork in your project to use for the image export. If	the network resource is in legacy mode, do not provide this property. If the network is in auto subnet mode, providing the subnetwork is optional. If the network is in custom subnet mode, then this field should be specified. Zone should be specified if this field is specified.")
	zone                 = flag.String("zone", "", "Zone of the image to export. The zone in which to do the work of exporting the image. Overrides the default compute/zone property value for this command invocation.")
	timeout              = flag.String("timeout", "", "Maximum time a build can last before it is failed as TIMEOUT. For example, specifying 2h will fail the process after 2 hours. See $ gcloud topic datetimes for information on duration formats.")
	scratchBucketGcsPath = flag.String("scratch_bucket_gcs_path", "", "GCS scratch bucket to use, overrides what is set in workflow.")
	oauth                = flag.String("oauth", "", "path to oauth json file, overrides what is set in workflow.")
	ce                   = flag.String("compute_endpoint_override", "", "API endpoint to override default.")
	gcsLogsDisabled      = flag.Bool("disable_gcs_logging", false, "do not stream logs to GCS.")
	cloudLogsDisabled    = flag.Bool("disable_cloud_logging", false, "do not stream logs to Cloud Logging.")
	stdoutLogsDisabled   = flag.Bool("disable_stdout_logging", false, "do not display individual workflow logs on stdout.")
	labels               = flag.String("labels", "", "List of label KEY=VALUE pairs to add. Keys must start with a lowercase character and contain only hyphens (-), underscores (_), lowercase characters, and numbers. Values must contain only hyphens (-), underscores (_), lowercase characters, and numbers.")
)

func exportEntry() (service.Loggable, error) {
	currentExecutablePath := string(os.Args[0])
	wf, err := exporter.Run(*clientID, *destinationURI, *sourceImage, *format, project,
		*network, *subnet, *zone, *timeout, *scratchBucketGcsPath, *oauth, *ce, *gcsLogsDisabled,
		*cloudLogsDisabled, *stdoutLogsDisabled, *labels, currentExecutablePath)
	return service.NewLoggableFromWorkflow(wf), err
}

func main() {
	flag.Parse()

	paramLog := service.InputParams{
		ImageExportParams: &service.ImageExportParams{
			CommonParams: &service.CommonParams{
				ClientID:                *clientID,
				ClientVersion:           *clientVersion,
				Network:                 *network,
				Subnet:                  *subnet,
				Zone:                    *zone,
				Timeout:                 *timeout,
				Project:                 *project,
				ObfuscatedProject:       service.Hash(*project),
				Labels:                  *labels,
				ScratchBucketGcsPath:    *scratchBucketGcsPath,
				Oauth:                   *oauth,
				ComputeEndpointOverride: *ce,
				DisableGcsLogging:       *gcsLogsDisabled,
				DisableCloudLogging:     *cloudLogsDisabled,
				DisableStdoutLogging:    *stdoutLogsDisabled,
			},
			DestinationURI: *destinationURI,
			SourceImage:    *sourceImage,
			Format:         *format,
		},
	}

	if err := service.RunWithServerLogging(service.ImageExportAction, paramLog, project, exportEntry); err != nil {
		os.Exit(1)
	}
}
