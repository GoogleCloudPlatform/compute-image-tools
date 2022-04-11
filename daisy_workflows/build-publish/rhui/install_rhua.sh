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
gcloud secrets versions access latest --secret enrollment_cert > \
  $tempdir/enrollment_cert.pem
gcloud secrets versions access latest --secret entitlement_cert > \
  $tempdir/entitlement_cert.pem
gcloud secrets versions access latest --secret rhua_tls_cert > \
  $tempdir/rhua.crt
gcloud secrets versions access latest --secret rhua_tls_key > \
  $tempdir/rhua.key

# RHUA node does not need the CA; it won't be issuing any new certs.
# Not providing files causes new CA to be generated unnecessarily.
echo "DUMMY CA KEY" > $tempdir/rhua_ca.key
echo "DUMMY CA CERT" > $tempdir/rhua_ca.crt

# Add cert entries to answers.yaml
# We defined the tasks which use the new user_supplied_tls_{crt,key} fields in
# rhua.patch. Indentation is important, we are appending to the 'rhua' mapping.
cat >>$tempdir/answers.yaml <<EOF
  user_supplied_ca_crt: $tempdir/rhua_ca.crt
  user_supplied_ca_key: $tempdir/rhua_ca.key
  user_supplied_tls_crt: $tempdir/rhua.crt
  user_supplied_tls_key: $tempdir/rhua.key
EOF

# Generate 'secret' answers in a separate file as top level scalars (mimic
# rhui-installer behavior).
rhuipassword=$(openssl rand -hex 16)
cat >>$tempdir/secret_answers.yaml <<EOF
rhui_active_login_file: null
rhui_manager_password: $rhuipassword
rhui_manager_password_changed: true
EOF

# Import subscription certificate.
subscription-manager import --certificate=$tempdir/enrollment_cert.pem

# Enable repos for installing RHUA.
subscription-manager repos --enable=rhel-8-for-x86_64-baseos-rhui-rpms
subscription-manager repos --enable=rhel-8-for-x86_64-appstream-rhui-rpms
subscription-manager repos --enable=rhui-4-for-rhel-8-x86_64-rpms

# Get rhui-installer and patch Ansible playbook
# The patch skips actually mounting NFS (update fstab only), and skips
# generating a unique RHUA cert, instead copying our pre-generated cert.
dnf install -y rhui-installer patch
( cd /usr/share/rhui-installer; patch -b -p0 < $tempdir/rhua.patch; )

# We don't use rhui-installer as it doesn't allow us to extend the answers file
# and it assumes install-over-SSH. Instead, invoke ansible-playbook directly.
build_status "Run Ansible playbook."
ansible-playbook \
  -i localhost, \
  --extra-vars @$tempdir/answers.yaml \
  --extra-vars @$tempdir/secret_answers.yaml \
  /usr/share/rhui-installer/playbooks/rhua-provision.yml
dnf remove -y rhui-installer

# Remove enrollment cert and RHUI repos from final image.
subscription-manager remove --all

# Add content cert and managed repos.
build_status "Add repos to RHUA."
password=$(awk '/password/ { print $NF }' /etc/rhui/rhui-subscription-sync.conf)
rhui-manager --noninteractive --user admin --password "$password" cert upload \
  --cert $tempdir/entitlement_cert.pem
rhui-manager --noninteractive --user admin --password "$password" repo \
  add_by_repo --repo_ids $(paste -sd "," "${tempdir}/reponames.txt")

# Install health checks.
install -D -t /opt/google-rhui-infra $tempdir/health_check.py
for unit in rhui-health-check.{service,timer}; do
  install -m 664 -t /etc/systemd/system $tempdir/$unit
  systemctl enable $unit
done
install -m 664 -t /etc/nginx/conf.d $tempdir/health_check.nginx.conf

# Add NFS dependencies to pulp worker units
# We do this via drop-ins instead of patching the Ansible templates, as these changes will make the service
# un-startable in the image build environment due to the NFS dependency. We need pulp3 to be running in order
# to run the above rhui-manager commands. TODO(liamh;041122): is this true if we only add deps to
# pulpcore-worker@ ? This may not need to be running to add repo definitions.
unitdir="/etc/systemd/system/pulpcore-worker@.service.d"
[[ -d $unitdir ]] || mkdir $unitdir
install -t $unitdir $tempdir/depend-nfs.conf $tempdir/create-dirs.conf
systemctl daemon-reload

cd $tempdir
curl -sSO https://dl.google.com/cloudagents/add-google-cloud-ops-agent-repo.sh
bash ./add-google-cloud-ops-agent-repo.sh --also-install --remove-repo
yum clean all
cd /

# Delete installer resources.
cp $tempdir/answers.yaml /root/.rhui/answers.yaml  # Expected to be found here.
rm -rf $tempdir
rm -rf /root/.ssh

build_success "RHUA setup complete."
