#!/usr/bin/env python2
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
import urllib2


import genips
from google import auth
from googleapiclient import discovery
import utils


class RequestError(Exception):
  def __init__(self, name, instance, ip_range):
    self.name = name
    self.instance = instance
    self.ip_range = ip_range

  def __str__(self):
    return "%s didn't redirect ip %s to instance %s" % (
        self.name, self.ip_range, self.instance)


def GetHostname(addr, timeout=10):
  URL = "http://%s/hostname" % addr
  logging.info('Retrieving: %s' % URL)
  return urllib2.urlopen(URL, None, timeout).read()


def TestIPAlias(testee, ip_alias, ip_mask):
  testee_hostname = GetHostname(testee)

  # caching ips out of range for using it on negative tests
  ips = list(genips.GenIPs(ip_alias, ip_mask))

  # positive test on expected IPs
  for ip in ips:
    if testee_hostname != GetHostname(ip):
      raise Exception("Alias hostname should be the same from host machine.")

  # negative test
  superset_ips = list(genips.GenIPs(ip_alias, int(ip_mask) - 1))
  invalid_ip = superset_ips[superset_ips.index(ips[0]) - 1]
  try:
    if testee_hostname == GetHostname(invalid_ip):
      e = "IP alias out of range (%s) should not respond." % invalid_ip
      raise Exception(e)
  except (RequestError, urllib2.URLError):
    pass


def TestForwardingRule(MD, instance, rule_name):
  ip = MD.GetForwardingRuleIP(rule_name)
  if GetHostname(instance) != GetHostname(ip):
    raise RequestError(rule_name, instance, ip)


def main():
  MM = utils.MetadataManager
  testee = MM.FetchMetadataDefault('testee')

  # Verify IP aliasing is working
  TestIPAlias(testee, MM.FetchMetadataDefault('alias_ip'),
      MM.FetchMetadataDefault('alias_ip_mask'))

  # Ensure routes are added when ForwardingRules are created
  credentials, _ = auth.default()
  compute = utils.GetCompute(discovery, credentials)

  testee_fr = MM.FetchMetadataDefault('testee_forwarding_rule')
  TestForwardingRule(MM(compute, testee), testee, testee_fr)


if __name__ == '__main__':
  utils.RunTest(main)
