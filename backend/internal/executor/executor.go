package executor

import (
	"context"

	"devops-platform/internal/compiler"
	"devops-platform/internal/repository"
)

// Executor defines the interface for submitting and monitoring pipeline runs
type Executor interface {
	Submit(ctx context.Context, run *repository.PipelineRun, spec *compiler.PipelineSpec) (string, error)
	Cancel(ctx context.Context, namespace, runName string) error
	GetLogsStream(ctx context.Context, namespace, runName string) (<-chan string, error)
}
