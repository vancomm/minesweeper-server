#!/bin/sh -xeu

CONTEXT_NAME="${CONTEXT_NAME:-remote}"
REMOTE_HOST="${REMOTE_HOST:-mskbox}"

docker context create "$CONTEXT_NAME" --docker "host=ssh://root@$REMOTE_HOST"