package repository

import (
	"testing"

	"go.uber.org/zap"
)

func TestInitDBMemory(t *testing.T) {
	logger := zap.NewNop()
	db, err := InitDB(":memory:", logger)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer db.Close()

	// Verify tables were created by migration
	var count int
	err = db.QueryRow("SELECT count(*) FROM users").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query users table: %v", err)
	}

	if count != 0 {
		t.Fatalf("expected 0 users, got %d", count)
	}
}
