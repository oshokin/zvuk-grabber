#!/usr/bin/env bash
# Bash script to determine next SemVer based on commit messages since the last tag.
# This script implements semantic versioning by analyzing Git commit messages.

# Exit immediately on error, treat unset variables as errors, and fail on pipe errors.
set -euo pipefail

# Semantic versioning rules (case-insensitive):
#   - any commit subject starting with "major:" -> MAJOR bump
#   - else any starting with "feat:"            -> MINOR bump
#   - else any starting with "fix:"             -> PATCH bump
#   - else any other commits                    -> PATCH bump (default)
# If no tag exists, current version is 1.0.0.
# When bumping MINOR, reset PATCH to 0; when bumping MAJOR, reset MINOR/PATCH to 0.
#
# Standard outputs (stdout by default):
#   LAST_TAG=... (normalized version without v prefix)
#   NEXT_TAG=... (new version with v prefix, e.g., v1.2.3)
#   BUMP=major|minor|patch|none
#   HAS_RELEASE=1|0
#
# If called with --emit-gh-output, writes outputs to $GITHUB_OUTPUT for GitHub Actions.

# Parse command line argument to determine output format.
emit_gh_output=false
if [[ "${1:-}" == "--emit-gh-output" ]]; then
  emit_gh_output=true
fi

# Attempt to find the latest Git tag to determine current version.
found_tag=true
last_tag=""
if last_tag=$(git describe --tags --abbrev=0 2>/dev/null); then
  # Successfully found a tag, use it as current version.
  :
else
  # No tags found, this will be the first release.
  found_tag=false
  last_tag="1.0.0"
fi

# Function to normalize version strings by removing 'v' prefix if present.
# This ensures consistent version parsing regardless of tag format.
normalize_version() {
  echo "$1" | sed 's/^v//'
}

# Get the normalized current version for processing.
curr_ver=$(normalize_version "$last_tag")

# Determine the range of commits to analyze based on tag existence.
if $found_tag; then
  # If tag exists, analyze commits since that tag to HEAD.
  range="${last_tag}..HEAD"
  # Collect commit subjects since the last tag.
  mapfile -t subjects < <(git log --format=%s ${range})
else
  # If no tag exists, analyze all commits in current branch.
  # Use git rev-list to avoid invalid range issues.
  mapfile -t subjects < <(git log --format=%s --all)
fi

# Determine the appropriate version bump based on commit message patterns.
# Enable case-insensitive matching for commit message analysis.
shopt -s nocasematch

# Check for version bump type with precedence: major > feat > fix.
# Default to patch for any commits (even non-semantic ones).
bump="patch"

# First pass: check for major version bump (breaking changes).
for s in "${subjects[@]}"; do
  if [[ $s =~ ^major: ]]; then 
    bump="major"
    break  # Major takes highest precedence, stop searching.
  fi
done

# Second pass: check for minor version bump (new features) if no major found.
if [[ $bump == "patch" ]]; then
  for s in "${subjects[@]}"; do
    if [[ $s =~ ^feat: ]]; then 
      bump="minor"
      break  # Minor found, stop searching.
    fi
  done
fi

# Third pass: check for patch version bump (bug fixes) if no major/minor found.
# Note: We already default to patch, so this preserves explicit fix: commits.
if [[ $bump == "patch" ]]; then
  for s in "${subjects[@]}"; do
    if [[ $s =~ ^fix: ]]; then 
      bump="patch"
      break  # Explicit patch found, maintain patch.
    fi
  done
fi

# Disable case-insensitive matching after commit analysis.
shopt -u nocasematch

# Parse the current version into major, minor, and patch components.
# IFS (Internal Field Separator) splits on dots, read assigns to variables.
IFS='.' read -r major minor patch <<<"$curr_ver"

# Calculate the next version based on the determined bump type.
next_ver="$curr_ver"
case "$bump" in
  major)
    # Major bump: increment major version, reset minor and patch to 0.
    major=$((major+1)); minor=0; patch=0;
    next_ver="$major.$minor.$patch";;
  minor)
    # Minor bump: increment minor version, reset patch to 0.
    minor=$((minor+1)); patch=0;
    next_ver="$major.$minor.$patch";;
  patch)
    # Patch bump: increment patch version only.
    patch=$((patch+1));
    next_ver="$major.$minor.$patch";;
  none)
    # No bump needed, keep current version unchanged.
    :;;
esac

# Determine if a release should be created based on findings.
has_release=0
if ! $found_tag; then
  # First release: always create a release with baseline version.
  has_release=1
  bump="none"
  next_ver="$curr_ver"
elif [[ "$bump" != "none" ]]; then
  # Subsequent releases: only if semantic commits were found.
  has_release=1
fi

# Output results in the appropriate format based on command line arguments.
if $emit_gh_output; then
  # GitHub Actions output format: append to GITHUB_OUTPUT file.
  {
    echo "last_tag=$curr_ver"
    echo "next_tag=v$next_ver"
    echo "bump=$bump"
    echo "has_release=$has_release"
  } >>"$GITHUB_OUTPUT"
else
  # Standard output format: write to stdout for manual use or other scripts.
  printf '%s\n' \
    "LAST_TAG=$curr_ver" \
    "NEXT_TAG=v$next_ver" \
    "BUMP=$bump" \
    "HAS_RELEASE=$has_release"
fi
