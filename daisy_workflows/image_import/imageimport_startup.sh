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

set -x

URL="http://metadata/computeMetadata/v1/instance/attributes"
IMAGEURL=$(curl -f -H Metadata-Flavor:Google ${URL}/daisy-sources-path)/image
IMAGEBUCKET=$(echo ${IMAGEURL} | awk -F/ '{print $3}')
IMAGEPATH=${IMAGEURL#"gs://"}
IMAGENAME=$(curl -f -H Metadata-Flavor:Google ${URL}/image-name)
ME=$(curl -f -H Metadata-Flavor:Google ${URL}/vm-name)
WF=$(curl -f -H Metadata-Flavor:Google ${URL}/workflow-name)
WFID=$(curl -f -H Metadata-Flavor:Google ${URL}/workflow-id)
ZONE=$(curl -f -H Metadata-Flavor:Google ${URL}/zone)

# Print info.
echo "#################"
echo "# Configuration #"
echo "#################"
echo "IMAGEURL: ${IMAGEURL}"
echo "IMAGEBUCKET: ${IMAGEBUCKET}"
echo "IMAGEPATH: ${IMAGEPATH}"
echo "IMAGENAME: ${IMAGENAME}"
echo "ME: ${ME}"
echo "WF: ${WF}"
echo "WFID: ${WFID}"
echo "ZONE: ${ZONE}"

# Set up GCS fuse repo.
export GCSFUSE_REPO=gcsfuse-`lsb_release -c -s`
echo "deb http://packages.cloud.google.com/apt $GCSFUSE_REPO main" | tee /etc/apt/sources.list.d/gcsfuse.list
curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -

# Install tools.
apt-get update
apt-get -q -y install qemu-utils gcsfuse

# Mount GCS bucket containing the image.
mkdir -p /gcs/${IMAGEBUCKET}
gcsfuse --implicit-dirs ${IMAGEBUCKET} /gcs/${IMAGEBUCKET}

# Image size info.
SIZEGB=$(qemu-img info /gcs/${IMAGEPATH} | perl -n -e'/^virtual size: (\d+)G/ && print $1')
SIZEGB=$(( (${SIZEGB} / 10 + 1) * 10 )) # Round up to next 10 GB increment (with a little extra space for rounding issues/overhead).

# Image disk.
IMAGEDISK=imagedisk-${WF}-${WFID}
gcloud compute disks create ${IMAGEDISK} --size=${SIZEGB}GB --zone=${ZONE} --type=pd-ssd
gcloud compute instances attach-disk ${ME} --disk=${IMAGEDISK} --zone=${ZONE}

# Write image to disk.
qemu-img convert /gcs/${IMAGEPATH} -O raw -S 512b /dev/sdb

# Create image.
gcloud compute instances detach-disk ${ME} --disk=${IMAGEDISK} --zone=${ZONE}
gcloud compute images create ${IMAGENAME} --source-disk=${IMAGEDISK} --source-disk-zone=${ZONE}

# Delete image disk.
gcloud compute disks delete ${IMAGEDISK} --zone=${ZONE}

shutdown
