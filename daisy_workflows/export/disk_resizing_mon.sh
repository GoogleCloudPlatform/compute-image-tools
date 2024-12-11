#!/bin/bash

BYTES_1GB=1073741824
METADATA_URL="http://metadata/computeMetadata/v1/instance"

MAX_SIZE=$1
BUFFER_MIN_SIZE=10
BUFFER_SIZE=25
INTERVAL=10

# Prepare parameters for resizing
ZONE=$(curl "${METADATA_URL}/zone" -H "Metadata-Flavor: Google"| cut -d'/' -f4)
BUFFER_DISK_NAME=$(curl "${METADATA_URL}/attributes/buffer-disk-name" -H "Metadata-Flavor: Google")
BUFFER_DISK=$(readlink -f /dev/disk/by-id/google-${BUFFER_DISK_NAME})

echo "Max disk size ${MAX_SIZE}GB, min buffer size ${BUFFER_SIZE}GB, starting monitoring available disk buffer every ${INTERVAL}s..."
while sleep ${INTERVAL}; do
  # Check whether available buffer space is lower than threshold.
  AVAILABLE_BUFFER=$(df -BG ${BUFFER_DISK} --output=avail | sed -n 2p)
  AVAILABLE_BUFFER=${AVAILABLE_BUFFER%?}
  if [[ ${AVAILABLE_BUFFER} -ge ${BUFFER_MIN_SIZE} ]]; then
    continue
  fi

  # Decide the new size of the device.
  CURRENT_DEVICE_SIZE_BYTES=$(lsblk ${BUFFER_DISK} --output=size -b | sed -n 2p)
  CURRENT_DEVICE_SIZE=$(awk "BEGIN {print int(((${CURRENT_DEVICE_SIZE_BYTES}-1)/${BYTES_1GB}) + 1)}")
  NEXT_SIZE=$(awk "BEGIN {print int(${CURRENT_DEVICE_SIZE} + ${BUFFER_SIZE})}")

  echo "GCEExport: Resizing buffer disk from ${CURRENT_DEVICE_SIZE}GB to ${NEXT_SIZE}GB..."
  if ! out=$(gcloud compute disks resize ${BUFFER_DISK_NAME} --size=${NEXT_SIZE}GB --quiet --zone "${ZONE}" 2>&1); then
    echo "ExportFailed: Failed to resize buffer disk. [Privacy-> Error: ${out} <-Privacy]"
    continue
  fi
  echo ${out}
  if ! out=$(sudo resize2fs ${BUFFER_DISK} 2>&1); then
    echo "ExportFailed: Failed to resize partition of buffer disk. [Privacy-> Error: ${out} <-Privacy]"
    continue
  fi
  echo ${out}

  # If current file system has reached or exceeded max size, then stop resizing.
  # We need to know the size of the available file system other than the size of
  # the partition, so "df" is used instead of "lsblk" here.
  CURRENT_FILESYSTEM_SIZE=$(df -BG ${BUFFER_DISK} --output=size | sed -n 2p)
  CURRENT_FILESYSTEM_SIZE=${CURRENT_FILESYSTEM_SIZE%?}
  if [[ ${CURRENT_FILESYSTEM_SIZE} -ge ${MAX_SIZE} ]]; then
    echo "Buffer disk reaches max size."
    exit
  fi
done
