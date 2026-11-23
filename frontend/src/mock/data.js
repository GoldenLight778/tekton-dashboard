/*
Copyright 2026 The Tekton Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

export const mockProperties = {
  dashboardNamespace: 'tekton-pipelines',
  dashboardVersion: 'v0.73.0-dev',
  isReadOnly: false,
  logoutURL: '',
  pipelinesNamespace: 'tekton-pipelines',
  pipelinesVersion: 'v0.58.0',
  triggersNamespace: 'tekton-pipelines',
  triggersVersion: 'v0.27.0'
};

export const mockNamespaces = {
  apiVersion: 'v1',
  kind: 'NamespaceList',
  metadata: { resourceVersion: '1' },
  items: [
    { metadata: { name: 'default', uid: 'ns-1' } },
    { metadata: { name: 'production', uid: 'ns-2' } },
    { metadata: { name: 'staging', uid: 'ns-3' } },
    { metadata: { name: 'tekton-pipelines', uid: 'ns-4' } }
  ]
};

export const mockPipelines = {
  apiVersion: 'tekton.dev/v1',
  kind: 'PipelineList',
  metadata: { resourceVersion: '1' },
  items: [
    {
      apiVersion: 'tekton.dev/v1',
      kind: 'Pipeline',
      metadata: {
        name: 'build-and-deploy-service',
        namespace: 'production',
        creationTimestamp: '2026-11-18T08:00:00Z',
        uid: 'pipe-1',
        labels: {
          'app.kubernetes.io/part-of': 'core-service',
          team: 'backend'
        }
      },
      spec: {
        description:
          'Compiles Golang microservice, runs unit & integration tests, builds Docker image, and deploys to Kubernetes cluster.',
        params: [
          {
            name: 'git-repo-url',
            type: 'string',
            description: 'Git repository URL'
          },
          {
            name: 'branch',
            type: 'string',
            default: 'main',
            description: 'Target git branch'
          },
          {
            name: 'image-tag',
            type: 'string',
            default: 'latest',
            description: 'Container image tag'
          }
        ],
        tasks: [
          { name: 'fetch-repository', taskRef: { name: 'git-clone' } },
          {
            name: 'lint-and-test',
            runAfter: ['fetch-repository'],
            taskRef: { name: 'golang-test' }
          },
          {
            name: 'build-push-image',
            runAfter: ['lint-and-test'],
            taskRef: { name: 'kaniko' }
          },
          {
            name: 'deploy-helm',
            runAfter: ['build-push-image'],
            taskRef: { name: 'helm-upgrade' }
          }
        ]
      }
    },
    {
      apiVersion: 'tekton.dev/v1',
      kind: 'Pipeline',
      metadata: {
        name: 'frontend-ci-cd',
        namespace: 'default',
        creationTimestamp: '2026-11-19T10:30:00Z',
        uid: 'pipe-2',
        labels: {
          'app.kubernetes.io/part-of': 'dashboard-web',
          team: 'frontend'
        }
      },
      spec: {
        description:
          'Vite & React CI/CD pipeline: linting, vitest suite, production build, and CDN sync.',
        params: [{ name: 'environment', type: 'string', default: 'staging' }],
        tasks: [
          { name: 'git-clone', taskRef: { name: 'git-clone' } },
          {
            name: 'run-lint-test',
            runAfter: ['git-clone'],
            taskRef: { name: 'npm-test' }
          },
          {
            name: 'vite-bundle',
            runAfter: ['run-lint-test'],
            taskRef: { name: 'npm-build' }
          }
        ]
      }
    },
    {
      apiVersion: 'tekton.dev/v1',
      kind: 'Pipeline',
      metadata: {
        name: 'nightly-security-audit',
        namespace: 'default',
        creationTimestamp: '2026-11-15T02:00:00Z',
        uid: 'pipe-3',
        labels: {
          category: 'security',
          schedule: 'nightly'
        }
      },
      spec: {
        description:
          'Nightly vulnerability and SBOM scanning for all base container images using Trivy and Grype.',
        tasks: [
          { name: 'trivy-scan', taskRef: { name: 'trivy' } },
          { name: 'generate-sbom', taskRef: { name: 'syft' } }
        ]
      }
    }
  ]
};

export const mockPipelineRuns = {
  apiVersion: 'tekton.dev/v1',
  kind: 'PipelineRunList',
  metadata: { resourceVersion: '1' },
  items: [
    {
      apiVersion: 'tekton.dev/v1',
      kind: 'PipelineRun',
      metadata: {
        name: 'build-and-deploy-service-prod-r9f2a',
        namespace: 'production',
        creationTimestamp: '2026-11-20T03:15:20Z',
        uid: 'pr-1',
        labels: {
          'tekton.dev/pipeline': 'build-and-deploy-service',
          'triggers.tekton.dev/eventlistener': 'github-webhook-listener',
          'triggers.tekton.dev/trigger': 'push-main-trigger'
        }
      },
      spec: {
        pipelineRef: { name: 'build-and-deploy-service' },
        params: [
          {
            name: 'git-repo-url',
            value: 'https://github.com/tektoncd/core-service.git'
          },
          { name: 'branch', value: 'main' },
          { name: 'image-tag', value: 'v2.4.0' }
        ]
      },
      status: {
        startTime: '2026-11-20T03:15:21Z',
        completionTime: '2026-11-20T03:19:33Z',
        lastTransitionTime: '2026-11-20T03:19:33Z',
        conditions: [
          {
            type: 'Succeeded',
            status: 'True',
            reason: 'Succeeded',
            message: 'Tasks Completed: 4 (Failed: 0, Cancelled 0), Skipped: 0',
            lastTransitionTime: '2026-11-20T03:19:33Z'
          }
        ],
        pipelineSpec: {
          tasks: [
            { name: 'fetch-repository' },
            { name: 'lint-and-test' },
            { name: 'build-push-image' },
            { name: 'deploy-helm' }
          ]
        },
        childReferences: [
          {
            apiVersion: 'tekton.dev/v1',
            kind: 'TaskRun',
            name: 'pr-1-fetch-repo',
            pipelineTaskName: 'fetch-repository'
          },
          {
            apiVersion: 'tekton.dev/v1',
            kind: 'TaskRun',
            name: 'pr-1-lint-test',
            pipelineTaskName: 'lint-and-test'
          },
          {
            apiVersion: 'tekton.dev/v1',
            kind: 'TaskRun',
            name: 'pr-1-build-push',
            pipelineTaskName: 'build-push-image'
          },
          {
            apiVersion: 'tekton.dev/v1',
            kind: 'TaskRun',
            name: 'pr-1-deploy-helm',
            pipelineTaskName: 'deploy-helm'
          }
        ]
      }
    },
    {
      apiVersion: 'tekton.dev/v1',
      kind: 'PipelineRun',
      metadata: {
        name: 'frontend-ci-cd-pr-412-active',
        namespace: 'default',
        creationTimestamp: '2026-11-20T03:32:10Z',
        uid: 'pr-2',
        labels: {
          'tekton.dev/pipeline': 'frontend-ci-cd',
          'triggers.tekton.dev/eventlistener': 'pr-webhook-listener',
          'triggers.tekton.dev/trigger': 'pr-opened-trigger'
        }
      },
      spec: {
        pipelineRef: { name: 'frontend-ci-cd' },
        params: [{ name: 'environment', value: 'staging' }]
      },
      status: {
        startTime: '2026-11-20T03:32:11Z',
        lastTransitionTime: '2026-11-20T03:32:11Z',
        conditions: [
          {
            type: 'Succeeded',
            status: 'Unknown',
            reason: 'Running',
            message:
              'Tasks Completed: 1 (Failed: 0, Cancelled 0), In Progress: 1',
            lastTransitionTime: '2026-11-20T03:32:11Z'
          }
        ],
        childReferences: [
          {
            apiVersion: 'tekton.dev/v1',
            kind: 'TaskRun',
            name: 'pr-2-clone',
            pipelineTaskName: 'git-clone'
          },
          {
            apiVersion: 'tekton.dev/v1',
            kind: 'TaskRun',
            name: 'pr-2-test',
            pipelineTaskName: 'run-lint-test'
          }
        ]
      }
    },
    {
      apiVersion: 'tekton.dev/v1',
      kind: 'PipelineRun',
      metadata: {
        name: 'nightly-security-audit-err-x188',
        namespace: 'default',
        creationTimestamp: '2026-11-20T02:00:00Z',
        uid: 'pr-3',
        labels: {
          'tekton.dev/pipeline': 'nightly-security-audit'
        }
      },
      spec: {
        pipelineRef: { name: 'nightly-security-audit' }
      },
      status: {
        startTime: '2026-11-20T02:00:01Z',
        completionTime: '2026-11-20T02:04:12Z',
        lastTransitionTime: '2026-11-20T02:04:12Z',
        conditions: [
          {
            type: 'Succeeded',
            status: 'False',
            reason: 'Failed',
            message:
              'TaskRun pr-3-trivy failed: Critical severity CVE-2026-4491 detected in base image',
            lastTransitionTime: '2026-11-20T02:04:12Z'
          }
        ],
        childReferences: [
          {
            apiVersion: 'tekton.dev/v1',
            kind: 'TaskRun',
            name: 'pr-3-trivy',
            pipelineTaskName: 'trivy-scan'
          }
        ]
      }
    },
    {
      apiVersion: 'tekton.dev/v1',
      kind: 'PipelineRun',
      metadata: {
        name: 'build-and-deploy-service-manual-cncl',
        namespace: 'staging',
        creationTimestamp: '2026-11-19T18:20:00Z',
        uid: 'pr-4',
        labels: {
          'tekton.dev/pipeline': 'build-and-deploy-service'
        }
      },
      spec: {
        pipelineRef: { name: 'build-and-deploy-service' }
      },
      status: {
        startTime: '2026-11-19T18:20:02Z',
        completionTime: '2026-11-19T18:22:15Z',
        lastTransitionTime: '2026-11-19T18:22:15Z',
        conditions: [
          {
            type: 'Succeeded',
            status: 'False',
            reason: 'Cancelled',
            message:
              'PipelineRun "build-and-deploy-service-manual-cncl" was cancelled by user admin',
            lastTransitionTime: '2026-11-19T18:22:15Z'
          }
        ]
      }
    }
  ]
};

export const mockTasks = {
  apiVersion: 'tekton.dev/v1',
  kind: 'TaskList',
  metadata: { resourceVersion: '1' },
  items: [
    {
      apiVersion: 'tekton.dev/v1',
      kind: 'Task',
      metadata: { name: 'git-clone', namespace: 'default', uid: 't-1' },
      spec: {
        description: 'Clones a target Git repository into the workspace.',
        steps: [{ name: 'clone', image: 'alpine/git:latest' }]
      }
    },
    {
      apiVersion: 'tekton.dev/v1',
      kind: 'Task',
      metadata: { name: 'golang-test', namespace: 'production', uid: 't-2' },
      spec: {
        description:
          'Executes go vet, staticcheck, and go test with race detector.',
        steps: [{ name: 'unit-tests', image: 'golang:1.24-alpine' }]
      }
    },
    {
      apiVersion: 'tekton.dev/v1',
      kind: 'Task',
      metadata: { name: 'kaniko', namespace: 'production', uid: 't-3' },
      spec: {
        description:
          'Builds and pushes a Docker container image inside Kubernetes without Docker daemon.',
        steps: [
          {
            name: 'build-and-push',
            image: 'gcr.io/kaniko-project/executor:latest'
          }
        ]
      }
    },
    {
      apiVersion: 'tekton.dev/v1',
      kind: 'Task',
      metadata: { name: 'npm-build', namespace: 'default', uid: 't-4' },
      spec: {
        description: 'Runs npm install and vite build.',
        steps: [
          { name: 'install', image: 'node:22-alpine' },
          { name: 'build', image: 'node:22-alpine' }
        ]
      }
    }
  ]
};

export const mockTaskRuns = {
  apiVersion: 'tekton.dev/v1',
  kind: 'TaskRunList',
  metadata: { resourceVersion: '1' },
  items: [
    {
      apiVersion: 'tekton.dev/v1',
      kind: 'TaskRun',
      metadata: {
        name: 'pr-1-fetch-repo',
        namespace: 'production',
        creationTimestamp: '2026-11-20T03:15:22Z',
        uid: 'tr-1',
        labels: {
          'tekton.dev/pipelineTask': 'fetch-repository',
          'tekton.dev/pipelineRun': 'build-and-deploy-service-prod-r9f2a'
        }
      },
      spec: {
        taskRef: { name: 'git-clone' }
      },
      status: {
        startTime: '2026-11-20T03:15:23Z',
        completionTime: '2026-11-20T03:16:05Z',
        podName: 'pod-fetch-repo-01',
        conditions: [
          {
            type: 'Succeeded',
            status: 'True',
            reason: 'Succeeded',
            message: 'All Steps have completed successfully'
          }
        ],
        steps: [
          {
            name: 'clone',
            container: 'step-clone',
            terminated: {
              exitCode: 0,
              reason: 'Completed',
              startedAt: '2026-11-20T03:15:25Z',
              finishedAt: '2026-11-20T03:16:04Z'
            }
          }
        ]
      }
    },
    {
      apiVersion: 'tekton.dev/v1',
      kind: 'TaskRun',
      metadata: {
        name: 'pr-1-lint-test',
        namespace: 'production',
        creationTimestamp: '2026-11-20T03:16:06Z',
        uid: 'tr-2',
        labels: {
          'tekton.dev/pipelineTask': 'lint-and-test',
          'tekton.dev/pipelineRun': 'build-and-deploy-service-prod-r9f2a'
        }
      },
      spec: {
        taskRef: { name: 'golang-test' }
      },
      status: {
        startTime: '2026-11-20T03:16:07Z',
        completionTime: '2026-11-20T03:17:42Z',
        podName: 'pod-lint-test-02',
        conditions: [
          {
            type: 'Succeeded',
            status: 'True',
            reason: 'Succeeded',
            message: 'All Steps have completed successfully'
          }
        ],
        steps: [
          {
            name: 'unit-tests',
            container: 'step-unit-tests',
            terminated: {
              exitCode: 0,
              reason: 'Completed',
              startedAt: '2026-11-20T03:16:08Z',
              finishedAt: '2026-11-20T03:17:41Z'
            }
          }
        ]
      }
    },
    {
      apiVersion: 'tekton.dev/v1',
      kind: 'TaskRun',
      metadata: {
        name: 'pr-1-build-push',
        namespace: 'production',
        creationTimestamp: '2026-11-20T03:17:43Z',
        uid: 'tr-3',
        labels: {
          'tekton.dev/pipelineTask': 'build-push-image',
          'tekton.dev/pipelineRun': 'build-and-deploy-service-prod-r9f2a'
        }
      },
      spec: {
        taskRef: { name: 'kaniko' }
      },
      status: {
        startTime: '2026-11-20T03:17:44Z',
        completionTime: '2026-11-20T03:19:10Z',
        podName: 'pod-build-push-03',
        conditions: [
          {
            type: 'Succeeded',
            status: 'True',
            reason: 'Succeeded',
            message: 'All Steps have completed successfully'
          }
        ],
        steps: [
          {
            name: 'build-and-push',
            container: 'step-build-push',
            terminated: {
              exitCode: 0,
              reason: 'Completed',
              startedAt: '2026-11-20T03:17:45Z',
              finishedAt: '2026-11-20T03:19:09Z'
            }
          }
        ]
      }
    },
    {
      apiVersion: 'tekton.dev/v1',
      kind: 'TaskRun',
      metadata: {
        name: 'pr-1-deploy-helm',
        namespace: 'production',
        creationTimestamp: '2026-11-20T03:19:11Z',
        uid: 'tr-4',
        labels: {
          'tekton.dev/pipelineTask': 'deploy-helm',
          'tekton.dev/pipelineRun': 'build-and-deploy-service-prod-r9f2a'
        }
      },
      spec: {
        taskRef: { name: 'helm-upgrade' }
      },
      status: {
        startTime: '2026-11-20T03:19:12Z',
        completionTime: '2026-11-20T03:19:33Z',
        podName: 'pod-deploy-helm-04',
        conditions: [
          {
            type: 'Succeeded',
            status: 'True',
            reason: 'Succeeded',
            message: 'Release upgraded successfully'
          }
        ],
        steps: [
          {
            name: 'helm-upgrade',
            container: 'step-helm-upgrade',
            terminated: {
              exitCode: 0,
              reason: 'Completed',
              startedAt: '2026-11-20T03:19:13Z',
              finishedAt: '2026-11-20T03:19:32Z'
            }
          }
        ]
      }
    },
    {
      apiVersion: 'tekton.dev/v1',
      kind: 'TaskRun',
      metadata: {
        name: 'pr-3-trivy',
        namespace: 'default',
        creationTimestamp: '2026-11-20T02:00:02Z',
        uid: 'tr-5',
        labels: {
          'tekton.dev/pipelineTask': 'trivy-scan',
          'tekton.dev/pipelineRun': 'nightly-security-audit-err-x188'
        }
      },
      spec: {
        taskRef: { name: 'trivy' }
      },
      status: {
        startTime: '2026-11-20T02:00:03Z',
        completionTime: '2026-11-20T02:04:12Z',
        podName: 'pod-trivy-05',
        conditions: [
          {
            type: 'Succeeded',
            status: 'False',
            reason: 'Failed',
            message: 'Critical vulnerability found exit code 1'
          }
        ],
        steps: [
          {
            name: 'scan',
            container: 'step-trivy-scan',
            terminated: {
              exitCode: 1,
              reason: 'Error',
              startedAt: '2026-11-20T02:00:04Z',
              finishedAt: '2026-11-20T02:04:11Z'
            }
          }
        ]
      }
    }
  ]
};

export const mockPlatformMetrics = {
  total_projects: 3,
  total_pipelines: 4,
  total_runs: 18,
  running_runs: 1,
  succeeded_runs: 16,
  failed_runs: 1,
  success_rate: 94.1,
  avg_duration_seconds: 42.5,
  recent_runs: [
    {
      id: 'run-8f3a9b1c',
      pipeline_name: 'build-and-deploy-service',
      project_name: 'core-microservice',
      status: 'Succeeded',
      triggered_by: 'git-push',
      duration_seconds: 38,
      created_at: '2026-11-20T03:15:00Z'
    },
    {
      id: 'run-2c7e4d9f',
      pipeline_name: 'frontend-ci-cd',
      project_name: 'frontend-dashboard',
      status: 'Running',
      triggered_by: 'manual',
      duration_seconds: 14,
      created_at: '2026-11-20T03:32:10Z'
    },
    {
      id: 'run-1a5b8c3d',
      pipeline_name: 'security-scan-trivy',
      project_name: 'core-microservice',
      status: 'Failed',
      triggered_by: 'cron',
      duration_seconds: 120,
      created_at: '2026-11-20T02:00:00Z'
    }
  ]
};

export const mockPlatformProjects = [
  {
    id: 'proj-1',
    name: 'core-microservice',
    description: '核心 Golang 微服务应用及自动化 CI/CD 交付流水线',
    git_url: 'https://github.com/tektoncd/core-service.git',
    default_branch: 'main',
    namespace: 'production',
    created_at: '2026-11-18T08:00:00Z'
  },
  {
    id: 'proj-2',
    name: 'frontend-dashboard',
    description: '现代云原生 DevOps 控制台前端 (React 18 / Vite)',
    git_url: 'https://github.com/tektoncd/dashboard.git',
    default_branch: 'main',
    namespace: 'staging',
    created_at: '2026-11-19T10:00:00Z'
  },
  {
    id: 'proj-3',
    name: 'infra-gitops',
    description: 'Kubernetes 集群基础架构声明式配置与 Helm 发布',
    git_url: 'https://github.com/tektoncd/infra.git',
    default_branch: 'master',
    namespace: 'default',
    created_at: '2026-11-19T14:30:00Z'
  }
];

export const mockPlatformPipelines = [
  {
    id: 'pipe-1',
    project_id: 'proj-1',
    name: 'build-and-deploy-service',
    dsl_content:
      'name: build-and-deploy-service\nstages:\n  - name: test\n    tasks:\n      - name: unit-test\n        image: golang:1.24\n        script: go test ./...\n  - name: build\n    tasks:\n      - name: compile\n        image: alpine:latest\n        script: echo "Building packages..."\n',
    created_at: '2026-11-18T08:30:00Z'
  },
  {
    id: 'pipe-2',
    project_id: 'proj-2',
    name: 'frontend-ci-cd',
    dsl_content:
      'name: frontend-ci-cd\nstages:\n  - name: lint\n    tasks:\n      - name: eslint\n        image: node:20\n        script: npm run lint\n  - name: build\n    tasks:\n      - name: vite-build\n        image: node:20\n        script: npm run build\n',
    created_at: '2026-11-19T11:00:00Z'
  }
];

export const mockPlatformCredentials = [
  {
    id: 'cred-1',
    project_id: 'proj-1',
    name: 'github-access-token',
    type: 'git_token',
    masked_value: 'ghp_••••••••••••••••',
    created_at: '2026-11-18T09:00:00Z'
  },
  {
    id: 'cred-2',
    project_id: 'proj-1',
    name: 'docker-registry-secret',
    type: 'secret_text',
    masked_value: 'dckr_••••••••••••••••',
    created_at: '2026-11-18T09:15:00Z'
  }
];

export const mockLogs = {
  'pod-fetch-repo-01': `[2026-11-20T03:15:25Z] [clone] Initialized empty Git repository in /workspace/source/.git/
[2026-11-20T03:15:26Z] [clone] Connecting to git remote https://github.com/tektoncd/core-service.git
[2026-11-20T03:15:32Z] [clone] Fetching refs/heads/main...
[2026-11-20T03:15:58Z] [clone] From https://github.com/tektoncd/core-service
 * branch            main       -> FETCH_HEAD
[2026-11-20T03:16:01Z] [clone] Checking out commit 4e912b7a8c3d0 (HEAD -> main)
[2026-11-20T03:16:04Z] [clone] Successfully cloned revision 4e912b7a8c3d0 (84 files, 1.2MB).`,

  'pod-lint-test-02': `[2026-11-20T03:16:08Z] [unit-tests] Running: go version -> go1.24.0 linux/amd64
[2026-11-20T03:16:09Z] [unit-tests] Running static code analysis (golangci-lint run ./...)...
[2026-11-20T03:16:25Z] [unit-tests] 0 lint issues found across 32 packages.
[2026-11-20T03:16:26Z] [unit-tests] Running test suite with race detector (go test -v -race -cover ./...)...
[2026-11-20T03:16:35Z] [unit-tests] === RUN   TestPipelineReconciler
[2026-11-20T03:16:48Z] [unit-tests] --- PASS: TestPipelineReconciler (0.42s)
[2026-11-20T03:16:50Z] [unit-tests] === RUN   TestTaskRunValidation
[2026-11-20T03:17:10Z] [unit-tests] --- PASS: TestTaskRunValidation (0.15s)
[2026-11-20T03:17:20Z] [unit-tests] === RUN   TestAuthWebhookHandler
[2026-11-20T03:17:35Z] [unit-tests] --- PASS: TestAuthWebhookHandler (0.28s)
[2026-11-20T03:17:40Z] [unit-tests] PASS
[2026-11-20T03:17:41Z] [unit-tests] coverage: 91.8% of statements
[2026-11-20T03:17:41Z] [unit-tests] ok      github.com/tektoncd/core-service/pkg/reconciler  1.25s`,

  'pod-build-push-03': `[2026-11-20T03:17:45Z] [build-and-push] Retrieving base image golang:1.24-alpine...
[2026-11-20T03:17:52Z] [build-and-push] Step 1/6 : FROM golang:1.24-alpine AS builder
[2026-11-20T03:17:53Z] [build-and-push] Step 2/6 : WORKDIR /workspace
[2026-11-20T03:18:02Z] [build-and-push] Step 3/6 : COPY go.mod go.sum ./ && RUN go mod download
[2026-11-20T03:18:18Z] [build-and-push] Step 4/6 : COPY . . && RUN CGO_ENABLED=0 go build -o /bin/service ./cmd/server
[2026-11-20T03:18:40Z] [build-and-push] Step 5/6 : FROM gcr.io/distroless/static:nonroot
[2026-11-20T03:18:45Z] [build-and-push] Step 6/6 : COPY --from=builder /bin/service /bin/service
[2026-11-20T03:18:50Z] [build-and-push] Pushing image registry.internal/services/core-service:v2.4.0...
[2026-11-20T03:19:08Z] [build-and-push] Digest: sha256:7b92138901c2384917a23019df7c9412
[2026-11-20T03:19:09Z] [build-and-push] Image pushed successfully.`,

  'pod-deploy-helm-04': `[2026-11-20T03:19:13Z] [helm-upgrade] Checking connection to Kubernetes cluster... Connected.
[2026-11-20T03:19:15Z] [helm-upgrade] Upgrading release 'core-service' in namespace 'production'...
[2026-11-20T03:19:22Z] [helm-upgrade] Release "core-service" has been upgraded. Happy Helming!
[2026-11-20T03:19:24Z] [helm-upgrade] NAME: core-service
[2026-11-20T03:19:24Z] [helm-upgrade] LAST DEPLOYED: Fri Nov 20 03:19:23 2026
[2026-11-20T03:19:24Z] [helm-upgrade] STATUS: deployed
[2026-11-20T03:19:24Z] [helm-upgrade] REVISION: 42
[2026-11-20T03:19:30Z] [helm-upgrade] Waiting for deployment rollout to finish... 3 of 3 pods ready.
[2026-11-20T03:19:32Z] [helm-upgrade] Rollout completed successfully.`,

  'pod-trivy-05': `[2026-11-20T02:00:04Z] [scan] Updating vulnerability database...
[2026-11-20T02:00:40Z] [scan] Scanning image: registry.internal/base/node:18-alpine...
[2026-11-20T02:01:10Z] [scan] Found 14 vulnerabilities (1 CRITICAL, 3 HIGH, 8 MEDIUM, 2 LOW)
[2026-11-20T02:01:11Z] [scan] --------------------------------------------------------------------------------
[2026-11-20T02:01:11Z] [scan] CRITICAL: CVE-2026-4491 in libcrypto3 (fixed in 3.1.6-r0)
[2026-11-20T02:01:11Z] [scan] Description: Heap buffer overflow in ASN.1 decoding leads to remote code execution
[2026-11-20T02:01:12Z] [scan] --------------------------------------------------------------------------------
[2026-11-20T02:04:10Z] [scan] Error: Threshold violation -- Found 1 CRITICAL vulnerabilities (limit: 0)
[2026-11-20T02:04:11Z] [scan] Command failed with exit code 1.`
};
