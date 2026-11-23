package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestM2PipelineRunLifecycle(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.db.Close()

	// 1. Login
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
	createProjPayload := []byte(`{"name":"run-test-project"}`)
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
	createPipePayload := []byte(`{"name":"build-pipeline","dsl_content":"stages:\n  - name: build\n    tasks:\n      - name: compile\n        script: echo 'compiling...'"}`)
	req, _ = http.NewRequest("POST", fmt.Sprintf("/api/platform/v1/projects/%s/pipelines", projectID), bytes.NewBuffer(createPipePayload))
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	var pipeResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &pipeResp)
	pipelineID := pipeResp.Data.ID

	// 4. Trigger Run
	req, _ = http.NewRequest("POST", fmt.Sprintf("/api/platform/v1/pipelines/%s/runs", pipelineID), nil)
	req.Header.Set("Authorization", authHeader)
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 triggering run, got %d: %s", w.Code, w.Body.String())
	}

	var runResp struct {
		Data struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &runResp)
	runID := runResp.Data.ID

	if runID == "" {
		t.Fatalf("empty run id returned: %+v", runResp)
	}

	// 5. Query Run Details
	req, _ = http.NewRequest("GET", fmt.Sprintf("/api/platform/v1/runs/%s", runID), nil)
	req.Header.Set("Authorization", authHeader)
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 getting run, got %d", w.Code)
	}

	// 6. List Runs by Project
	req, _ = http.NewRequest("GET", fmt.Sprintf("/api/platform/v1/projects/%s/runs", projectID), nil)
	req.Header.Set("Authorization", authHeader)
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	var listRunsResp struct {
		Data struct {
			Total int `json:"total"`
			Items []struct {
				ID string `json:"id"`
			} `json:"items"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &listRunsResp)
	if listRunsResp.Data.Total != 1 || listRunsResp.Data.Items[0].ID != runID {
		t.Fatalf("unexpected list project runs: %+v", listRunsResp)
	}

	// 7. Cancel Run
	req, _ = http.NewRequest("POST", fmt.Sprintf("/api/platform/v1/runs/%s/cancel", runID), nil)
	req.Header.Set("Authorization", authHeader)
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 cancelling run, got %d: %s", w.Code, w.Body.String())
	}

	// Verify status is Cancelled
	time.Sleep(50 * time.Millisecond)
	req, _ = http.NewRequest("GET", fmt.Sprintf("/api/platform/v1/runs/%s", runID), nil)
	req.Header.Set("Authorization", authHeader)
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	var checkRunResp struct {
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &checkRunResp)
	if checkRunResp.Data.Status != "Cancelled" {
		t.Fatalf("expected status 'Cancelled', got %s", checkRunResp.Data.Status)
	}
}
