package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/adapolytech/stock-manager/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const companiesCollection = "companies"

type companyRepository struct {
	collection *mongo.Collection
}

func NewCompanyRepository(database *mongo.Database) CompanyRepository {
	return &companyRepository{
		collection: database.Collection(companiesCollection),
	}
}

func (r *companyRepository) Create(ctx context.Context, company *domain.Company) (*domain.Company, error) {
	if company == nil {
		return nil, fmt.Errorf("company.Create: company is required")
	}

	now := time.Now().UTC()
	if company.ID.IsZero() {
		company.ID = bson.NewObjectID()
	}
	if company.CreatedAt.IsZero() {
		company.CreatedAt = now
	}
	company.UpdatedAt = now

	if _, err := r.collection.InsertOne(ctx, company); err != nil {
		return nil, fmt.Errorf("company.Create: insert company: %w", err)
	}

	return company, nil
}

func (r *companyRepository) GetByID(ctx context.Context, id bson.ObjectID) (*domain.Company, error) {
	var company domain.Company
	if err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&company); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("company.GetByID: company not found: %w", err)
		}
		return nil, fmt.Errorf("company.GetByID: find company: %w", err)
	}

	return &company, nil
}

func (r *companyRepository) DeleteByID(ctx context.Context, id bson.ObjectID) (*domain.Company, error) {
	var company domain.Company
	if err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&company); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("company.DeleteById: company not found: %w", err)
		}
		return nil, fmt.Errorf("company.DeleteById: find company: %w", err)
	}
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return nil, fmt.Errorf("Cannot delete company %s: err: %w", company.Name, err)
	}

	return &company, nil
}

func (r *companyRepository) GetBySlug(ctx context.Context, slug string) (*domain.Company, error) {
	var company domain.Company
	if err := r.collection.FindOne(ctx, bson.M{"slug": slug}).Decode(&company); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("company.GetBySlug: company not found: %w", err)
		}
		return nil, fmt.Errorf("company.GetBySlug: find company: %w", err)
	}

	return &company, nil
}

func (r *companyRepository) Update(ctx context.Context, id bson.ObjectID, request *domain.UpdateCompanyRequest) (*domain.Company, error) {
	if request == nil {
		return nil, fmt.Errorf("company.Update: update request is required")
	}

	updates := bson.M{}
	if request.Name != nil {
		updates["name"] = *request.Name
	}
	if request.Address != nil {
		updates["address"] = *request.Address
	}
	if request.Contact != nil {
		updates["contact"] = *request.Contact
	}
	if request.Billing != nil {
		updates["billing"] = *request.Billing
	}
	if request.Settings != nil {
		updates["settings"] = *request.Settings
	}
	if request.Plan != nil {
		updates["plan"] = *request.Plan
	}
	if request.IsActive != nil {
		updates["is_active"] = *request.IsActive
	}
	if len(updates) == 0 {
		return r.GetByID(ctx, id)
	}

	updates["updated_at"] = time.Now().UTC()
	var company domain.Company
	err := r.collection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": updates},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&company)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("company.Update: company not found: %w", err)
		}
		return nil, fmt.Errorf("company.Update: update company: %w", err)
	}

	return &company, nil
}

func (r *companyRepository) IncrementInvoiceCounter(ctx context.Context, id bson.ObjectID) (int, error) {
	var company domain.Company
	err := r.collection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": id},
		bson.M{"$inc": bson.M{"settings.invoice_counter": 1}, "$set": bson.M{"updated_at": time.Now().UTC()}},
		options.FindOneAndUpdate().SetReturnDocument(options.After).SetProjection(bson.M{"settings.invoice_counter": 1}),
	).Decode(&company)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return 0, fmt.Errorf("company.IncrementInvoiceCounter: company not found: %w", err)
		}
		return 0, fmt.Errorf("company.IncrementInvoiceCounter: increment counter: %w", err)
	}

	return company.Settings.InvoiceCounter, nil
}
