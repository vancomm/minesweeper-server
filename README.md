# minesweeper-server

## how to run

### production

```sh
docker compose -f deployments/compose.yaml up --build
```

### development (live reload)

```sh
make keys # create JWT keys first time you run
make dev
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