function Get-MetadataValue {
    param (
        [parameter(Mandatory=$true)]
        [string]$key,
        [parameter(Mandatory=$false)]
        [string]$default
    )

    # Returns the provided metadata value for a given key.
    $url = "http://metadata.google.internal/computeMetadata/v1/instance/attributes/$key"
    $max_attemps = 30
    for ($i=0; $i -le $max_attemps; $i++) {
        try {
            $client = New-Object Net.WebClient
            $client.Headers.Add('Metadata-Flavor', 'Google')
            $value = ($client.DownloadString($url)).Trim()
            Write-Host "Retrieved metadata for key $key with value $value."
            return $value
        }
        catch [System.Net.WebException] {
            if ($default) {
                Write-Host "Failed to retrieve metadata for $key, returning default $default."
                return $default
            }
            # Sleep after each failure with no default value to give the network adapters time to become functional.
            Start-Sleep -s 1
        }
    }
    Write-Host "Failed $max_attemps times to retrieve value from metadata for $key, returning null."
    return $null
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
    Write-Host 'Endding export windows package metadata'
}
catch {
    Write-Host 'Exception caught in script:'
    Write-Host $_.InvocationInfo.PositionMessage
    Write-Host "Message: $($_.Exception.Message)"
    Write-Host 'Windows export package metadata failed.'
    exit 1
}