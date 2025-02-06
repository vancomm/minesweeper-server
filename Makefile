JWT_KEYS=run/secrets/jwt-private-key.pem run/secrets/jwt-public-key.pem

.phony: test live/server dev

test:
	go test -v ./...

run/secrets:
	mkdir -p run/secrets && echo -e '!.gitignore\n*' > run/secrets/.gitignore

${JWT_KEYS}: run/secrets
	cd ./run/secrets && ../../scripts/create-jwt-keys.sh

live/server: ${JWT_KEYS}
	docker compose -f deployments/compose.live.yaml up --build

dev:
	make live/server