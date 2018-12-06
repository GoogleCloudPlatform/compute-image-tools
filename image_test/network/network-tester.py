#!/usr/bin/env python3
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

import logging
import time
import urllib.error
import urllib.request


from google import auth
from googleapiclient import discovery
import netaddr
import utils


class RequestError(Exception):
  def __init__(self, name, instance, ip_range, positive_test=True):
    self.name = name
    self.instance = instance
    self.ip_range = ip_range
    self.positive_test = positive_test

  def __str__(self):
    text = "%s didn't redirect ip traffic from %s to instance %s"
    if not self.positive_test:
      text = "%s redirected ip traffic from %s to instance %s but it shouldn't"
    return text % (self.name, self.ip_range, self.instance)


def _RequestInfo(URL, timeout=10):
  logging.info('Retrieving: %s' % URL)
  request = urllib.request.Request(URL)
  return urllib.request.urlopen(request).read().decode()


def RequestOS(addr):
  return _RequestInfo("http://%s/os" % addr)


def RequestHostname(addr):
  return _RequestInfo("http://%s/hostname" % addr)


def TestIPAlias(instance, ip_alias, ip_mask):
  testee_hostname = RequestHostname(instance)
  cidr = "%s/%s" % (ip_alias, ip_mask)

  # caching ips out of range for using it on negative tests
  ips = [str(ip) for ip in netaddr.IPNetwork(
      "{}/{}".format(ip_alias, ip_mask))]

  # positive test on expected IPs
  for ip in ips:
    try:
      if testee_hostname != RequestHostname(ip):
        raise Exception("Alias hostname should be the same from host machine.")
    except urllib.error.URLError:
      raise RequestError("ip_alias", instance, cidr)

  # negative test
  superset_ips = [str(ip) for ip in netaddr.IPNetwork(
      "{}/{}".format(ip_alias, int(ip_mask) - 1))]
  invalid_ip = superset_ips[superset_ips.index(ips[0]) - 1]
  try:
    if testee_hostname == RequestHostname(invalid_ip):
      raise RequestError("ip_alias", instance, cidr, positive_test=False)
  except urllib.error.URLError:
    # great, it didn't even respond! So no machine is using that ip.
    pass


def SetIPAlias(MD, instance, ip_alias=None, ip_mask=None):
  # by default, remove the aliasing
  alias_ip_ranges = []
  if ip_alias:
    cidr = "%s/%s" % (ip_alias, ip_mask)
    alias_ip_ranges = [{'ipCidrRange': cidr}]

  info = {
      'aliasIpRanges': alias_ip_ranges,
      # a previous fingerprint is needed in order to change the instance
      'fingerprint': MD.GetInstanceIfaces(instance)[0][u'fingerprint']
  }
  MD.Wait(MD.SetInstanceIface(instance, info))

  # The operation is completed but it might not guarantee that the agent has
  # already applied the rule. For that reason, wait
  time.sleep(30)


def TestForwardingRule(MD, instance, rule_name):
  ip = MD.GetForwardingRuleIP(rule_name)
  if RequestHostname(instance) != RequestHostname(ip):
    raise RequestError(rule_name, instance, ip)


def main():
  MM = utils.MetadataManager
  testee = MM.FetchMetadataDefault('testee')
  alias_ip = MM.FetchMetadataDefault('alias_ip')
  alias_ip_mask = MM.FetchMetadataDefault('alias_ip_mask')

  credentials, _ = auth.default()
  compute = utils.GetCompute(discovery, credentials)
  MD = MM(compute, testee)

  # Verify IP aliasing is working: not available on Windows environment
  if RequestOS(testee) != 'windows':
    #  1) Testing if it works on creation time
    TestIPAlias(testee, alias_ip, alias_ip_mask)

    #  2) Removing it and guaranteeing that it doesn't work anymore
    SetIPAlias(MD, testee, [])
    try:
      TestIPAlias(testee, alias_ip, alias_ip_mask)
    except RequestError as e:
      # the positive test is expected to fail, as no alias ip exists.
      if not e.positive_test:
        raise e

    #  3) Re-add it while the machine is running and verify that it works
    SetIPAlias(MD, testee, alias_ip, alias_ip_mask)
    TestIPAlias(testee, alias_ip, alias_ip_mask)

  # Ensure routes are added when ForwardingRules are created
  testee_fr = MM.FetchMetadataDefault('testee_forwarding_rule')
  TestForwardingRule(MD, testee, testee_fr)


if __name__ == '__main__':
  utils.RunTest(main)
