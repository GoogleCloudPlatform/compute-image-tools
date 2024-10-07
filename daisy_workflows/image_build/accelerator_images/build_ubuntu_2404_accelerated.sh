#!/bin/sh
set -x
export DEBIAN_FRONTEND=noninteractive
apt update -y || echo "BuildFailure"
apt upgrade -y || echo "BuildFailure"
# DKMS - not suitable for prod
apt -y install nvidia-utils-550 linux-modules-nvidia-550-open-gcp rdma-core || echo "BuildFailure"
# This is only for mlx5_ib - once that's in the main package stop including it
apt -y install linux-modules-extra-gcp || echo "BuildFailure"
tee /usr/bin/add-nvidia-repositories << EOF
#!/bin/bash
set -e
curl https://developer.download.nvidia.com/compute/cuda/repos/ubuntu2404/x86_64/cuda-keyring_1.1-1_all.deb -o /tmp/cuda-keyring_1.1-1_all.deb
dpkg -i /tmp/cuda-keyring_1.1-1_all.deb
rm -f /tmp/cuda-keyring_1.1-1_all.deb
EOF
chmod +x /usr/bin/add-nvidia-repositories || echo "BuildFailure"
echo "BuildSuccess"
