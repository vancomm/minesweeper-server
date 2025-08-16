# minesweeper-server

## installation and deployment

### local

```bash
export TARGET_DIR="./run/secrets"
./scripts/generate-jwt-keys.sh
./scripts/generate-password.sh
make dev
```

### remote

```bash
# on remote machine
export TARGET_DIR="/var/lib/minesweeper"
mkdir "$TARGET_DIR"
./scripts/generate-jwt-keys.sh
./scripts/generate-password.sh
```

```bash
# on development machine
export REMOTE_HOST="my-remote-host"
./scripts/create-context.sh
./scripts/remote-up.sh
```

## requirements

### dev

- [sqlc](https://github.com/sqlc-dev/sqlc)
- [air](https://github.com/air-verse/air)
- openssh
- openssl

## todo

### code quality
- [ ] refactor C-like parts with Go best practices
- [ ] add remaining tree234 functionality and tests from tree.c

### testing
- [x] add mines tests for different field params (table driven tests?)
- [ ] convert tree234's TestSuite to unit tests
    - [ ] describe existing tests and decompose
- [ ] add benchmarks