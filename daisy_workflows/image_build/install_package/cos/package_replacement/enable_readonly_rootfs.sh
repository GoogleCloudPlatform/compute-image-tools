#!/bin/bash

# This script re-enables the read-only root fs.

set -o errexit
set -o pipefail
set -x

enable_vboot(){
  local dir
  
  dir="$(mktemp -d)"

  mount /dev/disk/by-label/EFI-SYSTEM "${dir}"
  grub="${dir}/efi/boot/grub.cfg"

  sed -i -e 's/ rw / ro /' "${grub}"

  # Mount-etc-overlays is responsible for mounting a dir to make /etc writable
  # despite the root fs being read-only. Machine-id should be present in /run
  # according to systemd (more info in mount-etc-overlays file). And the
  # machine-id file in /etc should be present but empty for VM boot.
  rm /etc/machine-id
  touch /etc/machine-id

  umount "${dir}"
}

main() {
  echo "enable_vboot"
  enable_vboot
}

main
