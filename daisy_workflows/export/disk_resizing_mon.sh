#!/bin/bash
SIZE=210
while [ 1 ]
do
  echo "Resizing disk to ${SIZE}GB..."
  gcloud compute disks resize /dev/sdb --size=${SIZE}GB --quiet --zone us-east1-b
  SIZE=$(awk "BEGIN {print int(${SIZE} + 10)}")
  sleep 20
done
