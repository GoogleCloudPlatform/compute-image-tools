# What is being tested?

The image is being tested to be be able to on low resource machine types
(f1-micro, g1-small) and high CPU/Mem machine types (n1-standard-96).

The test consists in writing to a file, rebooting and then testing that the file
is intact.

# How this test works?

The instance prints a custom message "BOOTED" after starting up. After that, a
file is stored on disk with the "REBOOT" content, which will be printed after
reboot to guarantee that the file is intact.

# Setup

No setup needed
