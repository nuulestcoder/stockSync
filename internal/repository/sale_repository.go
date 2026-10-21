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

const salesCollection = "sales"

type saleRepository struct {
	collection *mongo.Collection
}

func NewSaleRepository(database *mongo.Database) SaleRepository {
	return &saleRepository{collection: database.Collection(salesCollection)}
}

func EnsureSaleIndexes(ctx context.Context, database *mongo.Database) error {
	_, err := database.Collection(salesCollection).Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "company_id", Value: 1}, {Key: "created_at", Value: -1}}},
		{Keys: bson.D{{Key: "company_id", Value: 1}, {Key: "status", Value: 1}}},
		{
			Keys:    bson.D{{Key: "company_id", Value: 1}, {Key: "invoice_number", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{Keys: bson.D{{Key: "company_id", Value: 1}, {Key: "customer.id", Value: 1}}},
	})
	if err != nil {
		return fmt.Errorf("sale.EnsureIndexes: create indexes: %w", err)
	}
	return nil
}

func (r *saleRepository) Create(ctx context.Context, companyID bson.ObjectID, sale *domain.Sale) (*domain.Sale, error) {
	if sale == nil {
		return nil, fmt.Errorf("sale.Create: sale is required")
	}
	sale.CompanyID = companyID
	if sale.ID.IsZero() {
		sale.ID = bson.NewObjectID()
	}
	now := time.Now().UTC()
	if sale.CreatedAt.IsZero() {
		sale.CreatedAt = now
	}
	sale.UpdatedAt = now
	if _, err := r.collection.InsertOne(ctx, sale); err != nil {
		return nil, fmt.Errorf("sale.Create: insert sale: %w", err)
	}
	return sale, nil
}

func (r *saleRepository) GetByID(ctx context.Context, companyID, id bson.ObjectID) (*domain.Sale, error) {
	var sale domain.Sale
	if err := r.collection.FindOne(ctx, bson.M{"company_id": companyID, "_id": id}).Decode(&sale); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("sale.GetByID: sale not found: %w", err)
		}
		return nil, fmt.Errorf("sale.GetByID: find sale: %w", err)
	}
	return &sale, nil
}

func (r *saleRepository) List(ctx context.Context, companyID bson.ObjectID, filter SaleFilter) (*SaleListResult, error) {
	query := bson.M{"company_id": companyID}
	if filter.Status != "" {
		query["status"] = filter.Status
	}
	if filter.PaymentStatus != "" {
		query["payment.status"] = filter.PaymentStatus
	}
	if filter.PaymentMethod != "" {
		query["payment.method"] = filter.PaymentMethod
	}

	total, err := r.collection.CountDocuments(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("sale.List: count sales: %w", err)
	}
	cursor, err := r.collection.Find(ctx, query, options.Find().
		SetSkip(int64((filter.Page-1)*filter.Limit)).
		SetLimit(int64(filter.Limit)).
		SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("sale.List: find sales: %w", err)
	}
	defer cursor.Close(ctx)

	var sales []*domain.Sale
	if err := cursor.All(ctx, &sales); err != nil {
		return nil, fmt.Errorf("sale.List: decode sales: %w", err)
	}
	return &SaleListResult{
		Sales: sales, Total: total, Page: filter.Page, Limit: filter.Limit,
		TotalPages: int(math.Ceil(float64(total) / float64(filter.Limit))),
	}, nil
}

func (r *saleRepository) UpdateStatus(ctx context.Context, companyID, id bson.ObjectID, status domain.SaleStatus) (*domain.Sale, error) {
	var sale domain.Sale
	if err := r.collection.FindOneAndUpdate(ctx, bson.M{"company_id": companyID, "_id": id},
		bson.M{"$set": bson.M{"status": status, "updated_at": time.Now().UTC()}},
		options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&sale); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("sale.UpdateStatus: sale not found: %w", err)
		}
		return nil, fmt.Errorf("sale.UpdateStatus: update sale: %w", err)
	}
	return &sale, nil
}

func (r *saleRepository) GetDashboard(ctx context.Context, companyID bson.ObjectID, filter SaleReportFilter) (*SaleDashboard, error) {
	match := bson.M{"company_id": companyID}
	createdAt := bson.M{}
	if filter.From != nil {
		createdAt["$gte"] = *filter.From
	}
	if filter.To != nil {
		createdAt["$lt"] = *filter.To
	}
	if len(createdAt) > 0 {
		match["created_at"] = createdAt
	}

	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: match}},
		bson.D{{Key: "$facet", Value: bson.M{
			"summary": bson.A{
				bson.D{{Key: "$group", Value: bson.M{
					"_id": nil, "totalSales": bson.M{"$sum": 1},
					"draftSales":     bson.M{"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$status", "draft"}}, 1, 0}}},
					"confirmedSales": bson.M{"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$status", "confirmed"}}, 1, 0}}},
					"paidSales":      bson.M{"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$status", "paid"}}, 1, 0}}},
					"cancelledSales": bson.M{"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$status", "cancelled"}}, 1, 0}}},
					"refundedSales":  bson.M{"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$status", "refunded"}}, 1, 0}}},
					"grossRevenue":   bson.M{"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$status", "paid"}}, "$total", 0}}},
					"discounts":      bson.M{"$sum": "$discount"}, "taxAmount": bson.M{"$sum": "$tax_amount"},
				}}},
			},
			"byPaymentMethod": bson.A{
				bson.D{{Key: "$match", Value: bson.M{"status": "paid"}}},
				bson.D{{Key: "$group", Value: bson.M{"_id": "$payment.method", "count": bson.M{"$sum": 1}, "amount": bson.M{"$sum": "$total"}}}},
				bson.D{{Key: "$project", Value: bson.M{"_id": 0, "key": "$_id", "count": 1, "amount": 1}}},
				bson.D{{Key: "$sort", Value: bson.D{{Key: "amount", Value: -1}}}},
			},
			"byChannel": bson.A{
				bson.D{{Key: "$match", Value: bson.M{"status": "paid"}}},
				bson.D{{Key: "$group", Value: bson.M{"_id": "$channel", "count": bson.M{"$sum": 1}, "amount": bson.M{"$sum": "$total"}}}},
				bson.D{{Key: "$project", Value: bson.M{"_id": 0, "key": "$_id", "count": 1, "amount": 1}}},
				bson.D{{Key: "$sort", Value: bson.D{{Key: "amount", Value: -1}}}},
			},
			"topProducts": bson.A{
				bson.D{{Key: "$match", Value: bson.M{"status": "paid"}}},
				bson.D{{Key: "$unwind", Value: "$items"}},
				bson.D{{Key: "$group", Value: bson.M{
					"_id":      bson.M{"productId": "$items.product_id", "sku": "$items.sku", "name": "$items.product_name"},
					"quantity": bson.M{"$sum": "$items.quantity"}, "revenue": bson.M{"$sum": "$items.total"},
				}}},
				bson.D{{Key: "$project", Value: bson.M{
					"_id": 0, "productId": "$_id.productId", "sku": "$_id.sku", "name": "$_id.name", "quantity": 1, "revenue": 1,
				}}},
				bson.D{{Key: "$sort", Value: bson.D{{Key: "quantity", Value: -1}, {Key: "revenue", Value: -1}}}},
				bson.D{{Key: "$limit", Value: 10}},
			},
			"dailyRevenue": bson.A{
				bson.D{{Key: "$match", Value: bson.M{"status": bson.M{"$in": bson.A{"paid", "refunded"}}}}},
				bson.D{{Key: "$group", Value: bson.M{
					"_id":   bson.M{"$dateToString": bson.M{"format": "%Y-%m-%d", "date": "$created_at", "timezone": "UTC"}},
					"count": bson.M{"$sum": 1}, "amount": bson.M{"$sum": "$total"},
				}}},
				bson.D{{Key: "$project", Value: bson.M{"_id": 0, "date": "$_id", "count": 1, "amount": 1}}},
				bson.D{{Key: "$sort", Value: bson.D{{Key: "date", Value: 1}}}},
			},
		}}},
	}
	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("sale.GetDashboard: aggregate sales: %w", err)
	}
	defer cursor.Close(ctx)

	var rows []saleDashboardRow
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, fmt.Errorf("sale.GetDashboard: decode dashboard: %w", err)
	}
	dashboard := &SaleDashboard{}
	if len(rows) == 0 {
		return dashboard, nil
	}
	dashboard = rows[0].toDashboard()
	return dashboard, nil
}

type saleDashboardRow struct {
	Summary         []saleSummary         `bson:"summary"`
	ByPaymentMethod []SaleDimensionMetric `bson:"byPaymentMethod"`
	ByChannel       []SaleDimensionMetric `bson:"byChannel"`
	TopProducts     []SaleProductMetric   `bson:"topProducts"`
	DailyRevenue    []SaleDailyMetric     `bson:"dailyRevenue"`
}

type saleSummary struct {
	TotalSales     int64   `bson:"totalSales"`
	DraftSales     int64   `bson:"draftSales"`
	ConfirmedSales int64   `bson:"confirmedSales"`
	PaidSales      int64   `bson:"paidSales"`
	CancelledSales int64   `bson:"cancelledSales"`
	RefundedSales  int64   `bson:"refundedSales"`
	GrossRevenue   float64 `bson:"grossRevenue"`
	Discounts      float64 `bson:"discounts"`
	TaxAmount      float64 `bson:"taxAmount"`
}

func (r saleDashboardRow) toDashboard() *SaleDashboard {
	d := &SaleDashboard{ByPaymentMethod: r.ByPaymentMethod, ByChannel: r.ByChannel, TopProducts: r.TopProducts, DailyRevenue: r.DailyRevenue}
	if len(r.Summary) == 0 {
		return d
	}
	s := r.Summary[0]
	d.TotalSales, d.DraftSales, d.ConfirmedSales = s.TotalSales, s.DraftSales, s.ConfirmedSales
	d.PaidSales, d.CancelledSales, d.RefundedSales = s.PaidSales, s.CancelledSales, s.RefundedSales
	d.GrossRevenue, d.Discounts, d.TaxAmount = s.GrossRevenue, s.Discounts, s.TaxAmount
	if d.PaidSales > 0 {
		d.AverageOrderValue = d.GrossRevenue / float64(d.PaidSales)
	}
	return d
}
