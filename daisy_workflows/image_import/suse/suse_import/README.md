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

## Public interface

**Prep**
* Install this package (`pip3 install suse_translate`)
* Mount the guest to `/dev/sdb`.

**Execution**
* Run the `translate-suse` command.

**Metadata variables**
* install_gce_packages: whether to install the GCE guest environment.
    * `true` or `false`; default `true`.
* subscription_model: whether to modify the instance to on-demand pricing (`gce`) or keep
  the currently-attached subscription (`byol`).
    * `gce` or `byol`; default `byol`.

## Code structure

 - Execution starts in `translate.py`, which is exposed as the `translate-suse` command;
see `setup.py` for the command's configuration.
 - The guest is mounted with libguestfs. If a supported version of SLES is contained, translation
continues. `translate.py` has a data structure containing supported versions.
 - If `subscription_model=gce` was specified, the `on_demand` package is used to modify the guest to
use on-demand pricing. This module is responsible for installing the guest environment, since its
execution environment has access to SLES's GCE update server.
 - If `subscription_model=byol`, the guest environment is installed using the guest's existing
subscription.
 - Lastly, optimize the guest to run on GCE. This includes installing drivers, modifying
kernel arguments, and updating the network to use DHCP.

## Tips

### On-demand conversion stops working

Check [SUSE-KB-000019633](https://www.suse.com/support/kb/doc/?id=000019633) for the current
conversion steps. If SUSE updates their tarballs of offline packages, copy the new versions
to the GCS mirror:

 - Project: compute-image-tools
 - Bucket and path: linux_import_tools/sles/{timestamp}

### Working in a chroot

When modifying the chroot structure, subtle errors lead to hard-to-spot bugs. For example:
 - Symlinks pointing out of the chroot
 - File and directory permissions causing unreadable files, or unlistable directories

To assist with this, the `on_demand.validate_chroot` module provides runtime assertions to
validate the structure and content of the chroot.
