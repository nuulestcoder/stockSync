# CLAUDE.md — Stock Manager SaaS / Nzo Business

## Project Overview

Multi-tenant SaaS application for computer reseller businesses (laptops, desktops, accessories).
Each company gets its own isolated workspace. Handles product catalog with variants, real-time stock tracking, sales, invoicing, customers, and suppliers.

**Target market:** Senegalese market (XOF currency, Wave/Orange Money payments)
**Multi-tenancy:** Single DB, `companyId` field on every document — filtered automatically via middleware
**Primary language:** French (UI, comments, error messages)
**Code language:** English (variables, functions, types, packages)

---

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.22+ |
| Framework | Gin |
| Database | MongoDB 7+ (mongo-driver/v2) |
| Auth | JWT (access + refresh tokens) |
| Background jobs | Asynq (Redis) |
| File storage | S3-compatible (Cloudflare R2) |
| Config | Viper + `.env` |
| Testing | `testing` + `testify` |
| Email | Resend (resend/resend-go/v4) |

---

## Project Structure

```
.
├── cmd/
│   └── api/
│       └── main.go
├── internal/
│   ├── config/          # Viper config loading
│   ├── domain/          # Pure types, no dependencies
│   │   ├── company.go
│   │   ├── user.go
│   │   ├── product.go
│   │   ├── stock.go
│   │   ├── sale.go
│   │   ├── customer.go
│   │   ├── supplier.go
│   │   ├── social_account.go
│   │   └── post.go
│   ├── repository/      # MongoDB data access
│   │   ├── interfaces.go
│   │   ├── company_repository.go
│   │   ├── user_repository.go
│   │   ├── product_repository.go
│   │   ├── stock_repository.go
│   │   ├── sale_repository.go
│   │   ├── customer_repository.go
│   │   ├── supplier_repository.go
│   │   ├── social_account_repository.go
│   │   └── post_repository.go
│   ├── service/         # Business logic
│   │   ├── company_service.go
│   │   ├── auth_service.go
│   │   ├── product_service.go
│   │   ├── stock_service.go
│   │   ├── sale_service.go
│   │   ├── customer_service.go
│   │   ├── supplier_service.go
│   │   ├── social_service.go   # OAuth flow + token management
│   │   └── post_service.go     # AI generation + publishing
│   ├── handler/         # Gin HTTP handlers
│   │   ├── company_handler.go
│   │   ├── auth_handler.go
│   │   ├── product_handler.go
│   │   ├── stock_handler.go
│   │   ├── sale_handler.go
│   │   ├── customer_handler.go
│   │   ├── supplier_handler.go
│   │   ├── social_handler.go   # OAuth connect + callback
│   │   ├── post_handler.go
│   │   └── middleware/
│   │       ├── auth.go        # JWT validation + inject claims
│   │       ├── tenant.go      # inject companyId from claims into context
│   │       └── rbac.go        # role-based access control
│   └── worker/          # Asynq background jobs
│       ├── stock_alert_worker.go
│       ├── invoice_worker.go
│       ├── publish_worker.go       # exécute la publication vers les APIs sociales
│       └── token_refresh_worker.go # refresh des tokens avant expiry
├── pkg/
│   ├── mongoclient/     # MongoDB connection setup
│   ├── jwt/             # JWT helpers
│   ├── password/        # Argon2id hashing
│   ├── resend/          # Email client handling mail notification
│   ├── invoice/         # Invoice number generator
│   ├── cipher/          # AES-256-GCM encrypt/decrypt pour les access tokens
│   └── response/        # Unified JSON response helpers
├── CLAUDE.md
├── go.mod
└── .env.example
```

---

## Collections Overview

| Collection | Description |
|------------|-------------|
| `companies` | Tenant — one doc per registered business |
| `users` | App users, scoped to a company |
| `products` | Product catalog with embedded variants |
| `stock` | Real-time stock level, one doc per variant |
| `stock_movements` | Immutable event log of every stock change |
| `sales` | Sales records with embedded items and payment info |
| `customers` | Customer directory per company |
| `suppliers` | Supplier directory per company |
| `social_accounts` | Connected Facebook/Instagram accounts with encrypted tokens |
| `posts` | Social media posts — draft, scheduled, or published |

> **Every collection except `companies` has a `company_id` field. Every query MUST filter by `company_id`.**

---

## Data Models

### Company (collection: `companies`)

```go
type Company struct {
    ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
    Name      string             `bson:"name"          json:"name"`
    Slug      string             `bson:"slug"          json:"slug"`   // unique, URL-friendly
    Logo      string             `bson:"logo"          json:"logo"`   // R2 URL
    Address   CompanyAddress     `bson:"address"       json:"address"`
    Contact   CompanyContact     `bson:"contact"       json:"contact"`
    Billing   CompanyBilling     `bson:"billing"       json:"billing"`
    Settings  CompanySettings    `bson:"settings"      json:"settings"`
    Plan      string             `bson:"plan"          json:"plan"`      // "starter" | "pro" | "enterprise"
    IsActive  bool               `bson:"is_active"     json:"isActive"`
    CreatedAt time.Time          `bson:"created_at"    json:"createdAt"`
    UpdatedAt time.Time          `bson:"updated_at"    json:"updatedAt"`
}

type CompanyAddress struct {
    Street  string `bson:"street"  json:"street"`
    City    string `bson:"city"    json:"city"`
    Country string `bson:"country" json:"country"` // ISO 3166-1 alpha-2, e.g. "SN"
}

type CompanyContact struct {
    Email    string `bson:"email"    json:"email"`
    Phone    string `bson:"phone"    json:"phone"`
    WhatsApp string `bson:"whatsapp" json:"whatsapp"`
}

type CompanyBilling struct {
    NINEA string `bson:"ninea" json:"ninea"` // Numéro d'Identification National des Entreprises et Associations
    RCCM  string `bson:"rccm"  json:"rccm"`  // Registre du Commerce et du Crédit Mobilier
}

type CompanySettings struct {
    Currency       string   `bson:"currency"        json:"currency"`       // "XOF"
    InvoicePrefix  string   `bson:"invoice_prefix"  json:"invoicePrefix"`  // e.g. "NZO"
    InvoiceCounter int      `bson:"invoice_counter" json:"invoiceCounter"` // auto-incremented
    TaxRate        float64  `bson:"tax_rate"        json:"taxRate"`        // 0 if not applicable
    PaymentMethods []string `bson:"payment_methods" json:"paymentMethods"` // ["wave", "orange_money", "cash"]
}
```

---

### User (collection: `users`)

```go
type User struct {
    ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    CompanyID primitive.ObjectID `bson:"company_id"    json:"companyId"`
    Name      string             `bson:"name"          json:"name"`
    Email     string             `bson:"email"         json:"email"`
    Phone     string             `bson:"phone"         json:"phone"`
    Password  string             `bson:"password"      json:"-"`       // Argon2id hash
    Role      string             `bson:"role"          json:"role"`    // "admin" | "manager" | "seller"
    IsActive  bool               `bson:"is_active"     json:"isActive"`
    CreatedAt time.Time          `bson:"created_at"    json:"createdAt"`
    UpdatedAt time.Time          `bson:"updated_at"    json:"updatedAt"`
}
```

---

### Product (collection: `products`)

```go
type Product struct {
    ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    CompanyID   primitive.ObjectID `bson:"company_id"    json:"companyId"`
    Name        string             `bson:"name"          json:"name"`
    Brand       string             `bson:"brand"         json:"brand"`
    Category    string             `bson:"category"      json:"category"`
    Description string             `bson:"description"   json:"description"`
    Images      []string           `bson:"images"        json:"images"`    // generic product photos
    Tags        []string           `bson:"tags"          json:"tags"`
    Variants    []Variant          `bson:"variants"      json:"variants"`
    IsDeleted   bool               `bson:"is_deleted"    json:"-"`
    CreatedAt   time.Time          `bson:"created_at"    json:"createdAt"`
    UpdatedAt   time.Time          `bson:"updated_at"    json:"updatedAt"`
}

type Variant struct {
    ID     primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    SKU    string             `bson:"sku"           json:"sku"`
    Specs  VariantSpecs       `bson:"specs"         json:"specs"`
    Price  Price              `bson:"price"         json:"price"`
    Images []string           `bson:"images"        json:"images"` // optional, fallback to product images
}

type VariantSpecs struct {
    CPU       string `bson:"cpu,omitempty"       json:"cpu,omitempty"`
    RAM       string `bson:"ram,omitempty"       json:"ram,omitempty"`
    Storage   string `bson:"storage,omitempty"   json:"storage,omitempty"`
    Color     string `bson:"color,omitempty"     json:"color,omitempty"`
    Condition string `bson:"condition,omitempty" json:"condition,omitempty"` // "new" | "refurbished"
}

type Price struct {
    Buying   float64 `bson:"buying"   json:"buying"`
    Selling  float64 `bson:"selling"  json:"selling"`
    Currency string  `bson:"currency" json:"currency"` // always "XOF"
}
```

---

### Stock (collection: `stock`)

```go
type Stock struct {
    ID            primitive.ObjectID `bson:"_id,omitempty"  json:"id"`
    CompanyID     primitive.ObjectID `bson:"company_id"     json:"companyId"`
    ProductID     primitive.ObjectID `bson:"product_id"     json:"productId"`
    VariantID     primitive.ObjectID `bson:"variant_id"     json:"variantId"`
    Quantity      int                `bson:"quantity"       json:"quantity"`
    Reserved      int                `bson:"reserved"       json:"reserved"`
    Available     int                `bson:"available"      json:"available"` // quantity - reserved
    Threshold     int                `bson:"threshold"      json:"threshold"` // reorder alert level
    SerialNumbers []SerialNumber     `bson:"serial_numbers" json:"serialNumbers"`
    Location      string             `bson:"location"       json:"location"`
    LastMovement  time.Time          `bson:"last_movement"  json:"lastMovement"`
    UpdatedAt     time.Time          `bson:"updated_at"     json:"updatedAt"`
}

type SerialNumber struct {
    SN     string `bson:"sn"     json:"sn"`
    Status string `bson:"status" json:"status"` // "available" | "reserved" | "sold"
}
```

---

### StockMovement (collection: `stock_movements`)

```go
type StockMovement struct {
    ID             primitive.ObjectID `bson:"_id,omitempty"   json:"id"`
    CompanyID      primitive.ObjectID `bson:"company_id"      json:"companyId"`
    ProductID      primitive.ObjectID `bson:"product_id"      json:"productId"`
    VariantID      primitive.ObjectID `bson:"variant_id"      json:"variantId"`
    Type           string             `bson:"type"            json:"type"` // "IN" | "OUT" | "RETURN" | "ADJUSTMENT"
    Quantity       int                `bson:"quantity"        json:"quantity"`
    QuantityBefore int                `bson:"quantity_before" json:"quantityBefore"`
    QuantityAfter  int                `bson:"quantity_after"  json:"quantityAfter"`
    Reason         string             `bson:"reason"          json:"reason"`
    Reference      string             `bson:"reference"       json:"reference"` // sale ID or purchase order ID
    PerformedBy    primitive.ObjectID `bson:"performed_by"    json:"performedBy"`
    CreatedAt      time.Time          `bson:"created_at"      json:"createdAt"`
}
```

---

### Sale (collection: `sales`)

```go
type Sale struct {
    ID            primitive.ObjectID `bson:"_id,omitempty"   json:"id"`
    CompanyID     primitive.ObjectID `bson:"company_id"      json:"companyId"`
    InvoiceNumber string             `bson:"invoice_number"  json:"invoiceNumber"` // e.g. "NZO-2024-043"
    Status        string             `bson:"status"          json:"status"`         // "draft" | "confirmed" | "paid" | "cancelled" | "refunded"
    Customer      SaleCustomer       `bson:"customer"        json:"customer"`       // snapshot at time of sale
    Items         []SaleItem         `bson:"items"           json:"items"`
    Subtotal      float64            `bson:"subtotal"        json:"subtotal"`
    Discount      float64            `bson:"discount"        json:"discount"`
    TaxAmount     float64            `bson:"tax_amount"      json:"taxAmount"`
    Total         float64            `bson:"total"           json:"total"`
    Currency      string             `bson:"currency"        json:"currency"` // "XOF"
    Payment       SalePayment        `bson:"payment"         json:"payment"`
    Channel       string             `bson:"channel"         json:"channel"`   // "in_store" | "whatsapp" | "facebook"
    Notes         string             `bson:"notes"           json:"notes"`
    CreatedBy     primitive.ObjectID `bson:"created_by"      json:"createdBy"`
    CreatedAt     time.Time          `bson:"created_at"      json:"createdAt"`
    UpdatedAt     time.Time          `bson:"updated_at"      json:"updatedAt"`
}

type SaleCustomer struct {
    ID    *primitive.ObjectID `bson:"id,omitempty" json:"id,omitempty"` // nil for anonymous sale
    Name  string              `bson:"name"         json:"name"`
    Phone string              `bson:"phone"        json:"phone"`
    Email string              `bson:"email"        json:"email"`
}

type SaleItem struct {
    ProductID   primitive.ObjectID `bson:"product_id"   json:"productId"`
    VariantID   primitive.ObjectID `bson:"variant_id"   json:"variantId"`
    ProductName string             `bson:"product_name" json:"productName"` // snapshot
    SKU         string             `bson:"sku"          json:"sku"`
    Specs       VariantSpecs       `bson:"specs"        json:"specs"`       // snapshot
    SerialNumber string            `bson:"serial_number" json:"serialNumber"` // optional
    Quantity    int                `bson:"quantity"     json:"quantity"`
    UnitPrice   float64            `bson:"unit_price"   json:"unitPrice"`
    Discount    float64            `bson:"discount"     json:"discount"`
    Total       float64            `bson:"total"        json:"total"`
}

type SalePayment struct {
    Method    string    `bson:"method"    json:"method"`    // "cash" | "wave" | "orange_money" | "bank_transfer"
    Status    string    `bson:"status"    json:"status"`    // "pending" | "completed" | "failed"
    PaidAt    time.Time `bson:"paid_at"   json:"paidAt"`
    Reference string    `bson:"reference" json:"reference"` // mobile money transaction ref
}
```

---

### Customer (collection: `customers`)

```go
type Customer struct {
    ID             primitive.ObjectID `bson:"_id,omitempty"    json:"id"`
    CompanyID      primitive.ObjectID `bson:"company_id"       json:"companyId"`
    Name           string             `bson:"name"             json:"name"`
    Phone          string             `bson:"phone"            json:"phone"`
    Email          string             `bson:"email"            json:"email"`
    Address        string             `bson:"address"          json:"address"`
    TotalPurchases int                `bson:"total_purchases"  json:"totalPurchases"` // denormalized
    TotalSpent     float64            `bson:"total_spent"      json:"totalSpent"`     // denormalized
    CreatedAt      time.Time          `bson:"created_at"       json:"createdAt"`
    UpdatedAt      time.Time          `bson:"updated_at"       json:"updatedAt"`
}
```

---

### Supplier (collection: `suppliers`)

```go
type Supplier struct {
    ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    CompanyID   primitive.ObjectID `bson:"company_id"    json:"companyId"`
    Name        string             `bson:"name"          json:"name"`
    Country     string             `bson:"country"       json:"country"` // "CN", "AE", "SN"...
    Contact     SupplierContact    `bson:"contact"       json:"contact"`
    LeadTimeDays int               `bson:"lead_time_days" json:"leadTimeDays"` // avg delivery time
    Notes       string             `bson:"notes"         json:"notes"`
    CreatedAt   time.Time          `bson:"created_at"    json:"createdAt"`
    UpdatedAt   time.Time          `bson:"updated_at"    json:"updatedAt"`
}

type SupplierContact struct {
    Name    string `bson:"name"    json:"name"`
    Phone   string `bson:"phone"   json:"phone"`
    Email   string `bson:"email"   json:"email"`
    WhatsApp string `bson:"whatsapp" json:"whatsapp"`
}
```

---

### SocialAccount (collection: `social_accounts`)

```go
type SocialAccount struct {
    ID          primitive.ObjectID `bson:"_id,omitempty"  json:"id"`
    CompanyID   primitive.ObjectID `bson:"company_id"     json:"companyId"`
    Platform    string             `bson:"platform"       json:"platform"`     // "facebook" | "instagram"
    AccountID   string             `bson:"account_id"     json:"accountId"`    // Facebook Page ID or IG Business Account ID
    AccountName string             `bson:"account_name"   json:"accountName"`  // display name
    AccessToken string             `bson:"access_token"   json:"-"`            // AES-256-GCM encrypted, NEVER returned in JSON
    TokenExpiry time.Time          `bson:"token_expiry"   json:"tokenExpiry"`
    IsActive    bool               `bson:"is_active"      json:"isActive"`
    CreatedAt   time.Time          `bson:"created_at"     json:"createdAt"`
    UpdatedAt   time.Time          `bson:"updated_at"     json:"updatedAt"`
}
```

---

### Post (collection: `posts`)

```go
type Post struct {
    ID          primitive.ObjectID  `bson:"_id,omitempty"  json:"id"`
    CompanyID   primitive.ObjectID  `bson:"company_id"     json:"companyId"`
    ProductID   *primitive.ObjectID `bson:"product_id"     json:"productId"`   // nil if manual post
    VariantID   *primitive.ObjectID `bson:"variant_id"     json:"variantId"`
    Content     string              `bson:"content"        json:"content"`      // post text
    Images      []string            `bson:"images"         json:"images"`       // R2 URLs
    Targets     []PostTarget        `bson:"targets"        json:"targets"`      // one entry per platform
    ScheduledAt *time.Time          `bson:"scheduled_at"   json:"scheduledAt"` // nil = publish immediately
    Status      string              `bson:"status"         json:"status"`       // "draft" | "scheduled" | "publishing" | "published" | "failed"
    GeneratedBy string              `bson:"generated_by"   json:"generatedBy"` // "ai" | "manual"
    CreatedBy   primitive.ObjectID  `bson:"created_by"     json:"createdBy"`
    CreatedAt   time.Time           `bson:"created_at"     json:"createdAt"`
    PublishedAt *time.Time          `bson:"published_at"   json:"publishedAt"`
}

type PostTarget struct {
    SocialAccountID primitive.ObjectID `bson:"social_account_id" json:"socialAccountId"`
    Platform        string             `bson:"platform"          json:"platform"`           // "facebook" | "instagram"
    ExternalPostID  string             `bson:"external_post_id"  json:"externalPostId"`    // ID returned by Graph API
    Status          string             `bson:"status"            json:"status"`              // "pending" | "published" | "failed"
    Error           string             `bson:"error"             json:"error"`
    PublishedAt     *time.Time         `bson:"published_at"      json:"publishedAt"`
}
```

---

## MongoDB Indexes

```js
// companies
db.companies.createIndex({ slug: 1 }, { unique: true })

// users
db.users.createIndex({ company_id: 1 })
db.users.createIndex({ email: 1 }, { unique: true })

// products
db.products.createIndex({ company_id: 1, created_at: -1 })
db.products.createIndex({ company_id: 1, brand: 1, category: 1 })
db.products.createIndex({ company_id: 1, tags: 1 })
db.products.createIndex({ company_id: 1, "variants.sku": 1 }, { unique: true })

// stock — unique per variant within a company
db.stock.createIndex({ company_id: 1, variant_id: 1 }, { unique: true })
db.stock.createIndex({ company_id: 1, available: 1 })   // reorder alerts

// stock_movements
db.stock_movements.createIndex({ company_id: 1, variant_id: 1, created_at: -1 })
db.stock_movements.createIndex({ company_id: 1, type: 1, created_at: -1 })

// sales
db.sales.createIndex({ company_id: 1, created_at: -1 })
db.sales.createIndex({ company_id: 1, status: 1 })
db.sales.createIndex({ company_id: 1, invoice_number: 1 }, { unique: true })
db.sales.createIndex({ company_id: 1, "customer.id": 1 })

// customers
db.customers.createIndex({ company_id: 1 })
db.customers.createIndex({ company_id: 1, phone: 1 })

// suppliers
db.suppliers.createIndex({ company_id: 1 })

// social_accounts
db.social_accounts.createIndex({ company_id: 1 })
db.social_accounts.createIndex({ company_id: 1, platform: 1 })
db.social_accounts.createIndex({ token_expiry: 1 })   // token refresh worker

// posts
db.posts.createIndex({ company_id: 1, created_at: -1 })
db.posts.createIndex({ company_id: 1, status: 1 })
db.posts.createIndex({ company_id: 1, product_id: 1 })
db.posts.createIndex({ scheduled_at: 1, status: 1 })  // scheduler worker
```

---

## Multi-Tenancy Rules

- **Every repository method receives `companyID primitive.ObjectID` as a parameter.**
- **Every MongoDB filter includes `company_id: companyID`** — no exceptions.
- `companyID` is extracted from the JWT claims and injected into the Gin context by `middleware/tenant.go`.
- Services retrieve `companyID` from context, never from the request body.

```go
// middleware/tenant.go
func TenantMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        claims := c.MustGet("claims").(*jwt.Claims)
        c.Set("companyID", claims.CompanyID)
        c.Next()
    }
}

// usage in any handler
companyID := c.MustGet("companyID").(primitive.ObjectID)
```

---

## Business Rules

### Invoice number generation
- Format: `{PREFIX}-{YEAR}-{COUNTER}` → `NZO-2024-043`
- `InvoiceCounter` on the company document is incremented atomically using `$inc` + `FindOneAndUpdate`
- Never generate invoice numbers manually in handlers

### Stock updates
- **Never** update `stock.quantity` directly from a handler.
- Always go through `StockService.Move()` which:
  1. Validates the product variant belongs to the authenticated company.
  2. Inserts a `StockMovement` document.
  3. Updates `stock.quantity`, `stock.available`, and `stock.last_movement`.
  4. Rejects `OUT` movements when available stock is insufficient.
- `available` is always calculated as `quantity - reserved`.
- `IN` and `RETURN` increase quantity; `OUT` decreases quantity; `ADJUSTMENT` sets the target quantity.
- Serial-number tracking is deferred for the MVP.
- When a sale or approved stock request is completed, `StockService.Move()` is called for each item.

### Stock request lifecycle
- New inbound and outbound requests start as `pending`.
- `pending` requests may be `approved`, `rejected`, or `cancelled`.
- Only an `approved` request can become `processing` and then `completed`.
- Completion applies the corresponding stock movement for every request item.
- `completed`, `rejected`, and `cancelled` requests are terminal and cannot be processed again.

### Sale → Stock lifecycle
```
draft      → no stock impact
confirmed  → stock.reserved += quantity
paid       → stock.quantity -= quantity, stock.reserved -= quantity (OUT movement)
cancelled  → stock.reserved -= quantity (if was confirmed)
refunded   → stock.quantity += quantity (RETURN movement)
```

### Customer snapshot in sales
- When creating a sale, copy customer fields (name, phone, email) directly into `sale.customer`
- This preserves the sale record even if the customer is later edited or deleted

### Photo resolution
- `variant.images` takes priority over `product.images`
- If `variant.images` is empty, fall back to `product.images`

### Pricing
- All amounts stored in **XOF** (no conversion needed)
- Margin computed on the fly: `(selling - buying) / buying * 100`
- Never store computed margin in DB

### SKU uniqueness
- SKU is unique within each company
- Format: `{BRAND}-{MODEL}-{RAM}-{STORAGE}-{COLOR_CODE}` → `MBP-M3-16-512-SG`

### Social publishing — OAuth flow
- Facebook OAuth2 flow returns a **short-lived user token** → exchanged immediately for a **long-lived user token** (60 days)
- `GET /me/accounts` with the long-lived user token returns **Page Tokens** — these are automatically long-lived
- Only the **Page Token** is stored in DB, never the user token
- Access tokens are **always encrypted with AES-256-GCM** before storage (`pkg/cipher`) — the `json:"-"` tag ensures they are never serialized in API responses
- Instagram Business accounts are linked to a Facebook Page — use the same Page Token to publish on both

### Social publishing — token lifecycle
- `TokenRefreshWorker` runs nightly via Asynq cron
- Finds all `social_accounts` where `token_expiry < now + 7 days`
- If token is still valid per `/debug_token` → attempts refresh via `/me/accounts`
- If expired → sets `is_active: false` and notifies the company admin
- Page Tokens auto-renew when the account is actively used — expiry is mainly a concern for inactive accounts

### Social publishing — post lifecycle
```
draft       → saved, no action
scheduled   → Asynq delayed job enqueued for scheduled_at
publishing  → worker picked up the job, calling Graph API
published   → all targets succeeded, published_at set
failed      → one or more targets failed, error stored in target.error
```
- When `scheduled_at` is nil on POST `/posts`, publish immediately (enqueue with no delay)
- `PublishWorker` updates each `PostTarget.status` individually — partial success is possible
- Rate limits: Facebook 200 calls/hour per token, Instagram 25 posts/24h per account

### AI post generation
- `POST /api/v1/posts/generate` calls Claude API with product specs + company name + tone instructions
- Prompt is built server-side in `post_service.go` — never trust client-provided prompts
- Response is returned as a draft post — user reviews and edits before publishing
- Generated content stored with `generated_by: "ai"` for audit

---

## API Conventions

- Base URL: `/api/v1`
- Auth: `Bearer <access_token>` header
- All responses use this envelope:

```json
{
  "success": true,
  "data": {},
  "message": "ok",
  "error": null
}
```

- Errors return appropriate HTTP status + `"success": false`
- Pagination: `?page=1&limit=20`, response includes `meta.total`, `meta.page`, `meta.limit`

### Key endpoints

```
# Auth
POST   /api/v1/auth/register          # create company + first admin user
POST   /api/v1/auth/login
POST   /api/v1/auth/refresh
POST   /api/v1/auth/logout

# Company
GET    /api/v1/company                 # current tenant info
PUT    /api/v1/company
PUT    /api/v1/company/logo

# Users
GET    /api/v1/users
POST   /api/v1/users
PUT    /api/v1/users/:id
DELETE /api/v1/users/:id

# Products
GET    /api/v1/products
POST   /api/v1/products
GET    /api/v1/products/:id
PUT    /api/v1/products/:id
DELETE /api/v1/products/:id
POST   /api/v1/products/:id/variants
PUT    /api/v1/products/:id/variants/:variantId
DELETE /api/v1/products/:id/variants/:variantId

# Stock
GET    /api/v1/stock                   # all stock (filterable)
GET    /api/v1/stock/alerts            # items below threshold
GET    /api/v1/stock/dashboard         # aggregate stock metrics and recent movements
GET    /api/v1/stock/:variantId
GET    /api/v1/stock/:variantId/movements
POST   /api/v1/stock/:variantId/move   # IN | OUT | RETURN | ADJUSTMENT
POST   /api/v1/stock/requests/inbound
POST   /api/v1/stock/requests/outbound
GET    /api/v1/stock/requests
POST   /api/v1/stock/requests/:id/approve
POST   /api/v1/stock/requests/:id/reject
POST   /api/v1/stock/requests/:id/cancel
POST   /api/v1/stock/requests/:id/complete

# Sales
GET    /api/v1/sales
POST   /api/v1/sales
GET    /api/v1/sales/:id
PUT    /api/v1/sales/:id/status        # status transition
GET    /api/v1/sales/:id/invoice       # PDF export

# Customers
GET    /api/v1/customers
POST   /api/v1/customers
GET    /api/v1/customers/:id
PUT    /api/v1/customers/:id
GET    /api/v1/customers/:id/sales

# Suppliers
GET    /api/v1/suppliers
POST   /api/v1/suppliers
GET    /api/v1/suppliers/:id
PUT    /api/v1/suppliers/:id
DELETE /api/v1/suppliers/:id

# Social accounts
GET    /api/v1/social/accounts                    # list connected accounts
DELETE /api/v1/social/accounts/:id                # disconnect an account
GET    /api/v1/social/connect/facebook            # redirect to Facebook OAuth
GET    /api/v1/social/callback/facebook           # OAuth callback (no auth required)

# Posts
GET    /api/v1/posts                              # list posts (filterable by status, platform)
POST   /api/v1/posts                              # create draft or scheduled post
GET    /api/v1/posts/:id
PUT    /api/v1/posts/:id                          # edit draft
DELETE /api/v1/posts/:id                          # delete draft or cancel scheduled
POST   /api/v1/posts/:id/publish                  # publish immediately
POST   /api/v1/posts/generate                     # AI-generate content from a product
```

---

## Auth & Roles

| Role | Permissions |
|------|-------------|
| `admin` | Full access including user management and company settings |
| `manager` | Products, stock, sales, customers, suppliers, reports |
| `seller` | View products/stock, create and view sales, view customers |

- Role stored on `User` document, included in JWT claims
- RBAC enforced via Gin middleware: `middleware.RequireRole("admin", "manager")`

---

## Environment Variables

```env
APP_ENV=development
APP_PORT=8080

MONGO_URI=mongodb://localhost:27017
MONGO_DB=saas_stock

REDIS_URL=redis://localhost:6379

JWT_ACCESS_SECRET=your_access_secret
JWT_REFRESH_SECRET=your_refresh_secret
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=7d

R2_ENDPOINT=https://<account>.r2.cloudflarestorage.com
R2_BUCKET=saas-stock-assets
R2_ACCESS_KEY=
R2_SECRET_KEY=

FACEBOOK_APP_ID=
FACEBOOK_APP_SECRET=
FACEBOOK_REDIRECT_URI=https://yourapp.com/api/v1/social/callback/facebook

ANTHROPIC_API_KEY=               # AI post generation

TOKEN_ENCRYPTION_KEY=            # 32-byte hex key for AES-256-GCM token encryption
```

---

## Code Style

- Errors wrapped with context: `fmt.Errorf("sale.Create: %w", err)`
- No `panic` in business logic — always return errors
- Repository methods return `(T, error)` — never return `nil, nil`
- Services never import `gin` — keep transport layer separate
- Use `context.Context` as first param on all repository and service methods
- All MongoDB writes touching multiple collections use `WithTransaction`

## File Uploads

- `POST /api/v1/uploads/single` accepts one image in the `file` multipart field.
- `POST /api/v1/uploads/multiple` accepts images in the `files` multipart field.
- Uploads require JWT authentication and are stored under the authenticated `company_id`.
- Supported formats are JPEG, PNG, GIF, and WebP; each file is limited to 10 MB.
- `STORAGE_MODE=local` serves files from `UPLOAD_DIR`; `STORAGE_MODE=r2` stores them in Cloudflare R2.

---

## Testing

- Unit tests for all service methods with mocked repositories
- Integration tests for handlers using `httptest`
- Use `testify/assert` and `testify/mock`
- Test file naming: `sale_service_test.go` beside `sale_service.go`

```bash
go test ./...                  # all tests
go test ./internal/service/... # service tests only
go test -race ./...            # race condition detection
```
