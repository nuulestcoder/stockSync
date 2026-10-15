package repository

import (
	"context"

	"github.com/adapolytech/stock-manager/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type CompanyRepository interface {
	Create(ctx context.Context, company *domain.Company) (*domain.Company, error)
	GetByID(ctx context.Context, id bson.ObjectID) (*domain.Company, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Company, error)
	Update(ctx context.Context, id bson.ObjectID, request *domain.UpdateCompanyRequest) (*domain.Company, error)
	IncrementInvoiceCounter(ctx context.Context, id bson.ObjectID) (int, error)
}

type UserRepository interface {
	Create(ctx context.Context, companyID bson.ObjectID, user *domain.User) (*domain.User, error)
	GetByID(ctx context.Context, companyID, id bson.ObjectID) (*domain.User, error)
	GetByEmail(ctx context.Context, companyID bson.ObjectID, email string) (*domain.User, error)
	List(ctx context.Context, companyID bson.ObjectID) ([]*domain.User, error)
	Update(ctx context.Context, companyID, id bson.ObjectID, request *domain.UpdateUserRequest) (*domain.User, error)
	Delete(ctx context.Context, companyID, id bson.ObjectID) error
}
