#!/bin/bash
#
# This script is responsible for performing the preloading (guest agent pkg
# replacement on COS images). It does the following:
#   1) Go through the repl_files.txt: contains list of files to be replaced...
#   2) Find the file name on the system, if it exists, replace the file at the
#   same lcoation.
#   3) Delete all temp folders/files that are no longer needed.
#   4) Enables logging in google-startup-scripts.service (needed for CIT tests).

set -o errexit
set -o pipefail
set -x

replace_files(){
  # Remove the guest agent repo from the cos customizer folder: this is important to make sure that replacement algorithms don't use this folder to perform replacement.
  sudo rm -rf /var/lib/.cos-customizer/user/guest-agent

  # Move all files to a temp folder under /.
  sudo mkdir /temp_debian_upload
  sudo mv debian_binaries /temp_debian_upload

  # Start guest agent replacement...
  file="repl_files.txt"
  logger -p daemon.info "repl_files.txt: $(cat $file)"

  # Go through every line in the deb file...
  while read -r line; do

    # For every line...
    read -ra arr <<< "$line"
    logger -p daemon.info "line: ${arr}"

    # Then this is a file we want to replace. Store the file name eg "google-guest-agent.service".
    file_name=${arr##*/}

    logger -p daemon.info $file_name

    # Try to find the file and determine the installation path (will be empty if file not found) eg "/usr/bin/".
    INSTALLATION_PATH=$(sudo find /usr /etc /opt /var /lib -type f -name ${file_name} | awk -F${file_name} '{print $1}')

    # If the file is found (results are not empty), then begin the replacement.
    if ! [[ -z "$INSTALLATION_PATH" ]]; then
      logger -p daemon.info "INSTALLATION_PATH found: $INSTALLATION_PATH${file_name}"
      # Determine the deb location path.
      path="${arr:1}"
      dest="${path%/*}"

      # Remove the pre-installed file.
      sudo rm -rf "${INSTALLATION_PATH}${file_name}"

      # If this is the startup script service file, enable logging so CIT tests
      # can exit successfully after 'finished-test' is written.
      if [[ "$file_name" == "google-startup-scripts.service" ]]; then
        logger -p daemon.info "Found google-startup-scripts.service, enabling logging"
        sudo sed -i '/KillMode=process/a StandardOutput=journal+console\nStandardError=journal+console' /temp_debian_upload/debian_binaries$dest/$file_name
      fi
      
      # Move the deb file to the correct installation path.
      sudo mv /temp_debian_upload/debian_binaries$dest/$file_name $INSTALLATION_PATH
    fi
  done <$file

  # Delete tmp folder and other guest agent files.
  sudo rm -rf /temp_debian_upload
}

main() {
  replace_files
}

main
