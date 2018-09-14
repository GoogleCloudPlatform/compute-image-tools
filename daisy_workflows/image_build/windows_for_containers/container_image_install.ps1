# Copyright 2017 Google Inc. All Rights Reserved.
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

# This script prepares a Windows Server version 1803 image to run Windows
# Server containers by following the steps in Microsoft's documentation:
# https://docs.microsoft.com/en-us/virtualization/windowscontainers/quick-start/quick-start-windows-server.
# These steps use the DockerMsftProvider OneGet module
# (https://github.com/OneGet/MicrosoftDockerProvider), which installs Docker
# Enterprise Edition (not "Docker for Windows", which is a distribution of
# Docker Community Edition meant for Windows client installations).

$ErrorActionPreference = 'Stop'

function Get-MetadataValue {
  param (
    [parameter(Mandatory=$true)]
      [string]$key,
    [parameter(Mandatory=$false)]
      [string]$default
  )

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

& googet -noconfirm update
try {
  $version = Get-MetadataValue 'version'
  if (-not $version) {
    throw 'Error retrieving "version" from metadata'
  }

  if (!(Get-Package -ProviderName DockerMsftProvider -ErrorAction SilentlyContinue| Where-Object {$_.Name -eq 'docker'})) {
    Write-Host 'Installing NuGet module'
    Install-PackageProvider -Name NuGet -MinimumVersion 2.8.5.201 -Force

    Write-Host 'Installing DockerMsftProvider module'
    Install-Module -Name DockerMsftProvider -Repository PSGallery -Force

    Write-Host 'Installing Docker EE 18.03'
    Install-Package -Name docker -ProviderName DockerMsftProvider -Force -RequiredVersion 18.03

    Write-Host 'Enabling IPv6'
    $ipv_path = 'HKLM:\SYSTEM\CurrentControlSet\services\TCPIP6\Parameters'
    Set-ItemProperty -Path $ipv_path -Name 'DisabledComponents' -Value 0x0

    Write-Host 'Restarting computer to finish install'
    Restart-Computer -Force
  }
  else {
    # For some reason the docker service may not be started automatically on the
    # first reboot, although it seems to work fine on subsequent reboots. The
    # docker service must be running or else the vEthernet interface may not be
    # present.
    Restart-Service docker

    Write-Host 'Setting host vEthernet MTU to 1460'
    Get-NetAdapter | Where-Object {$_.Name -like 'vEthernet*'} | ForEach-Object {
      & netsh interface ipv4 set subinterface $_.InterfaceIndex mtu=1460 store=persistent
    }

    # As most if not all Windows containers are based on one of these images
    # we pull it here so that running a container using this image is quick.
    Write-Host 'Pulling latest Windows containers'
    & docker pull "microsoft/windowsservercore:${version}"
    if (!$?) {
      throw "Error running 'docker pull microsoft/windowsservercore:${version}'"
    }
    & docker pull "microsoft/nanoserver:${version}"
    if (!$?) {
      throw "Error running 'docker pull microsoft/nanoserver:${version}'"
    }

    Write-Host 'Setting container vEthernet MTU to 1460'
    & docker run --rm "microsoft/windowsservercore:${version}" powershell.exe "Get-NetAdapter | Where-Object {`$_.Name -like 'vEthernet*'} | ForEach-Object { & netsh interface ipv4 set subinterface `$_.InterfaceIndex mtu=1460 store=persistent }"
    if (!$?) {
      throw "Error running 'docker run microsoft/windowsservercore:${version}'"
    }

    Write-Host 'Launching sysprep.'
    & 'C:\Program Files\Google\Compute Engine\sysprep\gcesysprep.bat'
  }
}
catch {
  Write-Host 'Exception caught in script:'
  Write-Host $_.InvocationInfo.PositionMessage
  Write-Host "Windows build failed: $($_.Exception.Message)"
  exit 1
}
