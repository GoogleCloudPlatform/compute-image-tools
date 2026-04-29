#!/bin/bash
#
# Disables vboot and unlocks rootfs.

set -o errexit
set -o pipefail
set -x

# Located in COS images, contains the func "enable_rw_mount".
# https://chromium.googlesource.com/chromiumos/platform/vboot_reference/+/master/scripts/image_signing/common_minimal.sh
source /usr/share/vboot/bin/common_minimal.sh

# This procedure may or may not trigger an immediate reboot. Cos-customizer is
# designed to re-run (and only from) the script that triggered the reboot, so
# it is important that this procedure handles re-entry gracefully.
disable_vboot() {
  # Make a temp dir.
  local dir
  dir="$(mktemp -d)"

# Mount EFI on the temp dir and modify the grub.cfg file...
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
    # Call enable_rw_mount function, it is responsible for enabling the read-write
    # mount and is sourced at the top of the file.
    local -r rootdev="$(rootdev -s)"
    sudo blockdev --setrw "${rootdev}"
    enable_rw_mount "${rootdev}"
    umount "${dir}"
    # Triggers an immediate reboot.
    reboot
    # Hang after reboot: the script should not continue executing (return) after
    # the reboot call due to COS customizer design.
    echo "NOTE: infinite loop to prevent this script from continuing execution after the reboot call"
    while true; do sleep 1; done
  else
    umount "${dir}"
  fi
}


main() {
  echo "disable_vboot"
  disable_vboot
}

main
