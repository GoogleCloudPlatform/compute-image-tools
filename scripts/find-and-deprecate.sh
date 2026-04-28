#!/bin/bash
set -euo pipefail

# This script finds all image destinations in the YAML and runs deprecation.
YAML_FILE="cli_tools_cloudbuild.yaml"
SCRIPT_DIR=$(dirname "$0")

echo "--- Scanning $YAML_FILE for repositories ---"

declare -A REPOS

while read -r line; do
  # Look for the destination pattern
  # Regex matches --destination= then captures the URL up to the colon
  if [[ $line =~ --destination=([^:]+) ]]; then
    RAW_REPO="${BASH_REMATCH[1]}"
    REPO_PATH="${RAW_REPO//\$PROJECT_ID/$PROJECT_ID}"
    REPOS["$REPO_PATH"]=1
  fi
done < "$YAML_FILE"

for REPO in "${!REPOS[@]}"; do
  # Call the deprecate-images script to deprecate
  "$SCRIPT_DIR/deprecate-images.sh" "$REPO"
done