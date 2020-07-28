//  Copyright 2020  Licensed under the Apache License, Version 2.0 (the "License");
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

package importer

import (
	"context"
	"fmt"
	"strings"
	"time"

	daisyUtils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/daisycommon"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	computeAlpha "google.golang.org/api/compute/v0.alpha"
	"google.golang.org/api/compute/v1"
)

func isCausedByUnsupportedFormat(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "INVALID_IMAGE_FILE")
}

func isCausedByAlphaAPIAccess(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "Required 'Alpha Access' permission")
}

// apiInflater implements `importer.inflater` using the Compute Engine API
type apiInflater struct {
	serialLogs    []string
	args          ImportArguments
	computeClient daisyCompute.Client
	storageClient storage.Client
}

func createAPIInflater(args ImportArguments, computeClient daisyCompute.Client, storageClient storage.Client) inflater {
	return &apiInflater{
		serialLogs:    []string{},
		args:          args,
		computeClient: computeClient,
		storageClient: storageClient,
	}
}

func (inflater *apiInflater) traceLogs() []string {
	return inflater.serialLogs
}

func (inflater *apiInflater) addTraceLog(l string) {
	inflater.serialLogs = append(inflater.serialLogs, l)
}

func (inflater *apiInflater) inflate() (persistentDisk, error) {
	ctx := context.Background()
	startTime := time.Now()
	diskName := fmt.Sprintf("shadow-disk-%v", inflater.args.ExecutionID)
	cd := computeAlpha.Disk{
		Name:                diskName,
		SourceStorageObject: inflater.args.SourceFile,
	}
	err := inflater.computeClient.CreateDiskAlpha(inflater.args.Project, inflater.args.Zone, &cd)
	if err != nil {
		return persistentDisk{}, err
	}

	// Cleanup the shadow disk ignoring error
	defer inflater.computeClient.DeleteDisk(inflater.args.Project, inflater.args.Zone, cd.Name)

	bkt, objPath, err := storage.GetGCSObjectPathElements(inflater.args.SourceFile)
	if err != nil {
		return persistentDisk{}, err
	}
	sourceFile := inflater.storageClient.GetObject(bkt, objPath).GetObjectHandle()
	attrs, err := sourceFile.Attrs(ctx)
	if err != nil {
		return persistentDisk{}, daisy.Errf("Failed to get source file attributes: %v", err)
	}
	sourceFileSizeGb := (attrs.Size-1)/1073741824 + 1

	diskURI := fmt.Sprintf("zones/%s/disks/%s", inflater.args.Zone, diskName)
	pd := persistentDisk{
		uri:           diskURI,
		sizeGb:        cd.SizeGb,
		sourceGb:      sourceFileSizeGb,
		sourceType:    "vmdk", // only vmdk is supported right now
		inflationTime: time.Since(startTime),
	}

	inflater.addTraceLog("Started checksum calculation.")
	pd.checksum, err = inflater.calculateChecksum(ctx, diskURI)
	return pd, err
}

// Can't cancel the single disk.insert API call
func (inflater *apiInflater) cancel(reason string) bool {
	return false
}

// run a workflow to calculate checksum
func (inflater *apiInflater) calculateChecksum(ctx context.Context, diskURI string) (string, error) {
	w := daisy.New()
	w.Name = "shadow-disk-checksum"
	checksumScript := checksumScriptConst
	w.Steps = map[string]*daisy.Step{
		"create-disks": {
			CreateDisks: &daisy.CreateDisks{
				{
					Disk: compute.Disk{
						Name:        "disk-${NAME}",
						SourceImage: "projects/compute-image-tools/global/images/family/debian-9-worker",
						Type:        "pd-ssd",
					},
					FallbackToPdStandard: true,
				},
			},
		},
		"create-instance": {
			CreateInstances: &daisy.CreateInstances{
				Instances: []*daisy.Instance{
					{
						Instance: compute.Instance{
							Name: "inst-${NAME}",
							Disks: []*compute.AttachedDisk{
								{Source: "disk-${NAME}"},
								{Source: diskURI, Mode: "READ_ONLY"},
							},
							MachineType: "n1-highcpu-4",
							Metadata: &compute.Metadata{
								Items: []*compute.MetadataItems{
									{
										Key:   "startup-script",
										Value: &checksumScript,
									},
								},
							},
							NetworkInterfaces: []*compute.NetworkInterface{
								{
									AccessConfigs: []*compute.AccessConfig{},
								},
							},
						},
					},
				},
			},
		},
		"wait-for-checksum": {
			WaitForInstancesSignal: &daisy.WaitForInstancesSignal{
				{
					Name: "inst-${NAME}",
					SerialOutput: &daisy.SerialOutput{
						Port:         1,
						SuccessMatch: "Checksum calculated.",
						StatusMatch:  "Checksum:",
					},
				},
			},
		},
	}
	w.Dependencies = map[string][]string{
		"create-instance":   {"create-disks"},
		"wait-for-checksum": {"create-instance"},
	}

	// Calculate checksum within 20min.
	daisycommon.SetWorkflowAttributes(w, daisycommon.WorkflowAttributes{
		Project:           inflater.args.Project,
		Zone:              inflater.args.Zone,
		GCSPath:           inflater.args.ScratchBucketGcsPath,
		OAuth:             inflater.args.Oauth,
		Timeout:           "20m",
		ComputeEndpoint:   inflater.args.ComputeEndpoint,
		DisableGCSLogs:    inflater.args.GcsLogsDisabled,
		DisableCloudLogs:  inflater.args.CloudLogsDisabled,
		DisableStdoutLogs: inflater.args.StdoutLogsDisabled,
	})
	err := daisyUtils.RunWorkflowWithCancelSignal(ctx, w)
	if err != nil {
		return "", err
	}
	return w.GetSerialConsoleOutputValue("disk-checksum"), nil
}

// Dup logic in import_image.sh. If change anything here, please change in both places.
const (
	checksumScriptConst = `
		function serialOutputKeyValuePair() {
			echo "<serial-output key:'$1' value:'$2'>"
		}
		CHECK_DEVICE=sdb
		BLOCK_COUNT=$(cat /sys/class/block/$CHECK_DEVICE/size)
	
		# Check size = 200000*512 = 100MB
		CHECK_COUNT=200000
		CHECKSUM1=$(sudo dd if=/dev/$CHECK_DEVICE ibs=512 skip=0 count=$CHECK_COUNT | md5sum)
		CHECKSUM2=$(sudo dd if=/dev/$CHECK_DEVICE ibs=512 skip=$(( 2000000 - $CHECK_COUNT )) count=$CHECK_COUNT | md5sum)
		CHECKSUM3=$(sudo dd if=/dev/$CHECK_DEVICE ibs=512 skip=$(( 20000000 - $CHECK_COUNT )) count=$CHECK_COUNT | md5sum)
		CHECKSUM4=$(sudo dd if=/dev/$CHECK_DEVICE ibs=512 skip=$(( $BLOCK_COUNT - $CHECK_COUNT )) count=$CHECK_COUNT | md5sum)
		echo "Checksum: $(serialOutputKeyValuePair "disk-checksum" "$CHECKSUM1-$CHECKSUM2-$CHECKSUM3-$CHECKSUM4")"
		echo "Checksum calculated."`
)
