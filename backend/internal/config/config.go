package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"devops-platform/configs"
	"devops-platform/internal/repository"
)

// Build variables injected via -ldflags
var (
	BuildEnv  string
	GitCommit string
	BuildTime string
)

// AppConfig holds HTTP server and authentication settings
type AppConfig struct {
	Env       string `yaml:"env"`
	Addr      string `yaml:"addr"`
	LogLevel  string `yaml:"log_level"`
	JWTSecret string `yaml:"jwt_secret"`
	SecretKey string `yaml:"secret_key"`
}

// DatabaseConfig holds DB driver, datasource connection string, and connection pool settings
type DatabaseConfig struct {
	Driver          string        `yaml:"driver"`
	Datasource      string        `yaml:"datasource"`
	DSN             string        `yaml:"dsn"` // alias for datasource
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

// KubernetesConfig holds cluster connection modes and credentials
type KubernetesConfig struct {
	Mode       string `yaml:"mode"` // minikube | kubeconfig | in-cluster | fake
	Kubeconfig string `yaml:"kubeconfig"`
	Context    string `yaml:"context"`
	Namespace  string `yaml:"namespace"`
	Insecure   bool   `yaml:"insecure"`
}

// Config holds all configuration options for the platform backend
type Config struct {
	App        AppConfig        `yaml:"app"`
	Database   DatabaseConfig   `yaml:"database"`
	Kubernetes KubernetesConfig `yaml:"kubernetes"`

	// Flat fields for backward compatibility with existing server code
	Addr       string
	DBPath     string
	Datasource string
	JWTSecret  string
	SecretKey  string
	Kubeconfig string
	LogLevel   string
	Env        string
}

// ActiveEnv returns the determined active environment based on precedence:
// 1. Explicit override passed as parameter
// 2. PLATFORM_ENV environment variable
// 3. BuildEnv (-ldflags)
// 4. CompileEnv (//go:build tags)
func ActiveEnv(override ...string) string {
	if len(override) > 0 && strings.TrimSpace(override[0]) != "" {
		return strings.TrimSpace(override[0])
	}
	if env := strings.TrimSpace(os.Getenv("PLATFORM_ENV")); env != "" {
		return env
	}
	if strings.TrimSpace(BuildEnv) != "" {
		return strings.TrimSpace(BuildEnv)
	}
	return CompileEnv
}

// Load loads configuration using the target environment or an optional config file path.
func Load(envName ...string) (*Config, error) {
	targetEnv := ActiveEnv(envName...)
	customFile := os.Getenv("PLATFORM_CONFIG_FILE")
	return LoadWithFile(targetEnv, customFile)
}

// LoadWithFile loads configuration from a custom file path or embedded environment defaults.
func LoadWithFile(envName, customFilePath string) (*Config, error) {
	targetEnv := ActiveEnv(envName)
	var rawData []byte
	var configOrigin string

	if customFilePath != "" {
		data, err := os.ReadFile(customFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file '%s': %w", customFilePath, err)
		}
		rawData = data
		configOrigin = fmt.Sprintf("file:%s", customFilePath)
	} else {
		// Read from embedded ConfigFS
		embeddedName := fmt.Sprintf("%s.yaml", targetEnv)
		data, err := configs.ConfigFS.ReadFile(embeddedName)
		if err != nil {
			// fallback to dev.yaml if specified env doesn't exist in embedded
			fallbackData, fbErr := configs.ConfigFS.ReadFile("dev.yaml")
			if fbErr != nil {
				return nil, fmt.Errorf("failed to load embedded config for env '%s': %w", targetEnv, err)
			}
			rawData = fallbackData
			configOrigin = fmt.Sprintf("embedded:dev.yaml (fallback from %s)", targetEnv)
		} else {
			rawData = data
			configOrigin = fmt.Sprintf("embedded:%s", embeddedName)
		}
	}

	// Expand environment variables: ${VAR} or ${VAR:-default}
	expanded := expandEnvVars(string(rawData))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config from %s: %w", configOrigin, err)
	}

	// Ensure Database.Datasource and Database.DSN are in sync
	if cfg.Database.Datasource == "" && cfg.Database.DSN != "" {
		cfg.Database.Datasource = cfg.Database.DSN
	}
	if cfg.Database.DSN == "" && cfg.Database.Datasource != "" {
		cfg.Database.DSN = cfg.Database.Datasource
	}

	// Environment variable overrides (higher priority than file)
	applyEnvOverrides(&cfg)

	// Populate backward-compatible flat fields
	cfg.Addr = cfg.App.Addr
	cfg.LogLevel = cfg.App.LogLevel
	cfg.JWTSecret = cfg.App.JWTSecret
	cfg.SecretKey = cfg.App.SecretKey
	cfg.Datasource = cfg.Database.Datasource
	cfg.DBPath = cfg.Database.Datasource
	cfg.Kubeconfig = cfg.Kubernetes.Kubeconfig
	cfg.Env = cfg.App.Env
	if cfg.Env == "" {
		cfg.Env = targetEnv
	}

	return &cfg, nil
}

// ToDBConfig converts Config's database section into repository.DBConfig
func (c *Config) ToDBConfig() repository.DBConfig {
	ds := c.Database.Datasource
	if ds == "" {
		ds = c.Database.DSN
	}
	return repository.DBConfig{
		Driver:          c.Database.Driver,
		Datasource:      ds,
		DSN:             ds,
		MaxOpenConns:    c.Database.MaxOpenConns,
		MaxIdleConns:    c.Database.MaxIdleConns,
		ConnMaxLifetime: c.Database.ConnMaxLifetime,
	}
}

var envVarPattern = regexp.MustCompile(`\$\{([A-Za-z0-9_]+)(?::-([^}]*))?\}`)

func expandEnvVars(raw string) string {
	return envVarPattern.ReplaceAllStringFunc(raw, func(match string) string {
		submatches := envVarPattern.FindStringSubmatch(match)
		if len(submatches) >= 2 {
			varName := submatches[1]
			defaultVal := ""
			if len(submatches) >= 3 {
				defaultVal = submatches[2]
			}
			if val, exists := os.LookupEnv(varName); exists && val != "" {
				return val
			}
			return defaultVal
		}
		return match
	})
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("PLATFORM_ADDR"); v != "" {
		cfg.App.Addr = v
	}
	if v := os.Getenv("PLATFORM_LOG_LEVEL"); v != "" {
		cfg.App.LogLevel = v
	}
	if v := os.Getenv("PLATFORM_JWT_SECRET"); v != "" {
		cfg.App.JWTSecret = v
	}
	if v := os.Getenv("PLATFORM_SECRET_KEY"); v != "" {
		cfg.App.SecretKey = v
	}
	if v := os.Getenv("PLATFORM_DB_DRIVER"); v != "" {
		cfg.Database.Driver = v
	}
	if v := os.Getenv("PLATFORM_DATASOURCE"); v != "" {
		cfg.Database.Datasource = v
		cfg.Database.DSN = v
	} else if v := os.Getenv("PLATFORM_DB_DSN"); v != "" {
		cfg.Database.Datasource = v
		cfg.Database.DSN = v
	}
	if v := os.Getenv("PLATFORM_K8S_MODE"); v != "" {
		cfg.Kubernetes.Mode = v
	}
	if v := os.Getenv("PLATFORM_KUBECONFIG"); v != "" {
		cfg.Kubernetes.Kubeconfig = v
	}
	if v := os.Getenv("PLATFORM_K8S_NAMESPACE"); v != "" {
		cfg.Kubernetes.Namespace = v
	}
}
