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
import { useIntl } from 'react-intl';
import { useLocation, useMatches, useParams } from 'react-router';
import { useQueryClient } from '@tanstack/react-query';
import {
  Button,
  InlineNotification,
  Modal,
  TextArea,
  TextInput
} from '@carbon/react';
import {
  Add as AddIcon,
  Code as CodeIcon,
  TrashCan as DeleteIcon,
  Events as EventsIcon,
  Renew as RenewIcon
} from '@carbon/react/icons';
import yaml from 'yaml';
import {
  ALL_NAMESPACES,
  getFilters,
  urls,
  useTitleSync
} from '@tektoncd/dashboard-utils';
import { FormattedDate, Link, Table } from '@tektoncd/dashboard-components';

import ListPageLayout from '../ListPageLayout';
import {
  useAPIResource,
  useCustomResources,
  useExtensions,
  useIsReadOnly,
  useSelectedNamespace,
  useTenantNamespaces
} from '../../api';
import { deleteRequest, post } from '../../api/comms';
import { getKubeAPI } from '../../api/utils';
import { usePermissions } from '../../api/platform';

const TRIGGERS_META = {
  eventlisteners: {
    singular: 'EventListener',
    cnTitle: '事件监听 (EventListeners)',
    desc: '暴露集群 HTTP Webhook 接收端点，监听 GitHub / GitLab 等外部代码仓事件并分发给 Tekton Trigger。',
    defaultName: 'github-webhook-listener',
    buildSpec: form => ({
      serviceAccountName: form.serviceAccount || 'default',
      triggers: [
        {
          name: form.refName || 'github-push-trigger',
          triggerRef: form.refName || 'github-push-trigger'
        }
      ]
    })
  },
  triggers: {
    singular: 'Trigger',
    cnTitle: '触发器 (Triggers)',
    desc: '串联 TriggerBinding（提取 Webhook 参数）、Interceptor（事件过滤）与 TriggerTemplate（实例化流水线）。',
    defaultName: 'github-push-trigger',
    buildSpec: form => ({
      serviceAccountName: form.serviceAccount || 'default',
      bindings: [{ ref: form.bindingRef || 'github-push-binding' }],
      template: { ref: form.refName || 'ci-pipeline-template' }
    })
  },
  triggerbindings: {
    singular: 'TriggerBinding',
    cnTitle: '触发绑定 (TriggerBindings)',
    desc: '从外部 Webhook 请求的 JSON Body 或 HTTP Header 中提取字段并映射为流水线构建参数。',
    defaultName: 'github-push-binding',
    buildSpec: form => ({
      params: [
        {
          name: form.paramName || 'gitrevision',
          value: form.paramValue || '$(body.head_commit.id)'
        },
        {
          name: 'gitrepositoryurl',
          value: '$(body.repository.clone_url)'
        }
      ]
    })
  },
  clustertriggerbindings: {
    singular: 'ClusterTriggerBinding',
    cnTitle: '集群触发绑定 (ClusterTriggerBindings)',
    desc: '跨所有 Kubernetes Namespace 共享的集群级 Webhook 参数提取绑定定义。',
    defaultName: 'cluster-git-binding',
    buildSpec: form => ({
      params: [
        {
          name: form.paramName || 'gitbranch',
          value: form.paramValue || '$(body.ref)'
        }
      ]
    })
  },
  triggertemplates: {
    singular: 'TriggerTemplate',
    cnTitle: '触发模板 (TriggerTemplates)',
    desc: '定义当 Webhook 事件满足条件时，动态渲染并创建的 Tekton PipelineRun 或 TaskRun 资源模板。',
    defaultName: 'ci-pipeline-template',
    buildSpec: form => ({
      params: [
        { name: 'gitrevision', default: 'main' },
        {
          name: 'gitrepositoryurl',
          default: 'https://github.com/tektoncd/pipeline'
        }
      ],
      resourcetemplates: [
        {
          apiVersion: 'tekton.dev/v1',
          kind: 'PipelineRun',
          metadata: {
            generateName: `${form.refName || 'fullstack-ci-pipeline'}-webhook-`
          },
          spec: {
            pipelineRef: {
              name: form.refName || 'fullstack-ci-pipeline'
            }
          }
        }
      ]
    })
  },
  interceptors: {
    singular: 'Interceptor',
    cnTitle: '命名空间拦截器 (Interceptors)',
    desc: '在触发流水线前对 Webhook 请求执行 CEL 表达式过滤、GitHub/GitLab Secret 签名校验或负载转换。',
    defaultName: 'github-cel-filter',
    buildSpec: form => ({
      clientConfig: {
        service: {
          name: 'tekton-triggers-core-interceptors',
          namespace: 'tekton-pipelines'
        }
      },
      params: [
        {
          name: 'filter',
          value:
            form.paramValue || "header.match('X-GitHub-Event', 'push')"
        }
      ]
    })
  },
  clusterinterceptors: {
    singular: 'ClusterInterceptor',
    cnTitle: '集群拦截器 (ClusterInterceptors)',
    desc: '集群全局共享的 Webhook 事件拦截器插件（如 CEL、GitHub、GitLab、Bitbucket 核心拦截器）。',
    defaultName: 'custom-cel-interceptor',
    buildSpec: () => ({
      clientConfig: {
        service: {
          name: 'tekton-triggers-core-interceptors',
          namespace: 'tekton-pipelines'
        }
      }
    })
  }
};

export function ResourceListContainer() {
  const intl = useIntl();
  const location = useLocation();
  const queryClient = useQueryClient();

  const matches = useMatches();
  const params = useParams();

  const tenantNamespaces = useTenantNamespaces();
  const { data: extensions = [] } = useExtensions(
    {
      namespace: tenantNamespaces[0] || ALL_NAMESPACES
    },
    { disableWebSocket: true, retryOnMount: false }
  );

  const { namespace: namespaceParam } = params;
  const match = matches.at(-1);
  const handle = match.handle || {};
  let { group, kind, version } = handle;
  const { resourceURL, title } = handle;

  let extension;
  if (!(group && kind && version)) {
    ({ group, kind, version } = params);
    extension = extensions?.find(
      ({ apiGroup, apiVersion, name }) =>
        apiGroup === group && apiVersion === version && kind === name
    );
  }

  const { selectedNamespace } = useSelectedNamespace();
  const namespace = namespaceParam || selectedNamespace;
  const filters = getFilters(location);

  const perm = usePermissions(namespace);
  const sysReadOnly = useIsReadOnly();
  const canWrite =
    !sysReadOnly && perm.canWrite && perm.rules?.triggers?.create !== false;

  const triggerMeta = TRIGGERS_META[kind];

  useTitleSync({
    page: triggerMeta?.cnTitle || title || `${group}/${version}/${kind}`
  });

  const {
    data: apiResource,
    error: apiResourceError,
    isPending: isLoadingAPIResource
  } = useAPIResource({ group, kind, version }, { enabled: !!extension });
  const isNamespaced = extension
    ? !isLoadingAPIResource && apiResource?.namespaced
    : handle?.isNamespaced;

  if (extension && typeof apiResource?.namespaced !== 'undefined') {
    match.handle.isNamespaced = isNamespaced;
  }

  const {
    data: resources = [],
    error: resourcesError,
    isLoading: isLoadingResources,
    refetch
  } = useCustomResources(
    {
      filters,
      group,
      kind,
      namespace: isNamespaced ? namespace : null,
      version
    },
    {
      disableWebSocket: true,
      enabled: !extension || (!isLoadingAPIResource && !apiResourceError)
    }
  );

  // Create & YAML Viewer Modal State
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [createMode, setCreateMode] = useState('form'); // 'form' | 'yaml'
  const [formState, setFormState] = useState({
    name: '',
    namespace:
      namespace && namespace !== ALL_NAMESPACES ? namespace : 'default',
    serviceAccount: 'default',
    refName: 'fullstack-ci-pipeline',
    bindingRef: 'github-push-binding',
    paramName: 'gitrevision',
    paramValue: '$(body.head_commit.id)'
  });
  const [yamlText, setYamlText] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [opError, setOpError] = useState(null);
  const [opSuccess, setOpSuccess] = useState(null);
  const [viewYamlResource, setViewYamlResource] = useState(null);

  function buildResourceFromForm(f) {
    const singular = triggerMeta?.singular || title || kind;
    const spec = triggerMeta ? triggerMeta.buildSpec(f) : {};
    const obj = {
      apiVersion: `${group}/${version}`,
      kind: singular,
      metadata: {
        name: (f.name || triggerMeta?.defaultName || `new-${kind}`).trim(),
        ...(isNamespaced
          ? {
              namespace: (
                f.namespace ||
                (namespace && namespace !== ALL_NAMESPACES
                  ? namespace
                  : 'default')
              ).trim()
            }
          : {})
      },
      spec
    };
    return obj;
  }

  function openCreateModal() {
    const defaultNS =
      namespace && namespace !== ALL_NAMESPACES ? namespace : 'default';
    const nextForm = {
      name: triggerMeta?.defaultName || '',
      namespace: defaultNS,
      serviceAccount: 'default',
      refName:
        kind === 'eventlisteners'
          ? 'github-push-trigger'
          : kind === 'triggers'
            ? 'ci-pipeline-template'
            : 'fullstack-ci-pipeline',
      bindingRef: 'github-push-binding',
      paramName: 'gitrevision',
      paramValue:
        kind === 'interceptors'
          ? "header.match('X-GitHub-Event', 'push')"
          : '$(body.head_commit.id)'
    };
    setFormState(nextForm);
    setYamlText(yaml.stringify(buildResourceFromForm(nextForm)));
    setCreateMode('form');
    setOpError(null);
    setIsCreateOpen(true);
  }

  async function handleCreateSubmit() {
    setOpError(null);
    try {
      setSubmitting(true);
      let payload;
      if (createMode === 'yaml') {
        try {
          payload = yaml.parse(yamlText);
        } catch (pErr) {
          setOpError(`YAML 语法解析错误: ${pErr.message}`);
          setSubmitting(false);
          return;
        }
      } else {
        if (!formState.name.trim()) {
          setOpError('请输入资源名称 (metadata.name)');
          setSubmitting(false);
          return;
        }
        payload = buildResourceFromForm(formState);
      }

      const targetNS = isNamespaced
        ? payload?.metadata?.namespace || formState.namespace || 'default'
        : undefined;
      const uri = getKubeAPI({
        group,
        kind,
        params: { namespace: targetNS },
        version
      });
      await post(uri, payload);
      setIsCreateOpen(false);
      setOpSuccess(`已成功创建 ${payload.kind} "${payload.metadata?.name}"`);
      await queryClient.invalidateQueries({ queryKey: [group] });
      if (typeof refetch === 'function') refetch();
    } catch (err) {
      setOpError(err.message || '创建触发器资源失败');
    } finally {
      setSubmitting(false);
    }
  }

  async function handleDeleteResource(resource) {
    const rName = resource.metadata?.name;
    const rNS = resource.metadata?.namespace;
    if (!window.confirm(`确定要删除 ${resource.kind || kind} "${rName}" 吗？`)) {
      return;
    }
    try {
      setOpError(null);
      const uri = getKubeAPI({
        group,
        kind,
        params: { name: rName, namespace: isNamespaced ? rNS : undefined },
        version
      });
      await deleteRequest(uri);
      setOpSuccess(`已删除 ${rName}`);
      await queryClient.invalidateQueries({ queryKey: [group] });
      if (typeof refetch === 'function') refetch();
    } catch (err) {
      setOpError(err.message || '删除资源失败');
    }
  }

  function getError() {
    if (!apiResourceError && !resourcesError) {
      return null;
    }
    return {
      error: apiResourceError || resourcesError,
      title: intl.formatMessage(
        {
          id: 'dashboard.resourceList.errorLoading',
          defaultMessage: 'Error loading {type}'
        },
        { type: kind }
      )
    };
  }

  function getResourceURL({ name, resourceNamespace }) {
    if (resourceURL) {
      return resourceURL({
        group,
        kind,
        name,
        namespace: resourceNamespace,
        version
      });
    }

    return resourceNamespace
      ? urls.kubernetesResources.byName({
          group,
          kind,
          name,
          namespace: resourceNamespace,
          version
        })
      : urls.kubernetesResources.cluster({
          group,
          kind,
          name,
          version
        });
  }

  function getSummaryCell(resource) {
    const spec = resource.spec || {};
    if (kind === 'eventlisteners') {
      const url =
        resource.status?.address?.url ||
        `http://el-${resource.metadata?.name}.${
          resource.metadata?.namespace || 'default'
        }.svc.cluster.local:8080`;
      return <code>Webhook: {url}</code>;
    }
    if (kind === 'triggers') {
      const tpl = spec.template?.ref || spec.template?.name || '-';
      return <span>关联模板: <code>{tpl}</code></span>;
    }
    if (kind === 'triggerbindings' || kind === 'clustertriggerbindings') {
      const cnt = spec.params?.length || 0;
      return <span>提取参数: {cnt} 项</span>;
    }
    if (kind === 'triggertemplates') {
      const cnt = spec.resourcetemplates?.length || 0;
      return <span>资源模板: {cnt} 个 PipelineRun/TaskRun</span>;
    }
    return <code>{`${group}/${version}`}</code>;
  }

  return (
    <div className="platform-container">
      <div className="platform-hero">
        <div className="platform-hero-content">
          <h1>
            <EventsIcon size={28} />
            {triggerMeta?.cnTitle || title || `${group}/${version}/${kind}`}
          </h1>
          <p>
            {triggerMeta?.desc ||
              `管理 Kubernetes 集群内的 ${group}/${version} ${kind} 声明式资源定义。`}
          </p>
        </div>
        <div className="platform-hero-actions">
          <Button
            kind="ghost"
            onClick={() => {
              queryClient.invalidateQueries({ queryKey: [group] });
              if (typeof refetch === 'function') refetch();
            }}
            renderIcon={RenewIcon}
            size="md"
          >
            刷新
          </Button>
          {canWrite && (
            <Button
              kind="primary"
              onClick={openCreateModal}
              renderIcon={AddIcon}
              size="md"
            >
              新建 {triggerMeta?.singular || title || kind}
            </Button>
          )}
        </div>
      </div>

      {opError && (
        <InlineNotification
          kind="error"
          lowContrast
          onCloseButtonClick={() => setOpError(null)}
          style={{ marginBottom: '16px', maxWidth: '100%' }}
          subtitle={opError}
          title="操作失败"
        />
      )}

      {opSuccess && (
        <InlineNotification
          kind="success"
          lowContrast
          onCloseButtonClick={() => setOpSuccess(null)}
          style={{ marginBottom: '16px', maxWidth: '100%' }}
          subtitle={opSuccess}
          title="操作成功"
        />
      )}

      <ListPageLayout
        error={getError()}
        filters={filters}
        resources={resources}
        title=""
      >
        {({ resources: paginatedResources }) => (
          <Table
            className="tkn--table--inline-actions"
            emptyTextAllNamespaces={intl.formatMessage(
              {
                id: 'dashboard.emptyState.allNamespaces',
                defaultMessage: 'No matching {kind} found'
              },
              { kind: title || kind }
            )}
            emptyTextSelectedNamespace={intl.formatMessage(
              {
                id: 'dashboard.emptyState.selectedNamespace',
                defaultMessage:
                  'No matching {kind} found in namespace {selectedNamespace}'
              },
              { kind: title || kind, selectedNamespace: namespace }
            )}
            headers={[
              {
                key: 'name',
                header: intl.formatMessage({
                  id: 'dashboard.tableHeader.name',
                  defaultMessage: 'Name'
                })
              },
              isNamespaced
                ? {
                    key: 'namespace',
                    header: 'Namespace'
                  }
                : null,
              {
                key: 'summary',
                header: '配置摘要 / Webhook 端点'
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
            ].filter(Boolean)}
            loading={(extension && isLoadingAPIResource) || isLoadingResources}
            rows={paginatedResources.map(resource => {
              const {
                creationTimestamp,
                name,
                namespace: resourceNamespace,
                uid
              } = resource.metadata;

              let resourceLink;
              if (!extension?.disableResourceDetailsLinks) {
                resourceLink = getResourceURL({ name, resourceNamespace });
              }

              return {
                id: uid || `${resourceNamespace || 'cluster'}-${name}`,
                name: resourceLink ? (
                  <Link to={resourceLink} title={name}>
                    {name}
                  </Link>
                ) : (
                  name
                ),
                namespace: resourceNamespace,
                summary: getSummaryCell(resource),
                createdTime: (
                  <FormattedDate date={creationTimestamp} relative />
                ),
                actions: (
                  <>
                    <Button
                      hasIconOnly
                      iconDescription="查看 YAML 定义"
                      kind="ghost"
                      onClick={() => setViewYamlResource(resource)}
                      renderIcon={CodeIcon}
                      size="sm"
                      tooltipPosition="left"
                    />
                    {canWrite && (
                      <Button
                        className="tkn--danger"
                        hasIconOnly
                        iconDescription="删除此资源"
                        kind="ghost"
                        onClick={() => handleDeleteResource(resource)}
                        renderIcon={DeleteIcon}
                        size="sm"
                        tooltipPosition="left"
                      />
                    )}
                  </>
                )
              };
            })}
            selectedNamespace={isNamespaced ? namespace : ALL_NAMESPACES}
            toolbarButtons={
              canWrite
                ? [
                    {
                      icon: AddIcon,
                      onClick: openCreateModal,
                      text: `新建 ${triggerMeta?.singular || title || kind}`
                    }
                  ]
                : []
            }
          />
        )}
      </ListPageLayout>

      {/* Create Trigger Resource Modal (Form + YAML Dual Mode) */}
      <Modal
        className="platform-modal"
        modalHeading={`新建 ${triggerMeta?.cnTitle || title || kind}`}
        onRequestClose={() => setIsCreateOpen(false)}
        onRequestSubmit={handleCreateSubmit}
        open={isCreateOpen}
        primaryButtonDisabled={submitting}
        primaryButtonText={submitting ? '正在创建...' : '创建资源'}
        secondaryButtonText="取消"
      >
        <div className="platform-modal-form">
          <div className="task-modal-toolbar">
            <span style={{ fontSize: '0.78rem', color: '#64748b' }}>
              APIVersion: <code>{`${group}/${version}`}</code>
            </span>
            <div className="task-mode-switch">
              <button
                className={`mode-tab ${createMode === 'form' ? 'active' : ''}`}
                onClick={() => setCreateMode('form')}
                type="button"
              >
                向导配置
              </button>
              <button
                className={`mode-tab ${createMode === 'yaml' ? 'active' : ''}`}
                onClick={() => {
                  if (createMode === 'form') {
                    setYamlText(
                      yaml.stringify(buildResourceFromForm(formState))
                    );
                  }
                  setCreateMode('yaml');
                }}
                type="button"
              >
                原生 YAML
              </button>
            </div>
          </div>

          {createMode === 'form' ? (
            <>
              <div className="form-grid-2col">
                <TextInput
                  id="trigger-res-name"
                  labelText="资源名称 (metadata.name) *"
                  onChange={e =>
                    setFormState({ ...formState, name: e.target.value })
                  }
                  placeholder={triggerMeta?.defaultName || 'my-trigger-resource'}
                  value={formState.name}
                />
                {isNamespaced && (
                  <TextInput
                    id="trigger-res-ns"
                    labelText="所属 K8s Namespace"
                    onChange={e =>
                      setFormState({ ...formState, namespace: e.target.value })
                    }
                    placeholder="default"
                    value={formState.namespace}
                  />
                )}
              </div>

              {(kind === 'eventlisteners' || kind === 'triggers') && (
                <div className="form-grid-2col">
                  <TextInput
                    id="trigger-res-sa"
                    labelText="运行 ServiceAccount"
                    onChange={e =>
                      setFormState({
                        ...formState,
                        serviceAccount: e.target.value
                      })
                    }
                    placeholder="default"
                    value={formState.serviceAccount}
                  />
                  <TextInput
                    id="trigger-res-ref"
                    labelText={
                      kind === 'eventlisteners'
                        ? '关联 Trigger 名称 (triggerRef)'
                        : '关联 TriggerTemplate 名称 (template.ref)'
                    }
                    onChange={e =>
                      setFormState({ ...formState, refName: e.target.value })
                    }
                    value={formState.refName}
                  />
                </div>
              )}

              {kind === 'triggers' && (
                <TextInput
                  id="trigger-res-binding"
                  labelText="关联 TriggerBinding 名称 (bindings[].ref)"
                  onChange={e =>
                    setFormState({ ...formState, bindingRef: e.target.value })
                  }
                  value={formState.bindingRef}
                />
              )}

              {kind === 'triggertemplates' && (
                <TextInput
                  id="trigger-tpl-pipeline"
                  labelText="触发目标 Tekton Pipeline 名称 (pipelineRef.name)"
                  onChange={e =>
                    setFormState({ ...formState, refName: e.target.value })
                  }
                  placeholder="fullstack-ci-pipeline"
                  value={formState.refName}
                />
              )}

              {(kind === 'triggerbindings' ||
                kind === 'clustertriggerbindings' ||
                kind === 'interceptors') && (
                <div className="form-grid-2col">
                  <TextInput
                    id="trigger-param-name"
                    labelText={
                      kind === 'interceptors'
                        ? '拦截规则类型'
                        : '提取参数名 (param.name)'
                    }
                    onChange={e =>
                      setFormState({ ...formState, paramName: e.target.value })
                    }
                    value={formState.paramName}
                  />
                  <TextInput
                    id="trigger-param-val"
                    labelText={
                      kind === 'interceptors'
                        ? 'CEL 过滤表达式 (filter)'
                        : 'JSONPath 提取表达式 (param.value)'
                    }
                    onChange={e =>
                      setFormState({ ...formState, paramValue: e.target.value })
                    }
                    value={formState.paramValue}
                  />
                </div>
              )}
            </>
          ) : (
            <div className="dsl-code-input">
              <TextArea
                helperText={`标准 ${group}/${version} ${
                  triggerMeta?.singular || kind
                } 资源 YAML`}
                id="trigger-res-yaml"
                labelText="Tekton Triggers YAML 定义"
                onChange={e => setYamlText(e.target.value)}
                rows={13}
                value={yamlText}
              />
            </div>
          )}
        </div>
      </Modal>

      {/* View YAML Modal */}
      <Modal
        className="platform-modal"
        modalHeading={
          viewYamlResource
            ? `${viewYamlResource.kind}: ${viewYamlResource.metadata?.name}`
            : '资源 YAML'
        }
        onRequestClose={() => setViewYamlResource(null)}
        onRequestSubmit={() => setViewYamlResource(null)}
        open={!!viewYamlResource}
        passiveModal
      >
        {viewYamlResource && (
          <div className="live-log-terminal">
            <pre className="terminal-body">
              {yaml.stringify(viewYamlResource)}
            </pre>
          </div>
        )}
      </Modal>
    </div>
  );
}

export default ResourceListContainer;
