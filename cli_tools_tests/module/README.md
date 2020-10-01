Module tests: Similar to unit tests, but with live, non-mocked dependencies.

## Running

**Quick start**

```shell script
# Required once:
gcloud auth application-default login
gcloud config set compute/zone $ZONE
gcloud config set project $PROJECT
cd cli_tools_module_test/testname

# Re-run as needed:
go test
```

**Details**

- The SDK performs **authentication** using the precedence
described in: https://cloud.google.com/docs/authentication/production
- All of the tests cases are marked to run as parallel. See the documentation
on `-parallel` for configuring how many cases run simultaneously:
https://golang.org/pkg/cmd/go/internal/test/
- **Configuration values** may be specified either as environment variables or
as [gcloud config](https://cloud.google.com/sdk/gcloud/reference/config/set).
Environment variables take precedence.


|         | Environment Variable   | gcloud config |
|---------|------------------------|---------------|
| project | GOOGLE\_CLOUD\_PROJECT | project       |
| zone    | GOOGLE\_CLOUD\_ZONE    | compute/zone  |
