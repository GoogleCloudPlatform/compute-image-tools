#!/bin/bash

set -e
set -x
echo "BuildStatus: Starting RHUA setup."

# Get sources from daisy workflow.
SRC_PATH=$(curl -f -H Metadata-Flavor:Google http://169.254.169.254/computeMetadata/v1/instance/attributes/daisy-sources-path)
tempdir=$(mktemp -d /tmp/daisy-rhuaXXX)
gsutil cp "${SRC_PATH}/*" $tempdir/
gcloud secrets versions access latest --secret enrollment_cert > \
  $tempdir/enrollment_cert.pem
gcloud secrets versions access latest --secret entitlement_cert > \
  $tempdir/entitlement_cert.pem

# Get subscription manager from RHUIv3 hosted repo
dnf --disablerepo='*' --enablerepo='rhui-rhel-8-for-x86_64-baseos-rhui-rpms' \
  install -y subscription-manager

# Remove RHUI config pointing to RHUIv3
rpm -e google-rhui-client-rhel8

# Download and import subscription certificate.
subscription-manager import --certificate=$tempdir/enrollment_cert.pem

# Enable repos for installing RHUA.
subscription-manager repos --enable=rhel-8-for-x86_64-baseos-rhui-rpms
subscription-manager repos --enable=rhel-8-for-x86_64-appstream-rhui-rpms
subscription-manager repos --enable=rhui-4-for-rhel-8-x86_64-rpms

# Permit root login for Ansible
cp /etc/ssh/sshd_config /etc/ssh/sshd_config.bak
sed -i"" -re '/PermitRootLogin/d' /etc/ssh/sshd_config
systemctl restart sshd.service

# Get and run rhui-installer.
dnf install -y rhui-installer
rhui-installer -u root --answers-file $tempdir/answers.yaml
dnf remove -y rhui-installer

# Remove enrollment cert and RHUI repos from final image.
subscription-manager remove --all

# Add content cert and managed repos.
password=$(awk '/password/ { print $NF }' /etc/rhui/rhui-subscription-sync.conf)
rhui-manager --noninteractive --user admin --password "$password" cert upload \
  --cert $tempdir/entitlement_cert.pem
rhui-manager --noninteractive --user admin --password "$password" repo \
  add_by_repo --repo_ids $(paste -sd "," "${tempdir}/reponames.txt")

# Stop pulp services, which may hold open the mount point.
systemctl stop 'pulpcore-worker@*' pulpcore-{api,content,resource-manager}

# Remove rhui-installer added fake mount point.
umount /var/lib/rhui/remote_share
sed -i"" -re '/remote_share/d' /etc/fstab

# Add NFS entry to fstab.
cat $tempdir/fstab >> /etc/fstab

# Add NFS dependency to pulp units
for unit in pulpcore-{worker@,api,content,resource-manager}; do
  unitdir="/etc/systemd/system/${unit}.service.d"
  [[ -d $unitdir ]] || mkdir $unitdir
  cp $tempdir/depend-nfs.conf $unitdir/
done
systemctl daemon-reload

# Delete installer resources.
rm -rf $tempdir
mv -f /etc/ssh/sshd_config.bak /etc/ssh/sshd_config
echo BuildSuccess: RHUA setup complete.
