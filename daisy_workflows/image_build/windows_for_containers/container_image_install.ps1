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

# This script prepares a Windows Server image to run Windows Server containers.
# It works on Windows Server version 1709 and later.
#
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

function Get-ContainerRepo {
  param (
    [parameter(Mandatory=$true)]
      [string]$windows_version
  )

  # The canonical source for Windows base container images used to be Docker
  # Hub ('microsoft/...'), now it is Microsoft Container Registry
  # ('mcr.microsoft.com/windows/...'). See:
  # https://azure.microsoft.com/en-us/blog/microsoft-syndicates-container-catalog/
  # https://blogs.technet.microsoft.com/virtualization/2018/11/13/windows-server-2019-now-available/
  if ($windows_version -eq '1709' -or $windows_version -eq '1803') {
    return 'microsoft'
  }
  return 'mcr.microsoft.com/windows'
}

function Get-ContainerVersionLabel {
  param (
    [parameter(Mandatory=$true)]
      [string]$windows_version
  )

  # For more information about Windows container version labels see:
  # https://hub.docker.com/r/microsoft/windowsservercore/
  # https://blogs.technet.microsoft.com/virtualization/2017/10/18/container-images-are-now-out-for-windows-server-version-1709/
  # https://azure.microsoft.com/en-us/blog/microsoft-syndicates-container-catalog/
  # https://blogs.technet.microsoft.com/virtualization/2018/11/13/windows-server-2019-now-available/
  if ($windows_version -eq '2019') {
    return 'ltsc2019'
  }
  # '1709', '1803', '1809':
  return $windows_version
}

function Supports-NanoserverContainerImage {
  param (
    [parameter(Mandatory=$true)]
      [string]$windows_version
  )

  # Windows Server 2019 does not support the nanoserver container image, only
  # Servercore:
  # https://blogs.technet.microsoft.com/virtualization/2018/11/13/windows-server-2019-now-available/.
  if ($windows_version -eq '2019') {
    return $false
  }
  return $true
}

function Get-ServerCoreImageName {
  param (
    [parameter(Mandatory=$true)]
      [string]$windows_version
  )

  $repo = Get-ContainerRepo $windows_version
  if ($windows_version -eq '1709' -or $windows_version -eq '1803') {
    $image = 'windowsservercore'
  } else {
    $image = 'servercore'
  }
  $label = Get-ContainerVersionLabel $windows_version
  return "${repo}/${image}:${label}"
}

function Get-NanoserverImageName {
  param (
    [parameter(Mandatory=$true)]
      [string]$windows_version
  )
  $repo = Get-ContainerRepo $windows_version
  $image = 'nanoserver'
  $label = Get-ContainerVersionLabel $windows_version
  return "${repo}/${image}:${label}"
}

function Get-BaseContainerImageNames {
  param (
    [parameter(Mandatory=$true)]
      [string]$windows_version
  )
  $images = @(Get-ServerCoreImageName $windows_version)
  if (Supports-NanoserverContainerImage $windows_version) {
    $images += (Get-NanoserverImageName $windows_version)
  }
  return $images
}

function Run-FirstBootSteps {
  Write-Host 'Installing NuGet module'
  Install-PackageProvider -Name NuGet -MinimumVersion 2.8.5.201 -Force

  Write-Host 'Installing DockerMsftProvider module'
  Install-Module -Name DockerMsftProvider -Repository PSGallery -Force

  $docker_version = "19.03"
  Write-Host "Installing Docker EE ${docker_version}"
  Install-Package -Name docker -ProviderName DockerMsftProvider -Force -RequiredVersion ${docker_version}
}

function Run-SecondBootSteps {
  param (
    [parameter(Mandatory=$true)]
      [string]$windows_version
  )

  # For some reason the docker service may not be started automatically on the
  # first reboot, although it seems to work fine on subsequent reboots. The
  # docker service must be running or else the vEthernet interface may not be
  # present.
  Restart-Service docker

  Write-Host 'Setting host vEthernet MTU to 1460'
  Get-NetAdapter | Where-Object {$_.Name -like 'vEthernet*'} | ForEach-Object {
    & netsh interface ipv4 set subinterface $_.InterfaceIndex mtu=1460 store=persistent
  }

  # As most if not all Windows containers are based on one of the base images
  # provided by Microsoft, we pull them here so that running a container using
  # this image is quick.
  $container_images = Get-BaseContainerImageNames $windows_version
  ForEach ($image in $container_images) {
    Write-Host "Pulling container image: $image"
    & docker pull $image
    if (!$?) {
      throw "Error running 'docker pull $image'"
    }
  }

  Write-Host 'Setting container vEthernet MTU to 1460'
  $servercore_image = Get-ServerCoreImageName $windows_version
  & docker run --rm "$servercore_image" powershell.exe "Get-NetAdapter | Where-Object {`$_.Name -like 'vEthernet*'} | ForEach-Object { & netsh interface ipv4 set subinterface `$_.InterfaceIndex mtu=1460 store=persistent }"
  if (!$?) {
    throw "Error running 'docker run $servercore_image'"
  }
}

& googet -noconfirm update
& ping 127.0.0.1 -n 60
try {
  $windows_version = Get-MetadataValue 'version'
  Write-Host "Windows version: $windows_version"
  if (-not $windows_version) {
    throw 'Error retrieving "version" from metadata'
  }

  if (!(Get-Package -ProviderName DockerMsftProvider -ErrorAction SilentlyContinue| Where-Object {$_.Name -eq 'docker'})) {
    Run-FirstBootSteps
    Write-Host 'Restarting computer to finish install'
    Restart-Computer -Force
  }
  else {
    Run-SecondBootSteps $windows_version
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
