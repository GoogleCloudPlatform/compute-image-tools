#  Copyright 2017 Google Inc. All Rights Reserved.
#
#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.

$ErrorActionPreference = 'Stop'

function Check-VMWareTools {
  Get-ChildItem HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall | Foreach-Object {
    if ((Get-ItemProperty $_.PSPath).DisplayName -eq 'VMWare Tools') {
      throw 'VMWare tools should not be installed'
    }
  }
}

function Check-MetadataAccessibility {
  @('metadata', 'metadata.google.internal') | ForEach-Object {
    if (-not (Test-Connection $_ -Count 1)) {
      throw "Failed to connect to $_"
    }
  }
}

function Check-Activation  {
  # Force activation as this is on a timer.
  & cscript c:\windows\system32\slmgr.vbs /ato

  $out = & cscript C:\Windows\system32\slmgr.vbs /dli
  Write-Output $out
  if ($out -notcontains 'License Status: Licensed') {
    throw 'Windows is not activated'
  }

  if ($out -notcontains '    Registered KMS machine name: kms.windows.googlecloud.com:1688') {
    throw 'Windows is not activated against GCE kms server'
  }
}

try {
  Write-Output 'Test: GCEAgent Service'
  Get-Service GCEAgent
  Write-Output 'Test: Check-VMWareTools'
  Check-VMWareTools
  Write-Output 'Test: Check-MetadataAccessibility'
  Check-MetadataAccessibility
  Write-Output 'Test: Check-Activation'
  Check-Activation

  Write-Output 'All Tests Passed'
}
catch {
  Write-Output 'Exception caught in script:'
  Write-Output $_.InvocationInfo.PositionMessage
  Write-Output "Test Failed: $($_.Exception.Message)"
}
