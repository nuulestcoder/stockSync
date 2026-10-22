package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/adapolytech/stock-manager/internal/domain"
	"github.com/adapolytech/stock-manager/internal/repository"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type StockService struct {
	products  repository.ProductRepository
	stocks    repository.StockRepository
	movements repository.StockMovementRepository
	requests  repository.StockRequestRepository
}

type StockMoveResult struct {
	Stock    *domain.Stock
	Movement *domain.StockMovement
}

func NewStockService(productRepository repository.ProductRepository, stockRepository repository.StockRepository, movementRepository repository.StockMovementRepository, requestRepository repository.StockRequestRepository) *StockService {
	return &StockService{products: productRepository, stocks: stockRepository, movements: movementRepository, requests: requestRepository}
}

func (s *StockService) List(ctx context.Context, companyID bson.ObjectID, filter domain.StockFilter) (*repository.StockListResult, error) {
	return s.stocks.List(ctx, companyID, filter)
}

func (s *StockService) Get(ctx context.Context, companyID, variantID bson.ObjectID) (*repository.StockView, error) {
	return s.stocks.GetViewByVariant(ctx, companyID, variantID)
}

func (s *StockService) Alerts(ctx context.Context, companyID bson.ObjectID, page, limit int) (*repository.StockListResult, error) {
	below := true
	return s.stocks.List(ctx, companyID, domain.StockFilter{BelowLevel: &below, Page: page, Limit: limit})
}

func (s *StockService) Dashboard(ctx context.Context, companyID bson.ObjectID) (*repository.StockDashboard, error) {
	return s.stocks.GetDashboard(ctx, companyID)
}

func (s *StockService) Movements(ctx context.Context, companyID, variantID bson.ObjectID, page, limit int) (*repository.StockMovementListResult, error) {
	return s.movements.ListByVariant(ctx, companyID, variantID, page, limit)
}

// Move applies one immediate inventory movement. Serial-number input is
// intentionally ignored until serialized inventory is implemented.
func (s *StockService) Move(ctx context.Context, companyID, performedBy, variantID bson.ObjectID, request *domain.MoveStockRequest) (*StockMoveResult, error) {
	if request == nil || !request.Type.IsValid() || request.Quantity <= 0 || strings.TrimSpace(request.Reason) == "" {
		return nil, fmt.Errorf("stock.Move: valid type, positive quantity, and reason are required")
	}
	product, err := s.products.GetByVariant(ctx, companyID, variantID)
	if err != nil || !containsVariant(product, variantID) {
		return nil, fmt.Errorf("stock.Move: product variant does not belong to this company")
	}
	stock, err := s.stocks.GetByVariant(ctx, companyID, variantID)
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		return nil, fmt.Errorf("stock.Move: get stock: %w", err)
	}

	quantityBefore, quantityAfter, reserved := 0, request.Quantity, 0
	if stock != nil {
		quantityBefore, reserved = stock.Quantity, stock.Reserved
		quantityAfter, err = movementQuantities(request.Type, request.Quantity, quantityBefore, stock.Available)
		if err != nil {
			return nil, fmt.Errorf("stock.Move: %w", err)
		}
		if quantityAfter < reserved {
			return nil, fmt.Errorf("stock.Move: adjustment cannot reduce quantity below reserved stock")
		}
		stock, err = s.stocks.Update(ctx, companyID, variantID, bson.M{
			"quantity": quantityAfter, "available": quantityAfter - reserved, "last_movement": time.Now().UTC(),
		})
		if err != nil {
			return nil, fmt.Errorf("stock.Move: update stock: %w", err)
		}
	} else {
		if request.Type == domain.StockMovementOut {
			return nil, fmt.Errorf("stock.Move: insufficient available stock")
		}
		stock, err = s.stocks.Create(ctx, companyID, &domain.Stock{
			CompanyID: companyID, ProductID: product.ID, VariantID: variantID,
			Quantity: quantityAfter, Available: quantityAfter, LastMovement: time.Now().UTC(),
		})
		if err != nil {
			return nil, fmt.Errorf("stock.Move: create stock: %w", err)
		}
	}

	movement, err := s.movements.Create(ctx, companyID, &domain.StockMovement{
		CompanyID: companyID, ProductID: product.ID, VariantID: variantID, Type: request.Type,
		Quantity: abs(quantityAfter - quantityBefore), QuantityBefore: quantityBefore,
		QuantityAfter: quantityAfter, Reason: strings.TrimSpace(request.Reason),
		Reference: strings.TrimSpace(request.Reference), PerformedBy: performedBy, CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		return nil, fmt.Errorf("stock.Move: record movement: %w", err)
	}
	return &StockMoveResult{Stock: stock, Movement: movement}, nil
}

func (s *StockService) CreateRequest(ctx context.Context, companyID, requestedBy bson.ObjectID, requestType domain.StockRequestType, request *domain.CreateStockRequest) (*domain.StockRequest, error) {
	if request == nil || request.Type != requestType || !requestType.IsValid() || len(request.Items) == 0 || strings.TrimSpace(request.Reason) == "" {
		return nil, fmt.Errorf("stock.CreateRequest: type, items, and reason are required")
	}
	for _, item := range request.Items {
		product, err := s.products.GetByVariant(ctx, companyID, item.VariantID)
		if item.Quantity <= 0 || err != nil || product.ID != item.ProductID || !containsVariant(product, item.VariantID) {
			return nil, fmt.Errorf("stock.CreateRequest: invalid product variant or quantity")
		}
	}
	return s.requests.Create(ctx, companyID, &domain.StockRequest{
		CompanyID: companyID, Type: requestType, Status: domain.StockRequestPending,
		Items: request.Items, Reason: strings.TrimSpace(request.Reason),
		Reference: strings.TrimSpace(request.Reference), RequestedBy: requestedBy,
	})
}

func (s *StockService) ListRequests(ctx context.Context, companyID bson.ObjectID, filter repository.StockRequestFilter) (*repository.StockRequestListResult, error) {
	return s.requests.List(ctx, companyID, filter)
}

func (s *StockService) transition(ctx context.Context, companyID, requestID, actorID bson.ObjectID, target domain.StockRequestStatus) (*domain.StockRequest, error) {
	request, err := s.requests.GetByID(ctx, companyID, requestID)
	if err != nil {
		return nil, err
	}
	if request.Status == domain.StockRequestCompleted || request.Status == domain.StockRequestCancelled || request.Status == domain.StockRequestRejected {
		return nil, fmt.Errorf("stock.Request: request is already terminal")
	}
	switch target {
	case domain.StockRequestApproved:
		if request.Status != domain.StockRequestPending {
			return nil, fmt.Errorf("stock.Request: only pending requests can be approved")
		}
		return s.requests.Update(ctx, companyID, requestID, bson.M{"status": target, "approved_by": actorID, "approved_at": time.Now().UTC()})
	case domain.StockRequestRejected:
		if request.Status != domain.StockRequestPending {
			return nil, fmt.Errorf("stock.Request: only pending requests can be rejected")
		}
		return s.requests.Update(ctx, companyID, requestID, bson.M{"status": target})
	case domain.StockRequestCancelled:
		return s.requests.Update(ctx, companyID, requestID, bson.M{"status": target})
	case domain.StockRequestCompleted:
		if request.Status != domain.StockRequestApproved && request.Status != domain.StockRequestProcessing {
			return nil, fmt.Errorf("stock.Request: only approved requests can be completed")
		}
		if request.Status == domain.StockRequestApproved {
			request, err = s.requests.Update(ctx, companyID, requestID, bson.M{"status": domain.StockRequestProcessing})
			if err != nil {
				return nil, fmt.Errorf("stock.Request: start processing: %w", err)
			}
		}
		movementType := domain.StockMovementIn
		if request.Type == domain.StockRequestOutbound {
			movementType = domain.StockMovementOut
		}
		for _, item := range request.Items {
			if _, err := s.Move(ctx, companyID, actorID, item.VariantID, &domain.MoveStockRequest{
				Type: movementType, Quantity: item.Quantity, Reason: request.Reason, Reference: request.ID.Hex(),
			}); err != nil {
				_, _ = s.requests.Update(ctx, companyID, requestID, bson.M{"status": domain.StockRequestApproved})
				return nil, fmt.Errorf("stock.Request: apply movement: %w", err)
			}
		}
		return s.requests.Update(ctx, companyID, requestID, bson.M{
			"status": target, "completed_by": actorID, "completed_at": time.Now().UTC(),
		})
	default:
		return nil, fmt.Errorf("stock.Request: unsupported transition")
	}
}

func (s *StockService) ApproveRequest(ctx context.Context, companyID, requestID, actorID bson.ObjectID) (*domain.StockRequest, error) {
	return s.transition(ctx, companyID, requestID, actorID, domain.StockRequestApproved)
}
func (s *StockService) RejectRequest(ctx context.Context, companyID, requestID, actorID bson.ObjectID) (*domain.StockRequest, error) {
	return s.transition(ctx, companyID, requestID, actorID, domain.StockRequestRejected)
}
func (s *StockService) CancelRequest(ctx context.Context, companyID, requestID, actorID bson.ObjectID) (*domain.StockRequest, error) {
	return s.transition(ctx, companyID, requestID, actorID, domain.StockRequestCancelled)
}
func (s *StockService) CompleteRequest(ctx context.Context, companyID, requestID, actorID bson.ObjectID) (*domain.StockRequest, error) {
	return s.transition(ctx, companyID, requestID, actorID, domain.StockRequestCompleted)
}

func movementQuantities(movementType domain.StockMovementType, quantity, current, available int) (int, error) {
	switch movementType {
	case domain.StockMovementIn, domain.StockMovementReturn:
		return current + quantity, nil
	case domain.StockMovementOut:
		if available < quantity {
			return 0, fmt.Errorf("insufficient available stock: requested %d, available %d", quantity, available)
		}
		return current - quantity, nil
	case domain.StockMovementAdjustment:
		return quantity, nil
	default:
		return 0, fmt.Errorf("unsupported movement type %q", movementType)
	}
}

func containsVariant(product *domain.Product, variantID bson.ObjectID) bool {
	if product == nil {
		return false
	}
	for _, variant := range product.Variants {
		if variant.ID == variantID {
			return true
		}
	}
	return false
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
