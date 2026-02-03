#!/usr/bin/env python3
# Copyright 2026 Google Inc. All Rights Reserved.
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

import json
import logging
import os

RHEL_MAJOR_VERSIONS = ["8", "9", "10"]
RHEL_MINOR_VERSIONS = {
    "8": ["8.6", "8.8", "8.10"],
    "9": ["9.0", "9.2", "9.4", "9.6"],
    "10": ["10.0"],
}

RHEL_EUS_VERSIONS = ["9.4", "9.6", "10.0"]
RHEL_LVM_VERSIONS = ["8", "9", "10"]
RHEL_SAP_VERSIONS = ["8.6", "8.8", "8.10", "9.0", "9.2", "9.4", "9.6"]

ARCHITECTURES = ["x86_64", "arm64"]
PLANS = ["payg", "byos"]


def get_licenses(major_version, plan, is_eus, is_lvm, is_sap):
    licenses = []
    if is_sap and plan == "byos":
        licenses.append(
            "projects/rhel-sap-cloud/global/licenses/rhel-"
            f"{major_version}-sap-byos"
        )
    elif is_sap:
        licenses.append(
            "projects/rhel-sap-cloud/global/licenses/rhel-"
            f"{major_version}-sap"
        )
    elif plan == "byos":
        licenses.append(
            "projects/rhel-cloud/global/licenses/rhel-"
            f"{major_version}-byos"
        )
    else:
        licenses.append(
            "projects/rhel-cloud/global/licenses/rhel-"
            f"{major_version}-server"
        )

    if is_eus:
        licenses.append(
            "projects/rhel-cloud/global/licenses/rhel-"
            f"{major_version}-server-eus"
        )
    if is_lvm:
        licenses.append("projects/rhel-cloud/global/licenses/rhel-lvm")
    return licenses


def get_guest_os_features(major_version, arch, is_sap, minor_version):
    if arch == "arm64":
        return ["UEFI_COMPATIBLE", "GVNIC", "IDPF"]
    if major_version == 10:  # RHEL 10 x86_64 images
        return [
            "UEFI_COMPATIBLE",
            "VIRTIO_SCSI_MULTIQUEUE",
            "SEV_CAPABLE",
            "SEV_SNP_CAPABLE",
            "SEV_LIVE_MIGRATABLE",
            "SEV_LIVE_MIGRATABLE_V2",
            "GVNIC",
            "IDPF",
            "TDX_CAPABLE"
        ]
    elif major_version == 9:  # RHEL 9 x86_64 images
        if is_sap and minor_version and minor_version == "9.0":
            return [
                "UEFI_COMPATIBLE",
                "VIRTIO_SCSI_MULTIQUEUE",
                "SEV_CAPABLE",
                "GVNIC"
            ]
        elif is_sap and minor_version and minor_version == "9.2":
            return [
                "UEFI_COMPATIBLE",
                "VIRTIO_SCSI_MULTIQUEUE",
                "SEV_CAPABLE",
                "SEV_SNP_CAPABLE",
                "GVNIC",
                "SEV_LIVE_MIGRATABLE",
                "SEV_LIVE_MIGRATABLE_V2"
            ]
        else:
            return [
                "UEFI_COMPATIBLE",
                "VIRTIO_SCSI_MULTIQUEUE",
                "SEV_CAPABLE",
                "SEV_SNP_CAPABLE",
                "SEV_LIVE_MIGRATABLE",
                "SEV_LIVE_MIGRATABLE_V2",
                "GVNIC",
                "IDPF",
                "TDX_CAPABLE"
            ]
    else:  # RHEL 8 x86_64 images
       if is_sap:
           if minor_version and minor_version == "8.6":
               return [
                   "VIRTIO_SCSI_MULTIQUEUE",
                   "UEFI_COMPATIBLE",
                   "SEV_CAPABLE",
                   "GVNIC"
                ]
           elif minor_version and minor_version == "8.8":
               return [
                   "VIRTIO_SCSI_MULTIQUEUE",
                   "UEFI_COMPATIBLE",
                   "SEV_CAPABLE",
                   "GVNIC",
                   "SEV_LIVE_MIGRATABLE_V2"
                ]
           else:  # RHEL 8.10 SAP
               return [
                   "VIRTIO_SCSI_MULTIQUEUE",
                   "UEFI_COMPATIBLE",
                   "SEV_CAPABLE",
                   "SEV_LIVE_MIGRATABLE",
                   "SEV_LIVE_MIGRATABLE_V2",
                   "GVNIC",
                   "IDPF"
                ]
       else:
           return [
               "UEFI_COMPATIBLE",
               "VIRTIO_SCSI_MULTIQUEUE",
               "SEV_CAPABLE",
               "SEV_SNP_CAPABLE",
               "SEV_LIVE_MIGRATABLE",
               "SEV_LIVE_MIGRATABLE_V2",
               "GVNIC",
               "IDPF"
            ]


def generate_workflow_file(image_name,
                           major_version,
                           licenses,
                           description,
                           guest_os_features):
    workflow_name = f"build-{image_name}"

    wf = {
        "_WARNING": (
            "This file is Generated. Do not edit manually. Modify the "
            "generator script (write_image_build_workflow.py) instead."
        ),
        "Name": workflow_name,
        "Vars": {
            "google_cloud_repo": {
               "Value": "stable",
               "Description": "The Google Cloud Repo branch to use."
            },
            "installer_iso": {
               "Required": True,
               "Description": (
                   f"The RHEL {major_version} installer ISO to build from."
               )
            },
            "build_date": {
               "Value": "${TIMESTAMP}",
               "Description": "Build datestamp used to version the image."
            },
            "publish_project": {
               "Value": "${PROJECT}",
               "Description": "A project to publish the resulting image to."
            },
            "el_release": {
                "Value": f"rhel-{major_version}",
                "Description": (
                    "The Enterprise Linux (EL) release for the image"
                )
            },
            "is_arm": {
                "Value": "false",
                "Description": "The system architecture of X86 or ARM"
            },
            "is_byos": {
                "Value": "false",
                "Description": (
                    "If the image is Pay as you Go (PAYG) "
                    "or Bring your own Service (BYOS)"
                )
            },
            "is_eus": {
                "Value": "false",
                "Description": "If the image has Extended Update Support(EUS)"
            },
            "is_lvm": {
                "Value": "false",
                "Description": (
                    "If the image supports the Logical Volume Manager (LVM)"
                )
            },
            "is_sap": {
                "Value": "false",
                "Description": "If the image is RHEL for SAP"
            },
            "rhui_package_name": {
                "Required": True,
                "Description": "Name of the RHUI client package"
            },
            "use_dynamic_template": {
                "Value": "false",
                "Description": (
                    "Use the dynamically created templates to create images "
                    "as part of the RHEL Build Workflow Consolidation path. "
                    "To remove once the consolidation/refactoring is complete"
                )
            },
            "version_lock": {
                "Value": "",
                "Description": "The minor release version for RHEL EUS images"
            }
        },
        "Steps": {
           "build-rhel": {
               "Timeout": "60m",
               "IncludeWorkflow": {
                   "Path": f"./rhel_{major_version}_consolidated.wf.json",
                   "Vars": {
                       "el_release": "${el_release}",
                       "kickstart_config": (
                           f"./kickstart/rhel_{major_version}"
                           "_consolidated.cfg"
                        ),
                       "google_cloud_repo": "${google_cloud_repo}",
                       "installer_iso": "${installer_iso}",
                       "rhel_byos": "${is_byos}",
                       "rhel_sap": "${is_sap}",
                       "is_arm": "${is_arm}",
                       "is_eus": "${is_eus}",
                       "is_lvm": "${is_lvm}",
                       "rhui_package_name": "${rhui_package_name}",
                       "use_dynamic_template": "${use_dynamic_template}",
                       "version_lock": "${version_lock}"
                    }
                }
            },
           "create-image": {
               "CreateImages": [
                   {
                       "Name": "${el_release}-v${{build_date}}",
                       "SourceDisk": "el-install-disk",
                       "Licenses": licenses,
                       "Description": description,
                       "Family": image_name,
                       "GuestOsFeatures": guest_os_features,
                       "Project": "${publish_project}",
                       "NoCleanup": True,
                       "ExactName": True
                   }
                ]
            }
        },
        "Dependencies": {
           "create-image": ["build-rhel"]
        }
    }
    return wf


def write_workflow_file(major_version,
                        plan,
                        is_eus,
                        is_lvm,
                        is_sap,
                        arch,
                        minor_version):
    image_name = "rhel-"
    if minor_version:
        image_name += minor_version.replace('.', '-')
    else:
        image_name += major_version
    if is_eus:
       image_name += "-eus"
    if is_sap:
       image_name += "-sap"
    if plan == "byos":
       image_name += "-byos"
    if arch == "arm64":
       image_name += "-arm64"

    description = "Red Hat Enterprise Linux"
    if is_sap:
        description += " for SAP"
    description += ", "
    if minor_version:
        description += minor_version.replace('.', '-')
    else:
        description += major_version
    if is_eus:
        description += " EUS"
    if is_lvm:
        description += " (LVM)"
    description += ", "
    if plan == "byos":
        description += " BYOS"
    description += arch
    if is_lvm:
        description += " with a LVM boot volume"

    licenses = get_licenses(major_version, plan, is_eus, is_lvm, is_sap)
    guest_os_features = get_guest_os_features(major_version,
                                              arch,
                                              is_sap,
                                              minor_version)
    wf = generate_workflow_file(image_name,
                                major_version,
                                licenses,
                                description,
                                guest_os_features)
    script_dir = os.path.dirname(os.path.abspath(__file__))
    image_name = image_name.replace('-', '_')
    file_name = os.path.join(script_dir, f"{image_name}.wf.json")
    with open(file_name, 'w') as f:
        json.dump(wf, f, indent=2)
    logging.info(f'Wrote workflow file: {file_name}')


def main():
    is_eus = False
    is_lvm = False
    is_sap = False

    for arch in ARCHITECTURES:
        for plan in PLANS:
            for major_version in RHEL_MAJOR_VERSIONS:
                # LVM is only for PAYG images
                # LVM is currently only avaliable for major versions
                if (plan == "payg"
                    and major_version in RHEL_LVM_VERSIONS):
                    is_lvm = True
                    write_workflow_file(major_version,
                                        plan,
                                        is_eus,
                                        is_lvm,
                                        is_sap,
                                        arch,
                                        '')  # LVM
                is_lvm = False
                write_workflow_file(major_version,
                                    plan,
                                    is_eus,
                                    is_lvm,
                                    is_sap,
                                    arch,
                                    '')  # Base image
                for minor_version in RHEL_MINOR_VERSIONS[major_version]:
                    if minor_version not in RHEL_EUS_VERSIONS and minor_version not in RHEL_SAP_VERSIONS:
                        continue
                    if minor_version in RHEL_EUS_VERSIONS:
                        is_eus = True
                        write_workflow_file(major_version,
                                            plan,
                                            is_eus,
                                            is_lvm,
                                            is_sap,
                                            arch,
                                            minor_version)  # EUS
                    is_eus = False
                    # SAP only supports x86_64
                    if (arch == "x86_64"
                        and minor_version in RHEL_SAP_VERSIONS):
                        is_sap = True
                        write_workflow_file(major_version,
                                            plan,
                                            is_eus,
                                            is_lvm,
                                            is_sap,
                                            arch,
                                            minor_version)  # SAP
                    is_sap = False


if __name__ == '__main__':
  try:
    main()
    logging.info('Daisy image_build workflow files written successful!')
  except Exception as e:
    logging.error(
        'Writing Daisy image_build workflow files failed: %s' % str(e))
