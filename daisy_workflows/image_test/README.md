# GCE Image Integration Test
These are workflows used by the GCE team to automate integration tests
with Linux images and GCE

# Test Cases

Boot

-   Tests images to prove that they boot on low resource machine types
    (f1-micro, g1-small) and high CPU/Mem machine types (n1-highcpu-96.
    n1-highmem-96).
-   Test that reboot works- writes a file, reboots, tests that the file
    is intact.

Metadata SSH

-   Ensure that SSH keys specified in project and instance level
    metadata work to grant access, and that the guest supports a
    consistent key semantics
-   Sets keys in project and instance level metadata and verifies login
    works when appropriate.
-   Sets combinations of the following keys in instance level metadata:

    -   ssh-keys: in addition to any other keys specified.
    -   sshKeys: ignores all project level SSH keys.
    -   block-project-ssh-keys: ignores all project level SSH keys.

-   Sets combinations of the following keys in project level metadata
    (neither are exclusive):

    -   ssh-keys
    -   sshKeys

OS Login SSH

-   Verifies guest behavior when the “enable-login=True” in project
    metadata.
-   Verifies guest module responses:

    -   nss (getent passwd)
    -   authorized keys command (/usr/bin/google\_authorized\_keys)
    -   calls to the metadata server for authorization checks and user
        lookups.

-   End to end test where we:

    -   Verify a user cannot log into a VM.
    -   Set IAM permission on a VM for login.
    -   Log in and verify no sudo.
    -   Add sudo IAM permission
    -   Log in again and verify sudo.

Metadata

-   Startup scripts

    -   Ensure startup scripts from metadata get executed on boot and
        that the script that gets executed matches the metadata
        script.
    -   Ensure startup scripts from URL’s (either https or gs://) get
        executed. Currently on distros that do not have gsutil
        installed, gs:// URL’s will fail.
    -   Ensure the VM did not crash after startup script execution.
    -   Ensure a random script (junk content) doesn’t crash.
    -   Ensure syslog messages are accurate and get written to syslog
        files (/var/log/syslog or /var/log/messages):
        -   start\_message = '{0}-script: INFO Starting {0}
            scripts.'.format(
        -   self.script\_type)
        -   finish\_message = '{0}-script: INFO Finished running {0}
            scripts.'.format(
        -   self.script\_type)
        -   not\_found = '{0}-script: INFO No {0} scripts found in
            metadata.'.format(
        -   self.script\_type)

-   Shutdown script

    -   Same as startup but for shutdown.
    -   Ensure shutdown scripts execute correctly on shutdown (before
        rsyslog is stopped), get logged to syslog, and finish
        executing before shutdown occurs.
    -   Ensure a shutdown script can run for at least 100 seconds before
        getting killed (make sure to “sync” if you are writing to a
        file).

Configuration

-   Ensure a package can be installed from distro archives (\`make\` or
    any other generic package).
-   Ensure that \`irqbalance\` is not installed or running.
-   Ensure boot loader configuration for console logging is correct.
-   Ensure boot loader kernel command line args (per distro).
-   Ensure hostname gets set to the instance name.
-   Ensure that rsyslog is installed and configured (if the distro uses
    rsyslog) and that the hostname is properly set in the logs on boot.
-   Ensure root password is disabled (/etc/passwd)
-   Ensure sshd config has sane default settings:

    -   PermitRootLogin no
    -   PasswordAuthentication no

-   Ensure apt/yum repos are setup for GCE repos.
-   Ensure that the network interface MTU is set to 1460.
-   Ensure that the NTP server is set to metadata.google.internal.
-   Ensure automatic security updates are enabled per distro specs.

    -   Unattended upgrades for the Debian security repos, and apt
        config is correct.
    -   Unattended upgrades for the Ubuntu security repos, and apt
        config is correct.
    -   Yum-cron for CentOS/RHEL 6 and 7.
    -   SUSE?

-   Ensure sysctl security parameters are set.

    -   CheckSecurityParameter('net.ipv4.ip\_forward', 0)
    -   CheckSecurityParameter('net.ipv4.tcp\_syncookies', 1)
    -   CheckSecurityParameter('net.ipv4.conf.all.accept\_source\_route', 0)
    -   CheckSecurityParameter('net.ipv4.conf.default.accept\_source\_route', 0)
    -   CheckSecurityParameter('net.ipv4.conf.all.accept\_redirects', 0)
    -   CheckSecurityParameter('net.ipv4.conf.default.accept\_redirects', 0)
    -   CheckSecurityParameter('net.ipv4.conf.all.secure\_redirects', 1)
    -   CheckSecurityParameter('net.ipv4.conf.default.secure\_redirects', 1)
    -   CheckSecurityParameter('net.ipv4.conf.all.send\_redirects', 0)
    -   CheckSecurityParameter('net.ipv4.conf.default.send\_redirects', 0)
    -   CheckSecurityParameter('net.ipv4.conf.all.rp\_filter', 1)
    -   CheckSecurityParameter('net.ipv4.conf.default.rp\_filter', 1)
    -   CheckSecurityParameter('net.ipv4.icmp\_echo\_ignore\_broadcasts', 1)
    -   CheckSecurityParameter('net.ipv4.icmp\_ignore\_bogus\_error\_responses', 1)
    -   CheckSecurityParameter('net.ipv4.conf.all.log\_martians', 1)
    -   CheckSecurityParameter('net.ipv4.conf.default.log\_martians', 1)
    -   CheckSecurityParameter('net.ipv4.tcp\_rfc1337', 1)
    -   CheckSecurityParameter('kernel.randomize\_va\_space', 2)

-   Test for gcloud/gsutil (some distros won’t have this) and validate
    that versions are up to date.

Disks

-   Ensure that boot disks auto expand up to 2TB for MBR disks.

    -   Ensure that disks that are larger than 2048GB still auto expand
        up to 2048GB but don’t overflow (test a 2049 GB disk).

-   Ensure that NVMe Local SSD disks properly work.
-   Ensure that SCSI Local SSD disks properly work and that Multiqueue
    SCSI is enabled on distros that support it (Debian 9, Ubuntu
    14.04+).
-   Ensure that disks can be attached, mounted, unmounted, and detached.

Networking

-   Ensure VM to VM and VM to external DNS connections work.
-   Ensure routes are added when enabling IP Forwarding (TBD, this may
    expand out).

\
Mutil NIC

-   Ensure two VM’s with multiple interfaces sharing one VPC network can
    talk to each other over the VPC. (second interface is setup and gets
    the correct routes).
