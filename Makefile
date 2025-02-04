test:
	go test -v ./...

live/server:
	docker compose -f deployments/compose.live.yaml up --build

dev:
	make live/server