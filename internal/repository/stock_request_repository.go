package repository

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/adapolytech/stock-manager/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const stockRequestsCollection = "stock_requests"

type stockRequestRepository struct {
	collection *mongo.Collection
}

func NewStockRequestRepository(database *mongo.Database) StockRequestRepository {
	return &stockRequestRepository{collection: database.Collection(stockRequestsCollection)}
}

func (r *stockRequestRepository) Create(ctx context.Context, companyID bson.ObjectID, request *domain.StockRequest) (*domain.StockRequest, error) {
	if request == nil {
		return nil, fmt.Errorf("stockRequest.Create: request is required")
	}
	request.CompanyID = companyID
	if request.ID.IsZero() {
		request.ID = bson.NewObjectID()
	}
	if request.CreatedAt.IsZero() {
		request.CreatedAt = time.Now().UTC()
	}
	request.UpdatedAt = request.CreatedAt
	if _, err := r.collection.InsertOne(ctx, request); err != nil {
		return nil, fmt.Errorf("stockRequest.Create: insert request: %w", err)
	}
	return request, nil
}

func (r *stockRequestRepository) GetByID(ctx context.Context, companyID, id bson.ObjectID) (*domain.StockRequest, error) {
	var request domain.StockRequest
	if err := r.collection.FindOne(ctx, stockRequestFilter(companyID, id)).Decode(&request); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("stockRequest.GetByID: request not found: %w", err)
		}
		return nil, fmt.Errorf("stockRequest.GetByID: find request: %w", err)
	}
	return &request, nil
}

func (r *stockRequestRepository) List(ctx context.Context, companyID bson.ObjectID, filter StockRequestFilter) (*StockRequestListResult, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	query := bson.M{"company_id": companyID}
	if filter.Type != "" {
		query["type"] = filter.Type
	}
	if filter.Status != "" {
		query["status"] = filter.Status
	}
	total, err := r.collection.CountDocuments(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("stockRequest.List: count requests: %w", err)
	}
	cursor, err := r.collection.Find(ctx, query, options.Find().
		SetSkip(int64((filter.Page-1)*filter.Limit)).
		SetLimit(int64(filter.Limit)).
		SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("stockRequest.List: find requests: %w", err)
	}
	defer cursor.Close(ctx)
	var requests []*domain.StockRequest
	if err := cursor.All(ctx, &requests); err != nil {
		return nil, fmt.Errorf("stockRequest.List: decode requests: %w", err)
	}
	return &StockRequestListResult{
		Requests: requests, Total: total, Page: filter.Page, Limit: filter.Limit,
		TotalPages: int(math.Ceil(float64(total) / float64(filter.Limit))),
	}, nil
}

func (r *stockRequestRepository) Update(ctx context.Context, companyID, id bson.ObjectID, updates bson.M) (*domain.StockRequest, error) {
	if updates == nil {
		updates = bson.M{}
	}
	updates["updated_at"] = time.Now().UTC()
	var request domain.StockRequest
	if err := r.collection.FindOneAndUpdate(ctx, stockRequestFilter(companyID, id), bson.M{"$set": updates},
		options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&request); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("stockRequest.Update: request not found: %w", err)
		}
		return nil, fmt.Errorf("stockRequest.Update: update request: %w", err)
	}
	return &request, nil
}

func stockRequestFilter(companyID, id bson.ObjectID) bson.M {
	return bson.M{"company_id": companyID, "_id": id}
}
