package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// UserRole identifies the permissions assigned to a user.
type UserRole string

const (
	UserRoleAdmin   UserRole = "ADMIN"
	UserRoleManager UserRole = "MANAGER"
	UserRoleSeller  UserRole = "SELLER"
)

// IsValid reports whether the role is supported by the application.
func (r UserRole) IsValid() bool {
	switch r {
	case UserRoleAdmin, UserRoleManager, UserRoleSeller:
		return true
	default:
		return false
	}
}

// User represents an application user belonging to a company tenant.
type User struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	CompanyID bson.ObjectID `bson:"company_id"    json:"companyId"`
	Name      string        `bson:"name"          json:"name"`
	Email     string        `bson:"email"         json:"email"`
	Phone     string        `bson:"phone"         json:"phone"`
	Password  string        `bson:"password"      json:"-"`
	Role      UserRole      `bson:"role"          json:"role"`
	IsActive  bool          `bson:"is_active"     json:"isActive"`
	IsDeleted bool          `bson:"is_deleted"     json:"isDeleted"`
	CreatedAt time.Time     `bson:"created_at"    json:"createdAt"`
	UpdatedAt time.Time     `bson:"updated_at"    json:"updatedAt"`
}

// CreateUserRequest is the DTO for creating a user within a company.
type CreateUserRequest struct {
	Name     string   `json:"name"     binding:"required"`
	Email    string   `json:"email"    binding:"required,email"`
	Phone    string   `json:"phone"`
	Password string   `json:"password" binding:"required,min=8"`
	Role     UserRole `json:"role"     binding:"required,oneof=ADMIN MANAGER SELLER"`
}

// UpdateUserRequest is the DTO for updating a user.
type UpdateUserRequest struct {
	Name     *string   `json:"name"`
	Email    *string   `json:"email"`
	Phone    *string   `json:"phone"`
	Password *string   `json:"password" binding:"omitempty,min=8"`
	Role     *UserRole `json:"role"     binding:"omitempty,oneof=ADMIN MANAGER SELLER"`
	IsActive *bool     `json:"isActive"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

type LoginResponse struct {
	AccessToken  string      `json:"accessToken"`
	RefreshToken string      `json:"refreshToken"`
	User         UserSummary `json:"user"`
}

type UserSummary struct {
	ID        bson.ObjectID `json:"id"`
	CompanyID bson.ObjectID `json:"companyId"`
	Name      string        `json:"name"`
	Email     string        `json:"email"`
	Role      UserRole      `json:"role"`
}
