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
DEB_RELEASE="$(curl -f -H Metadata-Flavor:Google ${URL}/debian_release)"
INSTALL_GCE="$(curl -f -H Metadata-Flavor:Google ${URL}/install_gce_packages)"
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
for f in proc sys dev; do
  mount -o bind /$f ${MNT}/$f
done
cp /etc/resolv.conf ${MNT}/etc/resolv.conf

# Install GCE packages if requested.
if [[ "${INSTALL_GCE}" == "true" ]]; then
  echo "Installing GCE packages."
  wget https://packages.cloud.google.com/apt/doc/apt-key.gpg -O ${MNT}/tmp/gce_key
  chroot ${MNT} apt-key add /tmp/gce_key
  rm -f ${MNT}/tmp/gce_key
  chroot ${MNT} cat > /etc/apt/sources.list.d/google-cloud.list <<EOF
deb http://packages.cloud.google.com/apt cloud-sdk-${DEB_RELEASE} main
deb http://packages.cloud.google.com/apt google-compute-engine-${DEB_RELEASE}-stable main
deb http://packages.cloud.google.com/apt google-cloud-packages-archive-keyring-${DEB_RELEASE} main
EOF

  # Remove Azure agent.
  chroot ${MNT} apt-get remove -y -f waagent walinuxagent

  chroot ${MNT} apt-get update
  chroot ${MNT} /bin/bash -c DEBIAN_FRONTEND=noninteractive \
    apt-get install --assume-yes --no-install-recommends \
    google-cloud-packages-archive-keyring \
    google-cloud-sdk \
    google-compute-engine \
    python-google-compute-engine \
    pythone3-google-compute-engine

  if [ $? -ne 0 ]; then
    echo "TranslateFailed: GCE package install failed."
    exit 1
  fi
fi

# Update grub config to log to console.
chroot ${MNT} sed -i \
  's#^\(GRUB_CMDLINE_LINUX=".*\)"$#\1 console=ttyS0,38400n8"#' \
  /etc/default/grub

# Disable predictive network interface naming in Stretch.
if [[ "${DEB_RELEASE}" == "stretch" ]]; then
  chroot ${MNT} sed -i \
    's#^\(GRUB_CMDLINE_LINUX=".*\)"$#\1 net.ifnames=0 biosdevname=0"#' \
    /etc/default/grub
fi

chroot ${MNT} update-grub2
if [ $? -ne 0 ]; then
  echo "TranslateFailed: Grub update failed."
  exit 1
fi

# Reset network for DHCP.
echo "Resetting network to DHCP for eth0."
chroot ${MNT} cat > /etc/network/interfaces <<EOF
source-directory /etc/network/interfaces.d
auto lo
iface lo inet loopback
auto eth0
iface eth0 inet dhcp
EOF

# Remove SSH host keys.
echo "Removing SSH host keys."
rm -f ${MNT}/etc/ssh/ssh_host_*

for f in proc sys dev; do
  umount -l ${MNT}/$f
done
umount -l ${MNT}

echo "TranslateSuccess: Translation finished."
