.PHONY: frontend-dev frontend-check backend-dev backend-check db-up db-down

frontend-dev:
	cd frontend && npm run dev

frontend-check:
	cd frontend && npm run typecheck && npm run build

backend-dev:
	cd backend && go run ./cmd/api

backend-check:
	cd backend && go test ./... && go vet ./... && go build ./cmd/api

db-up:
	docker compose up -d postgres

db-down:
	docker compose down
