function Get-MetadataValue {
    <#
    .SYNOPSIS
      Returns a value for a given metadata key.

    .PARAMETER $key
      The metadata key to retrieve.

    .PARAMETER $default
      The value to return if the key is not found.

    .RETURNS
      The value for the key or null.
  #>
    param (
        [parameter(Mandatory=$true)]
        [string]$key,
        [parameter(Mandatory=$false)]
        [string]$default
    )

    # Returns the provided metadata value for a given key.
    $url = "http://metadata.google.internal/computeMetadata/v1/instance/attributes/$key"
    try {
        $client = New-Object Net.WebClient
        $client.Headers.Add('Metadata-Flavor', 'Google')
        return ($client.DownloadString($url)).Trim()
    }
    catch [System.Net.WebException] {
        if ($default) {
            return $default
        }
        else {
            Write-Host "Failed to retrieve value for $key."
            return $null
        }
    }
}

function Export-ImageMetadata {
    $image_version = Get-Date -Format "yyyyMMdd"
    $build_date = Get-Date -Format "o"
    $image_metadata = @{'id' = $image_id;
                        'name' = $image_name;
                        'family' = $image_family;
                        'version' = $image_version;
                        'build_date' = $build_date;
                        'packages' = @()}

    # Get Googet packages.
    $out = & 'C:\ProgramData\GooGet\googet.exe' -root 'C:\ProgramData\GooGet' 'installed'
    $out = $out[1..$out.length]
    [array]::sort($out)

    foreach ($package_line in $out) {
        $name = $package_line.Trim().Split(' ')[0]
        # Get Package Info for each package
        $info = & 'C:\ProgramData\GooGet\googet.exe' -root 'C:\ProgramData\GooGet' 'installed' '-info' $name
        $version = $info[4].Split(":")[1].Trim()
        $source = $info[8].Split(":")
        $source = [String]::Concat($source[1..$source.length])
        $package_metadata = @{'name' = $name;
                            'version' = $version;
                            'commmit_hash' = $source}
        $image_metadata['packages'] += $package_metadata
    }

    # Save the JSON image_metadata.
    $image_metadata_json = $image_metadata | ConvertTo-Json -Compress
    $image_metadata_json | & 'gsutil' cp - "${metadata_dest}/metadata.json"
}

try {
    Write-Host 'Beginning export windows package metadata'
    $metadata_dest = Get-MetadataValue -key 'metadata_dest'
    $image_id = Get-MetadataValue -key 'image_id'
    $image_name = Get-MetadataValue -key 'image_name'
    $image_family = Get-MetadataValue -key 'image_family'
    Export-ImageMetadata
}
catch {
    Write-Host 'Exception caught in script:'
    Write-Host $_.InvocationInfo.PositionMessage
    Write-Host "Message: $($_.Exception.Message)"
    Write-Host 'Windows export package metadata failed.'
    exit 1
}