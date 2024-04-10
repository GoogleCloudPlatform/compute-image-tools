#!/bin/bash
#
# Disables vboot and unlocks rootfs.

set -o errexit
set -o pipefail
set -x

source /usr/share/vboot/bin/common_minimal.sh

# This procedure may or may not trigger an immediate reboot. Cos-customizer is
# designed to re-run (and only from) the script that triggered the reboot, so
# it is important that this procedure handles re-entry gracefully.
disable_vboot() {
  local dir
  dir="$(mktemp -d)"

  mount /dev/disk/by-label/EFI-SYSTEM "${dir}"
  grub="${dir}/efi/boot/grub.cfg"
  if ! grep "defaultA=0" "${grub}" || ! grep "defaultB=1" "${grub}"; then
    # Set defaultA=0 and defaultB=1
    sed -i \
      -e '/^defaultA=/s:=.*:=0:' \
      -e '/^defaultB=/s:=.*:=1:' \
      -e 's/module.sig_enforce=1//' \
      -e 's/cros_efi/cros_efi ima_appraise=off/' \
      -e 's/ ro / rw /' \
      "${grub}"
    sync
  fi

  local -r rootdev="$(rootdev -s)"
  sudo blockdev --setrw "${rootdev}"
  enable_rw_mount "${rootdev}"

  umount "${dir}"
  # Triggers an immediate reboot.
  reboot
}

main() {
  echo "disable_vboot"
  disable_vboot
}

main
