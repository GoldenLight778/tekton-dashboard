package executor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"devops-platform/internal/compiler"
	"devops-platform/internal/repository"
)

// StatusUpdateFunc is called when a fake run status changes
type StatusUpdateFunc func(id, status string, startedAt, finishedAt *time.Time)

// FakeExecutor simulates K8s Tekton execution for local development
type FakeExecutor struct {
	mu           sync.RWMutex
	logsMap      map[string][]string
	subscribers  map[string][]chan string
	cancelled    map[string]bool
	updateStatus StatusUpdateFunc
	logger       *zap.Logger
}

// NewFakeExecutor creates a new FakeExecutor instance
func NewFakeExecutor(updateStatus StatusUpdateFunc, logger *zap.Logger) *FakeExecutor {
	return &FakeExecutor{
		logsMap:      make(map[string][]string),
		subscribers:  make(map[string][]chan string),
		cancelled:    make(map[string]bool),
		updateStatus: updateStatus,
		logger:       logger,
	}
}

func (e *FakeExecutor) Submit(ctx context.Context, run *repository.PipelineRun, spec *compiler.PipelineSpec) (string, error) {
	k8sName := fmt.Sprintf("%s-fake-run", run.ID[:8])
	e.logger.Info("submitting pipeline to FakeExecutor", zap.String("run_id", run.ID), zap.String("k8s_name", k8sName))

	// Launch background execution simulation
	go e.simulateExecution(run.ID, k8sName, spec)

	return k8sName, nil
}

func (e *FakeExecutor) simulateExecution(runID, k8sName string, spec *compiler.PipelineSpec) {
	if e.isCancelled(k8sName) {
		return
	}

	now := time.Now().UTC()
	if e.updateStatus != nil {
		e.updateStatus(runID, "Running", &now, nil)
	}

	e.emitLog(k8sName, fmt.Sprintf("[%s] [PIPELINE] Initializing run %s for pipeline '%s'...", time.Now().Format("15:04:05"), runID[:8], spec.Name))
	time.Sleep(300 * time.Millisecond)

	for _, stage := range spec.Stages {
		if e.isCancelled(k8sName) {
			e.emitLog(k8sName, fmt.Sprintf("[%s] [CANCEL] Run was cancelled by user.", time.Now().Format("15:04:05")))
			finished := time.Now().UTC()
			if e.updateStatus != nil {
				e.updateStatus(runID, "Cancelled", nil, &finished)
			}
			return
		}

		e.emitLog(k8sName, fmt.Sprintf("[%s] [STAGE] Entering stage: %s", time.Now().Format("15:04:05"), stage.Name))
		time.Sleep(400 * time.Millisecond)

		for _, task := range stage.Tasks {
			if e.isCancelled(k8sName) {
				return
			}
			e.emitLog(k8sName, fmt.Sprintf("[%s] [TASK] Running task '%s' using image %s", time.Now().Format("15:04:05"), task.Name, task.Image))
			time.Sleep(300 * time.Millisecond)
			if task.Script != "" {
				e.emitLog(k8sName, fmt.Sprintf("[%s] > %s", time.Now().Format("15:04:05"), task.Script))
			}
			e.emitLog(k8sName, fmt.Sprintf("[%s] [TASK] Task '%s' completed successfully (exit code 0)", time.Now().Format("15:04:05"), task.Name))
			time.Sleep(300 * time.Millisecond)
		}
	}

	finished := time.Now().UTC()
	e.emitLog(k8sName, fmt.Sprintf("[%s] [SUCCESS] All stages completed successfully. Status: Succeeded.", time.Now().Format("15:04:05")))

	if e.updateStatus != nil {
		e.updateStatus(runID, "Succeeded", nil, &finished)
	}
}

func (e *FakeExecutor) emitLog(k8sName, line string) {
	e.mu.Lock()
	e.logsMap[k8sName] = append(e.logsMap[k8sName], line)
	subs := make([]chan string, len(e.subscribers[k8sName]))
	copy(subs, e.subscribers[k8sName])
	e.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- line:
		default:
		}
	}
}

func (e *FakeExecutor) isCancelled(k8sName string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.cancelled[k8sName]
}

func (e *FakeExecutor) Cancel(ctx context.Context, namespace, runName string) error {
	e.mu.Lock()
	e.cancelled[runName] = true
	e.mu.Unlock()
	return nil
}

func (e *FakeExecutor) GetLogsStream(ctx context.Context, namespace, runName string) (<-chan string, error) {
	out := make(chan string, 100)

	e.mu.Lock()
	// Send existing logs
	for _, l := range e.logsMap[runName] {
		out <- l
	}
	e.subscribers[runName] = append(e.subscribers[runName], out)
	e.mu.Unlock()

	// Clean up subscriber when context is cancelled
	go func() {
		<-ctx.Done()
		e.mu.Lock()
		defer e.mu.Unlock()
		subs := e.subscribers[runName]
		for i, ch := range subs {
			if ch == out {
				e.subscribers[runName] = append(subs[:i], subs[i+1:]...)
				close(out)
				break
			}
		}
	}()

	return out, nil
}
