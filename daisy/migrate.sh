old=github.com/GoogleCloudPlatform/compute-image-tools/daisy
new=github.com/GoogleCloudPlatform/compute-daisy
grep -rl compute-image-tools/daisy . --exclude-dir=.git |
  xargs sed -i "s|$old|$new|g"
