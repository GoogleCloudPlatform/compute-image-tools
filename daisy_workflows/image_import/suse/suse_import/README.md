This package supports translating OpenSUSE and SLES distributions to run on Google Compute Engine.

## SLES subscriptions

Bring-your-own-license (BYOL) and on-demand subscriptions are supported.  When using BYOL, 
the guest is imported without modifications to its license. During import, zypper is used
to install packages, so the license must be active. If the subscription requires an entitlement
server, then the server must reachable.

When importing as on-demand, the guest is configured to use SLES's GCE SCC servers.
An active SLES subscription is not required; if subscriptions are found on the guest,
they are deleted. See  [SUSE-KB-000019633](https://www.suse.com/support/kb/doc/?id=000019633)
for details on the conversion process.
