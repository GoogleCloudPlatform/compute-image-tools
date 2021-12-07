#!/bin/bash

set -e
echo "BuildStatus: Starting RHUA setup."

SRC_PATH=$(curl -f -H Metadata-Flavor:Google ${URL}/attributes/daisy-sources-path)
RH_USER=$(gcloud secrets versions access latest --secret rh_user)
RH_PASS=$(gcloud secrets versions access latest --secret rh_pass)

# Get sources from daisy workflow.
mkdir /root/daisy_sources
gsutil cp "${SRC_PATH}/" /root/daisy_sources/

# Get subscription manager from RHUIv3 hosted repo
dnf --disablerepo='*' --enablerepo='rhui-rhel-8-for-x86_64-baseos-rhui-rpms' \
  install subscription-manager

# Remove RHUI config pointing to RHUIv3
rpm -e google-rhui-client-rhel8

# Register as a RHUI
# TODO: This won't work without identity
subscription-manager register --type=rhui --name=rhua-installer \
  --user=$RH_USER --password $RH_PASSWORD
#  --consumerid=cf64bd4c-0e1c-407b-a5c5-969768ff6d13
# consumer ID means the literal same registration, just transferred to a new
# host. only works if there's only one build happening at a time. precludes the
# need to attach subscription by poolid

# Attach RHUI subscription
subscription-manager attach --pool=8a85f9a17c71102f017d05dbd9f72ee9

# Enable repos for installing RHUA.
subscription-manager repos --enable=rhel-8-for-x86_64-baseos-rhui-rpms \
  --enable=rhel-8-for-x86_64-appstream-rhui-rpms
subscription-manager repos --enable=rhui-4-for-rhel-8-x86_64-rpms

# Get and run rhui-installer.
# TODO: add our SSL certs
dnf install rhui-installer
rhui-installer --answers-file /root/daisy_sources/answers.yaml

# Unregister so we don't use up licenses.
subscription-manager unregister

# Add our repos.
rhui-manager --noninteractive repo add_by_repo \
  --repo_ids `paste -d "," /root/daisy_sources/reponames.txt`

# TODO: add NFS entry to fstab

echo BuildSuccess: RHUA software is installed and configured
