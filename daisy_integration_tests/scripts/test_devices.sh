#!/bin/bash
devices=("/dev/sdb" "/dev/sdc")

while [ $devices -gt 0 ]; do
  sleep 1
  for device in $devices; do 
    if [[ -b $device ]]; then
      devices=(${devices[@]/$device})
    fi
  done
done

echo "SUCCESS JO2Pd99h4qRK5HIpc5NP"