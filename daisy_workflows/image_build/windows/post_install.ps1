# This script can be run after image setup.  It contains commands that
# should be run once when creating an image but do not need to be run
# on every subsequent sysprep.

$ErrorActionPreference = 'Stop'

$script:gce_install_dir = 'C:\Program Files\Google\Compute Engine'
$script:hosts_file = "$env:windir\system32\drivers\etc\hosts"
$script:components_path = 'D:\sbomcomponents\components'

# Windows Updates Registry Key Paths
$windows_update_path = 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\WindowsUpdate'
$windows_update_au_path = "$windows_update_path\AU"

function Run-Command {
   param (
     [Parameter(Mandatory=$true, ValueFromPipelineByPropertyName=$true)]
       [string]$Executable,
     [Parameter(ValueFromRemainingArguments=$true,
                ValueFromPipelineByPropertyName=$true)]
       $Arguments = $null,
   )
   Write-Host "Running $Executable with arguments $Arguments."
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
    $value = ($client.DownloadString($url)).Trim()
    Write-Host "Retrieved metadata for key $key with value $value."
    return $value
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

function Set-WindowsUpdateServer {
  <#
    .SYNOPSIS
      Set the Windows update server to a WSUS server.
  #>

  if (-not (Test-Path $windows_update_path)) {
    New-Item -Path $windows_update_path -Value ""
    New-Item -Path $windows_update_au_path -Value ""

    Write-Host "Set-WindowsUpdateServer: Attempting to set Windows update server to $script:wu_server_url`:$script:wu_server_port"

    $wu_server_address = $script:wu_server_url.Remove(0,$script:wu_server_url.LastIndexOf('/')+1)
    Write-Host "Set-WindowsUpdateServer: Testing connection to $wu_server_address."
    if (-not (Test-Connection $wu_server_address -Count 1 -ErrorAction SilentlyContinue)) {
      if (-not (Test-Connection download.microsoft.com -Count 1 -ErrorAction SilentlyContinue)) {
        throw 'Set-WindowsUpdateServer: Windows update server is not reachable. Cannot complete image build.'
      }
      Write-Host "Set-WindowsUpdateServer: $wu_server_address not reachable, defaulting to Microsoft servers"
      return
    }

    # Set the registry keys to use a WSUS 6.x server.
    New-ItemProperty -Path $windows_update_path -Name WUServer -Value "$script:wu_server_url`:$script:wu_server_port" -PropertyType String
    New-ItemProperty -Path $windows_update_path -Name WUStatusServer -Value "$script:wu_server_url`:$script:wu_server_port" -PropertyType String
    New-ItemProperty -Path $windows_update_au_path -Name UseWUServer -Value 1 -PropertyType DWord
    New-ItemProperty -Path $windows_update_au_path -Name NoAutoUpdate -Value 1 -PropertyType DWord
    Write-Host "Set-WindowsUpdateServer: Set Windows update server to $script:wu_server_url`:$script:wu_server_port, rebooting."
    shutdown /r /t 00
    exit
  }
}

function Reset-WindowsUpdateServer {
  <#
    .SYNOPOSIS
      Reset the Windows update server to default settings and enable automatic updates.
  #>
  Write-Host 'Reset-WindowsUpdateServer: Setting Windows Update to the default value of Microsoft Update.'
  if (Test-Path $windows_update_path) {
    Remove-Item -Path $windows_update_path -Recurse -Force
  }
  New-Item -Path $windows_update_path -Value "" | Out-Null
  New-Item -Path $windows_update_au_path -Value "" | Out-Null
  New-ItemProperty -Path $windows_update_au_path -Name AUOptions -Value 4 -PropertyType DWord | Out-Null
  New-ItemProperty -Path $windows_update_au_path -Name ScheduledInstallDay -Value 0 -PropertyType DWord | Out-Null
  New-ItemProperty -Path $windows_update_au_path -Name ScheduledInstallTime -Value 3 -PropertyType DWord | Out-Null
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
    New-Item -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\QualityCompat' -Type Directory | Out-Null
  }
  New-ItemProperty -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\QualityCompat' -Name 'cadca5fe-87d3-4b96-b7fb-a231484277cc' -Value 0  -PropertyType DWORD -Force | Out-Null

  Write-Host 'Install-WindowsUpdates: Starting Windows update.'

  # In 2008R2 the initial search can fail with error 0x80244010. Retrying the search again generally resolves the issueis.
  $session = New-Object -ComObject 'Microsoft.Update.Session'
  $query = 'IsInstalled=0'
  $searcher = $session.CreateUpdateSearcher()
  $i = 1
  while ($i -lt 10) {
    try {
      Write-Host "Install-WindowsUpdates: Searching for updates, try $i."
      $updates = $searcher.Search($query).Updates

      # Skip Windows 7 optional language pack updates
      if ($pn -like 'Windows 7*') {
        if ($updates.Count -le 37 -and $updates.Count -ge 33) {
          Write-Host 'Install-WindowsUpdates: Windows 7 detected. Skipping ~35 language pack updates.'
          $query = 'IsInstalled=0 and AutoSelectOnWebsites=1'
          continue
        }
      }

      break
    } catch {
      Write-Host 'Install-WindowsUpdates: Update search failed.'
      $i++
      if ($i -ge 10) {
        Write-Host 'Install-WindowsUpdates: Reseting update server'
        Reset-WindowsUpdateServer | Out-Null
        Write-Host 'Install-WindowsUpdates: Searching for updates one last time.'
        $updates = $searcher.Search($query).Updates
      }
    }
  }

  if ($updates.Count -eq 0) {
    Write-Host 'Install-WindowsUpdates: No updates required!'
    return $false
  }

  # Windows 7 may enter a loop with a single update remaining
  if ($pn -like 'Windows 7*') {
    if ($updates.Count -eq 1) {
      Write-Host 'Install-WindowsUpdates: Windows 7 detected. Single update remaining. Displaying and continuing install.'
      foreach ($update in $updates) {
        Write-Host ($update.Description)
     }
     return $false
    }
  }

  # Windows 11 may get stuck installing KB5007651. $pn for Windows 11 reports as Win 10 Enterprise.
  # This is an intended behavior by Microsoft for backwards compatibility.
  # As such we skip the KB here instead of trying to target by $pn.
  if ($updates.Count -eq 1) {
    $productBuildNumber = [Environment]::OSVersion.Version.Build
    $productMajorVersion = [Environment]::OSVersion.Version.Major
    $productMinorVersion = [Environment]::OSVersion.Version.Minor
    if($productMajorVersion -eq 10 -and $productMinorVersion -eq 0 -and $productBuildNumber -ge 22000) {
      foreach ($update in $updates) {
        if ($update.Title -like '*KB5007651*') {
          Write-Host 'Install-WindowsUpdates: KB5007651 detected as a single update remaining. Skipping known issue KB.'
          return $false
        }
      }
    }
  }

  foreach ($update in $updates) {
    if (-not ($update.EulaAccepted)) {
      Write-Host 'The following update required a EULA to be accepted:'
      Write-Host '----------------------------------------------------'
      Write-Host ($update.Description)
      Write-Host '----------------------------------------------------'
      Write-Host ($update.EulaText)
      Write-Host '----------------------------------------------------'
      $update.AcceptEula()
    }
  }

  Write-Host "Install-WindowsUpdates: Downloading and installing $($updates.Count) updates."
  foreach ($update in $updates) {
    Write-Host "Install-WindowsUpdates: Update - Title:$($update.Title), Description:$($update.Description)"
  }

  $downloader = $session.CreateUpdateDownloader()
  $downloader.Updates = $updates
  $download_result = $downloader.Download()
  Write-Host "Install-WindowsUpdates: Download complete. Result: $(Get-ResultCodeDescription $download_result.ResultCode). Installing updates."
  $installer = $session.CreateUpdateInstaller()
  $installer.Updates = $updates
  $installer.AllowSourcePrompts = $false
  $install_result = $installer.Install()
  Write-Host "Install-WindowsUpdates: Update installation completed. Result: $(Get-ResultCodeDescription $install_result.ResultCode)"
  return $true
}

# Remove with Win2012 R2 EOL in Oct 2023. Temporary fix for issue following June 2023 .Net update.
function Install-NetFrameworkCore {
  <#
    .SYNOPSIS
      Checks for Windows Server 2012 and enables the .Net version 3.5 Framework.
  #>
  $productBuildNumber = [Environment]::OSVersion.Version.Build
  $productMajorVersion = [Environment]::OSVersion.Version.Major
  $productMinorVersion = [Environment]::OSVersion.Version.Minor
  if($productMajorVersion -eq 6 -and $productMinorVersion -eq 3 -and $productBuildNumber -eq 9600) {
    Write-Host 'Install-NetFrameworkCore: Enabling .Net Framework version 3.5.'
    Install-WindowsFeature Net-Framework-Core
  }
  else {
    Write-Host 'Install-NetFrameworkCore: Windows Server 2012 R2 not detected. Skipping .Net 3.5 install.'
  }
}

function Get-ResultCodeDescription {
  <#
    .SYNOPSIS
      Returns the description of the Windows Update download/install ResultCode.

    .PARAMETER $ResultCode
      The ResultCode to convert.

    .RETURNS
      The human readable description of the result code.
  #>
  param (
    [Parameter(Mandatory=$true)] [int]$ResultCode
  )
  $Result = switch ($ResultCode) {
    0 { 'Not Started' }
    1 { 'In Progress' }
    2 { 'Succeeded' }
    3 { 'SucceededWithErrors' }
    4 { 'Failed' }
    5 { 'Aborted' }
    default { "Unknown, ResultCode: $ResultCode" }
  }
  return $Result
}

function Setup-NTP {
  <#
    .SYNOPSIS
      Setup NTP sync for GCE.
  #>

  Write-Host 'Configure NTP for GCP.'

  # Set the CMOS clock to use UTC.
  $tzi_path = 'HKLM:\SYSTEM\CurrentControlSet\Control\TimeZoneInformation'
  Set-ItemProperty -Path $tzi_path -Name RealTimeIsUniversal -Value 1

  # Set up time sync...
  # Stop in case it's running; it probably won't be.
  Stop-Service W32time
  # w32tm /unregister is flaky, but using sc delete first helps to clean up
  # the service reliably.
  Run-Command $env:windir\system32\sc.exe delete W32Time

  # Unregister and re-register the service.
  $w32tm = "$env:windir\System32\w32tm.exe"
  Run-Command $w32tm /unregister
  Run-Command $w32tm /register

  # Get time from GCE NTP server every 15 minutes.
  Run-Command $w32tm /config '/manualpeerlist:metadata.google.internal,0x1' /syncfromflags:manual
  Start-Sleep -s 300
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
  Write-Host 'Configured W32Time to use GCE NTP server.'

  # Verify that the W32Time service is correctly installed. This has been
  # a source of flakiness in the past.
  try {
    Get-Service W32Time
  }
  catch {
    throw "Failed to configure NTP. Cannot complete image build: $($_.Exception.Message)"
  }

  # Sync time now.
  Start-Service W32time
  Run-Command $w32tm /resync
}

function Configure-Network {
  <#
    .SYNOPSIS
      Apply GCE networking related configuration changes.
  #>

  Write-Host 'Configuring network.'

  # Make sure metadata server is in etc/hosts file.
  Add-Content $script:hosts_file @'

# Google Compute Engine metadata server
    169.254.169.254    metadata.google.internal metadata

'@

  Write-Host 'Changing firewall settings.'
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

  Write-Host 'Disabling WPAD.'

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

  Write-Host 'Modify power settings to disable monitor power down.'
  Get-CimInstance -Namespace 'root\cimv2\power' -ClassName Win32_PowerSettingDataIndex -ErrorAction SilentlyContinue | ForEach-Object {
    $power_setting = $_ | Get-CimAssociatedInstance -ResultClassName 'Win32_PowerSetting' -OperationTimeoutSec 10 -ErrorAction SilentlyContinue
    # Change power configuration to never turn off monitor.  If Windows turns
    # off its monitor, it will respond to power button pushes by turning it back
    # on instead of shutting down as GCE expects.  We fix this by switching the
    # "Turn off display after" setting to 0 for all power configurations.
    if ($power_setting -and $power_setting.ElementName -eq 'Turn off display after') {
      Write-Host ('Changing power setting ' + $_.InstanceID)
      $_ | Set-CimInstance -Property @{SettingIndexValue = 0}
    }
    # Set the "Sleep button action" setting to 1 for all power configurations
    # so the instance responds to standby requests.
    if ($power_setting -and $power_setting.ElementName -eq 'Sleep button action') {
      Write-Host ('Changing power setting ' + $_.InstanceID)
      $_ | Set-CimInstance -Property @{SettingIndexValue = 1}
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

  Write-Host 'Setting instance properties.'

  # Enable EMS.
  Run-Command bcdedit /emssettings EMSPORT:2 EMSBAUDRATE:115200
  Run-Command bcdedit /ems on

  # Ignore boot failures.
  Run-Command bcdedit /set '{current}' bootstatuspolicy ignoreallfailures
  Write-Host 'bcdedit option set.'

  # Registry fix for PD cluster size issue.
  $vioscsi_path = 'HKLM:\SYSTEM\CurrentControlSet\Services\vioscsi\Parameters\Device'
  if (-not (Test-Path $vioscsi_path)) {
    New-Item -Path $vioscsi_path
  }
  New-ItemProperty -Path $vioscsi_path -Name EnableQueryAccessAlignment -Value 1 -PropertyType DWord

  # Change SanPolicy. Setting is persistent even after sysprep. This helps in
  # ensuring all attached disks are online when instance is built.
  $san_policy = 'san policy=OnlineAll' | diskpart | Select-String 'San Policy'
  Write-Host ($san_policy -replace '(?<=>)\s+(?=<)') # Remove newline and tabs

  # Prevent password from expiring after 42 days.
  Run-Command net accounts /maxpwage:unlimited

  # Change time zone to Coordinated Universal Time.
  Run-Command tzutil /s 'UTC'

  # Register netkvmco.dll.
  Run-Command rundll32 'netkvmco.dll,RegisterNetKVMNetShHelper'

  # Set pagefile to 1GB
  Get-CimInstance Win32_ComputerSystem | Set-CimInstance -Property @{AutomaticManagedPageFile=$False}
  Get-CimInstance Win32_PageFileSetting | Set-CimInstance -Property @{InitialSize=1024; MaximumSize=1024}

  # Disable Administartor user.
  Run-Command net user Administrator /ACTIVE:NO

  # Set minimum password length.
  Run-Command net accounts /MINPWLEN:8

  # Enable access to Windows administrative file share.
  Set-ItemProperty -Path 'HKLM:\Software\Microsoft\Windows\CurrentVersion\Policies\System' -Name 'LocalAccountTokenFilterPolicy' -Value 1 -Force

  # https://support.microsoft.com/en-us/help/4072698/windows-server-guidance-to-protect-against-the-speculative-execution
  # Not enabling by deault for now.
  #New-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\Session Manager\Memory Management' -Name 'FeatureSettingsOverride' -Value 0  -PropertyType DWORD -Force
  #New-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\Session Manager\Memory Management' -Name 'FeatureSettingsOverrideMask' -Value 3  -PropertyType DWORD -Force
}

function Configure-BGInfo {
  <#
    .SYNOPSIS
      Configure the information displayed by BGInfo.
  #>

  New-Item "${script:gce_install_dir}\tools" -Type Directory -ErrorAction SilentlyContinue
  $bginfo_src = "${script:components_path}\BGInfo.exe"
  if (-not (Test-Path $bginfo_src)) {
    return
  }
  Write-Host 'Setting up BGInfo.'

  $bginfo_exe = "${script:gce_install_dir}\tools\BGInfo.exe"
  Copy-Item $bginfo_src $bginfo_exe

  $config = @"
{\rtf1\ansi\ansicpg1252\deff0\deflang1033{\fonttbl{\f0\fnil\fcharset0 Arial;}}
{\colortbl ;\red255\green255\blue255;}
\viewkind4\uc1\pard\fi-2880\li2880\tx2880\cf1\b\fs24 Boot Time:\tab\protect <Boot Time>\protect0\par
CPU:\tab\protect <CPU>\protect0\par
Default Gateway:\tab\protect <Default Gateway>\protect0\par
DHCP Server:\tab\protect <DHCP Server>\protect0\par
DNS Server:\tab\protect <DNS Server>\protect0\par
Free Space:\tab\protect <Free Space>\protect0\par
Host Name:\tab\protect <Host Name>\protect0\par
IE Version:\tab\protect <IE Version>\protect0\par
IP Address:\tab\protect <IP Address>\protect0\par
Logon Domain:\tab\protect <Logon Domain>\protect0\par
Logon Server:\tab\protect <Logon Server>\protect0\par
MAC Address:\tab\protect <MAC Address>\protect0\par
Machine Domain:\tab\protect <Machine Domain>\protect0\par
Memory:\tab\protect <Memory>\protect0\par
Network Card:\tab\protect <Network Card>\protect0\par
Network Type:\tab\protect <Network Type>\protect0\par
OS Version:\tab\protect <OS Version>\protect0\par
Service Pack:\tab\protect <Service Pack>\protect0\par
Snapshot Time:\tab\protect <Snapshot Time>\protect0\par
Subnet Mask:\tab\protect <Subnet Mask>\protect0\par
System Type:\tab\protect <System Type>\protect0\par
User Name:\tab\protect <User Name>\protect0\par
Volumes:\tab\protect <Volumes>\protect0\par
\par
}
"@

  # Mount default user registry hive at HKLM:\DefaultUser.
  Run-Command reg load 'HKLM\DefaultUser' 'C:\Users\Default\NTUSER.DAT'

  # Remove network speed from the background info text.
  Run-Command reg add 'HKLM\DefaultUser\Software\Winternals\BGInfo' /v 'RTF' /d $config /t REG_SZ /f

  # Unmount default user hive.
  Run-Command reg unload 'HKLM\DefaultUser'

  # Set BGinfo to startup.
  $bginfo_lnk = $env:ProgramData + '\Microsoft\Windows\Start Menu\Programs\Startup\BGInfo.lnk'
  $ws_shell = New-Object -COM WScript.Shell
  $shortcut = $ws_shell.CreateShortcut($bginfo_lnk)
  $shortcut.TargetPath = $bginfo_exe
  $shortcut.Arguments = '/accepteula /timer:0 /silent'
  $shortcut.Save()
}

function Configure-RDP {
  Write-Host 'Modifying RDP settings.'
  $ts_path = 'HKLM:\SYSTEM\CurrentControlSet\Control\Terminal Server'
  $registryPath = "${ts_path}\WinStations\RDP-Tcp"

  # Set minimum encryption level to "High"
  New-ItemProperty -Path $registryPath -Name MinEncryptionLevel -Value 3 -PropertyType DWORD -Force
  # Specifies that Network-Level user authentication is required.
  New-ItemProperty -Path $registryPath -Name UserAuthentication -Value 1 -PropertyType DWORD -Force
  # Specifies that the Transport Layer Security (TLS) protocol is used by the server and the client
  # for authentication before a remote desktop connection is established.
  New-ItemProperty -Path $registryPath -Name SecurityLayer -Value 2 -PropertyType DWORD -Force

  # Enable remote desktop in registry.
  Set-ItemProperty -Path $ts_path -Name 'fDenyTSConnections' -Value 0 -Force

  # Disable Ctrl + Alt + Del.
  Set-ItemProperty -Path 'HKLM:\Software\Microsoft\Windows\CurrentVersion\Policies\System' -Name 'DisableCAD' -Value 1 -Force
  Run-Command netsh advfirewall firewall set rule group='remote desktop' new enable=Yes
}

function Install-Packages {
  Write-Host 'Installing GCE packages...'
  # Install each individually in order to catch individual errors
  Run-Command 'C:\ProgramData\GooGet\googet.exe' -root 'C:\ProgramData\GooGet' -noconfirm install google-compute-engine-windows
  Run-Command 'C:\ProgramData\GooGet\googet.exe' -root 'C:\ProgramData\GooGet' -noconfirm install google-compute-engine-powershell
  Run-Command 'C:\ProgramData\GooGet\googet.exe' -root 'C:\ProgramData\GooGet' -noconfirm install google-compute-engine-sysprep
  Run-Command 'C:\ProgramData\GooGet\googet.exe' -root 'C:\ProgramData\GooGet' -noconfirm install certgen
  Run-Command 'C:\ProgramData\GooGet\googet.exe' -root 'C:\ProgramData\GooGet' -noconfirm install google-compute-engine-driver-gvnic
  Run-Command 'C:\ProgramData\GooGet\googet.exe' -root 'C:\ProgramData\GooGet' -noconfirm install google-compute-engine-driver-vioscsi
  Run-Command 'C:\ProgramData\GooGet\googet.exe' -root 'C:\ProgramData\GooGet' -noconfirm install google-compute-engine-driver-netkvm
  Run-Command 'C:\ProgramData\GooGet\googet.exe' -root 'C:\ProgramData\GooGet' -noconfirm install google-compute-engine-driver-pvpanic
  Run-Command 'C:\ProgramData\GooGet\googet.exe' -root 'C:\ProgramData\GooGet' -noconfirm install google-compute-engine-driver-balloon
  Run-Command 'C:\ProgramData\GooGet\googet.exe' -root 'C:\ProgramData\GooGet' -noconfirm install google-compute-engine-diagnostics
  Run-Command 'C:\ProgramData\GooGet\googet.exe' -root 'C:\ProgramData\GooGet' -noconfirm install google-osconfig-agent
  # Google Graphics Array not supported on 2008R2/7 (6.1)
  if ($pn -notlike 'Windows Server 2008*' -or $pn -notlike 'Windows 7*') {
    Write-Host 'Installing GCE virtual display driver...'
    Run-Command 'C:\ProgramData\GooGet\googet.exe' -root 'C:\ProgramData\GooGet' -noconfirm install google-compute-engine-driver-gga
  }

  if ($pn -match 'Windows (Web )?Server (2008 R2|2012 R2|2016|2019|2022|Standard|Datacenter)') {
    Write-Host 'Installing GCE VSS agent and provider...'
    Run-Command 'C:\ProgramData\GooGet\googet.exe' -root 'C:\ProgramData\GooGet' -noconfirm install google-compute-engine-vss
  }

  Configure-BGInfo

  # makecert.exe is only used in 2008R2 images.
  if (Test-Path "${script:components_path}\makecert.exe") {
    Copy-Item "${script:components_path}\makecert.exe" "${script:gce_install_dir}\tools\makecert.exe"
  }
}

function Set-Repos {
  Write-Host 'Setting GooGet repos to stable.'
  Remove-Item 'C:\ProgramData\GooGet\repos' -Recurse -Force
  Run-Command 'C:\ProgramData\GooGet\googet.exe' -root 'C:\ProgramData\GooGet' addrepo 'google-compute-engine-stable' 'https://packages.cloud.google.com/yuck/repos/google-compute-engine-stable'
}

function Optimize-Image {
  <#
    .SYNOPSIS
      Runs storage optimizations operations on the system volume.
    .DESCRIPTION
      Runs Defrag and Retrim commands on the system volume.
      This can reduce disk usage and increase HDD performance.
  #>

  # No trim support in Windows versions prior to Windows 2012.
  # As defrag can possibly enlarge the image without trim,
  # skipping this scenario.
  if ($pn -like 'Windows Server 2008*' -or $pn -like 'Windows 7*') {
    return
  }

  Write-Host "Defragging $($env:SystemDrive)"

  # Defrags C:
  # This should increase performance on HDD disks.
  Optimize-Volume -Verbose -Defrag -DriveLetter $env:SystemDrive[0]

  Write-Host "Retrimming $($env:SystemDrive)"

  # Retrims C:
  # This can reduce disk usage of PD disks, and subsequently of images.
  Optimize-Volume -Verbose -ReTrim -DriveLetter $env:SystemDrive[0]
}

function Generate-NativeImage {
  <#
    .SYOPSIS
      Generates .Net Framework native image.
    .DESCRIPTION
      When .Net framework is updated during windows monthly update,
      the native image should also be regenerated during image build.
      It reduces the CPU load when a new VM is launched from the image.
    .NOTES
      Using PowerShell simple function to match the style.
  #>
  # Searching for ngen.exe location using the .Net CLR default path
  # Ref: https://docs.microsoft.com/en-us/dotnet/framework/migration-guide/versions-and-dependencies
  Write-Host 'Native Image Generation for .Net Framework'
  Write-Host 'Searching for ngen.exe'
  $ngenPath = (Get-ChildItem -Path "$env:SystemRoot\Microsoft.NET" -Recurse -Filter 'ngen.exe' -ErrorAction Stop).FullName
  Write-Host "Found $($ngenPath.count): $(($ngenPath -join ';'))"
  foreach ($ngen in $ngenPath) {
    Write-Host "NGEN start: [$ngen]"
    &$ngen executeQueuedItems /verbose
    &$ngen update /force
    Write-Host "NGEN finish: [$ngen]"
  }
}

function Enable-WinRM {
  if ($pn -like '*Enterprise') {
    Write-Host 'Windows Client detected, enabling WinRM (including on Public networks).'
    & winrm quickconfig -quiet -force
  }
}

function Install-PowerShell {
  if (!(Test-Path HKLM:\SOFTWARE\Microsoft\PowerShellCore\InstalledVersions\)) {
    Write-Host 'Installing PowerShell v7.'
    Start-Process -FilePath msiexec.exe -ArgumentList '/i',"$script:components_path\PowerShell.msi",'/quiet','REGISTER_MANIFEST=1' -Wait
    Write-Host 'PowerShell v7 installed, rebooting.'
    shutdown /r /t 120
    exit
  }
}

try {
  Write-Host 'Beginning post install powershell script.'

  $script:x86 = (Get-MetadataValue -key 'x86-build')
  $script:outs_dir = Get-MetadataValue -key 'daisy-outs-path'
  $script:wu_server_url = Get-MetadataValue -key 'wu_server_url' -default 'none'
  $script:wu_server_port = Get-MetadataValue -key 'wu_server_port' -default '0'

  # Windows Product Name https://renenyffenegger.ch/notes/Windows/versions/index
  $pn = (Get-ItemProperty -Path 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion' -Name ProductName).ProductName

  Install-PowerShell

  # Remove with Win2012 R2 EOL in Oct 2023. Temporary fix for issue following June 2023 .Net update.
  Install-NetFrameworkCore

  if ($script:wu_server_url.StartsWith('http') -and $script:wu_server_port -notlike '0') {
    Set-WindowsUpdateServer
  }
  if (Install-WindowsUpdates) {
    Write-Host 'Install-WindowsUpdates installed updates, rebooting.'
    shutdown /r /t 00
    exit
  }
  if (Reboot-Required) {
    Write-Host 'Reboot-Required returned true, rebooting.'
    shutdown /r /t 00
    exit
  }

  Reset-WindowsUpdateServer
  Change-InstanceProperties
  Configure-Network
  Configure-Power
  Configure-RDP
  Setup-NTP

  # Install script diverges here, since 32-bit googet packages are not in Rapture
  if ($script:x86 -eq 'true') {
    # Skip package install and repo setup, these two sections are still needed
    Configure-BGInfo

    # makecert.exe is only used in 2008R2 images.
    if (Test-Path "${script:components_path}\makecert.exe") {
      Copy-Item "${script:components_path}\makecert.exe" "${script:gce_install_dir}\tools\makecert.exe"
    }
  }
  else {
    Install-Packages
    Set-Repos
  }
  Enable-WinRM
  Generate-NativeImage

  # Only needed and applicable for 2008.
  & netsh interface ipv4 set dnsservers 'Local Area Connection' source=dhcp | Out-Null

  # Required for WMF 5.1 on Windows Server 2008R2
  # https://sccm-zone.com/fix-sysprep-error-on-windows-2008-r2-after-windows-management-framework-5-0-installation-b9e86b4c41e4
  if ($pn -like 'Windows Server 2008*') {
    New-ItemProperty -Path 'HKLM:\SOFTWARE\Microsoft\Windows\StreamProvider' -Name LastFullPayloadTime -Value 0 -PropertyType DWord -Force
  }

  Optimize-Image

  Write-Host 'Launching sysprep.'
  & "$script:gce_install_dir\sysprep\gcesysprep.bat"
}
catch {
  Write-Host 'Exception caught in script:'
  Write-Host $_.InvocationInfo.PositionMessage
  Write-Host "Message: $($_.Exception.Message)"
  Write-Host 'Windows build failed.'
  exit 1
}
