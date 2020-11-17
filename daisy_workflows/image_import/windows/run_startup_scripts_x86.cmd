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

echo "Translate: Starting image translate..." > COM1:

echo "Translate: Opening firewall ports for GCE metadata server." > COM1:
REM Enable inbound communication from the metadata server.
netsh advfirewall firewall add rule name="Allow incoming from GCE metadata server" protocol=ANY remoteip=169.254.169.254 dir=in action=allow

REM Enable outbound communication to the metadata server.
netsh advfirewall firewall add rule name="Allow outgoing to GCE metadata server" protocol=ANY remoteip=169.254.169.254 dir=out action=allow

echo "Translate: Network configuration for Windows 2008 R2/Windows 7" > COM1:
REM This is needed for 2008R2 networking to work, this will fail on post 2008R2 but that's fine.
for /f "tokens=2 delims=:" %%a in (
  'ipconfig ^| find "Gateway"'
) do (
  netsh interface ipv4 set dnsservers "Local Area Connection" static address=%%a primary
  netsh interface ipv4 set dnsservers "Local Area Connection 2" static address=%%a primary
  netsh interface ipv4 set dnsservers "Local Area Connection 3" static address=%%a primary
)

echo "Translate: Setting timezone" > COM1:
tzutil /s Greenwich Standard Time
w32tm /resync

echo "Translate: Verifying network connectivity" > COM1:
REM Attempt to connect to metadata service and packages.cloud.google.com 5 times, taking as short as 5 seconds and up to 310 second.
for /L %%I IN (1,1,5) do (
  ping 169.254.169.254 -n 1
  if NOT %ERRORLEVEL% == 0 (
    echo "Translate: Failed connectivity test of 169.254.169.254, waiting 30 seconds." > COM1:
    ping 127.0.0.1 -n 30
  )
  ping packages.cloud.google.com -n 1
  if NOT %ERRORLEVEL% == 0 (
    echo "Translate: Failed connectivity test of packages.cloud.google.com, waiting 30 seconds." > COM1:
    ping 127.0.0.1 -n 30
  )
)

REM Final connectivity test with metadata service, if unavailable output network interface setting and reboot once.
REM A connectivity failure after the reboot will be considered a fatal error.
ping 169.254.169.254 -n 1
if %ERRORLEVEL% == 0 (
   echo "Translate: Confirmed connectivity with 169.254.169.254." > COM1:
) else (
   netsh interface ipv4 dump > COM1:
   netsh interface ipv4 show addresses > COM1:
   IF EXIST "%temp%\run_startup_scripts_failed_to_connect.txt" (
     echo "TranslateFailed: Repeated failure of connectivity test with 169.254.169.254" > COM1:
     exit
   ) else (
     echo "Translate: Failed connectivity test with 169.254.169.254, restarting once to attempt remediation." > COM1:
     echo "failed to connect" >> "%temp%\run_startup_scripts_failed_to_connect.txt"
     shutdown /r /t 00
     exit
   )
)

echo "Translate: Installing GooGet." > COM1:
C:\ProgramData\GooGet\googet.exe -root C:\ProgramData\GooGet -noconfirm install C:\ProgramData\GooGet\components\googet-x86.x86_32.2.16.3@1.goo
echo "Translate: Installing metadata scripts runner." > COM1:
start cmd /c "C:\ProgramData\GooGet\googet.exe -root C:\ProgramData\GooGet -noconfirm install C:\ProgramData\GooGet\components\google-compute-engine-metadata-scripts-x86.x86_32.4.2.1@1.goo > COM1: && C:\ProgramData\GooGet\googet.exe -root C:\ProgramData\GooGet -noconfirm verify -reinstall C:\ProgramData\GooGet\components\google-compute-engine-metadata-scripts-x86.x86_32.4.2.1@1.goo > COM1: && schtasks /run /tn GCEStartup > COM1:"
