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
EL_RELEASE="$(curl -f -H Metadata-Flavor:Google ${URL}/el_release)"
INSTALL_GCE="$(curl -f -H Metadata-Flavor:Google ${URL}/install_gce_packages)"
RHEL_LICENSE="$(curl -f -H Metadata-Flavor:Google ${URL}/use_rhel_gce_license)"
MNT="/mnt/imported_disk"

# Mount the imported disk.
mkdir ${MNT}
if [ -b /dev/sdb1 ]; then
  echo "Trying to mount /dev/sdb1"
  mount /dev/sdb1 ${MNT}
  if [ $? -ne 0 ]; then
    echo "TranslateFailed: Unable to mount imported disk."
    exit 1
  fi
else
  echo "Trying to mount /dev/sdb"
  mount /dev/sdb ${MNT}
  if [ $? -ne 0 ]; then
    echo "TranslateFailed: Unable to mount imported disk."
    exit 1
  fi
fi

# Setup DNS for chroot.
for f in proc sys dev run; do
  mount -o bind /$f ${MNT}/$f
done
mount -o bind /dev/pts ${MNT}/dev/pts
cp /etc/resolv.conf ${MNT}/etc/resolv.conf
chroot ${MNT} restorecon /etc/resolv.conf

if [[ "${RHEL_LIENSE}" == "true" ]]; then
  if $(grep -q "Red Hat" /etc/redhat-release); then
    # Remove rhui packages and add the google rhui package.
    yum install -y --downloadonly --downloaddir=${MNT}/tmp google-rhui-client-rhel${EL_RELEASE}
    if [ $? -eq 0 ]; then
      chroot ${MNT} yum remove -y *rhui*
      echo "Adding in GCE RHUI package."
      chroot ${MNT} yum install -y /tmp/google-rhui-client-rhel${EL_RELEASE}*.rpm
    fi
  fi
fi

# Install GCE packages if requested.
if [[ "${INSTALL_GCE}" == "true" ]]; then
  echo "Installing GCE packages."
  cat > ${MNT}/etc/yum.repos.d/google-cloud.repo << EOM
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
    cat >> ${MNT}/etc/yum.repos.d/google-cloud.repo << EOM
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

  chroot ${MNT} yum -y install google-compute-engine python-google-compute-engine
  if [ $? -ne 0 ]; then
    echo "TranslateFailed: GCE package install failed."
    exit 1
  fi
fi

if [[ ${EL_RELEASE} == "7" ]]; then
  # Update grub config to log to console and disable menu
  cat > ${MNT}/etc/default/grub <<- "EOM"
GRUB_TIMEOUT=0
GRUB_DISTRIBUTOR="$(sed 's, release .*$,,g' /etc/system-release)"
GRUB_DEFAULT=saved
GRUB_DISABLE_SUBMENU=true
GRUB_TERMINAL="serial console"
GRUB_SERIAL_COMMAND="serial --speed=38400"
GRUB_CMDLINE_LINUX="crashkernel=auto console=ttyS0,38400n8"
GRUB_DISABLE_RECOVERY="true"
EOM
  chroot ${MNT} restorecon /etc/default/grub
  chroot ${MNT} grub2-mkconfig -o /boot/grub2/grub.cfg
  if [ $? -ne 0 ]; then
    echo "TranslateFailed: Grub update failed."
    exit 1
  fi
fi

# Reset network for DHCP.
echo "Resetting network to DHCP for eth0."
cat > ${MNT}/etc/sysconfig/network-scripts/ifcfg-eth0 <<EOF
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

umount -l ${MNT}/dev/pts
for f in proc sys pts dev run; do
  umount -l ${MNT}/$f
done
umount -l ${MNT}

echo "TranslateSuccess: Translation finished."
