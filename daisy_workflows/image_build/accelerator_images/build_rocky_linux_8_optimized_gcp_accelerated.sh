#!/bin/bash
dnf config-manager --add-repo https://developer.download.nvidia.com/compute/cuda/repos/rhel8/x86_64/cuda-rhel8.repo || echo "BuildFailure"
dnf install -y gcc make kernel-devel || echo "BuildFailure"
curl -L -o nvidia.run https://us.download.nvidia.com/tesla/550.90.12/NVIDIA-Linux-x86_64-550.90.12.run || echo "BuildFailure"
chmod +x ./nvidia.run || echo "BuildFailure"
# DKMS - not suitable for prod
./nvidia.run -s --kernel-source-path=/usr/src/kernels/4.18.0-553.8.1.el8_10.cloud.0.1.x86_64/ || echo "BuildFailure"
dnf install -y rdma-core || echo "BuildFailure"
nvidia-smi || echo "BuildFailure"
echo "BuildSuccess"
