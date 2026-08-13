package rpc

import (
	"errors"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	ReasonInvalidInput       = "INVALID_INPUT"
	ReasonEmailAlreadyExists = "EMAIL_ALREADY_EXISTS"
	ReasonInvalidCredentials = "INVALID_CREDENTIALS"
	ReasonSessionExpired     = "SESSION_EXPIRED"
	ReasonUnauthorized       = "UNAUTHORIZED"
	ReasonInvalidPetName     = "INVALID_PET_NAME"
	ReasonForbiddenPetName   = "FORBIDDEN_PET_NAME"
	ReasonInvalidAction      = "INVALID_ACTION"
	ReasonEventIDConflict    = "EVENT_ID_CONFLICT"
	ReasonTaskNotFound       = "TASK_NOT_FOUND"
	ReasonStoryNotFound      = "STORY_NOT_FOUND"
	ReasonListingNotFound    = "LISTING_NOT_FOUND"
	ReasonListingCategory    = "LISTING_CATEGORY_NOT_FOUND"
	ReasonListingForbidden   = "LISTING_FORBIDDEN"
	ReasonListingOwnAction   = "OWN_LISTING_ACTION"
	ReasonListingTransition  = "LISTING_INVALID_TRANSITION"
	ReasonListingEligibility = "LISTING_NOT_ELIGIBLE"
	ReasonDemoPurchaseDone   = "DEMO_PURCHASE_ALREADY_COMPLETED"
)

func AuthError(err error) error {
	switch {
	case errors.Is(err, usecase.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, ReasonInvalidInput)
	case errors.Is(err, usecase.ErrEmailAlreadyExists):
		return status.Error(codes.AlreadyExists, ReasonEmailAlreadyExists)
	case errors.Is(err, usecase.ErrInvalidCredentials):
		return status.Error(codes.Unauthenticated, ReasonInvalidCredentials)
	case errors.Is(err, usecase.ErrSessionExpired):
		return status.Error(codes.Unauthenticated, ReasonSessionExpired)
	case errors.Is(err, usecase.ErrUnauthorized):
		return status.Error(codes.Unauthenticated, ReasonUnauthorized)
	default:
		return status.Error(codes.Internal, "INTERNAL")
	}
}

func DecodeAuthError(err error) error {
	switch status.Convert(err).Message() {
	case ReasonInvalidInput:
		return usecase.ErrInvalidInput
	case ReasonEmailAlreadyExists:
		return usecase.ErrEmailAlreadyExists
	case ReasonInvalidCredentials:
		return usecase.ErrInvalidCredentials
	case ReasonSessionExpired:
		return usecase.ErrSessionExpired
	case ReasonUnauthorized:
		return usecase.ErrUnauthorized
	default:
		return usecase.ErrInternal
	}
}

func GameError(err error) error {
	switch {
	case errors.Is(err, usecase.ErrInvalidPetName):
		return status.Error(codes.InvalidArgument, ReasonInvalidPetName)
	case errors.Is(err, usecase.ErrForbiddenPetName):
		return status.Error(codes.InvalidArgument, ReasonForbiddenPetName)
	case errors.Is(err, usecase.ErrInvalidAction):
		return status.Error(codes.InvalidArgument, ReasonInvalidAction)
	case errors.Is(err, usecase.ErrEventIDConflict):
		return status.Error(codes.AlreadyExists, ReasonEventIDConflict)
	case errors.Is(err, usecase.ErrTaskNotFound):
		return status.Error(codes.NotFound, ReasonTaskNotFound)
	case errors.Is(err, usecase.ErrStoryNotFound):
		return status.Error(codes.NotFound, ReasonStoryNotFound)
	case errors.Is(err, usecase.ErrListingNotFound):
		return status.Error(codes.NotFound, ReasonListingNotFound)
	case errors.Is(err, usecase.ErrListingCategoryNotFound):
		return status.Error(codes.InvalidArgument, ReasonListingCategory)
	case errors.Is(err, usecase.ErrInvalidListingInput):
		return status.Error(codes.InvalidArgument, ReasonInvalidInput)
	case errors.Is(err, usecase.ErrListingForbidden):
		return status.Error(codes.PermissionDenied, ReasonListingForbidden)
	case errors.Is(err, usecase.ErrListingOwnAction):
		return status.Error(codes.FailedPrecondition, ReasonListingOwnAction)
	case errors.Is(err, usecase.ErrListingInvalidTransition):
		return status.Error(codes.FailedPrecondition, ReasonListingTransition)
	case errors.Is(err, usecase.ErrListingNotEligible):
		return status.Error(codes.FailedPrecondition, ReasonListingEligibility)
	case errors.Is(err, usecase.ErrDemoPurchaseCompleted):
		return status.Error(codes.AlreadyExists, ReasonDemoPurchaseDone)
	default:
		return status.Error(codes.Internal, "INTERNAL")
	}
}

func DecodeGameError(err error) error {
	switch status.Convert(err).Message() {
	case ReasonInvalidPetName:
		return usecase.ErrInvalidPetName
	case ReasonForbiddenPetName:
		return usecase.ErrForbiddenPetName
	case ReasonInvalidAction:
		return usecase.ErrInvalidAction
	case ReasonEventIDConflict:
		return usecase.ErrEventIDConflict
	case ReasonTaskNotFound:
		return usecase.ErrTaskNotFound
	case ReasonStoryNotFound:
		return usecase.ErrStoryNotFound
	case ReasonListingNotFound:
		return usecase.ErrListingNotFound
	case ReasonListingCategory:
		return usecase.ErrListingCategoryNotFound
	case ReasonListingForbidden:
		return usecase.ErrListingForbidden
	case ReasonListingOwnAction:
		return usecase.ErrListingOwnAction
	case ReasonListingTransition:
		return usecase.ErrListingInvalidTransition
	case ReasonListingEligibility:
		return usecase.ErrListingNotEligible
	case ReasonDemoPurchaseDone:
		return usecase.ErrDemoPurchaseCompleted
	case ReasonInvalidInput:
		return usecase.ErrInvalidListingInput
	default:
		return usecase.ErrUnexpectedStorage
	}
}
