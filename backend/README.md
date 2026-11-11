# Gin Template

A modern Go web application template built with the Gin framework, featuring Prisma for database management and OpenAPI documentation.

## Features

- **Gin Framework**: Fast and lightweight HTTP web framework
- **Prisma**: Type-safe database client and schema management
- **OpenAPI**: API documentation with Swagger
- **Clean Architecture**: Well-organized project structure
- **Middleware Support**: Built-in middleware architecture
- **Static File Serving**: Ready-to-use static file handlers

## Quick Start

### Prerequisites

- Go 1.21 or higher
- Node.js 18 or higher
- Yarn package manager

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd gin-template
```

2. Install Go dependencies:
```bash
go mod tidy
```

3. Install Node.js dependencies (for Prisma):
```bash
yarn install
```

4. Set up the database:
```bash
npx prisma generate
npx prisma migrate dev
```

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

Generate Prisma client:
```bash
npx prisma generate
```

Create and apply migrations:
```bash
npx prisma migrate dev --name <migration-name>
```

Open Prisma Studio:
```bash
npx prisma studio
```

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