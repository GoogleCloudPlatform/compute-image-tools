#!/bin/bash
#
# Disables auto updates on machine.

set -o errexit
set -o pipefail
set -x

# This function disables auto updates on a runnning machine.
disable_auto_updates() {
  sudo systemctl stop update-engine
  sudo systemctl mask update-engine
}


main() {
  echo "disable_auto_updates"
  disable_auto_updates
}

main
