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
# These steps use the DockerMsftProvider OneGet module:
# https://github.com/OneGet/MicrosoftDockerProvider.
#
# Docker provides its own "Docker for Windows" setup
# (https://docs.docker.com/docker-for-windows/), but ignore it. "Docker for
# Windows" is primarily targeted at running Linux containers on Windows hosts,
# so it relies on a physical Windows host with Hyper-V support. Their installer
# also adds several Docker components such as docker-compose and docker-machine
# which are not needed for running Windows Server containers. In the future we
# intend to satisfy the requirements for running Hyper-V containers in our
# Windows VMs, but we only want to include the minimum necessary Docker runtime
# support in our container images.

$ErrorActionPreference = 'Stop'

& googet -noconfirm update
try {
  if (!(Get-Package -ProviderName DockerMsftProvider -ErrorAction SilentlyContinue| Where-Object {$_.Name -eq 'docker'})) {
    Write-Host 'Installing NuGet module'
    Install-PackageProvider -Name NuGet -MinimumVersion 2.8.5.201 -Force

    Write-Host 'Installing DockerMsftProvider module'
    Install-Module -Name DockerMsftProvider -Repository PSGallery -Force

    Write-Host 'Installing Docker package'
    Install-Package -Name docker -ProviderName DockerMsftProvider -Force

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
    & docker pull microsoft/windowsservercore:1803
    if (!$?) {
      throw 'Error running "docker pull microsoft/windowsservercore:1803"'
    }
    & docker pull microsoft/nanoserver:1803
    if (!$?) {
      throw 'Error running "docker pull microsoft/nanoserver:1803"'
    }

    Write-Host 'Setting container vEthernet MTU to 1460'
    & docker run --rm microsoft/windowsservercore:1803 powershell.exe "Get-NetAdapter | Where-Object {`$_.Name -like 'vEthernet*'} | ForEach-Object { & netsh interface ipv4 set subinterface `$_.InterfaceIndex mtu=1460 store=persistent }"
    if (!$?) {
      throw 'Error running "docker run microsoft/windowsservercore:1803"'
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
