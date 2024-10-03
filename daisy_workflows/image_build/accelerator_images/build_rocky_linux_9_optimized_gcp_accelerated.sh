#!/bin/bash
dnf config-manager --add-repo https://developer.download.nvidia.com/compute/cuda/repos/rhel9/x86_64/cuda-rhel9.repo || echo "BuildFailure"
dnf install -y gcc make kernel-devel kernel || echo "BuildFailure"
test -f /var/tmp/kernel-upgrade-done || sh -c 'touch /var/tmp/kernel-upgrade-done && sync && reboot'
curl -L -o nvidia.run https://us.download.nvidia.com/tesla/550.90.12/NVIDIA-Linux-x86_64-550.90.12.run || echo "BuildFailure"
chmod +x ./nvidia.run || echo "BuildFailure"
# DKMS - not suitable for prod
./nvidia.run -s --kernel-source-path=/usr/src/kernels/$(uname -r)/ || echo "BuildFailure"
dnf install -y perl-File-Find perl-File-Copy perl-File-Compare perl-sigtrap wget lsof tk gcc-gfortran tcl pciutils || echo "BuildFailure"
wget https://content.mellanox.com/ofed/MLNX_OFED-23.10-3.2.2.0/MLNX_OFED_LINUX-23.10-3.2.2.0-rhel9.4-x86_64.tgz || echo "BuildFailure"
tar xf MLNX_OFED_LINUX-23.10-3.2.2.0-rhel9.4-x86_64.tgz || echo "BuildFailure"
cd MLNX_OFED_LINUX-23.10-3.2.2.0-rhel9.4-x86_64 || echo "BuildFailure"
./mlnxofedinstall --guest --force || echo "BuildFailure"
cd ..
rm -rf MLNX_OFED_LINUX-23.10-3.2.2.0-rhel9.4-x86_64 MLNX_OFED_LINUX-23.10-3.2.2.0-rhel9.4-x86_64.tgz
echo "BuildSuccess"
