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
$DockerPath = 'https://master.dockerproject.org/windows/x86_64/docker.exe'
$DockerDPath = 'https://master.dockerproject.org/windows/x86_64/dockerd.exe'
$DockerDataPath = "$($env:ProgramData)\docker"
$DockerServiceName = "docker"

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

function Install-ContainerHost {
  $installState = (Get-WindowsFeature containers)
  if($installstate.installed -ne 'True') {
    Write-Output "Containers Windows feature not installed. Installing."
    Install-WindowsFeature containers
    return
  }

  else {
    if (Test-Docker) {
      Write-Output "Docker is already installed, skipping Docker installation."
    }
    else {
      Install-Docker -DockerPath $DockerPath -DockerDPath $DockerDPath
    }
    Write-Output "Docker install complete."
  }
}

function Copy-File {
  param(
    [string]$SourcePath,
    [string]$DestinationPath
  )

  if ($SourcePath -eq $DestinationPath) {
    return
  }     
  if (Test-Path $SourcePath) {
    Copy-Item -Path $SourcePath -Destination $DestinationPath
  }
  elseif (($SourcePath -as [System.URI]).AbsoluteURI -ne $null) {
    if ($PSVersionTable.PSVersion.Major -ge 5) {
      # Disable progress display because it kills performance for large downloads (at least on 64-bit PowerShell)
      $ProgressPreference = 'SilentlyContinue'
      Invoke-WebRequest -Uri $SourcePath -OutFile $DestinationPath -UseBasicParsing
      $ProgressPreference = 'Continue'
    }
    else {
      $webClient = New-Object System.Net.WebClient
      $webClient.DownloadFile($SourcePath, $DestinationPath)
    } 
  }
  else {
    throw "Cannot copy from $SourcePath"
  }
}

function Install-Docker() {
  param(
    [string][ValidateNotNullOrEmpty()]$DockerPath = "https://master.dockerproject.org/windows/x86_64/docker.exe",
    [string][ValidateNotNullOrEmpty()]$DockerDPath = "https://master.dockerproject.org/windows/x86_64/dockerd.exe"
  )

  Write-Output "Installing Docker."
  Copy-File -SourcePath $DockerPath -DestinationPath $env:windir\System32\docker.exe        
  Write-Output "Installing Docker daemon."
  Copy-File -SourcePath $DockerDPath -DestinationPath $env:windir\System32\dockerd.exe
  $dockerConfigPath = Join-Path $DockerDataPath "config"
  if (!(Test-Path $dockerConfigPath)) {
    md -Path $dockerConfigPath | Out-Null
  }

  # Register the docker service.
  # Configuration options should be placed at %programdata%\docker\config\daemon.json
  Write-Output "Configuring the docker service."
  $daemonSettings = New-Object PSObject     
  $certsPath = Join-Path $DockerDataPath "certs.d"
  if (Test-Path $certsPath) {
    $daemonSettings | Add-Member NoteProperty hosts @("npipe://", "0.0.0.0:2376")
    $daemonSettings | Add-Member NoteProperty tlsverify true
    $daemonSettings | Add-Member NoteProperty tlscacert (Join-Path $certsPath "ca.pem")
    $daemonSettings | Add-Member NoteProperty tlscert (Join-Path $certsPath "server-cert.pem")
    $daemonSettings | Add-Member NoteProperty tlskey (Join-Path $certsPath "server-key.pem")
  }
  $daemonSettingsFile = Join-Path $dockerConfigPath "daemon.json"
  $daemonSettings | ConvertTo-Json | Out-File -FilePath $daemonSettingsFile -Encoding ASCII  
  & dockerd --register-service --service-name $DockerServiceName
  Start-Service -Name $DockerServiceName

  # Wait for docker to come to steady state
  Wait-Docker
}

function Test-Docker() {
  $service = Get-Service -Name $DockerServiceName -ErrorAction SilentlyContinue
  return ($service -ne $null)
}

function Wait-Docker() {
  Write-Output "Waiting for Docker daemon."
  $dockerReady = $false
  $startTime = Get-Date
  while (-not $dockerReady) {
    try {
      docker version | Out-Null
        if (-not $?) {
          throw "Docker daemon is not running yet."
        }
      $dockerReady = $true
    }
    catch {
      $timeElapsed = $(Get-Date) - $startTime
        if ($($timeElapsed).TotalMinutes -ge 1) {
          throw "Docker Daemon did not start successfully within 1 minute."
        }

      # Delay error and try again
      Start-Sleep -sec 1
    }
  }
  Write-Output "Successfully connected to Docker Daemon."
}

function Run-FirstBootSteps {
  Write-Host 'Installing NuGet module.'
  Install-PackageProvider -Name NuGet -MinimumVersion 2.8.5.201 -Force

  Write-Host 'Installing DockerMsftProvider module.'
  Install-Module -Name DockerMsftProvider -Repository PSGallery -Force

  Write-Host 'Installing Docker CE.'
  Install-ContainerHost
}

function Run-SecondBootSteps {
  param (
    [parameter(Mandatory=$true)]
      [string]$windows_version
  )

  Install-ContainerHost

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

  # Use the existence of the docker service to determine if all firstboot
  # steps completed, as the container install forces a reboot prior to
  # docker installation.
  $dockerStatus = Get-Service -Name 'docker' -ErrorAction SilentlyContinue
  Write-Host "Windows version: $windows_version"
  if (-not $windows_version) {
    throw 'Error retrieving "version" from metadata'
  }

  if ($dockerStatus -eq $null) {
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
