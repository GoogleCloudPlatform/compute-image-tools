#!/bin/bash
# Copyright 2019 Google Inc. All Rights Reserved.
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

# Tests to validate if Ubuntu OVF with 3 disks was imported successfully.


# To disable checking for the OS config agent, add an instance metadata
# value of osconfig_not_supported: true.
url=http://metadata.google.internal/computeMetadata/v1/instance/attributes/osconfig_not_supported
if command -v curl; then
  OSCONFIG_NOT_SUPPORTED="$(curl --header "Metadata-Flavor: Google" $url)"
elif command -v wget; then
  OSCONFIG_NOT_SUPPORTED="$(wget --header "Metadata-Flavor: Google" -O - $url)"
else
  echo "TestFailed: Either curl or wget is required to run test suite."
  exit 1
fi

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

# Assert that a service is running; first trying `initctl`,
# then trying `service`.
function assert_running {
  local service="${1}"
  status "Checking $service"
  if command -v initctl; then
    if ! initctl status "$service"; then
      fail "$service not running."
    fi
  else
    if ! service "$service" status; then
      fail "$service not running."
    fi
  fi
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

function check_metadata_connectivity {
  status "Checking metadata connection."
  for i in $(seq 1 20) ; do
    ping -q -w 10 metadata.google.internal && return 0
    status "Waiting for connectivity to metadata.google.internal."
    sleep $((i**2))
  done
  fail "Failed to connect to metadata.google.internal."
}

function check_internet_connectivity {
  status "Checking external connectivity."
  for i in $(seq 1 20) ; do
    ping -q -w 10 google.com && return 0
    status "Waiting for connectivity to google.com."
    sleep $((i**2))
  done
  fail "Failed to connect to google.com."
}

# Check Google services.
function check_google_services {
  if [[ -f /usr/bin/google_guest_agent ]]; then
    # Services expected when using the new guest environment (NGE).
    assert_running google-guest-agent
  else
     # Services expected when using the legacy guest environment.
    assert_running google-accounts-daemon
    assert_running google-network-daemon
    assert_running google-clock-skew-daemon
  fi

  if [[ $OSCONFIG_NOT_SUPPORTED =~ "true" ]]; then
    status "Skipping check for google-osconfig-agent, since it's not supported for this distro."
    return 0
  fi
  assert_running google-osconfig-agent
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

    # Skip for Debian 8, (Cloud SDK requires Python 3.5+)
  if [ -f /etc/os-release ]; then
    grep -qi "jessie" /etc/os-release
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

# Check package installs. Using iputils to ensure
# ping is available for the check_metadata_connectivity test.
function check_package_install {

  # The loop ensures that we allow the package manager to become
  # available; some systems perform updates on boot, which locks
  # the package manager.
  for i in $(seq 1 20); do
    # Apt
    if [[ -d /etc/apt ]]; then
      status "Checking if apt can install a package."
      apt-get update && apt-get install --reinstall iputils-ping && return 0
    fi

    # Yum
    if [[ -d /etc/yum ]]; then
      status "Checking if yum can install a package."
      if rpm -q iputils; then
        yum -y update iputils
      else
        yum -y install iputils
      fi
      yum -y reinstall iputils && return 0
    fi

    # Zypper
    if [[ -d /etc/zypp ]]; then
      status "Checking if zypper can install a package."
      zypper install -f -y iputils && return 0
    fi

    status "Waiting for package manager to be available"
    sleep $((i ** 2))
  done
  fail "cannot install iputils."
}

function check_data_disk_content {
  if [[ $(< /mnt/b/test_sdb.txt) != "on_sdb" ]]; then
      fail "/mnt/b/test_sdb.txt not found or content doesn't match."
  fi
  if [[ $(< /mnt/c/test_sdc.txt) != "on_sdc" ]]; then
      fail "/mnt/c/test_sdc.txt not found or content doesn't match."
  fi
}

# password_is_locked returns 0 if the user's password is locked,
# or 1 if it's unlocked. It's considered locked if /etc/shadow
# contains a hash of `*`, which means that the password is
# unusable by the system; the `*` technique is sometimes used
# as an idiom for locking the password.
function password_is_locked {
  local user=$1
  # Looping mitigates a race with the guest agent which adds users
  # at startup. The race occurs if `passwd` knows about the user,
  # but `/etc/shadow` hasn't been updated yet.
  for i in {1..5}; do
    if passwd -S "$user" | grep -qP "$user LK?"; then
      return 0
    elif grep -q "$user:\*:" /etc/shadow; then
      return 0
    fi
    sleep $((i ** 2))
  done
  return 1
}

# Check that the root account and all non-system accounts have their password locked.
# The locking doesn't occur during translation, but is rather an invariant we want
# to enforce in the test images.
#
# If there's a test failure, perform the following steps:
#  1. Export the image using gcloud:
#      gcloud compute images export \
#         --destination-uri=gs://bucket/image.qcow2 \
#         --image=image-name \
#         --export-format=qcow2
#  2. Download the image from GCS.
#  3  Reset the root password:
#       virt-customize -a image.qcow2 --root-password password:a-temp-password
#  4. Launch the VM:
#       qemu-system-x86_64 -enable-kvm -m 4G -hda image.qcow2
#  5. Login to root using the password from (3)
#  6. Lock the password:
#       passwd -l root
#  7. Shutdown the VM, then launch again to verify that
#     you cannot login to root using the temporary password.
#  8. Re-import the image as a data disk.
function check_passwords_locked {
  for user in root $(awk -F: '($1 != "nobody") && ($3 >= 1000){print $1}' /etc/passwd); do
    if password_is_locked "$user"; then
      status "<$user> password locked"
    else
      fail "<$user> password not locked"
    fi
  done
}

# Ensure there's network connectivity before running the tests.
wait_for_connectivity

# Ensure test images were created with locked passwords.
check_passwords_locked

# Run tests.
check_google_services
check_google_cloud_sdk
check_cloud_init
check_package_install
check_metadata_connectivity
check_internet_connectivity
check_data_disk_content

# Return results.
if [[ ${FAIL} -eq 0 ]]; then
  echo "PASSED: All tests passed!"
else
  echo "FAILED: ${FAIL} tests failed. ${FAILURES}"
fi
