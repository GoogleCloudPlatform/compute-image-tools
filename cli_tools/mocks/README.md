**cli_tools** uses [gomock](https://github.com/golang/mock). The mocks are stored as source code,
and need to be updated whenever the referenced interface is changed.

## Setup

To use existing mocks, no setup is required.

To generate new mocks, or update existing mocks, you need the `mockgen` tool:

```bash
go install github.com/golang/mock/mockgen
export PATH=$PATH:~/go/bin
```

## Example: Updating mocks with go generate

For interfaces within **cli_tools** that require mocking, we
create [go generate](https://go.dev/blog/generate)
definitions. These definitions codify naming convention, and simplify future updates.

For example, to update the mock for [`shell.Executor`](../common/utils/shell/executor.go):

```bash
cd cli_tools/common/utils/shell
go generate ./...
```

## Example: Update external mocks

For interfaces that are external to **cli_tools**, we run `mockgen`
directly.

For example, to update the mock for `daisy.Logger`:

```bash
cd cli_tools/mocks
mockgen -package mocks -mock_names Logger=MockDaisyLogger \
  github.com/GoogleCloudPlatform/compute-image-tools/daisy Logger  > mock_daisy_logger.go
```

Tips:

1. The headers of existing mocks typically include the package and interface names required to
   generate mocks.
2. To avoid namespace clashing, it may be required to use the `-mock_names`
   flag. Executing `git diff` will help you to determine whether that flag is required.

## Best Practices

* If the mock will be used throughout **cli_tools*, consider creating it in `cli_tools/mocks`.
* If the mock is used in a single place (to test a particular component), consider creating it near
  that component. For example: `cli_tools/common/image/importer/mocks/source_mocks.go`