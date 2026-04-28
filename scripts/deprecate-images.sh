#!/bin/bash
##
# deprecate-images.sh
# Tags older images in GCR/AR as deprecated using gcloud, keeping the latest alive.
##

set -euo pipefail

# Default to dry run unless explicitly disabled
DRY_RUN=${DRY_RUN:-true}
IMAGE_URL=${1:-}

if [[ -z "$IMAGE_URL" ]]; then
  echo "Usage: [DRY_RUN=false] $0 <image_url>"
  exit 1
fi

if [[ "$DRY_RUN" == "true" ]]; then
  echo "--- DRY RUN MODE ENABLED (No changes will be made) ---"
fi

echo "Scanning $IMAGE_URL for older versions..."

# Get all SHAs, sorted by upload time descending, skipping the latest one.
OLD_SHAS=$(gcloud container images list-tags "$IMAGE_URL" --format='get(digest)' --filter="NOT (tags:deprecated-public-image-* OR tags:latest OR tags:release)")
if [[ -z "$OLD_SHAS" ]]; then
  echo "No older images found for $IMAGE_URL. Skipping."
  exit 0
fi

count=0
for sha_with_prefix in $OLD_SHAS; do
  FULL_SHA=${sha_with_prefix#sha256:}
  TAG="deprecated-public-image-$FULL_SHA"

  if [[ "$DRY_RUN" == "true" ]]; then
    echo "[DRY-RUN] Would tag $IMAGE_URL@$sha_with_prefix as $TAG"
    count=$((count + 1))
  else
    echo "Tagging $IMAGE_URL@$sha_with_prefix as $TAG"
    gcloud container images add-tag "${IMAGE_URL}@${sha_with_prefix}" "${IMAGE_URL}:${TAG}" --quiet
    count=$((count + 1))
  fi
done

echo "---"
if [[ "$DRY_RUN" == "true" ]]; then
  echo "Summary: $count images would have been tagged as deprecated."
else
  echo "Summary: Successfully tagged $count images as deprecated."
fi