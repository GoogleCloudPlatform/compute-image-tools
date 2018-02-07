#!/bin/bash

for device in "/dev/sdb" "/dev/sdc" "/dev/sdd"; do 
  if [[ ! -b $device ]]; then
    echo "FAILURE JO2Pd99h4qRK5HIpc5NP: '${device}'' does not exist"
  fi
done

echo "SUCCESS JO2Pd99h4qRK5HIpc5NP"