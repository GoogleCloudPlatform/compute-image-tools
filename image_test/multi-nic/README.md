# What is being tested?

Network configuration. When having more than one Network Interface Card (NIC),
the agent should correctly configure the routes for **all** of the network
cards.

# How this test works?

Two VMS with 2 NICs will share only one NIC with a common network. Communication
between the two should be able to take place (the master instance can ping the
slave instance).

# Setup

No setup is needed.
