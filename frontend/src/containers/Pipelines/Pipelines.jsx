/*
Copyright 2019-2026 The Tekton Authors
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
/* istanbul ignore file */

import { useState } from 'react';
import { useLocation, useNavigate, useParams } from 'react-router';
import { useQueryClient } from '@tanstack/react-query';
import {
  Add as AddIcon,
  TrashCan as DeleteIcon,
  Renew as RenewIcon,
  Rocket as RocketIcon,
  PlayOutline as RunIcon,
  Playlist as RunsIcon
} from '@carbon/react/icons';
import { useIntl } from 'react-intl';
import {
  Button,
  InlineNotification,
  Modal,
  Select,
  SelectItem,
  TextArea,
  TextInput
} from '@carbon/react';
import yaml from 'yaml';
import {
  ALL_NAMESPACES,
  getFilters,
  urls,
  useTitleSync
} from '@tektoncd/dashboard-utils';
import {
  DeleteModal,
  FormattedDate,
  Link,
  Table
} from '@tektoncd/dashboard-components';

import ListPageLayout from '../ListPageLayout';
import {
  createPipelineRunRaw,
  createTektonPipeline,
  deletePipeline,
  useIsReadOnly,
  usePipelines,
  useSelectedNamespace,
  useTasks
} from '../../api';
import { usePermissions } from '../../api/platform';
import { keyBy } from '../../utils';

const PIPELINE_PRESETS = [
  {
    label: '多阶段 CI 全链路流水线 (Git + 测试 + 构建)',
    name: 'fullstack-ci-pipeline',
    description: '标准三阶段 Tekton 流水线：源码检出 -> 单元测试 -> 产物构建打包',
    workspaceName: 'shared-workspace',
    tasks: [
      {
        name: 'fetch-source',
        taskRefName: '',
        runAfter: '',
        stepImage: 'alpine/git:latest',
        stepScript:
          '#!/bin/sh\nset -e\necho "[Stage 1] Cloning source repository..."\ngit --version'
      },
      {
        name: 'unit-test',
        taskRefName: '',
        runAfter: 'fetch-source',
        stepImage: 'golang:1.22-alpine',
        stepScript:
          '#!/bin/sh\nset -e\necho "[Stage 2] Running unit tests and static analysis..."\ngo version'
      },
      {
        name: 'build-package',
        taskRefName: '',
        runAfter: 'unit-test',
        stepImage: 'alpine:latest',
        stepScript:
          '#!/bin/sh\nset -e\necho "[Stage 3] Packaging release artifact..."\necho "Build Succeeded!"'
      }
    ]
  },
  {
    label: 'Go 微服务编译流水线',
    name: 'go-microservice-pipeline',
    description: '编译 Go 服务二进制并执行单元测试套件',
    workspaceName: 'shared-workspace',
    tasks: [
      {
        name: 'go-test',
        taskRefName: '',
        runAfter: '',
        stepImage: 'golang:1.22-alpine',
        stepScript: '#!/bin/sh\nset -e\ngo version\necho "Running Go test suite..."'
      },
      {
        name: 'go-build',
        taskRefName: '',
        runAfter: 'go-test',
        stepImage: 'golang:1.22-alpine',
        stepScript: '#!/bin/sh\nset -e\necho "Building Go microservice binary..."'
      }
    ]
  },
  {
    label: '前端工程打包流水线',
    name: 'frontend-release-pipeline',
    description: '安装 Node 依赖并构建生产环境静态产物',
    workspaceName: '',
    tasks: [
      {
        name: 'npm-build',
        taskRefName: '',
        runAfter: '',
        stepImage: 'node:20-alpine',
        stepScript:
          '#!/bin/sh\nset -e\nnode -v\necho "Bundling production frontend assets..."'
      }
    ]
  }
];

function getFormattedResources({
  intl,
  isReadOnly,
  openDeleteModal,
  openQuickRunModal,
  resources
}) {
  return resources.map(pipeline => {
    const taskCount = pipeline.spec?.tasks?.length || 1;
    return {
      id: pipeline.metadata.uid,
      name: (
        <div>
          <Link
            to={urls.pipelines.byName({
              name: pipeline.metadata.name,
              namespace: pipeline.metadata.namespace
            })}
            title={pipeline.metadata.name}
          >
            {pipeline.metadata.name}
          </Link>
          <span
            style={{
              marginLeft: '8px',
              fontSize: '0.72rem',
              padding: '1px 7px',
              borderRadius: '999px',
              background: '#eef2ff',
              color: '#4338ca',
              fontWeight: 600
            }}
          >
            {taskCount} Tasks
          </span>
        </div>
      ),
      namespace: pipeline.metadata.namespace,
      createdTime: (
        <FormattedDate date={pipeline.metadata.creationTimestamp} relative />
      ),
      actions: (
        <>
          {!isReadOnly ? (
            <Button
              hasIconOnly
              iconDescription="一键运行此 Tekton 流水线 (Quick Run)"
              kind="ghost"
              onClick={() => openQuickRunModal(pipeline)}
              renderIcon={RunIcon}
              size="sm"
              tooltipPosition="left"
            />
          ) : null}
          <Button
            as={Link}
            hasIconOnly
            iconDescription={intl.formatMessage(
              {
                id: 'dashboard.resourceList.viewRuns',
                defaultMessage: 'View {kind} of {resource}'
              },
              { kind: 'PipelineRuns', resource: pipeline.metadata.name }
            )}
            kind="ghost"
            renderIcon={RunsIcon}
            size="sm"
            to={urls.pipelineRuns.byPipeline({
              namespace: pipeline.metadata.namespace,
              pipelineName: pipeline.metadata.name
            })}
            tooltipPosition="left"
          />
          {!isReadOnly ? (
            <Button
              className="tkn--danger"
              hasIconOnly
              iconDescription={intl.formatMessage({
                id: 'dashboard.actions.deleteButton',
                defaultMessage: 'Delete'
              })}
              kind="ghost"
              onClick={() =>
                openDeleteModal([{ id: pipeline.metadata.uid }], () => {})
              }
              renderIcon={DeleteIcon}
              size="sm"
              tooltipPosition="left"
            />
          ) : null}
        </>
      )
    };
  });
}

export function Pipelines() {
  const intl = useIntl();
  const location = useLocation();
  const navigate = useNavigate();
  const params = useParams();
  const queryClient = useQueryClient();
  const filters = getFilters(location);

  const { selectedNamespace } = useSelectedNamespace();
  const { namespace = selectedNamespace } = params;
  const perm = usePermissions(namespace);
  const sysReadOnly = useIsReadOnly();
  const isReadOnly = sysReadOnly || !perm.canWrite;

  const [cancelSelection, setCancelSelection] = useState(null);
  const [deleteError, setDeleteError] = useState(null);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [toBeDeleted, setToBeDeleted] = useState([]);

  // Create Tekton Pipeline Modal State (Multi-Task Visual Builder + YAML Dual Mode)
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [createMode, setCreateMode] = useState('form'); // 'form' | 'yaml'
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState(null);
  const [createSuccess, setCreateSuccess] = useState(null);
  const [pipelineForm, setPipelineForm] = useState({
    name: '',
    namespace:
      namespace && namespace !== ALL_NAMESPACES ? namespace : 'default',
    description: '',
    workspaceName: 'shared-workspace',
    tasks: [
      {
        name: 'fetch-source',
        taskRefName: '',
        runAfter: '',
        stepImage: 'alpine/git:latest',
        stepScript:
          '#!/bin/sh\nset -e\necho "1. Checking out source code..."\ngit --version'
      },
      {
        name: 'build-and-test',
        taskRefName: '',
        runAfter: 'fetch-source',
        stepImage: 'alpine:latest',
        stepScript:
          '#!/bin/sh\nset -e\necho "2. Running tests & building artifact..."\necho "Done!"'
      }
    ]
  });
  const [yamlText, setYamlText] = useState('');

  // Quick Run PipelineRun Modal State
  const [quickRunPipeline, setQuickRunPipeline] = useState(null);
  const [quickRunSA, setQuickRunSA] = useState('default');
  const [quickRunParams, setQuickRunParams] = useState({});
  const [triggeringRun, setTriggeringRun] = useState(false);

  useTitleSync({ page: 'Pipelines' });

  const {
    data: pipelines = [],
    error,
    isPending,
    refetch
  } = usePipelines({
    filters,
    namespace
  });

  const { data: availableTasks = [] } = useTasks({ namespace });

  function buildResourceObjectFromForm(form) {
    const taskEntries = (form.tasks || []).map((t, idx) => {
      const nodeName = (t.name || `step-${idx + 1}`).trim();
      const baseNode = t.taskRefName
        ? {
            name: nodeName,
            taskRef: {
              name: t.taskRefName
            }
          }
        : {
            name: nodeName,
            taskSpec: {
              steps: [
                {
                  name: 'execute',
                  image: (t.stepImage || 'alpine:latest').trim(),
                  script: t.stepScript || 'echo "Step completed"'
                }
              ]
            }
          };

      if (t.runAfter && t.runAfter.trim()) {
        baseNode.runAfter = [t.runAfter.trim()];
      }
      return baseNode;
    });

    const spec = {
      ...(form.description ? { description: form.description } : {}),
      ...(form.workspaceName
        ? {
            workspaces: [
              {
                name: form.workspaceName.trim(),
                description: '流水线各 Task 节点共享的工作空间目录'
              }
            ]
          }
        : {}),
      tasks: taskEntries
    };

    return {
      apiVersion: 'tekton.dev/v1',
      kind: 'Pipeline',
      metadata: {
        name: (form.name || 'my-tekton-pipeline').trim(),
        namespace: (form.namespace || 'default').trim()
      },
      spec
    };
  }

  function buildYamlFromForm(form) {
    return yaml.stringify(buildResourceObjectFromForm(form));
  }

  function openCreatePipelineModal() {
    const defaultNs =
      namespace && namespace !== ALL_NAMESPACES ? namespace : 'default';
    const preset = PIPELINE_PRESETS[0];
    const nextForm = {
      name: '',
      namespace: defaultNs,
      description: preset.description,
      workspaceName: preset.workspaceName,
      tasks: preset.tasks.map(t => ({ ...t }))
    };
    setPipelineForm(nextForm);
    setYamlText(buildYamlFromForm(nextForm));
    setCreateError(null);
    setCreateMode('form');
    setIsCreateModalOpen(true);
  }

  function applyPreset(preset) {
    const defaultNs =
      pipelineForm.namespace ||
      (namespace && namespace !== ALL_NAMESPACES ? namespace : 'default');
    const updated = {
      name: preset.name,
      namespace: defaultNs,
      description: preset.description,
      workspaceName: preset.workspaceName || '',
      tasks: preset.tasks.map(t => ({ ...t }))
    };
    setPipelineForm(updated);
    setYamlText(buildYamlFromForm(updated));
  }

  function updateTaskNode(index, field, value) {
    const nextTasks = pipelineForm.tasks.map((t, i) =>
      i === index ? { ...t, [field]: value } : t
    );
    const nextForm = { ...pipelineForm, tasks: nextTasks };
    setPipelineForm(nextForm);
  }

  function addTaskNode() {
    const prevTask =
      pipelineForm.tasks.length > 0
        ? pipelineForm.tasks[pipelineForm.tasks.length - 1].name
        : '';
    const nextIdx = pipelineForm.tasks.length + 1;
    const nextTasks = [
      ...pipelineForm.tasks,
      {
        name: `stage-${nextIdx}`,
        taskRefName: '',
        runAfter: prevTask,
        stepImage: 'alpine:latest',
        stepScript: `#!/bin/sh\nset -e\necho "Executing stage-${nextIdx}..."`
      }
    ];
    setPipelineForm({ ...pipelineForm, tasks: nextTasks });
  }

  function removeTaskNode(index) {
    if (pipelineForm.tasks.length <= 1) return;
    const removedName = pipelineForm.tasks[index].name;
    const nextTasks = pipelineForm.tasks
      .filter((_, i) => i !== index)
      .map(t => ({
        ...t,
        runAfter: t.runAfter === removedName ? '' : t.runAfter
      }));
    setPipelineForm({ ...pipelineForm, tasks: nextTasks });
  }

  async function handleCreatePipeline() {
    setCreateError(null);
    try {
      setCreating(true);
      if (createMode === 'yaml') {
        let parsed;
        try {
          parsed = yaml.parse(yamlText);
        } catch (parseErr) {
          setCreateError(`YAML 语法解析失败: ${parseErr.message}`);
          return;
        }
        if (!parsed?.metadata?.name) {
          setCreateError('YAML 中必须包含 metadata.name 字段');
          return;
        }
        await createTektonPipeline({
          rawResource: parsed,
          namespace: parsed.metadata.namespace || 'default'
        });
        setCreateSuccess(
          `Tekton 流水线 "${parsed.metadata.name}" 创建成功！`
        );
      } else {
        if (!pipelineForm.name.trim()) {
          setCreateError('请输入流水线名称（例如：fullstack-ci-pipeline）');
          return;
        }
        const rawResource = buildResourceObjectFromForm(pipelineForm);
        await createTektonPipeline({
          rawResource,
          namespace: rawResource.metadata.namespace
        });
        setCreateSuccess(
          `Tekton 流水线 "${rawResource.metadata.name}" (${rawResource.spec.tasks.length} 个阶段任务) 创建成功！`
        );
      }
      setIsCreateModalOpen(false);
      await queryClient.invalidateQueries({ queryKey: ['tekton.dev'] });
      if (typeof refetch === 'function') {
        refetch();
      }
    } catch (err) {
      let msg = err.message || '创建 Tekton 流水线失败';
      if (err.response && typeof err.response.text === 'function') {
        try {
          const text = await err.response.text();
          if (text) msg = text;
        } catch {
          // ignore
        }
      }
      setCreateError(msg);
    } finally {
      setCreating(false);
    }
  }

  function openQuickRunModal(pipeline) {
    const initialParams = {};
    (pipeline.spec?.params || []).forEach(p => {
      initialParams[p.name] = p.default || '';
    });
    setQuickRunParams(initialParams);
    setQuickRunSA('default');
    setQuickRunPipeline(pipeline);
  }

  async function handleQuickRunSubmit() {
    if (!quickRunPipeline) return;
    setTriggeringRun(true);
    try {
      const pName = quickRunPipeline.metadata.name;
      const pNS = quickRunPipeline.metadata.namespace || 'default';
      const runName = `${pName}-run-${Date.now().toString().slice(-5)}`;
      const paramList = Object.keys(quickRunParams).map(k => ({
        name: k,
        value: quickRunParams[k]
      }));
      const workspaceList = (quickRunPipeline.spec?.workspaces || []).map(
        w => ({
          name: w.name,
          emptyDir: {}
        })
      );

      const payload = {
        apiVersion: 'tekton.dev/v1',
        kind: 'PipelineRun',
        metadata: {
          name: runName,
          namespace: pNS,
          labels: {
            'tekton.dev/pipeline': pName
          }
        },
        spec: {
          pipelineRef: {
            name: pName
          },
          taskRunTemplate: {
            serviceAccountName: quickRunSA.trim() || 'default'
          },
          ...(paramList.length > 0 ? { params: paramList } : {}),
          ...(workspaceList.length > 0 ? { workspaces: workspaceList } : {})
        }
      };

      await createPipelineRunRaw({
        namespace: pNS,
        payload
      });
      setQuickRunPipeline(null);
      setCreateSuccess(
        `已成功触发 PipelineRun 实例 "${runName}"（使用 ServiceAccount: ${
          quickRunSA || 'default'
        }），可前往「流水线执行」或「平台大盘」查看实时容器日志！`
      );
      await queryClient.invalidateQueries({ queryKey: ['tekton.dev'] });
    } catch (err) {
      setDeleteError(err.message || '触发 PipelineRun 失败');
      setQuickRunPipeline(null);
    } finally {
      setTriggeringRun(false);
    }
  }

  function getError() {
    if (error) {
      return {
        error,
        title: intl.formatMessage({
          id: 'dashboard.pipelines.errorLoading',
          defaultMessage: 'Error loading Pipelines'
        })
      };
    }

    if (deleteError) {
      return {
        clear: () => setDeleteError(null),
        error: deleteError
      };
    }

    return null;
  }

  function closeDeleteModal() {
    setShowDeleteModal(false);
    setToBeDeleted([]);
  }

  function handleDeletePipeline(pipeline) {
    const { name, namespace: resourceNamespace } = pipeline.metadata;
    deletePipeline({ name, namespace: resourceNamespace }).catch(err => {
      err.response.text().then(text => {
        const statusCode = err.response.status;
        let errorMessage = `error code ${statusCode}`;
        if (text) {
          errorMessage = `${text} (error code ${statusCode})`;
        }
        setDeleteError(errorMessage);
      });
    });
  }

  async function handleDelete() {
    const deletions = toBeDeleted.map(resource =>
      handleDeletePipeline(resource)
    );
    closeDeleteModal();
    await Promise.all(deletions);
    cancelSelection();
  }

  function openDeleteModal(selectedRows, handleCancel) {
    const resourcesById = keyBy(pipelines, 'metadata.uid');
    const resourcesToBeDeleted = selectedRows.map(
      ({ id }) => resourcesById[id]
    );
    setShowDeleteModal(true);
    setToBeDeleted(resourcesToBeDeleted);
    setCancelSelection(() => handleCancel);
  }

  const batchActionButtons = isReadOnly
    ? []
    : [
        {
          onClick: openDeleteModal,
          text: intl.formatMessage({
            id: 'dashboard.actions.deleteButton',
            defaultMessage: 'Delete'
          }),
          icon: DeleteIcon
        }
      ];

  const initialHeaders = [
    {
      key: 'name',
      header: intl.formatMessage({
        id: 'dashboard.tableHeader.name',
        defaultMessage: 'Name'
      })
    },
    {
      key: 'namespace',
      header: 'Namespace'
    },
    {
      key: 'createdTime',
      header: intl.formatMessage({
        id: 'dashboard.tableHeader.createdTime',
        defaultMessage: 'Created'
      })
    },
    {
      key: 'actions',
      header: ''
    }
  ];

  const toolbarButtons = isReadOnly
    ? []
    : [
        {
          icon: AddIcon,
          onClick: openCreatePipelineModal,
          text: intl.formatMessage({
            id: 'dashboard.pipelines.createPipeline',
            defaultMessage: '新建 Tekton 流水线'
          })
        }
      ];

  return (
    <div className="platform-container">
      <div className="platform-hero">
        <div className="platform-hero-content">
          <h1>
            <RocketIcon size={28} />
            {intl.formatMessage({
              id: 'dashboard.pipelines.title',
              defaultMessage: 'Tekton 流水线 (Pipelines)'
            })}
          </h1>
          <p>
            {intl.formatMessage({
              id: 'dashboard.pipelines.description',
              defaultMessage:
                '基于 tekton.dev/v1 标准模型可视化串联任务库 Task、编排 DAG 依赖顺序并一键触发 PipelineRun。'
            })}
          </p>
        </div>
        <div className="platform-hero-actions">
          <Button
            disabled={isPending}
            kind="ghost"
            onClick={() => {
              queryClient.invalidateQueries({ queryKey: ['tekton.dev'] });
              if (typeof refetch === 'function') refetch();
            }}
            renderIcon={RenewIcon}
            size="md"
          >
            {intl.formatMessage({
              id: 'platform.refresh',
              defaultMessage: '刷新'
            })}
          </Button>
          {!isReadOnly && (
            <>
              <Button
                kind="secondary"
                onClick={() => navigate(urls.pipelineRuns.create())}
                renderIcon={RunIcon}
                size="md"
              >
                {intl.formatMessage({
                  id: 'dashboard.pipelineRuns.createPipelineRunButton',
                  defaultMessage: '高级触发执行'
                })}
              </Button>
              <Button
                kind="primary"
                onClick={openCreatePipelineModal}
                renderIcon={AddIcon}
                size="md"
              >
                {intl.formatMessage({
                  id: 'dashboard.pipelines.createPipeline',
                  defaultMessage: '新建 Tekton 流水线'
                })}
              </Button>
            </>
          )}
        </div>
      </div>

      {!perm.canWrite && (
        <InlineNotification
          kind="info"
          lowContrast
          style={{ marginBottom: '16px', maxWidth: '100%' }}
          subtitle={`当前登录用户 (${perm.username}) 在此命名空间下为只读角色 (${perm.k8s_cluster_role})，新建、删除与触发流水线按钮已自动隐藏。`}
          title="K8s RBAC 只读模式"
        />
      )}

      {createSuccess && (
        <InlineNotification
          kind="success"
          lowContrast
          onCloseButtonClick={() => setCreateSuccess(null)}
          statusIconDescription="成功"
          style={{ marginBottom: '16px', maxWidth: '100%' }}
          subtitle={createSuccess}
          title="操作成功"
        />
      )}

      <ListPageLayout
        error={getError()}
        filters={filters}
        resources={pipelines}
        title=""
      >
        {({ resources }) => (
          <>
            <Table
              batchActionButtons={batchActionButtons}
              className="tkn--table--inline-actions"
              emptyTextAllNamespaces={intl.formatMessage(
                {
                  id: 'dashboard.emptyState.allNamespaces',
                  defaultMessage: 'No matching {kind} found'
                },
                { kind: 'Pipelines' }
              )}
              emptyTextSelectedNamespace={intl.formatMessage(
                {
                  id: 'dashboard.emptyState.selectedNamespace',
                  defaultMessage:
                    'No matching {kind} found in namespace {selectedNamespace}'
                },
                { kind: 'Pipelines', selectedNamespace: namespace }
              )}
              headers={initialHeaders}
              loading={isPending}
              rows={getFormattedResources({
                intl,
                isReadOnly,
                openDeleteModal,
                openQuickRunModal,
                resources
              })}
              selectedNamespace={namespace}
              toolbarButtons={toolbarButtons}
            />
            {showDeleteModal ? (
              <DeleteModal
                kind="Pipelines"
                onClose={closeDeleteModal}
                onSubmit={handleDelete}
                resources={toBeDeleted}
                showNamespace={namespace === ALL_NAMESPACES}
              />
            ) : null}
          </>
        )}
      </ListPageLayout>

      {/* Multi-Task Visual DAG Builder & Native YAML Dual-Mode Modal */}
      <Modal
        className="platform-modal"
        modalHeading="新建 Tekton 流水线 (可视化 DAG 编排 + 原生 YAML)"
        onRequestClose={() => setIsCreateModalOpen(false)}
        onRequestSubmit={handleCreatePipeline}
        open={isCreateModalOpen}
        primaryButtonDisabled={creating}
        primaryButtonText={creating ? '正在创建...' : '创建 Tekton Pipeline'}
        secondaryButtonText="取消"
      >
        <div className="platform-modal-form">
          {createError && (
            <InlineNotification
              kind="error"
              lowContrast
              onCloseButtonClick={() => setCreateError(null)}
              statusIconDescription="错误"
              subtitle={createError}
              title="创建失败"
            />
          )}

          <div className="task-modal-toolbar">
            <div className="task-preset-group">
              <span className="preset-label">快速模板：</span>
              {PIPELINE_PRESETS.map(preset => (
                <button
                  className="task-preset-chip"
                  key={preset.name}
                  onClick={() => applyPreset(preset)}
                  type="button"
                >
                  {preset.label}
                </button>
              ))}
            </div>
            <div className="task-mode-switch">
              <button
                className={`mode-tab ${createMode === 'form' ? 'active' : ''}`}
                onClick={() => setCreateMode('form')}
                type="button"
              >
                可视化任务编排
              </button>
              <button
                className={`mode-tab ${createMode === 'yaml' ? 'active' : ''}`}
                onClick={() => {
                  if (createMode === 'form') {
                    setYamlText(buildYamlFromForm(pipelineForm));
                  }
                  setCreateMode('yaml');
                }}
                type="button"
              >
                原生 YAML 规格
              </button>
            </div>
          </div>

          {createMode === 'form' ? (
            <>
              <div className="form-grid-2col">
                <TextInput
                  id="tekton-pipe-name-input"
                  labelText="Pipeline 名称 (metadata.name) *"
                  onChange={e =>
                    setPipelineForm({ ...pipelineForm, name: e.target.value })
                  }
                  placeholder="例如: fullstack-ci-pipeline"
                  value={pipelineForm.name}
                />
                <TextInput
                  id="tekton-pipe-ns-input"
                  labelText="目标 K8s Namespace"
                  onChange={e =>
                    setPipelineForm({
                      ...pipelineForm,
                      namespace: e.target.value
                    })
                  }
                  placeholder="default"
                  value={pipelineForm.namespace}
                />
              </div>

              <div className="form-grid-2col">
                <TextInput
                  id="tekton-pipe-desc-input"
                  labelText="流水线描述 (spec.description)"
                  onChange={e =>
                    setPipelineForm({
                      ...pipelineForm,
                      description: e.target.value
                    })
                  }
                  placeholder="描述该 Tekton 流水线的构建与交付目标"
                  value={pipelineForm.description}
                />
                <TextInput
                  id="tekton-pipe-ws-input"
                  labelText="共享工作空间 (spec.workspaces，可选)"
                  onChange={e =>
                    setPipelineForm({
                      ...pipelineForm,
                      workspaceName: e.target.value
                    })
                  }
                  placeholder="例如: shared-workspace"
                  value={pipelineForm.workspaceName}
                />
              </div>

              {/* Multi-Task DAG Builder Section */}
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  marginTop: '4px'
                }}
              >
                <strong style={{ fontSize: '0.85rem', color: '#0f172a' }}>
                  流水线任务拓扑编排 (spec.tasks · 共{' '}
                  {pipelineForm.tasks.length} 个阶段节点)
                </strong>
                <Button
                  kind="tertiary"
                  onClick={addTaskNode}
                  renderIcon={AddIcon}
                  size="sm"
                >
                  添加任务节点
                </Button>
              </div>

              <div
                style={{
                  display: 'flex',
                  flexDirection: 'column',
                  gap: '12px',
                  maxHeight: '320px',
                  overflowY: 'auto',
                  paddingRight: '4px'
                }}
              >
                {pipelineForm.tasks.map((taskNode, idx) => (
                  <div
                    key={idx}
                    style={{
                      background: '#f8fafc',
                      border: '1px solid #cbd5e1',
                      borderRadius: '10px',
                      padding: '12px 14px'
                    }}
                  >
                    <div
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'space-between',
                        marginBottom: '10px'
                      }}
                    >
                      <span
                        style={{
                          fontSize: '0.78rem',
                          fontWeight: 700,
                          color: '#4f46e5',
                          background: '#eef2ff',
                          padding: '2px 10px',
                          borderRadius: '999px'
                        }}
                      >
                        阶段节点 #{idx + 1}: {taskNode.name || `step-${idx + 1}`}
                        {taskNode.runAfter
                          ? ` (runAfter: ${taskNode.runAfter})`
                          : ' (起始节点)'}
                      </span>
                      {pipelineForm.tasks.length > 1 && (
                        <Button
                          hasIconOnly
                          iconDescription="移除此任务节点"
                          kind="danger--ghost"
                          onClick={() => removeTaskNode(idx)}
                          renderIcon={DeleteIcon}
                          size="sm"
                        />
                      )}
                    </div>

                    <div className="form-grid-2col">
                      <TextInput
                        id={`pipe-task-name-${idx}`}
                        labelText="节点名称 (PipelineTask Name)"
                        onChange={e =>
                          updateTaskNode(idx, 'name', e.target.value)
                        }
                        placeholder={`stage-${idx + 1}`}
                        value={taskNode.name}
                      />
                      <Select
                        id={`pipe-task-runafter-${idx}`}
                        labelText="前置依赖 (runAfter 顺序控制)"
                        onChange={e =>
                          updateTaskNode(idx, 'runAfter', e.target.value)
                        }
                        value={taskNode.runAfter || ''}
                      >
                        <SelectItem
                          text="-- 无前置依赖 (起始/并行执行) --"
                          value=""
                        />
                        {pipelineForm.tasks
                          .filter((_, otherIdx) => otherIdx !== idx)
                          .map((other, oIdx) => (
                            <SelectItem
                              key={oIdx}
                              text={`等待完成后执行: ${other.name}`}
                              value={other.name}
                            />
                          ))}
                      </Select>
                    </div>

                    <div style={{ marginTop: '10px' }}>
                      <Select
                        id={`pipe-task-ref-${idx}`}
                        labelText="任务来源 (引用任务库 Task 或内联脚本)"
                        onChange={e =>
                          updateTaskNode(idx, 'taskRefName', e.target.value)
                        }
                        value={taskNode.taskRefName}
                      >
                        <SelectItem
                          text="-- 自定义内联容器镜像与脚本 (Inline taskSpec) --"
                          value=""
                        />
                        {availableTasks.map(t => (
                          <SelectItem
                            key={t.metadata?.uid || t.metadata?.name}
                            text={`引用任务库 Task: ${t.metadata?.name}`}
                            value={t.metadata?.name}
                          />
                        ))}
                      </Select>
                    </div>

                    {!taskNode.taskRefName && (
                      <div
                        className="form-grid-2col"
                        style={{ marginTop: '10px' }}
                      >
                        <TextInput
                          id={`pipe-task-img-${idx}`}
                          labelText="容器镜像 (image)"
                          onChange={e =>
                            updateTaskNode(idx, 'stepImage', e.target.value)
                          }
                          placeholder="alpine:latest"
                          value={taskNode.stepImage}
                        />
                        <TextInput
                          id={`pipe-task-scr-${idx}`}
                          labelText="执行命令 / 脚本 (script)"
                          onChange={e =>
                            updateTaskNode(idx, 'stepScript', e.target.value)
                          }
                          placeholder="echo 'Running task...'"
                          value={taskNode.stepScript}
                        />
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </>
          ) : (
            <div className="dsl-code-input">
              <TextArea
                helperText="100% 标准的 Tekton Pipeline (tekton.dev/v1) 资源定义，与可视化编排器实时双向同步。"
                id="tekton-pipe-yaml-input"
                labelText="Tekton Pipeline YAML 定义 (tekton.dev/v1)"
                onChange={e => setYamlText(e.target.value)}
                rows={14}
                value={yamlText}
              />
            </div>
          )}
        </div>
      </Modal>

      {/* Quick Run PipelineRun Modal */}
      <Modal
        className="platform-modal"
        modalHeading={
          quickRunPipeline
            ? `一键触发 Tekton PipelineRun: ${quickRunPipeline.metadata.name}`
            : '触发 PipelineRun'
        }
        onRequestClose={() => setQuickRunPipeline(null)}
        onRequestSubmit={handleQuickRunSubmit}
        open={!!quickRunPipeline}
        primaryButtonDisabled={triggeringRun}
        primaryButtonText={
          triggeringRun ? '正在下发 PipelineRun...' : '立即触发运行'
        }
        secondaryButtonText="取消"
      >
        {quickRunPipeline && (
          <div className="platform-modal-form">
            <div className="form-grid-2col">
              <TextInput
                disabled
                id="quickrun-pipeline-name"
                labelText="目标 Pipeline"
                value={quickRunPipeline.metadata.name}
              />
              <TextInput
                disabled
                id="quickrun-pipeline-ns"
                labelText="所属 Namespace"
                value={quickRunPipeline.metadata.namespace || 'default'}
              />
            </div>

            <TextInput
              helperText="自动加载该 ServiceAccount 下挂载的 Git / Docker Registry K8s Secret 凭据"
              id="quickrun-sa"
              labelText="运行身份 (taskRunTemplate.serviceAccountName)"
              onChange={e => setQuickRunSA(e.target.value)}
              placeholder="default"
              value={quickRunSA}
            />

            {(quickRunPipeline.spec?.params || []).map(param => (
              <TextInput
                helperText={param.description || ''}
                id={`quickrun-param-${param.name}`}
                key={param.name}
                labelText={`流水线参数: ${param.name}`}
                onChange={e =>
                  setQuickRunParams({
                    ...quickRunParams,
                    [param.name]: e.target.value
                  })
                }
                value={quickRunParams[param.name] || ''}
              />
            ))}
          </div>
        )}
      </Modal>
    </div>
  );
}

export default Pipelines;
