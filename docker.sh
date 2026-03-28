#!/bin/sh
set -e
VAULT="${1:?Usage: docker.sh /path/to/vault}"
DATA="${KHAYAL_DATA:-$HOME/.config/khayal}"
shift
docker run \
  --add-host host.docker.internal:host-gateway \
  -v "$VAULT:/vault" \
  -v "$DATA:/root/.config/khayal" \
  -p 1133:1133 \
  ghcr.io/rawnaqs/khayal "$@"
