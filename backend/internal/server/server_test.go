package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"devops-platform/internal/config"
	"devops-platform/internal/repository"
)

func setupTestServer(t *testing.T) *Server {
	t.Helper()
	cfg := &config.Config{
		LogLevel:  "debug",
		JWTSecret: "test-jwt-secret-key-12345",
		SecretKey: "0123456789abcdef0123456789abcdef",
	}
	logger := zap.NewNop()

	db, err := repository.InitDB(":memory:", logger)
	if err != nil {
		t.Fatalf("failed to init in-memory db: %v", err)
	}

	srv, err := New(cfg, logger, db)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	return srv
}

func TestHealthz(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.db.Close()

	req, _ := http.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Fatalf("expected status 'ok', got %v", resp["status"])
	}
}

func TestVersion(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.db.Close()

	req, _ := http.NewRequest("GET", "/api/platform/v1/version", nil)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["code"].(float64) != 0 {
		t.Fatalf("expected code 0, got %v", resp["code"])
	}
}

func TestAuthLoginAndMe(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.db.Close()

	// 1. Test Login with incorrect credentials
	badLoginPayload := []byte(`{"username":"admin","password":"wrongpassword"}`)
	req, _ := http.NewRequest("POST", "/api/platform/v1/auth/login", bytes.NewBuffer(badLoginPayload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 for bad login, got %d", w.Code)
	}

	// 2. Test Login with default admin credentials
	loginPayload := []byte(`{"username":"admin","password":"admin123"}`)
	req, _ = http.NewRequest("POST", "/api/platform/v1/auth/login", bytes.NewBuffer(loginPayload))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for valid login, got %d. Body: %s", w.Code, w.Body.String())
	}

	var loginResp struct {
		Code int `json:"code"`
		Data struct {
			Token string `json:"token"`
			User  struct {
				Username string `json:"username"`
				Role     string `json:"role"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("failed to decode login response: %v", err)
	}

	if loginResp.Code != 0 || loginResp.Data.Token == "" {
		t.Fatalf("login failed or empty token: %+v", loginResp)
	}
	if loginResp.Data.User.Username != "admin" || loginResp.Data.User.Role != "admin" {
		t.Fatalf("unexpected user info: %+v", loginResp.Data.User)
	}

	token := loginResp.Data.Token

	// 3. Test /me without token (should be 401)
	req, _ = http.NewRequest("GET", "/api/platform/v1/auth/me", nil)
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 for unauthenticated /me, got %d", w.Code)
	}

	// 4. Test /me with valid Bearer token
	req, _ = http.NewRequest("GET", "/api/platform/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for authenticated /me, got %d. Body: %s", w.Code, w.Body.String())
	}

	var meResp struct {
		Code int `json:"code"`
		Data struct {
			Username string `json:"username"`
			Role     string `json:"role"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &meResp); err != nil {
		t.Fatalf("failed to decode /me response: %v", err)
	}

	if meResp.Code != 0 || meResp.Data.Username != "admin" {
		t.Fatalf("unexpected /me response: %+v", meResp)
	}
}

func TestTektonApisFallback(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.db.Close()

	urls := []struct {
		url          string
		expectedKind string
	}{
		{"/apis/dashboard.tekton.dev/v1alpha1/extensions/", "ExtensionList"},
		{"/apis/dashboard.tekton.dev/v1alpha1/extensions", "ExtensionList"},
		{"/apis/tekton.dev/v1/pipelines/", "PipelineList"},
		{"/apis/tekton.dev/v1/pipelines", "PipelineList"},
		{"/apis/tekton.dev/v1/namespaces/default/pipelines/", "PipelineList"},
		{"/apis/tekton.dev/v1/namespaces/default/pipelines", "PipelineList"},
		{"/apis/tekton.dev/v1/pipelineruns/", "PipelineRunList"},
		{"/apis/tekton.dev/v1/pipelineruns", "PipelineRunList"},
		{"/apis/tekton.dev/v1/tasks/", "TaskList"},
		{"/apis/tekton.dev/v1/tasks", "TaskList"},
		{"/apis/triggers.tekton.dev/v1alpha1/eventlisteners/", "EventListenerList"},
		{"/api/v1/namespaces", "NamespaceList"},
		{"/api/v1/namespaces/", "NamespaceList"},
	}

	for _, item := range urls {
		req, _ := http.NewRequest("GET", item.url, nil)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200 for %s, got %d. Body: %s", item.url, w.Code, w.Body.String())
		}
		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Errorf("failed to parse JSON response for %s: %v", item.url, err)
		}
		if kind, ok := resp["kind"].(string); !ok || kind != item.expectedKind {
			t.Errorf("expected kind %s for %s, got %v", item.expectedKind, item.url, resp["kind"])
		}
		if meta, ok := resp["metadata"].(map[string]interface{}); !ok || meta["resourceVersion"] == nil {
			t.Errorf("expected metadata.resourceVersion for %s, got %v", item.url, resp["metadata"])
		}
	}
}

