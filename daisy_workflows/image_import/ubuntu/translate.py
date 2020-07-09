#!/usr/bin/env python3
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
import utils.diskutils as diskutils


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

  # If present, remove any hard coded DNS settings in resolvconf.
  if ubu_release != 'bionic' and \
      g.exists('/etc/resolvconf/resolv.conf.d/base'):
    logging.info('Resetting resolvconf base.')
    g.sh('echo "" > /etc/resolvconf/resolv.conf.d/base')

  # Try to reset the network to DHCP.
  if ubu_release == 'trusty':
    g.write('/etc/network/interfaces', trusty_network)
  elif ubu_release == 'xenial':
    g.write('/etc/network/interfaces', xenial_network)

  if install_gce == 'true':
    utils.update_apt(g)
    logging.info('Installing cloud-init.')
    utils.install_apt_packages(g, 'cloud-init')

    # Try to remove azure or aws configs so cloud-init has a chance.
    g.sh('rm -f /etc/cloud/cloud.cfg.d/*azure*')
    g.sh('rm -f /etc/cloud/cloud.cfg.d/*curtin*')
    g.sh('rm -f /etc/cloud/cloud.cfg.d/*waagent*')
    g.sh('rm -f /etc/cloud/cloud.cfg.d/*walinuxagent*')
    g.sh('rm -f /etc/cloud/cloud.cfg.d/*aws*')
    g.sh('rm -f /etc/cloud/cloud.cfg.d/*amazon*')
    if ubu_release == 'bionic':
      g.sh('rm -f /etc/netplan/*')
      logging.debug(g.sh('cloud-init clean'))

    remove_azure_agents(g)

    g.write(
        '/etc/apt/sources.list.d/partner.list',
        partner_list.format(ubu_release=ubu_release))

    g.write('/etc/cloud/cloud.cfg.d/91-gce-system.cfg', gce_system)

    # Use host machine as http proxy so cloud-init can access GCE API
    with open('/etc/tinyproxy/tinyproxy.conf', 'w') as cfg:
        cfg.write(tinyproxy_cfg)
    utils.Execute(['/etc/init.d/tinyproxy', 'restart'])
    default_gw = g.sh("ip route | awk '/default/ { printf $3 }'")
    try:
      logging.debug(
          g.sh('http_proxy="http://%s:8888" cloud-init -d init' % default_gw))
    except Exception as e:
      logging.debug('Failed to run cloud-init. Details: {}.'.format(e))
      raise RuntimeError(
        'Failed to run cloud-init. Connect to a shell in the original VM '
        'and ensure that the following command executes successfully: '
        'apt-get install -y --no-install-recommends cloud-init '
        '&& cloud-init -d init')
    logging.info('Installing GCE packages.')
    utils.update_apt(g)
    utils.install_apt_packages(g, 'gce-compute-image-packages',
                               'google-cloud-sdk')
  # Update grub config to log to console.
  g.command(
      ['sed', '-i',
      r's#^\(GRUB_CMDLINE_LINUX=".*\)"$#\1 console=ttyS0,38400n8"#',
      '/etc/default/grub'])

  g.command(['update-grub2'])


def remove_azure_agents(g):
  try:
    g.command(['apt-get', 'remove', '-y', '-f', 'walinuxagent'])
  except Exception as e:
    logging.debug(str(e))

  try:
    g.command(['apt-get', 'remove', '-y', '-f', 'waagent'])
  except Exception as e:
    logging.debug(str(e))


def main():
  g = diskutils.MountDisk('/dev/sdb')
  DistroSpecific(g)
  utils.CommonRoutines(g)
  diskutils.UnmountDisk(g)


if __name__ == '__main__':
  utils.RunTranslate(main)
