@echo off
echo Starting SetupComplete.cmd > COM1:

if not exist D:\ (
  (echo select disk 1 & echo online disk & echo attribute disk clear readonly) | diskpart
) > COM1:
echo. > COM1:

REM This sleep fixes a multitude of timing problems
REM We use ping instead of timeout as timeout seems to have timing
REM problems if the clock resets (as it does on first boot)
ping 127.0.0.1 -n 60

schtasks.exe /query /tn GCEStartup
if NOT %ERRORLEVEL%==0 (
  schtasks.exe /create /tn GCEStartup /sc onstart /ru System /f /tr "C:\Windows\Setup\Scripts\SetupComplete.cmd"
)

REM Check for .NET 4.8
REM https://support.microsoft.com/en-us/help/4503548/microsoft-net-framework-4-8-offline-installer-for-windows
reg query "HKLM\SOFTWARE\Microsoft\NET Framework Setup\NDP\v4\Full" /f 4.8
if NOT %ERRORLEVEL%==0 (
  if exist D:\builder\components\dotnet48-offline-installer.exe (
    echo Installing .NET 4.8. > COM1:
    D:\builder\components\dotnet48-offline-installer.exe /quiet /norestart > COM1:
    echo .NET install exited with code %ERRORLEVEL%. > COM1:
    echo Exiting for reboot. > COM1:
    shutdown /r /t 00
    exit /b 0
  ) else (
    echo Windows build failed: .NET 4.8 not installed and no installer found. > COM1:
    exit /b 1
  )
)

REM Check for PowerShell 5.1 on 2008R2/7
reg query "HKLM\SOFTWARE\Microsoft\PowerShell\3\PowerShellEngine" /f 5.1
if NOT %ERRORLEVEL%==0 (
  if exist D:\builder\components\Win7AndW2K8R2-KB3191566-x64.msu (
    REM 2008R2/7 networking require manually setting DNS servers to the default gateway
    for /f "tokens=2 delims=:" %%a in (
      'ipconfig ^| find "Gateway"'
    ) do netsh interface ipv4 set dnsservers "Local Area Connection" static address=%%a primary

    echo Installing WMF 5.1. > COM1:
    D:\builder\components\Win7AndW2K8R2-KB3191566-x64.msu /quiet
    echo WMF install exited with code %ERRORLEVEL%. > COM1:
    echo Exiting for reboot. > COM1:
    shutdown /r /t 00
    exit /b 0
  ) else (
    echo Windows x64 build failed: WMF 5.1 not installed and no installer found. > COM1:
    exit /b 1
  )
  if exist D:\builder\components\Win7-KB3191566-x86.msu (
    REM 2008R2/7 networking require manually setting DNS servers to the default gateway
    for /f "tokens=2 delims=:" %%a in (
      'ipconfig ^| find "Gateway"'
    ) do netsh interface ipv4 set dnsservers "Local Area Connection" static address=%%a primary

    echo Installing WMF 5.1. > COM1:
    D:\builder\components\Win7-KB3191566-x86.msu /quiet > COM1:
    echo WMF install exited with code %ERRORLEVEL%. > COM1:
    echo Exiting for reboot. > COM1:
    shutdown /r /t 00
    exit /b 0
  ) else (
    echo Windows x86 build failed: WMF 5.1 not installed and no installer found. > COM1:
    exit /b 1
  )
)

REM Enable .NET 3.5 on 2008R2 for Google Cloud SDK
reg query "HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion" /e /f 6.1 > COM1:
if %ERRORLEVEL%==0 (
  echo "Enabling .NET 3.5 for Google Cloud SDK installation."
  %WINDIR%\System32\WindowsPowerShell\v1.0\powershell.exe -ExecutionPolicy Unrestricted -NonInteractive -NoProfile -NoLogo "Add-WindowsFeature Net-Framework-Core | Format-List" > COM1:
)

netsh interface ipv4 show config > COM1:
REM Uncomment to troubleshoot
REM echo Network configuration > COM1:
REM ipconfig /all > COM1:
REM netsh interface ipv4 show route > COM1:
REM echo Ping package server > COM1:
REM ping packages.cloud.google.com -n 10 > COM1:

REM Set the Googet install path based on the architecture.
set GOOGETSOURCEPATH="D:\ProgramData\GooGet\googet.exe"
if %PROCESSOR_ARCHITECTURE%==x86 (set GOOGETSOURCEPATH="D:\builder\components\googet.exe")

echo Installing GooGet and GooGet packages. > COM1:
%GOOGETSOURCEPATH% -root C:\ProgramData\GooGet -noconfirm install googet > COM1: 2>&1
if NOT %ERRORLEVEL%==0 (
  echo GooGet install failed from %GOOGETSOURCEPATH%, retrying... > COM1:
  timeout /t 10
  %GOOGETSOURCEPATH% -root C:\ProgramData\GooGet -noconfirm install googet > COM1: 2>&1
)

if %PROCESSOR_ARCHITECTURE%==x86 (
  echo Coping x86 GooGet > COM1:
  copy /B %GOOGETSOURCEPATH% C:\ProgramData\GooGet\ > COM1: 2>&1

  echo GooGet x86 GCE Windows Agent > COM1:
  C:\ProgramData\GooGet\googet.exe -root C:\ProgramData\GooGet -noconfirm install D:\builder\components\google-compute-engine-windows-x86.x86_32.4.6.0@1.goo > COM1: 2>&1

  echo GooGet x86 PowerShell module > COM1:
  C:\ProgramData\GooGet\googet.exe -root C:\ProgramData\GooGet -noconfirm install D:\builder\components\google-compute-engine-powershell.noarch.1.1.1@4.goo > COM1: 2>&1

  echo GooGet x86 certificate generator > COM1:
  C:\ProgramData\GooGet\googet.exe -root C:\ProgramData\GooGet -noconfirm install D:\builder\components\certgen-x86.x86_32.1.0.0@2.goo > COM1: 2>&1

  echo GooGet x86 GCE Metadata Script runner > COM1:
  C:\ProgramData\GooGet\googet.exe -root C:\ProgramData\GooGet -noconfirm install D:\builder\components\google-compute-engine-metadata-scripts-x86.x86_32.4.2.1@1.goo > COM1: 2>&1

  echo GooGet x86 GCE Sysprep > COM1:
  C:\ProgramData\GooGet\googet.exe -root C:\ProgramData\GooGet -noconfirm install D:\builder\components\google-compute-engine-sysprep.noarch.3.10.1@1.goo > COM1: 2>&1

  echo GooGet x86 > COM1:
  C:\ProgramData\GooGet\googet.exe -root C:\ProgramData\GooGet -noconfirm install D:\builder\components\googet-x86.x86_32.2.16.3@1.goo > COM1: 2>&1
)

if NOT %PROCESSOR_ARCHITECTURE%==x86 (
  C:\ProgramData\GooGet\googet.exe -root C:\ProgramData\GooGet -noconfirm install google-compute-engine-metadata-scripts > COM1: 2>&1
)

echo Installing Google Cloud SDK. > COM1:
D:\builder\components\GoogleCloudSDKInstaller.exe /S /allusers > COM1: 2>&1
if NOT %ERRORLEVEL%==0 (
  echo Google Cloud SDK exited with error level %ERRORLEVEL%, retrying... > COM1:
  timeout /t 10
  D:\builder\components\GoogleCloudSDKInstaller.exe /S /allusers /logtofile > COM1: 2>&1
  if NOT %ERRORLEVEL%==0 (
    echo Windows build failed: Google Cloud SDK exited with error level %ERRORLEVEL% > COM1:
    type  D:\builder\components\CloudSDKInstall.log > COM1:
    exit 1
  )
)
echo Google Cloud SDK exited with error level %ERRORLEVEL% > COM1:

echo SetupComplete.cmd completed. Rebooting to launch post_install.ps1 via startup scripts. > COM1:
shutdown /r /t 01

echo Deleting SetupComplete.cmd to cleanup. > COM1:
del /F /Q C:\Windows\Setup\Scripts\SetupComplete.cmd
