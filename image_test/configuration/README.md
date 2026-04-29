# What is being tested?

Several system configurations:

- Ensure a package can be installed from distro archives (here, using the
  "tree" package as test).
- Ensure that \`irqbalance\` is not installed or running.
- Ensure boot loader configuration for console logging is correct.
- Ensure boot loader kernel command line args (per distro).
- Ensure hostname gets set to the instance name.
- Ensure that rsyslog is installed and configured (if the distro uses rsyslog)
  and that the hostname is properly set in the logs on boot.
- Ensure root password is disabled (/etc/passwd)
- Ensure sshd config has sane default settings:

 - PermitRootLogin no
 - PasswordAuthentication no

- Ensure apt/yum repos are setup for GCE repos.
- Ensure that the network interface MTU is set to 1460.
- Ensure that the NTP server is set to metadata.google.internal.
- Ensure automatic security updates are enabled per distro specs.

 - Unattended upgrades for the Debian security repos, and apt config is correct.
 - Unattended upgrades for the Ubuntu security repos, and apt config is correct.
 - Yum-cron for CentOS/RHEL 6 and 7.

- Ensure sysctl security parameters are set.

 - CheckSecurityParameter('net.ipv4.ip\_forward', 0)
 - CheckSecurityParameter('net.ipv4.tcp\_syncookies', 1)
 - CheckSecurityParameter('net.ipv4.conf.all.accept\_source\_route', 0)
 - CheckSecurityParameter('net.ipv4.conf.default.accept\_source\_route', 0)
 - CheckSecurityParameter('net.ipv4.conf.all.accept\_redirects', 0)
 - CheckSecurityParameter('net.ipv4.conf.default.accept\_redirects', 0)
 - CheckSecurityParameter('net.ipv4.conf.all.secure\_redirects', 1)
 - CheckSecurityParameter('net.ipv4.conf.default.secure\_redirects', 1)
 - CheckSecurityParameter('net.ipv4.conf.all.send\_redirects', 0)
 - CheckSecurityParameter('net.ipv4.conf.default.send\_redirects', 0)
 - CheckSecurityParameter('net.ipv4.conf.all.rp\_filter', 1)
 - CheckSecurityParameter('net.ipv4.conf.default.rp\_filter', 1)
 - CheckSecurityParameter('net.ipv4.icmp\_echo\_ignore\_broadcasts', 1)
 - CheckSecurityParameter('net.ipv4.icmp\_ignore\_bogus\_error\_responses', 1)
 - CheckSecurityParameter('net.ipv4.conf.all.log\_martians', 1)
 - CheckSecurityParameter('net.ipv4.conf.default.log\_martians', 1)
 - CheckSecurityParameter('net.ipv4.tcp\_rfc1337', 1)
 - CheckSecurityParameter('kernel.randomize\_va\_space', 2)

- Test for gcloud/gsutil (some distros won’t have this) and validate that
  versions are up to date.

# How this test works?

It basically tries to verify the parameters as described above. Below are some
explanation about some interesting cases:

- Boot loader configuration for console logging: Ideally, all of following
  parameters should be set but after some discussion, it depends on the distro
  so it basically verifies if there is no regression from the old configuration:
 - `console=ttyS0,38400n8`
 - `scsi_mod.use_blk_mq=Y`
 - `net.ifnames=0`
 - `biosdevname=0`

- Automatic security updates: the `unattended-upgrades` (debian/ubuntu) or
  `yum-cron` (redhat/centos) packages were verified if it's installed, if its
  service is running and if its configured with appropriate parameters.
  by running 

- Sshd config: On ubuntu there is no check for PermitRootLogin=no because not
  allowing root login is the default configuration.

# Setup

No setup is needed.
