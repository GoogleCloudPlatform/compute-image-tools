@echo off
REM Copyright 2019 Google Inc. All Rights Reserved.
REM
REM Licensed under the Apache License, Version 2.0 (the "License");
REM you may not use this file except in compliance with the License.
REM You may obtain a copy of the License at
REM
REM     http://www.apache.org/licenses/LICENSE-2.0
REM
REM Unless required by applicable law or agreed to in writing, software
REM distributed under the License is distributed on an "AS IS" BASIS,
REM WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
REM See the License for the specific language governing permissions and
REM limitations under the License.

REM Give the network time to initialize.
ping 127.0.0.1 -n 60

echo "Translate: Starting image translate..." > COM1:

for /f "tokens=4-5 delims=. " %%i in ('ver') do set version_number=%%i.%%j
echo "Windows Version Number: %version_number%" > COM1:

set has_network_interface_api=1
if %version_number% == "6.0" ( set has_network_interface_api=0 )
if %version_number% == "6.1" ( set has_network_interface_api=0 )

if %has_network_interface_api% EQU 0 (
  REM Restart the system in 10 minutes. Translate.ps1 will cancel this restart when it starts.
  REM This is to address initial boot issues, mostly on 2008R2, where the network interface is not yet usable.

  echo "Scheduling restart in 10 minutes. Translate.ps1 will cancel this restart." > COM1:
  shutdown /r /t 600
)

if %has_network_interface_api% EQU 1 (
  echo "Running network.ps1 to reconfigure network to DHCP if needed and log DNS and connectivity tests." > COM1:
  PowerShell.exe -NoProfile -NoLogo -ExecutionPolicy Unrestricted -File "%ProgramFiles%\Google\Compute Engine\metadata_scripts\network.ps1" > COM1:
)

echo "Translate: Opening firewall ports for GCE metadata server." > COM1:
REM Enable inbound communication from the metadata server.
netsh advfirewall firewall add rule name="Allow incoming from GCE metadata server" protocol=ANY remoteip=169.254.169.254 dir=in action=allow

REM Enable outbound communication to the metadata server.
netsh advfirewall firewall add rule name="Allow outgoing to GCE metadata server" protocol=ANY remoteip=169.254.169.254 dir=out action=allow

if %has_network_interface_api% EQU 0 (
  echo "Translate: Network configuration for Windows 2008 R2/Windows 7" > COM1:
  REM This is needed for 2008R2 networking to work
  for /f "tokens=2 delims=:" %%a in (
    'ipconfig ^| find "Gateway"'
  ) do (
    netsh interface ipv4 set dnsservers "Local Area Connection" static address=%%a primary
    netsh interface ipv4 set dnsservers "Local Area Connection 2" static address=%%a primary
    netsh interface ipv4 set dnsservers "Local Area Connection 3" static address=%%a primary
  )
)

echo "Translate: Setting timezone" > COM1:
tzutil /s Greenwich Standard Time
w32tm /resync

echo "Translate: Installing GooGet." > COM1:
C:\ProgramData\GooGet\googet.exe -root C:\ProgramData\GooGet -noconfirm install C:\ProgramData\GooGet\components\googet-x86.x86_32.2.16.3@1.goo
echo "Translate: Installing metadata scripts runner." > COM1:
start cmd /c "C:\ProgramData\GooGet\googet.exe -root C:\ProgramData\GooGet -noconfirm install C:\ProgramData\GooGet\components\google-compute-engine-metadata-scripts-x86.x86_32.4.2.1@1.goo > COM1: && C:\ProgramData\GooGet\googet.exe -root C:\ProgramData\GooGet -noconfirm verify -reinstall C:\ProgramData\GooGet\components\google-compute-engine-metadata-scripts-x86.x86_32.4.2.1@1.goo > COM1: && schtasks /run /tn GCEStartup > COM1:"
