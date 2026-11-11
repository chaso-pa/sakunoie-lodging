# Claude Code Project Configuration

## Overview

Go web API template using **Gin** as the HTTP router and **Huma v2** as the API framework.
Huma wraps Gin via `humagin` adapter, providing automatic OpenAPI generation, input validation, and typed error responses.

```
gin-template/
├── cmd/
│   └── server/
│       └── main.go          # Server entry point: Gin setup + Huma API init
├── internal/
│   ├── handlers/            # Huma handler functions (typed request/response)
│   ├── middlewares/         # Gin middlewares
│   ├── models/              # Input/Output structs for Huma endpoints
│   ├── routes/              # Route registration (SetupHumaRoutes, SetupStaticRoutes)
│   ├── services/            # Business logic (called by handlers)
│   └── utils/               # Shared utilities
├── prisma/
│   └── schema.prisma        # Database schema
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

**マイグレーション管理は必ずPrismaを使用すること。** 生のSQLファイルや他のマイグレーションツールは使用しない。

### ワークフロー

1. `prisma/schema.prisma` を編集してスキーマを変更する
2. `npx prisma migrate dev --name <migration_name>` でマイグレーションを生成・適用する
3. `npx prisma generate` でGoクライアントを再生成する

### マイグレーションルール

- スキーマ変更は **必ず** `prisma/schema.prisma` を通じて行う — DBを直接変更しない
- マイグレーションファイル (`prisma/migrations/`) は自動生成されるため手動編集しない
- 本番環境では `npx prisma migrate deploy` を使用する（`migrate dev` は開発専用）

## Commands

### Build and Run
```bash
go run cmd/server/main.go
```

### Database Operations
```bash
npx prisma migrate dev --name <name>  # マイグレーション生成・適用（開発用）
npx prisma migrate deploy             # マイグレーション適用（本番用）
npx prisma generate                   # Goクライアント再生成
npx prisma studio                     # Prisma Studio（DB GUI）
npx prisma migrate status             # マイグレーション適用状況確認
```

### Testing
```bash
go test ./...           # Run all tests
```
