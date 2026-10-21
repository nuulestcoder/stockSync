package repository

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/adapolytech/stock-manager/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const stockMovementsCollection = "stock_movements"

type stockMovementRepository struct {
	collection *mongo.Collection
}

func NewStockMovementRepository(database *mongo.Database) StockMovementRepository {
	return &stockMovementRepository{collection: database.Collection(stockMovementsCollection)}
}

func (r *stockMovementRepository) Create(ctx context.Context, companyID bson.ObjectID, movement *domain.StockMovement) (*domain.StockMovement, error) {
	if movement == nil {
		return nil, fmt.Errorf("stockMovement.Create: movement is required")
	}
	movement.CompanyID = companyID
	if movement.ID.IsZero() {
		movement.ID = bson.NewObjectID()
	}
	if movement.CreatedAt.IsZero() {
		movement.CreatedAt = time.Now().UTC()
	}
	if _, err := r.collection.InsertOne(ctx, movement); err != nil {
		return nil, fmt.Errorf("stockMovement.Create: insert movement: %w", err)
	}
	return movement, nil
}

func (r *stockMovementRepository) ListByVariant(ctx context.Context, companyID, variantID bson.ObjectID, page, limit int) (*StockMovementListResult, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	query := bson.M{"company_id": companyID, "variant_id": variantID}
	total, err := r.collection.CountDocuments(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("stockMovement.ListByVariant: count movements: %w", err)
	}
	cursor, err := r.collection.Find(ctx, query, options.Find().
		SetSkip(int64((page-1)*limit)).
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("stockMovement.ListByVariant: find movements: %w", err)
	}
	defer cursor.Close(ctx)
	var movements []*domain.StockMovement
	if err := cursor.All(ctx, &movements); err != nil {
		return nil, fmt.Errorf("stockMovement.ListByVariant: decode movements: %w", err)
	}
	return &StockMovementListResult{
		Movements: movements, Total: total, Page: page, Limit: limit,
		TotalPages: int(math.Ceil(float64(total) / float64(limit))),
	}, nil
}
