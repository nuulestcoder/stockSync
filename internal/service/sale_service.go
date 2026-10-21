package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/adapolytech/stock-manager/internal/domain"
	"github.com/adapolytech/stock-manager/internal/repository"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type SaleService struct {
	sales     repository.SaleRepository
	products  repository.ProductRepository
	companies repository.CompanyRepository
	stocks    repository.StockRepository
	movements repository.StockMovementRepository
}

func NewSaleService(saleRepository repository.SaleRepository, productRepository repository.ProductRepository, companyRepository repository.CompanyRepository, stockRepository repository.StockRepository, movementRepository repository.StockMovementRepository) *SaleService {
	return &SaleService{sales: saleRepository, products: productRepository, companies: companyRepository, stocks: stockRepository, movements: movementRepository}
}

func (s *SaleService) Create(ctx context.Context, companyID, createdBy bson.ObjectID, request *domain.CreateSaleRequest) (*domain.Sale, error) {
	if request == nil || len(request.Items) == 0 {
		return nil, fmt.Errorf("sale.Create: at least one item is required")
	}
	if request.Channel == "" {
		request.Channel = domain.SaleChannelInStore
	}
	if !request.Channel.IsValid() {
		return nil, fmt.Errorf("sale.Create: invalid sales channel")
	}
	if request.Payment.Method == "" {
		request.Payment.Method = domain.PaymentMethodCash
	}
	if !request.Payment.Method.IsValid() {
		return nil, fmt.Errorf("sale.Create: invalid payment method")
	}
	if request.Payment.Status == "" {
		request.Payment.Status = domain.PaymentStatusPending
	}
	if !request.Payment.Status.IsValid() {
		return nil, fmt.Errorf("sale.Create: invalid payment status")
	}
	if request.Discount < 0 {
		return nil, fmt.Errorf("sale.Create: discount cannot be negative")
	}

	items := make([]domain.SaleItem, 0, len(request.Items))
	var subtotal float64
	for index, itemRequest := range request.Items {
		if itemRequest.ProductID.IsZero() || itemRequest.VariantID.IsZero() || itemRequest.Quantity <= 0 {
			return nil, fmt.Errorf("sale.Create: invalid item at index %d", index)
		}
		if itemRequest.Discount < 0 {
			return nil, fmt.Errorf("sale.Create: item discount cannot be negative")
		}

		product, err := s.products.GetByVariant(ctx, companyID, itemRequest.VariantID)
		if err != nil {
			return nil, fmt.Errorf("sale.Create: load item %d: %w", index, err)
		}
		if product.ID != itemRequest.ProductID {
			return nil, fmt.Errorf("sale.Create: product and variant mismatch at index %d", index)
		}
		variant, found := findVariant(product, itemRequest.VariantID)
		if !found {
			return nil, fmt.Errorf("sale.Create: variant not found at index %d", index)
		}

		lineSubtotal := variant.Price.Selling * float64(itemRequest.Quantity)
		if itemRequest.Discount > lineSubtotal {
			return nil, fmt.Errorf("sale.Create: item discount exceeds line subtotal at index %d", index)
		}
		lineTotal := roundMoney(lineSubtotal - itemRequest.Discount)
		subtotal += lineSubtotal
		items = append(items, domain.SaleItem{
			ProductID: product.ID, VariantID: variant.ID, ProductName: product.Name,
			SKU: variant.SKU, Specs: variant.Specs, Quantity: itemRequest.Quantity,
			UnitPrice: variant.Price.Selling, Discount: roundMoney(itemRequest.Discount), Total: lineTotal,
		})
	}

	subtotal = roundMoney(subtotal)
	if request.Discount > subtotal {
		return nil, fmt.Errorf("sale.Create: discount exceeds subtotal")
	}
	netSubtotal := subtotal - request.Discount
	company, err := s.companies.GetByID(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("sale.Create: load company settings: %w", err)
	}
	taxRate := company.Settings.TaxRate
	if taxRate < 0 {
		return nil, fmt.Errorf("sale.Create: company tax rate cannot be negative")
	}
	taxAmount := roundMoney(netSubtotal * taxRate / 100)
	total := roundMoney(netSubtotal + taxAmount)
	counter, err := s.companies.IncrementInvoiceCounter(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("sale.Create: generate invoice number: %w", err)
	}
	prefix := strings.ToUpper(strings.TrimSpace(company.Settings.InvoicePrefix))
	if prefix == "" {
		prefix = "NZO"
	}

	sale := &domain.Sale{
		CompanyID: companyID, InvoiceNumber: fmt.Sprintf("%s-%d-%03d", prefix, time.Now().UTC().Year(), counter),
		Status: domain.SaleStatusDraft, Customer: normalizeSaleCustomer(request.Customer), Items: items,
		Subtotal: subtotal, Discount: roundMoney(request.Discount), TaxAmount: taxAmount, Total: total,
		Currency: "XOF", Channel: request.Channel, Notes: strings.TrimSpace(request.Notes), CreatedBy: createdBy,
		Payment: domain.SalePayment{
			Method: request.Payment.Method, Status: request.Payment.Status,
			Reference: strings.TrimSpace(request.Payment.Reference),
		},
	}
	created, err := s.sales.Create(ctx, companyID, sale)
	if err != nil {
		return nil, fmt.Errorf("sale.Create: persist sale: %w", err)
	}
	return created, nil
}

func (s *SaleService) GetByID(ctx context.Context, companyID, saleID bson.ObjectID) (*domain.Sale, error) {
	sale, err := s.sales.GetByID(ctx, companyID, saleID)
	if err != nil {
		return nil, fmt.Errorf("sale.GetByID: %w", err)
	}
	return sale, nil
}

func (s *SaleService) List(ctx context.Context, companyID bson.ObjectID, filter repository.SaleFilter) (*repository.SaleListResult, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	result, err := s.sales.List(ctx, companyID, filter)
	if err != nil {
		return nil, fmt.Errorf("sale.List: %w", err)
	}
	return result, nil
}

func (s *SaleService) Dashboard(ctx context.Context, companyID bson.ObjectID, filter repository.SaleReportFilter) (*repository.SaleDashboard, error) {
	result, err := s.sales.GetDashboard(ctx, companyID, filter)
	if err != nil {
		return nil, fmt.Errorf("sale.Dashboard: %w", err)
	}
	return result, nil
}

func (s *SaleService) Transition(ctx context.Context, companyID, saleID, actorID bson.ObjectID, target domain.SaleStatus) (*domain.Sale, error) {
	sale, err := s.sales.GetByID(ctx, companyID, saleID)
	if err != nil {
		return nil, fmt.Errorf("sale.Transition: get sale: %w", err)
	}
	if !target.IsValid() {
		return nil, fmt.Errorf("sale.Transition: invalid target status")
	}
	switch target {
	case domain.SaleStatusConfirmed:
		if sale.Status != domain.SaleStatusDraft {
			return nil, fmt.Errorf("sale.Transition: only draft sales can be confirmed")
		}
		if err := s.reserveItems(ctx, companyID, sale); err != nil {
			return nil, fmt.Errorf("sale.Transition: reserve stock: %w", err)
		}
	case domain.SaleStatusPaid:
		if sale.Status != domain.SaleStatusConfirmed {
			return nil, fmt.Errorf("sale.Transition: only confirmed sales can be marked paid")
		}
		if err := s.consumeReservedItems(ctx, companyID, actorID, sale); err != nil {
			return nil, fmt.Errorf("sale.Transition: consume stock: %w", err)
		}
	case domain.SaleStatusCancelled:
		if sale.Status != domain.SaleStatusDraft && sale.Status != domain.SaleStatusConfirmed {
			return nil, fmt.Errorf("sale.Transition: sale cannot be cancelled from %s", sale.Status)
		}
		if sale.Status == domain.SaleStatusConfirmed {
			if err := s.releaseItems(ctx, companyID, sale); err != nil {
				return nil, fmt.Errorf("sale.Transition: release stock: %w", err)
			}
		}
	case domain.SaleStatusRefunded:
		if sale.Status != domain.SaleStatusPaid {
			return nil, fmt.Errorf("sale.Transition: only paid sales can be refunded")
		}
		if err := s.returnItems(ctx, companyID, actorID, sale); err != nil {
			return nil, fmt.Errorf("sale.Transition: return stock: %w", err)
		}
	case domain.SaleStatusDraft:
		return nil, fmt.Errorf("sale.Transition: reverting to draft is not supported")
	}
	updated, err := s.sales.UpdateStatus(ctx, companyID, saleID, target)
	if err != nil {
		return nil, fmt.Errorf("sale.Transition: update status: %w", err)
	}
	return updated, nil
}

func (s *SaleService) reserveItems(ctx context.Context, companyID bson.ObjectID, sale *domain.Sale) error {
	var reserved []domain.SaleItem
	for index := range sale.Items {
		item := &sale.Items[index]
		stock, err := s.stocks.GetByVariant(ctx, companyID, item.VariantID)
		if err != nil {
			return fmt.Errorf("item %s: stock not found: %w", item.SKU, err)
		}
		if stock.Available < item.Quantity {
			return fmt.Errorf("item %s: requested %d, available %d", item.SKU, item.Quantity, stock.Available)
		}
		if _, err := s.stocks.Update(ctx, companyID, item.VariantID, bson.M{
			"reserved": stock.Reserved + item.Quantity, "available": stock.Available - item.Quantity,
		}); err != nil {
			if rollbackErr := s.releaseReservedItems(ctx, companyID, sale, reserved); rollbackErr != nil {
				return fmt.Errorf("item %s: reserve stock: %w; rollback failed: %v", item.SKU, err, rollbackErr)
			}
			return fmt.Errorf("item %s: reserve stock: %w", item.SKU, err)
		}
		reserved = append(reserved, *item)
	}
	return nil
}

func (s *SaleService) releaseItems(ctx context.Context, companyID bson.ObjectID, sale *domain.Sale) error {
	return s.releaseReservedItems(ctx, companyID, sale, sale.Items)
}

func (s *SaleService) releaseReservedItems(ctx context.Context, companyID bson.ObjectID, sale *domain.Sale, items []domain.SaleItem) error {
	for _, item := range items {
		stock, err := s.stocks.GetByVariant(ctx, companyID, item.VariantID)
		if err != nil {
			return fmt.Errorf("item %s: get stock: %w", item.SKU, err)
		}
		if stock.Reserved < item.Quantity {
			return fmt.Errorf("item %s: reserved quantity is insufficient", item.SKU)
		}
		if _, err := s.stocks.Update(ctx, companyID, item.VariantID, bson.M{
			"reserved": stock.Reserved - item.Quantity, "available": stock.Available + item.Quantity,
		}); err != nil {
			return fmt.Errorf("item %s: release stock: %w", item.SKU, err)
		}
	}
	return nil
}

func (s *SaleService) consumeReservedItems(ctx context.Context, companyID, actorID bson.ObjectID, sale *domain.Sale) error {
	for _, item := range sale.Items {
		stock, err := s.stocks.GetByVariant(ctx, companyID, item.VariantID)
		if err != nil || stock.Reserved < item.Quantity || stock.Quantity < item.Quantity {
			return fmt.Errorf("item %s: reserved stock is insufficient", item.SKU)
		}
		updated, err := s.stocks.Update(ctx, companyID, item.VariantID, bson.M{
			"quantity": stock.Quantity - item.Quantity, "reserved": stock.Reserved - item.Quantity,
			"available": stock.Available, "last_movement": time.Now().UTC(),
		})
		if err != nil {
			return fmt.Errorf("item %s: consume stock: %w", item.SKU, err)
		}
		if _, err := s.movements.Create(ctx, companyID, &domain.StockMovement{
			CompanyID: companyID, ProductID: item.ProductID, VariantID: item.VariantID,
			Type: domain.StockMovementOut, Quantity: item.Quantity, QuantityBefore: stock.Quantity,
			QuantityAfter: updated.Quantity, Reason: "sale paid", Reference: sale.ID.Hex(),
			PerformedBy: actorID, CreatedAt: time.Now().UTC(),
		}); err != nil {
			return fmt.Errorf("item %s: record movement: %w", item.SKU, err)
		}
	}
	return nil
}

func (s *SaleService) returnItems(ctx context.Context, companyID, actorID bson.ObjectID, sale *domain.Sale) error {
	for _, item := range sale.Items {
		stock, err := s.stocks.GetByVariant(ctx, companyID, item.VariantID)
		if err != nil {
			return fmt.Errorf("item %s: get stock: %w", item.SKU, err)
		}
		updated, err := s.stocks.Update(ctx, companyID, item.VariantID, bson.M{
			"quantity": stock.Quantity + item.Quantity, "available": stock.Available + item.Quantity,
			"last_movement": time.Now().UTC(),
		})
		if err != nil {
			return fmt.Errorf("item %s: return stock: %w", item.SKU, err)
		}
		if _, err := s.movements.Create(ctx, companyID, &domain.StockMovement{
			CompanyID: companyID, ProductID: item.ProductID, VariantID: item.VariantID,
			Type: domain.StockMovementReturn, Quantity: item.Quantity, QuantityBefore: stock.Quantity,
			QuantityAfter: updated.Quantity, Reason: "sale refunded", Reference: sale.ID.Hex(),
			PerformedBy: actorID, CreatedAt: time.Now().UTC(),
		}); err != nil {
			return fmt.Errorf("item %s: record movement: %w", item.SKU, err)
		}
	}
	return nil
}

func findVariant(product *domain.Product, variantID bson.ObjectID) (domain.Variant, bool) {
	for _, variant := range product.Variants {
		if variant.ID == variantID {
			return variant, true
		}
	}
	return domain.Variant{}, false
}

func normalizeSaleCustomer(customer domain.SaleCustomer) domain.SaleCustomer {
	customer.Name = strings.TrimSpace(customer.Name)
	customer.Phone = strings.TrimSpace(customer.Phone)
	customer.Email = strings.ToLower(strings.TrimSpace(customer.Email))
	return customer
}

func roundMoney(value float64) float64 {
	return float64(int64(value*100+0.5)) / 100
}
