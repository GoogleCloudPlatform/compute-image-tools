@echo off
echo Starting SetupComplete.cmd > COM1:

if not exist D:\ (
  (echo select disk 1 & echo online disk & echo attribute disk clear readonly) | diskpart
) > COM1:
echo. > COM1:

schtasks.exe /query /tn GCEStartup
if NOT %ERRORLEVEL%==0 (
  schtasks.exe /create /tn GCEStartup /sc onstart /ru System /f /tr "C:\Windows\Setup\Scripts\SetupComplete.cmd"
)

D:\ProgramData\GooGet\googet.exe -root C:\ProgramData\GooGet -noconfirm install googet > COM1: 2>&1
if NOT %ERRORLEVEL%==0 (
  echo GooGet failed, retrying... > COM1:
  timeout /t 10
  D:\ProgramData\GooGet\googet.exe -root C:\ProgramData\GooGet -noconfirm install googet > COM1: 2>&1
)
C:\ProgramData\GooGet\googet.exe -root C:\ProgramData\GooGet -noconfirm install google-compute-engine-metadata-scripts > COM1: 2>&1

echo Installing Google CloudSDK > COM1:
D:\builder\components\GoogleCloudSDKInstaller.exe /S /allusers > COM1: 2>&1
if NOT %ERRORLEVEL%==0 (
  echo Google Cloud SDK exited with error level %ERRORLEVEL%, retrying... > COM1:
  timeout /t 10
  D:\builder\components\GoogleCloudSDKInstaller.exe /S /allusers > COM1: 2>&1
  if NOT %ERRORLEVEL%==0 (
    echo Windows build failed: Google Cloud SDK exited with error level %ERRORLEVEL% > COM1:
    exit 1
  )
)
echo Google Cloud SDK exited with error level %ERRORLEVEL% > COM1:

echo Rebooting to launch postinstall via startup scripts > COM1:
shutdown /r /t 01

REM Cleanup SetupComplete.cmd
del /F /Q C:\Windows\Setup\Scripts\SetupComplete.cmd
