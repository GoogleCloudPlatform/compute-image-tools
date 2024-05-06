#!/bin/bash
#
# This script is responsible for invoking the preloading mechanism that creates
# a COS image with the new package version that is used for testing.
#   1) The script sets all the preloading scripts as executable (located in
#   /cos/package_replacement).
#   2) Then, collects all passed in data through metadata API (name of source
#   image, dest image, package version, cos branch).
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
  export PACKAGE_VERSION=$(curl -H "Metadata-Flavor:Google" http://metadata.google.internal/computeMetadata/v1/instance/attributes/package_version)
  export COS_BRANCH=$(curl -H "Metadata-Flavor:Google" http://metadata.google.internal/computeMetadata/v1/instance/attributes/cos_branch)
}

# The following additional parameters are passed into the cloudbuild script:
#   _BASE_IMAGE_PROJECT: The project that contains the base/source image.
#   _BASE_IMAGE: The name of the base/source image.
#   _DEST_PROJECT: The project that contains the resulting preloaded image.
#   _NEW_IMAGE: The name of the resulting preloaded image.
#   _NEW_IMAGE_FAMILY: The new image family for the preloaded image.
create_preloaded_image(){
  gcloud builds submit --config=dev_cloudbuild.yaml --disk-size=200 . --substitutions=_NEW_IMAGE_FAMILY="cos-preloaded-images",_BASE_IMAGE_PROJECT="cos-cloud",_BASE_IMAGE="${SOURCE_IMAGE}",_OVERLAYS_BRANCH="${COS_BRANCH}",_GUEST_AGENT_VERSION="${PACKAGE_VERSION}",_NEW_IMAGE="${DEST_IMAGE}",_DEST_PROJECT="gcp-guest"
}

main (){
  echo "Creating COS image with the new guest agent version..."
  set_files_executable
  get_vars
  create_preloaded_image
}

main
