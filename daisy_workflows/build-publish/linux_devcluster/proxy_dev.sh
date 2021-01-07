#!/bin/bash
apt-get update
if [ $? -ne 0 ]; then
  echo "Trying cache update again."
  apt-get update
  if [ $? -ne 0 ]; then
    echo "BuildFailed: Apt cache is failing to update."
    exit 1
  fi
fi

apt-get -y install tinyproxy
if [ $? -ne 0 ]; then
  echo "BuildFailed: Package install failed."
  exit 1
fi

cp /etc/tinyproxy.conf /etc/ssh/sshd_config /tmp
sed -e 's:^Port.*8888:Port 22:g' \
  -e 's:^MaxClients.*$:MaxClients 100000:g' \
  -e 's:^MaxSpareServers.*$:MaxSpareServers 10000:g' \
  -e 's:^MinSpareServers.*$:MinSpareServers 7500:g' \
  -e 's:^StartServers.*$:StartServers 10000:g' \
  -e 's:^Allow.*$:Allow 0.0.0.0/0\nAllow \:\:/0:g' \
  -e 's:^ConnectPort:#ConnectPort:g' \
  -e 's:^#DisableViaHeader:DisableViaHeader:g' \
  -i /etc/tinyproxy.conf

diff /tmp/tinyproxy.conf /etc/tinyproxy.conf
sed -e 's:^Port.*22:Port 8888:g' -i /etc/ssh/sshd_config
diff /tmp/sshd_config /etc/ssh/sshd_config

# b/38379154#comment23 shows that mtu shouldn't exceed 1424. I chose 1280
# to match min value of IPv6 MTU per RFC 2460, even though eth0 inside proxy vm
# is currently configured only for IPv4. It works, although it might not be
# the most efficient.
sed -e '$asupersede interface-mtu 1280;' -i /etc/dhcp/dhclient.conf

adduser --gecos ''  --disabled-password nebulatest
NEBULA_TEST_HOME=~nebulatest
SSH_KEY_DIR="${NEBULA_TEST_HOME}/.ssh"
if [[ ! -d ${SSH_KEY_DIR} ]]; then
   mkdir ${SSH_KEY_DIR}
   chown nebulatest: ${SSH_KEY_DIR}
fi

# This public key is from //cloud/cluster/borg/nebula.id_rsa.pub
cat >> ${SSH_KEY_DIR}/authorized_keys << EOF
ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC3O/syJjBNSsxUNPYoNJ2YtE6eoUlp0TycLSwD2832VIY0O8C/Qobrbi91S1GBN2iR9Pul6T6sHv6xGhBdM0KC4bwrldsZoDPPLflindFGU4pabVksrIENB4NyJcm5ycUpjTuj1jIydUdYHLR0Vy5P0TSvr8F0l0M2YlWmq4tDjGmDK9zfQJPVYUELDel4IBq5rritrf0DaxDWFY7ef112hpELY9RO6ximcjEHRFbRp1HB+kRg1DYeyfxw8uPYaQG3qnN8DhEQ+ogbzkFGApXQgt22AKiK5pFGm8MqhQAAqqmDZ/cHTU8ymajlc4eZKm/sByR7ahVIINjd5tDQBOWL nebula@nebula
EOF
chown nebulatest: ${SSH_KEY_DIR}/authorized_keys
chmod 0600 ${SSH_KEY_DIR}/authorized_keys

echo "BuildSuccess: Proxy build succeeded."
