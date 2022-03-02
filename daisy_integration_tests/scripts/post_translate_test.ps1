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

function Check-Hiberation {
  $HibernateEnabled = (Get-ItemProperty -Path "HKLM:\SYSTEM\CurrentControlSet\Control\Power" -ErrorAction SilentlyContinue).HibernateEnabled
  if ($HibernateEnabled -eq $null -or $HibernateEnabled -ne 0) {
    throw "Hibernation not disabled. HKLM:\SYSTEM\CurrentControlSet\Control\Power\HibernateEnabled = $HibernateEnabled"
  }
  $HibernateInfo = New-Object System.Diagnostics.ProcessStartInfo
  $HibernateInfo.FileName = "shutdown"
  $HibernateInfo.RedirectStandardError = $true
  $HibernateInfo.UseShellExecute = $false
  $HibernateInfo.Arguments = "/h"
  $Hibernate = New-Object System.Diagnostics.Process
  $Hibernate.StartInfo = $HibernateInfo
  $Hibernate.Start() | Out-Null
  $Hibernate.WaitForExit()
  $stderr = $Hibernate.StandardError.ReadToEnd()
  if ($stderr -notlike "*Hibernation is not enabled*") {
    throw "Unexpected response when attempting to place the system into hibernation. StdErr should contain 'Hibernation is not enabled', stderr=$stderr"
  }
}

function Check-Power-Settings {
  try {
    $pplan = Get-WMIObject `
                  -Namespace 'root\cimv2\power' `
                  -Class Win32_PowerSetting `
                  -ErrorAction SilentlyContinue `
                  | where {$_.ElementName -eq 'Turn off display after'}
  } catch {
    Write-Output "Get-WMIObject Win32_PowerSetting failed: " + $_
    return
  }


  $pplan | ForEach-Object {
    $_.GetRelated('Win32_PowerSettingDataIndex') | ForEach-Object {
      $powerOffSeconds=$_.SettingIndexValue
      if ($powerOffSeconds -ne 0) {
        throw "Turn off display: expected=0, actual=$powerOffSeconds"
      }
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

function Check-OSConfigAgent {
  # To disable checking for the OS config agent, add an instance metadata
  # value of osconfig_not_supported: true.
  $osconfig_not_supported = Get-MetadataValue -key 'osconfig_not_supported' -default 'false'
  if ($osconfig_not_supported.ToLower() -ne 'true') {
    Write-Output 'Test: OS Config agent'
    Get-Service google_osconfig_agent
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

  Write-Output 'Test: GCEAgent Service'
  Get-Service GCEAgent
  Write-Output 'Test: Check-VMWareTools'
  Check-VMWareTools
  Write-Output 'Test: Check-MetadataAccessibility'
  Check-MetadataAccessibility
  Write-Output 'Test: Check-OSConfigAgent'
  Check-OSConfigAgent
  Write-Output 'Test: Check-Hiberation'
  Check-Hiberation
  Write-Output 'Test: Check-Power-Settings'
  Check-Power-Settings
  if ($byol.ToLower() -eq 'true') {
    Write-Output 'Test: Check-SkipActivation'
    Check-SkipActivation
  }
  else {
    Write-Output 'Test: Check-Activation'
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
  Write-Output 'PASSED: All Tests Passed'
}
catch {
  Write-Output 'Exception caught in script:'
  Write-Output $_.InvocationInfo.PositionMessage
  Write-Output "Test Failed: $($_.Exception.Message)"
}
