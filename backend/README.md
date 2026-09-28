# Go API

Fiber v3 + GORM backend using Clean Architecture for Meta Super App.

```text
cmd/api                 composition root and process lifecycle
internal/domain         entities, claims and repository contracts
internal/usecase        application business rules
internal/infrastructure database, GORM repositories, JWT and bcrypt
internal/delivery/httpx Fiber handlers, middleware and JSON responses
```

## Run locally

```bash
cp backend/.env.example backend/.env
docker compose up -d postgres
cd backend
go run ./cmd/api
```

`STORAGE_DRIVER=memory` runs the complete login/account/claims flow without a database. Change it to `postgres` later to activate the existing GORM repository and migrations. Both adapters implement the same domain interfaces.

Configure Nuxt with `NUXT_API_BASE_URL=http://127.0.0.1:8080` and paths `/auth/login` and `/auth/me`.

The development seed creates `admin@example.com` / `password123`. Change or remove these values outside local development.

## Endpoints

- `GET /health`
- `POST /auth/login`
- `GET /auth/me` — bearer token
- `GET /api/v1/accounts` — bearer token
- `GET /api/v1/dashboard` — bearer token, `X-Account-ID`, `dashboard:read`

Authorization is account-scoped. A user can belong to many accounts, with a different role and claims in each. Every account resource must use `RequireAccount` and must also filter database queries by the verified account ID.
