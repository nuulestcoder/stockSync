package repository

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/adapolytech/stock-manager/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const productsCollection = "products"

type productRepository struct {
	collection *mongo.Collection
}

func NewProductRepository(database *mongo.Database) ProductRepository {
	return &productRepository{collection: database.Collection(productsCollection)}
}

func EnsureProductIndexes(ctx context.Context, database *mongo.Database) error {
	collection := database.Collection(productsCollection)
	_, err := collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "company_id", Value: 1}, {Key: "created_at", Value: -1}}},
		{Keys: bson.D{{Key: "company_id", Value: 1}, {Key: "brand", Value: 1}, {Key: "category", Value: 1}}},
		{Keys: bson.D{{Key: "company_id", Value: 1}, {Key: "tags", Value: 1}}},
		{
			Keys:    bson.D{{Key: "company_id", Value: 1}, {Key: "variants.sku", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	})
	if err != nil {
		return fmt.Errorf("product.EnsureIndexes: create indexes: %w", err)
	}
	return nil
}

func (r *productRepository) Create(ctx context.Context, companyID bson.ObjectID, product *domain.Product) (*domain.Product, error) {
	if product == nil {
		return nil, fmt.Errorf("product.Create: product is required")
	}

	product.CompanyID = companyID
	if product.ID.IsZero() {
		product.ID = bson.NewObjectID()
	}
	now := time.Now().UTC()
	if product.CreatedAt.IsZero() {
		product.CreatedAt = now
	}
	product.UpdatedAt = now

	if _, err := r.collection.InsertOne(ctx, product); err != nil {
		return nil, fmt.Errorf("product.Create: insert product: %w", err)
	}
	return product, nil
}

func (r *productRepository) GetByID(ctx context.Context, companyID, id bson.ObjectID) (*domain.Product, error) {
	var product domain.Product
	if err := r.collection.FindOne(ctx, productFilter(companyID, id)).Decode(&product); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("product.GetByID: product not found: %w", err)
		}
		return nil, fmt.Errorf("product.GetByID: find product: %w", err)
	}
	return &product, nil
}

func (r *productRepository) GetByVariant(ctx context.Context, companyID, variantID bson.ObjectID) (*domain.Product, error) {
	var product domain.Product
	query := bson.M{
		"company_id":   companyID,
		"_id":          bson.M{"$exists": true},
		"is_deleted":   bson.M{"$ne": true},
		"variants._id": variantID,
	}
	if err := r.collection.FindOne(ctx, query).Decode(&product); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("product.GetByVariant: product not found: %w", err)
		}
		return nil, fmt.Errorf("product.GetByVariant: find product: %w", err)
	}
	return &product, nil
}

func (r *productRepository) List(ctx context.Context, companyID bson.ObjectID, filter ProductFilter) (*ProductListResult, error) {
	query := bson.M{"company_id": companyID, "is_deleted": bson.M{"$ne": true}}
	conditions := bson.A{}
	if filter.Search != "" {
		search := regexp.QuoteMeta(strings.TrimSpace(filter.Search))
		conditions = append(conditions, bson.M{
			"$or": bson.A{
				bson.M{"name": bson.M{"$regex": search, "$options": "i"}},
				bson.M{"brand": bson.M{"$regex": search, "$options": "i"}},
				bson.M{"category": bson.M{"$regex": search, "$options": "i"}},
				bson.M{"variants.sku": bson.M{"$regex": search, "$options": "i"}},
			},
		})
	}
	if filter.Brand != "" {
		conditions = append(conditions, bson.M{"brand": filter.Brand})
	}
	if filter.Category != "" {
		conditions = append(conditions, bson.M{"category": filter.Category})
	}
	if len(conditions) > 0 {
		query["$and"] = conditions
	}

	total, err := r.collection.CountDocuments(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("product.List: count products: %w", err)
	}

	cursor, err := r.collection.Find(ctx, query, options.Find().
		SetSkip(int64((filter.Page-1)*filter.Limit)).
		SetLimit(int64(filter.Limit)).
		SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("product.List: find products: %w", err)
	}
	defer cursor.Close(ctx)

	var products []*domain.Product
	if err := cursor.All(ctx, &products); err != nil {
		return nil, fmt.Errorf("product.List: decode products: %w", err)
	}

	return &ProductListResult{
		Products:   products,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: int(math.Ceil(float64(total) / float64(filter.Limit))),
	}, nil
}

func (r *productRepository) Update(ctx context.Context, companyID, id bson.ObjectID, updates bson.M) (*domain.Product, error) {
	updates["updated_at"] = time.Now().UTC()
	var product domain.Product
	if err := r.collection.FindOneAndUpdate(
		ctx,
		productFilter(companyID, id),
		bson.M{"$set": updates},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&product); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("product.Update: product not found: %w", err)
		}
		return nil, fmt.Errorf("product.Update: update product: %w", err)
	}
	return &product, nil
}

func (r *productRepository) SoftDelete(ctx context.Context, companyID, id bson.ObjectID) error {
	result, err := r.collection.UpdateOne(ctx, productFilter(companyID, id), bson.M{
		"$set": bson.M{"is_deleted": true, "updated_at": time.Now().UTC()},
	})
	if err != nil {
		return fmt.Errorf("product.SoftDelete: delete product: %w", err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("product.SoftDelete: product not found")
	}
	return nil
}

func (r *productRepository) ReplaceVariants(ctx context.Context, companyID, id bson.ObjectID, variants []domain.Variant) (*domain.Product, error) {
	return r.Update(ctx, companyID, id, bson.M{"variants": variants})
}

func (r *productRepository) SKUExists(ctx context.Context, companyID bson.ObjectID, sku string, excludeProductID *bson.ObjectID) (bool, error) {
	query := bson.M{
		"company_id":   companyID,
		"is_deleted":   bson.M{"$ne": true},
		"variants.sku": strings.ToUpper(strings.TrimSpace(sku)),
	}
	if excludeProductID != nil {
		query["_id"] = bson.M{"$ne": *excludeProductID}
	}
	count, err := r.collection.CountDocuments(ctx, query, options.Count().SetLimit(1))
	if err != nil {
		return false, fmt.Errorf("product.SKUExists: check SKU: %w", err)
	}
	return count > 0, nil
}

func productFilter(companyID, id bson.ObjectID) bson.M {
	return bson.M{"company_id": companyID, "_id": id, "is_deleted": bson.M{"$ne": true}}
}
