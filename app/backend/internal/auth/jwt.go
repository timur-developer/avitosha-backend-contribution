package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
)

const refreshTokenEntropyBytes = 32

var ErrInvalidAccessToken = errors.New("invalid access token")

type JWTTokenProviderConfig struct {
	SigningKey []byte
	Issuer     string
	Audience   string
}

type JWTTokenProvider struct {
	signingKey []byte
	issuer     string
	audience   string
}

type accessTokenClaims struct {
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

func NewJWTTokenProvider(cfg JWTTokenProviderConfig) (*JWTTokenProvider, error) {
	switch {
	case len(cfg.SigningKey) == 0:
		return nil, fmt.Errorf("signing key is required")
	case cfg.Issuer == "":
		return nil, fmt.Errorf("issuer is required")
	case cfg.Audience == "":
		return nil, fmt.Errorf("audience is required")
	}

	signingKey := append([]byte(nil), cfg.SigningKey...)

	return &JWTTokenProvider{
		signingKey: signingKey,
		issuer:     cfg.Issuer,
		audience:   cfg.Audience,
	}, nil
}

func (p *JWTTokenProvider) CreateAccessToken(userID, sessionID uuid.UUID, issuedAt, expiresAt time.Time) (string, error) {
	if userID == uuid.Nil {
		return "", fmt.Errorf("user id is required")
	}
	if sessionID == uuid.Nil {
		return "", fmt.Errorf("session id is required")
	}
	if !expiresAt.After(issuedAt) {
		return "", fmt.Errorf("expires at must be after issued at")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, accessTokenClaims{
		SessionID: sessionID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			Issuer:    p.issuer,
			Audience:  jwt.ClaimStrings{p.audience},
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	})

	signedToken, err := token.SignedString(p.signingKey)
	if err != nil {
		return "", fmt.Errorf("sign jwt access token: %w", err)
	}

	return signedToken, nil
}

func (p *JWTTokenProvider) CreateRefreshToken() (string, error) {
	randomBytes := make([]byte, refreshTokenEntropyBytes)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("generate refresh token entropy: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(randomBytes), nil
}

func (p *JWTTokenProvider) HashRefreshToken(token string) []byte {
	hashSum := sha256.Sum256([]byte(token))
	hashedToken := make([]byte, sha256.Size)
	copy(hashedToken, hashSum[:])
	return hashedToken
}

func (p *JWTTokenProvider) VerifyAccessToken(token string) (model.AuthenticatedUser, error) {
	if strings.TrimSpace(token) == "" {
		return model.AuthenticatedUser{}, ErrInvalidAccessToken
	}

	var claims accessTokenClaims
	parsedToken, err := jwt.ParseWithClaims(token, &claims, func(parsedToken *jwt.Token) (any, error) {
		return p.signingKey, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(p.issuer),
		jwt.WithAudience(p.audience),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return model.AuthenticatedUser{}, errors.Join(ErrInvalidAccessToken, fmt.Errorf("parse jwt access token: %w", err))
	}
	if !parsedToken.Valid {
		return model.AuthenticatedUser{}, ErrInvalidAccessToken
	}
	if claims.IssuedAt == nil || claims.ExpiresAt == nil {
		return model.AuthenticatedUser{}, ErrInvalidAccessToken
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return model.AuthenticatedUser{}, errors.Join(ErrInvalidAccessToken, fmt.Errorf("parse subject uuid: %w", err))
	}

	sessionID, err := uuid.Parse(claims.SessionID)
	if err != nil {
		return model.AuthenticatedUser{}, errors.Join(ErrInvalidAccessToken, fmt.Errorf("parse session id uuid: %w", err))
	}

	return model.AuthenticatedUser{
		UserID:    userID,
		SessionID: sessionID,
	}, nil
}
