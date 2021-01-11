#!/bin/bash
# Sleep to allow CentOS 6 network to fully come up and avoid IPv6 lookups.
sleep 10
yum -y install httpd bzip2 gcc gdb make net-tools patch tcpdump zsh \
               java-1.8.0-openjdk.x86_64 epel-release mdadm libaio bind-utils

if [ $? -ne 0 ]; then
  echo "BuildFailed: Package install failed."
  exit 1
fi

# EPEL packages.
# python-websockify is not available on centos-8 repository
if grep -q '^\(CentOS\|Red Hat\)[^0-9]*[6-7]\..' /etc/redhat-release; then
  yum -y install python-websockify
fi

if [ $? -ne 0 ]; then
  echo "BuildFailed: EPEL package install failed."
  exit 1
fi

echo "BuildSuccess: CentOS build succeeded."
