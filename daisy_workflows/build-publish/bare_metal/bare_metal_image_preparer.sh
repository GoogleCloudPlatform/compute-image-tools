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
repo_gpgcheck=1
gpgkey=https://packages.cloud.google.com/yum/doc/yum-key.gpg
       https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg
EOF

# Install Packages.
echo "Install rpm packages."
yum -y install mcedaemon google-ecclesia-management-agent

# Check result
if [[ $? -ne 0 ]]; then
  echo "BuildFailed: Packages install failed."
  exit 1
fi
echo "BuildSuccess: Bare Metal image build succeeded."
