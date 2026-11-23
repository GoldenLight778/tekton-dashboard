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
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow
} from '@carbon/react';
import {
  Time as ClockIcon,
  Dashboard as DashboardIcon,
  Folder as FolderIcon,
  Launch as LaunchIcon,
  Renew as RefreshIcon,
  Rocket as RocketIcon,
  CheckmarkOutline as SuccessIcon
} from '@carbon/react/icons';

import { getDashboardMetrics } from '../../api/platform';

export function PlatformDashboard() {
  const intl = useIntl();
  /* eslint-disable formatjs/enforce-default-message */
  const t = (id, defaultMessage, values) =>
    intl.formatMessage({ id, defaultMessage }, values);
  /* eslint-enable formatjs/enforce-default-message */

  const navigate = useNavigate();
  const [metrics, setMetrics] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const fetchMetrics = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await getDashboardMetrics();
      setMetrics(data);
    } catch (err) {
      setError(
        err.message || t('platform.error.fetchMetrics', '加载大盘指标度量失败')
      );
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchMetrics();
  }, []);

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
    { key: 'run_id', header: t('platform.table.runId', '执行 ID') },
    {
      key: 'pipeline_name',
      header: t('platform.table.pipeline', '流水线名称')
    },
    { key: 'project_name', header: t('platform.table.workspace', '所属项目') },
    { key: 'status', header: t('platform.table.status', '状态') },
    { key: 'triggered_by', header: t('platform.table.triggeredBy', '触发源') },
    { key: 'duration', header: t('platform.table.duration', '执行耗时') },
    { key: 'created_at', header: t('platform.table.startedAt', '启动时间') }
  ];

  const recentRunRows =
    metrics?.recent_runs?.map(run => ({
      id: run.id,
      run_id: <span className="code-chip">#{run.id?.slice(0, 8)}</span>,
      pipeline_name: (
        <span style={{ fontWeight: 600, color: '#1e293b' }}>
          {run.pipeline_name || 'unnamed'}
        </span>
      ),
      project_name: run.project_name || 'default',
      status: getStatusPill(run.status),
      triggered_by: (
        <span className="code-chip">{run.triggered_by || 'manual'}</span>
      ),
      duration: run.duration_seconds ? `${run.duration_seconds}s` : '< 1s',
      created_at: new Date(run.created_at).toLocaleString()
    })) || [];

  return (
    <div className="platform-container">
      {/* 1. Hero Banner */}
      <div className="platform-hero">
        <div className="platform-hero-content">
          <h1>
            <DashboardIcon size={32} />{' '}
            {t('platform.title', 'DevOps 自动化交付控制台')}
          </h1>
          <p>
            {t(
              'platform.subtitle',
              '云原生 Tekton 流水线编排、容器构建与 GitOps 持续交付平台'
            )}
          </p>
          <div className="platform-status-badge">
            <span className="status-dot" />{' '}
            {t(
              'platform.status.online',
              'Tekton 执行引擎在线 • SQLite WAL 存储就绪'
            )}
          </div>
        </div>
        <div className="platform-hero-actions">
          <Button
            size="md"
            kind="ghost"
            renderIcon={RefreshIcon}
            onClick={fetchMetrics}
            disabled={loading}
          >
            {t('platform.refresh', '刷新')}
          </Button>
          <Button
            size="md"
            kind="secondary"
            renderIcon={FolderIcon}
            onClick={() => navigate('/platform/projects')}
          >
            {t('platform.projects', '项目管理')}
          </Button>
          <Button
            size="md"
            kind="primary"
            renderIcon={RocketIcon}
            onClick={() => navigate('/platform/pipelines')}
          >
            {t('platform.launchPipeline', '启动流水线')}
          </Button>
        </div>
      </div>

      {error && (
        <InlineNotification
          kind="error"
          title={t('dashboard.about.error', '发生错误')}
          subtitle={error}
          style={{ marginBottom: '20px' }}
        />
      )}

      {loading && !metrics ? (
        <Loading
          description={t(
            'platform.loading.metrics',
            '正在同步遥测数据与流水线指标...'
          )}
          withOverlay={false}
        />
      ) : (
        <>
          {/* 2. Metric Stat Cards */}
          <div className="platform-stats-grid">
            <div className="platform-stat-card card-indigo">
              <div className="stat-card-header">
                <span className="stat-label">
                  {t('platform.stats.workspaces', '项目空间')}
                </span>
                <div className="stat-icon-wrapper">
                  <FolderIcon size={18} />
                </div>
              </div>
              <div className="stat-value">{metrics?.total_projects || 0}</div>
              <div className="stat-sub">
                <SuccessIcon size={14} style={{ color: '#10b981' }} />{' '}
                {t('platform.stats.connectedRepos', '代码仓库集成')}
              </div>
            </div>

            <div className="platform-stat-card card-violet">
              <div className="stat-card-header">
                <span className="stat-label">
                  {t('platform.stats.pipelines', '流水线定义')}
                </span>
                <div className="stat-icon-wrapper">
                  <RocketIcon size={18} />
                </div>
              </div>
              <div className="stat-value">{metrics?.total_pipelines || 0}</div>
              <div className="stat-sub">
                <span>
                  {t('platform.stats.crdDefinitions', 'CRD 编排规则')}
                </span>
              </div>
            </div>

            <div className="platform-stat-card card-sky">
              <div className="stat-card-header">
                <span className="stat-label">
                  {t('platform.stats.runs', '执行实例')}
                </span>
                <div className="stat-icon-wrapper">
                  <ClockIcon size={18} />
                </div>
              </div>
              <div className="stat-value">{metrics?.total_runs || 0}</div>
              <div className="stat-sub">
                <span>
                  {t(
                    'platform.stats.runsDetail',
                    '{running} 个运行中, {succeeded} 个已完成',
                    {
                      running: metrics?.running_runs || 0,
                      succeeded: metrics?.succeeded_runs || 0
                    }
                  )}
                </span>
              </div>
            </div>

            <div className="platform-stat-card card-emerald">
              <div className="stat-card-header">
                <span className="stat-label">
                  {t('platform.stats.successRate', '构建成功率')}
                </span>
                <div className="stat-icon-wrapper">
                  <SuccessIcon size={18} />
                </div>
              </div>
              <div className="stat-value">
                {metrics?.success_rate !== undefined
                  ? `${metrics.success_rate.toFixed(1)}%`
                  : '100%'}
              </div>
              <div className="stat-sub">
                <span>
                  {t('platform.stats.avgDuration', '平均耗时')}:{' '}
                  {metrics?.avg_duration_seconds
                    ? `${metrics.avg_duration_seconds.toFixed(1)}s`
                    : t('platform.stats.instant', '秒级')}
                </span>
              </div>
            </div>
          </div>

          {/* 3. Visual Pipeline Flow DAG Preview */}
          <div className="platform-section-header">
            <div>
              <h2>
                {t('platform.workflow.title', '🚀 自动化交付流程 DAG 视图')}
              </h2>
              <div className="section-subtitle">
                {t(
                  'platform.workflow.subtitle',
                  '声明式流水线阶段图谱，由 Tekton 执行引擎驱动'
                )}
              </div>
            </div>
          </div>

          <div className="pipeline-dag-preview">
            <div className="dag-stage completed">
              <span style={{ color: '#10b981' }}>✓</span>{' '}
              {t('platform.workflow.step1', '1. 代码检出 (Git Clone)')}
            </div>
            <span className="dag-arrow">──▶</span>
            <div className="dag-stage completed">
              <span style={{ color: '#10b981' }}>✓</span>{' '}
              {t('platform.workflow.step2', '2. 单元测试与代码检查')}
            </div>
            <span className="dag-arrow">──▶</span>
            <div className="dag-stage active">
              <span
                className="status-pill status-running"
                style={{ padding: '0 4px', fontSize: '10px' }}
              >
                ●
              </span>{' '}
              {t('platform.workflow.step3', '3. 镜像构建与打包')}
            </div>
            <span className="dag-arrow">──▶</span>
            <div className="dag-stage">
              {t('platform.workflow.step4', '4. 集群发布部署')}
            </div>
            <span className="dag-arrow">──▶</span>
            <div className="dag-stage">
              {t('platform.workflow.step5', '5. 状态通知回传')}
            </div>
          </div>

          {/* 4. Recent Executions Table */}
          <div className="platform-section-header">
            <div>
              <h2>
                {t('platform.recentExecutions.title', '最近流水线执行记录')}
              </h2>
              <div className="section-subtitle">
                {t(
                  'platform.recentExecutions.subtitle',
                  '实时监控触发执行详情及步骤动作'
                )}
              </div>
            </div>
            <Button
              size="sm"
              kind="ghost"
              renderIcon={LaunchIcon}
              onClick={() => navigate('/platform/pipelines')}
            >
              {t('platform.viewAllPipelines', '查看所有流水线')}
            </Button>
          </div>

          {recentRunRows.length === 0 ? (
            <div className="platform-empty-state">
              <RocketIcon className="empty-icon" />
              <h3>{t('platform.noRunsTitle', '暂无流水线执行记录')}</h3>
              <p>
                {t(
                  'platform.noRunsDesc',
                  '通过创建您的第一个流水线配置，或从 Git 仓库触发自动化构建开始体验。'
                )}
              </p>
              <Button
                size="md"
                kind="primary"
                renderIcon={RocketIcon}
                onClick={() => navigate('/platform/pipelines')}
              >
                {t('platform.launchFirst', '立即创建流水线')}
              </Button>
            </div>
          ) : (
            <div className="platform-table-wrapper">
              <DataTable rows={recentRunRows} headers={headers}>
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
    </div>
  );
}

export default PlatformDashboard;
