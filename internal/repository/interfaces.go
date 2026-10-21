package repository

import (
	"context"
	"time"

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
	GetByVariant(ctx context.Context, companyID, variantID bson.ObjectID) (*domain.Product, error)
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

type StockRepository interface {
	Create(ctx context.Context, companyID bson.ObjectID, stock *domain.Stock) (*domain.Stock, error)
	GetByVariant(ctx context.Context, companyID, variantID bson.ObjectID) (*domain.Stock, error)
	GetViewByVariant(ctx context.Context, companyID, variantID bson.ObjectID) (*StockView, error)
	List(ctx context.Context, companyID bson.ObjectID, filter domain.StockFilter) (*StockListResult, error)
	ListViews(ctx context.Context, companyID bson.ObjectID, filter domain.StockFilter) (*StockListResult, error)
	Update(ctx context.Context, companyID, variantID bson.ObjectID, updates bson.M) (*domain.Stock, error)
	GetDashboard(ctx context.Context, companyID bson.ObjectID) (*StockDashboard, error)
}

type StockMovementRepository interface {
	Create(ctx context.Context, companyID bson.ObjectID, movement *domain.StockMovement) (*domain.StockMovement, error)
	ListByVariant(ctx context.Context, companyID, variantID bson.ObjectID, page, limit int) (*StockMovementListResult, error)
}

type StockRequestRepository interface {
	Create(ctx context.Context, companyID bson.ObjectID, request *domain.StockRequest) (*domain.StockRequest, error)
	GetByID(ctx context.Context, companyID, id bson.ObjectID) (*domain.StockRequest, error)
	List(ctx context.Context, companyID bson.ObjectID, filter StockRequestFilter) (*StockRequestListResult, error)
	Update(ctx context.Context, companyID, id bson.ObjectID, updates bson.M) (*domain.StockRequest, error)
}

type SaleRepository interface {
	Create(ctx context.Context, companyID bson.ObjectID, sale *domain.Sale) (*domain.Sale, error)
	GetByID(ctx context.Context, companyID, id bson.ObjectID) (*domain.Sale, error)
	List(ctx context.Context, companyID bson.ObjectID, filter SaleFilter) (*SaleListResult, error)
	UpdateStatus(ctx context.Context, companyID, id bson.ObjectID, status domain.SaleStatus) (*domain.Sale, error)
	GetDashboard(ctx context.Context, companyID bson.ObjectID, filter SaleReportFilter) (*SaleDashboard, error)
}

type SaleFilter struct {
	Status        domain.SaleStatus
	PaymentStatus domain.PaymentStatus
	PaymentMethod domain.PaymentMethod
	Page          int
	Limit         int
}

type SaleListResult struct {
	Sales      []*domain.Sale
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type SaleReportFilter struct {
	From *time.Time
	To   *time.Time
}

type SaleDashboard struct {
	TotalSales        int64                 `json:"totalSales" bson:"totalSales"`
	DraftSales        int64                 `json:"draftSales" bson:"draftSales"`
	ConfirmedSales    int64                 `json:"confirmedSales" bson:"confirmedSales"`
	PaidSales         int64                 `json:"paidSales" bson:"paidSales"`
	CancelledSales    int64                 `json:"cancelledSales" bson:"cancelledSales"`
	RefundedSales     int64                 `json:"refundedSales" bson:"refundedSales"`
	GrossRevenue      float64               `json:"grossRevenue" bson:"grossRevenue"`
	Discounts         float64               `json:"discounts" bson:"discounts"`
	TaxAmount         float64               `json:"taxAmount" bson:"taxAmount"`
	AverageOrderValue float64               `json:"averageOrderValue" bson:"averageOrderValue"`
	ByPaymentMethod   []SaleDimensionMetric `json:"byPaymentMethod" bson:"byPaymentMethod"`
	ByChannel         []SaleDimensionMetric `json:"byChannel" bson:"byChannel"`
	TopProducts       []SaleProductMetric   `json:"topProducts" bson:"topProducts"`
	DailyRevenue      []SaleDailyMetric     `json:"dailyRevenue" bson:"dailyRevenue"`
}

type SaleDimensionMetric struct {
	Key    string  `json:"key" bson:"key"`
	Count  int64   `json:"count" bson:"count"`
	Amount float64 `json:"amount" bson:"amount"`
}

type SaleProductMetric struct {
	ProductID bson.ObjectID `json:"productId" bson:"productId"`
	SKU       string        `json:"sku" bson:"sku"`
	Name      string        `json:"name" bson:"name"`
	Quantity  int64         `json:"quantity" bson:"quantity"`
	Revenue   float64       `json:"revenue" bson:"revenue"`
}

type SaleDailyMetric struct {
	Date   string  `json:"date" bson:"date"`
	Count  int64   `json:"count" bson:"count"`
	Amount float64 `json:"amount" bson:"amount"`
}

type StockListResult struct {
	Stocks     []*StockView
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type StockView struct {
	Stock   *domain.Stock    `json:"stock"`
	Product StockProductView `json:"product"`
	Variant domain.Variant   `json:"variant"`
}

type StockProductView struct {
	ID       bson.ObjectID `bson:"id" json:"id"`
	Name     string        `bson:"name" json:"name"`
	Brand    string        `bson:"brand" json:"brand"`
	Category string        `bson:"category" json:"category"`
	Images   []string      `bson:"images" json:"images"`
}

type StockMovementListResult struct {
	Movements  []*domain.StockMovement
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type StockRequestFilter struct {
	Type   domain.StockRequestType
	Status domain.StockRequestStatus
	Page   int
	Limit  int
}

type StockRequestListResult struct {
	Requests   []*domain.StockRequest
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type StockDashboard struct {
	TotalVariants      int64
	TotalQuantity      int64
	TotalReserved      int64
	TotalAvailable     int64
	OutOfStockVariants int64
	LowStockVariants   int64
	InboundQuantity    int64
	OutboundQuantity   int64
	RecentMovements    []*domain.StockMovement
}
