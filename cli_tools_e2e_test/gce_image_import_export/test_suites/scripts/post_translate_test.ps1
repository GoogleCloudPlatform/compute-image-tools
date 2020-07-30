#  Copyright 2017 Google Inc. All Rights Reserved.
#
#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.

$ErrorActionPreference = 'Stop'

function Check-VMWareTools {
  Get-ChildItem HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall | Foreach-Object {
    if ((Get-ItemProperty $_.PSPath).DisplayName -eq 'VMWare Tools') {
      throw 'VMWare tools should not be installed'
    }
  }
}

function Check-MetadataAccessibility {
  @('metadata', 'metadata.google.internal') | ForEach-Object {
    if (-not (Test-Connection $_ -Count 1)) {
      throw "Failed to connect to $_"
    }
  }
}

function Get-MetadataValue {
  param (
    [parameter(Mandatory=$true)]
      [string]$key,
    [parameter(Mandatory=$false)]
      [string]$default
  )

  # Returns the provided metadata value for a given key.
  $url = "http://metadata.google.internal/computeMetadata/v1/instance/attributes/$key"
  try {
    $client = New-Object Net.WebClient
    $client.Headers.Add('Metadata-Flavor', 'Google')
    return ($client.DownloadString($url)).Trim()
  }
  catch [System.Net.WebException] {
    if ($default) {
      return $default
    }
    else {
      Write-Output "Failed to retrieve value for $key."
      return $null
    }
  }
}

function Check-Activation {
  # Force activation as this is on a timer.
  & cscript c:\windows\system32\slmgr.vbs /ato

  $out = & cscript C:\Windows\system32\slmgr.vbs /dli
  Write-Output $out
  if ($out -notcontains 'License Status: Licensed') {
    Write-Output 'Windows is not activated'
    return $false
  }

  if ($out -notcontains '    Registered KMS machine name: kms.windows.googlecloud.com:1688') {
    Write-Output  'Windows is not activated against GCE kms server'
    return $false
  }

  return $true
}

function Check-SkipActivation {
  $kms_name = (Get-ItemProperty 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\SoftwareProtectionPlatform').KeyManagementServiceName
  if ($kms_name -ne $null) {
    throw "KMS server set when it should be blank: $kms_name"
  }
}

try {
  $byol = Get-MetadataValue -key 'byol' -default 'false'

  Write-Output 'STATUS: GCEAgent Service'
  Get-Service GCEAgent
  Write-Output 'STATUS: Check-VMWareTools'
  Check-VMWareTools
  Write-Output 'STATUS: Check-MetadataAccessibility'
  Check-MetadataAccessibility
  if ($byol.ToLower() -eq 'true') {
    Write-Output 'STATUS: Check-SkipActivation'
    Check-SkipActivation
  }
  else {
    Write-Output 'STATUS: Check-Activation'
    $activated = $false
    for ($i = 0; $i -le 10; $i += 1) {
      $activated = Check-Activation
      if ($activated) {
        break
      }
      Start-Sleep -s 10
    }
    if (!$activated) {
      throw 'Activation failed'
    }
  }
  Write-Output 'PASSED: All Tests Passed!'
}
catch {
  Write-Output 'Exception caught in script:'
  Write-Output $_.InvocationInfo.PositionMessage
  Write-Output "FAILED: $($_.Exception.Message)"
}
