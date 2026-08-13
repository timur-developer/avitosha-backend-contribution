package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestGameIdentityRejectsUserHeaderInProduction(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	handler := GameIdentity(nil, nil, "prod")(next)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/pet", nil)
	request.Header.Set("X-User-ID", uuid.NewString())
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestGameIdentityAllowsUserHeaderOutsideProduction(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if _, ok := gameUserID(r.Context()); !ok {
			t.Fatal("user ID was not added to request context")
		}
		w.WriteHeader(http.StatusNoContent)
	})
	handler := GameIdentity(nil, nil, "dev")(next)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/pet", nil)
	request.Header.Set("X-User-ID", uuid.NewString())
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || !called {
		t.Fatalf("status = %d, called = %v", response.Code, called)
	}
}
