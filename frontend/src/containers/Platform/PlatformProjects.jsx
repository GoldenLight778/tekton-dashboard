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
  TableRow,
  TextArea,
  TextInput
} from '@carbon/react';
import {
  Add as AddIcon,
  TrashCan as DeleteIcon,
  Folder as FolderIcon,
  Launch as LaunchIcon,
  Renew as RefreshIcon
} from '@carbon/react/icons';

import { createProject, deleteProject, getProjects } from '../../api/platform';

export function PlatformProjects() {
  const intl = useIntl();
  /* eslint-disable formatjs/enforce-default-message */
  const t = (id, defaultMessage, values) =>
    intl.formatMessage({ id, defaultMessage }, values);
  /* eslint-enable formatjs/enforce-default-message */

  const navigate = useNavigate();
  const [projects, setProjects] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  // Form State
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    git_url: '',
    default_branch: 'main',
    namespace: 'default'
  });

  const fetchProjects = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await getProjects();
      setProjects(data || []);
    } catch (err) {
      setError(
        err.message || t('platform.error.fetchProjects', '加载项目列表失败')
      );
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchProjects();
  }, []);

  const handleCreate = async () => {
    if (!formData.name.trim()) {
      setError(t('platform.error.projectNameReq', '项目名称不能为空'));
      return;
    }
    try {
      setSubmitting(true);
      setError(null);
      await createProject(formData);
      setIsModalOpen(false);
      setFormData({
        name: '',
        description: '',
        git_url: '',
        default_branch: 'main',
        namespace: 'default'
      });
      fetchProjects();
    } catch (err) {
      setError(
        err.message || t('platform.error.createProject', '创建项目失败')
      );
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async id => {
    if (
      !window.confirm(
        t('platform.confirm.deleteProject', '确定要删除此工作空间项目吗？')
      )
    ) {
      return;
    }
    try {
      await deleteProject(id);
      fetchProjects();
    } catch (err) {
      setError(
        err.message || t('platform.error.deleteProject', '删除项目失败')
      );
    }
  };

  const headers = [
    { key: 'name', header: t('platform.table.workspace', '项目 / 工作空间') },
    { key: 'git_url', header: t('platform.table.gitRepo', 'Git 仓库') },
    { key: 'default_branch', header: t('platform.table.branch', '分支') },
    { key: 'namespace', header: t('platform.table.namespace', 'K8s 命名空间') },
    { key: 'actions', header: t('platform.table.actions', '操作') }
  ];

  const rows = projects.map(proj => ({
    id: proj.id,
    name: (
      <div>
        <div style={{ fontWeight: 600, color: '#0f172a', fontSize: '0.95rem' }}>
          {proj.name}
        </div>
        {proj.description && (
          <div
            style={{ fontSize: '0.8rem', color: '#64748b', marginTop: '2px' }}
          >
            {proj.description}
          </div>
        )}
      </div>
    ),
    git_url: proj.git_url ? (
      <span className="code-chip">{proj.git_url}</span>
    ) : (
      <span style={{ color: '#94a3b8' }}>
        {t('platform.table.noRepo', '未绑定仓库')}
      </span>
    ),
    default_branch: (
      <span className="code-chip">{proj.default_branch || 'main'}</span>
    ),
    namespace: <span className="code-chip">{proj.namespace || 'default'}</span>,
    actions: (
      <div style={{ display: 'flex', gap: '8px' }}>
        <Button
          size="sm"
          kind="ghost"
          renderIcon={LaunchIcon}
          hasIconOnly
          iconDescription={t('platform.projects.viewPipelines', '查看流水线')}
          onClick={() => navigate(`/platform/pipelines?projectId=${proj.id}`)}
        />
        <Button
          size="sm"
          kind="danger--ghost"
          renderIcon={DeleteIcon}
          hasIconOnly
          iconDescription={t('platform.projects.deleteProject', '删除项目')}
          onClick={() => handleDelete(proj.id)}
        />
      </div>
    )
  }));

  return (
    <div className="platform-container">
      <div className="platform-hero">
        <div className="platform-hero-content">
          <h1>
            <FolderIcon size={32} />{' '}
            {t('platform.projects.title', '项目与工作空间管理')}
          </h1>
          <p>
            {t(
              'platform.projects.subtitle',
              '集中管理应用代码库、Git 仓库关联、Kubernetes 目标命名空间与构建流水线。'
            )}
          </p>
        </div>
        <div className="platform-hero-actions">
          <Button
            size="md"
            kind="ghost"
            renderIcon={RefreshIcon}
            onClick={fetchProjects}
          >
            {t('platform.refresh', '刷新')}
          </Button>
          <Button
            size="md"
            kind="primary"
            renderIcon={AddIcon}
            onClick={() => setIsModalOpen(true)}
          >
            {t('platform.projects.new', '新建项目')}
          </Button>
        </div>
      </div>

      {error && (
        <InlineNotification
          kind="error"
          title={t('dashboard.about.error', '操作失败')}
          subtitle={error}
          style={{ marginBottom: '20px' }}
        />
      )}

      {loading && projects.length === 0 ? (
        <Loading
          description={t(
            'platform.loading.workspaces',
            '正在加载工作空间列表...'
          )}
          withOverlay={false}
        />
      ) : rows.length === 0 ? (
        <div className="platform-empty-state">
          <FolderIcon className="empty-icon" />
          <h3>{t('platform.projects.emptyTitle', '暂无已注册项目')}</h3>
          <p>
            {t(
              'platform.projects.emptyDesc',
              '连接 Git 仓库或创建工作空间，开始编排自动化构建与交付流水线。'
            )}
          </p>
          <Button
            size="md"
            kind="primary"
            renderIcon={AddIcon}
            onClick={() => setIsModalOpen(true)}
          >
            {t('platform.projects.create', '创建工作空间')}
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
        modalHeading={t('platform.projects.modalHeading', '新建工作空间项目')}
        primaryButtonText={
          submitting
            ? t('platform.submitting', '创建中...')
            : t('platform.create', '创建项目')
        }
        secondaryButtonText={t('carbon.table.batch.cancel', '取消')}
        onRequestSubmit={handleCreate}
        onRequestClose={() => setIsModalOpen(false)}
        primaryButtonDisabled={submitting}
      >
        <div className="platform-modal-form">
          <TextInput
            id="proj-name"
            labelText={t('platform.projects.name', '项目名称')}
            placeholder={t(
              'platform.projects.namePlaceholder',
              '例如 backend-api 或 frontend-web'
            )}
            value={formData.name}
            onChange={e => setFormData({ ...formData, name: e.target.value })}
            required
          />
          <TextInput
            id="proj-git"
            labelText={t('platform.projects.gitUrl', 'Git 仓库地址')}
            placeholder={t(
              'platform.projects.gitPlaceholder',
              'https://github.com/organization/repo.git'
            )}
            value={formData.git_url}
            onChange={e =>
              setFormData({ ...formData, git_url: e.target.value })
            }
          />
          <TextInput
            id="proj-branch"
            labelText={t('platform.projects.defaultBranch', '默认分支')}
            placeholder="main"
            value={formData.default_branch}
            onChange={e =>
              setFormData({ ...formData, default_branch: e.target.value })
            }
          />
          <TextInput
            id="proj-ns"
            labelText={t(
              'platform.projects.namespace',
              'Kubernetes 目标命名空间'
            )}
            placeholder="default"
            value={formData.namespace}
            onChange={e =>
              setFormData({ ...formData, namespace: e.target.value })
            }
          />
          <TextArea
            id="proj-desc"
            labelText={t('platform.projects.description', '项目描述')}
            placeholder={t(
              'platform.projects.descPlaceholder',
              '服务描述与部署说明'
            )}
            value={formData.description}
            onChange={e =>
              setFormData({ ...formData, description: e.target.value })
            }
          />
        </div>
      </Modal>
    </div>
  );
}

export default PlatformProjects;
