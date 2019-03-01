#!/bin/bash

METADATA_URL="http://metadata/computeMetadata/v1/instance"

MAX_SIZE=$1
BUFFER_SIZE=25
INTERVAL=10

echo "Max disk size ${MAX_SIZE}GB, buffer size ${BUFFER_SIZE}GB, starting monitoring available disk buffer every ${INTERVAL}s..."
while [ 1 ]
do
  sleep ${INTERVAL}

  # Check whether available buffer space is lower than threshold.
  AVAILABLE_BUFFER=$(df -BG /dev/sdc --output=avail | sed 1d)
  AVAILABLE_BUFFER=${AVAILABLE_BUFFER%?}
  if [ ${AVAILABLE_BUFFER} -ge ${BUFFER_SIZE} ]; then
    continue
  fi

  # Decide the new size, which won't be higher than max size.
  CURRENT_SIZE=$(lsblk /dev/sdc --output=SIZE | sed -n 2p)
  CURRENT_SIZE=${CURRENT_SIZE%?}
  NEXT_SIZE=$(awk "BEGIN {print int(${CURRENT_SIZE} + ${BUFFER_SIZE})}")
  if [ ${NEXT_SIZE} -gt ${MAX_SIZE} ]; then
    NEXT_SIZE=${MAX_SIZE}
  fi

  # Prepare parameters for resizing
  ZONE=$(curl "${METADATA_URL}/zone" -H "Metadata-Flavor: Google"| cut -d'/' -f4)
  if [ $? -ne 0 ]; then
    echo "GCEExport: Failed to get metadata from ${METADATA_URL}/zone, will try again later."
    continue
  fi
  INSTANCE_NAME_PREFIX=$(curl "${METADATA_URL}/attributes/instance-name" -H "Metadata-Flavor: Google")
  if [ $? -ne 0 ]; then
    echo "GCEExport: Failed to get metadata from ${METADATA_URL}/attributes/instance-name, will try again later."
    continue
  fi
  HOST_NAME=$(curl "${METADATA_URL}/hostname" -H "Metadata-Flavor: Google"| cut -d'.' -f1)
  if [ $? -ne 0 ]; then
    echo "GCEExport: Failed to get metadata from ${METADATA_URL}/hostname, will try again later."
    continue
  fi
  BUFFER_DISK_PREFIX=$(curl "${METADATA_URL}/attributes/buffer-disk" -H "Metadata-Flavor: Google")
  if [ $? -ne 0 ]; then
    echo "GCEExport: Failed to get metadata from ${METADATA_URL}/attributes/buffer-disk, will try again later."
    continue
  fi
  BUFFER_DISK=${BUFFER_DISK_PREFIX}-$(echo ${HOST_NAME} | awk -F"${INSTANCE_NAME_PREFIX}-" '{ print $2}')

  echo "GCEExport: Resizing buffer disk from ${CURRENT_SIZE}GB to ${NEXT_SIZE}GB..."
  if ! out=$(gcloud compute disks resize ${BUFFER_DISK} --size=${NEXT_SIZE}GB --quiet --zone "${ZONE}" 2>&1); then
    echo "ExportFailed: Failed to resize buffer disk. Error: ${out}"
    continue
  fi
  echo ${out}
  if ! out=$(sudo resize2fs /dev/sdc 2>&1); then
    echo "ExportFailed: Failed to resize partition of buffer disk. Error: ${out}"
    continue
  fi
  echo ${out}

  if [ ${NEXT_SIZE} -eq ${MAX_SIZE} ]; then
    echo "Buffer disk reaches max size."
    exit
  fi
done
