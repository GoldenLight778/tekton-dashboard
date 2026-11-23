package executor

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"devops-platform/internal/compiler"
	"devops-platform/internal/repository"
)

var pipelineRunGVR = schema.GroupVersionResource{
	Group:    "tekton.dev",
	Version:  "v1",
	Resource: "pipelineruns",
}

// K8sExecutor submits and manages real Tekton PipelineRuns on a Kubernetes cluster
type K8sExecutor struct {
	dynClient    dynamic.Interface
	clientset    *kubernetes.Clientset
	namespace    string
	updateStatus StatusUpdateFunc
	logger       *zap.Logger
}

// NewK8sExecutor creates a new K8sExecutor connected to a live Kubernetes cluster
func NewK8sExecutor(
	dyn dynamic.Interface,
	cs *kubernetes.Clientset,
	namespace string,
	updateStatus StatusUpdateFunc,
	logger *zap.Logger,
) *K8sExecutor {
	if namespace == "" {
		namespace = "default"
	}
	return &K8sExecutor{
		dynClient:    dyn,
		clientset:    cs,
		namespace:    namespace,
		updateStatus: updateStatus,
		logger:       logger,
	}
}

// Submit creates a Tekton PipelineRun resource in the target namespace
func (e *K8sExecutor) Submit(ctx context.Context, run *repository.PipelineRun, spec *compiler.PipelineSpec) (string, error) {
	k8sName := fmt.Sprintf("%s-%d", run.PipelineID, time.Now().Unix())
	if len(k8sName) > 63 {
		k8sName = k8sName[:63]
	}

	tasks := make([]interface{}, 0, len(spec.Stages))
	for _, stage := range spec.Stages {
		steps := make([]interface{}, 0, len(stage.Tasks))
		for _, task := range stage.Tasks {
			steps = append(steps, map[string]interface{}{
				"name":   task.Name,
				"image":  task.Image,
				"script": task.Script,
			})
		}
		tasks = append(tasks, map[string]interface{}{
			"name": stage.Name,
			"taskSpec": map[string]interface{}{
				"steps": steps,
			},
		})
	}

	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "tekton.dev/v1",
			"kind":       "PipelineRun",
			"metadata": map[string]interface{}{
				"name":      k8sName,
				"namespace": e.namespace,
				"labels": map[string]interface{}{
					"devops.platform/pipeline-id": run.PipelineID,
					"devops.platform/run-id":      run.ID,
				},
			},
			"spec": map[string]interface{}{
				"pipelineSpec": map[string]interface{}{
					"tasks": tasks,
				},
			},
		},
	}

	created, err := e.dynClient.Resource(pipelineRunGVR).Namespace(e.namespace).Create(ctx, obj, metav1.CreateOptions{})
	if err != nil {
		e.logger.Error("failed to submit PipelineRun to Kubernetes", zap.Error(err), zap.String("name", k8sName))
		return "", fmt.Errorf("failed to submit PipelineRun to Kubernetes: %w", err)
	}

	now := time.Now().UTC()
	if e.updateStatus != nil {
		e.updateStatus(run.ID, "Running", &now, nil)
	}

	e.logger.Info("successfully submitted PipelineRun to Kubernetes",
		zap.String("name", created.GetName()),
		zap.String("namespace", e.namespace),
	)
	return created.GetName(), nil
}

// Cancel patches the PipelineRun spec.status to "Cancelled"
func (e *K8sExecutor) Cancel(ctx context.Context, namespace, runName string) error {
	if namespace == "" {
		namespace = e.namespace
	}
	patch := []byte(`{"spec":{"status":"Cancelled"}}`)
	_, err := e.dynClient.Resource(pipelineRunGVR).Namespace(namespace).Patch(
		ctx,
		runName,
		types.MergePatchType,
		patch,
		metav1.PatchOptions{},
	)
	if err != nil {
		e.logger.Error("failed to cancel PipelineRun", zap.String("name", runName), zap.Error(err))
		return fmt.Errorf("failed to cancel PipelineRun in Kubernetes: %w", err)
	}
	return nil
}

// GetLogsStream yields log lines for the given run
func (e *K8sExecutor) GetLogsStream(ctx context.Context, namespace, runName string) (<-chan string, error) {
	if namespace == "" {
		namespace = e.namespace
	}
	ch := make(chan string, 100)
	go func() {
		defer close(ch)
		ch <- fmt.Sprintf("[%s] [K8S] Watching PipelineRun '%s' in namespace '%s'...\n", time.Now().Format("15:04:05"), runName, namespace)
	}()
	return ch, nil
}
