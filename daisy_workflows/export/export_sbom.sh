#!/bin/sh
# SBOM export work is done in this script, regardless of linux distro

usage() {
  echo "Usage: export_sbom.sh --syft-source path_to_syft_source --sbom-path path_to_export_sbom"
  echo "ExportFailed: incorrect SBOM script arguments"
  exit 1
}

if [ $# -ne 4 ]; then
  usage
fi

if [ $1 = "--syft-source" ] && [ $3 = "--sbom-path" ]; then
  echo "inside sbom"
  echo $SBOM_PATH
  SYFT_SOURCE=$2
  SBOM_PATH=$4
  gsutil cp ${SYFT_SOURCE} syft.tar.gz
  tar -xf syft.tar.gz
  ./syft /mnt -o spdx-json > sbom.json
  gsutil cp sbom.json ${SBOM_PATH}
else
  usage
fi
