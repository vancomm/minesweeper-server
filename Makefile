SECRETS_DIR=./run/secrets

.phony: test keys live/server dev

test:
	go test -v ./...

${SECRETS_DIR}:
	mkdir -p ${SECRETS_DIR} \
	&& echo -e '!.gitignore\n*' > ${SECRETS_DIR}/.gitignore

keys: ${SECRETS_DIR}
	TARGET_DIR=${SECRETS_DIR} ./scripts/generate-jwt-keys.sh

live/server:
	docker compose -f deployments/compose.live.yaml up --build --remove-orphans

dev:
	make live/server