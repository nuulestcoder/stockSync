package jwt

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/adapolytech/stock-manager/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	tokenTypeAccess  = "access"
	tokenTypeRefresh = "refresh"
	algorithm        = "HS256"
)

var (
	ErrInvalidToken = errors.New("jwt: invalid token")
	ErrExpiredToken = errors.New("jwt: token expired")
)

type Claims struct {
	UserID    bson.ObjectID   `json:"-"`
	CompanyID bson.ObjectID   `json:"-"`
	Role      domain.UserRole `json:"role"`
	Type      string          `json:"type"`
	IssuedAt  time.Time       `json:"-"`
	ExpiresAt time.Time       `json:"-"`
}

type payload struct {
	UserID    string          `json:"sub"`
	CompanyID string          `json:"companyId"`
	Role      domain.UserRole `json:"role"`
	Type      string          `json:"type"`
	IssuedAt  int64           `json:"iat"`
	ExpiresAt int64           `json:"exp"`
}

type header struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
}

func GenerateAccessToken(userID, companyID bson.ObjectID, role domain.UserRole, secret string, ttl time.Duration) (string, error) {
	return generate(userID, companyID, role, tokenTypeAccess, secret, ttl)
}

func GenerateRefreshToken(userID, companyID bson.ObjectID, role domain.UserRole, secret string, ttl time.Duration) (string, error) {
	return generate(userID, companyID, role, tokenTypeRefresh, secret, ttl)
}

func ParseAccessToken(token, secret string) (*Claims, error) {
	return parse(token, secret, tokenTypeAccess)
}

func ParseRefreshToken(token, secret string) (*Claims, error) {
	return parse(token, secret, tokenTypeRefresh)
}

func generate(userID, companyID bson.ObjectID, role domain.UserRole, tokenKind, secret string, ttl time.Duration) (string, error) {
	if userID.IsZero() || companyID.IsZero() {
		return "", fmt.Errorf("jwt.generate: user and company IDs are required")
	}
	if !role.IsValid() {
		return "", fmt.Errorf("jwt.generate: invalid user role")
	}
	if secret == "" {
		return "", fmt.Errorf("jwt.generate: signing secret is required")
	}
	if ttl <= 0 {
		return "", fmt.Errorf("jwt.generate: token TTL must be positive")
	}

	now := time.Now().UTC()
	data := payload{
		UserID:    userID.Hex(),
		CompanyID: companyID.Hex(),
		Role:      role,
		Type:      tokenKind,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(ttl).Unix(),
	}
	return sign(header{Algorithm: algorithm, Type: "JWT"}, data, secret)
}

func parse(token, secret, expectedType string) (*Claims, error) {
	if token == "" || secret == "" {
		return nil, ErrInvalidToken
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	var tokenHeader header
	var tokenPayload payload
	if err := decodePart(parts[0], &tokenHeader); err != nil {
		return nil, ErrInvalidToken
	}
	if err := decodePart(parts[1], &tokenPayload); err != nil {
		return nil, ErrInvalidToken
	}
	if tokenHeader.Algorithm != algorithm || tokenHeader.Type != "JWT" || tokenPayload.Type != expectedType {
		return nil, ErrInvalidToken
	}

	expectedSignature := signPart(parts[0]+"."+parts[1], secret)
	actualSignature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(actualSignature, expectedSignature) {
		return nil, ErrInvalidToken
	}

	userID, err := bson.ObjectIDFromHex(tokenPayload.UserID)
	if err != nil {
		return nil, ErrInvalidToken
	}
	companyID, err := bson.ObjectIDFromHex(tokenPayload.CompanyID)
	if err != nil {
		return nil, ErrInvalidToken
	}
	if !tokenPayload.Role.IsValid() || tokenPayload.IssuedAt <= 0 || tokenPayload.ExpiresAt <= 0 {
		return nil, ErrInvalidToken
	}

	expiresAt := time.Unix(tokenPayload.ExpiresAt, 0)
	if !time.Now().Before(expiresAt) {
		return nil, ErrExpiredToken
	}

	return &Claims{
		UserID:    userID,
		CompanyID: companyID,
		Role:      tokenPayload.Role,
		Type:      tokenPayload.Type,
		IssuedAt:  time.Unix(tokenPayload.IssuedAt, 0),
		ExpiresAt: expiresAt,
	}, nil
}

func sign(tokenHeader header, tokenPayload payload, secret string) (string, error) {
	headerJSON, err := json.Marshal(tokenHeader)
	if err != nil {
		return "", fmt.Errorf("jwt.sign: encode header: %w", err)
	}
	payloadJSON, err := json.Marshal(tokenPayload)
	if err != nil {
		return "", fmt.Errorf("jwt.sign: encode payload: %w", err)
	}

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signingInput := encodedHeader + "." + encodedPayload
	signature := signPart(signingInput, secret)
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func signPart(signingInput, secret string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(signingInput))
	return mac.Sum(nil)
}

func decodePart(encoded string, target any) error {
	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return err
	}
	return json.Unmarshal(decoded, target)
}

func NewClaims(userID, companyID bson.ObjectID, role domain.UserRole, tokenType string, issuedAt, expiresAt time.Time) Claims {
	return Claims{
		UserID:    userID,
		CompanyID: companyID,
		Role:      role,
		Type:      tokenType,
		IssuedAt:  issuedAt,
		ExpiresAt: expiresAt,
	}
}

func GenerateRandomID() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("jwt.GenerateRandomID: generate ID: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}
