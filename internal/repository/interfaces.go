package repository

import (
	"context"

	"github.com/adapolytech/stock-manager/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type CompanyRepository interface {
	Create(ctx context.Context, company *domain.Company) (*domain.Company, error)
	GetByID(ctx context.Context, id bson.ObjectID) (*domain.Company, error)
	DeleteByID(ctx context.Context, id bson.ObjectID) (*domain.Company, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Company, error)
	Update(ctx context.Context, id bson.ObjectID, request *domain.UpdateCompanyRequest) (*domain.Company, error)
	IncrementInvoiceCounter(ctx context.Context, id bson.ObjectID) (int, error)
}

type UserRepository interface {
	Create(ctx context.Context, companyID bson.ObjectID, user *domain.User) (*domain.User, error)
	GetByID(ctx context.Context, companyID, id bson.ObjectID) (*domain.User, error)
	GetByEmail(ctx context.Context, companyID bson.ObjectID, email string) (*domain.User, error)
	GetByEmailForLogin(ctx context.Context, email string) (*domain.User, error)
	List(ctx context.Context, companyID bson.ObjectID) ([]*domain.User, error)
	Update(ctx context.Context, companyID, id bson.ObjectID, request *domain.UpdateUserRequest) (*domain.User, error)
	SoftDelete(ctx context.Context, companyID, id bson.ObjectID) error
}

type ProductRepository interface {
	Create(ctx context.Context, companyID bson.ObjectID, product *domain.Product) (*domain.Product, error)
	GetByID(ctx context.Context, companyID, id bson.ObjectID) (*domain.Product, error)
	List(ctx context.Context, companyID bson.ObjectID, filter ProductFilter) (*ProductListResult, error)
	Update(ctx context.Context, companyID, id bson.ObjectID, updates bson.M) (*domain.Product, error)
	SoftDelete(ctx context.Context, companyID, id bson.ObjectID) error
	ReplaceVariants(ctx context.Context, companyID, id bson.ObjectID, variants []domain.Variant) (*domain.Product, error)
	SKUExists(ctx context.Context, companyID bson.ObjectID, sku string, excludeProductID *bson.ObjectID) (bool, error)
}

type ProductFilter struct {
	Page     int
	Limit    int
	Search   string
	Brand    string
	Category string
}

type ProductListResult struct {
	Products   []*domain.Product
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}
