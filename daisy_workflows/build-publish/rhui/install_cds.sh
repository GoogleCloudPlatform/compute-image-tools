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
  #build_fail "$0:$1 \"$BASH_COMMAND\" returned $?"
  build_status "$0:$1 \"$BASH_COMMAND\" returned $?"
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
gcloud secrets versions access latest --secret enrollment_cert > \
  $tempdir/enrollment_cert.pem
gcloud secrets versions access latest --secret rhua_ca_cert > \
  $tempdir/ca.crt
# Now provided by daisy workflow.
#gcloud secrets versions access latest --secret rhui_tls_cert > \
#  $tempdir/rhui.crt
gcloud secrets versions access latest --secret rhui_tls_key > \
  $tempdir/rhui.key

# TODO: here we need to do the ACME steps via certbot w/ Google Cloud DNS
# plugin. We need to install both from somewhere, then run them. For now,
# self-signed cert.

# We define the tasks which use these new fields in cds.patch
cat >>$tempdir/answers.yaml <<EOF
rhui_tools_conf: $tempdir/rhui-tools.conf
user_supplied_ca_crt: $tempdir/ca.crt
user_supplied_tls_crt: $tempdir/rhui.crt
user_supplied_tls_key: $tempdir/rhui.key
EOF

# Import enrollment certificate.
subscription-manager import --certificate=$tempdir/enrollment_cert.pem

# Enable base repos for installing CDS. Using the RHUI versions bc we have the
# RHUI subscription attached.
subscription-manager repos --enable=rhel-8-for-x86_64-baseos-rhui-rpms
subscription-manager repos --enable=rhel-8-for-x86_64-appstream-rhui-rpms

# Enable RHUI repo to get rhui-tools, which contains the CDS Ansible playbook
subscription-manager repos --enable=rhui-4-for-rhel-8-x86_64-rpms

# Get rhui-tools and patch Ansible playbook
# The patch skips actually mounting NFS (update fstab only), makes backups of
# files from rhui-tools, and skips generating unique RHUI and CA certs, instead
# copying our pre-generated certs.
dnf install -y rhui-tools patch
( cd /usr/share/rhui-tools; patch -b -p0 < $tempdir/cds.patch; )

build_status "Run Ansible playbook."
ansible-playbook \
  -i localhost, \
  --extra-vars @$tempdir/answers.yaml \
  /usr/share/rhui-tools/playbooks/cds-register.yml

cp /usr/bin/rhui-services-restart /usr/bin/rhui-services-restart.bak
dnf remove -y rhui-tools rhui-tools-libs

# Restore files owned by rhui-tools package.
mv /usr/bin/rhui-services-restart.bak /usr/bin/rhui-services-restart
cat $tempdir/rhui-tools.conf > /etc/rhui/rhui-tools.conf

# Remove enrollment cert and repos from final image.
subscription-manager remove --all

# Install health checks.
install -D -t /opt/google-rhui-infra $tempdir/health_check.py
for unit in rhui-health-check.{service,timer}; do
  install -m 664 -t /etc/systemd/system $tempdir/$unit
  systemctl enable $unit
done

# Delete installer resources.
rm -rf $tempdir

build_success "CDS setup complete."
