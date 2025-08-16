#!/usr/bin/env bash
set -euxo pipefail

TARGET_DIR="${TARGET_DIR:-/var/lib/minesweeper/secrets}"

tr -cd \[:alnum:\] </dev/urandom | head -c 32 \
    >"$TARGET_DIR/postgres-password.txt"