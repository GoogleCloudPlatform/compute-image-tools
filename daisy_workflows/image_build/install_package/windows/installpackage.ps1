#  Copyright 2025 Google Inc. All Rights Reserved.
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


function Get-MetadataValue {
  <#
  .SYNOPSIS
    Returns a value for a given metadata key.
  .DESCRIPTION
    Attempts to retrieve the value for a given metadata key.
  .PARAMETER $key
    The metadata key to retrieve.
  .EXAMPLE
    Get-MetadataValue key
  #>
  param (
    [parameter(Mandatory=$true)]
      [string]$key
  )

  $url = "http://metadata.google.internal/computeMetadata/v1/instance/attributes/${key}"
  Write-Host "Fetching value from: ${url}"
  try {
    $client = New-Object Net.WebClient
    $client.Headers.Add('Metadata-Flavor', 'Google')
    $out = ($client.DownloadString($url)).Trim()
    return $out
  }
  catch {
    Write-Host 'Exception caught in downloading:'
    Write-Host $_.InvocationInfo.PositionMessage
    Write-Host "Package download failed: $($_.Exception.Message)"
    throw "Failed to download package"
  }
}

function Install-Package {
  <#
  .SYNOPSIS
    Installs a package.
  .DESCRIPTION
    Installs a package from 'gcs_package_path' Metadata attribute.
  .EXAMPLE
    Install-Package
  #>
  
    Write-Host "Getting value set at gcs_package_path metadata key"
    $gcs_path = Get-MetadataValue -key 'gcs_package_path'
    if (!$gcs_path) {
        throw "Package path is empty: $gcs_path"
    }

    $paths = $gcs_path -split ','
    foreach ($path in $paths) {
      & 'gcloud' storage cp $path "C:\Program Files\Google\Compute Engine\package.goo"
      & 'googet' -noconfirm=true install "C:\Program Files\Google\Compute Engine\package.goo"
      Remove-Item -Path "C:\Program Files\Google\Compute Engine\package.goo" -ErrorAction SilentlyContinue
    }
    Write-Host "Successfully installed packages"
}

$config = @'

[Core]
log_level = 4
log_verbosity = 4
'@

try {
    Install-Package

    Write-Host 'Enabling debug logging for guest-agent'
    Add-Content -Path "C:\Program Files\Google\Compute Engine\instance_configs.cfg" -Value $config

    Write-Host 'Launching sysprep.'
    & 'C:\Program Files\Google\Compute Engine\sysprep\gcesysprep.bat'
}
catch {
  Write-Host 'Exception caught in script:'
  Write-Host $_.InvocationInfo.PositionMessage
  Write-Host "Package install failed: $($_.Exception.Message)"
  exit 1
}