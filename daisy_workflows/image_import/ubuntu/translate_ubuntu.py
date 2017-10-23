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

partner_list = '''
# Enabled for Google Cloud SDK
deb http://archive.canonical.com/ubuntu {ubu_release} partner
'''

gce_system = '''
datasource_list: [ GCE ]
system_info:
   package_mirrors:
     - arches: [i386, amd64]
       failsafe:
         primary: http://archive.ubuntu.com/ubuntu
         security: http://security.ubuntu.com/ubuntu
       search:
         primary:
           - http://%(region)s.gce.archive.ubuntu.com/ubuntu/
           - http://%(availability_zone)s.gce.clouds.archive.ubuntu.com/ubuntu/
           - http://gce.clouds.archive.ubuntu.com/ubuntu/
         security: []
     - arches: [armhf, armel, default]
       failsafe:
         primary: http://ports.ubuntu.com/ubuntu-ports
         security: http://ports.ubuntu.com/ubuntu-ports
'''

def translate():
  s = StringIO()
  c = pycurl.Curl()
  c.setopt(pycurl.HTTPHEADER, ['Metadata-Flavor:Google'])
  c.setopt(c.WRITEFUNCTION, s.write)
  url='http://metadata/computeMetadata/v1/instance/attributes'

  c.setopt(pycurl.URL, url + '/ubuntu_release')
  c.perform()
  ubu_release = s.getvalue()
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
  # Set the product name as cloud-init checks it to confirm this is a VM in GCE
  g.config('-smbios', 'type=1,product=Google Compute Engine')
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
    g.command(['apt-get', 'update'])
    print 'Installing cloud-init.'
    g.sh(
        'DEBIAN_FRONTEND=noninteractive apt-get install -y'
        ' --no-install-recommends cloud-init')

    # Try to remove azure or aws configs so cloud-init has a chance.
    g.sh('rm -f /etc/cloud/cloud.cfg.d/*azure*')
    g.sh('rm -f /etc/cloud/cloud.cfg.d/*waagent*')
    g.sh('rm -f /etc/cloud/cloud.cfg.d/*walinuxagent*')
    g.sh('rm -f /etc/cloud/cloud.cfg.d/*aws*')
    g.sh('rm -f /etc/cloud/cloud.cfg.d/*amazon*')

    # Remove Azure agent.
    try:
      g.command(['apt-get', 'remove', '-y', '-f', 'waagent', 'walinuxagent'])
    except Exception as e:
      print str(e)
      print 'Could not uninstall Azure agent. Continuing anyway.'

    g.write(
        '/etc/apt/sources.list.d/partner.list',
        partner_list.format(ubu_release=ubu_release))

    g.write('/etc/cloud/cloud.cfg.d/91-gce-system.cfg', gce_system)

    # Use host machine as http proxy so cloud-init can access GCE API
    default_gw = g.sh("ip route | awk '/default/ { printf $3 }'")
    print g.sh('http_proxy="http://%s:8888" cloud-init -d init' % default_gw)

    print 'Installing GCE packages.'
    g.command(['apt-get', 'update'])
    g.sh(
        'DEBIAN_FRONTEND=noninteractive apt-get install -y'
        ' --no-install-recommends gce-compute-image-packages google-cloud-sdk')

  # Update grub config to log to console.
  g.command(
      ['sed', '-i',
      r's#^\(GRUB_CMDLINE_LINUX=".*\)"$#\1 console=ttyS0,38400n8"#',
      '/etc/default/grub'])

  g.command(['update-grub2'])

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
