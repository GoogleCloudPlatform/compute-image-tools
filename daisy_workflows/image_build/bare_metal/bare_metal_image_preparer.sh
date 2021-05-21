#!/bin/bash
# Script to convert GCE VM images to bare metal images.
# Support RHEL-7 and RHEL-8 only.

# Determine release version.
if grep -q '^Red Hat[^0-9]*7\..' /etc/redhat-release; then
    RELEASE="el7"
elif grep -q '^Red Hat[^0-9]*8\..' /etc/redhat-release; then
    RELEASE="el8"
else
    echo "Not implemented for Red Hat ${RELEASE}"
    return 1
fi

# Enable repository.
cat >> /etc/yum.repos.d/google-cloud.repo << EOF
[google-compute-engine-bare-metal]
name=Google Compute Engine Bare Metal
baseurl=https://packages.cloud.google.com/yum/repos/google-compute-engine-bare-metal-${RELEASE}-x86_64-stable
enabled=1
gpgcheck=1
repo_gpgcheck=0
gpgkey=https://packages.cloud.google.com/yum/doc/yum-key.gpg
       https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg
EOF

# Install Packages.
echo "Install rpm packages."
yum -y install mcedaemon google-ecclesia-management-agent
yum -y install gve

# Check result
if [[ $? -ne 0 ]]; then
  echo "BuildFailed: Packages install failed."
  exit 1
fi

URL="http://metadata/computeMetadata/v1/instance/attributes"
DEVELOPMENT=$(curl -f -H Metadata-Flavor:Google ${URL}/development)

if [[ ${DEVELOPMENT} == "True" ]]; then
  if [[ ${RELEASE} == "el8" ]]; then
    echo "Adding EPEL for RHEL 8."
    dnf -y install https://dl.fedoraproject.org/pub/epel/epel-release-latest-8.noarch.rpm
  fi
  # Temporary install of useful development tools.
  echo "Installing development tools."
  yum -y install net-tools pciutils tcpdump strongswan hping3
  # Auto login on root shell
  sed -i 's!ExecStart=-/sbin/agetty .*!# &\nExecStart=-/sbin/agetty -n --autologin root --keep-baud 115200,38400,9600 %I $TERM!' /lib/systemd/system/serial-getty@.service
  systemctl enable serial-getty@ttyS0.service
fi

# Temporary boot fix for RHEL 8.
# Removes grub2 in place of a BootLoaderSpec which is loaded from the firmware.
if [[ "${RELEASE}" == "el8" ]]; then
  echo "Configuring temporary RHEL 8 boot."
  # Remove grub2
  dnf -y remove grub2*
  dnf -y install file
  # Install systemd-bootd
  bootctl install
  rm -rf /boot/loader
  # Install the kernel into /boot
  kernel-install add $(uname -r) /lib/modules/$(uname -r)/vmlinuz
  # Disable auto updates
  systemctl disable dnf-automatic.timer dnf-automatic.service
fi
echo "BuildSuccess: Bare Metal image build succeeded."
