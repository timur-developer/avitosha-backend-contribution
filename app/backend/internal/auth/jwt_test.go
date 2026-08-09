package auth

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestJWTTokenProviderCreateAccessTokenClaims(t *testing.T) {
	provider, err := NewJWTTokenProvider(JWTTokenProviderConfig{
		SigningKey: []byte("test-signing-key"),
		Issuer:     "avitosha",
		Audience:   "avitosha-web",
	})
	if err != nil {
		t.Fatalf("NewJWTTokenProvider() error = %v", err)
	}

	userID := uuid.MustParse("8f0ed065-aefa-4f56-87d0-e2ef2ef43f0d")
	sessionID := uuid.MustParse("8f0ed065-aefa-4f56-87d0-e2ef2ef43f0e")
	issuedAt := time.Date(2026, time.August, 4, 12, 0, 0, 0, time.UTC)
	expiresAt := issuedAt.Add(15 * time.Minute)

	accessToken, err := provider.CreateAccessToken(userID, sessionID, issuedAt, expiresAt)
	if err != nil {
		t.Fatalf("CreateAccessToken() error = %v", err)
	}

	parsedToken, err := jwt.Parse(accessToken, func(token *jwt.Token) (any, error) {
		return []byte("test-signing-key"), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithTimeFunc(func() time.Time {
		return issuedAt
	}))
	if err != nil {
		t.Fatalf("jwt.Parse() error = %v", err)
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatalf("claims type = %T, want jwt.MapClaims", parsedToken.Claims)
	}

	expectedKeys := map[string]struct{}{
		"sub": {},
		"sid": {},
		"iat": {},
		"exp": {},
		"iss": {},
		"aud": {},
	}
	for key := range claims {
		if _, ok := expectedKeys[key]; !ok {
			t.Fatalf("unexpected JWT claim key %q", key)
		}
	}

	if claims["sub"] != userID.String() {
		t.Fatalf("sub claim = %v, want %s", claims["sub"], userID)
	}
	if claims["sid"] != sessionID.String() {
		t.Fatalf("sid claim = %v, want %s", claims["sid"], sessionID)
	}
	if claims["iss"] != "avitosha" {
		t.Fatalf("iss claim = %v, want avitosha", claims["iss"])
	}

	audience, ok := claims["aud"].([]any)
	if !ok || len(audience) != 1 || audience[0] != "avitosha-web" {
		t.Fatalf("aud claim = %#v, want [avitosha-web]", claims["aud"])
	}

	if _, exists := claims["email"]; exists {
		t.Fatal("JWT unexpectedly contains email claim")
	}
}

func TestJWTTokenProviderCreateRefreshTokenReturnsOpaqueValue(t *testing.T) {
	provider, err := NewJWTTokenProvider(JWTTokenProviderConfig{
		SigningKey: []byte("test-signing-key"),
		Issuer:     "avitosha",
		Audience:   "avitosha-web",
	})
	if err != nil {
		t.Fatalf("NewJWTTokenProvider() error = %v", err)
	}

	refreshToken, err := provider.CreateRefreshToken()
	if err != nil {
		t.Fatalf("CreateRefreshToken() error = %v", err)
	}

	if refreshToken == "" {
		t.Fatal("CreateRefreshToken() returned empty token")
	}
	if strings.Contains(refreshToken, ".") {
		t.Fatalf("refresh token = %q, want opaque token without JWT separators", refreshToken)
	}

	refreshTokenHash := provider.HashRefreshToken(refreshToken)
	if len(refreshTokenHash) != 32 {
		t.Fatalf("HashRefreshToken() length = %d, want 32", len(refreshTokenHash))
	}
	if string(refreshTokenHash) == refreshToken {
		t.Fatal("HashRefreshToken() returned raw refresh token")
	}
}

func TestJWTTokenProviderVerifyAccessToken(t *testing.T) {
	provider, err := NewJWTTokenProvider(JWTTokenProviderConfig{
		SigningKey: []byte("test-signing-key"),
		Issuer:     "avitosha",
		Audience:   "avitosha-web",
	})
	if err != nil {
		t.Fatalf("NewJWTTokenProvider() error = %v", err)
	}

	userID := uuid.MustParse("8f0ed065-aefa-4f56-87d0-e2ef2ef43f0d")
	sessionID := uuid.MustParse("8f0ed065-aefa-4f56-87d0-e2ef2ef43f0e")
	now := time.Now().UTC()

	accessToken, err := provider.CreateAccessToken(userID, sessionID, now.Add(-time.Minute), now.Add(time.Minute))
	if err != nil {
		t.Fatalf("CreateAccessToken() error = %v", err)
	}

	authenticatedUser, err := provider.VerifyAccessToken(accessToken)
	if err != nil {
		t.Fatalf("VerifyAccessToken() error = %v", err)
	}
	if authenticatedUser.UserID != userID {
		t.Fatalf("UserID = %s, want %s", authenticatedUser.UserID, userID)
	}
	if authenticatedUser.SessionID != sessionID {
		t.Fatalf("SessionID = %s, want %s", authenticatedUser.SessionID, sessionID)
	}
}

func TestJWTTokenProviderVerifyAccessTokenRejectsExpiredToken(t *testing.T) {
	provider, err := NewJWTTokenProvider(JWTTokenProviderConfig{
		SigningKey: []byte("test-signing-key"),
		Issuer:     "avitosha",
		Audience:   "avitosha-web",
	})
	if err != nil {
		t.Fatalf("NewJWTTokenProvider() error = %v", err)
	}

	accessToken, err := provider.CreateAccessToken(
		uuid.MustParse("8f0ed065-aefa-4f56-87d0-e2ef2ef43f0d"),
		uuid.MustParse("8f0ed065-aefa-4f56-87d0-e2ef2ef43f0e"),
		time.Now().UTC().Add(-2*time.Minute),
		time.Now().UTC().Add(-time.Minute),
	)
	if err != nil {
		t.Fatalf("CreateAccessToken() error = %v", err)
	}

	_, err = provider.VerifyAccessToken(accessToken)
	if !errors.Is(err, ErrInvalidAccessToken) {
		t.Fatalf("VerifyAccessToken() error = %v, want ErrInvalidAccessToken", err)
	}
}
