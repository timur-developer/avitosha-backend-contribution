package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/usecase"
)

type PhotoUploadUseCase interface {
	CreateUpload(usecase.CreatePhotoUploadParams) (usecase.PhotoUploadForm, error)
}

type PhotoUploadHandler struct {
	logger  *slog.Logger
	service PhotoUploadUseCase
	now     func() time.Time
}

func NewPhotoUploadHandler(logger *slog.Logger, service PhotoUploadUseCase, now func() time.Time) PhotoUploadHandler {
	if logger == nil {
		logger = slog.Default()
	}
	if now == nil {
		now = time.Now
	}
	return PhotoUploadHandler{logger: logger, service: service, now: now}
}

func (handler PhotoUploadHandler) Create(w http.ResponseWriter, r *http.Request) {
	if handler.service == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "photo_storage_unavailable", "Photo storage is not configured")
		return
	}
	userID, ok := gameUserID(r.Context())
	if !ok || userID == uuid.Nil {
		writeErrorResponse(w, http.StatusUnauthorized, unauthorizedCode, "Authentication is required")
		return
	}
	var request struct {
		FileName    string `json:"fileName"`
		ContentType string `json:"contentType"`
		Size        int64  `json:"size"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	form, err := handler.service.CreateUpload(usecase.CreatePhotoUploadParams{
		UserID: userID, FileName: request.FileName, ContentType: request.ContentType,
		Size: request.Size, Now: handler.now(),
	})
	if err != nil {
		if errors.Is(err, usecase.ErrPhotoUploadInvalid) {
			writeErrorResponse(w, http.StatusBadRequest, "invalid_photo", "Use a JPEG, PNG or WebP file up to the configured size limit")
			return
		}
		handler.logger.Error("create photo upload form", "error", err)
		writeErrorResponse(w, http.StatusInternalServerError, internalErrorCode, "Could not prepare photo upload")
		return
	}
	writeJSON(w, http.StatusCreated, form)
}
