package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSONDeclaresUTF8Charset(t *testing.T) {
	recorder := httptest.NewRecorder()

	writeJSON(recorder, http.StatusOK, map[string]string{"name": "Авитоша"})

	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want UTF-8 JSON", contentType)
	}
}
