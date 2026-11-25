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
import { useLocation } from 'react-router';
import {
  Button,
  DataTable,
  InlineNotification,
  Loading,
  Modal,
  Select,
  SelectItem,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  TextArea,
  TextInput
} from '@carbon/react';
import {
  Add as AddIcon,
  TrashCan as DeleteIcon,
  Folder as FolderIcon,
  Play as PlayIcon,
  Renew as RefreshIcon,
  Rocket as RocketIcon
} from '@carbon/react/icons';

import {
  createPipeline,
  deletePipeline,
  getPipelines,
  getProjects,
  triggerPipelineRun
} from '../../api/platform';

const DEFAULT_DSL_TEMPLATE = `name: build-and-test
stages:
  - name: test
    tasks:
      - name: unit-test
        image: golang:1.24
        script: go test ./...
  - name: build
    tasks:
      - name: compile
        image: alpine:latest
        script: echo 'Building distribution packages...'
`;

export function PlatformPipelines() {
  const intl = useIntl();
  /* eslint-disable formatjs/enforce-default-message */
  const t = (id, defaultMessage, values) =>
    intl.formatMessage({ id, defaultMessage }, values);
  /* eslint-enable formatjs/enforce-default-message */

  const location = useLocation();
  const searchParams = new URLSearchParams(location.search);
  const initialProjectId = searchParams.get('projectId') || '';

  const [projects, setProjects] = useState([]);
  const [selectedProjectId, setSelectedProjectId] = useState(initialProjectId);
  const [pipelines, setPipelines] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [successMsg, setSuccessMsg] = useState(null);

  // Modal State
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [pipeName, setPipeName] = useState('');
  const [dslContent, setDslContent] = useState(DEFAULT_DSL_TEMPLATE);
  const [submitting, setSubmitting] = useState(false);

  // Live Log Modal State
  const [logModalOpen, setLogModalOpen] = useState(false);
  const [activeLogRun, setActiveLogRun] = useState(null);
  const [logLines, setLogLines] = useState([]);
  const [logStreaming, setLogStreaming] = useState(false);

  const openLiveLogViewer = (run, pipelineName) => {
    if (window._activePipelineLogES) {
      window._activePipelineLogES.close();
      window._activePipelineLogES = null;
    }
    setActiveLogRun({ ...run, pipelineName });
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
    window._activePipelineLogES = es;
  };

  const closeLiveLogViewer = () => {
    if (window._activePipelineLogES) {
      window._activePipelineLogES.close();
      window._activePipelineLogES = null;
    }
    setLogStreaming(false);
    setLogModalOpen(false);
  };

  useEffect(() => {
    return () => {
      if (window._activePipelineLogES) {
        window._activePipelineLogES.close();
        window._activePipelineLogES = null;
      }
    };
  }, []);

  useEffect(() => {
    const loadProjects = async () => {
      try {
        const projs = await getProjects();
        const list = Array.isArray(projs) ? projs : projs?.items || [];
        setProjects(list);
        if (!selectedProjectId && list.length > 0) {
          setSelectedProjectId(list[0].id);
        }
      } catch (err) {
        setError(
          err.message || t('platform.error.fetchProjects', '加载项目列表失败')
        );
      }
    };
    loadProjects();
  }, []);

  const fetchPipelines = async (projId = selectedProjectId) => {
    if (!projId) {
      setPipelines([]);
      return;
    }
    try {
      setLoading(true);
      setError(null);
      const res = await getPipelines(projId);
      const pipeList = Array.isArray(res) ? res : res?.items || [];
      setPipelines(pipeList);
    } catch (err) {
      setError(
        err.message || t('platform.error.fetchPipelines', '加载流水线列表失败')
      );
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (selectedProjectId) {
      fetchPipelines(selectedProjectId);
    }
  }, [selectedProjectId]);

  const handleCreatePipeline = async () => {
    if (!pipeName.trim()) {
      setError(t('platform.error.pipelineNameReq', '流水线名称不能为空'));
      return;
    }
    try {
      setSubmitting(true);
      setError(null);
      await createPipeline(selectedProjectId, {
        name: pipeName,
        dsl_content: dslContent
      });
      setIsModalOpen(false);
      setPipeName('');
      setDslContent(DEFAULT_DSL_TEMPLATE);
      setSuccessMsg(
        t(
          'platform.success.pipelineCreated',
          '流水线 ‘{name}’ 已成功编译并存入配置中心。',
          { name: pipeName }
        )
      );
      fetchPipelines();
    } catch (err) {
      setError(
        err.message || t('platform.error.createPipeline', '创建流水线失败')
      );
    } finally {
      setSubmitting(false);
    }
  };

  const handleTriggerRun = async (pipelineId, pipelineName) => {
    try {
      setError(null);
      const run = await triggerPipelineRun(pipelineId);
      setSuccessMsg(
        t(
          'platform.success.pipelineTriggered',
          '流水线已成功触发执行 (执行 ID: #{id})！',
          { id: run.id?.slice(0, 8) }
        )
      );
      if (run?.id) {
        openLiveLogViewer(run, pipelineName);
      }
    } catch (err) {
      setError(
        err.message || t('platform.error.triggerRun', '触发流水线执行失败')
      );
    }
  };

  const handleDelete = async id => {
    if (
      !window.confirm(
        t('platform.confirm.deletePipeline', '确定要删除此流水线编排定义吗？')
      )
    ) {
      return;
    }
    try {
      await deletePipeline(id);
      fetchPipelines();
    } catch (err) {
      setError(
        err.message || t('platform.error.deletePipeline', '删除流水线失败')
      );
    }
  };

  const headers = [
    { key: 'name', header: t('platform.table.pipeline', '流水线名称') },
    { key: 'created_at', header: t('platform.table.createdDate', '创建时间') },
    { key: 'actions', header: t('platform.table.actions', '操作') }
  ];

  const pipelineList = Array.isArray(pipelines)
    ? pipelines
    : pipelines?.items || [];
  const rows = pipelineList.map(p => ({
    id: p.id,
    name: (
      <div>
        <div style={{ fontWeight: 600, color: '#0f172a', fontSize: '0.95rem' }}>
          {p.name}
        </div>
        <div style={{ fontSize: '0.75rem', color: '#64748b' }}>
          ID: <span className="code-chip">{p.id.slice(0, 8)}</span>
        </div>
      </div>
    ),
    created_at: new Date(p.created_at).toLocaleString(),
    actions: (
      <div style={{ display: 'flex', gap: '8px' }}>
        <Button
          size="sm"
          kind="primary"
          renderIcon={PlayIcon}
          onClick={() => handleTriggerRun(p.id, p.name)}
        >
          {t('platform.run', '运行')}
        </Button>
        <Button
          size="sm"
          kind="danger--ghost"
          renderIcon={DeleteIcon}
          hasIconOnly
          iconDescription={t('platform.deletePipeline', '删除流水线')}
          onClick={() => handleDelete(p.id)}
        />
      </div>
    )
  }));

  return (
    <div className="platform-container">
      <div className="platform-hero">
        <div className="platform-hero-content">
          <h1>
            <RocketIcon size={32} />{' '}
            {t('platform.pipelines.title', '流水线编排中心')}
          </h1>
          <p>
            {t(
              'platform.pipelines.subtitle',
              '编写声明式 YAML 流水线工作流，自动编译为 Tekton Pipeline CRD 并在集群中执行。'
            )}
          </p>
        </div>
        <div className="platform-hero-actions">
          <Button
            size="md"
            kind="ghost"
            renderIcon={RefreshIcon}
            onClick={() => fetchPipelines()}
          >
            {t('platform.refresh', '刷新')}
          </Button>
          <Button
            size="md"
            kind="primary"
            renderIcon={AddIcon}
            onClick={() => setIsModalOpen(true)}
            disabled={!selectedProjectId}
          >
            {t('platform.pipelines.new', '新建流水线')}
          </Button>
        </div>
      </div>

      <div className="platform-filter-bar">
        <div className="filter-bar-left">
          <span className="filter-bar-label">
            <FolderIcon size={16} />
            {t('platform.pipelines.selectProject', '所属工作空间')}:
          </span>
          <div className="filter-bar-select">
            <Select
              id="project-selector"
              labelText=""
              hideLabel
              size="sm"
              value={selectedProjectId}
              onChange={e => setSelectedProjectId(e.target.value)}
            >
              {(Array.isArray(projects)
                ? projects
                : projects?.items || []
              ).map(p => (
                <SelectItem key={p.id} value={p.id} text={p.name} />
              ))}
            </Select>
          </div>
        </div>
      </div>

      {successMsg && (
        <InlineNotification
          kind="success"
          title={t('platform.notification.success', '操作成功')}
          subtitle={successMsg}
          onCloseButtonClick={() => setSuccessMsg(null)}
          style={{ marginBottom: '20px' }}
        />
      )}

      {error && (
        <InlineNotification
          kind="error"
          title={t('dashboard.about.error', '操作失败')}
          subtitle={error}
          style={{ marginBottom: '20px' }}
        />
      )}

      {loading ? (
        <Loading
          description={t('platform.loading.pipelines', '正在加载流水线配置...')}
          withOverlay={false}
        />
      ) : rows.length === 0 ? (
        <div className="platform-empty-state">
          <RocketIcon className="empty-icon" />
          <h3>{t('platform.pipelines.emptyTitle', '该项目下暂无流水线')}</h3>
          <p>
            {t(
              'platform.pipelines.emptyDesc',
              '使用声明式 Tekton DSL 模版快速创建您的第一个 CI/CD 流水线。'
            )}
          </p>
          <Button
            size="md"
            kind="primary"
            renderIcon={AddIcon}
            onClick={() => setIsModalOpen(true)}
            disabled={!selectedProjectId}
          >
            {t('platform.pipelines.create', '创建流水线')}
          </Button>
        </div>
      ) : (
        <div className="platform-table-wrapper">
          <DataTable rows={rows} headers={headers}>
            {({
              rows: tableRows,
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
                        <TableHeader key={key || header.key} {...headerProps}>
                          {header.header}
                        </TableHeader>
                      );
                    })}
                  </TableRow>
                </TableHead>
                <TableBody>
                  {tableRows.map(row => {
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

      <Modal
        className="platform-modal"
        size="md"
        open={isModalOpen}
        modalHeading={t('platform.pipelines.modalHeading', '新建声明式流水线')}
        primaryButtonText={
          submitting
            ? t('platform.submitting', '编译与保存中...')
            : t('platform.pipelines.create', '创建流水线')
        }
        secondaryButtonText={t('carbon.table.batch.cancel', '取消')}
        onRequestSubmit={handleCreatePipeline}
        onRequestClose={() => setIsModalOpen(false)}
        primaryButtonDisabled={submitting}
      >
        <div className="platform-modal-form">
          <TextInput
            id="pipe-name-input"
            labelText={t('platform.pipelines.name', '流水线名称')}
            placeholder={t(
              'platform.pipelines.namePlaceholder',
              '例如 ci-build-test'
            )}
            value={pipeName}
            onChange={e => setPipeName(e.target.value)}
            required
          />
          <TextArea
            id="pipe-dsl-input"
            className="dsl-code-input"
            labelText={t('platform.pipelines.dsl', '流水线 YAML 规格定义')}
            helperText={t(
              'platform.pipelines.dslHelper',
              '自动编译为原生 Tekton Pipeline 与 PipelineRun CRD 资源'
            )}
            rows={12}
            value={dslContent}
            onChange={e => setDslContent(e.target.value)}
            style={{
              fontFamily: 'ui-monospace, Menlo, Consolas, monospace',
              fontSize: '13px'
            }}
          />
        </div>
      </Modal>

      <Modal
        className="platform-modal"
        modalHeading={`实时执行日志 ${
          activeLogRun
            ? `(${activeLogRun.pipelineName || 'Pipeline'} #${activeLogRun.id?.slice(0, 8)})`
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

export default PlatformPipelines;
