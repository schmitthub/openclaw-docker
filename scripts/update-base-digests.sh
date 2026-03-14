#!/usr/bin/env bash
set -euo pipefail

# Updates digest-pinned base images in config/defaults.ts.
# Fetches current multi-arch manifest digests via `docker buildx imagetools inspect`
# and replaces the @sha256:... suffix in-place.
#
# Images are extracted automatically from defaults.ts — no separate list to maintain.
# To change a base image tag (e.g. node:22→24), edit defaults.ts and re-run this script.
#
# Usage: ./scripts/update-base-digests.sh
# Requires: docker (with buildx plugin), jq

DEFAULTS_FILE="config/defaults.ts"

if [ ! -f "$DEFAULTS_FILE" ]; then
  echo "ERROR: $DEFAULTS_FILE not found. Run from repo root." >&2
  exit 1
fi

command -v jq >/dev/null 2>&1 || { echo "ERROR: jq is required but not installed" >&2; exit 1; }
command -v docker >/dev/null 2>&1 || { echo "ERROR: docker is required but not installed" >&2; exit 1; }

# Extract image:tag pairs from defaults.ts by finding all "image:tag@sha256:..." patterns.
# This makes defaults.ts the single source of truth — no separate list to keep in sync.
mapfile -t IMAGES < <(
  grep -oE '"[a-zA-Z0-9_./-]+:[a-zA-Z0-9._-]+@sha256:[a-f0-9]+"' "$DEFAULTS_FILE" \
    | sed 's/"//g' \
    | sed 's/@sha256:[a-f0-9]*//' \
    | sort -u
)

if [ ${#IMAGES[@]} -eq 0 ]; then
  echo "ERROR: no digest-pinned images found in $DEFAULTS_FILE" >&2
  exit 1
fi

echo "Found ${#IMAGES[@]} pinned image(s) in $DEFAULTS_FILE:"
changed=0

for image in "${IMAGES[@]}"; do
  printf "  %-40s " "$image"

  # Get the manifest LIST digest (multi-arch), not a platform-specific digest.
  # docker buildx imagetools inspect returns the list digest, which lets Docker
  # select the correct platform at pull time.
  if ! output=$(docker buildx imagetools inspect "$image" --format '{{json .Manifest}}' 2>&1); then
    echo "ERROR: docker buildx imagetools inspect failed for $image:" >&2
    echo "$output" >&2
    exit 1
  fi

  if ! digest=$(echo "$output" | jq -r '.digest // empty'); then
    echo "ERROR: jq failed to parse manifest for $image" >&2
    exit 1
  fi

  if [ -z "$digest" ]; then
    echo "ERROR: manifest for $image has no digest field. Raw output:" >&2
    echo "$output" >&2
    exit 1
  fi

  if [[ ! "$digest" =~ ^sha256:[a-f0-9]{64}$ ]]; then
    echo "ERROR: unexpected digest format for $image: $digest" >&2
    exit 1
  fi

  if grep -q "$digest" "$DEFAULTS_FILE"; then
    echo "up to date"
  else
    echo "UPDATED -> ${digest:0:19}..."
    changed=$((changed + 1))
  fi

  # Escape dots and slashes for sed pattern matching (\& = matched char)
  escaped_image=$(printf '%s' "$image" | sed 's|[./]|\\&|g')

  # Replace existing pinned reference: "image@sha256:old" -> "image@sha256:new"
  sed -i'' -e "s|\"${escaped_image}@sha256:[a-f0-9]*\"|\"${image}@${digest}\"|g" "$DEFAULTS_FILE"

  # Verify the digest actually landed in the file
  if ! grep -q "$digest" "$DEFAULTS_FILE"; then
    echo "ERROR: sed replacement failed for $image — digest not found in $DEFAULTS_FILE after update" >&2
    exit 1
  fi
done

# Clean up sed backup files (macOS BSD sed -i'' may create them)
rm -f "${DEFAULTS_FILE}-e" "${DEFAULTS_FILE}''"

if [ "$changed" -eq 0 ]; then
  echo "All digests already up to date."
else
  echo "$changed image(s) updated. Review: git diff $DEFAULTS_FILE"
fi
