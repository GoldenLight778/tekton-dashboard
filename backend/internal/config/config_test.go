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
	if cfg.Database.Datasource != "./data/platform-dev.db" {
		t.Errorf("expected Database.Datasource './data/platform-dev.db', got '%s'", cfg.Database.Datasource)
	}
	if cfg.Datasource != "./data/platform-dev.db" {
		t.Errorf("expected Datasource './data/platform-dev.db', got '%s'", cfg.Datasource)
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
	if cfg.Database.Datasource != ":memory:" {
		t.Errorf("expected Database.Datasource ':memory:', got '%s'", cfg.Database.Datasource)
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
	expectedProdDS := "postgres://devops:password@localhost:5432/devops_prod?sslmode=disable"
	if cfg.Database.Datasource != expectedProdDS {
		t.Errorf("expected Database.Datasource '%s', got '%s'", expectedProdDS, cfg.Database.Datasource)
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
  datasource: "${CUSTOM_DB_USER:-root}:${CUSTOM_DB_PASS:-secret}@tcp(127.0.0.1:3306)/devops"
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
	if cfg.Database.Datasource != expectedDSN {
		t.Errorf("expected Datasource '%s', got '%s'", expectedDSN, cfg.Database.Datasource)
	}
	if cfg.Database.DSN != expectedDSN {
		t.Errorf("expected DSN '%s', got '%s'", expectedDSN, cfg.Database.DSN)
	}
}

func TestDatasourceEnvOverride(t *testing.T) {
	expectedDS := "postgres://custom:pass@localhost:5432/custom_db?sslmode=disable"
	t.Setenv("PLATFORM_DATASOURCE", expectedDS)
	cfg, err := Load("prod")
	if err != nil {
		t.Fatalf("failed to load prod config: %v", err)
	}

	if cfg.Database.Datasource != expectedDS {
		t.Errorf("expected Database.Datasource '%s', got '%s'", expectedDS, cfg.Database.Datasource)
	}
	if cfg.Database.DSN != expectedDS {
		t.Errorf("expected Database.DSN '%s', got '%s'", expectedDS, cfg.Database.DSN)
	}
	if cfg.Datasource != expectedDS {
		t.Errorf("expected Datasource '%s', got '%s'", expectedDS, cfg.Datasource)
	}
}
