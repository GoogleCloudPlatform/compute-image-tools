# GCE Package Integration Test
These workflows will be used to validate how the `google-compute-engine-*`
packages will behave from a specific compute-image-package revision from a
specific git repository when being installed to a system.

It can be useful to validate compute-image-package's
development branch but also to validate Pull Requests from other contributors.

# Approach

1. Build the pointed packages to the desired system.
1. Generate an image from that system with that packages installed.
1. Run integration tests on the generated image
