package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Product struct {
	ID          bson.ObjectID `bson:"_id,omitempty" json:"id"`
	CompanyID   bson.ObjectID `bson:"company_id" json:"companyId"`
	Name        string        `bson:"name" json:"name"`
	Brand       string        `bson:"brand" json:"brand"`
	Category    string        `bson:"category" json:"category"`
	Description string        `bson:"description" json:"description"`
	Images      []string      `bson:"images" json:"images"`
	Tags        []string      `bson:"tags" json:"tags"`
	Variants    []Variant     `bson:"variants" json:"variants"`
	IsDeleted   bool          `bson:"is_deleted" json:"-"`
	CreatedAt   time.Time     `bson:"created_at" json:"createdAt"`
	UpdatedAt   time.Time     `bson:"updated_at" json:"updatedAt"`
}

type Variant struct {
	ID     bson.ObjectID `bson:"_id,omitempty" json:"id"`
	SKU    string        `bson:"sku" json:"sku"`
	Specs  VariantSpecs  `bson:"specs" json:"specs"`
	Price  Price         `bson:"price" json:"price"`
	Images []string      `bson:"images" json:"images"`
}

type VariantSpecs struct {
	CPU       string `bson:"cpu,omitempty" json:"cpu,omitempty"`
	RAM       string `bson:"ram,omitempty" json:"ram,omitempty"`
	Storage   string `bson:"storage,omitempty" json:"storage,omitempty"`
	Color     string `bson:"color,omitempty" json:"color,omitempty"`
	Condition string `bson:"condition,omitempty" json:"condition,omitempty"`
}

type Price struct {
	Buying   float64 `bson:"buying" json:"buying"`
	Selling  float64 `bson:"selling" json:"selling"`
	Currency string  `bson:"currency" json:"currency"`
}

type CreateProductRequest struct {
	Name        string    `json:"name" binding:"required"`
	Brand       string    `json:"brand" binding:"required"`
	Category    string    `json:"category" binding:"required"`
	Description string    `json:"description"`
	Images      []string  `json:"images"`
	Tags        []string  `json:"tags"`
	Variants    []Variant `json:"variants" binding:"required,min=1"`
}

type UpdateProductRequest struct {
	Name        *string   `json:"name"`
	Brand       *string   `json:"brand"`
	Category    *string   `json:"category"`
	Description *string   `json:"description"`
	Images      *[]string `json:"images"`
	Tags        *[]string `json:"tags"`
}

type CreateVariantRequest struct {
	SKU    string       `json:"sku" binding:"required"`
	Specs  VariantSpecs `json:"specs"`
	Price  Price        `json:"price" binding:"required"`
	Images []string     `json:"images"`
}

type UpdateVariantRequest struct {
	SKU    *string       `json:"sku"`
	Specs  *VariantSpecs `json:"specs"`
	Price  *Price        `json:"price"`
	Images *[]string     `json:"images"`
}
