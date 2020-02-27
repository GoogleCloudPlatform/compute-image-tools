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

$script:os_drive = ''
$script:components_dir = 'c:\components'

function Run-Command {
 [CmdletBinding(SupportsShouldProcess=$true)]
  param (
    [Parameter(Mandatory=$true, ValueFromPipelineByPropertyName=$true)]
      [string]$Executable,
    [Parameter(ValueFromRemainingArguments=$true,
               ValueFromPipelineByPropertyName=$true)]
      $Arguments = $null
  )
  Write-Output "Running $Executable with arguments $Arguments."
  $out = &$executable $arguments 2>&1 | Out-String
  $out.Trim()
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
      Write-Host "Failed to retrieve value for $key."
      return $null
    }
  }
}

function Setup-ScriptRunner {
  $metadata_scripts = "${script:os_drive}\Program Files\Google\Compute Engine\metadata_scripts"
  New-Item "${metadata_scripts}\"  -Force -ItemType Directory | Out-Null
  Copy-Item "${script:components_dir}\run_startup_scripts.cmd" "${metadata_scripts}\run_startup_scripts.cmd" -Verbose
  # This file must be unicode with no trailing new line and exactly match the source.
  (Get-Content "${script:components_dir}\GCEStartup" | Out-String).TrimEnd() | Out-File -Encoding Unicode -NoNewline "${script:os_drive}\Windows\System32\Tasks\GCEStartup"

  Run-Command reg load 'HKLM\MountedSoftware' "${script:os_drive}\Windows\System32\config\SOFTWARE"
  $dst = 'HKLM:\MountedSoftware\Microsoft\Windows NT\CurrentVersion\Schedule\TaskCache'

  $acl = Get-ACL $dst
  $admin = New-Object System.Security.Principal.NTAccount('Builtin', 'Administrators')
  $acl.SetOwner($admin)
  $ace = New-Object System.Security.AccessControl.RegistryAccessRule(
    $admin,
    [System.Security.AccessControl.RegistryRights]'FullControl',
    [System.Security.AccessControl.InheritanceFlags]::ObjectInherit,
    [System.Security.AccessControl.PropagationFlags]::InheritOnly,
    [System.Security.AccessControl.AccessControlType]::Allow
  )
  $acl.AddAccessRule($ace)
  Set-Acl -Path $dst -AclObject $acl

  & reg import "${script:components_dir}\GCEStartup.reg"
  # Garbage collect before unmounting.
  [gc]::collect()

  Run-Command reg unload 'HKLM\MountedSoftware'
}

function Test-ProductName {
  Run-Command reg load 'HKLM\MountedSoftware' "${script:os_drive}\Windows\System32\config\SOFTWARE"
  $pn_path = 'HKLM:\MountedSoftware\Microsoft\Windows NT\CurrentVersion'
  $pn = (Get-ItemProperty -Path $pn_path -Name ProductName).ProductName
  Write-Output "Product Name: ${pn}"
  Run-Command reg unload 'HKLM\MountedSoftware'
  $product_name = Get-MetadataValue -key 'product_name'
  Write-Output "*${product_name}*"
  if ($pn -like "*${product_name}*") {
    Write-Output 'TranslateBootstrap: Product and import workflow match.'
  }
  else {
    Write-Output "TranslateBootstrap: Incorrect translate workflow selected. Found: $pn, Expected: $product_name."
  }
}

function Copy-32bitPackages {
  Write-Output 'TranslateBootstrap: Creating directory.'
  $googet_dir = "${script:os_drive}\ProgramData\GooGet"
  New-Item -Path $googet_dir -Force -Type Directory
  Write-Output 'TranslateBootstrap: Copying googet.'
  Copy-Item "${script:components_dir}\googet.exe" "${script:os_drive}\ProgramData\GooGet\googet.exe" -Force -Verbose
  Write-Output 'TranslateBootstrap: Copying additional googet files.'
  $goofiles_dir = "${script:os_drive}\ProgramData\GooGet\components\"
  New-Item -Path $goofiles_dir -Force -Type Directory
  Copy-Item "${script:components_dir}\*.goo" "${script:os_drive}\ProgramData\GooGet\components\" -Force -Verbose -Recurse
}

try {
  Write-Output 'TranslateBootstrap: Beginning translation bootstrap powershell script.'
  $script:is_x86 = Get-MetadataValue -key 'is_x86'

  $partition_style = Get-Disk 1 | Select-Object -Expand PartitionStyle
  Get-Disk | Where-Object -Property OperationalStatus -EQ "Offline" | Set-Disk -IsOffline $false
  Get-Disk 1 | Get-Partition | ForEach-Object {
    if (-not $_.DriveLetter) {
      # Ensure all available partitions on the import disk are accessible via drive letter.
      Write-Output "Assigning drive letter to partition #$($_.PartitionNumber)"
      Add-PartitionAccessPath -DiskNumber 1 -PartitionNumber $_.PartitionNumber -AssignDriveLetter -ErrorAction SilentlyContinue
    }
  }

  $bcd_drive = ''
  Get-Disk 1 | Get-Partition | ForEach-Object {
    if (Test-Path "$($_.DriveLetter):\Windows") {
      $script:os_drive = "$($_.DriveLetter):"
    }
    elseif (Test-Path "$($_.DriveLetter):\Boot\BCD") {
      $bcd_drive = "$($_.DriveLetter):"
    }
  }
  if (!$bcd_drive) {
    $bcd_drive = $script:os_drive
  }
  if (!$script:os_drive) {
    $partitions = Get-Disk 1 | Get-Partition
    throw "No Windows folder found on any partition: $partitions"
  }
  Write-Output "Detected BCD folder drive letter: ${bcd_drive}"
  Write-Output "Detected Windows folder drive letter: ${script:os_drive}"

  $kernel32_ver = (Get-Command "${script:os_drive}\Windows\System32\kernel32.dll").Version
  $os_version = "$($kernel32_ver.Major).$($kernel32_ver.Minor)"

  $version = Get-MetadataValue -key 'version'
  if ($version -ne $os_version) {
    throw "Incorrect Windows version to translate, mounted image is $os_version, not $version"
  }

  if ($script:is_x86.ToLower() -eq 'true') {
    # For 32-bit image imports, test a new method to verify image matches the selected workflow.
    Test-ProductName
  }

  $driver_dir = 'c:\drivers'
  New-Item $driver_dir -Type Directory | Out-Null
  New-Item $script:components_dir -Type Directory | Out-Null

  $daisy_sources = Get-MetadataValue -key 'daisy-sources-path'

  Write-Output 'TranslateBootstrap: Pulling components.'
  & 'gsutil' -m cp -r "${daisy_sources}/components/*" $script:components_dir

  Write-Output 'TranslateBootstrap: Pulling drivers.'
  & 'gsutil' -m cp -r "${daisy_sources}/drivers/*" $driver_dir

  Copy-Item "${driver_dir}\netkvmco.dll" "${script:os_drive}\Windows\System32\netkvmco.dll" -Verbose

  Write-Output 'TranslateBootstrap: Slipstreaming drivers.'
  if ($script:is_x86.ToLower() -ne 'true') {
    Add-WindowsDriver -Path "${script:os_drive}\" -Driver $driver_dir -Recurse -Verbose
  }
  else {
    Run-Command DISM /Image:$script:os_drive /Add-Driver /Driver:$script:driver_dir
  }

  Write-Output 'TranslateBootstrap: Setting up script runner.'
  Setup-ScriptRunner

  if ($script:is_x86.ToLower() -ne 'true') {
    Write-Output 'Setting up cloud repo.'
    Run-Command 'C:\ProgramData\GooGet\googet.exe' -root "${script:os_drive}\ProgramData\GooGet" addrepo 'google-compute-engine-stable' 'https://packages.cloud.google.com/yuck/repos/google-compute-engine-stable'
    Write-Output 'Copying googet.'
    Copy-Item 'C:\ProgramData\GooGet\googet.exe' "${script:os_drive}\ProgramData\GooGet\googet.exe" -Force -Verbose
  }
  else {
    Copy-32bitPackages
  }
  if ($partition_style -eq "MBR") {
    Write-Output 'MBR partition detected. Resetting bootloader.'
    Run-Command bcdboot "${script:os_drive}\Windows" /s $bcd_drive
  }
  else {
    Write-Output 'GPT partition detected.'
  }

  # Turn off startup animation which breaks headless installation.
  # See http://support.microsoft.com/kb/2955372/en-us
  Run-Command reg load 'HKLM\MountedSoftware' "${script:os_drive}\Windows\System32\config\SOFTWARE"
  Run-Command reg add 'HKLM\MountedSoftware\Microsoft\Windows\CurrentVersion\Authentication\LogonUI' /v 'AnimationDisabled' /t 'REG_DWORD' /d 1 /f
  Run-Command reg unload 'HKLM\MountedSoftware'

  Write-Output 'TranslateBootstrap: Rewriting boot files.'
  Write-Output 'Translate bootstrap complete.'
}
catch {
  Write-Output 'Exception caught in script:'
  Write-Output $_.InvocationInfo.PositionMessage
  Write-Output "TranslateFailed: $($_.Exception.Message)"
  exit 1
}
