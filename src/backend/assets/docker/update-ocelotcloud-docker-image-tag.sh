#!/bin/bash

set -e

DOCKERFILE="$1"
latest_alpine_version=$(curl -s https://dl-cdn.alpinelinux.org/alpine/latest-stable/releases/x86_64/latest-releases.yaml | grep -oP 'version: \K[0-9]+\.[0-9]+\.[0-9]+' | head -1)

if [ -z "$latest_alpine_version" ]; then
    echo "Failed to fetch the latest Alpine version."
    exit 1
fi

sed -i "s|^FROM alpine:.*|FROM alpine:$latest_alpine_version|" "$DOCKERFILE"
echo "Dockerfile updated to use alpine:$latest_alpine_version"
