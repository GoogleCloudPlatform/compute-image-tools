#!/bin/bash
set -x

URL="http://metadata/computeMetadata/v1/instance/attributes"
SOURCE_URL=$(curl -f -H Metadata-Flavor:Google ${URL}/source-url)
SOURCE_BUCKET=$(echo ${SOURCE_URL} | awk -F/ '{print $3}')
SOURCE_PATH=${SOURCE_URL#"gs://"}
IMAGENAME=$(curl -f -H Metadata-Flavor:Google ${URL}/image-name)
ME=$(curl -f -H Metadata-Flavor:Google ${URL}/vm-name)
WF=$(curl -f -H Metadata-Flavor:Google ${URL}/workflow-name)
WFID=$(curl -f -H Metadata-Flavor:Google ${URL}/workflow-id)
ZONE=$(curl -f -H Metadata-Flavor:Google ${URL}/zone)

# Print info.
echo "#################"
echo "# Configuration #"
echo "#################"
echo "SOURCE_URL: ${SOURCE_URL}"
echo "SOURCE_BUCKET: ${SOURCE_BUCKET}"
echo "SOURCE_PATH: ${SOURCE_PATH}"
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
mkdir -p /gcs/${SOURCE_BUCKET}
gcsfuse --implicit-dirs ${SOURCE_BUCKET} /gcs/${SOURCE_BUCKET}

# Image size info.
SIZEGB=$(qemu-img info /gcs/${SOURCE_PATH} | perl -n -e'/^virtual size: (\d+)G/ && print $1')
SIZEGB=$(( (${SIZEGB} / 10 + 1) * 10 )) # Round up to next 10 GB increment (with a little extra space for rounding issues/overhead).

# Image disk.
IMAGEDISK=imagedisk-${WF}-${WFID}
gcloud compute disks create ${IMAGEDISK} --size=${SIZEGB}GB --zone=${ZONE} --type=pd-ssd
gcloud compute instances attach-disk ${ME} --disk=${IMAGEDISK} --zone=${ZONE}

# Write image to disk.
qemu-img convert /gcs/${SOURCE_PATH} -O raw -S 512b /dev/sdb

# Create image.
gcloud compute instances detach-disk ${ME} --disk=${IMAGEDISK} --zone=${ZONE}
gcloud compute images create ${IMAGENAME} --source-disk=${IMAGEDISK} --source-disk-zone=${ZONE}

# Delete image disk.
gcloud compute disks delete ${IMAGEDISK} --zone=${ZONE}

shutdown
