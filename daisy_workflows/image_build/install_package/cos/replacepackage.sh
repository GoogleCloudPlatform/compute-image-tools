#!/bin/bash
#
# This script is responsible for invoking the preloading mechanism that creates
# a COS image with the new package version that is used for testing.
#   1) The script sets all the preloading scripts as executable (located in
#   /cos/package_replacement).
#   2) Then, collects all passed in data through metadata API (name of source
#   image, dest image, commit sha, cos branch).
#     - cos_branch: the name of the branch associated with a COS milestone (ie
#     for cos-113, cos_branch=release-R113).
#   3) Runs a gcloud submit command to invoke the cloudbuild responsible for
#   preloading.

set -o errexit
set -o pipefail
set -x

set_files_executable(){
  cd /files/package_replacement

  search_dir=.
  for file in "$search_dir"/*
  do
    chmod +x $file
  done
}

get_vars(){
  export SOURCE_IMAGE=$(curl -H "Metadata-Flavor:Google" http://metadata.google.internal/computeMetadata/v1/instance/attributes/source_image)
  export DEST_IMAGE=$(curl -H "Metadata-Flavor:Google" http://metadata.google.internal/computeMetadata/v1/instance/attributes/dest_image)
  export COMMIT_SHA=$(curl -H "Metadata-Flavor:Google" http://metadata.google.internal/computeMetadata/v1/instance/attributes/commit_sha)
  export DAISY_LOGS_PATH=$(curl -H "Metadata-Flavor:Google" http://metadata.google.internal/computeMetadata/v1/instance/attributes/daisy-logs-path)
}

compile_debian_binaries(){
  logger -p daemon.info "Attempting to update apt package lists..."
  sudo apt update 2>&1 | logger -t apt-update -p daemon.info
  logger -p daemon.info "Attempting to install git..."
  sudo apt install -y git 2>&1 | logger -t apt-install -p daemon.info
  ./compile_debian_package.sh ${COMMIT_SHA} ${ARCH}
}

set_machine_and_arch_type(){
  export MACHINE_TYPE="n1-standard-1"
  export ARCH="amd"
  if [[ "$SOURCE_IMAGE" == *"arm"* ]]; then
      export MACHINE_TYPE="t2a-standard-1"
      export ARCH="arm"
  fi
}
# The following additional parameters are passed into the cloudbuild script:
#   _BASE_IMAGE_PROJECT: The project that contains the base/source image.
#   _BASE_IMAGE: The name of the base/source image.
#   _DEST_PROJECT: The project that contains the resulting preloaded image.
#   _NEW_IMAGE: The name of the resulting preloaded image.
#   _NEW_IMAGE_FAMILY: The new image family for the preloaded image.
#   _MACHINE_TYPE: The machine type for the source image: default (n1-standard-1) or t2a-standard-1 for ARM.
create_preloaded_image(){
  logger -p daemon.info "Attempting to submit gcloud build..."
  gcloud builds submit . --config=dev_cloudbuild.yaml --disk-size=200 --gcs-log-dir="${DAISY_LOGS_PATH}" --substitutions=_NEW_IMAGE_FAMILY="cos-preloaded-images",_BASE_IMAGE_PROJECT="cos-cloud",_BASE_IMAGE="${SOURCE_IMAGE}",_NEW_IMAGE="${DEST_IMAGE}",_DEST_PROJECT="cloud-kernel-build",_MACHINE_TYPE="${MACHINE_TYPE}" 2>&1 | logger -t gcloud-build -p daemon.info
  logger -p daemon.info "Finished attempting to submit gcloud build."
}

main (){
  logger -p daemon.info "Creating COS image with the new guest agent version..."
  set_files_executable
  get_vars
  set_machine_and_arch_type
  compile_debian_binaries
  create_preloaded_image
}

main
