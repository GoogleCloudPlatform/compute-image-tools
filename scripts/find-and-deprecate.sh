#!/bin/bash
set -euo pipefail

# This script finds all image destinations in the YAML and runs deprecation.
YAML_FILE="cli_tools_cloudbuild.yaml"

echo "--- Scanning $YAML_FILE for repositories ---"

declare -A UNIQUE_REPOS

# Read file line by line
while read -r line; do
  # Look for the destination pattern
  # Regex matches --destination= then captures the URL up to the colon
  if [[ $line =~ --destination=([^:]+) ]]; then
    # Get the captured group
    RAW_REPO="${BASH_REMATCH[1]}"
    
    # Replace the variable with the actual project ID
    REPO_PATH="${RAW_REPO//\$PROJECT_ID/$PROJECT_ID}"
    
    UNIQUE_REPOS["$REPO_PATH"]=1
  fi
done < "$YAML_FILE"

SCRIPT_DIR=$(dirname "$0")

for REPO in "${!UNIQUE_REPOS[@]}"; do
  # Call the deprecate-images script to deprecate
  "$SCRIPT_DIR/deprecate-images.sh" "$REPO"
done