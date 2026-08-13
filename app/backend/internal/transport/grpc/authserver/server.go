package authserver

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	avitoshav1 "github.com/guitaramust-sudo/Avitosha/app/backend/internal/gen/avitosha/v1"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
	internalrpc "github.com/guitaramust-sudo/Avitosha/app/backend/internal/rpc"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthUseCase interface {
	Register(context.Context, usecase.RegisterParams) (usecase.AuthenticationResult, error)
	Login(context.Context, usecase.LoginParams) (usecase.AuthenticationResult, error)
	Refresh(context.Context, usecase.RefreshParams) (usecase.RefreshResult, error)
	Logout(context.Context, usecase.LogoutParams) error
	GetCurrentUser(context.Context, usecase.GetCurrentUserParams) (model.User, error)
}

type TokenAuthenticator interface {
	AuthenticateAccessToken(context.Context, string) (model.AuthenticatedUser, error)
}

type Server struct {
	avitoshav1.UnimplementedAuthServiceServer
	auth          AuthUseCase
	authenticator TokenAuthenticator
}

func New(auth AuthUseCase, authenticator TokenAuthenticator) *Server {
	return &Server{auth: auth, authenticator: authenticator}
}

func (s *Server) Register(ctx context.Context, request *avitoshav1.RegisterRequest) (*avitoshav1.AuthenticationResponse, error) {
	result, err := s.auth.Register(ctx, usecase.RegisterParams{
		Email: request.GetEmail(), Password: request.GetPassword(), UserAgent: optionalString(request.GetUserAgent()),
	})
	if err != nil {
		return nil, internalrpc.AuthError(err)
	}
	return authenticationResponse(result), nil
}

func (s *Server) Login(ctx context.Context, request *avitoshav1.LoginRequest) (*avitoshav1.AuthenticationResponse, error) {
	result, err := s.auth.Login(ctx, usecase.LoginParams{
		Email: request.GetEmail(), Password: request.GetPassword(), UserAgent: optionalString(request.GetUserAgent()),
	})
	if err != nil {
		return nil, internalrpc.AuthError(err)
	}
	return authenticationResponse(result), nil
}

func (s *Server) Refresh(ctx context.Context, request *avitoshav1.RefreshRequest) (*avitoshav1.RefreshResponse, error) {
	result, err := s.auth.Refresh(ctx, usecase.RefreshParams{RefreshToken: request.GetRefreshToken()})
	if err != nil {
		return nil, internalrpc.AuthError(err)
	}
	return &avitoshav1.RefreshResponse{AccessToken: result.AccessToken, RefreshToken: result.RefreshToken}, nil
}

func (s *Server) Logout(ctx context.Context, request *avitoshav1.LogoutRequest) (*avitoshav1.Empty, error) {
	if err := s.auth.Logout(ctx, usecase.LogoutParams{RefreshToken: request.GetRefreshToken()}); err != nil {
		return nil, internalrpc.AuthError(err)
	}
	return &avitoshav1.Empty{}, nil
}

func (s *Server) GetCurrentUser(ctx context.Context, request *avitoshav1.GetCurrentUserRequest) (*avitoshav1.UserResponse, error) {
	identity, err := parseIdentity(request.GetUserId(), request.GetSessionId())
	if err != nil {
		return nil, err
	}
	user, err := s.auth.GetCurrentUser(ctx, usecase.GetCurrentUserParams{AuthenticatedUser: identity})
	if err != nil {
		return nil, internalrpc.AuthError(err)
	}
	return &avitoshav1.UserResponse{User: userMessage(user)}, nil
}

func (s *Server) AuthenticateAccessToken(ctx context.Context, request *avitoshav1.AuthenticateAccessTokenRequest) (*avitoshav1.AuthenticatedUserResponse, error) {
	identity, err := s.authenticator.AuthenticateAccessToken(ctx, request.GetAccessToken())
	if err != nil {
		return nil, internalrpc.AuthError(err)
	}
	return &avitoshav1.AuthenticatedUserResponse{
		UserId: identity.UserID.String(), SessionId: identity.SessionID.String(),
	}, nil
}

func authenticationResponse(result usecase.AuthenticationResult) *avitoshav1.AuthenticationResponse {
	return &avitoshav1.AuthenticationResponse{
		User: userMessage(result.User), AccessToken: result.AccessToken, RefreshToken: result.RefreshToken,
	}
}

func userMessage(user model.User) *avitoshav1.User {
	return &avitoshav1.User{
		Id: user.ID.String(), Email: user.Email,
		CreatedAt: user.CreatedAt.UTC().Format(time.RFC3339Nano), UpdatedAt: user.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func parseIdentity(userIDValue, sessionIDValue string) (model.AuthenticatedUser, error) {
	userID, err := uuid.Parse(userIDValue)
	if err != nil {
		return model.AuthenticatedUser{}, status.Error(codes.InvalidArgument, "user_id must be a UUID")
	}
	sessionID, err := uuid.Parse(sessionIDValue)
	if err != nil {
		return model.AuthenticatedUser{}, status.Error(codes.InvalidArgument, "session_id must be a UUID")
	}
	return model.AuthenticatedUser{UserID: userID, SessionID: sessionID}, nil
}
