package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Company represents a tenant — one reseller business using the SaaS platform
type Company struct {
	ID        bson.ObjectID   `bson:"_id,omitempty" json:"id"`
	Name      string          `bson:"name"          json:"name"`
	Slug      string          `bson:"slug"          json:"slug"` // unique, URL-friendly
	Logo      string          `bson:"logo"          json:"logo"` // R2 URL
	Address   CompanyAddress  `bson:"address"       json:"address"`
	Contact   CompanyContact  `bson:"contact"       json:"contact"`
	Billing   CompanyBilling  `bson:"billing"       json:"billing"`
	Settings  CompanySettings `bson:"settings"      json:"settings"`
	Plan      string          `bson:"plan"          json:"plan"` // "starter" | "pro" | "enterprise"
	IsActive  bool            `bson:"is_active"     json:"isActive"`
	CreatedAt time.Time       `bson:"created_at"    json:"createdAt"`
	UpdatedAt time.Time       `bson:"updated_at"    json:"updatedAt"`
}

// CompanyAddress holds physical location information
type CompanyAddress struct {
	Street  string `bson:"street"  json:"street"`
	City    string `bson:"city"    json:"city"`
	Country string `bson:"country" json:"country"` // ISO 3166-1 alpha-2, e.g. "SN" for Senegal
}

// CompanyContact holds communication information
type CompanyContact struct {
	Email    string `bson:"email"    json:"email"`
	Phone    string `bson:"phone"    json:"phone"`
	WhatsApp string `bson:"whatsapp" json:"whatsapp"`
}

// CompanyBilling holds regulatory/tax information (Senegal-specific)
type CompanyBilling struct {
	NINEA string `bson:"ninea" json:"ninea"` // Numéro d'Identification National des Entreprises et Associations
	RCCM  string `bson:"rccm"  json:"rccm"`  // Registre du Commerce et du Crédit Mobilier
}

// CompanySettings holds tenant-specific configuration
type CompanySettings struct {
	Currency       string   `bson:"currency"        json:"currency"`       // "XOF" for West African CFA franc
	InvoicePrefix  string   `bson:"invoice_prefix"  json:"invoicePrefix"`  // e.g. "NZO"
	InvoiceCounter int      `bson:"invoice_counter" json:"invoiceCounter"` // auto-incremented per company
	TaxRate        float64  `bson:"tax_rate"        json:"taxRate"`        // 0 if not applicable
	PaymentMethods []string `bson:"payment_methods" json:"paymentMethods"` // ["wave", "orange_money", "cash", "bank_transfer"]
}

// CreateCompanyRequest is the DTO for creating a new company
type CreateCompanyRequest struct {
	Name     string          `json:"name" binding:"required"`
	Address  CompanyAddress  `json:"address" binding:"required"`
	Contact  CompanyContact  `json:"contact" binding:"required"`
	Billing  CompanyBilling  `json:"billing"`
	Settings CompanySettings `json:"settings"`
	Plan     string          `json:"plan" binding:"required"` // default: "starter"
}

// RegisterRequest contains the company and first administrator details.
type RegisterRequest struct {
	Company  CreateCompanyRequest `json:"company" binding:"required"`
	Name     string               `json:"name"     binding:"required"`
	Email    string               `json:"email"    binding:"required,email"`
	Phone    string               `json:"phone"`
	Password string               `json:"password" binding:"required,min=8"`
}

// UpdateCompanyRequest is the DTO for updating a company
type UpdateCompanyRequest struct {
	Name     *string          `json:"name"`
	Address  *CompanyAddress  `json:"address"`
	Contact  *CompanyContact  `json:"contact"`
	Billing  *CompanyBilling  `json:"billing"`
	Settings *CompanySettings `json:"settings"`
	Plan     *string          `json:"plan"`
	IsActive *bool            `json:"isActive"`
}
