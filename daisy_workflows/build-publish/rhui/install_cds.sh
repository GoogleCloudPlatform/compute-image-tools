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
  exit 1
}

trap 'exit_error $LINENO' ERR

build_status "Starting CDS setup."

# Get sources from daisy workflow.
MDS_URL="http://169.254.169.254/computeMetadata/v1"
SRC_PATH=$(curl -f -H Metadata-Flavor:Google \
  ${MDS_URL}/instance/attributes/daisy-sources-path)
tempdir=$(mktemp -d /tmp/daisy-cdsXXX)
gsutil cp "${SRC_PATH}/cds_artifacts/*" $tempdir/

# Get secrets.
gcloud secrets versions access latest --secret entitlement_cert > \
  $tempdir/entitlement_cert.pem
gcloud secrets versions access latest --secret rhua_ca_cert > \
  $tempdir/ca.crt
gcloud secrets versions access latest --secret rhui_tls_key > \
  $tempdir/rhui.key
# Note rhui.crt is provided via Daisy, not secret.

# The user_supplied_* fields would normally be prompted for when you choose the
# 'add CDS' screen in rhui-manager. The _src fields are defined by us in
# cds.patch and allow local Ansible installation. As with RHUA, all 3 CAs are
# the same.
cat >>$tempdir/answers.yaml <<EOF
user_supplied_ssl_crt: $tempdir/rhui.crt
user_supplied_ssl_key: $tempdir/rhui.key
rhui_ca_crt_src: $tempdir/ca.crt
ssl_ca_crt_src: $tempdir/ca.crt
entitlement_ca_crt_src: $tempdir/ca.crt
rhui_tools_conf_src: $tempdir/rhui-tools.conf
EOF

# Make 'RHUA' resolveable so the NginX service can be started in the Ansible
# playbook. TODO: remove RHUA connectivity from NginX config.
echo '127.0.0.2 rhua.rhui.google' >> /etc/hosts

# Import entitlement certificate.
subscription-manager import --certificate=$tempdir/entitlement_cert.pem

# Enable base repos for installing CDS. Using the RHUI versions bc we have the
# RHUI subscription attached.
subscription-manager repos --enable=rhel-8-for-x86_64-baseos-rhui-rpms
subscription-manager repos --enable=rhel-8-for-x86_64-appstream-rhui-rpms

# Enable RHUI repo to get rhui-tools, which contains the CDS Ansible playbook
subscription-manager repos --enable=rhui-4-for-rhel-8-x86_64-rpms

# Get rhui-tools and patch Ansible playbook
# The patch skips mounting NFS and enables local Ansible installation.
# TODO: temporarily pin version to match patch.
dnf install -y rhui-tools-4.2.0.9-1.el8ui patch
( cd /usr/share/rhui-tools; patch -b -p1 < $tempdir/cds.patch; )

build_status "Run Ansible playbook."
ansible-playbook \
  -i localhost, \
  --extra-vars @$tempdir/answers.yaml \
  /usr/share/rhui-tools/playbooks/cds-register.yml

# Disable repos from final image.
subscription-manager repos --disable=rhel-8-for-x86_64-baseos-rhui-rpms
subscription-manager repos --disable=rhel-8-for-x86_64-appstream-rhui-rpms

# Install health checks.
install -D -t /opt/google-rhui-infra $tempdir/health_check.py
for unit in rhui-health-check.{service,timer}; do
  install -m 664 -t /etc/systemd/system $tempdir/$unit
  systemctl enable $unit
done
install -m 664 -t /etc/nginx/conf.d $tempdir/health_check.nginx.conf

cd $tempdir
curl -sSO https://dl.google.com/cloudagents/add-google-cloud-ops-agent-repo.sh
bash ./add-google-cloud-ops-agent-repo.sh --also-install --remove-repo
cd /
install -m 664 -T $tempdir/config.opsagent.yaml /etc/google-cloud-ops-agent/config.yaml
# Status handler is used by the ops agent to collect nginx metrics.
install -m 664 -t /etc/nginx/conf.d $tempdir/status.nginx.conf

# Delete installer resources.
rm -rf $tempdir
# Remove the RHUA hack entry added on line 58. TODO: remove when installer is updated.
sed -i"" -e '/rhua.rhui.google/d' /etc/hosts

build_success "CDS setup complete."
