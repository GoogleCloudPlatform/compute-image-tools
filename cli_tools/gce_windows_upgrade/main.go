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

// gce_windows_upgrade is a tool for upgrading GCE Windows instances.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging/service"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_windows_upgrade/upgrader"
)

const logPrefix = "[windows-upgrade]"

var (
	clientID               = flag.String(upgrader.ClientIDFlagKey, "", "Identifies the upgrade client. Set to `gcloud`, `api` or `pantheon`.")
	clientVersion          = flag.String("client-version", "", "Identifies the version of the upgrade client.")
	project                = flag.String("project", "", "Project containing the instance to upgrade.")
	zone                   = flag.String("zone", "", "Project containing the instance to upgrade.")
	instance               = flag.String("instance", "Instance to upgrade. Can be either the instance name or the full path to the instance in the following format: 'projects/<project>/zones/<zone>/instances/'. If the full path is specified, flags -project and -zone will be ignored.", "")
	createMachineBackup    = flag.Bool("create-machine-backup", true, "When enabled, a machine image is created that backs up the original state of your instance.")
	autoRollback           = flag.Bool("auto-rollback", false, "When auto rollback is enabled, the instance and its resources are restored to their original state. Otherwise, the instance and any temporary resources are left in the intermediate state of the time of failure. This is useful for debugging.")
	sourceOS               = flag.String("source-os", "", fmt.Sprintf("OS version of the source instance to upgrade. Supported values: %v", strings.Join(upgrader.SupportedSourceOSVersions(), ", ")))
	targetOS               = flag.String("target-os", "", fmt.Sprintf("Version of the OS after upgrade. Supported values: %v", strings.Join(upgrader.SupportedTargetOSVersions(), ", ")))
	timeout                = flag.String("timeout", "", "Maximum time limit for an upgrade. For example, if the time limit is set to 2h, the upgrade times out after two hours. For more information about time duration formats, see $ gcloud topic datetimes")
	useStagingInstallMedia = flag.Bool("use-staging-install-media", false, "Use staging install media. This flag is for testing only. Set to true to upgrade with staging windows install media.")
	scratchBucketGcsPath   = flag.String("scratch-bucket-gcs-path", "", "Location to store logs and intermediate artifacts. If omitted, a bucket will be created.")
	oauth                  = flag.String("oauth", "", "Path to OAuth .json file. This setting overrides the workflow setting.")
	ce                     = flag.String("compute-endpoint-override", "", "API endpoint. This setting overrides the default API endpoint setting.")
	gcsLogsDisabled        = flag.Bool("disable-gcs-logging", false, "Set to true to prevent logs from being saved to GCS.")
	cloudLogsDisabled      = flag.Bool("disable-cloud-logging", false, "Set to true to prevent logs from being saved to Cloud Logging.")
	stdoutLogsDisabled     = flag.Bool("disable-stdout-logging", false, "Set to true to disable detailed stdout information.")
)

func upgradeEntry() (service.Loggable, error) {
	logger := logging.NewToolLogger(logPrefix)
	logging.RedirectGlobalLogsToUser(logger)
	p := &upgrader.InputParams{
		ClientID:               strings.TrimSpace(*clientID),
		Instance:               strings.TrimSpace(*instance),
		CreateMachineBackup:    *createMachineBackup,
		AutoRollback:           *autoRollback,
		SourceOS:               strings.TrimSpace(*sourceOS),
		TargetOS:               strings.TrimSpace(*targetOS),
		ProjectPtr:             project,
		Zone:                   *zone,
		Timeout:                strings.TrimSpace(*timeout),
		UseStagingInstallMedia: *useStagingInstallMedia,
		ScratchBucketGcsPath:   strings.TrimSpace(*scratchBucketGcsPath),
		Oauth:                  strings.TrimSpace(*oauth),
		Ce:                     strings.TrimSpace(*ce),
		GcsLogsDisabled:        *gcsLogsDisabled,
		CloudLogsDisabled:      *cloudLogsDisabled,
		StdoutLogsDisabled:     *stdoutLogsDisabled,
	}
	err := upgrader.Run(p, logger)
	return service.NewOutputInfoLoggable(logger.ReadOutputInfo()), err
}

func main() {
	flag.Parse()

	paramLog := service.InputParams{
		WindowsUpgradeParams: &service.WindowsUpgradeParams{
			CommonParams: &service.CommonParams{
				ClientID:                *clientID,
				ClientVersion:           *clientVersion,
				Timeout:                 *timeout,
				Zone:                    *zone,
				ScratchBucketGcsPath:    *scratchBucketGcsPath,
				Oauth:                   *oauth,
				ComputeEndpointOverride: *ce,
				DisableGcsLogging:       *gcsLogsDisabled,
				DisableCloudLogging:     *cloudLogsDisabled,
				DisableStdoutLogging:    *stdoutLogsDisabled,
			},
			SourceOS:               *sourceOS,
			TargetOS:               *targetOS,
			Instance:               *instance,
			CreateMachineBackup:    *createMachineBackup,
			AutoRollback:           *autoRollback,
			UseStagingInstallMedia: *useStagingInstallMedia,
		},
	}

	if err := service.RunWithServerLogging(service.WindowsUpgrade, paramLog, project, upgradeEntry); err != nil {
		os.Exit(1)
	}
}
