#!/bin/bash

# Script to commit updated CHANGELOG.md after GoReleaser runs
# Usage: ./scripts/commit-changelog.sh <tag>

set -e

TAG="${1}"
if [[ -z "$TAG" ]]; then
    echo "Usage: $0 <tag>"
    echo "Example: $0 v0.6.0"
    exit 1
fi

# Determine the default branch - try multiple methods for CI reliability
if [[ -n "$GITHUB_REF_NAME" ]]; then
    # In GitHub Actions, use the base ref or default to master
    DEFAULT_BRANCH="${GITHUB_BASE_REF:-master}"
elif git symbolic-ref refs/remotes/origin/HEAD &>/dev/null; then
    # If symbolic ref exists, use it
    DEFAULT_BRANCH=$(git symbolic-ref refs/remotes/origin/HEAD | sed 's@^refs/remotes/origin/@@')
else
    # Fallback to fetching default branch from GitHub API or assume master
    DEFAULT_BRANCH=$(git remote show origin | grep 'HEAD branch' | cut -d' ' -f5 || echo "master")
fi

echo "Target branch: $DEFAULT_BRANCH"

# Check if CHANGELOG.md has changes
if [[ -n "$(git status --porcelain CHANGELOG.md)" ]]; then
    echo "Committing updated CHANGELOG.md for $TAG"

    # Configure git
    git config --global user.name "github-actions[bot]"
    git config --global user.email "github-actions[bot]@users.noreply.github.com"

    # Fetch the latest state of the default branch
    git fetch origin "$DEFAULT_BRANCH"

    # Checkout the default branch (handles detached HEAD)
    git checkout "$DEFAULT_BRANCH"

    # Pull latest changes to avoid conflicts
    git pull origin "$DEFAULT_BRANCH" --rebase || {
        echo "Warning: Could not rebase. Attempting to continue..."
    }

    # Add and commit the changelog
    git add CHANGELOG.md
    git commit -m "chore: update CHANGELOG.md for $TAG [skip ci]"

    # Push to the default branch
    git push origin "$DEFAULT_BRANCH"

    echo "CHANGELOG.md committed and pushed to $DEFAULT_BRANCH"
else
    echo "No changes to CHANGELOG.md to commit"
fi
