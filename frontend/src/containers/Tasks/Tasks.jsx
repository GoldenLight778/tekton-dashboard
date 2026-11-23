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
import { Button } from '@carbon/react';
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
  deleteTask,
  useIsReadOnly,
  useSelectedNamespace,
  useTasks
} from '../../api';
import { keyBy } from '../../utils';

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

  const { selectedNamespace } = useSelectedNamespace();
  const { namespace = selectedNamespace } = params;

  const [deleteError, setDeleteError] = useState(null);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [toBeDeleted, setToBeDeleted] = useState([]);
  const [cancelSelection, setCancelSelection] = useState(null);
  const filters = getFilters(location);

  const isReadOnly = useIsReadOnly();

  const {
    data: tasks = [],
    error,
    isPending
  } = useTasks({
    filters,
    namespace
  });

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
          onClick: () => {
            let queryString;
            if (namespace !== ALL_NAMESPACES) {
              queryString = new URLSearchParams({ namespace }).toString();
            }
            navigate(
              urls.taskRuns.create() + (queryString ? `?${queryString}` : '')
            );
          },
          text: intl.formatMessage({
            id: 'dashboard.tasks.createTaskRun',
            defaultMessage: '创建任务执行'
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
                '管理与浏览 Kubernetes 集群中已注册的 Tekton 任务定义及步骤执行规格。'
            })}
          </p>
        </div>
        <div className="platform-hero-actions">
          <Button
            disabled={isPending}
            kind="ghost"
            onClick={() => window.location.reload()}
            renderIcon={RenewIcon}
            size="md"
          >
            {intl.formatMessage({
              id: 'platform.refresh',
              defaultMessage: '刷新'
            })}
          </Button>
          {!isReadOnly && (
            <Button
              kind="primary"
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
              renderIcon={AddIcon}
              size="md"
            >
              {intl.formatMessage({
                id: 'dashboard.tasks.createTaskRun',
                defaultMessage: '创建任务执行'
              })}
            </Button>
          )}
        </div>
      </div>
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
    </div>
  );
}

export default Tasks;
