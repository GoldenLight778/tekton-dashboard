package repository

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestAllRepositories(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	db, err := InitDBFromConfig(DBConfig{
		Driver: "sqlite",
		DSN:    ":memory:",
	}, logger)
	if err != nil {
		t.Fatalf("InitDBFromConfig failed: %v", err)
	}
	defer db.Close()

	// 1. User
	userRepo := NewUserRepository(db, "sqlite")
	user := &User{
		ID:           "u-1",
		Username:     "alice",
		PasswordHash: "hashed",
		Role:         "admin",
	}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("userRepo.Create failed: %v", err)
	}
	foundUser, err := userRepo.GetByUsername(ctx, "alice")
	if err != nil || foundUser.ID != "u-1" {
		t.Fatalf("userRepo.GetByUsername failed: %v", err)
	}

	// 2. Project
	projRepo := NewProjectRepository(db, "sqlite")
	proj := &Project{
		ID:          "proj-1",
		Name:        "Test Project",
		Description: "A demo project",
		CreatedBy:   "u-1",
	}
	if err := projRepo.Create(ctx, proj); err != nil {
		t.Fatalf("projRepo.Create failed: %v", err)
	}
	listProj, total, err := projRepo.List(ctx, 1, 10)
	if err != nil || total != 1 || len(listProj) != 1 {
		t.Fatalf("projRepo.List failed: total=%d, err=%v", total, err)
	}

	// 3. Pipeline
	pipeRepo := NewPipelineRepository(db, "sqlite")
	pipe := &Pipeline{
		ID:         "pipe-1",
		ProjectID:  "proj-1",
		Name:       "build-pipeline",
		DSLContent: "stages: []",
	}
	if err := pipeRepo.Create(ctx, pipe); err != nil {
		t.Fatalf("pipeRepo.Create failed: %v", err)
	}
	listPipe, totalPipe, err := pipeRepo.ListByProject(ctx, "proj-1", 1, 10)
	if err != nil || totalPipe != 1 || len(listPipe) != 1 {
		t.Fatalf("pipeRepo.ListByProject failed: total=%d, err=%v", totalPipe, err)
	}

	// 4. PipelineRun
	runRepo := NewPipelineRunRepository(db, "sqlite")
	start := time.Now().Add(-10 * time.Second)
	finish := time.Now()
	run := &PipelineRun{
		ID:                 "run-1",
		PipelineID:         "pipe-1",
		K8sPipelineRunName: "k8s-run-1",
		Namespace:          "default",
		Status:             "Succeeded",
		TriggeredBy:        "alice",
		StartedAt:          &start,
		FinishedAt:         &finish,
	}
	if err := runRepo.Create(ctx, run); err != nil {
		t.Fatalf("runRepo.Create failed: %v", err)
	}
	runs, totalRuns, err := runRepo.ListByPipeline(ctx, "pipe-1", 1, 10)
	if err != nil || totalRuns != 1 || len(runs) != 1 {
		t.Fatalf("runRepo.ListByPipeline failed: total=%d, err=%v", totalRuns, err)
	}

	// 5. Credential
	credRepo := NewCredentialRepository(db, "sqlite")
	cred := &Credential{
		ID:            "cred-1",
		ProjectID:     "proj-1",
		Name:          "github-token",
		Type:          "git_token",
		EncryptedData: "encrypted",
	}
	if err := credRepo.Create(ctx, cred); err != nil {
		t.Fatalf("credRepo.Create failed: %v", err)
	}
	creds, err := credRepo.ListByProject(ctx, "proj-1")
	if err != nil || len(creds) != 1 {
		t.Fatalf("credRepo.ListByProject failed: %v", err)
	}

	// 6. Audit
	auditRepo := NewAuditRepository(db, "sqlite")
	log := &AuditLog{
		UserID:       "u-1",
		Action:       "CREATE_PROJECT",
		ResourceType: "project",
		ResourceID:   "proj-1",
		Detail:       "Created project",
		IP:           "127.0.0.1",
	}
	if err := auditRepo.Record(ctx, log); err != nil {
		t.Fatalf("auditRepo.Record failed: %v", err)
	}
	if log.ID == 0 {
		t.Fatalf("expected audit log ID > 0, got %d", log.ID)
	}
	logs, totalLogs, err := auditRepo.List(ctx, 1, 10)
	if err != nil || totalLogs != 1 || len(logs) != 1 {
		t.Fatalf("auditRepo.List failed: total=%d, err=%v", totalLogs, err)
	}

	// 7. Metrics
	metricsRepo := NewMetricsRepository(db, "sqlite")
	m, err := metricsRepo.GetDashboardMetrics(ctx)
	if err != nil {
		t.Fatalf("metricsRepo.GetDashboardMetrics failed: %v", err)
	}
	if m.TotalProjects != 1 || m.TotalPipelines != 1 || m.TotalRuns != 1 || m.SucceededRuns != 1 {
		t.Fatalf("unexpected metrics: %+v", m)
	}
}
