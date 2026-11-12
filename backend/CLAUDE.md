# Claude Code Project Configuration

## Overview

Go web API template using **Gin** as the HTTP router and **Huma v2** as the API framework.
Huma wraps Gin via `humagin` adapter, providing automatic OpenAPI generation, input validation, and typed error responses.

```
backend/
├── api/openapi.yaml         # Committed OpenAPI 3.1 spec (drift-tested)
├── cmd/
│   ├── server/main.go       # Server entry point: config → DB → Gin + Huma
│   ├── openapi/             # Spec exporter
│   └── admin-grant/         # Owner grant CLI (guarded targets)
├── internal/
│   ├── config/              # Env loading and startup refusals
│   ├── database/            # GORM open; forces loc=UTC / time_zone +00:00
│   ├── entities/            # GORM table structs (explicit TableName)
│   ├── handlers/            # Huma handler functions (typed request/response)
│   ├── middlewares/         # Gin middlewares
│   ├── models/              # Input/Output structs for Huma endpoints
│   ├── routes/              # Route registration (SetupHumaRoutes, SetupStaticRoutes)
│   ├── services/            # Business logic (called by handlers)
│   ├── testutil/            # Compose-only DB helpers for integration tests
│   └── utils/               # Shared utilities
├── prisma/
│   ├── schema.prisma        # Database schema (Prisma 7, no client)
│   ├── migrations/
│   └── seed.sql
├── scripts/                 # assert-safe-db.sh, db-reset-test.sh, db-deploy-dev.sh
├── go.mod
└── package.json             # Node deps for Prisma tooling
```

## Architecture

### Gin + Huma integration

```go
r := gin.Default()
api := humagin.New(r, huma.DefaultConfig("My API", "1.0.0"))
```

- Gin handles raw HTTP (middleware, static routes, non-API routes)
- Huma registers typed endpoints on top of Gin via `huma.Get/Post/Put/Delete/Patch`
- All API endpoints MUST go through Huma — do NOT add raw `r.GET/POST` for API routes

### Route setup pattern

Register routes in `internal/routes/`:

```go
// internal/routes/greeting_routes.go
func SetupGreetingRoutes(api huma.API) {
    huma.Get(api, "/greeting/{name}", handlers.GetGreeting)
    huma.Post(api, "/greetings", handlers.CreateGreeting)
}
```

Call from `main.go`:

```go
routes.SetupGreetingRoutes(api)
```

## Handler Conventions

### Signature

All Huma handlers MUST follow this exact signature:

```go
func HandlerName(ctx context.Context, input *InputType) (*OutputType, error)
```

### Input struct

Define in `internal/models/`. Use struct tags for validation and OpenAPI docs:

```go
type CreateUserInput struct {
    // Path params
    OrgID string `path:"orgId" doc:"Organization ID"`

    // Query params
    Verbose bool `query:"verbose" default:"false" doc:"Include extra details"`

    // Request body
    Body struct {
        Name  string `json:"name"  minLength:"1" maxLength:"100" doc:"User's full name"`
        Email string `json:"email" format:"email"                doc:"User's email address"`
        Age   int    `json:"age"   minimum:"0"   maximum:"150"   doc:"User's age"`
    }
}
```

Supported tags: `path`, `query`, `header`, `cookie`, `required`, `default`, `minLength`, `maxLength`, `minimum`, `maximum`, `format`, `pattern`, `enum`, `example`, `doc`

### Output struct

```go
type CreateUserOutput struct {
    Body struct {
        ID    string `json:"id"    doc:"Created user ID"`
        Name  string `json:"name"  doc:"User's full name"`
        Email string `json:"email" doc:"User's email address"`
    }
}
```

Always nest the response payload inside `Body`. Top-level fields are for response headers.

### Error handling

Use Huma's typed errors — never return raw `errors.New`:

```go
import "github.com/danielgtaylor/huma/v2"

// 404
return nil, huma.Error404NotFound("user not found")

// 400
return nil, huma.Error400BadRequest("invalid input: " + reason)

// 409
return nil, huma.Error409Conflict("user already exists")

// 422 with field-level detail
return nil, huma.NewError(http.StatusUnprocessableEntity, "validation failed",
    &huma.ErrorDetail{Message: "must be positive", Location: "body.age", Value: -1},
)
```

## File Organization

| Concern | Location |
|---|---|
| Handler functions | `internal/handlers/<resource>_handlers.go` |
| Input/Output models | `internal/models/<resource>.go` |
| Route registration | `internal/routes/<resource>_routes.go` |
| Business logic | `internal/services/<resource>_service.go` |
| DB access | `internal/services/<resource>_service.go` (or separate `repository`) |
| Shared utilities | `internal/utils/` |
| GORM table structs | `internal/entities/<table>.go` |
| Env config and startup checks | `internal/config/` |
| DB connection (UTC enforcement) | `internal/database/` |

Group files by resource (e.g., `user_handlers.go`, `user_routes.go`, `user.go`).

## Naming Conventions

- Files: `snake_case` (e.g., `user_handlers.go`)
- Exported types/functions: `PascalCase`
- Unexported: `camelCase`
- Input structs: `<Action><Resource>Input` (e.g., `CreateUserInput`, `GetUserInput`)
- Output structs: `<Action><Resource>Output` (e.g., `CreateUserOutput`)
- Handler functions: `<Action><Resource>` (e.g., `CreateUser`, `GetUser`)
- Route setup functions: `Setup<Resource>Routes(api huma.API)`

## Development Guidelines

- Business logic belongs in `internal/services/`, not in handlers
- Handlers only: parse input, call service, map to output
- Every endpoint needs `doc:` tags on input/output fields — this drives the OpenAPI spec
- Do not add raw Gin routes for API endpoints; use `huma.Get/Post/...`
- Gin raw routes (`r.GET/POST`) are only for non-API concerns (health check, static files, metrics)

## Database Migration

**マイグレーション管理は必ずPrismaを使用すること。** Go からの DB アクセスは GORM（CRUD のみ、AutoMigrate 禁止）。Prisma Client は使わないため `generator` ブロックはなく、`prisma generate` は不要。

### 安全ルール（共有 DB サーバー）

- `.env` の `DATABASE_URL` は共有サーバー上の `lodging-master-dev`。ここで許可されるのは `migrate deploy` / `migrate status` / seed（`npm run db:deploy:dev`）のみ。
- `migrate dev` / `migrate diff` / `migrate reset` / `db push` は **compose の MariaDB（`compose.test.yaml`, `127.0.0.1:33306`）に対してのみ** 実行する。`prisma.config.ts` が argv を見て、URL が compose でなければ拒否する。
- DB を触るスクリプトは必ず `scripts/assert-safe-db.sh` を source する。パスワードを argv に載せない（`--defaults-extra-file`）。

### ワークフロー

1. `docker compose -f compose.test.yaml up -d --wait`
2. `prisma/schema.prisma` を編集する（モデルは PascalCase、`@@map`/`@map` で snake_case）
3. compose の URL をインラインで指定して `migrate dev` を実行する（非対話シェルでは pty が必要）:
   ```bash
   DATABASE_URL=mysql://root:test@127.0.0.1:33306/lodging_migrate \
   SHADOW_DATABASE_URL=mysql://root:test@127.0.0.1:33306/lodging_shadow \
   bash -c '(sleep 8; printf "y\n"; sleep 15) | script -qec "npx prisma migrate dev --name <name>" /dev/null'
   ```
4. 同じ URL で `npx prisma migrate diff --from-migrations prisma/migrations --to-schema prisma/schema.prisma --exit-code` が 0 になることを確認する
5. 共有 dev DB へは `npm run db:deploy:dev`

### マイグレーションルール

- スキーマ変更は **必ず** `prisma/schema.prisma` を通じて行う — DBを直接変更しない
- マイグレーションファイル (`prisma/migrations/`) は自動生成されるため手動編集しない。**例外**: Prisma で表現できない CHECK 制約は `migrate dev --create-only` で作った空マイグレーションに手で書く（`*_add_check_constraints`、ルート `CLAUDE.md` §13）
- JSON カラムは `String @db.LongText` ＋ CHECK マイグレーションの `json_valid()`（MariaDB の `JSON` の実体。Prisma の `Json` 型は MariaDB で偽のドリフトを出す）
- 本番環境では `npx prisma migrate deploy` を使用する（`migrate dev` は開発専用）

## Commands

### Build and Run
```bash
go run cmd/server/main.go
```

### Database Operations
```bash
docker compose -f compose.test.yaml up -d --wait  # テスト用 MariaDB（127.0.0.1:33306, tmpfs）
bash scripts/db-reset-test.sh lodging_test        # compose DB を作り直して migrate deploy + seed
npm run db:deploy:dev                             # 共有 lodging-master-dev に migrate deploy + seed
npx prisma migrate status                         # 適用状況確認（.env の URL）
docker compose -f compose.test.yaml down -v       # 後片付け
```

### Testing
```bash
go test ./...           # Run all tests
```
