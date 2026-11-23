package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestM5DashboardMetrics(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.db.Close()

	router := srv.Router()

	// 1. Admin login
	loginBody, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "admin123",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/platform/v1/auth/login", bytes.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var loginResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &loginResp))
	token := loginResp.Data.Token

	// 2. Create Project
	projBody, _ := json.Marshal(map[string]string{
		"name":        "metrics-project",
		"description": "Project for metrics testing",
	})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/api/platform/v1/projects", bytes.NewReader(projBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var projResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &projResp))
	projectID := projResp.Data.ID

	// 3. Create 2 Pipelines
	for _, pipeName := range []string{"ci-pipeline", "cd-pipeline"} {
		pipeBody, _ := json.Marshal(map[string]string{
			"name":        pipeName,
			"dsl_content": "name: test\nstages:\n  - name: test\n    tasks:\n      - name: t1\n        image: alpine:latest\n        script: echo ok",
		})
		w = httptest.NewRecorder()
		req, _ = http.NewRequest(http.MethodPost, "/api/platform/v1/projects/"+projectID+"/pipelines", bytes.NewReader(pipeBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
	}

	// 4. Get pipelines to fetch their IDs
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/api/platform/v1/projects/"+projectID+"/pipelines", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var listPipeResp struct {
		Data struct {
			Items []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &listPipeResp))
	require.Len(t, listPipeResp.Data.Items, 2)

	// 5. Trigger runs on both pipelines
	for _, pipe := range listPipeResp.Data.Items {
		w = httptest.NewRecorder()
		req, _ = http.NewRequest(http.MethodPost, "/api/platform/v1/pipelines/"+pipe.ID+"/runs", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
	}

	// Allow fake executor goroutine to advance runs from Pending -> Running -> Succeeded
	time.Sleep(300 * time.Millisecond)

	// 6. Query Dashboard Metrics
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/api/platform/v1/metrics/dashboard", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var metricsResp struct {
		Code int `json:"code"`
		Data struct {
			TotalProjects  int            `json:"total_projects"`
			TotalPipelines int            `json:"total_pipelines"`
			TotalRuns      int            `json:"total_runs"`
			SucceededRuns  int            `json:"succeeded_runs"`
			SuccessRate    float64        `json:"success_rate"`
			StatusCounts   map[string]int `json:"status_counts"`
			RecentRuns     []struct {
				ID           string `json:"id"`
				PipelineName string `json:"pipeline_name"`
				ProjectName  string `json:"project_name"`
				Status       string `json:"status"`
			} `json:"recent_runs"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &metricsResp))

	assert.Equal(t, 1, metricsResp.Data.TotalProjects)
	assert.Equal(t, 2, metricsResp.Data.TotalPipelines)
	assert.Equal(t, 2, metricsResp.Data.TotalRuns)
	assert.NotEmpty(t, metricsResp.Data.RecentRuns)
	assert.Equal(t, "metrics-project", metricsResp.Data.RecentRuns[0].ProjectName)
}
