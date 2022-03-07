The source for Daisy has moved to [GoogleCloudPlatform/compute-daisy](https://github.com/GoogleCloudPlatform/compute-daisy).

## Quick Migration

For most users, run the following in the same directory as your `go.mod` file:

```bash
old=github.com/GoogleCloudPlatform/compute-image-tools/daisy
new=github.com/GoogleCloudPlatform/compute-daisy
find . -path ./.git -prune -o -name "*.go" \
  -exec sed -i"" "s|$old|$new|g" "{}" +
go get "$new"
go mod tidy
```

## Detailed Migration

1. Run `go get github.com/GoogleCloudPlatform/compute-daisy`
2. Replace imports with the following:

| Old Import                                                         | New Import                                             |
|--------------------------------------------------------------------|--------------------------------------------------------|
| `github.com/GoogleCloudPlatform/compute-image-tools/daisy`         | `github.com/GoogleCloudPlatform/compute-daisy`         |
| `github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute` | `github.com/GoogleCloudPlatform/compute-daisy/compute` |
