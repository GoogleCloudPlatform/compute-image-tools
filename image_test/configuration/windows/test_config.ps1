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

$script:errors = @()
Import-Module 'C:\Program Files\Google\Compute Engine\sysprep\gce_base.psm1' -ErrorAction Stop

function Test-Activation {
  Write-Host 'Running' $MyInvocation.MyCommand
  $status = cscript C:\Windows\system32\slmgr.vbs /dli
  if ($status -notcontains 'License Status: Licensed') {
    Write-Host 'Image does not appear to be properly licensed:'
    Write-Host $status
    $script:errors += $MyInvocation.MyCommand
  }
}

function Test-MTU {
  Write-Host 'Running' $MyInvocation.MyCommand
  $want = 1460
  $interface = (Get-CimInstance Win32_NetworkAdapter -filter "ServiceName='netkvm'")[0]
  $result = netsh interface ipv4 show subinterface $interface.NetConnectionID
  $result[3] -match '^\s*(\d+)' | Out-Null
  $mtu = $Matches[1]
  if ($mtu -ne $want) {
    Write-Host "Improper MTU set: ${mtu}"
    Write-Host $result
    $script:errors += $MyInvocation.MyCommand
  }
}

function Test-PowershellVersion {
  Write-Host 'Running' $MyInvocation.MyCommand
  $want = '5.1'
  $version = "$($PSVersionTable.PSVersion.Major).$($PSVersionTable.PSVersion.Minor)"
  if ($version -lt $want) {
    Write-Host "Improper Powershell version installed: ${version}, want >= ${want}"
    $script:errors += $MyInvocation.MyCommand
  }
}

function Test-DotNetVersion {
  Write-Host 'Running' $MyInvocation.MyCommand
  $want = '4.7'
  $version = Get-ItemProperty 'HKLM:\SOFTWARE\Microsoft\NET Framework Setup\NDP\v4\Full' -Name Version | Select-Object -ExpandProperty Version
  if ($version -lt $want) {
    Write-Host "Improper .Net version installed: ${version}, want >= ${want}"
    $script:errors += $MyInvocation.MyCommand
  }
}

function Test-NTP {
  Write-Host 'Running' $MyInvocation.MyCommand
  w32tm /resync | Out-Null
  $ntp = w32tm /query /peers /verbose
  @('#Peers: 1', 
    'Peer: metadata.google.internal,0x1', 
    'State: Active',
    'LastSyncErrorMsgId: 0x00000000 (Succeeded)') | Foreach-Object {
      if ($ntp -notcontains $_) {
        Write-Host "NTP not setup correctly, $_ not in:"
        Write-Host $ntp
        $script:errors += $MyInvocation.MyCommand
        return
      }
    }

    $($ntp -match 'Time Remaining') -match '([0-9\.]+)s' | Out-Null
    if (15.0 -gt $Matches[1]) {
      Write-Host 'NTP not setup correctly, Time remaining is longer than the 15 minute poll interval:'
      Write-Host $ntp
      $script:errors += $MyInvocation.MyCommand
    }
}

function Test-EMSEnabled {
  Write-Host 'Running' $MyInvocation.MyCommand
  $bcd = bcdedit 
  if (-not ($bcd | Select-String -Quiet -Pattern "ems * Yes")) {
    Write-Host 'EMS does not appear to be enabled, result of bcdedit command:'
    Write-Host $bcd
    $script:errors += $MyInvocation.MyCommand
  }
}

function Test-TimeZone {
  Write-Host 'Running' $MyInvocation.MyCommand
  $timezone = (Get-CimInstance Win32_OperatingSystem).CurrentTimeZone
  if ($timezone -ne 0) {
    Write-Host "Improper timezone ${timezone}, want 0"
    $script:errors += $MyInvocation.MyCommand
  }
}

function Test-Hostname {
  Write-Host 'Running' $MyInvocation.MyCommand
  $hostname = hostname
  $hostname_parts = (Get-Metadata -property 'hostname') -split '\.'
  $want = $hostname_parts[0]
  if ($hostname -ne $want) {
    Write-Host "Improper hostname ${hostname}, want ${want}"
    $script:errors += $MyInvocation.MyCommand
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

if ($script:errors) {
  Write-Host 'TestFailed, the following tests failed:' $errors
  exit 1
}

Write-Host TestSuccess
