#!/bin/sh
# Copyright 2018 Google Inc. All Rights Reserved.
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

set -xeu

GetMetadataAttribute() {
    attribute="$1"

    url="$metadata_url/v1/instance/attributes/$attribute"
    attribute_value=$(curl -H "Metadata-Flavor: Google" -X GET "$url")
}

DebianInstallGoogleApiPythonClient() {
    logger -p daemon.info "Status: Installing google-api-python-client"

    apt-get update && DEBIAN_FRONTEND=noninteractive apt-get -q -y install python3-pip
    pip3 install -U google-api-python-client google-cloud-storage
}

DebianInstallNetaddrPythonLibrary() {
    logger -p daemon.info "Status: Installing netaddr python module"

    apt-get update && DEBIAN_FRONTEND=noninteractive apt-get -q -y install python3-netaddr
}

GetAccessToken() {
    url="$metadata_url/v1/instance/service-accounts/default/token"
    response=$(curl -H "Metadata-Flavor: Google" -X GET "$url")

    access_token_value=$(echo "$response" | cut -d '"' -f 4)
    token_type_value=$(echo "$response" | grep -o '"token_type":".*"' | cut -d '"' -f 4)

    TOKEN="$token_type_value $access_token_value"
}

GetBucketContent() {
    bucket_name="$1"
    bucket_path="$2"
    token="$3"
    storage_url="https://www.googleapis.com/storage"

    url="$storage_url/v1/b/$bucket_name/o?prefix=$bucket_path"

    logger -p daemon.info "Status: Bucket listing with $bucket_path prefix: $url"
    response=$(curl -H "Metadata-Flavor: Google" -H "Authorization: $token" -X GET "$url")

    name_attributes=$(echo "$response" | tr "," "\\n" | grep name)
    bucket_content=$(echo "$name_attributes" | tr " " "\\n" | grep "$bucket_path" | sed -e 's/\"//g')
}

SaveBucketFile() {
    bucket_target="$1"
    bucket_file="$2"
    file_dest="$3"

    url="https://storage.googleapis.com/$bucket_target/$bucket_file"

    dir=$(dirname "$file_dest")
    [ ! -d "$dir" ] && mkdir -p "$dir"

    logger -p daemon.info "Status: Bucket save: $url => $file_dest"
    curl -H "Metadata-Flavor: Google" -H "Authorization: $token" -X GET "$url" -o "$file_dest"
}

CheckPython3Installation() {
    if ! [ -x "$(command -v python3)" ]; then
        if [ -e /etc/redhat-release ]; then
            yum install -y epel-release
            yum install -y python34
        fi
    fi
}

trap "NotifyExit" EXIT
NotifyExit() {
    if [ "$?" -eq 0 ]; then
        logger -p daemon.info "${prefix}Success: exits with return code 0"
    else
        logger -p daemon.info "${prefix}Failed: error: exits with return code $?"
    fi
}

logger -p daemon.info "Status: Starting bootstrap.sh"

metadata_url="http://metadata.google.internal/computeMetadata"
attribute_value=""

GetMetadataAttribute "prefix" && prefix="$attribute_value"
GetMetadataAttribute "files_gcs_dir" && files_gcs_dir="$attribute_value"
GetMetadataAttribute "script" && script="$attribute_value"

GetMetadataAttribute "debian_install_google_api_python_client"
[ "$attribute_value" = "yes" ] && DebianInstallGoogleApiPythonClient

GetMetadataAttribute "debian_install_netaddr_python_module"
[ "$attribute_value" = "yes" ] && DebianInstallNetaddrPythonLibrary

TOKEN="" && GetAccessToken

DIR="/files" && mkdir -p $DIR

path_stripped=$(echo $files_gcs_dir | sed -r 's#^gs://##')
IFS='/' read -r bucket bucket_dir <<EOF
$path_stripped
EOF

bucket_content="" && GetBucketContent "$bucket" "$bucket_dir" "$TOKEN"

for file in $bucket_content; do
    dest_filepath=$(echo $file | sed 's|'"$bucket_dir"'|'"$DIR"'|')
    SaveBucketFile "$bucket" "$file" "$dest_filepath"
done

path_script="$DIR/$script"

logger -p daemon.info "Status: Making script $path_script executable"
chmod +x "$path_script"

CheckPython3Installation

logger -p daemon.info "Status: Running $path_script"
"$path_script"
