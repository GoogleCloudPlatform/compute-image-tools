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

"""Translate the Ubuntu image on a GCE VM.

Parameters (retrieved from instance metadata):

ubuntu_release: The version of the distro
install_gce_packages: True if GCE agent and SDK should be installed
"""

import logging

import utils

utils.AptGetInstall(['python-guestfs', 'tinyproxy'])

import guestfs


tinyproxy_cfg = '''
User tinyproxy
Group tinyproxy
Port 8888
Timeout 600
LogLevel Info
PidFile "/run/tinyproxy/tinyproxy.pid"
MaxClients 100
MinSpareServers 5
MaxSpareServers 20
StartServers 10
MaxRequestsPerChild 0
Allow 127.0.0.1
ViaProxyName "tinyproxy"
ConnectPort 443
ConnectPort 563
'''

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

trusty_network = '''
# The loopback network interface
auto lo
iface lo inet loopback

# The primary network interface
auto eth0
iface eth0 inet dhcp

source /etc/network/interfaces.d/*.cfg
'''

xenial_network = '''
# The loopback network interface
auto lo
iface lo inet loopback

# The primary network interface
auto ens4
iface ens4 inet dhcp

source /etc/network/interfaces.d/*.cfg
'''


def DistroSpecific(g):
  ubu_release = utils.GetMetadataAttribute('ubuntu_release')
  install_gce = utils.GetMetadataAttribute('install_gce_packages')

  # Remove any hard coded DNS settings in resolvconf.
  if ubu_release != 'bionic':
    logging.info('Resetting resolvconf base.')
    g.sh('echo "" > /etc/resolvconf/resolv.conf.d/base')

  # Try to reset the network to DHCP.
  if ubu_release == 'trusty':
    g.write('/etc/network/interfaces', trusty_network)
  elif ubu_release in ('xenial', 'bionic'):
    g.write('/etc/network/interfaces', xenial_network)

  if install_gce == 'true':
    g.command(['apt-get', 'update'])
    logging.info('Installing cloud-init.')
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
      logging.debug(str(e))
      logging.warn('Could not uninstall Azure agent. Continuing anyway.')

    g.write(
        '/etc/apt/sources.list.d/partner.list',
        partner_list.format(ubu_release=ubu_release))

    g.write('/etc/cloud/cloud.cfg.d/91-gce-system.cfg', gce_system)

    # Use host machine as http proxy so cloud-init can access GCE API
    with open('/etc/tinyproxy/tinyproxy.conf', 'w') as cfg:
        cfg.write(tinyproxy_cfg)
    utils.Execute(['/etc/init.d/tinyproxy', 'restart'])
    default_gw = g.sh("ip route | awk '/default/ { printf $3 }'")
    logging.debug(
        g.sh('http_proxy="http://%s:8888" cloud-init -d init' % default_gw))

    logging.info('Installing GCE packages.')
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


def main():
  g = utils.MountDisk('/dev/sdb')
  DistroSpecific(g)
  utils.CommonRoutines(g)
  utils.UnmountDisk(g)


if __name__ == '__main__':
  utils.RunTranslate(main)
