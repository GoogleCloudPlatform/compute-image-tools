#!/bin/bash

# This script checks whether debian worker packages are installed correctly.

if ! qemu-img --version; then
  echo "FAILED JO2Pd99h4qRK5HIpc5NP: qemu-utils"
  exit
fi

if ! rsync --version; then
  echo "FAILED JO2Pd99h4qRK5HIpc5NP: rsync"
  exit
fi

if ! python3 --version; then
  echo "FAILED JO2Pd99h4qRK5HIpc5NP: python3"
  exit
fi

echo "SUCCESS JO2Pd99h4qRK5HIpc5NP"
