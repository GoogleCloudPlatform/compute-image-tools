#!/bin/bash
# * PREVIEW *
# Script to convert GCE VM images to c3 bare metal images.
# Supports Ubuntu Only


sed -i \
  '/if pname == "Google Compute Engine" or pname == "Google":/c\    if pname in ("Google Compute Engine", "Google", "Izumi"):' \
  /usr/lib/python3/dist-packages/cloudinit/sources/DataSourceGCE.py

apt-mark hold cloud-init

echo "BuildSuccess:  Bare Metal Ubuntu image build succeeded."
