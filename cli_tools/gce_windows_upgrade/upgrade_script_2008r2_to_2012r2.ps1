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

$script:install_media_drive = ''

try {
  Write-Host 'GCEMetadataScripts: Beginning upgrade startup script.'

  # Cleanup garbage files left by the previous failed upgrade to unblock a new upgrade.
  Remove-Item 'C:\$WINDOWS.~BT' -Recurse -ErrorAction SilentlyContinue

  $ver=[System.Environment]::OSVersion.Version
  if ("$($ver.Major).$($ver.Minor)" -ne "6.1") {
    if ("$($ver.Major).$($ver.Minor)" -eq "6.3") {
      Write-Host "GCEMetadataScripts: The instance is already running Windows 2012R2!"
      Write-Host "GCEMetadataScripts: Rerunning upgrade.ps1 as the post-upgrade step."
    } else {
      throw "The instance is not running Windows 2008R2 (6.1). It is $($new_ver.Major).$($new_ver.Minor)."
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
    if (Test-Path "$($Drive.DeviceID)\*2012_R2_64Bit*") {
      $script:install_media_drive = "$($Drive.DeviceID)"
    }
  }
  if (!$script:install_media_drive) {
    throw "No install media found."
  }
  Write-Host "Detected install media folder drive letter: $script:install_media_drive"

  # Run upgrade script from the install media.
  Set-ExecutionPolicy Unrestricted
  Set-Location "$($script:install_media_drive)/*2012_R2_64Bit*"
  ./upgrade.ps1
  $new_ver=[System.Environment]::OSVersion.Version
  if ("$($ver.Major).$($ver.Minor)" -eq "6.3")
  {
    Write-Host "GCEMetadataScripts: post-upgrade step is done."
  }
  Write-Host "windows_upgrade_current_version=$($new_ver.Major).$($new_ver.Minor)"
}
catch {
  Write-Host "UpgradeFailed: $($_.Exception.Message)"
  exit 1
}
