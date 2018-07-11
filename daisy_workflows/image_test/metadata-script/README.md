# What is being tested?

If startup and shutdown metadata script are behaving properly.

- Startup scripts

 - Ensure startup scripts from metadata get executed on boot and that the
   script that gets executed matches the metadata script.
 - Ensure startup scripts from URL’s (https) get executed.
 - Ensure the VM did not crash after startup script execution.
 - Ensure a random script (junk content) doesn’t crash.
 - Ensure syslog messages are accurate and get written to syslog
        files (/var/log/syslog or /var/log/messages):
  - start\_message = '{0}-script: INFO Starting {0} scripts.'.format(self.script\_type)
  - finish\_message = '{0}-script: INFO Finished running {0} scripts.'.format(self.script\_type)
  - not\_found = '{0}-script: INFO No {0} scripts found in metadata.'.format(self.script\_type)

- Shutdown script

 - Same as startup but for shutdown.
 - Ensure shutdown scripts execute correctly on shutdown (before rsyslog is
   stopped), get logged to syslog, and finish executing before shutdown occurs.
 - Ensure a shutdown script can run for at least 100 seconds before getting
   killed (make sure to “sync” if you are writing to a file).

# How this test works?

There are 3 flavors of this test and all of them are checking for
the output of messages indicating the existance (or not) of the startup/shutdown
script:

1. Integrity: It sends a script passed via URL that prints its md5 hash. That
   hash is verified for integrity purposes. Additionally the shutdown script
   forces 100s execution time and only then it prints its md5 hash.

1. Junk: It sends via URL random bits of data and verifies that the instance
   tries to execute it but doesn't crash.

1. No-script: It doesn't define a startup/shutdown script.

# Setup

No setup is needed but please make sure to use either
metadata-script-linux.wf.json or metadata-script-windows.wf.json as the other
files are only subworkflows.
