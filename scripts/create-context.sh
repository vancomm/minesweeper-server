#!/usr/bin/env bash
set -x

CONTEXT_NAME="${CONTEXT_NAME:-remote}"
REMOTE_USER=root
REMOTE_HOST="${REMOTE_HOST:-mskbox}"

docker context create "$CONTEXT_NAME" \
    --docker "host=ssh://$REMOTE_USER@$REMOTE_HOST"