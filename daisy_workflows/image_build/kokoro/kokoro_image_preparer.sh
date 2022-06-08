#!/bin/bash
# Creating a custom kokoro image from go/kokoro-custom-prod-linux-vm

metadata_url="http://metadata.google.internal/computeMetadata"

GetMetadataAttribute() {
    attribute="$1"

    echo "Fetching ${attribute}"
    url="$metadata_url/v1/instance/attributes/$attribute"
    attribute_value=$(curl -H "Metadata-Flavor: Google" -X GET "$url")
}

# Must have a "kbuilder" user
useradd -c "kbuilder user" -m -s /bin/bash kbuilder
groupadd kokoro
usermod -aG kokoro kbuilder

# kbuilder must allow login with ssh public/private key authentication
GetMetadataAttribute "kokoro_authorized_keys" && authorized_keys="$attribute_value"
mkdir ~kbuilder/.ssh
echo $authorized_keys >> ~kbuilder/.ssh/authorized_keys

# kbuilder needs to be able to login non-interactively and launch commands
cat >> ~kbuilder/.profile <<EOF
if [[ -z $SSH_TTY ]]; then
  source /etc/profile
fi
EOF
# Symlink .bashrc to .profile
ln -s ~kbuilder/.profile ~kbuilder/.bashrc

# Need a /tmpfs directory where Kokoro will upload build scripts and inputs
build_dir="/tmpfs"
build_tmp="/tmpfs/tmp"
# Create build tmp directory
mkdir -p "${build_dir}"
mkdir -p "${build_tmp}"
chmod 1777 "${build_tmp}"
chown -R kbuilder:kokoro "${build_dir}"
chmod 2775 "${build_dir}"
# Create ready file
su - kbuilder -c "touch ${build_dir}/READY"

# Must have a running rsync daemon so that build inputs and outputs can be uploaded and downloaded
dnf -y install rsync-daemon
cat >> /etc/rsyncd.conf <<EOF
use chroot = yes

[tmpfs]
path = /tmpfs
comment = tmpfs
read only = false
write only = false
list = false
uid = kbuilder
gid = kokoro
incoming chmod = g+rw
munge symlinks = no

[kbuilder_home]
path = /home/kbuilder
comment = kbuilder_home
read only = false
write only = false
list = false
uid = kbuilder
munge symlinks = no
EOF
systemctl enable rsyncd.service

# Set network settings
cat >> /etc/sysconfig/network-scripts/ifcfg-eth0 <<EOF
IPADDR=169.254.0.2/24
GATEWAY=169.254.0.1
DNS=8.8.8.8
EOF

# Install needed packages
dnf -y install git nmap-ncat python3-psutil

# Disable Google metadata services
cat > /etc/default/instance_configs.cfg << EOF
[Daemons]
accounts_daemon = false
clock_skew_daemon = false
ip_forwarding_daemon = false

[InstanceSetup]
network_enabled = false
set_boto_config = false

[MetadataScripts]
shutdown = false
startup = false

[NetworkInterfaces]
setup = false
EOF

echo "BuildSuccess: Kokoro signing image build succeeded."