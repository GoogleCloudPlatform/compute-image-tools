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

$script:gce_install_dir = 'C:\Program Files\Google\Compute Engine'
$script:hosts_file = "$env:windir\system32\drivers\etc\hosts"

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

function Remove-VMWareTools {
  Get-ChildItem HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall | Foreach-Object {
    if ((Get-ItemProperty $_.PSPath).DisplayName -eq 'VMWare Tools') {
      Write-Output 'Translate: Found VMWare Tools installed, removing...'
      Start-Process msiexec.exe -ArgumentList @('/x', $_.PSChildName, '/quiet', '/norestart') -Wait -ErrorAction SilentlyContinue
      Restart-Computer -Force
      exit 0
    }
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
  Write-Output 'Translate: Configuring network.'

  # Register netkvmco.dll.
  Run-Command rundll32 'netkvmco.dll,RegisterNetKVMNetShHelper'

  # Make sure metadata server is in etc/hosts file.
  Add-Content $script:hosts_file @'

# Google Compute Engine metadata server
    169.254.169.254    metadata.google.internal metadata

'@

  # Change KeepAliveTime to 5 minutes.
  $tcp_params = 'HKLM:\System\CurrentControlSet\Services\Tcpip\Parameters'
  New-ItemProperty -Path $tcp_params -Name 'KeepAliveTime' -Value 300000 -PropertyType DWord -Force

  # Disable IPv6
  Write-Output 'Disabling IPv6.'
  $ipv_path = 'HKLM:\SYSTEM\CurrentControlSet\services\TCPIP6\Parameters'
  Set-ItemProperty -Path $ipv_path -Name 'DisabledComponents' -Value 0xFF

  Write-Output 'Disabling WPAD.'

  # Mount default user registry hive at HKLM:\DefaultUser.
  Run-Command reg load 'HKLM\DefaultUser' 'C:\Users\Default\NTUSER.DAT'

  # Loop over default user and current (SYSTEM) user.
  foreach ($reg_base in 'HKLM\DefaultUser', 'HKCU') {
    # Disable Web Proxy Auto Discovery.
    $WPAD = "$reg_base\Software\Microsoft\Windows\CurrentVersion\Internet Settings"
    # Make change with reg add, because it will work with the mounted hive and
    # because it will recursively add any necessary subkeys.
    Run-Command reg add $WPAD /v AutoDetect /t REG_DWORD /d 0 /f
  }

  # Unmount default user hive.
  Run-Command reg unload 'HKLM\DefaultUser'

  $netkvm = Get-WMIObject Win32_NetworkAdapter -filter "ServiceName='netkvm'"
  $netkvm | ForEach-Object {
    Run-Command netsh interface ipv4 set interface "$($_.NetConnectionID)" mtu=1460 | Out-Null
  }
  Write-Output 'MTU set to 1460.'

  Run-Command route /p add 169.254.169.254 mask 255.255.255.255 0.0.0.0 if $netkvm[0].InterfaceIndex metric 1 -ErrorAction SilentlyContinue
  Write-Output 'Added persistent route to metadata netblock via first netkvm adapter.'
}

function Configure-Power {
  if (-not (Get-Command Get-CimInstance -ErrorAction SilentlyContinue)) {
    return
  }

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
  Write-Output 'Translate: Setting instance properties.'

  # Enable EMS.
  Run-Command bcdedit /emssettings EMSPORT:2 EMSBAUDRATE:115200
  Run-Command bcdedit /ems on

  # Ignore boot failures.
  Run-Command bcdedit /set '{current}' bootstatuspolicy ignoreallfailures
  Write-Output 'bcdedit option set.'

  # Registry fix for PD cluster size issue.
  $vioscsi_path = 'HKLM:\SYSTEM\CurrentControlSet\Services\vioscsi\Parameters\Device'
  New-Item -Path $vioscsi_path -Force
  New-ItemProperty -Path $vioscsi_path -Name EnableQueryAccessAlignment -Value 1 -PropertyType DWord -Force

  # Change SanPolicy. Setting is persistent even after sysprep. This helps in
  # ensuring all attached disks are online when instance is built.
  $san_policy = 'san policy=OnlineAll' | diskpart | Select-String 'San Policy'
  Write-Output ($san_policy -replace '(?<=>)\s+(?=<)') # Remove newline and tabs

  # Change time zone to Coordinated Universal Time.
  Run-Command tzutil /s 'UTC'

  # Not supported on 6.1 client, but is supported on 6.1 server
  $pn_path = 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion'
  $pn = (Get-ItemProperty -Path $pn_path -Name ProductName).ProductName
  if ($pn -notlike '*Windows 7*') {
    Run-Command powercfg /hibernate off
  }
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

function Enable-RemoteDesktop {
  $ts_path = 'HKLM:\SYSTEM\CurrentControlSet\Control\Terminal Server'
  if (-not (Test-Path $ts_path)) {
    return
  }
  # Enable remote desktop in registry.
  Set-ItemProperty -Path $ts_path -Name 'fDenyTSConnections' -Value 0 -Force

  Write-Output 'Disabling Ctrl + Alt + Del.'
  Set-ItemProperty -Path 'HKLM:\Software\Microsoft\Windows\CurrentVersion\Policies\System' -Name 'DisableCAD' -Value 1 -Force

  Write-Output 'Enable RDP firewall rules.'
  Run-Command netsh advfirewall firewall set rule group='@FirewallAPI.dll,-28752' new enable=Yes
}

function Install-Packages {
  Run-Command 'C:\ProgramData\GooGet\googet.exe' -root 'C:\ProgramData\GooGet' -noconfirm install googet
  # We always install google-compute-engine-sysprep because it is required for instance activation, it gets removed later
  # if install_packages is set to false.
  Run-Command 'C:\ProgramData\GooGet\googet.exe' -root 'C:\ProgramData\GooGet' -noconfirm install google-compute-engine-sysprep
  if ($script:install_packages.ToLower() -eq 'true') {
    Write-Output 'Translate: Installing GCE packages...'
    # Install each individually in order to catch individual errors
    Run-Command 'C:\ProgramData\GooGet\googet.exe' -root 'C:\ProgramData\GooGet' -noconfirm install google-compute-engine-windows
    Run-Command 'C:\ProgramData\GooGet\googet.exe' -root 'C:\ProgramData\GooGet' -noconfirm install google-compute-engine-auto-updater
    Run-Command 'C:\ProgramData\GooGet\googet.exe' -root 'C:\ProgramData\GooGet' -noconfirm install google-compute-engine-driver-balloon -ErrorAction SilentlyContinue
    Run-Command 'C:\ProgramData\GooGet\googet.exe' -root 'C:\ProgramData\GooGet' -noconfirm install google-compute-engine-driver-pvpanic -ErrorAction SilentlyContinue
    Run-Command 'C:\ProgramData\GooGet\googet.exe' -root 'C:\ProgramData\GooGet' -noconfirm install google-compute-engine-vss -ErrorAction SilentlyContinue
  }
}

function Install-32bitPackages {
  & C:\ProgramData\GooGet\googet.exe -root C:\ProgramData\GooGet -noconfirm install C:\ProgramData\GooGet\components\googet-x86.x86_32.2.16.3@1.goo
  # We always install google-compute-engine-sysprep because it is required for instance activation, it gets removed later
  # if install_packages is set to false.
  & C:\ProgramData\GooGet\googet.exe -root C:\ProgramData\GooGet -noconfirm install C:\ProgramData\GooGet\components\google-compute-engine-powershell.noarch.1.1.1@4.goo
  & C:\ProgramData\GooGet\googet.exe -root C:\ProgramData\GooGet -noconfirm install C:\ProgramData\GooGet\components\certgen-x86.x86_32.1.0.0@2.goo
  & C:\ProgramData\GooGet\googet.exe -root C:\ProgramData\GooGet -noconfirm install C:\ProgramData\GooGet\components\google-compute-engine-sysprep.noarch.3.10.1@1.goo
  & C:\ProgramData\GooGet\googet.exe -root C:\ProgramData\GooGet -noconfirm install -reinstall C:\ProgramData\GooGet\components\google-compute-engine-metadata-scripts-x86.x86_32.4.2.1@1.goo
  if ($script:install_packages.ToLower() -eq 'true') {
    Write-Output 'Translate: Installing GCE packages...'
    # Install each individually in order to catch individual errors
    & C:\ProgramData\GooGet\googet.exe -root C:\ProgramData\GooGet -noconfirm install C:\ProgramData\GooGet\components\google-compute-engine-windows-x86.x86_32.4.6.0@1.goo
  }
}

try {
  Write-Output 'Translate: Beginning translate PowerShell script.'
  $script:outs_dir = Get-MetadataValue -key 'daisy-outs-path'
  $script:install_packages = Get-MetadataValue -key 'install-gce-packages'
  $script:sysprep = Get-MetadataValue -key 'sysprep'
  $script:is_byol = Get-MetadataValue -key 'is_byol'
  $script:is_x86 = Get-MetadataValue -key 'is_x86'

  Remove-VMWareTools
  Change-InstanceProperties
  Configure-Network
  Setup-NTP
  Configure-RDPSecurity

  if ($script:is_x86.ToLower() -ne 'true') {
    Configure-Power
    Install-Packages
  }
  else {
    # Since 32-bit GooGet packages are not provided via repository, the only option is to install them from a local source.
    Install-32bitPackages 
    # The following function will halt a 32-bit Windows 10 version 1909 import, so skip it.
    $pn_path = 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion'
    $pn = (Get-ItemProperty -Path $pn_path -Name ProductName).ProductName
    Write-Output "Product Name: ${pn}"
    if ($pn -notlike '*Windows 10*') {
      Configure-Power
    }
  }

  # Only needed and applicable to 2008R2.
  $netkvm = Get-WMIObject Win32_NetworkAdapter -filter "ServiceName='netkvm'"
  $netkvm | ForEach-Object {
    & netsh interface ipv4 set dnsservers "$($_.NetConnectionID)" dhcp | Out-Null
  }

  if ($script:sysprep.ToLower() -ne 'true') {
    Enable-RemoteDesktop

    if ($script:is_byol.ToLower() -ne 'true') {
      Write-Output 'Translate: Setting up KMS activation'
      . 'C:\Program Files\Google\Compute Engine\sysprep\activate_instance.ps1' | Out-Null
    }

    if ($script:install_packages.ToLower() -ne 'true') {
      Run-Command 'C:\ProgramData\GooGet\googet.exe' -root 'C:\ProgramData\GooGet' -noconfirm remove google-compute-engine-metadata-scripts
      Run-Command 'C:\ProgramData\GooGet\googet.exe' -root 'C:\ProgramData\GooGet' -noconfirm remove google-compute-powershell
    }
  } else {
    Write-Output 'Translate: Launching sysprep.'
    & 'C:\Program Files\Google\Compute Engine\sysprep\gcesysprep.bat' -NoShutdown
  }

  if ($script:is_byol.ToLower() -eq 'true') {
    'Image imported into GCE using BYOL worklfow' > 'C:\Program Files\Google\Compute Engine\sysprep\byol_image'
  }

  Write-Output 'Translate complete.'
  Stop-Computer -force
  exit 0

}
catch {
  Write-Output 'Exception caught in script:'
  Write-Output $_.InvocationInfo.PositionMessage
  Write-Output "TranslateFailed: $($_.Exception.Message)"
  exit 1
}
