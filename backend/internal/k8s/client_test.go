package k8s

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"

	"devops-platform/internal/config"
)

func TestBuildClientBundleFake(t *testing.T) {
	logger := zap.NewNop()
	bundle, err := BuildClientBundle(config.KubernetesConfig{
		Mode: "fake",
	}, logger)
	if !errors.Is(err, ErrFakeMode) {
		t.Fatalf("expected ErrFakeMode, got err=%v, bundle=%v", err, bundle)
	}
}

func TestBuildClientBundleUnsupported(t *testing.T) {
	logger := zap.NewNop()
	_, err := BuildClientBundle(config.KubernetesConfig{
		Mode: "unknown-mode",
	}, logger)
	if err == nil {
		t.Fatal("expected error for unsupported mode, got nil")
	}
}

func TestBuildClientBundleKubeconfigMissing(t *testing.T) {
	logger := zap.NewNop()
	_, err := BuildClientBundle(config.KubernetesConfig{
		Mode:       "kubeconfig",
		Kubeconfig: filepath.Join(t.TempDir(), "nonexistent-config"),
	}, logger)
	if err == nil {
		t.Fatal("expected error for missing kubeconfig file, got nil")
	}
}

func TestBuildClientBundleWithValidKubeconfigFile(t *testing.T) {
	logger := zap.NewNop()
	tmpDir := t.TempDir()
	kcPath := filepath.Join(tmpDir, "test-kubeconfig")
	kcContent := `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://127.0.0.1:6443
    insecure-skip-tls-verify: true
  name: test-cluster
contexts:
- context:
    cluster: test-cluster
    user: test-user
  name: test-context
current-context: test-context
users:
- name: test-user
  user:
    token: test-token
`
	if err := os.WriteFile(kcPath, []byte(kcContent), 0644); err != nil {
		t.Fatalf("failed to write test kubeconfig: %v", err)
	}

	bundle, err := BuildClientBundle(config.KubernetesConfig{
		Mode:       "kubeconfig",
		Kubeconfig: kcPath,
		Namespace:  "custom-ns",
	}, logger)
	if err != nil {
		t.Fatalf("BuildClientBundle failed: %v", err)
	}

	if bundle == nil || bundle.RestConfig == nil {
		t.Fatal("expected non-nil bundle and RestConfig")
	}
	if bundle.RestConfig.Host != "https://127.0.0.1:6443" {
		t.Errorf("expected host https://127.0.0.1:6443, got %s", bundle.RestConfig.Host)
	}
	if bundle.Namespace != "custom-ns" {
		t.Errorf("expected namespace custom-ns, got %s", bundle.Namespace)
	}
}

