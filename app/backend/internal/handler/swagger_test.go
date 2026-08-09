package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAPISpecEndpoint(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t, testRouterConfig{})
	req := httptest.NewRequest(http.MethodGet, "/api/openapi.yaml", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if contentType := rec.Header().Get("Content-Type"); contentType != openAPIContentType {
		t.Fatalf("Content-Type = %q, want %q", contentType, openAPIContentType)
	}

	body := rec.Body.String()
	for _, want := range []string{
		"openapi: 3.0.3",
		"/api/auth/register:",
		"/api/auth/login:",
		"/api/auth/refresh:",
		"/api/auth/logout:",
		"/api/me:",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body does not contain %q", want)
		}
	}
}

func TestSwaggerUIEndpoint(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t, testRouterConfig{})
	req := httptest.NewRequest(http.MethodGet, "/swagger/", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if contentType := rec.Header().Get("Content-Type"); contentType != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want text/html; charset=utf-8", contentType)
	}

	body := rec.Body.String()
	for _, want := range []string{
		"/swagger/swagger-ui-bundle.js",
		"/swagger/swagger-ui.css",
		"/swagger/swagger-initializer.js",
		"Avitosha Auth API",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body does not contain %q", want)
		}
	}
	if strings.Contains(body, "https://") {
		t.Fatal("swagger ui page still references external assets")
	}
}

func TestSwaggerUIRedirect(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t, testRouterConfig{})
	req := httptest.NewRequest(http.MethodGet, "/swagger", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusMovedPermanently {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMovedPermanently)
	}
	if location := rec.Header().Get("Location"); location != "/swagger/" {
		t.Fatalf("Location = %q, want %q", location, "/swagger/")
	}
}

func TestSwaggerUIAssetsServedLocally(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t, testRouterConfig{})
	req := httptest.NewRequest(http.MethodGet, "/swagger/swagger-ui-bundle.js", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if body := rec.Body.String(); !strings.Contains(body, "SwaggerUIBundle") {
		t.Fatal("swagger bundle asset was not served")
	}
}
