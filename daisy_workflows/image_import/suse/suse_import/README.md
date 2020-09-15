This package supports translating OpenSUSE and SLES distributions to run on Google Compute Engine.

## SLES subscriptions

Bring-your-own-license (BYOL) and on-demand subscriptions are supported.  When using BYOL, 
the guest is imported without modifications to its license. During import, zypper is used
to install packages, so the license must be active. If the subscription requires an entitlement
server, then the server must reachable.

When importing as on-demand, the guest is configured to use SLES's GCE SCC servers.
An active SLES subscription is not required; if subscriptions are found on the guest,
they are deleted. See [SUSE-KB-000019633](https://www.suse.com/support/kb/doc/?id=000019633)
for details on the conversion process.


## Offline RPM cache

The conversion steps in [SUSE-KB-000019633](https://www.suse.com/support/kb/doc/?id=000019633)
include manually installing RPMs. To ensure availability of the RPMs,
we have mirrored the tarballs to GCS:

 - Project: compute-image-tools
 - Bucket and path: linux_import_tools/sles/{timestamp}

If SUSE releases a new version of tarballs, copy them to a new `{timestamp}`
directory, and update the tarball metadata in `translate.py`.
