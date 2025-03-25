#!/usr/bin/env bash

PROJECT_ROOT="${1:-/var/lib/minesweeper}"

tr -cd \[:print:\] </dev/urandom | head -c 32 \
    | tee "$PROJECT_ROOT/secrets/postgres-password.txt" 
#   \ | docker secret create postgres_password -