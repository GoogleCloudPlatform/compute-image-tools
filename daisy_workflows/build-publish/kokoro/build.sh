#!/bin/bash

function build_fail() {
  echo "BuildFailed: $@"
  exit 1
}

function exit_error() {
  build_fail "$0:$1 \"$BASH_COMMAND\" returned $?"
}

trap 'exit_error $LINENO' ERR
set -x

systemctl stop google-guest-agent
systemctl stop google-osconfig-agent

systemctl disable google-osconfig-agent
systemctl disable google-guest-agent
systemctl disable google-startup-scripts
systemctl disable google-shutdown-scripts
systemctl disable google-oslogin-cache.timer

# Remove any users created on the build host.
if [[ -e /var/lib/google/google_users ]]; then
  for f in $(</var/lib/google/google_users); do
    userdel -rf $f
  done
fi

useradd -c "kbuilder user" -m -s /bin/bash kbuilder
groupadd kokoro
gpasswd -a kbuilder kokoro

(
cd ~kbuilder

mkdir .ssh
chmod 0700 .ssh

metadata='http://169.254.169.254/computeMetadata/v1'
curl -Ss -H Metadata-Flavor:Google \
  "${metadata}/instance/attributes/kokoro_authorized_keys" > .ssh/authorized_keys
chmod 0600 .ssh/authorized_keys

chown -R kbuilder:kbuilder .ssh
)

mkdir -p /tmpfs
echo "tmpfs /tmpfs tmpfs rw,nosuid,nodev 0 0" >> /etc/fstab

dnf -y install rsync-daemon git nmap-ncat python3-psutil

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

cat >> /etc/sysconfig/network-scripts/ifcfg-eth0 <<EOF
DEVICE=eth0
PREFIX=24
IPADDR=169.254.0.2
GATEWAY=169.254.0.1
DNS=8.8.8.8
ONBOOT=yes
BOOTPROTO=none

IPV6INIT=yes
IPV6_AUTOCONF=yes
IPV6ADDR=fd00::1/128
EOF

hostname="kokoro-ubuntu"
fqdn="kokoro-ubuntu.prod.google.com"
hostname "$hostname"
hosts_str="$fqdn $hostname"
sed -i'' -e '/127\.0\.1\.1/d' /etc/hosts
sed -i'' -e "/^127\.0\.0\.1.*/c 127\.0\.0\.1 ${hosts_str} localhost" /etc/hosts

# Increase SSHD logging and direct to console for debugging
echo 'LogLevel DEBUG' >> /etc/ssh/sshd_config
echo 'authpriv.* /dev/console' >> /etc/rsyslog.d/90-google.conf

# Re-enable ssh-rsa key algorithms, due to old paramiko.
echo 'HostKeyAlgorithms +ssh-rsa' >> /etc/ssh/sshd_config
echo 'PubkeyAcceptedKeyTypes +ssh-rsa' >> /etc/ssh/sshd_config

cat > /usr/bin/show-net <<EOF
#!/bin/bash

set -x

ip route list
ip -6 route list
ip addr list
EOF

chmod +x /usr/bin/show-net

cat > /etc/systemd/system/show-net.service <<EOF
[Unit]
After=sshd.service
After=network-online.target NetworkManager.service
Wants=network-online.target
Description=Show network config

[Service]
Type=simple
ExecStart=/usr/bin/show-net

[Install]
WantedBy=multi-user.target
WantedBy=network-online.target NetworkManager.service
EOF

systemctl enable show-net

cat > /etc/systemd/system/build-ready.service <<EOF
[Unit]
After=sshd.service
Description=Touch ready file

[Service]
Type=simple
ExecStartPre=touch /tmpfs/READY
ExecStart=chown -R kbuilder:kokoro /tmpfs

[Install]
WantedBy=multi-user.target
EOF

systemctl enable build-ready

# TODO: build SELinux policy to enable rsyncd for /tmpfs and /home/kbuilder
setenforce 0
sed -i'' -e '/^SELINUX=/s/=enforcing/=disabled/' /etc/selinux/config

echo "BuildSuccess: Kokoro signing image build succeeded."
