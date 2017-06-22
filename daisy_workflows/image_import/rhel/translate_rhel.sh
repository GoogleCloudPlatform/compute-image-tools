#!/bin/bash
# Copyright 2017 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -x

URL="http://metadata/computeMetadata/v1/instance/attributes"
EL_RELEASE="$(curl -f -H Metadata-Flavor:Google ${URL}/el-release)"
INSTALL_GCE="$(curl -f -H Metadata-Flavor:Google ${URL}/install-gce-packages)"
GCE_LICENSE="$(curl -f -H Metadata-Flavor:Google ${URL}/use-gce-license)"
MNT="/mnt/imported_disk"

# Mount the imported disk.
mkdir ${MNT}
if [ -b /dev/sdb1 ]; then
  echo "Trying to mount /dev/sdb1"
  mount /dev/sdb1 ${MNT}
  if [ $? -ne 0 ]; then
    echo "TranslateFailed: Unable to mount imported disk."
  fi
else
  echo "Trying to mount /dev/sdb"
  mount /dev/sdb ${MNT}
  if [ $? -ne 0 ]; then
    echo "TranslateFailed: Unable to mount imported disk."
  fi
fi

# Setup DNS for chroot.
for f in proc sys dev run, do
  mount -o bind /$f ${MNT}/$f
done
cp /etc/resolv.conf ${MNT}/etc/resolv.conf
chroot ${MNT} restorecon /etc/resolv.conf

if [[ "${GCE_LIENSE}" == "true" ]]; then
  # Remove rhui packages and add the google rhui package.
  yum install -y --downloadonly --downloaddir=${MNT}/tmp google-rhui-client-rhel${EL_RELEASE}
  if [ $? -eq 0 ]; then
    chroot ${MNT} yum remove -y *rhui*
    echo "Adding in GCE RHUI package."
    chroot ${MNT} yum install -y /tmp/google-rhui-client-rhel${EL_RELEASE}*.rpm
  fi
fi

# Install GCE packages if requested.
if [[ "${INSTALL_GCE}" == "true" ]]; then
  echo "Installing GCE packages."
  cat > /etc/yum.repos.d/google-cloud.repo << EOM
[google-cloud-compute]
name=Google Cloud Compute
baseurl=https://packages.cloud.google.com/yum/repos/google-cloud-compute-el${EL_RELEASE}-x86_64
enabled=1
gpgcheck=1
repo_gpgcheck=1
gpgkey=https://packages.cloud.google.com/yum/doc/yum-key.gpg
       https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg
EOM

  if [[ ${EL_RELEASE} == "7" ]]; then
    cat >> /etc/yum.repos.d/google-cloud.repo << EOM
[google-cloud-sdk]
name=Google Cloud SDK
baseurl=https://packages.cloud.google.com/yum/repos/cloud-sdk-el${EL_RELEASE}-x86_64
enabled=1
gpgcheck=1
repo_gpgcheck=1
gpgkey=https://packages.cloud.google.com/yum/doc/yum-key.gpg
       https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg
EOM
  chroot ${MNT} yum -y install google-cloud-sdk
fi

  chroot ${MNT} yum -y install google-compute-engine google-compute-engine-init google-config
  if [ $? -ne 0 ]; then
    echo "TranslateFailed: GCE package install failed."
  fi
fi

if [[ ${EL_RELEASE} == "7" ]]; then
  # Update grub config to log to console.
  chroot ${MNT} sed -i \
    's#^\(GRUB_CMDLINE_LINUX=".*\)"$#\1 console=ttyS0,38400n8"#' \
    /etc/default/grub
  chroot ${MNT} restorecon /etc/default/grub
  chroot ${MNT} grub2-mkconfig -o /boot/grub2/grub.cfg
  if [ $? -ne 0 ]; then
    echo "TranslateFailed: Grub update failed."
  fi
fi

# Reset network for DHCP.
echo "Resetting network to DHCP for eth0."
chroot ${MNT} cat > /etc/sysconfig/network-scripts/ifcfg-eth0 <<EOF
BOOTPROTO=dhcp
DEVICE=eth0
ONBOOT=yes
TYPE=Ethernet
DEFROUTE=yes
PEERDNS=yes
PEERROUTES=yes
DHCP_HOSTNAME=localhost
IPV4_FAILURE_FATAL=no
NAME="System eth0"
MTU=1460
PERSISTENT_DHCLIENT="y"
EOF
chroot ${MNT} restorecon /etc/sysconfig/network-scripts/ifcfg-eth0

# Remove SSH host keys.
echo "Removing SSH host keys."
rm -f ${MNT}/etc/ssh/ssh_host_*

for f in proc sys dev run, do
  umount -l ${MNT}/$f
done
umount -l ${MNT}

echo "TranslateSuccess: Translation finished."
sync
shutdown -h now
