#!/usr/bin/env bash
set -euxo pipefail

TARGET_DIR="${TARGET_DIR:-/var/lib/minesweeper/secrets}"
PRIVATE_KEY_FILE="${PRIVATE_KEY_FILE:-$TARGET_DIR/jwt-private-key.pem}"
PUBLIC_KEY_FILE="${PUBLIC_KEY_FILE:-$TARGET_DIR/jwt-public-key.pem}"

ssh-keygen -N "" -t rsa -m pem -f "$PRIVATE_KEY_FILE"
rm "$PRIVATE_KEY_FILE".pub
openssl rsa -in "$PRIVATE_KEY_FILE" -pubout -out "$PUBLIC_KEY_FILE"