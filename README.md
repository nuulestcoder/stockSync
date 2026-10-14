# Nzo Business — Stock Manager SaaS Backend

A multi-tenant SaaS platform for computer resellers in Senegal.

## Tech Stack

- **Language**: Go 1.22+
- **Web Framework**: Gin
- **Database**: MongoDB 7+
- **Background Jobs**: Asynq + Redis
- **File Storage**: S3-compatible (Cloudflare R2)
- **Config**: Viper + environment variables

## Quick Start

### Prerequisites

- Go 1.22+
- MongoDB 7+ (local or cloud)
- Redis (for background jobs)
- Make (optional, for easier commands)

### Setup

1. Clone the repository:
```bash
git clone <repo> nzo_project
cd nzo_project
```

2. Install dependencies:
```bash
go mod download
```

3. Configure environment:
```bash
cp .env.example .env
# Edit .env with your local/dev values
```

4. Run the server:
```bash
go run ./cmd/api
```

The server will start on `http://localhost:8080`

5. Test health check:
```bash
curl http://localhost:8080/health
```

## Development

### Build

```bash
go build -o bin/stock-manager ./cmd/api
```

### Run

```bash
go run ./cmd/api/main.go
```

### Test

```bash
# Run all tests
go test ./...

# Run tests with race detection
go test -race ./...

# Run tests for a specific package
go test ./internal/config/...

# Run a specific test
go test -run TestHealthCheck ./internal/handler/...
```

### Lint

```bash
# Install golangci-lint if needed
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run ./...
```

## Project Structure

See `CLAUDE.md` for a complete architecture overview. Quick reference:

```
.
├── cmd/api/              # Entry point
├── internal/
│   ├── config/           # Config loading (Viper)
│   ├── domain/           # Domain models (pure types)
│   ├── repository/       # MongoDB data access
│   ├── service/          # Business logic
│   ├── handler/          # Gin HTTP handlers
│   ├── middleware/       # Auth, tenant scoping, RBAC
│   └── worker/           # Asynq background jobs
└── pkg/                  # Shared utilities
    ├── mongoclient/
    ├── jwt/
    ├── password/
    └── response/
```

## Key Architecture Decisions

### Multi-Tenancy

- **Single database**, `company_id` field on every document (except `companies`)
- **Tenant isolation enforced at repository layer** — every query filters by `company_id`
- `company_id` injected via JWT claims → Gin context → middleware

### Clean Architecture

- **No business logic in handlers** — handlers only parse requests/responses
- **No database calls outside repositories** — repositories are the sole data access layer
- **Services orchestrate** — repositories + external dependencies
- **Domain types are pure** — no dependencies, framework-agnostic

### Error Handling

- Errors wrapped with context: `fmt.Errorf("operation: %w", err)`
- No `panic` in business logic
- Repository methods return `(T, error)` — never return `nil, nil`

## Environment Variables

Copy `.env.example` to `.env` and adjust:

```env
# App
APP_ENV=development           # development | staging | production
APP_PORT=8080

# Database
MONGO_URI=mongodb://localhost:27017
MONGO_DB=saas_stock

# Cache & Jobs
REDIS_URL=redis://localhost:6379
ASYNQ_CONCURRENCY=10

# Auth (will be implemented)
JWT_ACCESS_SECRET=change_me
JWT_REFRESH_SECRET=change_me

# File Storage (Cloudflare R2)
R2_ENDPOINT=https://...
R2_ACCESS_KEY=...

# Token Encryption
TOKEN_ENCRYPTION_KEY=...  # 32-byte hex key
```

## Contributing

See `CLAUDE.md` for conventions and business rules.

Before adding a dependency:
1. Verify it aligns with the tech stack
2. Prefer standard library / already-included packages
3. Discuss in PR if unsure

## Deployment

Not yet configured. See `CLAUDE.md` for architecture notes.

## License

TBD
