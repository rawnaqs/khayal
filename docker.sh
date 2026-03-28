#!/bin/sh
set -e
VAULT="${1:?Usage: khayal-docker /path/to/vault}"
DATA="${KHAYAL_DATA:-$HOME/.config/khayal}"
shift
docker run \
  -v "$VAULT:/vault" \
  -v "$DATA:/root/.config/khayal" \
  -p 1133:1133 \
  ghcr.io/rawnaqs/khayal "$@"
