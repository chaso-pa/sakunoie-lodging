# Saku no Ie API (backend)

Go API (Gin + Huma v2) for the Saku no Ie guest-registration system. Prisma manages the MariaDB schema and migrations; the Go code accesses the database through GORM.

## Features

- **Gin Framework**: Fast and lightweight HTTP web framework
- **Prisma**: Schema and migrations only (no Prisma Client)
- **OpenAPI**: API documentation with Swagger
- **Clean Architecture**: Well-organized project structure
- **Middleware Support**: Built-in middleware architecture
- **Static File Serving**: Ready-to-use static file handlers

## Quick Start

### Prerequisites

- Go 1.21 or higher
- Node.js 18 or higher
- npm
- Docker with Compose v2 (throwaway test MariaDB)
- `mysql` client

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd sakunoie-lodging/backend
```

2. Install Go dependencies:
```bash
go mod tidy
```

3. Install Node.js dependencies (for Prisma):
```bash
npm ci
```

4. Set up the database. Copy `.env.example` to `.env` and fill in the shared
   dev DB URL (`lodging-master-dev`). Then apply migrations and the seed:
```bash
npm run db:deploy:dev   # CREATE DATABASE IF NOT EXISTS + migrate deploy + seed
npx prisma migrate status
```
   Only `migrate deploy`, `migrate status` and the seed may run against the
   shared server. See "Database Management" for everything else.

### Running the Application

Start the development server:
```bash
go run cmd/server/main.go
```

The server will start on `http://localhost:8080`

## Project Structure

```
├── api/                    # API specifications
├── cmd/server/            # Application entry point
├── internal/              # Private application code
│   ├── handlers/          # HTTP request handlers
│   ├── middlewares/       # HTTP middlewares
│   ├── models/           # Data models
│   ├── routes/           # Route definitions
│   ├── services/         # Business logic
│   └── utils/            # Utility functions
├── prisma/               # Database schema
├── go.mod               # Go module definition
└── package.json         # Node.js dependencies
```

## Development

### Database Management

Migrations are authored against the throwaway compose MariaDB only
(`compose.test.yaml`, `127.0.0.1:33306`, databases `lodging_migrate`,
`lodging_shadow`, `lodging_test`, `lodging_e2e`). `prisma.config.ts` refuses
`migrate dev|diff|reset` and `db push` unless both `DATABASE_URL` and
`SHADOW_DATABASE_URL` point there. URLs are in `.env.test.example`.

```bash
docker compose -f compose.test.yaml up -d --wait
export DATABASE_URL=mysql://root:test@127.0.0.1:33306/lodging_migrate
export SHADOW_DATABASE_URL=mysql://root:test@127.0.0.1:33306/lodging_shadow
npx prisma migrate dev --name <migration-name>   # needs a TTY
npx prisma migrate diff --from-migrations prisma/migrations --to-schema prisma/schema.prisma --exit-code
bash scripts/db-reset-test.sh lodging_test       # recreate + migrate + seed a test DB
docker compose -f compose.test.yaml down -v
```

CHECK constraints live in the hand-written `*_add_check_constraints`
migration (Prisma cannot express them). The facility seed is
`prisma/seed.sql` (`npm run db:seed`, idempotent).

### Testing

Run all tests:
```bash
go test ./...
```

Run tests with coverage:
```bash
go test -cover ./...
```

### API Documentation

The OpenAPI specification is available at `api/openapi.yaml`. You can view the documentation using any OpenAPI-compatible tool.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.