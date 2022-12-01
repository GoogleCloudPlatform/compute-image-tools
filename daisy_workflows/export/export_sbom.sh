#!/bin/sh
# SBOM export work is done in this script, regardless of linux distro

usage() {
  echo "Usage: export_sbom.sh -s syft_source_location -p path_to_export_sbom"
  echo "ExportFailed: incorrect SBOM script arguments"
  exit 1
}

unset -v SBOM_PATH
unset -v SYFT_SOURCE

while getopts p:s: opt; do
  case $opt in
    p) SBOM_PATH=$OPTARG ;;
    s) SYFT_SOURCE=$OPTARG ;;
    *) usage
  esac
done

if [ -z "$SBOM_PATH" ] || [ -z "$SYFT_SOURCE" ]; then
  usage
fi

gsutil cp $SYFT_SOURCE syft.tar.gz
tar -xf syft.tar.gz
./syft /mnt -o spdx-json > sbom.json
gsutil cp sbom.json $SBOM_PATH
