package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestActiveEnv(t *testing.T) {
	// 1. By default, with no overrides, should match CompileEnv
	if env := ActiveEnv(); env != CompileEnv {
		t.Fatalf("expected ActiveEnv() = %s, got %s", CompileEnv, env)
	}

	// 2. Explicit parameter override
	if env := ActiveEnv("prod"); env != "prod" {
		t.Fatalf("expected ActiveEnv('prod') = 'prod', got %s", env)
	}

	// 3. Environment variable override
	t.Setenv("PLATFORM_ENV", "test")
	if env := ActiveEnv(); env != "test" {
		t.Fatalf("expected ActiveEnv() = 'test', got %s", env)
	}
}

func TestLoadEmbeddedDev(t *testing.T) {
	cfg, err := Load("dev")
	if err != nil {
		t.Fatalf("failed to load dev config: %v", err)
	}

	if cfg.App.Env != "dev" {
		t.Errorf("expected App.Env 'dev', got '%s'", cfg.App.Env)
	}
	if cfg.Database.Driver != "sqlite" {
		t.Errorf("expected Database.Driver 'sqlite', got '%s'", cfg.Database.Driver)
	}
	if cfg.Kubernetes.Mode != "minikube" {
		t.Errorf("expected Kubernetes.Mode 'minikube', got '%s'", cfg.Kubernetes.Mode)
	}
}

func TestLoadEmbeddedTest(t *testing.T) {
	cfg, err := Load("test")
	if err != nil {
		t.Fatalf("failed to load test config: %v", err)
	}

	if cfg.App.Env != "test" {
		t.Errorf("expected App.Env 'test', got '%s'", cfg.App.Env)
	}
	if cfg.Database.DSN != ":memory:" {
		t.Errorf("expected Database.DSN ':memory:', got '%s'", cfg.Database.DSN)
	}
	if cfg.Kubernetes.Mode != "fake" {
		t.Errorf("expected Kubernetes.Mode 'fake', got '%s'", cfg.Kubernetes.Mode)
	}
}

func TestLoadEmbeddedProd(t *testing.T) {
	cfg, err := Load("prod")
	if err != nil {
		t.Fatalf("failed to load prod config: %v", err)
	}

	if cfg.App.Env != "prod" {
		t.Errorf("expected App.Env 'prod', got '%s'", cfg.App.Env)
	}
	if cfg.Database.Driver != "postgres" {
		t.Errorf("expected Database.Driver 'postgres', got '%s'", cfg.Database.Driver)
	}
	if cfg.Kubernetes.Mode != "in-cluster" {
		t.Errorf("expected Kubernetes.Mode 'in-cluster', got '%s'", cfg.Kubernetes.Mode)
	}
}

func TestLoadWithCustomFileAndEnvExpansion(t *testing.T) {
	tmpDir := t.TempDir()
	customPath := filepath.Join(tmpDir, "custom.yaml")
	content := `
app:
  env: custom
  addr: ":9999"
database:
  driver: mysql
  dsn: "${CUSTOM_DB_USER:-root}:${CUSTOM_DB_PASS:-secret}@tcp(127.0.0.1:3306)/devops"
kubernetes:
  mode: kubeconfig
`
	if err := os.WriteFile(customPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write custom config file: %v", err)
	}

	t.Setenv("CUSTOM_DB_USER", "myuser")
	// CUSTOM_DB_PASS is unset, should fallback to "secret"

	cfg, err := LoadWithFile("custom", customPath)
	if err != nil {
		t.Fatalf("LoadWithFile failed: %v", err)
	}

	if cfg.App.Addr != ":9999" {
		t.Errorf("expected Addr ':9999', got '%s'", cfg.App.Addr)
	}
	if cfg.Database.Driver != "mysql" {
		t.Errorf("expected Driver 'mysql', got '%s'", cfg.Database.Driver)
	}
	expectedDSN := "myuser:secret@tcp(127.0.0.1:3306)/devops"
	if cfg.Database.DSN != expectedDSN {
		t.Errorf("expected DSN '%s', got '%s'", expectedDSN, cfg.Database.DSN)
	}
}
