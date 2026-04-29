#  Copyright 2020 Google Inc. All Rights Reserved.
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

function Get-MetadataValue {
  param (
    [parameter(Mandatory=$true)]
    [string]$key
  )

  # Returns the provided metadata value for a given key.
  $url = "http://metadata.google.internal/computeMetadata/v1/instance/attributes/$key"
  $max_attemps = 5
  for ($i=0; $i -le $max_attemps; $i++) {
    try {
      $client = New-Object Net.WebClient
      $client.Headers.Add('Metadata-Flavor', 'Google')
      $value = ($client.DownloadString($url)).Trim()
      Write-Host "Retrieved metadata for key $key with value $value."
      return $value
    }
    catch [System.Net.WebException] {
      # Sleep after each failure with no default value to give the network adapters time to become functional.
      Start-Sleep -s 1
    }
  }
  throw "Failed $max_attemps times to retrieve value from metadata for $key."
}

$script:install_media_drive = ''

try {
  Write-Host 'Beginning upgrade startup script.'

  $script:expected_current_version = Get-MetadataValue -key 'expected-current-version'
  $script:expected_new_version = Get-MetadataValue -key 'expected-new-version'
  $script:install_folder = Get-MetadataValue -key 'install-folder'

  # Cleanup garbage files left by the previous failed upgrade to unblock a new upgrade.
  Remove-Item 'C:\$WINDOWS.~BT' -Recurse -ErrorAction SilentlyContinue

  $ver=(Get-ItemProperty "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion").ProductName
  if ($ver -ne $script:expected_current_version) {
    if ($ver -eq $script:expected_new_version) {
      Write-Host "The instance is already running $script:expected_new_version!"
      Write-Host "Rerunning upgrade.ps1 as the post-upgrade step."
    } else {
      throw "The instance is not running $script:expected_current_version. It is '$ver'."
    }
  }

  # Bring all disks online to ensure install media is accessible.
  $Disks = Get-WmiObject Win32_DiskDrive
  foreach ($Disk in $Disks)
  {
    $DiskID = $Disk.index
    $DiskPartScript = @"
select disk $DiskID
online disk noerr
"@
    $DiskPartScript | diskpart
  }

  # Find the drive which contains install media.
  $Drives = Get-WmiObject Win32_LogicalDisk
  ForEach ($Drive in $Drives) {
    if (Test-Path "$($Drive.DeviceID)\$script:install_folder") {
      $script:install_media_drive = "$($Drive.DeviceID)"
    }
  }
  if (!$script:install_media_drive) {
    throw "No install media found."
  }
  Write-Host "Detected install media folder drive letter: $script:install_media_drive"

  # Run upgrade script from the install media.
  Set-ExecutionPolicy Unrestricted
  Set-Location "$($script:install_media_drive)/$script:install_folder"
  ./upgrade.ps1
  $new_ver=(Get-ItemProperty "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion").ProductName
  if ($new_ver -eq $script:expected_new_version)
  {
    Write-Host "post-upgrade step is done."
  }
  Write-Host "windows_upgrade_current_version='$new_ver'"
}
catch {
  Write-Host "UpgradeFailed: $($_.Exception.Message)"
  exit 1
}
