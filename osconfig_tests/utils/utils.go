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

// Package utils contains helper utils for osconfig_tests.
package utils

import (
	"fmt"
	"math/rand"
	"time"

	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/compute"
	api "google.golang.org/api/compute/v1"
	"google.golang.org/grpc/status"
)

var (
	// TODO: the startup script installs the osconfig agent and then keeps querying
	// for list of installed packages. Though, this is required for package
	// management tests, it is not required for inventory tests.
	// Running multiple startup script is not supported. We have project level and
	// instance level startup scripts, but only one of them gets executed by virtue
	// of concept of overriding.
	// A way to solve this is to spin up the vm with a project level startup script
	// that installs the osconfig-agent and once the agent is installed, replace
	// the startup script that queries the packages and redirects to serial console.
	// The only caveat is that it requires a reboot which would ultimately increase
	// test running time.

	yumInstallAgent = `
while ! yum install -y google-osconfig-agent; do
if [[ n -gt 3 ]]; then
  exit 1
fi
n=$[$n+1]
sleep 5
done
`
	curlPost = `
uri=http://metadata.google.internal/computeMetadata/v1/instance/guest-attributes/osconfig_tests/install_done
curl -X PUT --data "1" $uri -H "Metadata-Flavor: Google"
`

	// InstallOSConfigDeb installs the osconfig agent on deb based systems.
	InstallOSConfigDeb = `echo 'deb http://packages.cloud.google.com/apt google-osconfig-agent-stretch-unstable main' >> /etc/apt/sources.list
curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
apt-get update
apt-get install -y google-osconfig-agent` + curlPost

	// InstallOSConfigGooGet installs the osconfig agent on googet based systems.
	InstallOSConfigGooGet = `Start-Sleep 10
c:\programdata\googet\googet.exe -noconfirm install -sources https://packages.cloud.google.com/yuck/repos/google-osconfig-agent-unstable google-osconfig-agent
$uri = 'http://metadata.google.internal/computeMetadata/v1/instance/guest-attributes/osconfig_tests/install_done'
Invoke-RestMethod -Method PUT -Uri $uri -Headers @{"Metadata-Flavor" = "Google"} -Body 1`

	// InstallOSConfigYumEL7 installs the osconfig agent on el7 based systems.
	InstallOSConfigYumEL7 = `cat > /etc/yum.repos.d/google-osconfig-agent.repo <<EOM
[google-osconfig-agent]
name=Google OSConfig Agent Repository
baseurl=https://packages.cloud.google.com/yum/repos/google-osconfig-agent-el7-unstable
enabled=1
gpgcheck=0
repo_gpgcheck=1
gpgkey=https://packages.cloud.google.com/yum/doc/yum-key.gpg
       https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg
EOM` + yumInstallAgent + curlPost

	// InstallOSConfigYumEL6 installs the osconfig agent on el6 based systems.
	InstallOSConfigYumEL6 = `sleep 10
cat > /etc/yum.repos.d/google-osconfig-agent.repo <<EOM
[google-osconfig-agent]
name=Google OSConfig Agent Repository
baseurl=https://packages.cloud.google.com/yum/repos/google-osconfig-agent-el6-unstable
enabled=1
gpgcheck=0
repo_gpgcheck=1
gpgkey=https://packages.cloud.google.com/yum/doc/yum-key.gpg
       https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg
EOM` + yumInstallAgent + curlPost
)

// HeadAptImages is a map of names to image paths for public image families that use APT.
var HeadAptImages = map[string]string{
	// Debian images.
	"debian-cloud/debian-9": "projects/debian-cloud/global/images/family/debian-9",

	// Ubuntu images.
	"ubuntu-os-cloud/ubuntu-1604-lts": "projects/ubuntu-os-cloud/global/images/family/ubuntu-1604-lts",
	"ubuntu-os-cloud/ubuntu-1804-lts": "projects/ubuntu-os-cloud/global/images/family/ubuntu-1804-lts",
}

// OldAptImages is a map of names to image paths for old images that use APT.
var OldAptImages = map[string]string{
	// Debian images.
	"old/debian-9": "projects/debian-cloud/global/images/debian-9-stretch-v20190116",

	// Ubuntu images.
	"old/ubuntu-1604-lts": "projects/ubuntu-os-cloud/global/images/ubuntu-1604-xenial-v20190122a",
	"old/ubuntu-1804-lts": "projects/ubuntu-os-cloud/global/images/ubuntu-1804-bionic-v20190122",
}

// HeadEL6Images is a map of names to image paths for public EL6 image families.
var HeadEL6Images = map[string]string{
	"centos-cloud/centos-6": "projects/centos-cloud/global/images/family/centos-6",
	"rhel-cloud/rhel-6":     "projects/rhel-cloud/global/images/family/rhel-6",
}

// OldEL6Images is a map of names to image paths for old EL6 images.
var OldEL6Images = map[string]string{
	"old/centos-6": "projects/centos-cloud/global/images/centos-6-v20181113",
	"old/rhel-6":   "projects/rhel-cloud/global/images/rhel-6-v20181113",
}

// HeadEL7Images is a map of names to image paths for public EL7 image families.
var HeadEL7Images = map[string]string{
	"centos-cloud/centos-7": "projects/centos-cloud/global/images/family/centos-7",
	"rhel-cloud/rhel-7":     "projects/rhel-cloud/global/images/family/rhel-7",
}

// HeadEL8Images is a map of names to image paths for public EL8 image families.
var HeadEL8Images = map[string]string{
	"rhel-cloud/rhel-8": "projects/rhel-cloud/global/images/family/rhel-8",
}

// OldEL7Images is a map of names to image paths for old EL7 images.
var OldEL7Images = map[string]string{
	"old/centos-7": "projects/centos-cloud/global/images/centos-7-v20181113",
	"old/rhel-7":   "projects/rhel-cloud/global/images/rhel-7-v20181113",
}

// HeadELImages is a map of names to image paths for public EL image families.
var HeadELImages = func() (newMap map[string]string) {
	newMap = make(map[string]string)
	for k, v := range HeadEL6Images {
		newMap[k] = v
	}
	for k, v := range HeadEL7Images {
		newMap[k] = v
	}
	for k, v := range HeadEL8Images {
		newMap[k] = v
	}
	return
}()

// HeadWindowsImages is a map of names to image paths for public Windows image families.
var HeadWindowsImages = map[string]string{
	"windows-cloud/windows-2008-r2":      "projects/windows-cloud/global/images/family/windows-2008-r2",
	"windows-cloud/windows-2012-r2":      "projects/windows-cloud/global/images/family/windows-2012-r2",
	"windows-cloud/windows-2012-r2-core": "projects/windows-cloud/global/images/family/windows-2012-r2-core",
	"windows-cloud/windows-2016":         "projects/windows-cloud/global/images/family/windows-2016",
	"windows-cloud/windows-2016-core":    "projects/windows-cloud/global/images/family/windows-2016-core",
	"windows-cloud/windows-2019":         "projects/windows-cloud/global/images/family/windows-2019",
	"windows-cloud/windows-2019-core":    "projects/windows-cloud/global/images/family/windows-2019-core",
	"windows-cloud/windows-1803-core":    "projects/windows-cloud/global/images/family/windows-1803-core",
	"windows-cloud/windows-1809-core":    "projects/windows-cloud/global/images/family/windows-1809-core",
}

// OldWindowsImages is a map of names to image paths for old Windows images prepped for tests.
var OldWindowsImages = map[string]string{
	"old/windows-2008-r2":      "projects/compute-image-osconfig-agent/global/images/windows-2008-r2-v20190515",
	"old/windows-2012-r2":      "projects/compute-image-osconfig-agent/global/images/windows-2012-r2-v20190515",
	"old/windows-2012-r2-core": "projects/compute-image-osconfig-agent/global/images/windows-2012-r2-core-v20190515",
	"old/windows-2016":         "projects/compute-image-osconfig-agent/global/images/windows-2016-v20190515",
	"old/windows-2016-core":    "projects/compute-image-osconfig-agent/global/images/windows-2016-core-v20190515",
	"old/windows-2019":         "projects/compute-image-osconfig-agent/global/images/windows-2019-v20190515",
	"old/windows-2019-core":    "projects/compute-image-osconfig-agent/global/images/windows-2019-core-v20190515",
	"old/windows-1803-core":    "projects/compute-image-osconfig-agent/global/images/windows-1803-core-v20190515",
	"old/windows-1809-core":    "projects/compute-image-osconfig-agent/global/images/windows-1809-core-v20190515",
}

// RandString generates a random string of n length.
func RandString(n int) string {
	gen := rand.New(rand.NewSource(time.Now().UnixNano()))
	letters := "bdghjlmnpqrstvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[gen.Int63()%int64(len(letters))]
	}
	return string(b)
}

// GetStatusFromError return a string that contains all information
// about the error that is created from a status
func GetStatusFromError(err error) string {
	if s, ok := status.FromError(err); ok {
		return fmt.Sprintf("code: %q, message: %q, details: %q", s.Code(), s.Message(), s.Details())
	}
	return fmt.Sprintf("%v", err)
}

// CreateComputeInstance is an utility function to create gce instance
func CreateComputeInstance(metadataitems []*api.MetadataItems, client daisyCompute.Client, machineType, image, name, projectID, zone, serviceAccountEmail string, serviceAccountScopes []string) (*compute.Instance, error) {
	var items []*api.MetadataItems

	// enable debug logging and guest-attributes for all test instances
	items = append(items, compute.BuildInstanceMetadataItem("enable-os-config-debug", "true"))
	items = append(items, compute.BuildInstanceMetadataItem("enable-guest-attributes", "true"))

	for _, item := range metadataitems {
		items = append(items, item)
	}

	i := &api.Instance{
		Name:        name,
		MachineType: fmt.Sprintf("projects/%s/zones/%s/machineTypes/%s", projectID, zone, machineType),
		NetworkInterfaces: []*api.NetworkInterface{
			&api.NetworkInterface{
				Network: "global/networks/default",
				AccessConfigs: []*api.AccessConfig{
					&api.AccessConfig{
						Type: "ONE_TO_ONE_NAT",
					},
				},
			},
		},
		Metadata: &api.Metadata{
			Items: items,
		},
		Disks: []*api.AttachedDisk{
			&api.AttachedDisk{
				AutoDelete: true,
				Boot:       true,
				InitializeParams: &api.AttachedDiskInitializeParams{
					SourceImage: image,
					DiskType:    fmt.Sprintf("projects/%s/zones/%s/diskTypes/pd-ssd", projectID, zone),
				},
			},
		},
		ServiceAccounts: []*api.ServiceAccount{
			&api.ServiceAccount{
				Email:  serviceAccountEmail,
				Scopes: serviceAccountScopes,
			},
		},
	}

	inst, err := compute.CreateInstance(client, projectID, zone, i)
	if err != nil {
		return nil, err
	}

	return inst, nil
}
