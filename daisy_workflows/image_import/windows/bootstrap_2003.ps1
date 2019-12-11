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
$script:driver_dir = 'c:\drivers'

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

try {
  Write-Output 'TranslateBootstrap: Beginning translation bootstrap powershell script.'

  Get-Disk 1 | Get-Partition | ForEach-Object {
    if (Test-Path "$($_.DriveLetter):\Windows") {
      $script:os_drive = "$($_.DriveLetter):"
    }
  }
  if (!$script:os_drive) {
    $partitions = Get-Disk 1 | Get-Partition
    throw "No Windows folder found on any partition: $partitions"
  }

  $kernel32_ver = (Get-Command "${script:os_drive}\Windows\System32\kernel32.dll").Version
  $os_version = "$($kernel32_ver.Major).$($kernel32_ver.Minor)"

  $version = Get-MetadataValue -key 'version'
  if ($version -ne $os_version) {
    throw "Incorrect Windows version to translate, mounted image is $os_version, not $version"
  }

  New-Item $script:driver_dir -Type Directory | Out-Null
  New-Item $script:components_dir -Type Directory | Out-Null

  $daisy_sources = Get-MetadataValue -key 'daisy-sources-path'

  Write-Output 'TranslateBootstrap: Pulling components.'
  & 'gsutil' -m cp -r "${daisy_sources}/components/*" $script:components_dir

  Write-Output 'TranslateBootstrap: Pulling drivers.'
  & 'gsutil' -m cp -r "${daisy_sources}/drivers/*" $driver_dir
 
  # Setup Agent
  New-Item -Type Directory "${script:os_drive}\Program Files\Google\Compute Engine\agent"
  Copy-Item "${script:driver_dir}\x86_agent\GCEWindowsAgent.exe" "${script:os_drive}\Program Files\Google\Compute Engine\agent\"
  New-Item -Type Directory "${script:os_drive}\Program Files\Google\Compute Engine\metadata_scripts"

  # Copy script runner, we do not enable it
  Copy-Item "${script:driver_dir}\x86_agent\GCEMetadataScripts.exe" "${script:os_drive}\Program Files\Google\Compute Engine\metadata_scripts\"
  Copy-Item "${script:driver_dir}\x86_agent\run_startup_scripts.cmd" "${script:os_drive}\Program Files\Google\Compute Engine\metadata_scripts\"

  $arch = 'x86'
  if (Test-Path "${script:os_drive}\Program Files (x86)") {
    $arch = 'x64'
  }
  Copy-Item "${script:driver_dir}\${arch}\*.sys" "${script:os_drive}\Windows\system32\drivers\"
  Copy-Item "${script:driver_dir}\${arch}\*.inf" "${script:os_drive}\Windows\inf\"

  Run-Command reg load HKLM\msystem "${script:os_drive}\Windows\system32\config\system"
  Run-Command reg load HKLM\msoftware "${script:os_drive}\Windows\system32\config\software"
  & reg import "${script:components_dir}\agent.reg"
  & reg import "${script:components_dir}\shutdown.reg"
  & reg import "${script:components_dir}\drivers.reg"
  Run-Command reg unload HKLM\msystem
  Run-Command reg unload HKLM\msoftware

  # Replace boot.ini
  Remove-Item "${script:os_drive}\boot.ini" -Force
  Copy-Item "${script:components_dir}\boot.ini" "${script:os_drive}\boot.ini" -Force

  Write-Output 'Translate bootstrap complete'
}
catch {
  Write-Output 'Exception caught in script:'
  Write-Output $_.InvocationInfo.PositionMessage
  Write-Output "TranslateFailed: $($_.Exception.Message)"
  exit 1
}
