package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type SaleStatus string

const (
	SaleStatusDraft     SaleStatus = "draft"
	SaleStatusConfirmed SaleStatus = "confirmed"
	SaleStatusPaid      SaleStatus = "paid"
	SaleStatusCancelled SaleStatus = "cancelled"
	SaleStatusRefunded  SaleStatus = "refunded"
)

func (s SaleStatus) IsValid() bool {
	switch s {
	case SaleStatusDraft, SaleStatusConfirmed, SaleStatusPaid, SaleStatusCancelled, SaleStatusRefunded:
		return true
	default:
		return false
	}
}

type PaymentMethod string

const (
	PaymentMethodCash         PaymentMethod = "cash"
	PaymentMethodWave         PaymentMethod = "wave"
	PaymentMethodOrangeMoney  PaymentMethod = "orange_money"
	PaymentMethodBankTransfer PaymentMethod = "bank_transfer"
)

func (m PaymentMethod) IsValid() bool {
	switch m {
	case PaymentMethodCash, PaymentMethodWave, PaymentMethodOrangeMoney, PaymentMethodBankTransfer:
		return true
	default:
		return false
	}
}

type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusCompleted PaymentStatus = "completed"
	PaymentStatusFailed    PaymentStatus = "failed"
)

func (s PaymentStatus) IsValid() bool {
	switch s {
	case PaymentStatusPending, PaymentStatusCompleted, PaymentStatusFailed:
		return true
	default:
		return false
	}
}

type SaleChannel string

const (
	SaleChannelInStore  SaleChannel = "in_store"
	SaleChannelWhatsApp SaleChannel = "whatsapp"
	SaleChannelFacebook SaleChannel = "facebook"
)

func (c SaleChannel) IsValid() bool {
	switch c {
	case SaleChannelInStore, SaleChannelWhatsApp, SaleChannelFacebook:
		return true
	default:
		return false
	}
}

type Sale struct {
	ID            bson.ObjectID `bson:"_id,omitempty" json:"id"`
	CompanyID     bson.ObjectID `bson:"company_id" json:"companyId"`
	InvoiceNumber string        `bson:"invoice_number" json:"invoiceNumber"`
	Status        SaleStatus    `bson:"status" json:"status"`
	Customer      SaleCustomer  `bson:"customer" json:"customer"`
	Items         []SaleItem    `bson:"items" json:"items"`
	Subtotal      float64       `bson:"subtotal" json:"subtotal"`
	Discount      float64       `bson:"discount" json:"discount"`
	TaxAmount     float64       `bson:"tax_amount" json:"taxAmount"`
	Total         float64       `bson:"total" json:"total"`
	Currency      string        `bson:"currency" json:"currency"`
	Payment       SalePayment   `bson:"payment" json:"payment"`
	Channel       SaleChannel   `bson:"channel" json:"channel"`
	Notes         string        `bson:"notes" json:"notes"`
	CreatedBy     bson.ObjectID `bson:"created_by" json:"createdBy"`
	CreatedAt     time.Time     `bson:"created_at" json:"createdAt"`
	UpdatedAt     time.Time     `bson:"updated_at" json:"updatedAt"`
}

type SaleCustomer struct {
	ID    *bson.ObjectID `bson:"id,omitempty" json:"id,omitempty"`
	Name  string         `bson:"name" json:"name"`
	Phone string         `bson:"phone" json:"phone"`
	Email string         `bson:"email" json:"email"`
}

type SaleItem struct {
	ProductID    bson.ObjectID `bson:"product_id" json:"productId"`
	VariantID    bson.ObjectID `bson:"variant_id" json:"variantId"`
	ProductName  string        `bson:"product_name" json:"productName"`
	SKU          string        `bson:"sku" json:"sku"`
	Specs        VariantSpecs  `bson:"specs" json:"specs"`
	SerialNumber string        `bson:"serial_number,omitempty" json:"serialNumber,omitempty"`
	Quantity     int           `bson:"quantity" json:"quantity"`
	UnitPrice    float64       `bson:"unit_price" json:"unitPrice"`
	Discount     float64       `bson:"discount" json:"discount"`
	Total        float64       `bson:"total" json:"total"`
}

type SalePayment struct {
	Method    PaymentMethod `bson:"method" json:"method"`
	Status    PaymentStatus `bson:"status" json:"status"`
	PaidAt    time.Time     `bson:"paid_at,omitempty" json:"paidAt,omitempty"`
	Reference string        `bson:"reference,omitempty" json:"reference,omitempty"`
}

type CreateSaleRequest struct {
	Customer SaleCustomer      `json:"customer"`
	Items    []CreateSaleItem  `json:"items" binding:"required,min=1"`
	Discount float64           `json:"discount"`
	Channel  SaleChannel       `json:"channel" binding:"required"`
	Notes    string            `json:"notes"`
	Payment  CreateSalePayment `json:"payment"`
}

type CreateSaleItem struct {
	ProductID bson.ObjectID `json:"productId" binding:"required"`
	VariantID bson.ObjectID `json:"variantId" binding:"required"`
	Quantity  int           `json:"quantity" binding:"required,min=1"`
	Discount  float64       `json:"discount"`
}

type CreateSalePayment struct {
	Method    PaymentMethod `json:"method" binding:"required"`
	Status    PaymentStatus `json:"status"`
	Reference string        `json:"reference"`
}

type UpdateSaleStatusRequest struct {
	Status    SaleStatus   `json:"status" binding:"required"`
	Payment   *SalePayment `json:"payment"`
	Reference string       `json:"reference"`
}
