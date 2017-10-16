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

google_cloud = '''
deb http://packages.cloud.google.com/apt cloud-sdk-{deb_release} main
deb http://packages.cloud.google.com/apt google-compute-engine-{deb_release}-stable main
deb http://packages.cloud.google.com/apt google-cloud-packages-archive-keyring-{deb_release} main
'''

interfaces = '''
source-directory /etc/network/interfaces.d
auto lo
iface lo inet loopback
auto eth0
iface eth0 inet dhcp
'''

def translate():
  s = StringIO()
  c = pycurl.Curl()
  c.setopt(pycurl.HTTPHEADER, ['Metadata-Flavor:Google'])
  c.setopt(c.WRITEFUNCTION, s.write)
  url='http://metadata/computeMetadata/v1/instance/attributes'

  c.setopt(pycurl.URL, url + '/debian_release')
  c.perform()
  deb_release = s.getvalue()
  s.truncate(0)

  c.setopt(pycurl.URL, url + '/install_gce_packages')
  c.perform()
  install_gce = s.getvalue()

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

  if install_gce == 'true':
    print 'Installing GCE packages.'
    g.command(
        ['wget', 'https://packages.cloud.google.com/apt/doc/apt-key.gpg',
        '-O', '/tmp/gce_key'])
    g.command(['apt-key', 'add', '/tmp/gce_key'])
    g.rm('/tmp/gce_key')
    g.write(
        '/etc/apt/sources.list.d/google-cloud.list',
        google_cloud.format(deb_release=deb_release))
    # Remove Azure agent.
    g.command(['apt-get', 'remove', '-y', '-f', 'waagent', 'walinuxagent'])
    g.command(['apt-get', 'update'])
    g.sh(
        'DEBIAN_FRONTEND=noninteractive '
        'apt-get install --assume-yes --no-install-recommends '
        'google-cloud-packages-archive-keyring')
    g.sh(
        'DEBIAN_FRONTEND=noninteractive '
        'apt-get install --assume-yes --no-install-recommends '
        'google-compute-engine python-google-compute-engine '
        'python3-google-compute-engine')
    # TODO: install google-cloud-sdk using apt when post install scripts are fixed
    print g.sh(r'apt-get download google-cloud-sdk;'
        r'dpkg --unpack google-cloud-sdk*.deb;'
        r'rm /var/lib/dpkg/info/google-cloud-sdk.postinst -f;'
        r'dpkg --configure google-cloud-sdk;'
        r'apt-get install -yf')

  # Update grub config to log to console.
  g.command(
      ['sed', '-i',
      r's#^\(GRUB_CMDLINE_LINUX=".*\)"$#\1 console=ttyS0,38400n8"#',
      '/etc/default/grub'])

  # Disable predictive network interface naming in Stretch.
  if deb_release == 'stretch':
    g.command(
        ['sed', '-i',
        r's#^\(GRUB_CMDLINE_LINUX=".*\)"$#\1 net.ifnames=0 biosdevname=0"#',
        '/etc/default/grub'])

  g.command(['update-grub2'])

  # Reset network for DHCP.
  print 'Resetting network to DHCP for eth0.'
  g.write('/etc/network/interfaces', interfaces)

  # Remove SSH host keys.
  print 'Removing SSH host keys.'
  g.sh('rm -f /etc/ssh/ssh_host_*')

  g.umount_all()

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
