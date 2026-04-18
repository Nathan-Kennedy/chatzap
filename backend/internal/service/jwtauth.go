package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"wa-saas/backend/internal/config"
)

type AccessClaims struct {
	UserID      string `json:"uid"`
	WorkspaceID string `json:"wid"`
	Role        string `json:"role"`
	Email       string `json:"email"`
	Name        string `json:"name"`
	jwt.RegisteredClaims
}

// IssueAccessToken emite JWT de acesso HS256.
func IssueAccessToken(cfg *config.Config, userID, workspaceID uuid.UUID, role, email, name string) (string, error) {
	now := time.Now()
	exp := now.Add(time.Duration(cfg.JWTAccessTTLMinutes) * time.Minute)
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, AccessClaims{
		UserID:      userID.String(),
		WorkspaceID: workspaceID.String(),
		Role:        role,
		Email:       email,
		Name:        name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.NewString(),
		},
	})
	return t.SignedString([]byte(cfg.JWTSecret))
}

// ParseAccessToken valida JWT de acesso.
func ParseAccessToken(cfg *config.Config, tokenStr string) (*AccessClaims, error) {
	tok, err := jwt.ParseWithClaims(tokenStr, &AccessClaims{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(cfg.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := tok.Claims.(*AccessClaims)
	if !ok || !tok.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

// NewRefreshToken gera token opaco e o respetivo hash SHA256 (hex).
func NewRefreshToken() (raw string, hashHex string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	raw = base64.RawURLEncoding.EncodeToString(b)
	h := sha256.Sum256([]byte(raw))
	return raw, hex.EncodeToString(h[:]), nil
}

func HashRefreshToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
