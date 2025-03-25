#!/usr/bin/env bash

PROJECT_ROOT="${1:-/var/lib/minesweeper}"

tr -cd \[:alnum:\] </dev/urandom | head -c 32 \
    >"$PROJECT_ROOT/secrets/postgres-password.txt"