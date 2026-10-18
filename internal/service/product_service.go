package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/adapolytech/stock-manager/internal/domain"
	"github.com/adapolytech/stock-manager/internal/repository"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type ProductService struct {
	repository repository.ProductRepository
}

func NewProductService(productRepository repository.ProductRepository) *ProductService {
	return &ProductService{repository: productRepository}
}

func (s *ProductService) Create(ctx context.Context, companyID bson.ObjectID, request *domain.CreateProductRequest) (*domain.Product, error) {
	if request == nil {
		return nil, fmt.Errorf("product.Create: request is required")
	}
	if len(request.Variants) == 0 {
		return nil, fmt.Errorf("product.Create: at least one variant is required")
	}

	product := &domain.Product{
		Name:        normalizeText(request.Name),
		Brand:       normalizeText(request.Brand),
		Category:    normalizeText(request.Category),
		Description: strings.TrimSpace(request.Description),
		Images:      request.Images,
		Tags:        normalizeTags(request.Tags),
		Variants:    request.Variants,
	}
	if product.Name == "" || product.Brand == "" || product.Category == "" {
		return nil, fmt.Errorf("product.Create: name, brand, and category are required")
	}

	if err := s.normalizeAndValidateVariants(ctx, companyID, product.Variants, nil); err != nil {
		return nil, fmt.Errorf("product.Create: validate variants: %w", err)
	}
	created, err := s.repository.Create(ctx, companyID, product)
	if err != nil {
		return nil, fmt.Errorf("product.Create: create product: %w", err)
	}
	return created, nil
}

func (s *ProductService) GetByID(ctx context.Context, companyID, productID bson.ObjectID) (*domain.Product, error) {
	product, err := s.repository.GetByID(ctx, companyID, productID)
	if err != nil {
		return nil, fmt.Errorf("product.GetByID: %w", err)
	}
	return product, nil
}

func (s *ProductService) List(ctx context.Context, companyID bson.ObjectID, filter repository.ProductFilter) (*repository.ProductListResult, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	filter.Search = strings.TrimSpace(filter.Search)
	filter.Brand = normalizeText(filter.Brand)
	filter.Category = normalizeText(filter.Category)

	products, err := s.repository.List(ctx, companyID, filter)
	if err != nil {
		return nil, fmt.Errorf("product.List: %w", err)
	}
	return products, nil
}

func (s *ProductService) Update(ctx context.Context, companyID, productID bson.ObjectID, request *domain.UpdateProductRequest) (*domain.Product, error) {
	if request == nil {
		return nil, fmt.Errorf("product.Update: request is required")
	}

	updates := bson.M{}
	if request.Name != nil {
		updates["name"] = normalizeText(*request.Name)
	}
	if request.Brand != nil {
		updates["brand"] = normalizeText(*request.Brand)
	}
	if request.Category != nil {
		updates["category"] = normalizeText(*request.Category)
	}
	if request.Description != nil {
		updates["description"] = strings.TrimSpace(*request.Description)
	}
	if request.Images != nil {
		updates["images"] = *request.Images
	}
	if request.Tags != nil {
		updates["tags"] = normalizeTags(*request.Tags)
	}
	if len(updates) == 0 {
		return s.GetByID(ctx, companyID, productID)
	}
	updated, err := s.repository.Update(ctx, companyID, productID, updates)
	if err != nil {
		return nil, fmt.Errorf("product.Update: %w", err)
	}
	return updated, nil
}

func (s *ProductService) Delete(ctx context.Context, companyID, productID bson.ObjectID) error {
	if err := s.repository.SoftDelete(ctx, companyID, productID); err != nil {
		return fmt.Errorf("product.Delete: %w", err)
	}
	return nil
}

func (s *ProductService) AddVariant(ctx context.Context, companyID, productID bson.ObjectID, request *domain.CreateVariantRequest) (*domain.Product, error) {
	if request == nil {
		return nil, fmt.Errorf("product.AddVariant: request is required")
	}
	product, err := s.GetByID(ctx, companyID, productID)
	if err != nil {
		return nil, err
	}

	variant := domain.Variant{
		ID: bson.NewObjectID(), SKU: request.SKU, Specs: request.Specs, Price: request.Price, Images: request.Images,
	}
	variants := append(product.Variants, variant)
	if err := s.normalizeAndValidateVariants(ctx, companyID, []domain.Variant{variant}, nil); err != nil {
		return nil, fmt.Errorf("product.AddVariant: validate variant: %w", err)
	}
	updated, err := s.repository.ReplaceVariants(ctx, companyID, productID, variants)
	if err != nil {
		return nil, fmt.Errorf("product.AddVariant: save variant: %w", err)
	}
	return updated, nil
}

func (s *ProductService) UpdateVariant(ctx context.Context, companyID, productID, variantID bson.ObjectID, request *domain.UpdateVariantRequest) (*domain.Product, error) {
	if request == nil {
		return nil, fmt.Errorf("product.UpdateVariant: request is required")
	}
	product, err := s.GetByID(ctx, companyID, productID)
	if err != nil {
		return nil, err
	}

	found := false
	updatedSKU := ""
	for index := range product.Variants {
		variant := &product.Variants[index]
		if variant.ID != variantID {
			continue
		}
		found = true
		if request.SKU != nil {
			variant.SKU = *request.SKU
			updatedSKU = variant.SKU
		}
		if request.Specs != nil {
			variant.Specs = *request.Specs
		}
		if request.Price != nil {
			variant.Price = *request.Price
		}
		if request.Images != nil {
			variant.Images = *request.Images
		}
	}
	if !found {
		return nil, fmt.Errorf("product.UpdateVariant: variant not found")
	}

	if err := s.normalizeAndValidateVariants(ctx, companyID, product.Variants, &productID); err != nil {
		return nil, fmt.Errorf("product.UpdateVariant: validate variants: %w", err)
	}
	if request.SKU != nil {
		exists, err := s.repository.SKUExists(ctx, companyID, updatedSKU, &productID)
		if err != nil {
			return nil, fmt.Errorf("product.UpdateVariant: check SKU: %w", err)
		}
		if exists {
			return nil, fmt.Errorf("product.UpdateVariant: SKU already exists in this company")
		}
	}
	updated, err := s.repository.ReplaceVariants(ctx, companyID, productID, product.Variants)
	if err != nil {
		return nil, fmt.Errorf("product.UpdateVariant: save variant: %w", err)
	}

	return updated, nil
}

func (s *ProductService) DeleteVariant(ctx context.Context, companyID, productID, variantID bson.ObjectID) (*domain.Product, error) {
	product, err := s.GetByID(ctx, companyID, productID)
	if err != nil {
		return nil, err
	}

	variants := make([]domain.Variant, 0, len(product.Variants))
	found := false
	for _, variant := range product.Variants {
		if variant.ID == variantID {
			found = true
			continue
		}
		variants = append(variants, variant)
	}
	if !found {
		return nil, fmt.Errorf("product.DeleteVariant: variant not found")
	}
	if len(variants) == 0 {
		return nil, fmt.Errorf("product.DeleteVariant: product must have at least one variant")
	}

	updated, err := s.repository.ReplaceVariants(ctx, companyID, productID, variants)
	if err != nil {
		return nil, fmt.Errorf("product.DeleteVariant: save variants: %w", err)
	}
	return updated, nil
}

func (s *ProductService) normalizeAndValidateVariants(ctx context.Context, companyID bson.ObjectID, variants []domain.Variant, excludeProductID *bson.ObjectID) error {
	seen := make(map[string]struct{}, len(variants))
	for index := range variants {
		variant := &variants[index]
		variant.SKU = strings.ToUpper(strings.TrimSpace(variant.SKU))
		variant.Price.Currency = strings.ToUpper(strings.TrimSpace(variant.Price.Currency))
		if variant.Price.Currency == "" {
			variant.Price.Currency = "XOF"
		}
		if variant.ID.IsZero() {
			variant.ID = bson.NewObjectID()
		}
		if variant.SKU == "" {
			return fmt.Errorf("variant %d SKU is required", index)
		}
		if variant.Price.Buying < 0 || variant.Price.Selling < 0 {
			return fmt.Errorf("variant %s prices cannot be negative", variant.SKU)
		}
		if _, exists := seen[variant.SKU]; exists {
			return fmt.Errorf("duplicate SKU %s", variant.SKU)
		}
		seen[variant.SKU] = struct{}{}

		exists, err := s.repository.SKUExists(ctx, companyID, variant.SKU, excludeProductID)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("SKU %s already exists in this company", variant.SKU)
		}
	}
	return nil
}

func normalizeText(value string) string {
	return strings.TrimSpace(value)
}

func normalizeTags(tags []string) []string {
	normalized := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag == "" {
			continue
		}
		if _, exists := seen[tag]; exists {
			continue
		}
		seen[tag] = struct{}{}
		normalized = append(normalized, tag)
	}
	return normalized
}
