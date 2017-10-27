#!/usr/bin/env python
# Copyright 2017 Google Inc. All Rights Reserved.
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

import guestfs
import pycurl
import sys
import trace
from StringIO import StringIO


disk = '/dev/sdb'

repo_compute = '''
[google-cloud-compute]
name=Google Cloud Compute
baseurl=https://packages.cloud.google.com/yum/repos/google-cloud-compute-el%s-x86_64
enabled=1
gpgcheck=1
repo_gpgcheck=1
gpgkey=https://packages.cloud.google.com/yum/doc/yum-key.gpg
       https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg
'''

repo_sdk = '''
[google-cloud-sdk]
name=Google Cloud SDK
baseurl=https://packages.cloud.google.com/yum/repos/cloud-sdk-el%s-x86_64
enabled=1
gpgcheck=1
repo_gpgcheck=1
gpgkey=https://packages.cloud.google.com/yum/doc/yum-key.gpg
       https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg
'''

ifcfg_eth0 = '''
BOOTPROTO=dhcp
DEVICE=eth0
ONBOOT=yes
TYPE=Ethernet
DEFROUTE=yes
PEERDNS=yes
PEERROUTES=yes
DHCP_HOSTNAME=localhost
IPV4_FAILURE_FATAL=no
NAME="System eth0"
MTU=1460
PERSISTENT_DHCLIENT="y"
'''

grub2_cfg = '''
GRUB_TIMEOUT=0
GRUB_DISTRIBUTOR="$(sed 's, release .*$,,g' /etc/system-release)"
GRUB_DEFAULT=saved
GRUB_DISABLE_SUBMENU=true
GRUB_TERMINAL="serial console"
GRUB_SERIAL_COMMAND="serial --speed=38400"
GRUB_CMDLINE_LINUX="crashkernel=auto console=ttyS0,38400n8"
GRUB_DISABLE_RECOVERY="true"
'''

grub_cfg = '''
default=0
timeout=0
serial --unit=0 --speed=38400
terminal --timeout=0 serial console
'''

def translate():
  s = StringIO()
  c = pycurl.Curl()
  c.setopt(pycurl.HTTPHEADER, ['Metadata-Flavor:Google'])
  c.setopt(c.WRITEFUNCTION, s.write)
  url='http://metadata/computeMetadata/v1/instance/attributes'

  c.setopt(pycurl.URL, url + '/el_release')
  c.perform()
  el_release = s.getvalue()
  s.truncate(0)

  c.setopt(pycurl.URL, url + '/install_gce_packages')
  c.perform()
  install_gce = s.getvalue()
  s.truncate(0)

  c.setopt(pycurl.URL, url + '/use_rhel_gce_license')
  c.perform()
  rhel_license = s.getvalue()

  c.close()

  # All new Python code should pass python_return_dict=True
  # to the constructor.  It indicates that your program wants
  # to receive Python dicts for methods in the API that return
  # hashtables.
  g = guestfs.GuestFS(python_return_dict=True)
  g.set_verbose(1)
  g.set_trace(1)

  # Enable network
  g.set_network(True)

  # Attach the disk image to libguestfs.
  g.add_drive_opts(disk)

  # Run the libguestfs back-end.
  g.launch()

  # Ask libguestfs to inspect for operating systems.
  roots = g.inspect_os()
  if len(roots) == 0:
    raise Exception('inspect_vm: no operating systems found')

  # Sort keys by length, shortest first, so that we end up
  # mounting the filesystems in the correct order.
  mps = g.inspect_get_mountpoints(roots[0])
  def compare(a, b):
    return len(a) - len(b)

  for device in sorted(mps.keys(), compare):
    try:
      g.mount(mps[device], device)
    except RuntimeError as msg:
      print '%s (ignored)' % msg

  if rhel_license == 'true':
    if 'Red Hat' in g.cat('/etc/redhat-release'):
      g.command(['yum', 'remove', '-y', '*rhui*'])
      print 'Adding in GCE RHUI package.'
      g.write('/etc/yum.repos.d/google-cloud.repo', repo_compute % el_release)
      g.command(
          ['yum', 'install', '-y', 'google-rhui-client-rhel%s' % el_release])

  if install_gce == 'true':
    print 'Installing GCE packages.'
    g.write('/etc/yum.repos.d/google-cloud.repo', repo_compute % el_release)
    if el_release == 7:
      g.write_append(
          '/etc/yum.repos.d/google-cloud.repo', repo_sdk % el_release)
      g.command(['yum', '-y', 'install', 'google-cloud-sdk'])
    g.command([
        'yum', '-y', 'install', 'google-compute-engine',
        'python-google-compute-engine'])

  print 'Updating initramfs'
  for kver in g.ls('/lib/modules'):
    if el_release == '6':
      # Version 6 doesn't have option --kver
      g.command(['dracut', '-v', '-f', kver])
    else:
      g.command(['dracut', '-v', '-f', '--kver', kver])

  print 'Update grub configuration'
  if el_release == '6':
    # Version 6 doesn't have grub2, file grub.conf needs to be updated by hand
    g.write('/tmp/grub_gce_generated', grub_cfg)
    g.sh(
        r'grep -P "^[\t ]*initrd|^[\t ]*root|^[\t ]*kernel|^[\t ]*title" '
            r'/boot/grub/grub.conf >> /tmp/grub_gce_generated;'
        r'sed -i "s/console=ttyS0[^ ]*//g" /tmp/grub_gce_generated;'
        r'sed -i "/^[\t ]*kernel/s/$/ console=ttyS0,38400n8/" '
            r'/tmp/grub_gce_generated;'
        r'mv /tmp/grub_gce_generated /boot/grub/grub.conf')
  else:
    g.write('/etc/default/grub', grub2_cfg)
    g.command(['grub2-mkconfig', '-o', '/boot/grub2/grub.cfg'])


  # Remove udev file to force it to be re-generated
  g.rm_f('/etc/udev/rules.d/70-persistent-net.rules')

  # Reset network for DHCP.
  print 'Resetting network to DHCP for eth0.'
  g.write('/etc/sysconfig/network-scripts/ifcfg-eth0', ifcfg_eth0)

  # Remove SSH host keys.
  print "Removing SSH host keys."
  g.sh("rm -f /etc/ssh/ssh_host_*")

  try:
    g.umount_all()
  except Exception as e:
    print str(e)
    print 'Unmount failed. Continuing anyway.'


def main():
  try:
    translate()
    print 'TranslateSuccess: Translation finished.'
  except Exception as e:
    print 'TranslateFailed: error: '
    print str(e)

if __name__=='__main__':
  tracer = trace.Trace(
      ignoredirs=[sys.prefix, sys.exec_prefix], trace=1, count=0)
  tracer.run('main()')
