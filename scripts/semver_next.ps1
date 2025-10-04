#!/usr/bin/env pwsh
# PowerShell script to determine next SemVer based on commit messages since the last tag.
# This script implements semantic versioning by analyzing Git commit messages.

# Stop execution on any error.
$ErrorActionPreference = "Stop"

# Semantic versioning rules (case-insensitive):
#   - any commit subject starting with "major:" -> MAJOR bump
#   - else any starting with "feat:"            -> MINOR bump
#   - else any starting with "fix:"             -> PATCH bump
#   - else any other commits                    -> PATCH bump (default)
# If no tag exists, current version is 1.0.0 (initial release).
# When bumping MINOR, reset PATCH to 0; when bumping MAJOR, reset MINOR/PATCH to 0.
#
# Standard outputs (stdout by default):
#   LAST_TAG=... (normalized version without v prefix)
#   NEXT_TAG=... (new version with v prefix, e.g., v1.2.3)
#   BUMP=major|minor|patch|none
#   HAS_RELEASE=1|0
#
# If called with --emit-gh-output, writes outputs to $env:GITHUB_OUTPUT for GitHub Actions.

# Parse command line argument to determine output format.
$emitGhOutput = $false
if ($args.Count -gt 0 -and $args[0] -eq "--emit-gh-output") {
    $emitGhOutput = $true
}

# Attempt to find the latest Git tag to determine current version.
$foundTag = $true
$lastTag = ""
try {
    # Successfully found a tag, use it as current version.
    $lastTag = git describe --tags --abbrev=0 2>$null
    if (-not $lastTag) {
        throw "No tags found"
    }
} catch {
    # No tags found, this will be the first release.
    $foundTag = $false
    $lastTag = "1.0.0"
}

# Function to normalize version strings by removing 'v' prefix if present.
# This ensures consistent version parsing regardless of tag format.
function Normalize-Version {
    param([string]$Version)
    return $Version -replace '^v', ''
}

# Get the normalized current version for processing.
$currVer = Normalize-Version -Version $lastTag

# Determine the range of commits to analyze based on tag existence.
$subjects = @()
if ($foundTag) {
    # If tag exists, analyze commits since that tag to HEAD.
    $range = "$lastTag..HEAD"
    # Collect commit subjects since the last tag.
    $subjects = git log --format=%s $range 2>$null
    if (-not $subjects) {
        $subjects = @()
    }
} else {
    # If no tag exists, analyze all commits in current branch.
    $subjects = git log --format=%s --all 2>$null
    if (-not $subjects) {
        $subjects = @()
    }
}

# Convert subjects to array if it's a single string.
if ($subjects -is [string]) {
    $subjects = @($subjects)
}

# Determine the appropriate version bump based on commit message patterns.
# Check for version bump type with precedence: major > feat > fix.
# Default to patch for any commits (even non-semantic ones).
$bump = "patch"

# First pass: check for major version bump (breaking changes).
foreach ($subject in $subjects) {
    if ($subject -match '^major:') {
        $bump = "major"
        break  # Major takes highest precedence, stop searching.
    }
}

# Second pass: check for minor version bump (new features) if no major found.
if ($bump -eq "patch") {
    foreach ($subject in $subjects) {
        if ($subject -match '^feat:') {
            $bump = "minor"
            break  # Minor found, stop searching.
        }
    }
}

# Third pass: check for patch version bump (bug fixes) if no major/minor found.
# Note: We already default to patch, so this preserves explicit fix: commits.
if ($bump -eq "patch") {
    foreach ($subject in $subjects) {
        if ($subject -match '^fix:') {
            $bump = "patch"
            break  # Explicit patch found, maintain patch.
        }
    }
}

# Parse the current version into major, minor, and patch components.
$versionParts = $currVer -split '\.'
$major = [int]$versionParts[0]
$minor = [int]$versionParts[1]
$patch = [int]$versionParts[2]

# Calculate the next version based on the determined bump type.
$nextVer = $currVer
switch ($bump) {
    "major" {
        # Major bump: increment major version, reset minor and patch to 0.
        $major++
        $minor = 0
        $patch = 0
        $nextVer = "$major.$minor.$patch"
    }
    "minor" {
        # Minor bump: increment minor version, reset patch to 0.
        $minor++
        $patch = 0
        $nextVer = "$major.$minor.$patch"
    }
    "patch" {
        # Patch bump: increment patch version only.
        $patch++
        $nextVer = "$major.$minor.$patch"
    }
    "none" {
        # No bump needed, keep current version unchanged.
    }
}

# Determine if a release should be created based on findings.
$hasRelease = 0
if (-not $foundTag) {
    # First release: always create a release with baseline version.
    $hasRelease = 1
    $bump = "none"
    $nextVer = $currVer
} elseif ($bump -ne "none") {
    # Subsequent releases: only if semantic commits were found.
    $hasRelease = 1
}

# Output results in the appropriate format based on command line arguments.
if ($emitGhOutput) {
    # GitHub Actions output format: append to GITHUB_OUTPUT file.
    $outputFile = $env:GITHUB_OUTPUT
    Add-Content -Path $outputFile -Value "last_tag=$currVer"
    Add-Content -Path $outputFile -Value "next_tag=v$nextVer"
    Add-Content -Path $outputFile -Value "bump=$bump"
    Add-Content -Path $outputFile -Value "has_release=$hasRelease"
} else {
    # Standard output format: write to stdout for manual use or other scripts.
    Write-Output "LAST_TAG=$currVer"
    Write-Output "NEXT_TAG=v$nextVer"
    Write-Output "BUMP=$bump"
    Write-Output "HAS_RELEASE=$hasRelease"
}

