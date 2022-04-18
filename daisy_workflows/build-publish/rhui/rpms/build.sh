#!/bin/bash

set -ex

PROJECT=$1
if [[ "$#" -ne 1 ]]; then
  echo "Usage: $0 <project name>"
  exit 1
fi

RPMBUILD=/tmp/rpmbuild
rm -rf $RPMBUILD; mkdir -p "$RPMBUILD"/{BUILD,RPMS,SOURCES,SPECS}
TARBALL=/tmp/tarball
rm -rf $TARBALL; mkdir $TARBALL

for cert in *.crt; do
  package=${cert//.crt}

  # Make tar archive directory.
  tardir="${TARBALL}/${package}-4.0"
  mkdir -p "$tardir"

  # Copy sources to tar archive dir.
  cp "${package}.crt" "${tardir}/content.crt"
  cp "${package}.repo" "${tardir}/rh-cloud.repo"
  cp rhui-set-release "${tardir}/rhui-set-release"

  # Get entitlement key, which requires access permissions.
  gcloud secrets versions access latest --secret "${package}-key" --project "$PROJECT" > "${tardir}/key.pem"

  # Create source tar archive.
  ( cd $TARBALL; tar cvzf "${RPMBUILD}/SOURCES/${package}-4.0.tar.gz" "${package}-4.0"; )

  # Copy spec file.
  cp "rh-client-config.spec" "${RPMBUILD}/SPECS/${package}-4.0.spec"

  # Build package.
  rpmbuild -bb \
    --buildroot "${RPMBUILD}/BUILDROOT" \
    --define "_builddir ${RPMBUILD}/BUILD" \
    --define "_sourcedir ${RPMBUILD}/SOURCES" \
    --define "_rpmdir ${RPMBUILD}/RPMS"  \
    --define "_name ${package}" \
    "${RPMBUILD}/SPECS/${package}-4.0.spec"
done

echo "Done. Built RPMs in ${RPMBUILD}/RPMS/"
