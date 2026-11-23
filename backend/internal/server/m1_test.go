package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestM1ProjectsAndPipelinesCRUD(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.db.Close()

	// 1. Login as admin to get token
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
	token := loginResp.Data.Token
	authHeader := "Bearer " + token

	// 2. Create Project
	createProjPayload := []byte(`{"name":"demo-service","description":"A microservice project"}`)
	req, _ = http.NewRequest("POST", "/api/platform/v1/projects", bytes.NewBuffer(createProjPayload))
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 creating project, got %d: %s", w.Code, w.Body.String())
	}

	var projResp struct {
		Code int `json:"code"`
		Data struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &projResp)
	projectID := projResp.Data.ID

	if projectID == "" || projResp.Data.Name != "demo-service" {
		t.Fatalf("unexpected project response: %+v", projResp)
	}

	// 3. Try to create duplicate project (should be 409 Conflict)
	req, _ = http.NewRequest("POST", "/api/platform/v1/projects", bytes.NewBuffer(createProjPayload))
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for duplicate project, got %d", w.Code)
	}

	// 4. Create Pipeline in project
	createPipePayload := []byte(`{"name":"build-and-test","dsl_content":"stages:\n  - name: build\n    tasks: []"}`)
	pipeURL := fmt.Sprintf("/api/platform/v1/projects/%s/pipelines", projectID)
	req, _ = http.NewRequest("POST", pipeURL, bytes.NewBuffer(createPipePayload))
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 creating pipeline, got %d: %s", w.Code, w.Body.String())
	}

	var pipeResp struct {
		Data struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &pipeResp)
	pipelineID := pipeResp.Data.ID

	// 5. List Pipelines in Project
	req, _ = http.NewRequest("GET", pipeURL, nil)
	req.Header.Set("Authorization", authHeader)
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 listing pipelines, got %d", w.Code)
	}

	var listPipeResp struct {
		Data struct {
			Total int `json:"total"`
			Items []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"items"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &listPipeResp)
	if listPipeResp.Data.Total != 1 || listPipeResp.Data.Items[0].ID != pipelineID {
		t.Fatalf("unexpected list pipelines: %+v", listPipeResp)
	}

	// 6. Check Audit Logs
	req, _ = http.NewRequest("GET", "/api/platform/v1/audit-logs", nil)
	req.Header.Set("Authorization", authHeader)
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 listing audit logs, got %d", w.Code)
	}

	var auditResp struct {
		Data struct {
			Total int `json:"total"`
			Items []struct {
				Action string `json:"action"`
			} `json:"items"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &auditResp)
	if auditResp.Data.Total < 2 {
		t.Fatalf("expected at least 2 audit logs (project + pipeline creation), got %d", auditResp.Data.Total)
	}
}
