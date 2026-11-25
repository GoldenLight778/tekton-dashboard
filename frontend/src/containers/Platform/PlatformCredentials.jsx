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
import {
  Button,
  Checkbox,
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
  ApiKey as KeyIcon,
  Renew as RefreshIcon
} from '@carbon/react/icons';
import { ALL_NAMESPACES } from '@tektoncd/dashboard-utils';

import { useSelectedNamespace } from '../../api';
import {
  createK8sSecret,
  deleteK8sSecret,
  getK8sNamespaces,
  getK8sSecrets,
  usePermissions
} from '../../api/platform';

const SECRET_TYPES = [
  {
    value: 'kubernetes.io/basic-auth',
    label: 'Git 账号密码 / Personal Access Token (kubernetes.io/basic-auth)',
    annotation: 'tekton.dev/git-0',
    defaultTarget: 'https://github.com'
  },
  {
    value: 'kubernetes.io/ssh-auth',
    label: 'Git SSH 私钥凭据 (kubernetes.io/ssh-auth)',
    annotation: 'tekton.dev/git-0',
    defaultTarget: 'github.com'
  },
  {
    value: 'kubernetes.io/dockerconfigjson',
    label: 'Docker / Harbor 镜像仓库凭据 (tekton.dev/docker-0)',
    annotation: 'tekton.dev/docker-0',
    defaultTarget: 'https://index.docker.io/v1/'
  },
  {
    value: 'Opaque',
    label: '通用环境变量密钥 (Opaque Secret)',
    annotation: 'none',
    defaultTarget: ''
  }
];

export function PlatformCredentials() {
  const { selectedNamespace } = useSelectedNamespace();
  const perm = usePermissions(selectedNamespace);

  const [namespaces, setNamespaces] = useState([]);
  const [secrets, setSecrets] = useState([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState(null);
  const [successMsg, setSuccessMsg] = useState(null);

  // Create Modal State
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [secName, setSecName] = useState('');
  const [targetNS, setTargetNS] = useState(
    selectedNamespace && selectedNamespace !== ALL_NAMESPACES
      ? selectedNamespace
      : 'default'
  );
  const [secType, setSecType] = useState('kubernetes.io/basic-auth');
  const [targetURL, setTargetURL] = useState('https://github.com');
  const [username, setUsername] = useState('git');
  const [secretValue, setSecretValue] = useState('');
  const [serviceAccount, setServiceAccount] = useState('default');
  const [autoBindSA, setAutoBindSA] = useState(true);
  const [submitting, setSubmitting] = useState(false);

  const fetchData = async (isManual = false) => {
    try {
      if (isManual) {
        setRefreshing(true);
      } else {
        setLoading(true);
      }
      setError(null);
      const [secList, nsList] = await Promise.all([
        getK8sSecrets(selectedNamespace),
        getK8sNamespaces()
      ]);
      setSecrets(Array.isArray(secList) ? secList : []);
      setNamespaces(Array.isArray(nsList) ? nsList : []);
    } catch (err) {
      setError(err.message || '加载凭据列表失败');
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };

  useEffect(() => {
    if (selectedNamespace && selectedNamespace !== ALL_NAMESPACES) {
      setTargetNS(selectedNamespace);
    }
    fetchData(false);
  }, [selectedNamespace]);

  const handleTypeChange = newType => {
    setSecType(newType);
    const preset = SECRET_TYPES.find(t => t.value === newType);
    if (preset) {
      setTargetURL(preset.defaultTarget);
    }
  };

  const handleCreate = async () => {
    if (!secName.trim()) {
      setError('Secret 名称不能为空（小写字母、数字、连字符）');
      return;
    }
    if (!secretValue.trim()) {
      setError('密钥内容 (Token / Password / SSH 私钥) 不能为空');
      return;
    }
    try {
      setSubmitting(true);
      setError(null);
      await createK8sSecret(targetNS, {
        name: secName.trim().toLowerCase(),
        namespace: targetNS,
        secret_type: secType,
        target_url: targetURL.trim(),
        username: username.trim(),
        secret_value: secretValue,
        service_account: serviceAccount.trim() || 'default',
        auto_bind_sa: autoBindSA
      });
      setIsModalOpen(false);
      setSecName('');
      setSecretValue('');
      setSuccessMsg(
        `已在命名空间 [${targetNS}] 创建 K8s Secret "${secName.trim()}"${
          autoBindSA
            ? ` 并自动挂载至 ServiceAccount "${serviceAccount || 'default'}"`
            : ''
        }`
      );
      fetchData();
    } catch (err) {
      setError(err.message || '创建 K8s Secret 失败');
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async item => {
    if (
      !window.confirm(
        `确定要删除命名空间 [${item.namespace}] 下的 K8s Secret "${item.name}" 吗？`
      )
    ) {
      return;
    }
    try {
      await deleteK8sSecret(item.namespace, item.name);
      setSuccessMsg(`已删除 K8s Secret: ${item.name}`);
      fetchData();
    } catch (err) {
      setError(err.message || '删除 Secret 失败');
    }
  };

  const headers = [
    { key: 'name', header: 'Secret 名称' },
    { key: 'namespace', header: '所属 Namespace' },
    { key: 'secret_type', header: 'Kubernetes Secret 类型' },
    { key: 'annotation', header: 'Tekton 认证注解与目标地址' },
    { key: 'service_account', header: '自动挂载 ServiceAccount' },
    { key: 'masked_value', header: '脱敏指纹' },
    { key: 'actions', header: '操作' }
  ];

  const rows = secrets.map(sec => ({
    id: `${sec.namespace}/${sec.name}`,
    name: sec.name,
    namespace: sec.namespace,
    secret_type: sec.secret_type,
    annotation:
      sec.annotation_type && sec.annotation_type !== 'none'
        ? `${sec.annotation_type}: ${sec.target_url}`
        : '通用 Opaque Secret',
    service_account: sec.auto_bind_sa
      ? `已挂载 -> ${sec.service_account || 'default'}`
      : '未自动挂载',
    masked_value: sec.masked_value || '******',
    raw: sec
  }));

  return (
    <div className="platform-container">
      <div className="platform-header">
        <div className="title-area">
          <h1>构建凭据</h1>
          <p className="subtitle">
            按命名空间隔离托管 Git 代码仓库与容器镜像仓库 Secret，并自动关联至流水线 ServiceAccount
          </p>
        </div>
        <div className="header-actions" style={{ display: 'flex', gap: '10px' }}>
          <Button
            className={refreshing ? 'is-spinning-icon' : ''}
            disabled={refreshing || loading}
            kind="ghost"
            onClick={() => fetchData(true)}
            renderIcon={RefreshIcon}
            size="md"
          >
            {refreshing ? '刷新中...' : '刷新'}
          </Button>
          <Button
            disabled={!perm.canWrite}
            kind="primary"
            onClick={() => setIsModalOpen(true)}
            renderIcon={AddIcon}
            size="md"
            title={
              !perm.canWrite
                ? `当前角色 (${perm.k8s_cluster_role}) 为只读权限`
                : '新建凭据'
            }
          >
            新建凭据
          </Button>
        </div>
      </div>

      {!perm.canWrite && (
        <InlineNotification
          kind="info"
          lowContrast
          style={{ marginBottom: '1rem', maxWidth: '100%' }}
          subtitle={`当前账号 (${perm.username}) 在此命名空间下为只读角色 (${perm.k8s_cluster_role})。`}
          title="只读模式"
        />
      )}

      {error && (
        <InlineNotification
          kind="error"
          lowContrast
          onCloseButtonClick={() => setError(null)}
          style={{ marginBottom: '1rem', maxWidth: '100%' }}
          subtitle={error}
          title="操作异常"
        />
      )}

      {successMsg && (
        <InlineNotification
          kind="success"
          lowContrast
          onCloseButtonClick={() => setSuccessMsg(null)}
          style={{ marginBottom: '1rem', maxWidth: '100%' }}
          subtitle={successMsg}
          title="成功"
        />
      )}

      <div className="platform-table-wrapper">
        {loading ? (
          <div style={{ padding: '24px' }}>
            <Loading description="正在加载构建凭据..." withOverlay={false} />
          </div>
        ) : secrets.length === 0 ? (
          <div className="platform-empty-state" style={{ border: 'none' }}>
            <KeyIcon className="empty-icon" />
            <h3>
              当前作用域 (
              {selectedNamespace === ALL_NAMESPACES || !selectedNamespace
                ? 'All Namespaces'
                : selectedNamespace}
              ) 暂无构建凭据
            </h3>
            <p>
              新建 Git Token、SSH 私钥或镜像仓库凭据，系统将自动完成注解写入与 ServiceAccount 挂载。
            </p>
            {perm.canWrite && (
              <Button
                kind="primary"
                onClick={() => setIsModalOpen(true)}
                renderIcon={AddIcon}
                size="md"
              >
                新建凭据
              </Button>
            )}
          </div>
        ) : (
          <DataTable headers={headers} rows={rows}>
            {({ rows, headers, getHeaderProps, getRowProps, getTableProps }) => (
              <Table {...getTableProps()}>
                <TableHead>
                  <TableRow>
                    {headers.map(header => (
                      <TableHeader
                        {...getHeaderProps({ header })}
                        key={header.key}
                      >
                        {header.header}
                      </TableHeader>
                    ))}
                  </TableRow>
                </TableHead>
                <TableBody>
                  {rows.map(row => {
                    const secItem = secrets.find(
                      s => `${s.namespace}/${s.name}` === row.id
                    );
                    return (
                      <TableRow {...getRowProps({ row })} key={row.id}>
                        {row.cells.map(cell => {
                          if (
                            cell.info.header === 'name' ||
                            cell.info.header === 'namespace' ||
                            cell.info.header === 'masked_value'
                          ) {
                            return (
                              <TableCell key={cell.id}>
                                <code className="row-code-tag">{cell.value}</code>
                              </TableCell>
                            );
                          }
                          if (cell.info.header === 'actions') {
                            return (
                              <TableCell key={cell.id}>
                                {perm.canWrite && secItem && (
                                  <Button
                                    hasIconOnly
                                    iconDescription="删除凭据"
                                    kind="danger--ghost"
                                    onClick={() => handleDelete(secItem)}
                                    renderIcon={DeleteIcon}
                                    size="sm"
                                  />
                                )}
                              </TableCell>
                            );
                          }
                          return (
                            <TableCell key={cell.id}>{cell.value}</TableCell>
                          );
                        })}
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            )}
          </DataTable>
        )}
      </div>

      <Modal
        className="platform-modal"
        modalHeading="新建构建凭据"
        onRequestClose={() => setIsModalOpen(false)}
        onRequestSubmit={handleCreate}
        open={isModalOpen}
        primaryButtonDisabled={submitting}
        primaryButtonText={
          submitting ? '创建中...' : '确定创建'
        }
        secondaryButtonText="取消"
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: '1fr 1fr',
              gap: '1rem'
            }}
          >
            <TextInput
              id="k8s-sec-name"
              labelText="Secret 资源名称 (metadata.name)"
              onChange={e => setSecName(e.target.value)}
              placeholder="例如: github-pat-secret"
              value={secName}
            />
            <Select
              id="k8s-sec-ns"
              labelText="所属 Kubernetes Namespace"
              onChange={e => setTargetNS(e.target.value)}
              value={targetNS}
            >
              {namespaces.map(ns => (
                <SelectItem
                  key={ns.name}
                  text={`${ns.name}${ns.alias ? ` (${ns.alias})` : ''}`}
                  value={ns.name}
                />
              ))}
            </Select>
          </div>

          <Select
            id="k8s-sec-type"
            labelText="凭据类型 (K8s Secret Type & Tekton Annotation)"
            onChange={e => handleTypeChange(e.target.value)}
            value={secType}
          >
            {SECRET_TYPES.map(st => (
              <SelectItem key={st.value} text={st.label} value={st.value} />
            ))}
          </Select>

          {secType !== 'Opaque' && (
            <TextInput
              helperText={
                secType === 'kubernetes.io/dockerconfigjson'
                  ? '将自动写入 metadata.annotations["tekton.dev/docker-0"]'
                  : '将自动写入 metadata.annotations["tekton.dev/git-0"]'
              }
              id="k8s-sec-url"
              labelText="目标服务器地址 (Server URL / Host)"
              onChange={e => setTargetURL(e.target.value)}
              placeholder="例如: https://github.com 或 https://harbor.example.com"
              value={targetURL}
            />
          )}

          {(secType === 'kubernetes.io/basic-auth' ||
            secType === 'kubernetes.io/dockerconfigjson') && (
            <TextInput
              id="k8s-sec-user"
              labelText="认证账号 (Username)"
              onChange={e => setUsername(e.target.value)}
              placeholder="例如: git 或 docker-user"
              value={username}
            />
          )}

          <TextArea
            id="k8s-sec-val"
            labelText={
              secType === 'kubernetes.io/ssh-auth'
                ? 'SSH 私钥内容 (id_rsa / id_ed25519)'
                : '密钥 / Token / 密码内容 (stringData)'
            }
            onChange={e => setSecretValue(e.target.value)}
            placeholder={
              secType === 'kubernetes.io/ssh-auth'
                ? '-----BEGIN OPENSSH PRIVATE KEY-----\n...'
                : '请输入 Access Token 或密码...'
            }
            rows={3}
            value={secretValue}
          />

          <div
            style={{
              background: '#f8fafc',
              border: '1px solid #e2e8f0',
              borderRadius: '8px',
              padding: '0.85rem 1rem'
            }}
          >
            <Checkbox
              checked={autoBindSA}
              id="k8s-sec-autobind"
              labelText="自动将此 Secret 挂载至该 Namespace 的流水线 ServiceAccount（推荐，免去手动配置 YAML）"
              onChange={(_, { checked }) => setAutoBindSA(checked)}
            />
            {autoBindSA && (
              <div style={{ marginTop: '0.5rem' }}>
                <TextInput
                  id="k8s-sec-sa"
                  labelText="目标 ServiceAccount 名称"
                  onChange={e => setServiceAccount(e.target.value)}
                  placeholder="default 或 pipeline"
                  size="sm"
                  value={serviceAccount}
                />
              </div>
            )}
          </div>
        </div>
      </Modal>
    </div>
  );
}

export default PlatformCredentials;
