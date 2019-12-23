@echo off
REM Copyright 2017 Google Inc. All Rights Reserved.
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

echo "Translate: Enable DHCP" > COM1:
powershell -command "Get-CimInstance Win32_NetworkAdapterConfiguration -Filter IPEnabled=True | Invoke-CimMethod -Name EnableDHCP" > COM1: 2>&1
powershell -command "Get-CimInstance Win32_NetworkAdapterConfiguration -Filter IPEnabled=True | Invoke-CimMethod -Name SetDNSServerSearchOrder -Arguments @{DNSServerSearchOrder=$null}" > COM1: 2>&1

REM Enable inbound communication from the metadata server.
echo "Translate: Enable inbound communication from the metadata server." > COM1:
netsh advfirewall firewall add rule name="Allow incoming from GCE metadata server" protocol=ANY remoteip=169.254.169.254 dir=in action=allow

REM Enable outbound communication to the metadata server.
echo "Translate: Enable outbound communication to the metadata server." > COM1:
netsh advfirewall firewall add rule name="Allow outgoing to GCE metadata server" protocol=ANY remoteip=169.254.169.254 dir=out action=allow

REM This is needed for 2008R2 networking to work, this will fail on post 2008R2 but that's fine.
echo "Translate: This is needed for 2008R2 networking to work, this will fail on post 2008R2 but that's fine." > COM1:
for /f "tokens=2 delims=:" %%a in (
  'ipconfig ^| find "Gateway"'
) do (
  echo "Translate: " [%%a] 2>&1 > COM1:
  netsh interface ipv4 set dnsservers "Local Area Connection" static address=%%a primary
  netsh interface ipv4 set dnsservers "Local Area Connection 2" static address=%%a primary
  netsh interface ipv4 set dnsservers "Local Area Connection 3" static address=%%a primary
)

echo "Translate: tzutil." > COM1: 2>&1
tzutil /s 'UTC'

echo "Translate: w32tm." > COM1: 2>&1
w32tm /resync

echo "Translate: ipconfig." > COM1: 2>&1
Run ipconfig /all > COM1: 2>&1

echo "Translate: GooGet." > COM1: 2>&1
C:\ProgramData\GooGet\googet.exe -root C:\ProgramData\GooGet -noconfirm install googet 2>&1 > COM1:
REM Install google-compute-engine-metadata-scripts and then run the task.
REM This needs to be on one line as this file will get overwritten.

echo "Translate: dir GooGet" > COM1: 2>&1
dir C:\ProgramData\GooGet\ > COM1: 2>&1

echo "Translate: dir c:" > COM1: 2>&1
dir C:\ > COM1: 2>&1

echo "Translate: available" > COM1: 2>&1
C:\ProgramData\GooGet\googet.exe available -root C:\ProgramData\GooGet\ > COM1: 2>&1
echo "Translate: google-compute-engine-metadata-scripts." > COM1: 2>&1
C:\ProgramData\GooGet\googet.exe -root C:\ProgramData\GooGet -noconfirm install google-compute-engine-metadata-scripts > COM1: 2>&1

echo "Translate: GCEStartup." > COM1: 2>&1
schtasks /run /tn GCEStartup > COM1: 2>&1
REM start cmd /c "C:\ProgramData\GooGet\googet.exe -root C:\ProgramData\GooGet -noconfirm install google-compute-engine-metadata-scripts > COM1: && schtasks /run /tn GCEStartup > COM1:"

echo "Translate: Done." > COM1: 2>&1
