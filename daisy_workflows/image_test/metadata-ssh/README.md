# What is being tested?

Tests if metadata parameters related to ssh connections are working properly:

- Ensure that SSH keys specified in project and instance level metadata work to
  grant access, and that the guest supports a consistent key semantics
- Sets keys in project and instance level metadata and verifies login works
  when appropriate.
- Sets combinations of the following keys in instance level metadata:
 - ssh-keys: in addition to any other keys specified.
 - sshKeys: ignores all project level SSH keys.
 - block-project-ssh-keys: ignores all project level SSH keys.

- Sets combinations of the following keys in project level metadata (neither
  are exclusive):
 - ssh-keys
 - sshKeys

# How this test works?

- testee: target image boots and wait for interaction with tester.
- tester: creates keys and tries to execute commands on testee with the
  different methods described above.

# Setup

No setup needed
