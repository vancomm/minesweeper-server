#!/usr/bin/env bash

set -euo pipefail

export PROJECT_ROOT="${1:-/var/lib/minesweeper}"

PRIVATE_KEY_FILE="$PROJECT_ROOT/secrets/jwt-private-key.pem"
PUBLIC_KEY_FILE="$PROJECT_ROOT/secrets/jwt-public-key.pem"

ssh-keygen -t rsa -m pem -f "$PRIVATE_KEY_FILE"
rm "$PRIVATE_KEY_FILE.pub"
openssl rsa -in "$PRIVATE_KEY_FILE" -pubout -out "$PUBLIC_KEY_FILE"

docker secret inspect jwt-private-key >/dev/null \
    && docker secret rm jwt-private-key >/dev/null

docker secret inspect jwt-public-key >/dev/null \
    && docker secret rm jwt-public-key >/dev/null

docker secret create jwt-private-key "$PRIVATE_KEY_FILE" >/dev/null

docker secret create jwt-public-key "$PUBLIC_KEY_FILE" >/dev/null

docker secret inspect postgres-password >/dev/null \
    && docker secret rm postgres-password >/dev/null

tr -cd \[:alnum:\] </dev/urandom | head -c 32 \
    | tee "$PROJECT_ROOT/secrets/postgres-password.txt" \
    | docker secret create postgres-password - >/dev/null