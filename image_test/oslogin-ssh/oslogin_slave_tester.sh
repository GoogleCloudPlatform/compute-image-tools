#!/bin/bash
# Copyright 2018 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Prepare /bin/slave_tester.sh to be called later
cat > /bin/slave_tester.sh <<'EOF'
#!/bin/bash
# Copyright 2018 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Save STDOUT in fd_stdout
exec {fd_stdout}>&1
# Redirect STDOUT to STDERR
exec 1>&2
# Redirect STDERR to Serial port
exec 2>/dev/ttyS0

set -eux

echo "Executing $0 $@"

KEY=daisy-key

case $1 in
  add_key)
    if [ ! -f ${KEY}.pub ]; then
      ssh-keygen -t rsa -N '' -f daisy-key -C "$(uname -n)"
    fi
    cat ${KEY}.pub >&"$fd_stdout"
    NKEYS=$(gcloud compute os-login ssh-keys list | grep -v FINGERPRINT | wc -l)
    echo "Total number of keys currently on os-login: $NKEYS"
    gcloud compute os-login ssh-keys add --key-file=${KEY}.pub --ttl=30m
    ;;
  remove_key)
    gcloud compute os-login ssh-keys remove --key-file=${KEY}.pub
    ;;
  test_login)
    HOST=$2
    ssh -i $KEY -o IdentitiesOnly=yes -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=10 $HOST echo Logged
    ;;
  test_login_sudo)
    HOST=$2
    ssh -i $KEY -o IdentitiesOnly=yes -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=10 $HOST sudo echo Logged
    ;;
esac
EOF

chmod a+x /bin/slave_tester.sh

# Allow writting to serial port
chmod a+w /dev/ttyS0
logger -p daemon.info "BOOTED"
