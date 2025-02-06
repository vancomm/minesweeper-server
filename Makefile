.phony: test keys live/server dev

test:
	go test -v ./...

keys:
	mkdir -p run/secrets && \
	echo -e '!.gitignore\n*' > run/secrets/.gitignore && \
	cd ./run/secrets && \
	../../scripts/create-jwt-keys.sh

live/server: ${JWT_KEYS}
	docker compose -f deployments/compose.live.yaml up --build --remove-orphans

dev:
	make live/server