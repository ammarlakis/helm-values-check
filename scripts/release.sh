#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "Usage: $0 major|minor|patch" >&2
  exit 1
}

if [ "$#" -ne 1 ]; then
  usage
fi

ARG="$1"

compute_next_from_latest() {
  local bump="$1"
  local latest base major minor patch

  latest="$(git tag --list 'v[0-9]*.[0-9]*.[0-9]*' --sort=-version:refname | head -n1 || true)"
  if [ -z "$latest" ]; then
    latest="v0.0.0"
  fi

  base="${latest#v}"
  IFS='.' read -r major minor patch <<<"$base"

  case "$bump" in
    major)
      major=$((major + 1))
      minor=0
      patch=0
      ;;
    minor)
      minor=$((minor + 1))
      patch=0
      ;;
    patch)
      patch=$((patch + 1))
      ;;
  esac

  echo "v${major}.${minor}.${patch}"
}

TAG=""

case "$ARG" in
  major|minor|patch)
    TAG="$(compute_next_from_latest "$ARG")"
    ;;
  *)
    usage
    ;;
esac

if ! git diff --quiet || ! git diff --cached --quiet; then
  echo "Working tree is not clean. Commit or stash changes first." >&2
  exit 1
fi

if command -v just >/dev/null 2>&1; then
  just test
else
  go test ./...
fi

echo "Creating git tag $TAG"
git tag -a "$TAG" -m "Release $TAG"

echo "Pushing tag $TAG to origin"
git push origin "$TAG"

echo "Tag pushed. GitHub Actions release workflow will build and publish binaries for $TAG."
