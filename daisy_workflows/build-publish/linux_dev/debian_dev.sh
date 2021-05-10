#!/bin/bash
apt-get update
if [[ $? -ne 0 ]]; then
  echo "Trying cache update again."
  apt-get update
  if [[ $? -ne 0 ]]; then
    echo "BuildFailed: Apt cache is failing to update."
    exit 1
  fi
fi

# Note websockify will change to python-websockify in Debian 9.
export DEBIAN_FRONTEND="noninteractive"
apt-get -y install \
  apache2 arping bzip2 gcc gdb make patch tcpdump zsh fio g++ hping3 libaio1 \
  libaio-dev libevent-dev libmemcached-dev python-dev python-eventlet \
  python-pip sendip strongswan tcpflow traceroute zlib1g-dev dnsutils \
  default-jdk websockify iperf iperf3 pciutils mdadm libaio1 bind9 netcat \
  nmap telnet mz socat sg3-utils

if [[ $? -ne 0 ]]; then
  echo "BuildFailed: Package install failed."
  exit 1
fi

# In order to install NGINX we need to stop apache2 as it binds to port 80
# Install nginx, stop it, restart apache2. Disable nginx from startup.
service apache2 stop
apt-get -y install nginx
if [[ $? -ne 0 ]]; then
  echo "BuildFailed: Installing NGINX failed."
  exit 1
fi
service nginx stop
service apache2 start
update-rc.d -f nginx disable

pip install --upgrade pip
pip install pylibmc python-memcached protobuf google-cloud-storage
if [[ $? -ne 0 ]]; then
  echo "BuildFailed: pip install failed."
  exit 1
fi

echo "BuildSuccess: Debian build succeeded."
