package k8s

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"devops-platform/internal/config"
)

var (
	ErrFakeMode        = errors.New("kubernetes mode is set to fake")
	ErrNoClusterConfig = errors.New("no valid kubernetes cluster configuration found")
)

// ClientBundle holds REST config and client instances
type ClientBundle struct {
	Mode       string
	RestConfig *rest.Config
	Clientset  *kubernetes.Clientset
	Dynamic    dynamic.Interface
	Namespace  string
}

// BuildClientBundle constructs a *ClientBundle according to the KubernetesConfig mode:
// minikube, kubeconfig, in-cluster, fake
func BuildClientBundle(cfg config.KubernetesConfig, logger *zap.Logger) (*ClientBundle, error) {
	mode := strings.ToLower(strings.TrimSpace(cfg.Mode))
	if mode == "" || mode == "fake" {
		logger.Info("Kubernetes mode is 'fake', using offline simulation")
		return nil, ErrFakeMode
	}

	var restConfig *rest.Config
	var err error

	switch mode {
	case "in-cluster":
		logger.Info("Connecting to Kubernetes via in-cluster service account")
		restConfig, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to load in-cluster config: %w", err)
		}

	case "minikube", "kubeconfig":
		kubeconfigPath := resolveKubeconfigPath(cfg.Kubeconfig)
		logger.Info("Connecting to Kubernetes via kubeconfig",
			zap.String("mode", mode),
			zap.String("kubeconfig", kubeconfigPath),
			zap.String("context", cfg.Context),
		)

		if _, statErr := os.Stat(kubeconfigPath); statErr != nil {
			return nil, fmt.Errorf("kubeconfig file not found at '%s': %w", kubeconfigPath, statErr)
		}

		loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath}
		configOverrides := &clientcmd.ConfigOverrides{}
		if cfg.Context != "" {
			configOverrides.CurrentContext = cfg.Context
		} else if mode == "minikube" {
			configOverrides.CurrentContext = "minikube"
		}

		clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
		restConfig, err = clientConfig.ClientConfig()
		if err != nil {
			// If minikube context was not found, retry with default context
			if mode == "minikube" && configOverrides.CurrentContext == "minikube" {
				logger.Warn("context 'minikube' not found in kubeconfig, falling back to current context")
				configOverrides.CurrentContext = ""
				clientConfig = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
				restConfig, err = clientConfig.ClientConfig()
			}
			if err != nil {
				return nil, fmt.Errorf("failed to build kubeconfig client: %w", err)
			}
		}

	default:
		return nil, fmt.Errorf("unsupported kubernetes mode '%s'", mode)
	}

	if cfg.Insecure {
		restConfig.Insecure = true
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	dynClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	ns := cfg.Namespace
	if ns == "" {
		ns = "default"
	}

	return &ClientBundle{
		Mode:       mode,
		RestConfig: restConfig,
		Clientset:  clientset,
		Dynamic:    dynClient,
		Namespace:  ns,
	}, nil
}

func resolveKubeconfigPath(explicit string) string {
	if explicit != "" {
		return explicit
	}
	if env := os.Getenv("KUBECONFIG"); env != "" {
		return env
	}
	home, err := os.UserHomeDir()
	if err == nil {
		return filepath.Join(home, ".kube", "config")
	}
	return ""
}
