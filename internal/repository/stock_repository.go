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

const stockCollection = "stock"

type stockRepository struct {
	collection *mongo.Collection
	database   *mongo.Database
}

func NewStockRepository(database *mongo.Database) StockRepository {
	return &stockRepository{collection: database.Collection(stockCollection), database: database}
}

func EnsureStockIndexes(ctx context.Context, database *mongo.Database) error {
	stockIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "company_id", Value: 1}, {Key: "variant_id", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "company_id", Value: 1}, {Key: "available", Value: 1}}},
	}
	if _, err := database.Collection(stockCollection).Indexes().CreateMany(ctx, stockIndexes); err != nil {
		return fmt.Errorf("stock.EnsureIndexes: create stock indexes: %w", err)
	}

	movementIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "company_id", Value: 1}, {Key: "variant_id", Value: 1}, {Key: "created_at", Value: -1}}},
		{Keys: bson.D{{Key: "company_id", Value: 1}, {Key: "type", Value: 1}, {Key: "created_at", Value: -1}}},
	}
	if _, err := database.Collection(stockMovementsCollection).Indexes().CreateMany(ctx, movementIndexes); err != nil {
		return fmt.Errorf("stock.EnsureIndexes: create movement indexes: %w", err)
	}

	requestIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "company_id", Value: 1}, {Key: "status", Value: 1}, {Key: "created_at", Value: -1}}},
	}
	if _, err := database.Collection(stockRequestsCollection).Indexes().CreateMany(ctx, requestIndexes); err != nil {
		return fmt.Errorf("stock.EnsureIndexes: create request indexes: %w", err)
	}
	return nil
}

func (r *stockRepository) Create(ctx context.Context, companyID bson.ObjectID, stock *domain.Stock) (*domain.Stock, error) {
	if stock == nil {
		return nil, fmt.Errorf("stock.Create: stock is required")
	}
	stock.CompanyID = companyID
	if stock.ID.IsZero() {
		stock.ID = bson.NewObjectID()
	}
	stock.Available = stock.Quantity - stock.Reserved
	stock.UpdatedAt = time.Now().UTC()
	if _, err := r.collection.InsertOne(ctx, stock); err != nil {
		return nil, fmt.Errorf("stock.Create: insert stock: %w", err)
	}
	return stock, nil
}

func (r *stockRepository) GetByVariant(ctx context.Context, companyID, variantID bson.ObjectID) (*domain.Stock, error) {
	var stock domain.Stock
	if err := r.collection.FindOne(ctx, stockFilter(companyID, variantID)).Decode(&stock); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("stock.GetByVariant: stock not found: %w", err)
		}
		return nil, fmt.Errorf("stock.GetByVariant: find stock: %w", err)
	}
	return &stock, nil
}

func (r *stockRepository) List(ctx context.Context, companyID bson.ObjectID, filter domain.StockFilter) (*StockListResult, error) {
	return r.ListViews(ctx, companyID, filter)
}

func (r *stockRepository) GetViewByVariant(ctx context.Context, companyID, variantID bson.ObjectID) (*StockView, error) {
	result, err := r.ListViews(ctx, companyID, domain.StockFilter{VariantID: &variantID, Page: 1, Limit: 1})
	if err != nil {
		return nil, err
	}
	if len(result.Stocks) == 0 {
		return nil, fmt.Errorf("stock.GetViewByVariant: stock not found")
	}
	return result.Stocks[0], nil
}

func (r *stockRepository) ListViews(ctx context.Context, companyID bson.ObjectID, filter domain.StockFilter) (*StockListResult, error) {
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
	if filter.ProductID != nil {
		query["product_id"] = *filter.ProductID
	}
	if filter.VariantID != nil {
		query["variant_id"] = *filter.VariantID
	}
	if filter.Location != "" {
		query["location"] = filter.Location
	}
	if filter.BelowLevel != nil && *filter.BelowLevel {
		query["$expr"] = bson.M{"$lte": bson.A{"$available", "$threshold"}}
	}
	if filter.OutOfStock != nil && *filter.OutOfStock {
		query["available"] = bson.M{"$lte": 0}
	}

	total, err := r.collection.CountDocuments(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("stock.List: count stock: %w", err)
	}
	pipeline := stockViewPipeline(query, filter.Page, filter.Limit)
	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("stock.List: aggregate stock views: %w", err)
	}
	defer cursor.Close(ctx)

	var rows []stockViewRow
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, fmt.Errorf("stock.List: decode stock views: %w", err)
	}
	stocks := make([]*StockView, 0, len(rows))
	for _, row := range rows {
		stock := row.Stock
		stocks = append(stocks, &StockView{
			Stock: &stock, Product: row.Product, Variant: row.Variant,
		})
	}
	return &StockListResult{
		Stocks: stocks, Total: total, Page: filter.Page, Limit: filter.Limit,
		TotalPages: int(math.Ceil(float64(total) / float64(filter.Limit))),
	}, nil
}

type stockViewRow struct {
	Stock   domain.Stock     `bson:",inline"`
	Product StockProductView `bson:"product"`
	Variant domain.Variant   `bson:"variant"`
}

func stockViewPipeline(query bson.M, page, limit int) mongo.Pipeline {
	return mongo.Pipeline{
		bson.D{{Key: "$match", Value: query}},
		bson.D{{Key: "$lookup", Value: bson.M{
			"from":         "products",
			"localField":   "product_id",
			"foreignField": "_id",
			"as":           "product",
		}}},
		bson.D{{Key: "$unwind", Value: "$product"}},
		bson.D{{Key: "$match", Value: bson.M{
			"product.company_id": companyIDFromStockQuery(query),
			"product.is_deleted": bson.M{"$ne": true},
		}}},
		bson.D{{Key: "$project", Value: bson.M{
			"_id": 1, "company_id": 1, "product_id": 1, "variant_id": 1,
			"quantity": 1, "reserved": 1, "available": 1, "threshold": 1,
			"serial_numbers": 1, "location": 1, "last_movement": 1, "updated_at": 1,
			"product": bson.M{
				"id": "$product._id", "name": "$product.name", "brand": "$product.brand",
				"category": "$product.category", "images": "$product.images",
			},
			"variant": bson.M{"$arrayElemAt": bson.A{
				bson.M{"$filter": bson.M{
					"input": "$product.variants", "as": "variant",
					"cond": bson.M{"$eq": bson.A{"$$variant._id", "$variant_id"}},
				}}, 0,
			}},
		}}},
		bson.D{{Key: "$match", Value: bson.M{"variant._id": bson.M{"$exists": true}}}},
		bson.D{{Key: "$sort", Value: bson.D{{Key: "updated_at", Value: -1}}}},
		bson.D{{Key: "$skip", Value: int64((page - 1) * limit)}},
		bson.D{{Key: "$limit", Value: int64(limit)}},
	}
}

func companyIDFromStockQuery(query bson.M) bson.ObjectID {
	companyID, _ := query["company_id"].(bson.ObjectID)
	return companyID
}

func (r *stockRepository) Update(ctx context.Context, companyID, variantID bson.ObjectID, updates bson.M) (*domain.Stock, error) {
	if updates == nil {
		updates = bson.M{}
	}
	updates["updated_at"] = time.Now().UTC()
	var stock domain.Stock
	if err := r.collection.FindOneAndUpdate(ctx, stockFilter(companyID, variantID), bson.M{"$set": updates},
		options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&stock); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("stock.Update: stock not found: %w", err)
		}
		return nil, fmt.Errorf("stock.Update: update stock: %w", err)
	}
	return &stock, nil
}

func (r *stockRepository) GetDashboard(ctx context.Context, companyID bson.ObjectID) (*StockDashboard, error) {
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.M{"company_id": companyID}}},
		bson.D{{Key: "$group", Value: bson.M{
			"_id":            nil,
			"totalVariants":  bson.M{"$sum": 1},
			"totalQuantity":  bson.M{"$sum": "$quantity"},
			"totalReserved":  bson.M{"$sum": "$reserved"},
			"totalAvailable": bson.M{"$sum": "$available"},
			"outOfStockVariants": bson.M{"$sum": bson.M{"$cond": bson.A{
				bson.M{"$lte": bson.A{"$available", 0}}, 1, 0,
			}}},
			"lowStockVariants": bson.M{"$sum": bson.M{"$cond": bson.A{
				bson.M{"$lte": bson.A{"$available", "$threshold"}}, 1, 0,
			}}},
		}}},
	}
	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("stock.GetDashboard: aggregate stock metrics: %w", err)
	}
	defer cursor.Close(ctx)

	var metrics []struct {
		TotalVariants      int64 `bson:"totalVariants"`
		TotalQuantity      int64 `bson:"totalQuantity"`
		TotalReserved      int64 `bson:"totalReserved"`
		TotalAvailable     int64 `bson:"totalAvailable"`
		OutOfStockVariants int64 `bson:"outOfStockVariants"`
		LowStockVariants   int64 `bson:"lowStockVariants"`
	}
	if err := cursor.All(ctx, &metrics); err != nil {
		return nil, fmt.Errorf("stock.GetDashboard: decode stock metrics: %w", err)
	}
	dashboard := &StockDashboard{}
	if len(metrics) > 0 {
		dashboard.TotalVariants = metrics[0].TotalVariants
		dashboard.TotalQuantity = metrics[0].TotalQuantity
		dashboard.TotalReserved = metrics[0].TotalReserved
		dashboard.TotalAvailable = metrics[0].TotalAvailable
		dashboard.OutOfStockVariants = metrics[0].OutOfStockVariants
		dashboard.LowStockVariants = metrics[0].LowStockVariants
	}
	totals, err := r.movementTotals(ctx, companyID)
	if err != nil {
		return nil, err
	}
	dashboard.InboundQuantity = totals.InboundQuantity
	dashboard.OutboundQuantity = totals.OutboundQuantity
	dashboard.RecentMovements, err = r.recentMovements(ctx, companyID, 10)
	if err != nil {
		return nil, err
	}
	return dashboard, nil
}

type movementTotalsResult struct {
	InboundQuantity  int64
	OutboundQuantity int64
}

func (r *stockRepository) movementTotals(ctx context.Context, companyID bson.ObjectID) (*movementTotalsResult, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{
			{Key: "company_id", Value: companyID},
			{Key: "type", Value: bson.D{{Key: "$in", Value: bson.A{domain.StockMovementIn, domain.StockMovementReturn, domain.StockMovementOut}}}},
		}}},
		{{Key: "$group", Value: bson.D{{Key: "_id", Value: "$type"}, {Key: "quantity", Value: bson.D{{Key: "$sum", Value: "$quantity"}}}}}},
	}
	cursor, err := r.database.Collection(stockMovementsCollection).Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("stock.GetDashboard: aggregate movement totals: %w", err)
	}
	defer cursor.Close(ctx)

	var totals []struct {
		Type     domain.StockMovementType `bson:"_id"`
		Quantity int64                    `bson:"quantity"`
	}
	if err := cursor.All(ctx, &totals); err != nil {
		return nil, fmt.Errorf("stock.GetDashboard: decode movement totals: %w", err)
	}
	result := &movementTotalsResult{}
	for _, total := range totals {
		if total.Type == domain.StockMovementOut {
			result.OutboundQuantity += total.Quantity
		} else {
			result.InboundQuantity += total.Quantity
		}
	}
	return result, nil
}

func (r *stockRepository) recentMovements(ctx context.Context, companyID bson.ObjectID, limit int64) ([]*domain.StockMovement, error) {
	cursor, err := r.database.Collection(stockMovementsCollection).Find(
		ctx,
		bson.M{"company_id": companyID},
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(limit),
	)
	if err != nil {
		return nil, fmt.Errorf("stock.GetDashboard: find recent movements: %w", err)
	}
	defer cursor.Close(ctx)

	var movements []*domain.StockMovement
	if err := cursor.All(ctx, &movements); err != nil {
		return nil, fmt.Errorf("stock.GetDashboard: decode recent movements: %w", err)
	}
	if movements == nil {
		movements = make([]*domain.StockMovement, 0)
	}
	return movements, nil
}

func stockFilter(companyID, variantID bson.ObjectID) bson.M {
	return bson.M{"company_id": companyID, "variant_id": variantID}
}
