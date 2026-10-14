# Copilot Instructions — Nzo Stock Manager SaaS

**See CLAUDE.md for comprehensive architecture, data models, and business rules.**

This file covers build/test commands and key development conventions specific to this codebase.

---

## Quick Commands

| Command | Purpose |
|---------|---------|
| `go run ./cmd/api` | Start dev server (http://localhost:8080) |
| `go build -o bin/stock-manager ./cmd/api` | Build binary |
| `go test ./...` | Run all tests |
| `go test -race ./...` | Run tests with race detection |
| `go test -run TestName ./...` | Run specific test |
| `golangci-lint run ./...` | Lint code |

---

## Architecture Overview

### Layers (Clean Architecture)

1. **Handlers** (`internal/handler`) — Parse HTTP requests, inject `companyID` from context, call services
   - Never write business logic here
   - Never touch DB directly
   
2. **Services** (`internal/service`) — Orchestrate business logic, call repositories + external APIs
   - All business rules live here
   - Import only domain + repository + external packages
   
3. **Repositories** (`internal/repository`) — MongoDB data access, tenant filtering
   - **Every query MUST include `company_id` filter** — critical for multi-tenancy
   - Never call other repositories directly; return domain types for services to use
   
4. **Domain** (`internal/domain`) — Pure Go types, no dependencies
   - Define all data structures here
   - No framework imports

5. **Middleware** (`internal/handler/middleware`) — Auth token validation, tenant injection, RBAC
   - `auth.go` — validates JWT and extracts claims
   - `tenant.go` — injects `companyID` into context from claims
   - `rbac.go` — enforces role-based access control

6. **Workers** (`internal/worker`) — Asynq background jobs
   - Scheduled tasks, async processing
   - Receive enqueued job payloads, call services

### Multi-Tenancy Rule (CRITICAL)

**Every repository method receives `companyID bson.ObjectID` as a parameter.**
**Every MongoDB query includes `company_id: companyID` in the filter — no exceptions.**

Example (repository):
```go
// ❌ WRONG — exposes data across tenants
func (r *ProductRepository) GetAll(ctx context.Context) ([]Product, error) {
    return r.collection.Find(ctx, bson.M{})
}

// ✅ CORRECT — tenant-scoped
func (r *ProductRepository) GetAll(ctx context.Context, companyID bson.ObjectID) ([]Product, error) {
    return r.collection.Find(ctx, bson.M{"company_id": companyID})
}
```

Example (service calling repository):
```go
func (s *ProductService) List(ctx context.Context, companyID bson.ObjectID) ([]Product, error) {
    return s.repo.GetAll(ctx, companyID)
}
```

Example (handler getting companyID from context):
```go
func GetProducts(c *gin.Context) {
    companyID := c.MustGet("companyID").(bson.ObjectID)  // injected by middleware
    products, err := s.productService.List(c.Request.Context(), companyID)
    // ...
}
```

---

## Key Conventions

### Error Handling

- Always wrap errors with context: `fmt.Errorf("operation: %w", err)`
- Example: `fmt.Errorf("product.Create: failed to insert: %w", err)`
- Never use bare `return err`

### Method Signatures

**Repository methods:**
```go
func (r *ProductRepository) Create(ctx context.Context, companyID bson.ObjectID, product *Product) (*Product, error)
func (r *ProductRepository) GetByID(ctx context.Context, companyID bson.ObjectID, id bson.ObjectID) (*Product, error)
```

**Service methods:**
```go
func (s *ProductService) Create(ctx context.Context, companyID bson.ObjectID, req *CreateProductRequest) (*Product, error)
```

**Handler methods:**
```go
func CreateProduct(c *gin.Context) {
    companyID := c.MustGet("companyID").(bson.ObjectID)
    // ...
}
```

### Response Envelope

All API responses use this structure (via `pkg/response`):
```json
{
  "success": true,
  "data": {},
  "message": "ok",
  "error": null
}
```

Use helper functions:
```go
response.Success(c, http.StatusOK, data, "success message")
response.Error(c, http.StatusBadRequest, "error message")
```

### Testing

- **Unit tests** — test services and repositories in isolation with mocked dependencies
- **File naming** — `sale_service_test.go` (beside `sale_service.go`)
- **Use testify** — `testify/assert`, `testify/mock`

Example:
```bash
go test -run TestSaleService ./internal/service/...
```

---

## Important Notes

### Before Adding a Dependency

Ask first. Only these are pre-approved:
- `gin-gonic/gin` — HTTP framework
- `go.mongodb.org/mongo-driver` — MongoDB driver
- `hibiken/asynq` — Background jobs
- `go-redis/redis` — Redis client
- `spf13/viper` — Config management

For anything else, discuss in the PR.

### Multi-Agent Coordination

When touching `internal/repository`:
1. Re-read the "Multi-Tenancy Rule" section above
2. Verify every MongoDB query filters by `company_id`
3. Verify every repository method signature includes `companyID` parameter

---

## Project Setup (Reference)

See README.md for detailed setup instructions.

```bash
cp .env.example .env           # Configure environment
go mod download                 # Install dependencies
go run ./cmd/api               # Start server
curl http://localhost:8080/health  # Test
```

---

## Feature Roadmap (v1 scope)

- [x] Project scaffold (config, handlers, services, repositories)
- [ ] Company tenant management (`internal/domain/company.go`, services, handlers)
- [ ] User auth (register, login, JWT refresh)
- [ ] Product catalog (CRUD with variants)
- [ ] Stock tracking (movement history, alerts)
- [ ] Sales module (invoicing, payment tracking)
- [ ] Customers & Suppliers
- [ ] Background jobs (stock alerts, invoice generation)
- [ ] Social media posting (Facebook/Instagram)

---

## Deployment

Not yet configured. See `CLAUDE.md` for architecture decisions.

When ready:
1. Define env-specific configs (dev/staging/prod)
2. Set up MongoDB Atlas or similar
3. Choose deployment target (Fly.io, Railway, Cloud Run, etc.)
4. Configure Redis (e.g., Upstash)
