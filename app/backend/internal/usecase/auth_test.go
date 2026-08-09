package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
)

func TestAuthServiceRegisterSuccess(t *testing.T) {
	store := newFakeAuthStore()
	userRepository := &fakeUserRepository{store: store}
	sessionRepository := &fakeSessionRepository{store: store}
	txManager := &fakeTxManager{store: store}
	passwordHasher := &fakePasswordHasher{
		hashFn: func(password string) (string, error) {
			return "hashed:" + password, nil
		},
	}
	tokenProvider := &fakeTokenProvider{
		createAccessTokenFn: func(userID, sessionID uuid.UUID, issuedAt, expiresAt time.Time) (string, error) {
			return fmt.Sprintf("access:%s:%s", userID, sessionID), nil
		},
		createRefreshTokenFn: func() (string, error) {
			return "refresh-token", nil
		},
	}
	registrationHook := &fakeRegistrationHook{}

	authService := newTestAuthService(t, AuthDependencies{
		PasswordHasher:    passwordHasher,
		TokenProvider:     tokenProvider,
		UserRepository:    userRepository,
		SessionRepository: sessionRepository,
		TxManager:         txManager,
		RegistrationHook:  registrationHook,
	})

	result, err := authService.Register(context.Background(), RegisterParams{
		Email:     "  User@Example.com  ",
		Password:  "password123",
		UserAgent: stringPointer("test-agent"),
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if result.User.Email != "user@example.com" {
		t.Fatalf("Register() email = %q, want normalized email", result.User.Email)
	}
	if result.AccessToken == "" {
		t.Fatal("Register() returned empty access token")
	}
	if result.RefreshToken != "refresh-token" {
		t.Fatalf("Register() refresh token = %q", result.RefreshToken)
	}
	if !txManager.committed {
		t.Fatal("Register() did not commit transaction")
	}
	if !registrationHook.called {
		t.Fatal("Register() did not invoke registration hook")
	}

	storedUser, ok := store.usersByEmail["user@example.com"]
	if !ok {
		t.Fatal("Register() did not persist user")
	}
	if storedUser.PasswordHash != "hashed:password123" {
		t.Fatalf("stored password hash = %q", storedUser.PasswordHash)
	}

	if len(store.sessionsByID) != 1 {
		t.Fatalf("stored sessions = %d, want 1", len(store.sessionsByID))
	}
	for _, session := range store.sessionsByID {
		if string(session.RefreshTokenHash) == result.RefreshToken {
			t.Fatal("session stored raw refresh token instead of hash")
		}
		if string(session.RefreshTokenHash) != string(tokenProvider.HashRefreshToken("refresh-token")) {
			t.Fatalf("session refresh hash = %x, want sha256 hash", session.RefreshTokenHash)
		}
	}
}

func TestAuthServiceRegisterDuplicateEmail(t *testing.T) {
	store := newFakeAuthStore()
	userRepository := &fakeUserRepository{store: store}
	sessionRepository := &fakeSessionRepository{store: store}
	txManager := &fakeTxManager{store: store}

	existingUser, err := userRepository.Create(context.Background(), CreateUserParams{
		Email:        "user@example.com",
		PasswordHash: "hashed-password",
	})
	if err != nil {
		t.Fatalf("Create() setup error = %v", err)
	}
	if existingUser.ID == uuid.Nil {
		t.Fatal("Create() setup returned empty user ID")
	}

	authService := newTestAuthService(t, AuthDependencies{
		PasswordHasher: &fakePasswordHasher{
			hashFn: func(password string) (string, error) {
				return "hashed:" + password, nil
			},
		},
		TokenProvider: &fakeTokenProvider{
			createAccessTokenFn: func(userID, sessionID uuid.UUID, issuedAt, expiresAt time.Time) (string, error) {
				return "access-token", nil
			},
			createRefreshTokenFn: func() (string, error) {
				return "refresh-token", nil
			},
		},
		UserRepository:    userRepository,
		SessionRepository: sessionRepository,
		TxManager:         txManager,
	})

	_, err = authService.Register(context.Background(), RegisterParams{
		Email:    "USER@example.com",
		Password: "password123",
	})
	if !errors.Is(err, ErrEmailAlreadyExists) {
		t.Fatalf("Register() error = %v, want ErrEmailAlreadyExists", err)
	}
}

func TestAuthServiceRegisterRollsBackWhenSessionCreateFails(t *testing.T) {
	store := newFakeAuthStore()
	txManager := &fakeTxManager{store: store}
	authService := newTestAuthService(t, AuthDependencies{
		PasswordHasher: &fakePasswordHasher{
			hashFn: func(password string) (string, error) {
				return "hashed:" + password, nil
			},
		},
		TokenProvider: &fakeTokenProvider{
			createAccessTokenFn: func(userID, sessionID uuid.UUID, issuedAt, expiresAt time.Time) (string, error) {
				return "access-token", nil
			},
			createRefreshTokenFn: func() (string, error) {
				return "refresh-token", nil
			},
		},
		UserRepository:    &fakeUserRepository{store: store},
		SessionRepository: &fakeSessionRepository{store: store, createErr: ErrUnexpectedStorage},
		TxManager:         txManager,
	})

	_, err := authService.Register(context.Background(), RegisterParams{
		Email:    "user@example.com",
		Password: "password123",
	})
	if !errors.Is(err, ErrInternal) {
		t.Fatalf("Register() error = %v, want ErrInternal", err)
	}
	if !txManager.rolledBack {
		t.Fatal("Register() did not roll back transaction")
	}
	if len(store.usersByEmail) != 0 {
		t.Fatalf("users in store after rollback = %d, want 0", len(store.usersByEmail))
	}
}

func TestAuthServiceRegisterRollsBackWhenRegistrationHookFails(t *testing.T) {
	store := newFakeAuthStore()
	txManager := &fakeTxManager{store: store}
	authService := newTestAuthService(t, AuthDependencies{
		PasswordHasher: &fakePasswordHasher{
			hashFn: func(password string) (string, error) {
				return "hashed:" + password, nil
			},
		},
		TokenProvider: &fakeTokenProvider{
			createAccessTokenFn: func(userID, sessionID uuid.UUID, issuedAt, expiresAt time.Time) (string, error) {
				return "access-token", nil
			},
			createRefreshTokenFn: func() (string, error) {
				return "refresh-token", nil
			},
		},
		UserRepository:    &fakeUserRepository{store: store},
		SessionRepository: &fakeSessionRepository{store: store},
		TxManager:         txManager,
		RegistrationHook:  &fakeRegistrationHook{err: ErrUnexpectedStorage},
	})

	_, err := authService.Register(context.Background(), RegisterParams{
		Email:    "user@example.com",
		Password: "password123",
	})
	if !errors.Is(err, ErrInternal) {
		t.Fatalf("Register() error = %v, want ErrInternal", err)
	}
	if !txManager.rolledBack {
		t.Fatal("Register() did not roll back transaction after registration hook failure")
	}
	if len(store.usersByEmail) != 0 {
		t.Fatalf("users in store after rollback = %d, want 0", len(store.usersByEmail))
	}
	if len(store.sessionsByID) != 0 {
		t.Fatalf("sessions in store after rollback = %d, want 0", len(store.sessionsByID))
	}
}

func TestAuthServiceLoginSuccess(t *testing.T) {
	store := newFakeAuthStore()
	userRepository := &fakeUserRepository{store: store}
	createdUser, err := userRepository.Create(context.Background(), CreateUserParams{
		Email:        "user@example.com",
		PasswordHash: "stored-hash",
	})
	if err != nil {
		t.Fatalf("Create() setup error = %v", err)
	}

	authService := newTestAuthService(t, AuthDependencies{
		PasswordHasher: &fakePasswordHasher{
			matchesFn: func(hash, password string) (bool, error) {
				return hash == "stored-hash" && password == "password123", nil
			},
		},
		TokenProvider: &fakeTokenProvider{
			createAccessTokenFn: func(userID, sessionID uuid.UUID, issuedAt, expiresAt time.Time) (string, error) {
				return "access-token", nil
			},
			createRefreshTokenFn: func() (string, error) {
				return "refresh-token", nil
			},
		},
		UserRepository:    userRepository,
		SessionRepository: &fakeSessionRepository{store: store},
		TxManager:         &fakeTxManager{store: store},
	})

	result, err := authService.Login(context.Background(), LoginParams{
		Email:     " User@example.com ",
		Password:  "password123",
		UserAgent: stringPointer("login-agent"),
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if result.User.ID != createdUser.ID {
		t.Fatalf("Login() user ID = %s, want %s", result.User.ID, createdUser.ID)
	}
	if len(store.sessionsByID) != 1 {
		t.Fatalf("sessions after Login() = %d, want 1", len(store.sessionsByID))
	}
}

func TestAuthServiceLoginRejectsWrongPassword(t *testing.T) {
	store := newFakeAuthStore()
	userRepository := &fakeUserRepository{store: store}
	if _, err := userRepository.Create(context.Background(), CreateUserParams{
		Email:        "user@example.com",
		PasswordHash: "stored-hash",
	}); err != nil {
		t.Fatalf("Create() setup error = %v", err)
	}

	authService := newTestAuthService(t, AuthDependencies{
		PasswordHasher: &fakePasswordHasher{
			matchesFn: func(hash, password string) (bool, error) {
				return false, nil
			},
		},
		TokenProvider: &fakeTokenProvider{
			createAccessTokenFn: func(userID, sessionID uuid.UUID, issuedAt, expiresAt time.Time) (string, error) {
				return "access-token", nil
			},
			createRefreshTokenFn: func() (string, error) {
				return "refresh-token", nil
			},
		},
		UserRepository:    userRepository,
		SessionRepository: &fakeSessionRepository{store: store},
		TxManager:         &fakeTxManager{store: store},
	})

	_, err := authService.Login(context.Background(), LoginParams{
		Email:    "user@example.com",
		Password: "wrong-password",
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want ErrInvalidCredentials", err)
	}
	if len(store.sessionsByID) != 0 {
		t.Fatalf("sessions after failed Login() = %d, want 0", len(store.sessionsByID))
	}
}

func TestAuthServiceLoginRejectsUnknownEmail(t *testing.T) {
	store := newFakeAuthStore()
	authService := newTestAuthService(t, AuthDependencies{
		PasswordHasher: &fakePasswordHasher{
			matchesFn: func(hash, password string) (bool, error) {
				return true, nil
			},
		},
		TokenProvider: &fakeTokenProvider{
			createAccessTokenFn: func(userID, sessionID uuid.UUID, issuedAt, expiresAt time.Time) (string, error) {
				return "access-token", nil
			},
			createRefreshTokenFn: func() (string, error) {
				return "refresh-token", nil
			},
		},
		UserRepository:    &fakeUserRepository{store: store},
		SessionRepository: &fakeSessionRepository{store: store},
		TxManager:         &fakeTxManager{store: store},
	})

	_, err := authService.Login(context.Background(), LoginParams{
		Email:    "missing@example.com",
		Password: "password123",
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want ErrInvalidCredentials", err)
	}
}

func TestAuthServiceRefreshSuccess(t *testing.T) {
	store := newFakeAuthStore()
	userRepository := &fakeUserRepository{store: store}
	sessionRepository := &fakeSessionRepository{store: store}
	user, err := userRepository.Create(context.Background(), CreateUserParams{
		Email:        "user@example.com",
		PasswordHash: "stored-hash",
	})
	if err != nil {
		t.Fatalf("Create() setup error = %v", err)
	}

	now := testNow()
	session, err := sessionRepository.Create(context.Background(), CreateSessionParams{
		UserID:           user.ID,
		RefreshTokenHash: (&fakeTokenProvider{}).HashRefreshToken("old-refresh-token"),
		ExpiresAt:        now.Add(24 * time.Hour),
		LastUsedAt:       now,
	})
	if err != nil {
		t.Fatalf("CreateSession() setup error = %v", err)
	}

	authService := newTestAuthService(t, AuthDependencies{
		PasswordHasher: &fakePasswordHasher{},
		TokenProvider: &fakeTokenProvider{
			createAccessTokenFn: func(userID, sessionID uuid.UUID, issuedAt, expiresAt time.Time) (string, error) {
				if sessionID != session.ID {
					t.Fatalf("CreateAccessToken() session ID = %s, want %s", sessionID, session.ID)
				}
				return "new-access-token", nil
			},
			createRefreshTokenFn: func() (string, error) {
				return "new-refresh-token", nil
			},
		},
		UserRepository:    userRepository,
		SessionRepository: sessionRepository,
		TxManager:         &fakeTxManager{store: store},
		Now:               func() time.Time { return now },
	})

	result, err := authService.Refresh(context.Background(), RefreshParams{RefreshToken: "old-refresh-token"})
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if result.AccessToken != "new-access-token" {
		t.Fatalf("Refresh() access token = %q", result.AccessToken)
	}
	if result.RefreshToken != "new-refresh-token" {
		t.Fatalf("Refresh() refresh token = %q", result.RefreshToken)
	}

	_, err = sessionRepository.GetActiveByRefreshTokenHash(context.Background(), (&fakeTokenProvider{}).HashRefreshToken("old-refresh-token"), now)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("old refresh token lookup error = %v, want ErrSessionNotFound", err)
	}

	rotatedSession, err := sessionRepository.GetActiveByRefreshTokenHash(context.Background(), (&fakeTokenProvider{}).HashRefreshToken("new-refresh-token"), now)
	if err != nil {
		t.Fatalf("GetActiveByRefreshTokenHash(new) error = %v", err)
	}
	if rotatedSession.ID != session.ID {
		t.Fatalf("rotated session ID = %s, want %s", rotatedSession.ID, session.ID)
	}
}

func TestAuthServiceRefreshRejectsExpiredSession(t *testing.T) {
	store := newFakeAuthStore()
	userRepository := &fakeUserRepository{store: store}
	sessionRepository := &fakeSessionRepository{store: store}
	user, err := userRepository.Create(context.Background(), CreateUserParams{
		Email:        "user@example.com",
		PasswordHash: "stored-hash",
	})
	if err != nil {
		t.Fatalf("Create() setup error = %v", err)
	}

	now := testNow()
	if _, err := sessionRepository.Create(context.Background(), CreateSessionParams{
		UserID:           user.ID,
		RefreshTokenHash: (&fakeTokenProvider{}).HashRefreshToken("expired-token"),
		ExpiresAt:        now.Add(-time.Minute),
		LastUsedAt:       now.Add(-2 * time.Minute),
	}); err != nil {
		t.Fatalf("CreateSession() setup error = %v", err)
	}

	authService := newTestAuthService(t, AuthDependencies{
		PasswordHasher:    &fakePasswordHasher{},
		TokenProvider:     &fakeTokenProvider{},
		UserRepository:    userRepository,
		SessionRepository: sessionRepository,
		TxManager:         &fakeTxManager{store: store},
		Now:               func() time.Time { return now },
	})

	_, err = authService.Refresh(context.Background(), RefreshParams{RefreshToken: "expired-token"})
	if !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("Refresh() error = %v, want ErrSessionExpired", err)
	}
}

func TestAuthServiceRefreshRejectsRevokedSession(t *testing.T) {
	store := newFakeAuthStore()
	userRepository := &fakeUserRepository{store: store}
	sessionRepository := &fakeSessionRepository{store: store}
	user, err := userRepository.Create(context.Background(), CreateUserParams{
		Email:        "user@example.com",
		PasswordHash: "stored-hash",
	})
	if err != nil {
		t.Fatalf("Create() setup error = %v", err)
	}

	now := testNow()
	session, err := sessionRepository.Create(context.Background(), CreateSessionParams{
		UserID:           user.ID,
		RefreshTokenHash: (&fakeTokenProvider{}).HashRefreshToken("revoked-token"),
		ExpiresAt:        now.Add(24 * time.Hour),
		LastUsedAt:       now,
	})
	if err != nil {
		t.Fatalf("CreateSession() setup error = %v", err)
	}

	if err := sessionRepository.Revoke(context.Background(), session.ID, now.Add(time.Minute)); err != nil {
		t.Fatalf("Revoke() setup error = %v", err)
	}

	authService := newTestAuthService(t, AuthDependencies{
		PasswordHasher:    &fakePasswordHasher{},
		TokenProvider:     &fakeTokenProvider{},
		UserRepository:    userRepository,
		SessionRepository: sessionRepository,
		TxManager:         &fakeTxManager{store: store},
		Now:               func() time.Time { return now },
	})

	_, err = authService.Refresh(context.Background(), RefreshParams{RefreshToken: "revoked-token"})
	if !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("Refresh() error = %v, want ErrSessionExpired", err)
	}
}

func TestAuthServiceRefreshRejectsReusedToken(t *testing.T) {
	store := newFakeAuthStore()
	userRepository := &fakeUserRepository{store: store}
	sessionRepository := &fakeSessionRepository{store: store}
	user, err := userRepository.Create(context.Background(), CreateUserParams{
		Email:        "user@example.com",
		PasswordHash: "stored-hash",
	})
	if err != nil {
		t.Fatalf("Create() setup error = %v", err)
	}

	now := testNow()
	if _, err := sessionRepository.Create(context.Background(), CreateSessionParams{
		UserID:           user.ID,
		RefreshTokenHash: (&fakeTokenProvider{}).HashRefreshToken("old-refresh-token"),
		ExpiresAt:        now.Add(24 * time.Hour),
		LastUsedAt:       now,
	}); err != nil {
		t.Fatalf("CreateSession() setup error = %v", err)
	}

	tokenProvider := &fakeTokenProvider{
		createAccessTokenFn: func(userID, sessionID uuid.UUID, issuedAt, expiresAt time.Time) (string, error) {
			return "new-access-token", nil
		},
		createRefreshTokenFn: func() (string, error) {
			return "new-refresh-token", nil
		},
	}
	authService := newTestAuthService(t, AuthDependencies{
		PasswordHasher:    &fakePasswordHasher{},
		TokenProvider:     tokenProvider,
		UserRepository:    userRepository,
		SessionRepository: sessionRepository,
		TxManager:         &fakeTxManager{store: store},
		Now:               func() time.Time { return now },
	})

	if _, err := authService.Refresh(context.Background(), RefreshParams{RefreshToken: "old-refresh-token"}); err != nil {
		t.Fatalf("Refresh() first call error = %v", err)
	}

	_, err = authService.Refresh(context.Background(), RefreshParams{RefreshToken: "old-refresh-token"})
	if !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("Refresh() second call error = %v, want ErrSessionExpired", err)
	}
}

func TestAuthServiceLogoutRevokesSession(t *testing.T) {
	store := newFakeAuthStore()
	userRepository := &fakeUserRepository{store: store}
	sessionRepository := &fakeSessionRepository{store: store}
	user, err := userRepository.Create(context.Background(), CreateUserParams{
		Email:        "user@example.com",
		PasswordHash: "stored-hash",
	})
	if err != nil {
		t.Fatalf("Create() setup error = %v", err)
	}

	now := testNow()
	session, err := sessionRepository.Create(context.Background(), CreateSessionParams{
		UserID:           user.ID,
		RefreshTokenHash: (&fakeTokenProvider{}).HashRefreshToken("logout-token"),
		ExpiresAt:        now.Add(24 * time.Hour),
		LastUsedAt:       now,
	})
	if err != nil {
		t.Fatalf("CreateSession() setup error = %v", err)
	}

	authService := newTestAuthService(t, AuthDependencies{
		PasswordHasher:    &fakePasswordHasher{},
		TokenProvider:     &fakeTokenProvider{},
		UserRepository:    userRepository,
		SessionRepository: sessionRepository,
		TxManager:         &fakeTxManager{store: store},
		Now:               func() time.Time { return now },
	})

	if err := authService.Logout(context.Background(), LogoutParams{RefreshToken: "logout-token"}); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}

	storedSession := store.sessionsByID[session.ID]
	if storedSession.RevokedAt == nil {
		t.Fatal("Logout() did not revoke session")
	}
}

func TestAuthServiceGetCurrentUser(t *testing.T) {
	store := newFakeAuthStore()
	userRepository := &fakeUserRepository{store: store}
	user, err := userRepository.Create(context.Background(), CreateUserParams{
		Email:        "user@example.com",
		PasswordHash: "stored-hash",
	})
	if err != nil {
		t.Fatalf("Create() setup error = %v", err)
	}

	authService := newTestAuthService(t, AuthDependencies{
		PasswordHasher:    &fakePasswordHasher{},
		TokenProvider:     &fakeTokenProvider{},
		UserRepository:    userRepository,
		SessionRepository: &fakeSessionRepository{store: store},
		TxManager:         &fakeTxManager{store: store},
	})

	currentUser, err := authService.GetCurrentUser(context.Background(), GetCurrentUserParams{
		AuthenticatedUser: model.AuthenticatedUser{UserID: user.ID},
	})
	if err != nil {
		t.Fatalf("GetCurrentUser() error = %v", err)
	}
	if currentUser.ID != user.ID {
		t.Fatalf("GetCurrentUser() user ID = %s, want %s", currentUser.ID, user.ID)
	}
}

func TestAuthServiceMapsInternalErrors(t *testing.T) {
	store := newFakeAuthStore()
	authService := newTestAuthService(t, AuthDependencies{
		PasswordHasher: &fakePasswordHasher{
			hashFn: func(password string) (string, error) {
				return "hashed:" + password, nil
			},
		},
		TokenProvider: &fakeTokenProvider{
			createRefreshTokenFn: func() (string, error) {
				return "refresh-token", nil
			},
			createAccessTokenFn: func(userID, sessionID uuid.UUID, issuedAt, expiresAt time.Time) (string, error) {
				return "access-token", nil
			},
		},
		UserRepository:    &fakeUserRepository{store: store, createErr: ErrUnexpectedStorage},
		SessionRepository: &fakeSessionRepository{store: store},
		TxManager:         &fakeTxManager{store: store},
	})

	_, err := authService.Register(context.Background(), RegisterParams{
		Email:    "user@example.com",
		Password: "password123",
	})
	if !errors.Is(err, ErrInternal) {
		t.Fatalf("Register() error = %v, want ErrInternal", err)
	}
}

func newTestAuthService(t *testing.T, deps AuthDependencies) *AuthService {
	t.Helper()

	if deps.Now == nil {
		deps.Now = testNow
	}

	service, err := NewAuthService(AuthConfig{
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 30 * 24 * time.Hour,
	}, deps)
	if err != nil {
		t.Fatalf("NewAuthService() error = %v", err)
	}

	return service
}

func testNow() time.Time {
	return time.Date(2026, time.August, 4, 12, 0, 0, 0, time.UTC)
}

type fakePasswordHasher struct {
	hashFn    func(password string) (string, error)
	matchesFn func(hash, password string) (bool, error)
}

func (h *fakePasswordHasher) Hash(password string) (string, error) {
	if h.hashFn != nil {
		return h.hashFn(password)
	}
	return password, nil
}

func (h *fakePasswordHasher) Matches(hash, password string) (bool, error) {
	if h.matchesFn != nil {
		return h.matchesFn(hash, password)
	}
	return hash == password, nil
}

type fakeTokenProvider struct {
	createAccessTokenFn  func(userID, sessionID uuid.UUID, issuedAt, expiresAt time.Time) (string, error)
	createRefreshTokenFn func() (string, error)
}

func (p *fakeTokenProvider) CreateAccessToken(userID, sessionID uuid.UUID, issuedAt, expiresAt time.Time) (string, error) {
	if p.createAccessTokenFn != nil {
		return p.createAccessTokenFn(userID, sessionID, issuedAt, expiresAt)
	}
	return "access-token", nil
}

func (p *fakeTokenProvider) CreateRefreshToken() (string, error) {
	if p.createRefreshTokenFn != nil {
		return p.createRefreshTokenFn()
	}
	return "refresh-token", nil
}

func (p *fakeTokenProvider) HashRefreshToken(token string) []byte {
	return []byte("hash:" + token)
}

type fakeRegistrationHook struct {
	called bool
	err    error
}

func (h *fakeRegistrationHook) AfterRegister(ctx context.Context, user model.User) error {
	h.called = true
	return h.err
}

type fakeAuthStore struct {
	usersByEmail  map[string]model.AuthUser
	usersByID     map[uuid.UUID]model.User
	sessionsByID  map[uuid.UUID]model.Session
	sessionByHash map[string]uuid.UUID
}

func newFakeAuthStore() *fakeAuthStore {
	return &fakeAuthStore{
		usersByEmail:  make(map[string]model.AuthUser),
		usersByID:     make(map[uuid.UUID]model.User),
		sessionsByID:  make(map[uuid.UUID]model.Session),
		sessionByHash: make(map[string]uuid.UUID),
	}
}

func (s *fakeAuthStore) clone() *fakeAuthStore {
	cloned := newFakeAuthStore()

	for email, user := range s.usersByEmail {
		cloned.usersByEmail[email] = user
	}
	for id, user := range s.usersByID {
		cloned.usersByID[id] = user
	}
	for id, session := range s.sessionsByID {
		cloned.sessionsByID[id] = cloneSession(session)
	}
	for hash, id := range s.sessionByHash {
		cloned.sessionByHash[hash] = id
	}

	return cloned
}

func (s *fakeAuthStore) restore(snapshot *fakeAuthStore) {
	s.usersByEmail = snapshot.usersByEmail
	s.usersByID = snapshot.usersByID
	s.sessionsByID = snapshot.sessionsByID
	s.sessionByHash = snapshot.sessionByHash
}

type fakeTxManager struct {
	store         *fakeAuthStore
	committed     bool
	rolledBack    bool
	withinTxCalls int
}

func (m *fakeTxManager) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	m.withinTxCalls++
	snapshot := m.store.clone()

	if err := fn(ctx); err != nil {
		m.rolledBack = true
		m.store.restore(snapshot)
		return err
	}

	m.committed = true
	return nil
}

type fakeUserRepository struct {
	store         *fakeAuthStore
	createErr     error
	getByIDErr    error
	getByEmailErr error
}

func (r *fakeUserRepository) Create(ctx context.Context, params CreateUserParams) (model.User, error) {
	if r.createErr != nil {
		return model.User{}, r.createErr
	}
	if _, exists := r.store.usersByEmail[params.Email]; exists {
		return model.User{}, ErrEmailAlreadyExists
	}

	now := testNow()
	user := model.User{
		ID:        uuid.New(),
		Email:     params.Email,
		CreatedAt: now,
		UpdatedAt: now,
	}
	authUser := model.AuthUser{
		User:         user,
		PasswordHash: params.PasswordHash,
	}

	r.store.usersByEmail[user.Email] = authUser
	r.store.usersByID[user.ID] = user
	return user, nil
}

func (r *fakeUserRepository) GetByEmail(ctx context.Context, email string) (model.AuthUser, error) {
	if r.getByEmailErr != nil {
		return model.AuthUser{}, r.getByEmailErr
	}
	user, ok := r.store.usersByEmail[email]
	if !ok {
		return model.AuthUser{}, ErrUserNotFound
	}
	return user, nil
}

func (r *fakeUserRepository) GetByID(ctx context.Context, id uuid.UUID) (model.User, error) {
	if r.getByIDErr != nil {
		return model.User{}, r.getByIDErr
	}
	user, ok := r.store.usersByID[id]
	if !ok {
		return model.User{}, ErrUserNotFound
	}
	return user, nil
}

type fakeSessionRepository struct {
	store        *fakeAuthStore
	createErr    error
	getActiveErr error
	rotateErr    error
	revokeErr    error
}

func (r *fakeSessionRepository) Create(ctx context.Context, params CreateSessionParams) (model.Session, error) {
	if r.createErr != nil {
		return model.Session{}, r.createErr
	}

	now := testNow()
	session := model.Session{
		ID:               uuid.New(),
		UserID:           params.UserID,
		RefreshTokenHash: append([]byte(nil), params.RefreshTokenHash...),
		ExpiresAt:        params.ExpiresAt,
		CreatedAt:        now,
		LastUsedAt:       params.LastUsedAt,
		UserAgent:        cloneStringPointer(params.UserAgent),
	}

	r.store.sessionsByID[session.ID] = session
	r.store.sessionByHash[string(session.RefreshTokenHash)] = session.ID
	return session, nil
}

func (r *fakeSessionRepository) GetActiveByRefreshTokenHash(ctx context.Context, refreshTokenHash []byte, now time.Time) (model.Session, error) {
	if r.getActiveErr != nil {
		return model.Session{}, r.getActiveErr
	}

	sessionID, ok := r.store.sessionByHash[string(refreshTokenHash)]
	if !ok {
		return model.Session{}, ErrSessionNotFound
	}

	session, ok := r.store.sessionsByID[sessionID]
	if !ok {
		return model.Session{}, ErrSessionNotFound
	}
	if session.RevokedAt != nil || !session.ExpiresAt.After(now) {
		return model.Session{}, ErrSessionNotFound
	}

	return cloneSession(session), nil
}

func (r *fakeSessionRepository) GetActiveByIDAndUserID(ctx context.Context, sessionID, userID uuid.UUID, now time.Time) (model.Session, error) {
	if r.getActiveErr != nil {
		return model.Session{}, r.getActiveErr
	}

	session, ok := r.store.sessionsByID[sessionID]
	if !ok {
		return model.Session{}, ErrSessionNotFound
	}
	if session.UserID != userID {
		return model.Session{}, ErrSessionNotFound
	}
	if session.RevokedAt != nil || !session.ExpiresAt.After(now) {
		return model.Session{}, ErrSessionNotFound
	}

	return cloneSession(session), nil
}

func (r *fakeSessionRepository) Rotate(ctx context.Context, params RotateSessionParams) (model.Session, error) {
	if r.rotateErr != nil {
		return model.Session{}, r.rotateErr
	}

	session, ok := r.store.sessionsByID[params.SessionID]
	if !ok {
		return model.Session{}, ErrSessionNotFound
	}
	if string(session.RefreshTokenHash) != string(params.OldRefreshTokenHash) {
		return model.Session{}, ErrSessionNotFound
	}
	if session.RevokedAt != nil || !session.ExpiresAt.After(params.LastUsedAt) {
		return model.Session{}, ErrSessionNotFound
	}

	delete(r.store.sessionByHash, string(session.RefreshTokenHash))
	session.RefreshTokenHash = append([]byte(nil), params.NewRefreshTokenHash...)
	session.ExpiresAt = params.NewExpiresAt
	session.LastUsedAt = params.LastUsedAt
	r.store.sessionsByID[session.ID] = session
	r.store.sessionByHash[string(session.RefreshTokenHash)] = session.ID

	return cloneSession(session), nil
}

func (r *fakeSessionRepository) Revoke(ctx context.Context, sessionID uuid.UUID, revokedAt time.Time) error {
	if r.revokeErr != nil {
		return r.revokeErr
	}

	session, ok := r.store.sessionsByID[sessionID]
	if !ok {
		return ErrSessionNotFound
	}
	if session.RevokedAt == nil {
		session.RevokedAt = &revokedAt
		r.store.sessionsByID[sessionID] = session
	}

	return nil
}

func cloneSession(session model.Session) model.Session {
	cloned := session
	cloned.RefreshTokenHash = append([]byte(nil), session.RefreshTokenHash...)
	cloned.UserAgent = cloneStringPointer(session.UserAgent)
	if session.RevokedAt != nil {
		revokedAt := *session.RevokedAt
		cloned.RevokedAt = &revokedAt
	}
	return cloned
}

func cloneStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func stringPointer(value string) *string {
	return &value
}
