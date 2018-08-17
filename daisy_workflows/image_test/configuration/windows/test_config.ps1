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

$errors = @()

function Test-Activation {
  Write-Host "Running " + $MyInvocation.MyCommand
}

function Test-MTU {
  Write-Host "Running " + $MyInvocation.MyCommand
  $want = 1460
  $interface = (Get-CimInstance Win32_NetworkAdapter -filter "ServiceName='netkvm'")[0]
  $result = netsh interface ipv4 show subinterface $interface.NetConnectionID
  $result[3] -match '^\s*(\d+)'
  $mtu = $Matches[1]
  if ($mtu -ne $want) {
    Write-Host "Improper MTU set: ${mtu}"
    Write-Host $result
    $errors += $MyInvocation.MyCommand
  }
}

function Test-PowershellVersion {
  Write-Host "Running " + $MyInvocation.MyCommand
  $want = "5.1"
  $version = "$($PSVersionTable.PSVersion.Major).$($PSVersionTable.PSVersion.Minor)"
  if ($version -ge $want) {
    Write-Host "Improper Powershell version installed: ${version}, want >= ${want}"
    $errors += $MyInvocation.MyCommand
  }
}

function Test-DotNetVersion {
  Write-Host "Running " + $MyInvocation.MyCommand
  $want = "4.7"
  $version = Get-ItemProperty "HKLM:\SOFTWARE\Microsoft\NET Framework Setup\NDP\v4\Full" -Name Version | Select-Object -ExpandProperty Version
  if ($version -ge $want) {
    Write-Host "Improper .Net version installed: ${version}, want >= ${want}"
    $errors += $MyInvocation.MyCommand
  }
}

function Test-NTP {
  Write-Host "Running " + $MyInvocation.MyCommand
}

function Test-EMSEnabled {
  Write-Host "Running " + $MyInvocation.MyCommand
  $bcd = bcdedit 
  if (-not ($bcd | Select-String -Quiet -Pattern "ems * Yes")) {
    Write-Host "EMS does not appear to be enabled, result of bcdedit command:"
    Write-Host $bcd
    $errors += $MyInvocation.MyCommand
  }
}

function Test-TimeZone {
  Write-Host "Running " + $MyInvocation.MyCommand
  $timezone = (Get-CimInstance Win32_OperatingSystem).CurrentTimeZone
  if ($timezone -ne 0) {
    Write-Host "Improper timezone ${timezone}, want 0"
    $errors += $MyInvocation.MyCommand
  }
}

function Test-Hostname {
  Write-Host "Running " + $MyInvocation.MyCommand
  $hostname = hostname
  $want = "foo"
  if ($hostname -ne $want) {
    Write-Host "Improper hostname ${hostname}, want ${want}"
    $errors += $MyInvocation.MyCommand
  }
}

Test-MTU
Test-PowershellVersion
Test-DotNetVersion
Test-NTP
Test-EMSEnabled
Test-TimeZone
Test-Hostname
# Test activation last in order to give plenty of time for it to run in the 
# background
Test-Activation

if ($errors.Length -eq 0) {
  Write-Host TestSuccess
  exit 0
}

$msg = "TestFailure, the following tests failed: " + $errors -join ", "


<#

  def testActivationStatus(self):
    windows_test_utils.GetCommandOutputOrRaiseOnFailure(
        self.vm, r'cscript C:\Windows\system32\slmgr.vbs /ato')
    activation_info = windows_test_utils.GetCommandOutputOrRaiseOnFailure(
        self.vm, r'cscript C:\Windows\system32\slmgr.vbs /dli')
    expected_info = []

    # Check for KMS activation. Internal images do not get licensed from KMS.
    if FLAGS.clustermanager == 'prod':
      if (windows_test_utils.IsWinInternal(self.vm) and
          windows_test_utils.IsWin2008R2(self.vm)):
        expected_info.append('License Status: Initial grace period')
      elif (windows_test_utils.IsWinInternal(self.vm) and
            windows_test_utils.IsWin2012R2(self.vm)):
        expected_info.append('License Status: Notification')
      else:
        expected_info.append('License Status: Licensed')

    for s in expected_info:
      self.assertIn(
          s, activation_info,
          'activation_info %s does not contain "%s"' % (
              activation_info, s))

  def testPowershellVersion(self):
    ps_version = windows_test_utils.GetCommandOutputOrRaiseOnFailure(
        self.vm, ('"$($PSVersionTable.PSVersion.Major).'
                  '$($PSVersionTable.PSVersion.Minor)"'))
    expected = ('4.0' if 'server-2008-r2' in FLAGS.vm_image_path else '5.1')
    self.assertGreaterEqual(
        ps_version, expected,
        'PowerShell version %s is not at least version %s.' %
        (ps_version, expected))

  def testNtp(self):
    windows_test_utils.RunCommandOrRaiseOnFailure(self.vm, 'w32tm /resync')
    peer_info = windows_test_utils.GetCommandOutputOrRaiseOnFailure(
        self.vm, 'w32tm /query /peers /verbose')
    for expected_info in ['#Peers: 1', 'Peer: metadata.google.internal,0x1',
                          'State: Active',
                          'LastSyncErrorMsgId: 0x00000000 (Succeeded)']:
      self.assertIn(expected_info, peer_info,
                    'expected_info (%s) is NOT in peer_info (%s)' % (
                        expected_info, peer_info))
    match = re.search(r'Time Remaining: ([0-9\.]+)s', peer_info)
    self.assertIsNotNone(match, 'Couldn\'t find time remaining in output.')
    remaining = float(match.group(1))
    self.assertGreater(remaining, 0.0, 'Invalid time remaining.')
    # Next NTP sync should be less than 15 minutes away.
    self.assertLess(remaining, 900.0, 'Time remaining is longer '
                    'than the 15 minute poll interval.')

  @skip_annotations.SkipTestInEnvironments(
      ['dev'], reason_msg='Cannot ping externally in dev on Guitar.')
  def testNetworkConnection(self):
    """Do basic network test (connect to www.google.com)."""
    result = windows_test_utils.GetCommandOutputOrRaiseOnFailure(
        self.vm,
        'Test-Connection www.google.com -Count 1 -ErrorAction stop -quiet')
    self.assertIn('True', result)

#>