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
  Launch as LaunchIcon,
  Renew as RefreshIcon
} from '@carbon/react/icons';
import { urls } from '@tektoncd/dashboard-utils';

import { useSelectedNamespace } from '../../api';
import { getK8sNamespaces } from '../../api/platform';

export function PlatformNamespaces() {
  const navigate = useNavigate();
  const { selectedNamespace, selectNamespace } = useSelectedNamespace();

  const [namespaces, setNamespaces] = useState([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState(null);

  const fetchNamespaces = async (isManual = false) => {
    try {
      if (isManual) {
        setRefreshing(true);
      } else {
        setLoading(true);
      }
      setError(null);
      const list = await getK8sNamespaces();
      setNamespaces(Array.isArray(list) ? list : []);
    } catch (err) {
      setError(err.message || '加载 Kubernetes 集群命名空间列表失败');
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };

  useEffect(() => {
    fetchNamespaces(false);
  }, []);

  const handleActivateNamespace = name => {
    selectNamespace(name);
    navigate(urls.pipelines.byNamespace({ namespace: name }));
  };

  const headers = [
    { key: 'name', header: 'Kubernetes Namespace' },
    { key: 'status', header: '状态' },
    { key: 'description', header: '说明' },
    { key: 'assets', header: '空间内资源统计' },
    { key: 'role_display', header: '当前账号角色' },
    { key: 'actions', header: '操作' }
  ];

  const rows = namespaces.map(ns => ({
    id: ns.name,
    name: ns.name,
    status: ns.status || 'Active',
    description: ns.description || ns.alias || 'Kubernetes 集群命名空间',
    assets: `${ns.pipelines_count || 0} 流水线 · ${ns.tasks_count || 0} 任务 · ${
      ns.triggers_count || 0
    } 触发器 · ${ns.secrets_count || 0} 凭据`,
    role_display: ns.role_display || '平台管理员',
    k8s_role_name: ns.k8s_role_name || 'tekton-devops-admin',
    effective_role: ns.effective_role || 'admin'
  }));

  return (
    <div className="platform-container">
      <div className="platform-header">
        <div className="title-area">
          <h1>命名空间</h1>
          <p className="subtitle">
            展示当前账号有权限访问的 Kubernetes 集群命名空间（命名空间由集群管理员在 K8s 侧维护）
          </p>
        </div>
        <div className="header-actions" style={{ display: 'flex', gap: '10px' }}>
          <Button
            className={refreshing ? 'is-spinning-icon' : ''}
            disabled={refreshing || loading}
            kind="ghost"
            onClick={() => fetchNamespaces(true)}
            renderIcon={RefreshIcon}
            size="md"
          >
            {refreshing ? '刷新中...' : '刷新'}
          </Button>
        </div>
      </div>

      {error && (
        <InlineNotification
          kind="error"
          lowContrast
          onCloseButtonClick={() => setError(null)}
          style={{ marginBottom: '1rem', maxWidth: '100%' }}
          subtitle={error}
          title="获取集群命名空间失败"
        />
      )}

      <div className="platform-table-wrapper">
        {loading ? (
          <div style={{ padding: '24px' }}>
            <Loading description="正在读取集群命名空间..." withOverlay={false} />
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
                    const nsNameVal = row.id;
                    const isCurrent = selectedNamespace === nsNameVal;
                    const rowMeta = namespaces.find(n => n.name === nsNameVal);
                    return (
                      <TableRow {...getRowProps({ row })} key={row.id}>
                        {row.cells.map(cell => {
                          if (cell.info.header === 'name') {
                            return (
                              <TableCell key={cell.id}>
                                <div
                                  style={{
                                    display: 'flex',
                                    alignItems: 'center',
                                    gap: '8px'
                                  }}
                                >
                                  <code className="row-code-tag">{cell.value}</code>
                                  {isCurrent && (
                                    <span className="ns-active-pill">
                                      当前作用域
                                    </span>
                                  )}
                                </div>
                              </TableCell>
                            );
                          }
                          if (cell.info.header === 'status') {
                            return (
                              <TableCell key={cell.id}>
                                <span className="status-pill status-succeeded">
                                  <span className="dot" /> {cell.value}
                                </span>
                              </TableCell>
                            );
                          }
                          if (cell.info.header === 'role_display') {
                            return (
                              <TableCell key={cell.id}>
                                <div style={{ display: 'inline-flex', alignItems: 'center', gap: '8px' }}>
                                  <span
                                    className={`role-pill role-${
                                      rowMeta?.effective_role || 'admin'
                                    }`}
                                  >
                                    {cell.value}
                                  </span>
                                  <code className="row-code-tag">
                                    {rowMeta?.k8s_role_name || 'tekton-devops-admin'}
                                  </code>
                                </div>
                              </TableCell>
                            );
                          }
                          if (cell.info.header === 'actions') {
                            return (
                              <TableCell key={cell.id}>
                                <Button
                                  kind="ghost"
                                  onClick={() =>
                                    handleActivateNamespace(nsNameVal)
                                  }
                                  renderIcon={LaunchIcon}
                                  size="sm"
                                >
                                  查看流水线
                                </Button>
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
    </div>
  );
}

export default PlatformNamespaces;
