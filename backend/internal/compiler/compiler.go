package compiler

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	ErrInvalidDSL = errors.New("invalid pipeline DSL format")
)

// TaskSpec represents a single task execution in a stage
type TaskSpec struct {
	Name   string            `json:"name" yaml:"name"`
	Image  string            `json:"image" yaml:"image"`
	Script string            `json:"script" yaml:"script"`
	Env    map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
}

// StageSpec represents a sequential stage containing tasks
type StageSpec struct {
	Name  string     `json:"name" yaml:"name"`
	Tasks []TaskSpec `json:"tasks" yaml:"tasks"`
}

// PipelineSpec represents the parsed simplified DSL pipeline
type PipelineSpec struct {
	Name   string      `json:"name" yaml:"name"`
	Stages []StageSpec `json:"stages" yaml:"stages"`
}

// Parse parses YAML or JSON DSL string into a PipelineSpec
func Parse(content string) (*PipelineSpec, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, ErrInvalidDSL
	}

	var spec PipelineSpec
	// Try JSON first
	if strings.HasPrefix(content, "{") {
		if err := json.Unmarshal([]byte(content), &spec); err == nil && len(spec.Stages) > 0 {
			return &spec, nil
		}
	}

	// Try YAML
	if err := yaml.Unmarshal([]byte(content), &spec); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidDSL, err)
	}

	if len(spec.Stages) == 0 {
		return nil, fmt.Errorf("%w: pipeline must have at least one stage", ErrInvalidDSL)
	}

	return &spec, nil
}

// CompileToTektonCRD compiles the DSL into standard Tekton PipelineRun CRD YAML
func CompileToTektonCRD(spec *PipelineSpec, runName, namespace string) (string, error) {
	manifest := map[string]interface{}{
		"apiVersion": "tekton.dev/v1",
		"kind":       "PipelineRun",
		"metadata": map[string]interface{}{
			"name":      runName,
			"namespace": namespace,
			"labels": map[string]string{
				"devops.platform/managed": "true",
				"devops.platform/pipeline": spec.Name,
			},
		},
		"spec": map[string]interface{}{
			"pipelineSpec": map[string]interface{}{
				"tasks": buildTektonTasks(spec.Stages),
			},
		},
	}

	out, err := yaml.Marshal(manifest)
	if err != nil {
		return "", fmt.Errorf("failed to marshal Tekton manifest: %w", err)
	}
	return string(out), nil
}

func buildTektonTasks(stages []StageSpec) []map[string]interface{} {
	var tasks []map[string]interface{}
	var prevTaskName string

	for _, stage := range stages {
		for _, task := range stage.Tasks {
			taskName := fmt.Sprintf("%s-%s", sanitize(stage.Name), sanitize(task.Name))
			image := task.Image
			if image == "" {
				image = "alpine:latest"
			}

			tektonTask := map[string]interface{}{
				"name": taskName,
				"taskSpec": map[string]interface{}{
					"steps": []map[string]interface{}{
						{
							"name":   "step-exec",
							"image":  image,
							"script": task.Script,
						},
					},
				},
			}

			if prevTaskName != "" {
				tektonTask["runAfter"] = []string{prevTaskName}
			}
			prevTaskName = taskName

			tasks = append(tasks, tektonTask)
		}
	}
	return tasks
}

func sanitize(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	return s
}
