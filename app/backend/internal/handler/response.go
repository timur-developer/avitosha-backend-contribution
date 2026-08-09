package handler

import (
	"encoding/json"
	"net/http"
)

const jsonContentType = "application/json; charset=utf-8"

type errorResponse struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type authResponse struct {
	AccessToken string       `json:"access_token"`
	User        responseUser `json:"user"`
}

type refreshResponse struct {
	AccessToken string `json:"access_token"`
}

type userResponse struct {
	User responseUser `json:"user"`
}

type responseUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", jsonContentType)
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeErrorResponse(w http.ResponseWriter, statusCode int, code, message string) {
	writeJSON(w, statusCode, errorResponse{
		Error: errorDetail{
			Code:    code,
			Message: message,
		},
	})
}
