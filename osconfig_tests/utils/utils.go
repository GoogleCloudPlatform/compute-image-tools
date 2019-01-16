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
	"math/rand"
	"time"
)

var (
	// InstallOSConfigDeb installs the osconfig agent on deb based systems.
	InstallOSConfigDeb = `echo 'deb http://packages.cloud.google.com/apt google-osconfig-agent-stretch-unstable main' >> /etc/apt/sources.list
curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
apt-get update
apt-get install -y google-osconfig-agent
echo 'osconfig install done'`

	// InstallOSConfigGooGet installs the osconfig agent on googet based systems.
	InstallOSConfigGooGet = `c:\programdata\googet\googet.exe -noconfirm install -sources https://packages.cloud.google.com/yuck/repos/google-osconfig-agent-unstable google-osconfig-agent
echo 'osconfig install done'`

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
EOM
n=0
while ! yum install -y google-osconfig-agent; do
  if [[ n -gt 3 ]]; then
    exit 1
  fi
  n=$[$n+1]
  sleep 5
done
echo 'osconfig install done'`

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
EOM
n=0
while ! yum install -y google-osconfig-agent; do
  if [[ n -gt 3 ]]; then
    exit 1
  fi
  n=$[$n+1]
  sleep 5
done
echo 'osconfig install done'`
)

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
