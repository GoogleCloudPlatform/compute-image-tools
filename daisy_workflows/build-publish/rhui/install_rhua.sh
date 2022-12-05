#!/bin/bash


function build_success() {
  echo "BuildSuccess: $@"
  exit 0
}

function build_status() {
  echo "BuildStatus: $@"
}

function build_fail() {
  echo "BuildFailed: $@"
  exit 1
}

function exit_error() {
  build_fail "$0:$1 \"$BASH_COMMAND\" returned $?"
}

trap 'exit_error $LINENO' ERR

build_status "Starting RHUA setup."

# Get sources from daisy workflow.
MDS_URL="http://169.254.169.254/computeMetadata/v1"
SRC_PATH=$(curl -f -H Metadata-Flavor:Google \
  ${MDS_URL}/instance/attributes/daisy-sources-path)
tempdir=$(mktemp -d /tmp/daisy-rhuaXXX)
gsutil cp "${SRC_PATH}/rhua_artifacts/*" $tempdir/

# Get secrets.

# Entitlement cert: used to both enable access to RHUI content repos (where we get the installer and
# dependencies) and used to enable the RHUA to sync content from RH CDN
gcloud secrets versions access latest --secret entitlement_cert > \
  $tempdir/entitlement_cert.pem
# CA cert & key, used to generate the RHUA cert
gcloud secrets versions access latest --secret rhua_ca_cert > \
  $tempdir/rhua_ca.crt
gcloud secrets versions access latest --secret rhua_ca_key > \
  $tempdir/rhua_ca.key

# Import subscription certificate.
subscription-manager import --certificate=$tempdir/entitlement_cert.pem

# Enable repos for installing RHUA.
subscription-manager repos --enable=rhel-8-for-x86_64-baseos-rhui-rpms
subscription-manager repos --enable=rhel-8-for-x86_64-appstream-rhui-rpms
subscription-manager repos --enable=rhui-4-for-rhel-8-x86_64-rpms
subscription-manager repos --enable=ansible-2-for-rhel-8-x86_64-rhui-rpms

# Get rhui-installer and patch Ansible playbook
# TODO: temporarily pin version to match patch until we move to ISO install
dnf install -y rhui-installer-4.2.0.4-1.el8ui patch

# Patch the rhua playbook and some templates. Disables need for access to
# NFS during install.
( cd /usr/share/rhui-installer; patch -b -p0 < $tempdir/rhua.patch; )

rhui-installer -u root --log-level debug \
  --cds-lb-hostname rhui.googlecloud.com \
  --remote-fs-server nfs.rhui.google:/rhui \
  --rhua-hostname rhua.rhui.google \
  --user-supplied-rhui-ca-crt $tempdir/rhua_ca.crt \
  --user-supplied-rhui-ca-key $tempdir/rhua_ca.key


# Remove rhui-installer, and disable RHUI repos in final image.
dnf remove -y rhui-installer
subscription-manager repos --disable=rhel-8-for-x86_64-baseos-rhui-rpms
subscription-manager repos --disable=rhel-8-for-x86_64-appstream-rhui-rpms
subscription-manager repos --disable=rhui-4-for-rhel-8-x86_64-rpms
subscription-manager repos --disable=ansible-2-for-rhel-8-x86_64-rhui-rpms

# Add content cert and managed repos.
build_status "Add repos to RHUA."
password=$(awk '/password/ { print $NF }' /etc/rhui/rhui-subscription-sync.conf)
rhui-manager --noninteractive --user admin --password "$password" cert upload \
  --cert $tempdir/entitlement_cert.pem
rhui-manager --noninteractive --user admin --password "$password" repo \
  add_by_repo --repo_ids $(paste -sd "," "${tempdir}/reponames.txt")

# Install health checks and RHUA sync status.
install -D -t /opt/google-rhui-infra $tempdir/health_check.py
install -D -t /opt/google-rhui-infra $tempdir/rhua_sync_status.py
for unit in {rhui-health-check,rhua-sync-status}.{service,timer}; do
  install -m 664 -t /etc/systemd/system $tempdir/$unit
  systemctl enable $unit
done
install -m 664 -t /etc/nginx/conf.d $tempdir/health_check.nginx.conf

# Add NFS dependencies to pulp worker units
#
# We do this via drop-ins instead of patching the Ansible templates, as these
# changes will make the service un-startable in the image build environment due
# to the NFS dependency. We need pulp3 to be running in order to run the above
# rhui-manager commands. It's also hoped we can eventually drop the patch.
unitdir="/etc/systemd/system/pulpcore-worker@.service.d"
[[ -d $unitdir ]] || mkdir $unitdir
install -t $unitdir $tempdir/depend-nfs.conf $tempdir/create-dirs.conf
systemctl daemon-reload

# Install the ops-agent
cd $tempdir
curl -sSO https://dl.google.com/cloudagents/add-google-cloud-ops-agent-repo.sh
bash ./add-google-cloud-ops-agent-repo.sh --also-install --remove-repo
yum clean all
cd /
install -m 664 -T $tempdir/config.opsagent.yaml /etc/google-cloud-ops-agent/config.yaml
# Status handler is used by the ops agent to collect nginx metrics.
install -m 664 -t /etc/nginx/conf.d $tempdir/status.nginx.conf

# Delete installer resources.
rm -rf $tempdir
rm -rf /root/.ssh
# No need to keep the CA active on running RHUA.
echo 'DUMMY CA KEY' > /etc/pki/rhui/private/ca.key
echo 'DUMMY CA KEY' > /etc/pki/rhui/private/client_entitlement_ca.key
echo 'DUMMY CA KEY' > /etc/pki/rhui/private/client_ssl_ca.key

build_success "RHUA setup complete."
