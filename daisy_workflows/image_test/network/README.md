# What is being tested?

Network configuration. The instance should be able to reach other instances
running on the same network as well as having access to the internet. The
instance should also configure internal routing to accept packages if a
forwarding rule send traffic to it.

# How this test works?

- testee: target that will serve 2 simple files on its http port: hostname
  (that returns the hostname) and os (returns linux or windows)
- testee-checker: target that verifies if VM to VM and VM to external DNS
  connections work.
- tester: will verify if the IP alias and Forwarding Rule is working on testee
  by comparing the hostname given by the IP aliased machine and by the IP that
  the Forwarding Rule should be forwarding traffic to. It also tests if IP
  alias works when the configuration is modified while the instance is running.
  - *Note: On Windows systems the IP alias is not tested as those kind of packages are dropped.*

# Setup

Some IP alias and mask should be defined. By Default the 10.128.3.128/31 is used
but keep in mind this IP should be available on the instance subnet. These
parameters can be changed through the variables "alias_ip" and "alias_ip_mask".
