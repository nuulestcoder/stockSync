package service

import (
	"context"
	"fmt"
	"net/mail"
	"strings"

	"github.com/adapolytech/stock-manager/internal/domain"
	"github.com/adapolytech/stock-manager/internal/repository"
	"github.com/adapolytech/stock-manager/pkg/password"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type AuthService struct {
	companyRepository repository.CompanyRepository
	userRepository    repository.UserRepository
	mongoClient       *mongo.Client
	mailFrom          string
}

func NewAuthService(
	companyRepository repository.CompanyRepository,
	userRepository repository.UserRepository,
	mongoClient *mongo.Client,
) *AuthService {
	return &AuthService{
		companyRepository: companyRepository,
		userRepository:    userRepository,
		mongoClient:       mongoClient,
	}
}

func (s *AuthService) Register(ctx context.Context, request *domain.RegisterRequest) (*domain.User, error) {
	if request == nil {
		return nil, fmt.Errorf("auth.Register: request is required")
	}
	if _, err := mail.ParseAddress(request.Email); err != nil {
		return nil, fmt.Errorf("auth.Register: invalid email: %w", err)
	}

	hashedPassword, err := password.Hash(request.Password)
	if err != nil {
		return nil, fmt.Errorf("auth.Register: hash password: %w", err)
	}

	company := &domain.Company{
		Name:     strings.TrimSpace(request.Company.Name),
		Slug:     slugify(request.Company.Name),
		Address:  request.Company.Address,
		Contact:  request.Company.Contact,
		Billing:  request.Company.Billing,
		Settings: request.Company.Settings,
		Plan:     request.Company.Plan,
		IsActive: true,
	}
	company.Contact.Email = strings.ToLower(strings.TrimSpace(request.Email))
	if company.Settings.Currency == "" {
		company.Settings.Currency = "XOF"
	}
	if company.Settings.InvoicePrefix == "" {
		company.Settings.InvoicePrefix = "NZO"
	}

	user := &domain.User{
		Name:      strings.TrimSpace(request.Name),
		Email:     strings.ToLower(strings.TrimSpace(request.Email)),
		Phone:     strings.TrimSpace(request.Phone),
		Password:  hashedPassword,
		Role:      domain.UserRoleAdmin,
		IsActive:  true,
		IsDeleted: false,
	}

	// session, err := s.mongoClient.StartSession()
	// if err != nil {
	// 	return nil, fmt.Errorf("auth.Register: start transaction session: %w", err)
	// }
	// defer session.EndSession(ctx)

	// _, err = session.WithTransaction(ctx, func(transactionContext context.Context) (any, error) {
	createdCompany, err := s.companyRepository.Create(ctx, company)
	if err != nil {
		return nil, fmt.Errorf("auth.Register: create company: %w", err)
	}

	createdUser, err := s.userRepository.Create(ctx, createdCompany.ID, user)
	if err != nil {
		// Delete created company
		s.companyRepository.DeleteByID(ctx, company.ID)
		return nil, fmt.Errorf("auth.Register: create admin user: %w", err)
	}

	return createdUser, nil
	// })
	// if err != nil {
	// 	return nil, fmt.Errorf("auth.Register: transaction: %w", err)
	// }

	// add support of sendind notification email later
	// if s.mailClient == nil {
	// 	return nil, fmt.Errorf("auth.Register: email client is not configured")
	// }
	// if _, err := s.mailClient.Send(ctx, resend.VerificationMessage(s.mailFrom, user.Email, company.Name)); err != nil {
	// 	return nil, fmt.Errorf("auth.Register: send verification email: %w", err)
	// }

	// return user, nil
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastWasDash := false
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') {
			builder.WriteRune(char)
			lastWasDash = false
			continue
		}
		if !lastWasDash && builder.Len() > 0 {
			builder.WriteByte('-')
			lastWasDash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}
