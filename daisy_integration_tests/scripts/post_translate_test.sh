#!/bin/bash
# Copyright 2017 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Basic image tests to validate if Linux image translation was successful.

FAIL=0
FAILURES=""

function wait_for_connectivity {
  if grep -q 'CentOS release 6' /etc/centos-release && command -v nmcli; then
    status "CentOS 6: Waiting for network connectivity using nmcli."
  else
    return
  fi

  for i in {0..60}; do
    if [[ $(nmcli -t -f STATE nm) == "connected" ]]; then
      status "Network ready."
      return
    fi
    sleep 5s
  done

  echo "FAILED: Unable to initialize network connection."
  exit 1
}

function status {
  local message="${1}"
  echo "STATUS: ${message}"
}

function fail {
  local message="${1}"
  FAIL=$((FAIL+1))
  FAILURES+="TestFailed: $message"$'\n'
}

# Check network connectivity.
function check_connectivity {
  status "Checking metadata connection."
  ping -q -c 2 metadata.google.internal
  if [[ $? -ne 0 ]]; then
    fail "Failed to connect to metadata.google.internal."
  fi

  status "Checking external connectivity."
  ping -q -c 2 google.com
  if [[ $? -ne 0 ]]; then
    fail "Failed to ping google.com."
  fi
}

# Check Google services.
function check_google_services {
  status "Checking if instance setup ran."
  if [[ ! -f /etc/default/instance_configs.cfg ]]; then
    fail "Instance setup failed."
  fi

  # Upstart
  if [[ -d /etc/init ]]; then
    status "Checking google-accounts-daemon."
    if initctl status google-accounts-daemon | grep -qv 'running'; then
      fail "Google accounts daemon not running."
    fi

    status "Checking google-network-daemon."
    if initctl status google-network-daemon | grep -qv 'running'; then
      fail "Google Network daemon not running."
    fi

    status "Checking google-clock-skew-daemon."
    if initctl status google-clock-skew-daemon | grep -qv 'running'; then
      fail "Google Clock Skew daemon not running."
    fi
  else
    status "Checking google-accounts-daemon."
    service google-accounts-daemon status
    if [[ $? -ne 0 ]]; then
      fail "Google accounts daemon not running."
    fi

    status "Checking google-network-daemon."
    service google-network-daemon status
    if [[ $? -ne 0 ]]; then
      fail "Google Network daemon not running."
    fi

    status "Checking google-clock-skew-daemon."
    service google-clock-skew-daemon status
    if [[ $? -ne 0 ]]; then
      fail "Google Clock Skew daemon not running."
    fi
  fi
}

# Check Google Cloud SDK.
function check_google_cloud_sdk {
  # Skip for EL6
  if [ -f /etc/redhat-release ]; then
    grep -q "release 6" /etc/redhat-release
    if [ $? -eq 0 ]; then
      return
    fi
  fi

  # Skip for SUSE (gcloud and gsutil aren't in all of their repos)
  if [ -f /etc/os-release ]; then
    grep -qi "suse" /etc/os-release
    if [ $? -eq 0 ]; then
      return
    fi
  fi

  status "Checking for gcloud."
  gcloud version
  if [[ $? -ne 0 ]]; then
    fail "gcloud is not installed."
  fi

  status "Checking for gsutil."
  gsutil -v
  if [[ $? -ne 0 ]]; then
    fail "gsutil is not installed."
  fi
}

# Check cloud-init if it exists.
function check_cloud_init {
  if [[ -x /usr/bin/cloud-init ]]; then
    status "Checking if cloud-init runs."
    for i in $(seq 1 20) ; do
      cloud-init init && return 0
      status "Waiting until cloud-init finishes its startup run."
      sleep $((i**2))
    done
    fail "cloud-init failed to run."
  fi
}

# Check package installs.
function check_package_install {
  # Apt
  if [[ -d /etc/apt ]]; then
    status "Checking if apt can install a package."
    for i in $(seq 1 20) ; do
      apt-get update && apt-get install --reinstall make && return 0
      status "Waiting until apt is available."
      sleep $((i**2))
    done
    fail "apt-get cannot install make."
  fi

  # Yum
  if [[ -d /etc/yum ]]; then
    status "Checking if yum can install a package."
    if rpm -q make; then
      yum -y update make
    else
      yum -y install make
    fi
    yum -y reinstall make
    if [[ $? -ne 0 ]]; then
      fail "yum cannot install make."
    fi
  fi

  # Zypper
  if [[ -d /etc/zypp ]]; then
    status "Checking if zypper can install a package."
    zypper install -f -y make
    if [[ $? -ne 0 ]]; then
      fail "zypper cannot install make."
    fi
  fi
}

# Ensure there's network connectivity before running the tests.
wait_for_connectivity

# Run tests.
check_google_services
check_google_cloud_sdk
check_cloud_init
check_package_install
check_connectivity

# Return results.
if [[ ${FAIL} -eq 0 ]]; then
  echo "PASSED: All tests passed!"
else
  echo "FAILED: ${FAIL} tests failed. ${FAILURES}"
fi
