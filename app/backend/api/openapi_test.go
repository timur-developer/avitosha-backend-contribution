package api

import (
	"context"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestOpenAPIYAMLIsValid(t *testing.T) {
	t.Parallel()

	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(OpenAPIYAML())
	if err != nil {
		t.Fatalf("LoadFromData() error = %v", err)
	}

	if err := doc.Validate(context.Background()); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestRefreshUnauthorizedContract(t *testing.T) {
	t.Parallel()

	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(OpenAPIYAML())
	if err != nil {
		t.Fatalf("LoadFromData() error = %v", err)
	}

	refreshPath := doc.Paths.Find("/api/auth/refresh")
	if refreshPath == nil || refreshPath.Post == nil {
		t.Fatal("refresh operation is missing")
	}

	responseRef := refreshPath.Post.Responses.Status(httpStatusUnauthorized)
	if responseRef == nil || responseRef.Value == nil {
		t.Fatal("refresh 401 response is missing")
	}

	content := responseRef.Value.Content.Get("application/json")
	if content == nil {
		t.Fatal("refresh 401 response content is missing")
	}

	missingCookie, ok := content.Examples["missing_refresh_cookie"]
	if !ok || missingCookie.Value == nil {
		t.Fatal("refresh 401 missing_refresh_cookie example is missing")
	}
	expiredSession, ok := content.Examples["expired_session"]
	if !ok || expiredSession.Value == nil {
		t.Fatal("refresh 401 expired_session example is missing")
	}
}

func TestGameAPIContract(t *testing.T) {
	t.Parallel()

	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(OpenAPIYAML())
	if err != nil {
		t.Fatalf("LoadFromData() error = %v", err)
	}

	tests := []struct {
		path   string
		method string
	}{
		{path: "/api/v1/pet", method: "GET"},
		{path: "/api/v1/pet", method: "PATCH"},
		{path: "/api/v1/tasks", method: "GET"},
		{path: "/api/v1/tasks/{task_id}", method: "GET"},
		{path: "/api/v1/tasks/{task_id}/advice", method: "GET"},
		{path: "/api/v1/actions", method: "POST"},
		{path: "/api/v1/room", method: "GET"},
		{path: "/api/v1/story", method: "GET"},
		{path: "/api/v1/daily-summary", method: "GET"},
		{path: "/api/v1/leaderboard", method: "GET"},
		{path: "/api/v1/achievements", method: "GET"},
		{path: "/api/v1/rewards/balance", method: "GET"},
		{path: "/api/v1/rewards/wallet", method: "GET"},
		{path: "/api/v1/ws", method: "GET"},
	}
	for _, tt := range tests {
		path := doc.Paths.Find(tt.path)
		if path == nil {
			t.Errorf("pet path %s is missing", tt.path)
			continue
		}
		operation := path.GetOperation(tt.method)
		if operation == nil {
			t.Errorf("%s %s operation is missing", tt.method, tt.path)
			continue
		}
		if operation.Security == nil || len(*operation.Security) != 2 {
			t.Errorf("%s %s must support bearer and X-User-ID authentication", tt.method, tt.path)
		}
	}

	for _, schema := range []string{
		"PetProfile", "RenamePetRequest", "TaskProgress", "ActionRequest", "ActionResult", "RoomResponse",
		"StoryResponse", "DailySummary", "LeaderboardResponse", "Achievement",
		"RewardBalance", "RewardBalanceResponse", "RewardWallet", "RetentionOverview", "TaskAdvice",
	} {
		if _, ok := doc.Components.Schemas[schema]; !ok {
			t.Errorf("schema %s is missing", schema)
		}
	}
}

const httpStatusUnauthorized = 401
