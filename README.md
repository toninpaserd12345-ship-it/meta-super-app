# Meta Super App

Frontend and backend are independent applications kept in separate directories.

```text
META-SUPPER-APP/
├── frontend/          Nuxt 4 + Vue + Vuetify web application
├── backend/           Go + Fiber + GORM REST API
├── docker-compose.yml Local PostgreSQL and backend services
└── Makefile           Common development commands
```

## Development

```bash
make db-up
make backend-dev
make frontend-dev
```

Run the frontend and backend commands in separate terminals. Configuration examples are available in `frontend/.env.example` and `backend/.env.example`.

The default development configuration uses in-memory backend data, so PostgreSQL is optional. Use `admin@example.com` / `password123` to test the integrated login flow.
