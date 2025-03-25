#!/usr/bin/env bash
set -x

PROJECT_ROOT=/var/lib/minesweeper

export POSTGRES_PASSWORD_FILE="$PROJECT_ROOT/secrets/postgres-password.txt"
export JWT_PRIVATE_KEY_FILE="$PROJECT_ROOT/secrets/jwt-private-key.pem"
export JWT_PUBLIC_KEY_FILE="$PROJECT_ROOT/secrets/jwt-public-key.pem"
docker --context remote compose --file deployments/compose.yaml \
    up --detach --remove-orphans --build