package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestM4CredentialsLifecycle(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.db.Close()

	router := srv.Router()

	// 1. Login as default admin
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
		Code int `json:"code"`
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &loginResp))
	adminToken := loginResp.Data.Token
	require.NotEmpty(t, adminToken)

	// 2. Create Project
	projBody, _ := json.Marshal(map[string]string{
		"name":        "m4-project",
		"description": "Project for testing credentials",
	})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/api/platform/v1/projects", bytes.NewReader(projBody))
	req.Header.Set("Authorization", "Bearer "+adminToken)
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

	// 3. Create Credentials
	cred1Body, _ := json.Marshal(map[string]string{
		"name":   "github-token",
		"type":   "git_token",
		"secret": "ghp_superSecretToken1234567890",
	})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/api/platform/v1/projects/"+projectID+"/credentials", bytes.NewReader(cred1Body))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var credResp struct {
		Data struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Type        string `json:"type"`
			MaskedValue string `json:"masked_value"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &credResp))
	credID := credResp.Data.ID
	assert.NotEmpty(t, credID)
	assert.Equal(t, "github-token", credResp.Data.Name)
	assert.Equal(t, "git_token", credResp.Data.Type)
	assert.Equal(t, "******", credResp.Data.MaskedValue)
	// Verify raw secret is NOT leaked in response
	assert.NotContains(t, w.Body.String(), "ghp_superSecretToken1234567890")

	// 4. List Credentials
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/api/platform/v1/projects/"+projectID+"/credentials", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "github-token")
	assert.NotContains(t, w.Body.String(), "ghp_superSecretToken1234567890")

	// 5. Delete Credential
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodDelete, "/api/platform/v1/credentials/"+credID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 6. Verify Deletion
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/api/platform/v1/projects/"+projectID+"/credentials", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), "github-token")

	// 7. Verify Audit Log contains create and delete credential
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/api/platform/v1/audit-logs", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "create_credential")
	assert.Contains(t, w.Body.String(), "delete_credential")
}
