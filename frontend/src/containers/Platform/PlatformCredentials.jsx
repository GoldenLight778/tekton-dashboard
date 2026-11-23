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
  ApiKey as KeyIcon,
  Renew as RefreshIcon
} from '@carbon/react/icons';

import {
  createCredential,
  deleteCredential,
  getCredentials,
  getProjects
} from '../../api/platform';

export function PlatformCredentials() {
  const intl = useIntl();
  /* eslint-disable formatjs/enforce-default-message */
  const t = (id, defaultMessage, values) =>
    intl.formatMessage({ id, defaultMessage }, values);
  /* eslint-enable formatjs/enforce-default-message */

  const [projects, setProjects] = useState([]);
  const [selectedProjectId, setSelectedProjectId] = useState('');
  const [credentials, setCredentials] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [successMsg, setSuccessMsg] = useState(null);

  // Modal State
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [credName, setCredName] = useState('');
  const [credType, setCredType] = useState('git_token');
  const [secretVal, setSecretVal] = useState('');
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    const loadProjects = async () => {
      try {
        const projs = await getProjects();
        setProjects(projs || []);
        if (projs?.length > 0) {
          setSelectedProjectId(projs[0].id);
        }
      } catch (err) {
        setError(
          err.message || t('platform.error.fetchProjects', '加载项目列表失败')
        );
      }
    };
    loadProjects();
  }, []);

  const fetchCredentials = async (projId = selectedProjectId) => {
    if (!projId) {
      setCredentials([]);
      return;
    }
    try {
      setLoading(true);
      setError(null);
      const data = await getCredentials(projId);
      setCredentials(data || []);
    } catch (err) {
      setError(
        err.message || t('platform.error.fetchCredentials', '加载凭据列表失败')
      );
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (selectedProjectId) {
      fetchCredentials(selectedProjectId);
    }
  }, [selectedProjectId]);

  const handleCreate = async () => {
    if (!credName.trim()) {
      setError(t('platform.error.credNameReq', '凭据名称不能为空'));
      return;
    }
    if (!secretVal.trim()) {
      setError(t('platform.error.secretValReq', '密钥内容不能为空'));
      return;
    }
    try {
      setSubmitting(true);
      setError(null);
      await createCredential(selectedProjectId, {
        name: credName,
        type: credType,
        secret: secretVal
      });
      setIsModalOpen(false);
      setCredName('');
      setSecretVal('');
      setSuccessMsg(
        t(
          'platform.success.credStored',
          '凭据 ‘{name}’ 已通过 AES-256-GCM 高度加密存入安全保险库。',
          { name: credName }
        )
      );
      fetchCredentials();
    } catch (err) {
      setError(
        err.message || t('platform.error.storeCredential', '存储凭据失败')
      );
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async id => {
    if (
      !window.confirm(
        t('platform.confirm.deleteCredential', '确定要删除此加密机密凭据吗？')
      )
    ) {
      return;
    }
    try {
      await deleteCredential(id);
      fetchCredentials();
    } catch (err) {
      setError(
        err.message || t('platform.error.deleteCredential', '删除凭据失败')
      );
    }
  };

  const headers = [
    { key: 'name', header: t('platform.table.credential', '凭据名称') },
    { key: 'type', header: t('platform.table.type', '凭据类型') },
    {
      key: 'masked',
      header: t('platform.table.securityStatus', '安全加密状态')
    },
    { key: 'created_at', header: t('platform.table.createdDate', '创建时间') },
    { key: 'actions', header: t('platform.table.actions', '操作') }
  ];

  const rows = credentials.map(c => ({
    id: c.id,
    name: <strong style={{ color: '#0f172a' }}>{c.name}</strong>,
    type: <span className="code-chip">{c.type}</span>,
    masked: (
      <span
        style={{ display: 'inline-flex', alignItems: 'center', gap: '6px' }}
      >
        <span style={{ color: '#10b981', fontSize: '10px' }}>🔒</span>
        <span className="code-chip">{c.masked_value || '••••••••'}</span>
        <span style={{ fontSize: '0.75rem', color: '#64748b' }}>
          ({t('platform.table.encrypted', 'AES-256 加密')})
        </span>
      </span>
    ),
    created_at: new Date(c.created_at).toLocaleString(),
    actions: (
      <Button
        size="sm"
        kind="danger--ghost"
        renderIcon={DeleteIcon}
        hasIconOnly
        iconDescription={t('platform.deleteCredential', '删除凭据')}
        onClick={() => handleDelete(c.id)}
      />
    )
  }));

  return (
    <div className="platform-container">
      <div className="platform-hero">
        <div className="platform-hero-content">
          <h1>
            <KeyIcon size={32} />{' '}
            {t('platform.credentials.title', '安全与凭据保险库')}
          </h1>
          <p>
            {t(
              'platform.credentials.subtitle',
              '基于 AES-256-GCM 高强度加密保险库，安全存储 Git 访问令牌、镜像仓库密钥及 Kubeconfig 凭据。'
            )}
          </p>
        </div>
        <div className="platform-hero-actions">
          <Button
            size="md"
            kind="ghost"
            renderIcon={RefreshIcon}
            onClick={() => fetchCredentials()}
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
            {t('platform.credentials.new', '添加凭据')}
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
              id="cred-project-selector"
              labelText=""
              hideLabel
              size="sm"
              value={selectedProjectId}
              onChange={e => setSelectedProjectId(e.target.value)}
            >
              {projects.map(p => (
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
          description={t(
            'platform.loading.credentials',
            '正在安全加载凭据元数据...'
          )}
          withOverlay={false}
        />
      ) : rows.length === 0 ? (
        <div className="platform-empty-state">
          <KeyIcon className="empty-icon" />
          <h3>
            {t('platform.credentials.emptyTitle', '该项目下暂无加密凭据')}
          </h3>
          <p>
            {t(
              'platform.credentials.emptyDesc',
              '安全注入 Git 访问令牌或镜像仓库凭据到 Tekton 流水线任务中。'
            )}
          </p>
          <Button
            size="md"
            kind="primary"
            renderIcon={AddIcon}
            onClick={() => setIsModalOpen(true)}
            disabled={!selectedProjectId}
          >
            {t('platform.credentials.create', '添加新凭据')}
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
        modalHeading={t('platform.credentials.modalHeading', '存储加密凭据')}
        primaryButtonText={
          submitting
            ? t('platform.submitting', '加密存储中...')
            : t('platform.credentials.create', '存储凭据')
        }
        secondaryButtonText={t('carbon.table.batch.cancel', '取消')}
        onRequestSubmit={handleCreate}
        onRequestClose={() => setIsModalOpen(false)}
        primaryButtonDisabled={submitting}
      >
        <div className="platform-modal-form">
          <TextInput
            id="cred-name-input"
            labelText={t('platform.credentials.name', '凭据名称')}
            placeholder={t(
              'platform.credentials.namePlaceholder',
              '例如 github-pat 或 docker-auth'
            )}
            value={credName}
            onChange={e => setCredName(e.target.value)}
            required
          />
          <Select
            id="cred-type-select"
            labelText={t('platform.credentials.type', '凭据类型')}
            value={credType}
            onChange={e => setCredType(e.target.value)}
          >
            <SelectItem value="git_token" text="Git Access Token (PAT)" />
            <SelectItem value="ssh_key" text="SSH Private Key" />
            <SelectItem value="kubeconfig" text="Kubernetes Kubeconfig" />
            <SelectItem value="secret_text" text="Secret Text / API Token" />
          </Select>
          <TextArea
            id="cred-secret-input"
            labelText={t('platform.credentials.secret', '密钥内容')}
            helperText={t(
              'platform.credentials.secretHelper',
              '经服务端主密钥通过 AES-256-GCM 算法加密，API 响应中绝不泄露明文。'
            )}
            rows={4}
            value={secretVal}
            onChange={e => setSecretVal(e.target.value)}
            required
          />
        </div>
      </Modal>
    </div>
  );
}

export default PlatformCredentials;
