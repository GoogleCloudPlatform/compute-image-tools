#! /bin/bash
version=$(cat google-osconfig-agent.goospec | sed -nE 's/.*"version":.*"(.+)".*/\1/p')
if [[ $? -ne 0 ]]; then
  echo "could not match version in goospec"
  exit 1
fi

pushd ../../
GOOS=windows /tmp/go/bin/go build -ldflags "-X main.version=${version}" -o google_osconfig_agent.exe
popd