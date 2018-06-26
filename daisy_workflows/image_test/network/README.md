# What is being tested?

Network configuration. The instance should be able to reach other instances
running on the same network as well as having access to the internet. The
instance should also configure internal routing to accept packages if a
forwarding rule send traffic to it.

# How this test works?

- testee: target that will print hostname on port 80.
- testee-checker: target that verifies VM to VM and VM to external DNS
  connections work.
- tester: will verify if the IP alias and Forwarding Rule is working on testee
  by comparing the hostname given by the IP aliased machine and by the IP that
  the Forwarding Rule should be forwarding traffic to.

# Setup

Some IP alias and mask should be defined. By Default the 10.128.3.128 is used
but keep in mind this IP should be available on the instance subnet. These
parameters can be changed through the variables "alias_ip" and "alias_ip_mask".
