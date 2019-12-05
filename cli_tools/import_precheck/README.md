# Image Import Precheck Tool
Precheck runs on your virtual machine before attempting to import it into
Google Compute Engine (GCE). It attempts to identify compatibility issues that
will either cause import to fail or will cause potentially unexpected behavior
post-import. See our [image import documentation](https://googlecloudplatform.github.io/compute-image-tools/image-import.html)
for more information.

Precheck must be run as root or Administrator on the running system you want to import.

## Binaries
Windows: https://storage.googleapis.com/compute-image-tools/release/windows/import_precheck.exe

Linux: https://storage.googleapis.com/compute-image-tools/release/linux/import_precheck

## Building from Source
`go get -u github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/import_precheck`

Then, `$GOPATH/bin/import_precheck`.

Or, if `$GOPATH/bin` is in your `$PATH`, just `import_precheck`.
