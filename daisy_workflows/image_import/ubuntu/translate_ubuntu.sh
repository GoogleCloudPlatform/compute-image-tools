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
UBU_RELEASE="$(curl -f -H Metadata-Flavor:Google ${URL}/ubuntu_release)"
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
for f in proc sys dev run; do
  mount -o bind /$f ${MNT}/$f
done
mkdir -p /run/resolvconf
cp /etc/resolv.conf /run/resolvconf/resolv.conf

# Install GCE packages if requested.
if [[ "${INSTALL_GCE}" == "true" ]]; then
  chroot ${MNT} apt-get update
  echo "Installing cloud-init."
  chroot ${MNT} apt-get install -y --no-install-recommends cloud-init

  # Try to remove azure or aws configs so cloud-init has a chance.
  rm -f ${MNT}/etc/cloud/cloud.cfg.d/*azure*
  rm -f ${MNT}/etc/cloud/cloud.cfg.d/*waagent*
  rm -f ${MNT}/etc/cloud/cloud.cfg.d/*walinuxagent*
  rm -f ${MNT}/etc/cloud/cloud.cfg.d/*aws*
  rm -f ${MNT}/etc/cloud/cloud.cfg.d/*amazon*

  # Remove Azure agent.
  chroot ${MNT} apt-get remove -y -f waagent walinuxagent

  cat > ${MNT}/etc/apt/sources.list.d/partner.list <<EOF
# Enabled for Google Cloud SDK
deb http://archive.canonical.com/ubuntu ${UBU_RELEASE} partner
EOF

  cat > ${MNT}/etc/cloud/cloud.cfg.d/91-gce-system.cfg <<EOF
datasource_list: [ GCE ]
system_info:
   package_mirrors:
     - arches: [i386, amd64]
       failsafe:
         primary: http://archive.ubuntu.com/ubuntu
         security: http://security.ubuntu.com/ubuntu
       search:
         primary:
           - http://%(region)s.gce.archive.ubuntu.com/ubuntu/
           - http://%(availability_zone)s.gce.clouds.archive.ubuntu.com/ubuntu/
           - http://gce.clouds.archive.ubuntu.com/ubuntu/
         security: []
     - arches: [armhf, armel, default]
       failsafe:
         primary: http://ports.ubuntu.com/ubuntu-ports
         security: http://ports.ubuntu.com/ubuntu-ports
EOF

  chroot ${MNT} cloud-init init
  echo "Installing GCE packages."
  chroot ${MNT} apt-get update
  chroot ${MNT} apt-get install --assume-yes --no-install-recommends \
    gce-compute-image-packages \
    google-cloud-sdk

  if [ $? -ne 0 ]; then
    echo "TranslateFailed: GCE package install failed."
    exit 1
  fi
fi

# Update grub config to log to console.
chroot ${MNT} sed -i \
  's#^\(GRUB_CMDLINE_LINUX=".*\)"$#\1 console=ttyS0,38400n8"#' \
  /etc/default/grub

chroot ${MNT} update-grub2
if [ $? -ne 0 ]; then
  echo "TranslateFailed: Grub update failed."
  exit 1
fi

# Remove SSH host keys.
echo "Removing SSH host keys."
rm -f ${MNT}/etc/ssh/ssh_host_*

for f in proc sys dev run; do
  umount -l ${MNT}/$f
done
umount -l ${MNT}

echo "TranslateSuccess: Translation finished."
