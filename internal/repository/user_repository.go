package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/adapolytech/stock-manager/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const usersCollection = "users"

type userRepository struct {
	collection *mongo.Collection
}

func NewUserRepository(database *mongo.Database) UserRepository {
	return &userRepository{
		collection: database.Collection(usersCollection),
	}
}

func (r *userRepository) Create(ctx context.Context, companyID bson.ObjectID, user *domain.User) (*domain.User, error) {
	if user == nil {
		return nil, fmt.Errorf("user.Create: user is required")
	}

	now := time.Now().UTC()
	user.CompanyID = companyID
	user.Email = strings.ToLower(strings.TrimSpace(user.Email))
	if user.ID.IsZero() {
		user.ID = bson.NewObjectID()
	}
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	user.UpdatedAt = now

	if _, err := r.collection.InsertOne(ctx, user); err != nil {
		return nil, fmt.Errorf("user.Create: insert user: %w", err)
	}

	return user, nil
}

func (r *userRepository) GetByID(ctx context.Context, companyID, id bson.ObjectID) (*domain.User, error) {
	var user domain.User
	filter := activeUserFilter(companyID)
	filter["_id"] = id

	if err := r.collection.FindOne(ctx, filter).Decode(&user); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("user.GetByID: user not found: %w", err)
		}
		return nil, fmt.Errorf("user.GetByID: find user: %w", err)
	}

	return &user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, companyID bson.ObjectID, email string) (*domain.User, error) {
	var user domain.User
	filter := activeUserFilter(companyID)
	filter["email"] = strings.ToLower(strings.TrimSpace(email))

	if err := r.collection.FindOne(ctx, filter).Decode(&user); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("user.GetByEmail: user not found: %w", err)
		}
		return nil, fmt.Errorf("user.GetByEmail: find user: %w", err)
	}

	return &user, nil
}

func (r *userRepository) GetByEmailForLogin(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	filter := bson.M{
		"email":      strings.ToLower(strings.TrimSpace(email)),
		"is_deleted": bson.M{"$ne": true},
		"is_active":  true,
	}

	if err := r.collection.FindOne(ctx, filter).Decode(&user); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("user.GetByEmailForLogin: user not found: %w", err)
		}
		return nil, fmt.Errorf("user.GetByEmailForLogin: find user: %w", err)
	}

	return &user, nil
}

func (r *userRepository) List(ctx context.Context, companyID bson.ObjectID) ([]*domain.User, error) {
	cursor, err := r.collection.Find(ctx, activeUserFilter(companyID), options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("user.List: find users: %w", err)
	}
	defer cursor.Close(ctx)

	var users []*domain.User
	if err := cursor.All(ctx, &users); err != nil {
		return nil, fmt.Errorf("user.List: decode users: %w", err)
	}

	return users, nil
}

func (r *userRepository) Update(ctx context.Context, companyID, id bson.ObjectID, request *domain.UpdateUserRequest) (*domain.User, error) {
	if request == nil {
		return nil, fmt.Errorf("user.Update: update request is required")
	}

	updates := bson.M{}
	if request.Name != nil {
		updates["name"] = *request.Name
	}
	if request.Email != nil {
		updates["email"] = strings.ToLower(strings.TrimSpace(*request.Email))
	}
	if request.Phone != nil {
		updates["phone"] = *request.Phone
	}
	if request.Password != nil {
		updates["password"] = *request.Password
	}
	if request.Role != nil {
		updates["role"] = *request.Role
	}
	if request.IsActive != nil {
		updates["is_active"] = *request.IsActive
	}
	if len(updates) == 0 {
		return r.GetByID(ctx, companyID, id)
	}

	updates["updated_at"] = time.Now().UTC()
	var user domain.User
	filter := activeUserFilter(companyID)
	filter["_id"] = id

	err := r.collection.FindOneAndUpdate(
		ctx,
		filter,
		bson.M{"$set": updates},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("user.Update: user not found: %w", err)
		}
		return nil, fmt.Errorf("user.Update: update user: %w", err)
	}

	return &user, nil
}

func (r *userRepository) SoftDelete(ctx context.Context, companyID, id bson.ObjectID) error {
	filter := activeUserFilter(companyID)
	filter["_id"] = id

	result, err := r.collection.UpdateOne(
		ctx,
		filter,
		bson.M{"$set": bson.M{"is_deleted": true, "is_active": false, "updated_at": time.Now().UTC()}},
	)
	if err != nil {
		return fmt.Errorf("user.Delete: soft-delete user: %w", err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("user.Delete: user not found")
	}

	return nil
}

func activeUserFilter(companyID bson.ObjectID) bson.M {
	return bson.M{
		"company_id": companyID,
		"is_deleted": bson.M{"$ne": true},
	}
}
