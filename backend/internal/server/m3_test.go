package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestM3WebhookTrigger(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.db.Close()

	// 1. Login as admin
	loginPayload := []byte(`{"username":"admin","password":"admin123"}`)
	req, _ := http.NewRequest("POST", "/api/platform/v1/auth/login", bytes.NewBuffer(loginPayload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	var loginResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &loginResp)
	authHeader := "Bearer " + loginResp.Data.Token

	// 2. Create Project
	createProjPayload := []byte(`{"name":"webhook-project"}`)
	req, _ = http.NewRequest("POST", "/api/platform/v1/projects", bytes.NewBuffer(createProjPayload))
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	var projResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &projResp)
	projectID := projResp.Data.ID

	// 3. Create Pipeline
	createPipePayload := []byte(`{"name":"ci-pipeline","dsl_content":"stages:\n  - name: ci\n    tasks:\n      - name: test\n        script: echo ok"}`)
	req, _ = http.NewRequest("POST", fmt.Sprintf("/api/platform/v1/projects/%s/pipelines", projectID), bytes.NewBuffer(createPipePayload))
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	// 4. Send GitHub Push Webhook (Public POST to /webhooks/:id)
	githubPayload := []byte(`{
		"ref": "refs/heads/feature/login",
		"head_commit": {
			"id": "a1b2c3d4",
			"message": "Add user authentication",
			"author": { "name": "developer" }
		}
	}`)
	webhookURL := fmt.Sprintf("/api/platform/v1/webhooks/%s", projectID)
	req, _ = http.NewRequest("POST", webhookURL, bytes.NewBuffer(githubPayload))
	req.Header.Set("X-GitHub-Event", "push")
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from webhook, got %d: %s", w.Code, w.Body.String())
	}

	var hookResp struct {
		Code int `json:"code"`
		Data struct {
			Triggered bool   `json:"triggered"`
			Branch    string `json:"branch"`
			Commit    string `json:"commit"`
			RunID     string `json:"run_id"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &hookResp)

	if !hookResp.Data.Triggered || hookResp.Data.Branch != "feature/login" || hookResp.Data.Commit != "a1b2c3d4" {
		t.Fatalf("unexpected webhook trigger response: %+v", hookResp)
	}

	// 5. Verify the run was created and triggered_by has webhook
	req, _ = http.NewRequest("GET", fmt.Sprintf("/api/platform/v1/runs/%s", hookResp.Data.RunID), nil)
	req.Header.Set("Authorization", authHeader)
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	var runResp struct {
		Data struct {
			TriggeredBy string `json:"triggered_by"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &runResp)
	if runResp.Data.TriggeredBy != "webhook:github:feature/login" {
		t.Fatalf("unexpected triggered_by: %s", runResp.Data.TriggeredBy)
	}
}
