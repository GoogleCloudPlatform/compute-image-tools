@echo off
REM Copyright 2018 Google Inc. All Rights Reserved.
REM
REM Licensed under the Apache License, Version 2.0 (the "License");
REM you may not use this file except in compliance with the License.
REM You may obtain a copy of the License at
REM
REM http://www.apache.org/licenses/LICENSE-2.0
REM
REM Unless required by applicable law or agreed to in writing, software
REM distributed under the License is distributed on an "AS IS" BASIS,
REM WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
REM See the License for the specific language governing permissions and
REM limitations under the License.
REM
REM Test if the local ssd is working or not. Partition it, format it, place some
REM file, unmount it and check if it's still there after remount.
REM
REM This script runs for testing SCSI and NVMe interfaces.

REM Create a primary partition allocating the whole disk and format it
echo select disk 1 > cmds
echo create partition primary >> cmds
echo assign letter D >> cmds
echo select volume 1 >> cmds
echo format fs=ntfs quick label=local_ssd >> cmds
type cmds | diskpart
IF %ERRORLEVEL% NEQ 0 GOTO :ERROR

REM place some file on it with SuccessMatch string
echo CheckSuccessful > D:\test

REM umount it and certify that file does not exist anymore
echo select volume 1 > cmds
echo remove letter=D >> cmds
type cmds | diskpart
IF %ERRORLEVEL% NEQ 0 GOTO :ERROR

echo D:\test should not be reachable now
type D:\test

REM remount it and echo file content
echo select volume 1 > cmds
echo assign letter D >> cmds
type cmds | diskpart
IF %ERRORLEVEL% NEQ 0 GOTO :ERROR

REM if no error occured, it's working fine, print SuccessMatch
type D:\test

GOTO :EOF

:ERROR
  REM print FailureMatch
  echo "CheckFailed"

:EOF
