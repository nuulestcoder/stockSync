package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// StockMovementType identifies how inventory changes.
type StockMovementType string

const (
	StockMovementIn         StockMovementType = "IN"
	StockMovementOut        StockMovementType = "OUT"
	StockMovementReturn     StockMovementType = "RETURN"
	StockMovementAdjustment StockMovementType = "ADJUSTMENT"
)

func (t StockMovementType) IsValid() bool {
	switch t {
	case StockMovementIn, StockMovementOut, StockMovementReturn, StockMovementAdjustment:
		return true
	default:
		return false
	}
}

// SerialNumberStatus identifies the lifecycle state of a serialized unit.
type SerialNumberStatus string

const (
	SerialNumberAvailable SerialNumberStatus = "available"
	SerialNumberReserved  SerialNumberStatus = "reserved"
	SerialNumberSold      SerialNumberStatus = "sold"
)

func (s SerialNumberStatus) IsValid() bool {
	switch s {
	case SerialNumberAvailable, SerialNumberReserved, SerialNumberSold:
		return true
	default:
		return false
	}
}

// StockRequestType identifies whether a stock request adds or removes inventory.
type StockRequestType string

const (
	StockRequestInbound  StockRequestType = "INBOUND"
	StockRequestOutbound StockRequestType = "OUTBOUND"
)

func (t StockRequestType) IsValid() bool {
	return t == StockRequestInbound || t == StockRequestOutbound
}

// StockRequestStatus identifies the lifecycle state of a stock request.
type StockRequestStatus string

const (
	StockRequestDraft      StockRequestStatus = "draft"
	StockRequestPending    StockRequestStatus = "pending"
	StockRequestApproved   StockRequestStatus = "approved"
	StockRequestProcessing StockRequestStatus = "processing"
	StockRequestCompleted  StockRequestStatus = "completed"
	StockRequestCancelled  StockRequestStatus = "cancelled"
	StockRequestRejected   StockRequestStatus = "rejected"
)

func (s StockRequestStatus) IsValid() bool {
	switch s {
	case StockRequestDraft, StockRequestPending, StockRequestApproved,
		StockRequestProcessing, StockRequestCompleted, StockRequestCancelled,
		StockRequestRejected:
		return true
	default:
		return false
	}
}

// Stock stores the current inventory state for one product variant.
type Stock struct {
	ID            bson.ObjectID  `bson:"_id,omitempty" json:"id"`
	CompanyID     bson.ObjectID  `bson:"company_id" json:"companyId"`
	ProductID     bson.ObjectID  `bson:"product_id" json:"productId"`
	VariantID     bson.ObjectID  `bson:"variant_id" json:"variantId"`
	Quantity      int            `bson:"quantity" json:"quantity"`
	Reserved      int            `bson:"reserved" json:"reserved"`
	Available     int            `bson:"available" json:"available"`
	Threshold     int            `bson:"threshold" json:"threshold"`
	SerialNumbers []SerialNumber `bson:"serial_numbers" json:"serialNumbers"`
	Location      string         `bson:"location" json:"location"`
	LastMovement  time.Time      `bson:"last_movement" json:"lastMovement"`
	UpdatedAt     time.Time      `bson:"updated_at" json:"updatedAt"`
}

// SerialNumber tracks an individual serialized inventory unit.
type SerialNumber struct {
	SN     string             `bson:"sn" json:"sn"`
	Status SerialNumberStatus `bson:"status" json:"status"`
}

// StockMovement is an immutable record of an inventory change.
type StockMovement struct {
	ID             bson.ObjectID     `bson:"_id,omitempty" json:"id"`
	CompanyID      bson.ObjectID     `bson:"company_id" json:"companyId"`
	ProductID      bson.ObjectID     `bson:"product_id" json:"productId"`
	VariantID      bson.ObjectID     `bson:"variant_id" json:"variantId"`
	Type           StockMovementType `bson:"type" json:"type"`
	Quantity       int               `bson:"quantity" json:"quantity"`
	QuantityBefore int               `bson:"quantity_before" json:"quantityBefore"`
	QuantityAfter  int               `bson:"quantity_after" json:"quantityAfter"`
	Reason         string            `bson:"reason" json:"reason"`
	Reference      string            `bson:"reference" json:"reference"`
	PerformedBy    bson.ObjectID     `bson:"performed_by" json:"performedBy"`
	CreatedAt      time.Time         `bson:"created_at" json:"createdAt"`
}

// StockRequest represents an inbound or outbound inventory request.
type StockRequest struct {
	ID          bson.ObjectID      `bson:"_id,omitempty" json:"id"`
	CompanyID   bson.ObjectID      `bson:"company_id" json:"companyId"`
	Type        StockRequestType   `bson:"type" json:"type"`
	Status      StockRequestStatus `bson:"status" json:"status"`
	Items       []StockRequestItem `bson:"items" json:"items"`
	Reason      string             `bson:"reason" json:"reason"`
	Reference   string             `bson:"reference" json:"reference"`
	RequestedBy bson.ObjectID      `bson:"requested_by" json:"requestedBy"`
	ApprovedBy  *bson.ObjectID     `bson:"approved_by,omitempty" json:"approvedBy,omitempty"`
	CompletedBy *bson.ObjectID     `bson:"completed_by,omitempty" json:"completedBy,omitempty"`
	ApprovedAt  *time.Time         `bson:"approved_at,omitempty" json:"approvedAt,omitempty"`
	CompletedAt *time.Time         `bson:"completed_at,omitempty" json:"completedAt,omitempty"`
	CreatedAt   time.Time          `bson:"created_at" json:"createdAt"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updatedAt"`
}

// StockRequestItem describes one variant and quantity in a stock request.
type StockRequestItem struct {
	ProductID     bson.ObjectID `bson:"product_id" json:"productId"`
	VariantID     bson.ObjectID `bson:"variant_id" json:"variantId"`
	Quantity      int           `bson:"quantity" json:"quantity"`
	SerialNumbers []string      `bson:"serial_numbers,omitempty" json:"serialNumbers,omitempty"`
	Location      string        `bson:"location,omitempty" json:"location,omitempty"`
}

// MoveStockRequest requests an immediate inventory movement.
type MoveStockRequest struct {
	Type          StockMovementType `json:"type" binding:"required,oneof=IN OUT RETURN ADJUSTMENT"`
	Quantity      int               `json:"quantity" binding:"required,gt=0"`
	Reason        string            `json:"reason" binding:"required"`
	Reference     string            `json:"reference"`
	SerialNumbers []string          `json:"serialNumbers"`
}

// CreateStockRequest is the DTO for creating an inbound or outbound request.
type CreateStockRequest struct {
	Type      StockRequestType   `json:"type" binding:"required,oneof=INBOUND OUTBOUND"`
	Items     []StockRequestItem `json:"items" binding:"required,min=1"`
	Reason    string             `json:"reason" binding:"required"`
	Reference string             `json:"reference"`
}

// UpdateStockRequest updates configurable stock metadata.
type UpdateStockRequest struct {
	Threshold *int    `json:"threshold" binding:"omitempty,gte=0"`
	Location  *string `json:"location"`
}

// StockFilter contains supported stock list filters.
type StockFilter struct {
	ProductID  *bson.ObjectID
	VariantID  *bson.ObjectID
	BelowLevel *bool
	OutOfStock *bool
	Location   string
	Page       int
	Limit      int
}
