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

REM Verify VM to VM DNS connection
ping -n 1 %INSTANCE%
if %errorlevel% NEQ 0 (
    echo DNS_Failed
)

REM Verify VM to external DNS connection
ping -n 1 www.google.com

REM Signalize wait-for-instance that instance is ready or error occurred
if %errorlevel% NEQ 0 (
    echo DNS_Failed
) else (
    echo DNS_Success
)
