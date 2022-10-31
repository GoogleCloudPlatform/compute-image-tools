# RHUI node image builds

The Red Hat Update Infrastructure is a package mirror service with client TLS
authentication, offered by Red Hat and operated on Cloud Providers. This
directory contains the build workflows and artifacts for creating deployable
node images for RHUI's RHUA and CDS nodes. We do not use the HAProxy node type,
instead offering RHUI behind a Google Cloud Internet Load Balancer.

We are currently building RHUA and CDS nodes using the v4.2 installation media.

## RHUA image builds

The RHUA node is responsible for synchronizing content from RH CDN to an NFS
file store. In `install_rhua.sh` we perform the preparation steps specified in
RHUA installer docs, gather relevant inputs to the installer, patch the
underlying Ansible template, then invoke the `rhui-installer` CLI. After
installation, we configure RHUA to synchronize the specific content repositories
appropriate for RHEL users on Google Cloud, add the [Ops Agent], add our safety
modifications to RHUA service definitions, and add our health check agent used
for both Internet Load Balancers and Managed Instance Groups.

[Ops Agent]: https://cloud.google.com/stackdriver/docs/solutions/agents/ops-agent

### RHUA patch and modifications

The `rhui-installer` principally installs software for the RHUA node, most steps
of which are contained in Ansible playbooks and templates. We modify the install
playbook (via `rhua.patch`) to not require the real NFS mount at image build
time. We also modify the NginX configuration template to include the drop-in
directory, which we use to add our health check endpoint.

#### rhua\_artifacts/create-dirs.conf

Ensures necessary directories on the NFS are created on startup. These
directories are usually only created during the `rhui-installer` process, as it
assumes it is being installed onto the target system. RHUI components can crash
if these directories do not already exist. Added to pulpcore-worker@.service. 

#### rhua\_artifacts/depend-nfs.conf

Adds a mount dependency on the NFS mount. Without this, services could start
without the NFS mounted, causing the RHUA to mirror content onto the local disk
instead, which could fill it. Added to pulpcore-worker@.service.

#### rhua\_artifacts/config.opsagent.yaml

Ops-agent configuration specific to the RHUA node. Specifies certain RHUA
specific logs to stream, and configures ops-agent to monitor the nginx service
using the status endpoint we expose on localhost.

#### rhua\_artifacts/rhui-health-check.{service,timer}

This service and unit timer is used to instantiate the health check service
every minute. The RHUA and CDS versions only differ by the node argument passed
on start.

## CDS image builds.

CDS nodes serve RHUI content to RHEL PAYGO instances on Google Cloud. Requests
are validated using a client TLS cert (mTLS), and content is served off of the
NFS.

In `install_cds.sh` we patch the Ansible playbook and invoke it locally, then
add ops-agent and health check services similar to the RHUA node.

### CDS patch and modifications

Typically a CDS node is provisioned by running `rhui-installer` on a live RHUA
node interactively. It would prepare an answers.yaml and run the cds-register
Ansible playbook against the target host. We patch the playbook so it can be run
locally and provide a stored answers.yaml. The patch also removes the NFS
requirement, same as the RHUA patch.

#### cds\_artifacts/config.opsagent.yaml

Ops-agent configuration specific to the CDS node. Specifies certain CDS specific
logs to stream, and configures ops-agent to monitor the nginx service using the
status endpoint we expose on localhost.

#### cds\_artifacts/rhui-health-check.{service,timer}

This service and unit timer is used to instantiate the health check service
every minute. The RHUA and CDS versions only differ by the node argument passed
on start.

## Common modifications

#### status.nginx.conf

Exposes a status endpoint on localhost, used by ops-agent to monitor the nginx
service. Copied to /etc/nginx/conf.d/

#### health\_check.nginx.conf

Exposes the latest health check service results file, used by MIGs and ILBs to
determine node health.

## Updating the installers

In general, these installers represent a reverse-engineering approach to
provisioning RHUA and CDS nodes automatically. Therefore, updating them may not
be straightforward.

For the RHUA install, we aim to be able to use the `rhui-installer` tool. This
means our patch must not introduce new required parameters in answers.yaml, as
`rhui-installer` validates this file. Review the documentation and the command
line parameters, and make adjustments as necessary. It may be required to read
the code. Currently, `rhui-installer` provides options for up to 3 separate
role-based CAs, but we only use one single CA. We provide our CA cert&key during
installation, allowing the install to provision a new TLS cert for the RHUA node
itself, then we erase the CA key after installation as the RHUA nodes do not
need to be able to act as CAs in production use.

For the CDS install it is more complex, because the 'official' way to provision
a CDS node is to use the interactive `rhui-manager` tool on a running RHUA node.
To get the diff, invocation commands, answers.yaml content etc. we performed one
normal build this way and captured the resulting files and actions. This
involved reading the code, so it may require close inspection on future
upgrades.

As a process, it is convenient to modify the relevant installer script to `exit
0` at the point you would like to inspect, then run the daisy workflow. This
saves you having to manually run all the setup steps.
