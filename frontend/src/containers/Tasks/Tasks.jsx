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
import { useIntl } from 'react-intl';
import { useQueryClient } from '@tanstack/react-query';
import {
  Button,
  InlineNotification,
  Modal,
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
import {
  Add as AddIcon,
  TrashCan as DeleteIcon,
  Renew as RenewIcon,
  PlayOutline as RunIcon,
  Playlist as RunsIcon,
  Task as TaskIcon
} from '@carbon/react/icons';

import ListPageLayout from '../ListPageLayout';
import {
  createTask,
  deleteTask,
  useIsReadOnly,
  useSelectedNamespace,
  useTasks
} from '../../api';
import { usePermissions } from '../../api/platform';
import { keyBy } from '../../utils';

const TASK_PRESETS = [
  {
    label: 'Shell 基础任务',
    name: 'basic-shell-task',
    description: '执行通用 Shell 构建或检测脚本',
    stepName: 'run-script',
    image: 'alpine:latest',
    script: '#!/bin/sh\nset -e\necho "Starting basic task execution..."\nuname -a\necho "Task completed successfully!"'
  },
  {
    label: 'Git 代码检出',
    name: 'git-clone-task',
    description: '从远程 Git 仓库拉取源代码到工作空间',
    stepName: 'git-clone',
    image: 'alpine/git:latest',
    script: '#!/bin/sh\nset -e\ngit version\necho "Cloning source repository..."'
  },
  {
    label: 'Go 编译测试',
    name: 'golang-build-task',
    description: '执行 Go 单元测试与二进制编译构建',
    stepName: 'go-build',
    image: 'golang:1.22-alpine',
    script: '#!/bin/sh\nset -e\ngo version\ngo test ./...\ngo build -v ./...'
  },
  {
    label: 'Node 前端打包',
    name: 'nodejs-build-task',
    description: '安装依赖并执行前端工程生产环境打包',
    stepName: 'npm-build',
    image: 'node:20-alpine',
    script: '#!/bin/sh\nset -e\nnode -v\nnpm ci\nnpm run build'
  }
];

function getFormattedResources({
  intl,
  isReadOnly,
  openDeleteModal,
  resources
}) {
  return resources.map(task => ({
    id: task.metadata.uid,
    name: (
      <Link
        to={urls.tasks.byName({
          name: task.metadata.name,
          namespace: task.metadata.namespace
        })}
        title={task.metadata.name}
      >
        {task.metadata.name}
      </Link>
    ),
    namespace: task.metadata.namespace,
    createdTime: (
      <FormattedDate date={task.metadata.creationTimestamp} relative />
    ),
    actions: (
      <>
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
              openDeleteModal([{ id: task.metadata.uid }], () => {})
            }
            renderIcon={DeleteIcon}
            size="sm"
            tooltipAlignment="center"
            tooltipPosition="left"
          />
        ) : null}
        {!isReadOnly ? (
          <Button
            as={Link}
            hasIconOnly
            iconDescription={intl.formatMessage(
              {
                id: 'dashboard.actions.createRunButton',
                defaultMessage: 'Create {kind}'
              },
              { kind: 'TaskRun' }
            )}
            kind="ghost"
            renderIcon={RunIcon}
            size="sm"
            to={`${urls.taskRuns.create()}?${new URLSearchParams({
              kind: 'Task',
              namespace: task.metadata.namespace,
              taskName: task.metadata.name
            }).toString()}`}
            tooltipAlignment="center"
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
            { kind: 'TaskRuns', resource: task.metadata.name }
          )}
          kind="ghost"
          renderIcon={RunsIcon}
          size="sm"
          to={urls.taskRuns.byTask({
            namespace: task.metadata.namespace,
            taskName: task.metadata.name
          })}
          tooltipAlignment="center"
          tooltipPosition="left"
        />
      </>
    )
  }));
}

function Tasks() {
  const intl = useIntl();
  const location = useLocation();
  const navigate = useNavigate();
  const params = useParams();
  const queryClient = useQueryClient();

  const { selectedNamespace } = useSelectedNamespace();
  const { namespace = selectedNamespace } = params;

  const [deleteError, setDeleteError] = useState(null);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [toBeDeleted, setToBeDeleted] = useState([]);
  const [cancelSelection, setCancelSelection] = useState(null);
  const filters = getFilters(location);

  // Create Basic Task Modal State
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [createMode, setCreateMode] = useState('form'); // 'form' | 'yaml'
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState(null);
  const [createSuccess, setCreateSuccess] = useState(null);
  const [taskForm, setTaskForm] = useState({
    name: '',
    namespace:
      namespace && namespace !== ALL_NAMESPACES ? namespace : 'default',
    description: '',
    stepName: 'execute',
    image: 'alpine:latest',
    script:
      '#!/bin/sh\nset -e\necho "Executing Tekton basic task..."\necho "Done!"'
  });
  const [yamlText, setYamlText] = useState('');

  const sysReadOnly = useIsReadOnly();
  const perm = usePermissions(namespace);
  const isReadOnly = sysReadOnly || !perm.canWrite;

  const {
    data: tasks = [],
    error,
    isPending,
    refetch
  } = useTasks({
    filters,
    namespace
  });

  function openCreateTaskModal() {
    const defaultNs =
      namespace && namespace !== ALL_NAMESPACES ? namespace : 'default';
    const nextForm = {
      name: '',
      namespace: defaultNs,
      description: '通用基础构建任务',
      stepName: 'execute',
      image: 'alpine:latest',
      script:
        '#!/bin/sh\nset -e\necho "Executing Tekton basic task..."\necho "Done!"'
    };
    setTaskForm(nextForm);
    setYamlText(
      yaml.stringify({
        apiVersion: 'tekton.dev/v1',
        kind: 'Task',
        metadata: {
          name: 'my-basic-task',
          namespace: defaultNs
        },
        spec: {
          description: nextForm.description,
          steps: [
            {
              name: nextForm.stepName,
              image: nextForm.image,
              script: nextForm.script
            }
          ]
        }
      })
    );
    setCreateError(null);
    setCreateMode('form');
    setIsCreateModalOpen(true);
  }

  function applyPreset(preset) {
    const defaultNs =
      taskForm.namespace ||
      (namespace && namespace !== ALL_NAMESPACES ? namespace : 'default');
    const updated = {
      name: preset.name,
      namespace: defaultNs,
      description: preset.description,
      stepName: preset.stepName,
      image: preset.image,
      script: preset.script
    };
    setTaskForm(updated);
    setYamlText(
      yaml.stringify({
        apiVersion: 'tekton.dev/v1',
        kind: 'Task',
        metadata: {
          name: updated.name,
          namespace: updated.namespace
        },
        spec: {
          description: updated.description,
          steps: [
            {
              name: updated.stepName,
              image: updated.image,
              script: updated.script
            }
          ]
        }
      })
    );
  }

  async function handleCreateTask() {
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
        await createTask({
          rawResource: parsed,
          namespace: parsed.metadata.namespace || 'default'
        });
        setCreateSuccess(`基础任务 "${parsed.metadata.name}" 创建成功！`);
      } else {
        if (!taskForm.name.trim()) {
          setCreateError('请输入任务名称（例如：git-clone-task）');
          return;
        }
        await createTask({
          name: taskForm.name.trim(),
          namespace: taskForm.namespace.trim() || 'default',
          description: taskForm.description.trim(),
          stepName: taskForm.stepName.trim() || 'execute',
          image: taskForm.image.trim() || 'alpine:latest',
          script: taskForm.script
        });
        setCreateSuccess(`基础任务 "${taskForm.name.trim()}" 创建成功！`);
      }
      setIsCreateModalOpen(false);
      await queryClient.invalidateQueries({ queryKey: ['tekton.dev'] });
      if (typeof refetch === 'function') {
        refetch();
      }
    } catch (err) {
      let msg = err.message || '创建基础任务失败';
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

  function getError() {
    if (error) {
      return {
        error,
        title: intl.formatMessage({
          id: 'dashboard.tasks.errorLoading',
          defaultMessage: 'Error loading Tasks'
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

  useTitleSync({ page: 'Tasks' });

  function closeDeleteModal() {
    setShowDeleteModal(false);
    setToBeDeleted([]);
  }

  function handleDeleteTask(task) {
    deleteTask({
      name: task.metadata.name,
      namespace: task.metadata.namespace
    }).catch(err => {
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
    const deletions = toBeDeleted.map(resource => handleDeleteTask(resource));
    closeDeleteModal();
    await Promise.all(deletions);
    cancelSelection();
  }

  function openDeleteModal(selectedRows, handleCancelSelection) {
    const resourcesById = keyBy(tasks, 'metadata.uid');
    setShowDeleteModal(true);
    setToBeDeleted(selectedRows.map(({ id }) => resourcesById[id]));
    setCancelSelection(() => handleCancelSelection);
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
          onClick: openCreateTaskModal,
          text: intl.formatMessage({
            id: 'dashboard.tasks.createTask',
            defaultMessage: '新建基础任务'
          })
        }
      ];

  return (
    <div className="platform-container">
      <div className="platform-hero">
        <div className="platform-hero-content">
          <h1>
            <TaskIcon size={28} />
            {intl.formatMessage({
              id: 'dashboard.tasks.title',
              defaultMessage: '任务库 (Tasks)'
            })}
          </h1>
          <p>
            {intl.formatMessage({
              id: 'dashboard.tasks.description',
              defaultMessage:
                '创建、管理与复用基础 Tekton 任务模板（Task）及容器步骤执行规格。'
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
                onClick={() => {
                  let queryString;
                  if (namespace !== ALL_NAMESPACES) {
                    queryString = new URLSearchParams({ namespace }).toString();
                  }
                  navigate(
                    urls.taskRuns.create() +
                      (queryString ? `?${queryString}` : '')
                  );
                }}
                renderIcon={RunIcon}
                size="md"
              >
                {intl.formatMessage({
                  id: 'dashboard.tasks.createTaskRun',
                  defaultMessage: '执行任务 (TaskRun)'
                })}
              </Button>
              <Button
                kind="primary"
                onClick={openCreateTaskModal}
                renderIcon={AddIcon}
                size="md"
              >
                {intl.formatMessage({
                  id: 'dashboard.tasks.createTask',
                  defaultMessage: '新建基础任务'
                })}
              </Button>
            </>
          )}
        </div>
      </div>

      {createSuccess && (
        <InlineNotification
          kind="success"
          lowContrast
          onCloseButtonClick={() => setCreateSuccess(null)}
          statusIconDescription="成功"
          subtitle={createSuccess}
          style={{ marginBottom: '16px' }}
          title="操作成功"
        />
      )}
      <ListPageLayout
        error={getError()}
        filters={filters}
        resources={tasks}
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
                { kind: 'Tasks' }
              )}
              emptyTextSelectedNamespace={intl.formatMessage(
                {
                  id: 'dashboard.emptyState.selectedNamespace',
                  defaultMessage:
                    'No matching {kind} found in namespace {selectedNamespace}'
                },
                { kind: 'Tasks', selectedNamespace: namespace }
              )}
              headers={initialHeaders}
              loading={isPending}
              rows={getFormattedResources({
                intl,
                isReadOnly,
                openDeleteModal,
                resources
              })}
              selectedNamespace={namespace}
              toolbarButtons={toolbarButtons}
            />
            {showDeleteModal ? (
              <DeleteModal
                danger
                kind="Tasks"
                onClose={closeDeleteModal}
                onSubmit={handleDelete}
                resources={toBeDeleted}
                showNamespace={namespace === ALL_NAMESPACES}
              />
            ) : null}
          </>
        )}
      </ListPageLayout>

      <Modal
        className="platform-modal"
        modalHeading="新建基础任务 (Tekton Task)"
        onRequestClose={() => setIsCreateModalOpen(false)}
        onRequestSubmit={handleCreateTask}
        open={isCreateModalOpen}
        primaryButtonDisabled={creating}
        primaryButtonText={creating ? '正在创建...' : '创建基础任务'}
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
              {TASK_PRESETS.map(preset => (
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
                表单配置
              </button>
              <button
                className={`mode-tab ${createMode === 'yaml' ? 'active' : ''}`}
                onClick={() => {
                  if (createMode === 'form') {
                    setYamlText(
                      yaml.stringify({
                        apiVersion: 'tekton.dev/v1',
                        kind: 'Task',
                        metadata: {
                          name: taskForm.name || 'my-basic-task',
                          namespace: taskForm.namespace || 'default'
                        },
                        spec: {
                          ...(taskForm.description
                            ? { description: taskForm.description }
                            : {}),
                          steps: [
                            {
                              name: taskForm.stepName || 'execute',
                              image: taskForm.image || 'alpine:latest',
                              script: taskForm.script
                            }
                          ]
                        }
                      })
                    );
                  }
                  setCreateMode('yaml');
                }}
                type="button"
              >
                YAML 规格
              </button>
            </div>
          </div>

          {createMode === 'form' ? (
            <>
              <div className="form-grid-2col">
                <TextInput
                  id="task-name-input"
                  labelText="任务名称 (Metadata Name) *"
                  onChange={e =>
                    setTaskForm({ ...taskForm, name: e.target.value })
                  }
                  placeholder="例如: git-clone-task"
                  value={taskForm.name}
                />
                <TextInput
                  id="task-namespace-input"
                  labelText="命名空间 (Namespace)"
                  onChange={e =>
                    setTaskForm({ ...taskForm, namespace: e.target.value })
                  }
                  placeholder="default"
                  value={taskForm.namespace}
                />
              </div>
              <TextInput
                id="task-desc-input"
                labelText="任务用途描述"
                onChange={e =>
                  setTaskForm({ ...taskForm, description: e.target.value })
                }
                placeholder="简述该基础任务的功能，供流水线编排复用"
                value={taskForm.description}
              />
              <div className="form-grid-2col">
                <TextInput
                  id="task-step-name-input"
                  labelText="步骤名称 (Step Name)"
                  onChange={e =>
                    setTaskForm({ ...taskForm, stepName: e.target.value })
                  }
                  placeholder="execute"
                  value={taskForm.stepName}
                />
                <TextInput
                  id="task-image-input"
                  labelText="执行容器镜像 (Container Image) *"
                  onChange={e =>
                    setTaskForm({ ...taskForm, image: e.target.value })
                  }
                  placeholder="例如: alpine:latest 或 golang:1.22"
                  value={taskForm.image}
                />
              </div>
              <div className="dsl-code-input">
                <TextArea
                  helperText="支持标准 Shell / Sh 脚本，将在步骤容器中自动挂载执行。"
                  id="task-script-input"
                  labelText="步骤执行脚本 (Step Script)"
                  onChange={e =>
                    setTaskForm({ ...taskForm, script: e.target.value })
                  }
                  rows={6}
                  value={taskForm.script}
                />
              </div>
            </>
          ) : (
            <div className="dsl-code-input">
              <TextArea
                helperText="直接编写标准 Tekton Task (tekton.dev/v1) 资源定义，支持多 Steps、Params 与 Workspaces。"
                id="task-yaml-input"
                labelText="Tekton Task YAML 定义"
                onChange={e => setYamlText(e.target.value)}
                rows={13}
                value={yamlText}
              />
            </div>
          )}
        </div>
      </Modal>
    </div>
  );
}

export default Tasks;
