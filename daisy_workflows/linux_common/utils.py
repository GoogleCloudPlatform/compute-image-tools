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

"""Utility functions for all VM scripts."""

import functools
import logging
import os
import re
import stat
import subprocess
import sys
import time
import trace
import traceback
import urllib2
import uuid


SUCCESS_LEVELNO = logging.ERROR - 5


def YumInstall(package_list):
  if YumInstall.first_run:
    Execute(['yum', 'update'])
    YumInstall.first_run = False
  Execute(['yum', '-y', 'install'] + package_list)


YumInstall.first_run = True


def AptGetInstall(package_list):
  if AptGetInstall.first_run:
    try:
      Execute(['apt-get', 'update'])
    except subprocess.CalledProcessError as error:
      logging.info('Apt update failed, trying again: %s' % error)
      Execute(['apt-get', 'update'], raise_errors=False)
    AptGetInstall.first_run = False

  env = os.environ.copy()
  env['DEBIAN_FRONTEND'] = 'noninteractive'
  return Execute(['apt-get', '-q', '-y', 'install'] + package_list, env=env)


AptGetInstall.first_run = True


def PipInstall(package_list):
  """Install Python modules via pip. Assumes pip is already installed."""
  return Execute(['pip', 'install', '-U'] + package_list)


def Execute(cmd, cwd=None, capture_output=False, env=None, raise_errors=True):
  """Execute an external command (wrapper for Python subprocess)."""
  logging.info('Executing command: %s' % str(cmd))
  stdout = subprocess.PIPE if capture_output else None
  p = subprocess.Popen(cmd, cwd=cwd, env=env, stdout=stdout)
  output = p.communicate()[0]
  returncode = p.returncode
  if returncode != 0:
    # Error
    if raise_errors:
      raise subprocess.CalledProcessError(returncode, cmd)
    else:
      logging.info('Command returned error status %d' % returncode)
  if output:
    logging.info(output)
  return returncode, output


def HttpGet(url, headers=None):
  request = urllib2.Request(url)
  if headers:
    for key in headers.keys():
      request.add_unredirected_header(key, headers[key])
  return urllib2.urlopen(request).read()


def GetMetadataParam(name, default_value=None, raise_on_not_found=False):
  try:
    url = 'http://metadata.google.internal/computeMetadata/v1/instance/attributes/%s' % name
    return HttpGet(url, headers={'Metadata-Flavor': 'Google'})
  except urllib2.HTTPError:
    if raise_on_not_found:
      raise ValueError('Metadata key "%s" not found' % name)
    else:
      return default_value


def MountDisk(disk):
  # Note: guestfs is not imported in the beginning of the file as it might not
  # be installed when this module is loaded
  import guestfs

  # All new Python code should pass python_return_dict=True
  # to the constructor.  It indicates that your program wants
  # to receive Python dicts for methods in the API that return
  # hashtables.
  g = guestfs.GuestFS(python_return_dict=True)
  # Set the product name as cloud-init checks it to confirm this is a VM in GCE
  g.config('-smbios', 'type=1,product=Google Compute Engine')
  g.set_verbose(1)
  g.set_trace(1)

  g.set_memsize(4096)

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
      logging.warn('%s (ignored)' % msg)

  return g


def UnmountDisk(g):
  try:
    g.umount_all()
  except Exception as e:
    logging.debug(str(e))
    logging.warn('Unmount failed. Continuing anyway.')


def CommonRoutines(g):
  # Remove udev file to force it to be re-generated
  logging.info('Removing udev 70-persistent-net.rules.')
  g.rm_f('/etc/udev/rules.d/70-persistent-net.rules')

  # Remove SSH host keys.
  logging.info('Removing SSH host keys.')
  g.sh("rm -f /etc/ssh/ssh_host_*")


def RunTranslate(translate_func):
  try:
    tracer = trace.Trace(
        ignoredirs=[sys.prefix, sys.exec_prefix], trace=1, count=0)
    tracer.runfunc(translate_func)
    logging.success('Translation finished.')
  except Exception as e:
    logging.error('error: %s', str(e))


def GetMetadataParamBool(name, default_value):
  value = GetMetadataParam(name, default_value)
  if not value:
    return False
  return True if value.lower() == 'yes' else False


def MakeExecutable(file_path):
  os.chmod(file_path, os.stat(file_path).st_mode | stat.S_IEXEC)


def ReadFile(file_path, strip=False):
  content = open(file_path).read()
  if strip:
    return content.strip()
  return content


def WriteFile(file_path, content, mode='w'):
  with open(file_path, mode) as fp:
    fp.write(content)


def GenSshKey(user):
  """Generate ssh key for user.

  Args:
    user: string, the user to create the ssh key for.

  Returns:
    ret, out if capture_output=True.
  """
  key_name = 'daisy-test-key-' + str(uuid.uuid4())
  Execute(
      ['ssh-keygen', '-t', 'rsa', '-N', '', '-f', key_name, '-C', key_name])
  with open(key_name + '.pub', 'r') as original:
    data = original.read().strip()
  return "%s:%s" % (user, data), key_name


def RetryOnFailure(func):
  """Function decorator to retry on an exception."""

  @functools.wraps(func)
  def Wrapper(*args, **kwargs):
    ratio = 1.5
    wait = 3
    ntries = 0
    start_time = time.time()
    end_time = start_time + 15 * 60
    while time.time() < end_time:
      ntries += 1
      try:
        response = func(*args, **kwargs)
      except Exception as e:
        logging.info(str(e))
        logging.info(
            'Function %s failed, waiting %d seconds, retrying %d ...',
            str(func), wait, ntries)
        time.sleep(wait)
        wait = wait * ratio
      else:
        logging.info(
            'Function %s executed in less then %d sec, with %d tentative(s)',
            str(func), time.time() - start_time, ntries)
        return response
    raise
  return Wrapper


def ExecuteInSsh(
    key, user, machine, cmds, expect_fail=False, capture_output=False):
  """Execute commands through ssh.

  Args:
    key: string, the path of the private key to use in the ssh connection.
    user: string, the user used to connect through ssh.
    machine: string, the hostname of the machine to connect.
    cmds: list[string], the commands to be execute in the ssh session.
    expect_fail: bool, indicates if the failure in the execution is expected.
    capture_output: bool, indicates if the output of the command should be
        captured.

  Returns:
    ret, out if capture_output=True.
  """
  ssh_command = [
      'ssh', '-i', key, '-o', 'IdentitiesOnly=yes', '-o', 'ConnectTimeout=10',
      '-o', 'StrictHostKeyChecking=no', '-o', 'UserKnownHostsFile=/dev/null',
      '%s@%s' % (user, machine),
  ]
  ret, out = Execute(
      ssh_command + cmds, raise_errors=False, capture_output=capture_output)
  if expect_fail and ret == 0:
    raise ValueError('SSH command succeeded when expected to fail')
  elif not expect_fail and ret != 0:
    raise ValueError('SSH command failed when expected to succeed')
  else:
    return ret, out


def GetCompute(discovery, credentials):
  """Get google compute api cli object.

  Args:
    discovery: object, from googleapiclient.
    credentials: object, from google.auth.

  Returns:
    compute: object, the google compute api object.
  """
  compute = discovery.build('compute', 'v1', credentials=credentials)
  return compute


def RunTest(test_func):
  """Run main test function and print logging.success() or logging.error().

  Args:
    test_func: function, the function to be tested.
  """
  try:
    tracer = trace.Trace(
        ignoredirs=[sys.prefix, sys.exec_prefix], trace=1, count=0)
    tracer.runfunc(test_func)
    logging.success('Test finished.')
  except Exception as e:
    logging.error('error: ' + str(e))
    traceback.print_exc()


# Cache CLIENT and BUCKET to improve consecutive UploadFile calls
CLIENT = None
BUCKET = None


def UploadFile(filename, dest):
  # import 'google.cloud.storage' locally as 'google-cloud-storage' pip package
  # is not a mandatory package for all utils users
  from google.cloud import storage
  global CLIENT, BUCKET

  dest_stripped = dest[len('gs://'):]
  dest_splitted = dest_stripped.split('/')
  bucket_name, blob_path = dest_splitted[0], '/'.join(dest_splitted[1:])

  if not CLIENT:
    CLIENT = storage.client.Client()

  if not BUCKET or BUCKET.name != bucket_name:
    BUCKET = storage.bucket.Bucket(CLIENT, bucket_name)

  blob = storage.blob.Blob(blob_path, BUCKET)
  blob.upload_from_filename(filename)

  BUCKET.copy_blob(blob, BUCKET)


class LogFormatter(logging.Formatter):
  default_formatter = logging.Formatter('%(levelname)s:%(name)s:%(message)s')
  formatters = {}

  def __init__(self):
    prefix = GetMetadataParam('prefix', default_value='')
    prefix_level = {
        logging.DEBUG: '%sDebug: ' % prefix,
        logging.INFO: '%sStatus: ' % prefix,
        logging.WARNING: '%sWarn: ' % prefix,
        logging.ERROR: '%sFailed: ' % prefix,
        SUCCESS_LEVELNO: '%sSuccess: ' % prefix
    }
    for loglevel in prefix_level:
      self.formatters[loglevel] = logging.Formatter(
          prefix_level[loglevel] + '%(message)s')

  def format(self, record):
    formatter = self.formatters.get(record.levelno, self.default_formatter)
    return formatter.format(record)


def SetupLogging():
  """Configure Logging system."""
  logger = logging.getLogger()
  logger.setLevel(logging.DEBUG)
  stdout = logging.StreamHandler(sys.stdout)
  stdout.setLevel(logging.DEBUG)
  formatter = LogFormatter()
  stdout.setFormatter(formatter)
  logger.addHandler(stdout)
  logging.addLevelName(SUCCESS_LEVELNO, 'SUCCESS')

  def success(self, message, *args, **kws):
    self._log(SUCCESS_LEVELNO, message, args, **kws)
  logger.success = success
  logging.success = lambda *args: logging.log(SUCCESS_LEVELNO, *args)


SetupLogging()


class MetadataManager:
  """Utilities to manage metadata."""

  SSH_KEYS = 'ssh-keys'
  SSHKEYS_LEGACY = 'sshKeys'
  INSTANCE_LEVEL = 1
  PROJECT_LEVEL = 2

  def __init__(self, compute, instance, ssh_user='tester'):
    """Constructor.

    Args:
      compute: object, from GetCompute.
      instance: string, the instance to manage the metadata.
      user: string, the user to create ssh keys and perform ssh tests.
    """
    self.zone = self.FetchMetadataDefault('zone')
    self.region = self.zone[:-2]  # clears the "-[a-z]$" of the zone
    self.project = self.FetchMetadataDefault('project')
    self.compute = compute
    self.instance = instance
    self.last_fingerprint = None
    self.ssh_user = ssh_user
    self.md_items = {}
    md_obj = self._FetchMetadata(self.INSTANCE_LEVEL)
    self.md_items[self.INSTANCE_LEVEL] = (
        md_obj['items'] if 'items' in md_obj else [])
    md_obj = self._FetchMetadata(self.PROJECT_LEVEL)
    self.md_items[self.PROJECT_LEVEL] = (
        md_obj['items'] if 'items' in md_obj else [])

  def _FetchMetadata(self, level):
    """Fetch metadata from the server.

    Args:
      level: enum, INSTANCE_LEVEL or PROJECT_LEVEL to fetch the metadata.
    """
    if level == self.PROJECT_LEVEL:
      request = self.compute.projects().get(project=self.project)
      md_id = 'commonInstanceMetadata'
    else:
      request = self.compute.instances().get(
          project=self.project, zone=self.zone, instance=self.instance)
      md_id = 'metadata'
    response = request.execute()
    return response[md_id]

  @RetryOnFailure
  def StoreMetadata(self, level):
    """Store Metadata.

    Args:
      level: enum, INSTANCE_LEVEL or PROJECT_LEVEL to store the metadata.
    """
    md_obj = self._FetchMetadata(level)
    md_obj['items'] = self.md_items[level]
    if level == self.PROJECT_LEVEL:
      request = self.compute.projects().setCommonInstanceMetadata(
          project=self.project, body=md_obj)
    else:
      request = self.compute.instances().setMetadata(
          project=self.project, zone=self.zone, instance=self.instance,
          body=md_obj)
    response = request.execute()
    self.Wait(response)

  def ExtractKeyItem(self, md_key, level):
    """Extract a given key value from the metadata.

    Args:
      md_key: string, the key of the metadata value to be searched.
      level: enum, INSTANCE_LEVEL or PROJECT_LEVEL to fetch the metadata.

    Returns:
      md_item: dict, in the format {'key', md_key, 'value', md_value}.
      None: if md_key was not found.
    """
    for md_item in self.md_items[level]:
      if md_item['key'] == md_key:
        return md_item

  def SetMetadata(self, md_key, md_value, level=None, store=True):
    """Add or update a metadata key with a new value in a given level.

    Args:
      md_key: string, the key of the metadata.
      md_value: string, value to be added or updated.
      level: enum, INSTANCE_LEVEL (default) or PROJECT_LEVEL to fetch the
          metadata.
      store: bool, if True, saves metadata to GCE server.
    """
    if not level:
      level = self.INSTANCE_LEVEL
    md_item = self.ExtractKeyItem(md_key, level)
    if md_item and md_value is None:
      self.md_items[level].remove(md_item)
    elif not md_item:
      md_item = {'key': md_key, 'value': md_value}
      self.md_items[level].append(md_item)
    else:
      md_item['value'] = md_value
    if store:
      self.StoreMetadata(level)

  def AddSshKey(self, md_key, level=None, store=True):
    """Generate and add an ssh key to the metadata

    Args:
      md_key: string, SSH_KEYS or SSHKEYS_LEGACY, defines where to add the key.
      level: enum, INSTANCE_LEVEL (default) or PROJECT_LEVEL to fetch the
          metadata.
      store: bool, if True, saves metadata to GCE server.

    Returns:
      key_name: string, the name of the file with the generated private key.
    """
    if not level:
      level = self.INSTANCE_LEVEL
    key, key_name = GenSshKey(self.ssh_user)
    md_item = self.ExtractKeyItem(md_key, level)
    if not md_item:
      md_item = {'key': md_key, 'value': key}
      self.md_items[level].append(md_item)
    else:
      md_item['value'] = '\n'.join([md_item['value'], key])
    if store:
      self.StoreMetadata(level)
    return key_name

  def RemoveSshKey(self, key, md_key, level=None, store=True):
    """Remove an ssh key to the metadata

    Args:
      key: string, the key to be removed.
      md_key: string, SSH_KEYS or SSHKEYS_LEGACY, defines where to add the key.
      level: enum, INSTANCE_LEVEL (default) or PROJECT_LEVEL to fetch the
          metadata.
      store: bool, if True, saves metadata to GCE server.
    """
    if not level:
      level = self.INSTANCE_LEVEL
    md_item = self.ExtractKeyItem(md_key, level)
    md_item['value'] = re.sub('.*%s.*\n?' % key, '', md_item['value'])
    if not md_item['value']:
      self.md_items[level].remove(md_item)
    if store:
      self.StoreMetadata(level)

  @RetryOnFailure
  def TestSshLogin(self, key, expect_fail=False):
    """Try to login to self.instance using key.

    Args:
      key: string, the private key to be used in the ssh connection.
    """
    ExecuteInSsh(
        key, self.ssh_user, self.instance, ['echo', 'Logged'],
        expect_fail=expect_fail)

  @classmethod
  def FetchMetadataDefault(cls, name):
    """Fetch Metadata from default metadata server (local machine).

    Args:
      name: string, the metadata key to be fetched.

    Returns:
      value: the metadata value.
    """
    try:
      url = 'http://metadata/computeMetadata/v1/instance/attributes/%s' % name
      return HttpGet(url, headers={'Metadata-Flavor': 'Google'})
    except urllib2.HTTPError:
      raise ValueError('Metadata key "%s" not found' % name)

  def GetInstanceState(self, instance):
    """Get an instance state (e.g: RUNNING, TERMINATED, STOPPING...)

    Args:
      instance: string, the name of the instance to fetch its state.

    Returns:
      value: string, the status string.
    """
    request = self.compute.instances().get(
        project=self.project, zone=self.zone, instance=instance)
    return request.execute()[u'status']

  def StartInstance(self, instance):
    """Start an instance

    Args:
      instance: string, the name of the instance to be started.
    """
    self.compute.instances().start(
        project=self.project, zone=self.zone, instance=instance).execute()

  def ResizeDiskGb(self, disk_name, new_size):
    """Resize a disk to a new size. Note: Only allows size growing.

    Args:
      disk_name: string, the name of the disks to be resized.
      new_size: int, the new size in gigabytes to be resized
    """
    body = {'sizeGb': "%d" % new_size}
    request = self.compute.disks().resize(
        project=self.project, zone=self.zone, disk=disk_name, body=body)
    return request.execute()

  def AttachDisk(self, instance, disk_name):
    """Attach disk on instance.

    Args:
      instance: string, the name of the instance to attach disk.
      disk_name: string, the name of the disks to be attached.

    Returns:
      response: dict, the request's response.
    """
    body = {'source': 'projects/%s/zones/%s/disks/%s' % (
        self.project, self.zone, disk_name)}
    request = self.compute.instances().attachDisk(
        project=self.project, zone=self.zone, instance=instance, body=body)
    return request.execute()

  def GetDiskDeviceNameFromAttached(self, instance, disk_name):
    """Retrieve deviceName of an attached disk based on disk source name

    Args:
      instance: string, the name of the instance to detach disk.
      disk_name: string, the disk name to be compared to.
    """
    request = self.compute.instances().get(
        project=self.project, zone=self.zone, instance=instance)
    response = request.execute()
    for disk in response[u'disks']:
      if disk_name in disk[u'source']:
        return disk[u'deviceName']

  def DetachDisk(self, instance, device_name):
    """Detach disk on instance.

    Args:
      instance: string, the name of the instance to detach disk.
      device_name: string, the device name of the disk to be detached.

    Returns:
      response: dictionary, the request's response.
    """
    request = self.compute.instances().detachDisk(
        project=self.project, zone=self.zone, instance=instance,
        deviceName=device_name)
    return request.execute()

  def Wait(self, response):
    """Blocks until operation completes.
    Code from GitHub's GoogleCloudPlatform/python-docs-samples

    Args:
      response: dict, a request's response
    """
    def _OperationGetter(response):
      operation = response[u'name']
      if response.get(u'zone'):
        return self.compute.zoneOperations().get(
            project=self.project, zone=self.zone, operation=operation)
      elif response.get(u'region'):
        return self.compute.regionOperations().get(
            project=self.project, region=self.region, operation=operation)
      else:
        return self.compute.globalOperations().get(
            project=self.project, operation=operation)

    while True:
      result = _OperationGetter(response).execute()

      if result['status'] == 'DONE':
        if 'error' in result:
          raise Exception(result['error'])
        return result

      time.sleep(1)

  def GetForwardingRuleIP(self, name):
    """Retrieves a forwarding rule ip

    Args:
      name: string, the name of the forwarding rule.

    Returns:
      response: string, the forwarding rule ip.
    """
    request = self.compute.forwardingRules().get(
        project=self.project, region=self.region, forwardingRule=name)
    response = request.execute()
    return response[u"IPAddress"]
