#!/usr/bin/env bash
set -euo pipefail

export POSTGRES_HOST=db
export POSTGRES_PASSWORD_FILE=run/secrets/db_password
export POSTGRES_USER=postgres
export POSTGRES_DB=app
export POSTGRES_PORT=5432
export POSTGRES_SSLMODE=disable
export COOKIES_DOMAIN=""
export COOKIES_SECURE=""
export COOKIES_SAMESITE=none
export JWT_PRIVATE_KEY_FILE=run/secrets/jwt-private-key.pem
export JWT_PUBLIC_KEY_FILE=run/secrets/jwt-public-key.pem

bin/gateway