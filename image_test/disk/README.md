# What is being tested?

If the image can handle the following cases related to disks:

-   Ensure that boot disks auto expand up to 2TB for MBR disks.

    -   Ensure that disks that are larger than 2048GB still auto expand
        up to 2048GB but don’t overflow (test a 2049 GB disk).

-   Ensure that NVMe Local SSD disks properly work.
-   Ensure that SCSI Local SSD disks properly work and that Multiqueue
    SCSI is enabled on distros that support it (Debian 9, Ubuntu
    14.04+).
-   Ensure that disks can be attached, mounted, unmounted, and detached.

# How this test works?

- testee: after booting, it'll write a file to disk and power off. The tester
  will restart it after resizing the disk to 2049 GB and the testee will ensure
  that the file is still there and that there are unallocated sectors on the
  disk (hence the resizing also treated the case of not being bigger than
  2048GB)

- tester: it'll resize testee disk after the instance is stopped and after it's
  up and running, it'll verify that attaching disks can be made available during
  run and that detaching them also makes it unavailable.

- testee-local-ssd: verifies if the local-ssd (SCSI and NVMe) is working:
  creates a partition, formats it and writes a file. Then unmounts it, mounts it
  again and checks if the file is there. BONUS: points out as Status if
  multiqueue is enabled on SCSI disks.

# Setup

No setup needed, but be aware that a lot of disk space and several instances
will be created on this test.
