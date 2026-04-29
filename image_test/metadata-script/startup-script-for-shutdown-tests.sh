#!/bin/sh

# FreeBSD's kernel aborts if the shutdown process takes more than 90 seconds,
# and for some tests we need more than that.
if [ "$(uname)" = 'FreeBSD' ]; then
    sysctl kern.init_shutdown_timeout=180
    sysrc rcshutdown_timeout="180"
fi

echo "Ready to stop instance."
