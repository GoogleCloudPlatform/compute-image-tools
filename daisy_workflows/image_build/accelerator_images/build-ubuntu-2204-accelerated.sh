#!/bin/sh
set -x
export DEBIAN_FRONTEND=noninteractive
apt update -y || echo "BuildFailure"
apt upgrade -y || echo "BuildFailure"
# DKMS - not suitable for prod
apt -y install nvidia-driver-550-server rdma-core || echo "BuildFailure"
echo "BuildSuccess"
