# This script can be run after image setup.  It contains commands that
# should be run once when creating an image but do not need to be run
# on every subsequent sysprep.

$ErrorActionPreference = 'Stop'

$script:gce_install_dir = 'C:\Program Files\Google\Compute Engine'
$script:hosts_file = "$env:windir\system32\drivers\etc\hosts"
$script:components_path = 'D:\builder\components'

function Run-Command {
 [CmdletBinding(SupportsShouldProcess=$true)]
  param (
    [Parameter(Mandatory=$true, ValueFromPipelineByPropertyName=$true)]
      [string]$Executable,
    [Parameter(ValueFromRemainingArguments=$true,
               ValueFromPipelineByPropertyName=$true)]
      $Arguments = $null
  )
  Write-Output "Running $Executable with arguments $Arguments."
  $out = &$executable $arguments 2>&1 | Out-String
  $out.Trim()
}

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
      Write-Output "Failed to retrieve value for $key."
      return $null
    }
  }
}

function Reboot-Required {
  return (Test-Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update\RebootRequired')
}

function Install-WindowsUpdates {
  <#
    .SYNOPSIS
      Check for updates, returns true if restart is required.
  #>

  # https://support.microsoft.com/en-us/help/4072698/windows-server-guidance-to-protect-against-the-speculative-execution
  if (-not (Test-Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\QualityCompat')) {
    New-Item -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\QualityCompat' -Type Directory
  }
  New-ItemProperty -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\QualityCompat' -Name 'cadca5fe-87d3-4b96-b7fb-a231484277cc' -Value 0  -PropertyType DWORD -Force

  Write-Output 'Starting Windows update.'
  if (-not (Test-Connection download.microsoft.com -Count 1 -ErrorAction SilentlyContinue)) {
    throw 'Windows update server is not reachable. Cannot complete image build.'
  }

  # If MSFT_WUOperationsSession exists use that.
  $ci = New-CimInstance -Namespace root/Microsoft/Windows/WindowsUpdate -ClassName MSFT_WUOperationsSession -ErrorAction SilentlyContinue
  if ($ci) {
    $scan = $ci | Invoke-CimMethod -MethodName ScanForUpdates -Arguments @{SearchCriteria='IsInstalled=0';OnlineScan=$true}
    if ($scan.Updates.Count -eq 0) {
      Write-Output 'No updates to install'
      return $false
    }
    Write-Output "Downloading $($scan.Updates.Count) updates."
    $download = $ci | Invoke-CimMethod -MethodName DownloadUpdates -Arguments @{Updates=$scan.Updates}
    Write-Output "Download finished with HResult: $($download.HResult)"
    Write-Output "Installing $($scan.Updates.Count) updates."
    $install = $ci | Invoke-CimMethod -MethodName InstallUpdates -Arguments @{Updates=$scan.Updates}
    Write-Output "Install finished with HResult: $($install.HResult)"
    Write-Output 'Finished Windows update.'

    return
  }

  # In 2008 R2 the initial search can fail with error 0x80244010, something
  # to do with the number of trips the client is making to the WSUS server.
  # Trying the search again fixes this. Searching around the net didn't
  # yield any actual fixes other than trying again. It does seem to be a
  # somewhat common issue that has been around for years and has no other fix.
  # http://blogs.technet.com/b/sus/archive/2008/09/18/wsus-clients-fail-with-warning-syncserverupdatesinternal-failed-0x80244010.aspx
  $session = New-Object -ComObject 'Microsoft.Update.Session'
  $query = 'IsInstalled=0'
  $searcher = $session.CreateUpdateSearcher()
  $i = 1
  while ($i -lt 10) {
    try {
      Write-Output "Searching for updates, try $i."
      $updates = $searcher.Search($query).Updates
      break
    } catch {
      Write-Output 'Update search failed.'
      $i++
      if ($i -ge 10) {
        Write-Output 'Searching for updates one last time.'
        $updates = $searcher.Search($query).Updates
      }
    }
  }

  if ($updates.Count -eq 0) {
    Write-Output 'No updates required!'
    return $false
  }
  else {
    foreach ($update in $updates) {
      if (-not ($update.EulaAccepted)) {
        Write-Output 'The following update required a EULA to be accepted:'
        Write-Output '----------------------------------------------------'
        Write-Output ($update.Description)
        Write-Output '----------------------------------------------------'
        Write-Output ($update.EulaText)
        Write-Output '----------------------------------------------------'
        $update.AcceptEula()
      }
    }
    $count = $updates.Count
    if ($count -eq 1) {
      # Sometimes we have a bug where we get stuck on one update. Let's
      # log what this one update is in case we are having trouble with it.
      Write-Output 'Downloading the following update:'
      Write-Output ($updates | Out-String)
    }
    else {
      Write-Output "Downloading $count updates."
    }
    $downloader = $session.CreateUpdateDownloader()
    $downloader.Updates = $updates
    $downloader.Download()
    Write-Output 'Download complete. Installing updates.'
    $installer = $session.CreateUpdateInstaller()
    $installer.Updates = $updates
    $installer.AllowSourcePrompts = $false
    $result = $installer.Install()
    $hresult = $result.HResult
    Write-Output "Install Finished with HResult: $hresult"
    Write-Output 'Finished Windows update.'
  }
}

function Setup-NTP {
  Write-Output 'Translate: Setting up NTP.'

  # Set the CMOS clock to use UTC.
  $tzi_path = 'HKLM:\SYSTEM\CurrentControlSet\Control\TimeZoneInformation'
  Set-ItemProperty -Path $tzi_path -Name RealTimeIsUniversal -Value 1

  # Set up time sync...
  # Stop in case it's running; it probably won't be.
  Stop-Service W32time
  $w32tm = "$env:windir\System32\w32tm.exe"

  # Get time from GCE NTP server every 15 minutes.
  Run-Command $w32tm /config '/manualpeerlist:metadata.google.internal,0x1' /syncfromflags:manual
  Set-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\W32Time\TimeProviders\NtpClient' `
    -Name SpecialPollInterval -Value 900
  # Set in Control Panel -- Append to end of list, set default.
  $server_key = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\DateTime\Servers'
  $server_item = Get-Item $server_key
  $server_num = ($server_item.GetValueNames() | Measure-Object -Maximum).Maximum + 1
  Set-ItemProperty -Path $server_key -Name $server_num -Value 'metadata.google.internal'
  Set-ItemProperty -Path $server_key -Name '(Default)' -Value $server_num
  # Configure to run automatically on every start.
  Set-Service W32Time -StartupType Automatic
  Run-Command $env:windir\system32\sc.exe triggerinfo w32time start/networkon
  Write-Output 'Configured W32Time to use GCE NTP server.'

  # Sync time now.
  Start-Service W32time
  Run-Command $w32tm /resync
}

function Configure-Network {
  <#
    .SYNOPSIS
      Apply GCE networking related configuration changes.
  #>

  Write-Output 'Configuring network.'

  # Make sure metadata server is in etc/hosts file.
  Add-Content $script:hosts_file @'

# Google Compute Engine metadata server
    169.254.169.254    metadata.google.internal metadata

'@

  Write-Output 'Changing firewall settings.'
  # Change Windows Server firewall settings.
  # Enable ping in Windows Server 2008.
  Run-Command netsh advfirewall firewall add rule `
      name='ICMP Allow incoming V4 echo request' `
      protocol='icmpv4:8,any' dir=in action=allow

  # Enable inbound communication from the metadata server.
  Run-Command netsh advfirewall firewall add rule `
      name='Allow incoming from GCE metadata server' `
      protocol=ANY remoteip=169.254.169.254 dir=in action=allow

  # Enable outbound communication to the metadata server.
  Run-Command netsh advfirewall firewall add rule `
      name='Allow outgoing to GCE metadata server' `
      protocol=ANY remoteip=169.254.169.254 dir=out action=allow

  # Change KeepAliveTime to 5 minutes.
  $tcp_params = 'HKLM:\System\CurrentControlSet\Services\Tcpip\Parameters'
  New-ItemProperty -Path $tcp_params -Name 'KeepAliveTime' -Value 300000 -PropertyType DWord

  Write-Output 'Disabling WPAD.'

  # Mount default user registry hive at HKLM:\DefaultUser.
  Run-Command reg load 'HKLM\DefaultUser' 'C:\Users\Default\NTUSER.DAT'

  # Loop over default user and current (SYSTEM) user.
  foreach ($reg_base in 'HKLM\DefaultUser', 'HKCU') {
    # Disable Web Proxy Auto Discovery.
    $WPAD = "$reg_base\Software\Microsoft\Windows\CurrentVersion\Internet Settings"

    # Make change with reg add, because it will work with the mounted hive and
    # because it will recursively add any necessary subkeys.
    Run-Command reg add $WPAD /v AutoDetect /t REG_DWORD /d 0
  }

  # Unmount default user hive.
  Run-Command reg unload 'HKLM\DefaultUser'
}

function Configure-Power {
  <#
    .SYNOPSIS
      Change power settings to never turn off monitor.
  #>

  # Change power configuration to never turn off monitor.  If Windows turns
  # off its monitor, it will respond to power button pushes by turning it back
  # on instead of shutting down as GCE expects.  We fix this by switching the
  # "Turn off display after" setting to 0 for all power configurations.
  Get-CimInstance -Namespace 'root\cimv2\power' -ClassName Win32_PowerSettingDataIndex -ErrorAction SilentlyContinue | ForEach-Object {
    $power_setting = $_ | Get-CimAssociatedInstance -ResultClassName 'Win32_PowerSetting' -OperationTimeoutSec 10 -ErrorAction SilentlyContinue
    if ($power_setting -and $power_setting.ElementName -eq 'Turn off display after') {
      Write-Output ('Changing power setting ' + $_.InstanceID)
      $_ | Set-CimInstance -Property @{SettingIndexValue = 0}
    }
  }
}

function Change-InstanceProperties {
  <#
    .SYNOPSIS
      Apply GCE specific changes.

    .DESCRIPTION
      Apply GCE specific changes to this instance.
  #>

  Write-Output 'Setting instance properties.'

  # Enable EMS.
  Run-Command bcdedit /emssettings EMSPORT:2 EMSBAUDRATE:115200
  Run-Command bcdedit /ems on

  # Ignore boot failures.
  Run-Command bcdedit /set '{current}' bootstatuspolicy ignoreallfailures
  Write-Output 'bcdedit option set.'

  # Registry fix for PD cluster size issue.
  $vioscsi_path = 'HKLM:\SYSTEM\CurrentControlSet\Services\vioscsi\Parameters\Device'
  if (-not (Test-Path $vioscsi_path)) {
    New-Item -Path $vioscsi_path
  }
  New-ItemProperty -Path $vioscsi_path -Name EnableQueryAccessAlignment -Value 1 -PropertyType DWord

  # Change SanPolicy. Setting is persistent even after sysprep. This helps in
  # ensuring all attached disks are online when instance is built.
  $san_policy = 'san policy=OnlineAll' | diskpart | Select-String 'San Policy'
  Write-Output ($san_policy -replace '(?<=>)\s+(?=<)') # Remove newline and tabs

  # Prevent password from expiring after 42 days.
  Run-Command net accounts /maxpwage:unlimited

  # Change time zone to Coordinated Universal Time.
  Run-Command tzutil /s 'UTC'

  # Set pagefile to 1GB
  Get-CimInstance Win32_ComputerSystem | Set-CimInstance -Property @{AutomaticManagedPageFile=$False}
  Get-CimInstance Win32_PageFileSetting | Set-CimInstance -Property @{InitialSize=1024; MaximumSize=1024}

  # Disable Administartor user.
  Run-Command net user Administrator /ACTIVE:NO

  # Set minimum password length.
  Run-Command net accounts /MINPWLEN:8
}

function Configure-RDPSecurity {
  $registryPath = 'HKLM:\SYSTEM\CurrentControlSet\Control\Terminal Server\WinStations\RDP-Tcp'

  # Set minimum encryption level to "High"
  New-ItemProperty -Path $registryPath -Name MinEncryptionLevel -Value 3 -PropertyType DWORD -Force
  # Specifies that Network-Level user authentication is required.
  New-ItemProperty -Path $registryPath -Name UserAuthentication -Value 1 -PropertyType DWORD -Force
  # Specifies that the Transport Layer Security (TLS) protocol is used by the server and the client
  # for authentication before a remote desktop connection is established.
  New-ItemProperty -Path $registryPath -Name SecurityLayer -Value 2 -PropertyType DWORD -Force
}

function Install-Packages {
  Write-Output 'Installing GCE packages...'
  # Install each individually in order to catch individual errors
  Run-Command 'C:\ProgramData\GooGet\googet.exe' -root 'C:\ProgramData\GooGet' -noconfirm install google-compute-engine-windows
  Run-Command 'C:\ProgramData\GooGet\googet.exe' -root 'C:\ProgramData\GooGet' -noconfirm install google-compute-engine-powershell
  Run-Command 'C:\ProgramData\GooGet\googet.exe' -root 'C:\ProgramData\GooGet' -noconfirm install google-compute-engine-sysprep
  Run-Command 'C:\ProgramData\GooGet\googet.exe' -root 'C:\ProgramData\GooGet' -noconfirm install google-compute-engine-auto-updater
  Run-Command 'C:\ProgramData\GooGet\googet.exe' -root 'C:\ProgramData\GooGet' -noconfirm install certgen
  Run-Command 'C:\ProgramData\GooGet\googet.exe' -root 'C:\ProgramData\GooGet' -noconfirm install google-compute-engine-driver-gvnic
  Run-Command 'C:\ProgramData\GooGet\googet.exe' -root 'C:\ProgramData\GooGet' -noconfirm install google-compute-engine-driver-vioscsi
  Run-Command 'C:\ProgramData\GooGet\googet.exe' -root 'C:\ProgramData\GooGet' -noconfirm install google-compute-engine-driver-netkvm
  Run-Command 'C:\ProgramData\GooGet\googet.exe' -root 'C:\ProgramData\GooGet' -noconfirm install google-compute-engine-driver-pvpanic
  Run-Command 'C:\ProgramData\GooGet\googet.exe' -root 'C:\ProgramData\GooGet' -noconfirm install google-compute-engine-driver-gga

  $product_name = (Get-ItemProperty -Path 'HKLM:\Software\Microsoft\Windows NT\CurrentVersion' -Name ProductName).ProductName
  if ($product_name -match 'Windows (Web )?Server (2008 R2|2012 R2|2016|2019|Standard|Datacenter)') {
    Run-Command 'C:\ProgramData\GooGet\googet.exe' -root 'C:\ProgramData\GooGet' -noconfirm install google-compute-engine-vss
  }
}

function Set-Repos {
  Write-Output 'Setting GooGet repos to stable.'
  Remove-Item 'C:\ProgramData\GooGet\repos' -Recurse -Force
  Run-Command 'C:\ProgramData\GooGet\googet.exe' -root 'C:\ProgramData\GooGet' addrepo 'google-compute-engine-stable' 'https://packages.cloud.google.com/yuck/repos/google-compute-engine-stable'
}

function Export-ImageMetadata {
  $computer_info = Get-ComputerInfo
  $version = $computer_info.OsVersion
  $family = 'windows-' + $computer_info.windowsversion
  $name =  $computer_info.OSName
  $release_date = (Get-Date).ToUniversalTime()
  $image_metadata = @{'family' = $family;
                      'version' = $edition;
                      'name' = $name;
                      'location' = ${script:outs_dir};
                      'build_date' = $release_date;
                      'packages' = @()}

  # Get Googet packages.
  $out = Run-Command 'C:\ProgramData\GooGet\googet.exe' -root 'C:\ProgramData\GooGet' 'installed'
  $out = $out[1..$out.length]
  [array]::sort($out)

  foreach ($package_line in $out) {
    $name = $package_line.split(' ')[0]
    # Get Package Info for each package
    $info = Run-Command 'C:\ProgramData\GooGet\googet.exe' -root 'C:\ProgramData\GooGet' 'installed' '-info' $name
    $version = $info[2]
    $source = $info[6]
    $package_metadata = @{'name' = $name;
                          'version' = $version;
                          'commmit_hash' = $source}
    $image_metadata['packages'] += $package_metadata
  }

  # Save the JSON image_metadata.
  $image_metadata_json = $image_metadata | ConvertTo-Json -Compress
  $image_metadata_json | & 'gsutil' -m cp - "${script:outs_dir}/metadata.json"
}


try {
  Write-Output 'Beginning post install powershell script.'

  $script:outs_dir = Get-MetadataValue -key 'daisy-outs-path'

  Install-WindowsUpdates
  if (Reboot-Required) {
    Write-Output 'Reboot required.'
    shutdown /r /t 00
    exit
  }

  Change-InstanceProperties
  Configure-Network
  Configure-Power
  Configure-RDPSecurity
  Setup-NTP
  Install-Packages
  Set-Repos
  Export-ImageMetadata

  Write-Output 'Launching sysprep.'
  & "$script:gce_install_dir\sysprep\gcesysprep.bat"
}
catch {
  Write-Output 'Exception caught in script:'
  Write-Output $_.InvocationInfo.PositionMessage
  Write-Output "Message: $($_.Exception.Message)"
  Write-Output 'Windows build failed.'
  exit 1
}
