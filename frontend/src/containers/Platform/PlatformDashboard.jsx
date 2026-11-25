/*
Copyright 2026 The Tekton Authors & Platform Team
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

import { useEffect, useState } from 'react';
import { useIntl } from 'react-intl';
import { useNavigate } from 'react-router';
import {
  Button,
  DataTable,
  InlineNotification,
  Loading,
  Modal,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow
} from '@carbon/react';
import {
  Dashboard as DashboardIcon,
  Folder as FolderIcon,
  ApiKey as KeyIcon,
  Launch as LaunchIcon,
  Renew as RefreshIcon,
  Rocket as RocketIcon,
  UserAdmin as ShieldIcon,
  CheckmarkOutline as SuccessIcon,
  Task as TaskIcon,
  Terminal as TerminalIcon
} from '@carbon/react/icons';
import { ALL_NAMESPACES, urls } from '@tektoncd/dashboard-utils';

import { useSelectedNamespace } from '../../api';
import { getDashboardMetrics, usePermissions } from '../../api/platform';

export function PlatformDashboard() {
  const intl = useIntl();
  /* eslint-disable formatjs/enforce-default-message */
  const t = (id, defaultMessage, values) =>
    intl.formatMessage({ id, defaultMessage }, values);
  /* eslint-enable formatjs/enforce-default-message */

  const navigate = useNavigate();
  const { selectedNamespace } = useSelectedNamespace();
  const perm = usePermissions(selectedNamespace);

  const [metrics, setMetrics] = useState(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState(null);
  const [logModalOpen, setLogModalOpen] = useState(false);
  const [activeLogRun, setActiveLogRun] = useState(null);
  const [logLines, setLogLines] = useState([]);
  const [logStreaming, setLogStreaming] = useState(false);

  const pipelinesPath =
    selectedNamespace && selectedNamespace !== ALL_NAMESPACES
      ? urls.pipelines.byNamespace({ namespace: selectedNamespace })
      : urls.pipelines.all();

  const tasksPath =
    selectedNamespace && selectedNamespace !== ALL_NAMESPACES
      ? urls.tasks.byNamespace({ namespace: selectedNamespace })
      : urls.tasks.all();

  const pipelineRunsPath =
    selectedNamespace && selectedNamespace !== ALL_NAMESPACES
      ? urls.pipelineRuns.byNamespace({ namespace: selectedNamespace })
      : urls.pipelineRuns.all();

  const openLiveLogViewer = run => {
    setActiveLogRun(run);
    setLogLines([]);
    setLogModalOpen(true);
    setLogStreaming(true);

    const es = new EventSource(`/api/platform/v1/runs/${run.id}/logs/sse`);
    es.onmessage = event => {
      if (event.data) {
        setLogLines(prev => [...prev, event.data]);
      }
    };
    es.onerror = () => {
      setLogStreaming(false);
      es.close();
    };
    window._activeDashboardLogES = es;
  };

  const closeLiveLogViewer = () => {
    if (window._activeDashboardLogES) {
      window._activeDashboardLogES.close();
      window._activeDashboardLogES = null;
    }
    setLogStreaming(false);
    setLogModalOpen(false);
  };

  useEffect(() => {
    return () => {
      if (window._activeDashboardLogES) {
        window._activeDashboardLogES.close();
        window._activeDashboardLogES = null;
      }
    };
  }, []);

  const fetchMetrics = async (isManual = false) => {
    try {
      if (isManual) {
        setRefreshing(true);
      } else {
        setLoading(true);
      }
      setError(null);
      const data = await getDashboardMetrics(selectedNamespace);
      setMetrics(data);
    } catch (err) {
      setError(
        err.message || t('platform.error.fetchMetrics', '加载大盘指标度量失败')
      );
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };

  useEffect(() => {
    fetchMetrics(false);
  }, [selectedNamespace]);

  const getStatusPill = status => {
    switch (status) {
      case 'Succeeded':
        return (
          <span className="status-pill status-succeeded">
            <span className="dot" /> {t('platform.status.succeeded', '成功')}
          </span>
        );
      case 'Running':
        return (
          <span className="status-pill status-running">
            <span className="dot" /> {t('platform.status.running', '运行中')}
          </span>
        );
      case 'Failed':
        return (
          <span className="status-pill status-failed">
            <span className="dot" /> {t('platform.status.failed', '失败')}
          </span>
        );
      case 'Cancelled':
        return (
          <span className="status-pill status-cancelled">
            <span className="dot" /> {t('platform.status.cancelled', '已取消')}
          </span>
        );
      default:
        return (
          <span className="status-pill status-cancelled">
            <span className="dot" />{' '}
            {status || t('platform.status.pending', '等待中')}
          </span>
        );
    }
  };

  const headers = [
    { key: 'run_id', header: t('platform.table.runId', 'PipelineRun 实例') },
    {
      key: 'pipeline_name',
      header: t('platform.table.pipeline', 'Tekton 流水线')
    },
    { key: 'namespace', header: t('platform.table.namespace', 'K8s 命名空间') },
    { key: 'status', header: t('platform.table.status', '状态') },
    { key: 'triggered_by', header: t('platform.table.triggeredBy', '触发源') },
    { key: 'duration', header: t('platform.table.duration', '执行耗时') },
    { key: 'created_at', header: t('platform.table.startedAt', '启动时间') },
    { key: 'actions', header: t('platform.table.actions', '操作') }
  ];

  const recentRunRows =
    metrics?.recent_runs?.map(run => ({
      id: run.id,
      run_id: (
        <code className="row-code-tag">
          {run.k8s_name || `#${run.id?.slice(0, 8)}`}
        </code>
      ),
      pipeline_name: (
        <span style={{ fontWeight: 600, color: '#1e293b' }}>
          {run.pipeline_name || 'unnamed'}
        </span>
      ),
      namespace: (
        <code className="row-code-tag">
          {run.namespace || run.project_name || 'default'}
        </code>
      ),
      status: getStatusPill(run.status),
      triggered_by: (
        <code className="row-code-tag">{run.triggered_by || 'manual'}</code>
      ),
      duration: run.duration_seconds ? `${run.duration_seconds}s` : '< 1s',
      created_at: new Date(run.created_at).toLocaleString(),
      actions: (
        <Button
          kind="ghost"
          onClick={() => openLiveLogViewer(run)}
          renderIcon={TerminalIcon}
          size="sm"
        >
          实时日志
        </Button>
      )
    })) || [];

  const nsCount =
    metrics?.total_namespaces ?? metrics?.total_projects ?? 2;
  const secretsCount =
    metrics?.total_secrets ?? metrics?.total_credentials ?? 0;

  return (
    <div className="platform-container">
      {/* 1. Hero Banner */}
      <div className="platform-hero">
        <div className="platform-hero-content">
          <h1>
            <DashboardIcon size={32} />{' '}
            {t('platform.title', 'Tekton 云原生 CI/CD 控制台')}
          </h1>
          <p>
            {t(
              'platform.subtitle',
              '基于 Kubernetes Namespace、Tekton 原生 CRD 与 RBAC 鉴权的持续交付大盘'
            )}
          </p>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '10px' }}>
            <div className="platform-status-badge">
              <span className="status-dot" />{' '}
              <span>
                当前作用域:{' '}
                <strong>
                  {selectedNamespace === ALL_NAMESPACES || !selectedNamespace
                    ? 'All Namespaces'
                    : selectedNamespace}
                </strong>
              </span>
            </div>
            <div className="platform-status-badge">
              <ShieldIcon size={14} />{' '}
              <span>
                身份: <strong>{perm.username}</strong> ({perm.k8s_cluster_role})
              </span>
            </div>
          </div>
        </div>
        <div className="platform-hero-actions">
          <Button
            className={refreshing ? 'is-spinning-icon' : ''}
            disabled={loading || refreshing}
            kind="ghost"
            onClick={() => fetchMetrics(true)}
            renderIcon={RefreshIcon}
            size="md"
          >
            {refreshing
              ? '刷新中...'
              : t('platform.refresh', '刷新')}
          </Button>
          <Button
            kind="secondary"
            onClick={() => navigate('/platform/namespaces')}
            renderIcon={FolderIcon}
            size="md"
          >
            {t('platform.namespaces', '命名空间')}
          </Button>
          <Button
            kind="primary"
            onClick={() => navigate(pipelinesPath)}
            renderIcon={RocketIcon}
            size="md"
          >
            {t('platform.launchPipeline', 'Tekton 流水线')}
          </Button>
        </div>
      </div>

      {error && (
        <InlineNotification
          kind="error"
          style={{ marginBottom: '20px' }}
          subtitle={error}
          title={t('dashboard.about.error', '发生错误')}
        />
      )}

      {loading && !metrics ? (
        <Loading
          description={t(
            'platform.loading.metrics',
            '正在同步集群 Tekton CRD 资源与流水线指标...'
          )}
          withOverlay={false}
        />
      ) : (
        <>
          {/* 2. Metric Stat Cards: K8s 命名空间、Tekton 流水线、任务库、K8s Secret 凭据 */}
          <div className="platform-stats-grid">
            <div
              className="platform-stat-card card-indigo is-clickable"
              onClick={() => navigate('/platform/namespaces')}
              onKeyDown={e =>
                e.key === 'Enter' && navigate('/platform/namespaces')
              }
              role="button"
              tabIndex={0}
            >
              <div className="stat-card-header">
                <span className="stat-label">
                  {t('platform.stats.namespaces', 'K8s 命名空间')}
                </span>
                <div className="stat-icon-wrapper">
                  <FolderIcon size={18} />
                </div>
              </div>
              <div className="stat-value">{nsCount}</div>
              <div className="stat-sub">
                <SuccessIcon size={14} style={{ color: '#10b981' }} />{' '}
                {t('platform.stats.namespacesSub', 'Kubernetes 资源与 RBAC 隔离空间')}
              </div>
            </div>

            <div
              className="platform-stat-card card-violet is-clickable"
              onClick={() => navigate(pipelinesPath)}
              onKeyDown={e => e.key === 'Enter' && navigate(pipelinesPath)}
              role="button"
              tabIndex={0}
            >
              <div className="stat-card-header">
                <span className="stat-label">
                  {t('platform.stats.pipelines', 'Tekton 流水线')}
                </span>
                <div className="stat-icon-wrapper">
                  <RocketIcon size={18} />
                </div>
              </div>
              <div className="stat-value">{metrics?.total_pipelines || 0}</div>
              <div className="stat-sub">
                <span>
                  {t('platform.stats.crdDefinitions', 'tekton.dev/v1 Pipeline 定义总数')}
                </span>
              </div>
            </div>

            <div
              className="platform-stat-card card-sky is-clickable"
              onClick={() => navigate(tasksPath)}
              onKeyDown={e => e.key === 'Enter' && navigate(tasksPath)}
              role="button"
              tabIndex={0}
            >
              <div className="stat-card-header">
                <span className="stat-label">
                  {t('platform.stats.tasks', '任务模板 (Tasks)')}
                </span>
                <div className="stat-icon-wrapper">
                  <TaskIcon size={18} />
                </div>
              </div>
              <div className="stat-value">{metrics?.total_tasks || 0}</div>
              <div className="stat-sub">
                <span>
                  {t('platform.stats.tasksDetail', 'tekton.dev/v1 Task 原子任务总数')}
                </span>
              </div>
            </div>

            <div
              className="platform-stat-card card-emerald is-clickable"
              onClick={() => navigate('/platform/credentials')}
              onKeyDown={e =>
                e.key === 'Enter' && navigate('/platform/credentials')
              }
              role="button"
              tabIndex={0}
            >
              <div className="stat-card-header">
                <span className="stat-label">
                  {t('platform.stats.credentials', 'K8s Secret 凭据')}
                </span>
                <div className="stat-icon-wrapper">
                  <KeyIcon size={18} />
                </div>
              </div>
              <div className="stat-value">{secretsCount}</div>
              <div className="stat-sub">
                <span>
                  {t(
                    'platform.stats.credentialsDetail',
                    '自动挂载 ServiceAccount 的构建密钥'
                  )}
                </span>
              </div>
            </div>
          </div>

          {/* 3. Visual Pipeline Flow DAG Preview */}
          <div className="platform-section-header">
            <div>
              <h2>
                {t('platform.workflow.title', '🚀 Tekton 原生流水线标准交付链路')}
              </h2>
              <div className="section-subtitle">
                {t(
                  'platform.workflow.subtitle',
                  '声明式 Task 与 Pipeline 拓扑编排，由 Tekton Pipelines 控制器驱动'
                )}
              </div>
            </div>
          </div>

          <div className="pipeline-dag-preview">
            <div className="dag-stage completed">
              <span style={{ color: '#10b981' }}>✓</span>{' '}
              {t('platform.workflow.step1', '1. 挂载 Secret & 检出源码 (git-clone)')}
            </div>
            <span className="dag-arrow">──▶</span>
            <div className="dag-stage completed">
              <span style={{ color: '#10b981' }}>✓</span>{' '}
              {t('platform.workflow.step2', '2. 单元测试与静态扫描 (unit-test)')}
            </div>
            <span className="dag-arrow">──▶</span>
            <div className="dag-stage active">
              <span
                className="status-pill status-running"
                style={{ padding: '0 4px', fontSize: '10px' }}
              >
                ●
              </span>{' '}
              {t('platform.workflow.step3', '3. 容器镜像构建推送 (kaniko / buildah)')}
            </div>
            <span className="dag-arrow">──▶</span>
            <div className="dag-stage">
              {t('platform.workflow.step4', '4. K8s 命名空间滚动发布')}
            </div>
            <span className="dag-arrow">──▶</span>
            <div className="dag-stage">
              {t('platform.workflow.step5', '5. Finally 结果收敛与通知')}
            </div>
          </div>

          {/* 4. Recent Executions Table */}
          <div className="platform-section-header">
            <div>
              <h2>
                {t('platform.recentExecutions.title', '最近 PipelineRun 执行记录')}
              </h2>
              <div className="section-subtitle">
                {t(
                  'platform.recentExecutions.subtitle',
                  '实时追踪集群内 Tekton PipelineRun 状态及容器输出日志流'
                )}
              </div>
            </div>
            <Button
              kind="ghost"
              onClick={() => navigate(pipelineRunsPath)}
              renderIcon={LaunchIcon}
              size="sm"
            >
              {t('platform.viewAllPipelines', '查看全部执行记录')}
            </Button>
          </div>

          {recentRunRows.length === 0 ? (
            <div className="platform-empty-state">
              <RocketIcon className="empty-icon" />
              <h3>{t('platform.noRunsTitle', '当前命名空间暂无 PipelineRun 执行记录')}</h3>
              <p>
                {t(
                  'platform.noRunsDesc',
                  '前往「Tekton 流水线」通过可视化向导组装 Task 并一键触发您的第一条 PipelineRun。'
                )}
              </p>
              <Button
                kind="primary"
                onClick={() => navigate(pipelinesPath)}
                renderIcon={RocketIcon}
                size="md"
              >
                {t('platform.launchFirst', '前往 Tekton 流水线')}
              </Button>
            </div>
          ) : (
            <div className="platform-table-wrapper">
              <DataTable headers={headers} rows={recentRunRows}>
                {({
                  rows,
                  headers: tableHeaders,
                  getTableProps,
                  getHeaderProps,
                  getRowProps
                }) => (
                  <Table {...getTableProps()}>
                    <TableHead>
                      <TableRow>
                        {tableHeaders.map(header => {
                          const { key, ...headerProps } = getHeaderProps({
                            header
                          });
                          return (
                            <TableHeader
                              key={key || header.key}
                              {...headerProps}
                            >
                              {header.header}
                            </TableHeader>
                          );
                        })}
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {rows.map(row => {
                        const { key, ...rowProps } = getRowProps({ row });
                        return (
                          <TableRow key={key || row.id} {...rowProps}>
                            {row.cells.map(cell => (
                              <TableCell key={cell.id}>{cell.value}</TableCell>
                            ))}
                          </TableRow>
                        );
                      })}
                    </TableBody>
                  </Table>
                )}
              </DataTable>
            </div>
          )}
        </>
      )}

      <Modal
        className="platform-modal"
        modalHeading={`实时执行日志 ${
          activeLogRun
            ? `(${activeLogRun.pipeline_name || 'Pipeline'} · ${
                activeLogRun.k8s_name || activeLogRun.id?.slice(0, 8)
              })`
            : ''
        }`}
        onRequestClose={closeLiveLogViewer}
        onRequestSubmit={closeLiveLogViewer}
        open={logModalOpen}
        passiveModal
      >
        <div className="live-log-terminal">
          <div className="terminal-status-bar">
            <span className="terminal-dot-group">
              <span className="tdot red" />
              <span className="tdot yellow" />
              <span className="tdot green" />
            </span>
            <span className="terminal-title">
              {logStreaming
                ? '● 正在实时接收容器输出流 (SSE)...'
                : '✓ 日志流传输完毕'}
            </span>
          </div>
          <pre className="terminal-body">
            {logLines.length === 0
              ? '正在连接日志流服务器...\n'
              : logLines.join('\n')}
          </pre>
        </div>
      </Modal>
    </div>
  );
}

export default PlatformDashboard;
