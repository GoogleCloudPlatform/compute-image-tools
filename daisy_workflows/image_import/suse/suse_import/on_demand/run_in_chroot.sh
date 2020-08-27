#!/usr/bin/env bash
# Copyright 2020 Google Inc. All Rights Reserved.
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

# Enables GCE on-demand for a SLES instance. This occurs in three phases:
# 1. Manually install dependencies required for registration. These are cached as files
#    to allow conversion of instances that don't have an active SLES subscription.
# 2. Register with SLES's GCP SCC servers
# 3. Use the on-demand subscription to re-install the packages from step 1 using zypper

# Interface:
#
# Prior to invoking this, setup a directory that contains the following:
#
#  runtime_dir
#  ├── cloud_product.txt
#  ├── post_convert_packages.txt
#  ├── run_in_chroot.sh
#  ├── pre_convert_py
#  │   └── python_package.wheel
#  └── pre_convert_rpm
#      └── package.rpm
#
# cloud_product.txt
#   The name of the cloud product to install, eg: sle-module-public-cloud/12/x86_64
#
# post_convert_packages.txt
#   One zypper package per line. Installed after conversion to on-demand.
#
# run_in_chroot.sh
#   This script
#
# pre_convert_py/
#   Directory of python dependencies required for offline conversion.
#   The `--root` parameter of pip3 will create this, eg: pip3 install <pkg> --root=py/
#   This is version specific, and allows for providing dependencies that are not available
#   in the KB tarball from SLES.
#
# pre_convert_rpm/
#   Directory of RPM dependencies required for offline conversion.
#   The RPMs are provided by SLES as a tarball: https://www.suse.com/support/kb/doc/?id=000019633
set -eux

function setup_env() {
  cd "$(dirname "$0")"
  if [[ -f cloud_product.txt ]]; then
    CLOUD_PRODUCT=$(cat cloud_product.txt)
  else
    echo "cloud_product.txt: File not found"
    return 1
  fi

  if [[ -f post_convert_packages.txt ]]; then
    POST_CONVERT_PACKAGES=$(cat post_convert_packages.txt)
  else
    echo "post_convert_packages.txt: File not found"
    return 1
  fi

  if [[ -d pre_convert_py ]]; then
    PYTHONPATH=$(realpath pre_convert_py)
    export PYTHONPATH
  else
    echo "pre_convert_py: Directory not found"
    return 1
  fi

  if [[ -d pre_convert_rpm ]]; then
    PRE_CONVERT_RPM_DIR=$(realpath pre_convert_rpm)
  else
    echo "pre_convert_rpm: Directory not found"
    return 1
  fi

  # REQUESTS_CA_BUNDLE is used by the 'requests' Python package. Without
  # the variable, 'requests' uses its own cert store. The store doesn't
  # contain SLES's custom SSL certs.
  export REQUESTS_CA_BUNDLE=/var/lib/ca-certificates/ca-bundle.pem
}

# Install RPMs hosted on https://www.suse.com/support/kb/doc/?id=000019633
function manual_package_install() {
  for r in "$PRE_CONVERT_RPM_DIR"/*.rpm; do
    echo "Installing $r"
    set +e
    # `--nodeps` is used since https://www.suse.com/support/kb/doc/?id=000019633
    # is missing some dependencies. Examples of missing dependencies:
    # 1. pcitools, which is modeled as a required dependency, but is considered
    # optional in the runtime of their code. (It looks for GPUs during the sniffing
    # phase of registration. Sniff runs on every boot, and pcitools is installed once
    # registration finishes.)
    # 2. Python requests, which is used for making HTTP calls, but is not included
    # in the linked tarballs.
    #
    # After registration, the dependencies are installed
    #
    # As a side effect of 'nodeps',
    rpm -i --nodeps "$r"
    set -e
  done
}

# Register the VM against the SLES GCE SCC servers.
function register_as_on_demand() {
  /usr/sbin/registercloudguest --force-new
  SUSEConnect --debug -p $CLOUD_PRODUCT
  zypper --debug refresh
}

# Ensure the on-demand packages are fully installed with Zypper.
function install_with_zypper() {
  for r in "$PRE_CONVERT_RPM_DIR"/*.rpm; do
    rpm -q --package "$r" --qf "%{NAME}\n" >>to_update
  done
  zypper --debug --non-interactive install -f --no-recommends \
    $(cat to_update) $POST_CONVERT_PACKAGES
  zypper --debug refresh
}

function emit_logs() {
  logfiles=( "/var/log/zypper.log" "/var/log/cloudregister" "/var/log/suseconnect.log" )

  for logfile in "${logfiles[@]}"; do
    echo "[$logfile]"
    if [[ -f logfile ]]; then
      cat "$logfile"
    else
      echo "$logfile: file not found"
    fi
  done

}

trap 'emit_logs' EXIT

setup_env && manual_package_install && register_as_on_demand && install_with_zypper
