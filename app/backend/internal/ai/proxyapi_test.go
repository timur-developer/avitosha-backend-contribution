package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/usecase"
)

func TestProxyAPIAdviceGenerator(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/openrouter/v1/chat/completions" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("authorization = %q", r.Header.Get("Authorization"))
		}
		var request chatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request.Model != "qwen/test" || len(request.Messages) != 2 {
			t.Errorf("request = %+v", request)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"advice\":\"Сравни несколько вариантов и сохрани подходящий.\"}"}}]}`))
	}))
	defer server.Close()

	generator, err := NewProxyAPIAdviceGenerator(ProxyAPIConfig{
		APIKey: "test-key", BaseURL: server.URL + "/openrouter/v1", Model: "qwen/test", Client: server.Client(),
	})
	if err != nil {
		t.Fatalf("NewProxyAPIAdviceGenerator() error = %v", err)
	}
	advice, err := generator.Generate(context.Background(), usecase.AdviceGenerationInput{
		PetName: "Авитоша", TaskTitle: "Выбери стол", ActionType: model.ActionTypeAdViewed,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if advice != "Сравни несколько вариантов и сохрани подходящий." {
		t.Fatalf("advice = %q", advice)
	}
}

func TestProxyAPIAdviceGeneratorRejectsInvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer server.Close()
	generator, err := NewProxyAPIAdviceGenerator(ProxyAPIConfig{
		APIKey: "test-key", BaseURL: server.URL, Model: "qwen/test", Client: server.Client(),
	})
	if err != nil {
		t.Fatalf("NewProxyAPIAdviceGenerator() error = %v", err)
	}
	if _, err = generator.Generate(context.Background(), usecase.AdviceGenerationInput{}); err == nil {
		t.Fatal("Generate() error = nil, want error")
	}
}
