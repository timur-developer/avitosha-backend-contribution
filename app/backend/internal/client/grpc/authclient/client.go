package authclient

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	avitoshav1 "github.com/guitaramust-sudo/Avitosha/app/backend/internal/gen/avitosha/v1"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
	internalrpc "github.com/guitaramust-sudo/Avitosha/app/backend/internal/rpc"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/usecase"
)

type Client struct{ rpc avitoshav1.AuthServiceClient }

func New(rpc avitoshav1.AuthServiceClient) *Client { return &Client{rpc: rpc} }

func (c *Client) Register(ctx context.Context, params usecase.RegisterParams) (usecase.AuthenticationResult, error) {
	response, err := c.rpc.Register(ctx, &avitoshav1.RegisterRequest{
		Email: params.Email, Password: params.Password, UserAgent: stringValue(params.UserAgent),
	})
	if err != nil {
		return usecase.AuthenticationResult{}, fmt.Errorf("register over grpc: %w", internalrpc.DecodeAuthError(err))
	}
	user, err := parseUser(response.GetUser())
	if err != nil {
		return usecase.AuthenticationResult{}, err
	}
	return usecase.AuthenticationResult{User: user, AccessToken: response.GetAccessToken(), RefreshToken: response.GetRefreshToken()}, nil
}

func (c *Client) Login(ctx context.Context, params usecase.LoginParams) (usecase.AuthenticationResult, error) {
	response, err := c.rpc.Login(ctx, &avitoshav1.LoginRequest{
		Email: params.Email, Password: params.Password, UserAgent: stringValue(params.UserAgent),
	})
	if err != nil {
		return usecase.AuthenticationResult{}, fmt.Errorf("login over grpc: %w", internalrpc.DecodeAuthError(err))
	}
	user, err := parseUser(response.GetUser())
	if err != nil {
		return usecase.AuthenticationResult{}, err
	}
	return usecase.AuthenticationResult{User: user, AccessToken: response.GetAccessToken(), RefreshToken: response.GetRefreshToken()}, nil
}

func (c *Client) Refresh(ctx context.Context, params usecase.RefreshParams) (usecase.RefreshResult, error) {
	response, err := c.rpc.Refresh(ctx, &avitoshav1.RefreshRequest{RefreshToken: params.RefreshToken})
	if err != nil {
		return usecase.RefreshResult{}, fmt.Errorf("refresh over grpc: %w", internalrpc.DecodeAuthError(err))
	}
	return usecase.RefreshResult{AccessToken: response.GetAccessToken(), RefreshToken: response.GetRefreshToken()}, nil
}

func (c *Client) Logout(ctx context.Context, params usecase.LogoutParams) error {
	_, err := c.rpc.Logout(ctx, &avitoshav1.LogoutRequest{RefreshToken: params.RefreshToken})
	if err != nil {
		return fmt.Errorf("logout over grpc: %w", internalrpc.DecodeAuthError(err))
	}
	return nil
}

func (c *Client) GetCurrentUser(ctx context.Context, params usecase.GetCurrentUserParams) (model.User, error) {
	response, err := c.rpc.GetCurrentUser(ctx, &avitoshav1.GetCurrentUserRequest{
		UserId: params.AuthenticatedUser.UserID.String(), SessionId: params.AuthenticatedUser.SessionID.String(),
	})
	if err != nil {
		return model.User{}, fmt.Errorf("get current user over grpc: %w", internalrpc.DecodeAuthError(err))
	}
	return parseUser(response.GetUser())
}

func (c *Client) AuthenticateAccessToken(ctx context.Context, token string) (model.AuthenticatedUser, error) {
	response, err := c.rpc.AuthenticateAccessToken(ctx, &avitoshav1.AuthenticateAccessTokenRequest{AccessToken: token})
	if err != nil {
		return model.AuthenticatedUser{}, fmt.Errorf("authenticate access token over grpc: %w", internalrpc.DecodeAuthError(err))
	}
	userID, err := uuid.Parse(response.GetUserId())
	if err != nil {
		return model.AuthenticatedUser{}, fmt.Errorf("authenticate access token over grpc: invalid user id: %w", usecase.ErrInternal)
	}
	sessionID, err := uuid.Parse(response.GetSessionId())
	if err != nil {
		return model.AuthenticatedUser{}, fmt.Errorf("authenticate access token over grpc: invalid session id: %w", usecase.ErrInternal)
	}
	return model.AuthenticatedUser{UserID: userID, SessionID: sessionID}, nil
}

func parseUser(message *avitoshav1.User) (model.User, error) {
	if message == nil {
		return model.User{}, fmt.Errorf("decode auth grpc response: missing user: %w", usecase.ErrInternal)
	}
	id, err := uuid.Parse(message.GetId())
	if err != nil {
		return model.User{}, fmt.Errorf("decode auth grpc response: invalid user id: %w", usecase.ErrInternal)
	}
	createdAt, err := time.Parse(time.RFC3339Nano, message.GetCreatedAt())
	if err != nil {
		return model.User{}, fmt.Errorf("decode auth grpc response: invalid created_at: %w", usecase.ErrInternal)
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, message.GetUpdatedAt())
	if err != nil {
		return model.User{}, fmt.Errorf("decode auth grpc response: invalid updated_at: %w", usecase.ErrInternal)
	}
	return model.User{ID: id, Email: message.GetEmail(), CreatedAt: createdAt, UpdatedAt: updatedAt}, nil
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
