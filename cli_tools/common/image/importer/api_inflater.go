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
	"sync"
	"time"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	daisyUtils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
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
	request          ImageImportRequest
	computeClient    daisyCompute.Client
	storageClient    domain.StorageClientInterface
	guestOsFeatures  []*compute.GuestOsFeature
	wg               sync.WaitGroup
	cancelChan       chan string
	logger           logging.Logger
	isShadowInflater bool
}

func createAPIInflater(request ImageImportRequest, computeClient daisyCompute.Client, storageClient domain.StorageClientInterface, logger logging.Logger, isShadowInflater bool) *apiInflater {
	inflater := apiInflater{
		request:          request,
		computeClient:    computeClient,
		storageClient:    storageClient,
		cancelChan:       make(chan string, 1),
		logger:           logger,
		isShadowInflater: isShadowInflater,
	}
	if request.UefiCompatible {
		inflater.guestOsFeatures = []*compute.GuestOsFeature{{Type: "UEFI_COMPATIBLE"}}
	}
	return &inflater
}

func (inflater *apiInflater) Inflate() (persistentDisk, shadowTestFields, error) {
	if inflater.isShadowInflater {
		return inflater.inflateForShadowTest()
	}

	ctx := context.Background()
	startTime := time.Now()
	diskName := "disk-" + inflater.request.ExecutionID

	cd, err := inflater.createDisk(diskName)
	if err != nil {
		return persistentDisk{}, shadowTestFields{}, daisy.Errf("Failed to create disk by api inflater: %v", err)
	}

	return inflater.getDiskAttributes(ctx, diskName, cd, startTime)
}

func (inflater *apiInflater) inflateForShadowTest() (persistentDisk, shadowTestFields, error) {
	inflater.wg.Add(1)
	defer inflater.wg.Done()

	ctx := context.Background()
	startTime := time.Now()
	diskName := inflater.getShadowDiskName()

	cd, err := inflater.createDisk(diskName)
	if err != nil {
		return persistentDisk{}, shadowTestFields{}, daisy.Errf("Failed to create shadow disk: %v", err)
	}

	// Cleanup the shadow disk ignoring error
	defer inflater.computeClient.DeleteDisk(inflater.request.Project, inflater.request.Zone, cd.Name)

	// If received a cancel signal from cancel(), then return early. Otherwise, it will waste
	// 2 min+ on calculateChecksum().
	select {
	case <-inflater.cancelChan:
		return persistentDisk{}, shadowTestFields{}, nil
	default:
	}

	pd, ii, err := inflater.getDiskAttributes(ctx, diskName, cd, startTime)
	if err != nil {
		return pd, ii, err
	}

	// Calculate checksum by daisy workflow
	inflater.logger.Debug("Started checksum calculation.")
	ii.checksum, err = inflater.calculateChecksum(ctx, pd.uri)
	if err != nil {
		err = daisy.Errf("Failed to calculate checksum: %v", err)
	}
	return pd, ii, err
}

func (inflater *apiInflater) createDisk(diskName string) (compute.Disk, error) {
	cd := compute.Disk{
		Name:                diskName,
		SourceStorageObject: inflater.request.Source.Path(),
		GuestOsFeatures:     inflater.guestOsFeatures,
	}
	err := inflater.computeClient.CreateDisk(inflater.request.Project, inflater.request.Zone, &cd)
	return cd, err
}

func (inflater *apiInflater) getDiskAttributes(ctx context.Context, diskName string, cd compute.Disk, startTime time.Time) (persistentDisk, shadowTestFields, error) {
	// Prepare return value
	bkt, objPath, err := storage.GetGCSObjectPathElements(inflater.request.Source.Path())
	if err != nil {
		return persistentDisk{}, shadowTestFields{}, err
	}
	sourceFile := inflater.storageClient.GetObject(bkt, objPath).GetObjectHandle()
	attrs, err := sourceFile.Attrs(ctx)
	if err != nil {
		return persistentDisk{}, shadowTestFields{}, daisy.Errf("Failed to get source file attributes: %v", err)
	}
	sourceFileSizeGb := (attrs.Size-1)/1073741824 + 1

	diskURI := fmt.Sprintf("zones/%s/disks/%s", inflater.request.Zone, diskName)
	pd := persistentDisk{
		uri:        diskURI,
		sizeGb:     cd.SizeGb,
		sourceGb:   sourceFileSizeGb,
		sourceType: "vmdk", // only vmdk is supported right now
	}
	ii := shadowTestFields{
		inflationType: "api",
		inflationTime: time.Since(startTime),
	}

	return pd, ii, nil
}

func (inflater *apiInflater) getShadowDiskName() string {
	return fmt.Sprintf("shadow-disk-%v", inflater.request.ExecutionID)
}

func (inflater *apiInflater) Cancel(reason string) bool {
	if !inflater.isShadowInflater {
		// We don't have to do any actual cancellation for the single CreateDisk API call.
		// Only the daisy workflow is worth cancelling.
		return false
	}

	// Send cancel signal
	inflater.cancelChan <- reason

	// Wait for inflate() to finish. Otherwise, the whole program might exit
	// before DeleteDisk() was executed.
	inflater.wg.Wait()

	// Expect 404 error to ensure shadow disk has been cleaned up.
	_, err := inflater.computeClient.GetDisk(inflater.request.Project, inflater.request.Zone, inflater.getShadowDiskName())
	if apiErr, ok := err.(*googleapi.Error); !ok || apiErr.Code != 404 {
		if err == nil {
			inflater.logger.Debug(fmt.Sprintf("apiInflater.inflate is canceled, cleanup is failed: %v", reason))
		} else {
			inflater.logger.Debug(fmt.Sprintf("apiInflater.inflate is canceled, cleanup failed to verify: %v", reason))
		}
		return false
	}
	inflater.logger.Debug(fmt.Sprintf("apiInflater.inflate is canceled: %v", reason))
	return true
}

// run a workflow to calculate checksum
func (inflater *apiInflater) calculateChecksum(ctx context.Context, diskURI string) (string, error) {
	w := inflater.getCalculateChecksumWorkflow(diskURI)
	err := daisyUtils.RunWorkflowWithCancelSignal(ctx, w)
	if err != nil {
		return "", err
	}
	return w.GetSerialConsoleOutputValue("disk-checksum"), nil
}

func (inflater *apiInflater) getCalculateChecksumWorkflow(diskURI string) *daisy.Workflow {
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
							ServiceAccounts: []*compute.ServiceAccount{
								{
									Email:  "${compute_service_account}",
									Scopes: []string{"https://www.googleapis.com/auth/devstorage.read_write"},
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
	env := inflater.request.EnvironmentSettings()
	env.Timeout = "20m"
	env.ApplyToWorkflow(w)
	computeServiceAccount := "default"
	if inflater.request.ComputeServiceAccount != "" {
		computeServiceAccount = inflater.request.ComputeServiceAccount
	}
	w.AddVar("compute_service_account", computeServiceAccount)
	return w
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
