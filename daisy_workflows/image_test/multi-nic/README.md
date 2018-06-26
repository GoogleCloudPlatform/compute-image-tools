# What is being tested?

Network configuration. When having more than one Network Interface Card (NIC),
the agent should correctly configure the routes for **all** of the network
cards.

# How this test works?

Two VMS with 2 NICs will share only one NIC with a common network. Communication
between the two should be able to take place (the master instance can ping the
slave instance).

# Setup

By default, this test assumes that 3 network exists on your project:

- "multi-nic-test-network-1" with a subnetwork "a"
- "multi-nic-test-network-2" with a subnetwork "b"
- "multi-nic-test-network-3" with a subnetwork "c"

If another naming is used, change the variables "network_1", "subnetwork_1", "network_2",
"subnetwork_2", "common_network" and "common_subnetwork".
