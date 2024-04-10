#!/bin/bash

set -o errexit
set -o pipefail
set -x

replace_files(){
  # Remove the guest agent repo from the cos customizer folder: this is important to make sure that replacement algorithms don't use this folder to perform replacement.
  sudo rm -rf /var/lib/.cos-customizer/user/guest-agent

  # Move all files to a temp folder under /.
  sudo mkdir /temp_debian_upload
  sudo mv upload /temp_debian_upload

  # Start guest agent replacement...
  file="repl_files.txt"

  # Go through every line in the deb file...
  while read -r line; do

    # For every line...
    read -ra arr <<< "$line"
    echo "${arr}"

    # Then this is a file we want to replace. Store the file name eg "google-guest-agent.service".
    file_name=${arr##*/}

    echo $file_name

    # Try to find the file and determine the installation path (will be empty if file not found) eg "/usr/bin/".
    INSTALLATION_PATH=$(sudo find / -type f -name ${file_name} -not -path "/temp_debian_upload/*" | awk -F${file_name} '{print $1}')

    # If the file is found (results are not empty), then begin the replacement.
    if ! [[ -z "$INSTALLATION_PATH" ]]; then

      # Determine the deb location path.
      path="${arr:1}"
      dest="${path%/*}"

      # Remove the pre-installed file.
      sudo rm -rf "${INSTALLATION_PATH}${file_name}"

      # Move the deb file to the correct installation path.
      sudo mv /temp_debian_upload/upload$dest/$file_name $INSTALLATION_PATH
    fi
  done <$file

  # Delete tmp folder and other guest agent files.
  sudo rm -rf /temp_debian_upload
}

main() {
  echo "replace_files"
  replace_files
}

main
