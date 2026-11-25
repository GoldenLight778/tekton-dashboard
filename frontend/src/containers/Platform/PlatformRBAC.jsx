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
  InlineNotification,
  Loading,
  Modal,
  Select,
  SelectItem,
  TextInput
} from '@carbon/react';
import {
  Add as AddIcon,
  TrashCan as DeleteIcon,
  Renew as RefreshIcon,
  Settings as SettingsIcon,
  UserAdmin as ShieldIcon,
  User as UserIcon
} from '@carbon/react/icons';

import { useSelectedNamespace } from '../../api';
import {
  createRBACUser,
  deleteRBACRole,
  deleteRBACUser,
  getK8sNamespaces,
  getRBACRoles,
  getRBACUsers,
  saveRBACRole,
  updateRBACUser,
  usePermissions
} from '../../api/platform';

const BUILTIN_ROLE_NAMES = new Set([
  'tekton-devops-admin',
  'tekton-devops-worker',
  'tekton-devops-readonly'
]);

const PERMISSION_MODULES = [
  { key: 'pipelines', label: '流水线', supportsRun: true },
  { key: 'tasks', label: '任务库', supportsRun: true },
  { key: 'pipelineruns', label: '执行记录', supportsRun: false },
  { key: 'triggers', label: '触发器与事件', supportsRun: false },
  { key: 'secrets', label: '构建凭据', supportsRun: false }
];

const DEFAULT_MATRIX = {
  pipelines: { get: true, list: true, create: true, update: true, delete: true, run: true },
  tasks: { get: true, list: true, create: true, update: true, delete: true, run: true },
  pipelineruns: { get: true, list: true, create: true, update: true, delete: true },
  triggers: { get: true, list: true, create: true, update: true, delete: true },
  secrets: { get: true, list: true, create: true, update: true, delete: true }
};

function isBuiltinRole(roleObj) {
  if (!roleObj) return false;
  return !!roleObj.is_builtin || BUILTIN_ROLE_NAMES.has(roleObj.name);
}

function getConciseRoleDescription(roleObj) {
  if (!roleObj) return '-';
  if (roleObj.name === 'tekton-devops-admin') return '全局最高管理权限';
  if (roleObj.name === 'tekton-devops-worker') return '命名空间内资源读写与执行权限';
  if (roleObj.name === 'tekton-devops-readonly') return '命名空间内只读查看权限';
  return roleObj.description || '自定义角色';
}

export function PlatformRBAC() {
  const { selectedNamespace } = useSelectedNamespace();
  const perm = usePermissions(selectedNamespace);

  // Main View Switch: 'roles' | 'users'
  const [activeTab, setActiveTab] = useState('roles');

  const [roles, setRoles] = useState([]);
  const [users, setUsers] = useState([]);
  const [namespaces, setNamespaces] = useState([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState(null);
  const [successMsg, setSuccessMsg] = useState(null);
  const [saving, setSaving] = useState(false);

  // Role Configuration Modal State (ClusterRole + Namespace-level RoleBindings)
  const [isRoleConfigOpen, setIsRoleConfigOpen] = useState(false);
  const [selectedRoleName, setSelectedRoleName] = useState('');
  const [roleDraft, setRoleDraft] = useState(null);

  // User Configuration Modal State (User simply binds to a Role)
  const [isUserConfigOpen, setIsUserConfigOpen] = useState(false);
  const [selectedUserId, setSelectedUserId] = useState('');
  const [userDraft, setUserDraft] = useState(null);

  // Create Role Modal
  const [isNewRoleOpen, setIsNewRoleOpen] = useState(false);
  const [newRoleForm, setNewRoleForm] = useState({
    name: '',
    display_name: '',
    role_type: 'worker',
    description: '',
    allNamespaces: false,
    selectedNamespaces: ['default']
  });

  // Create User Modal
  const [isNewUserOpen, setIsNewUserOpen] = useState(false);
  const [newUserForm, setNewUserForm] = useState({
    username: '',
    password: '',
    role: 'worker',
    role_name: 'tekton-devops-worker'
  });

  const openRoleConfigModal = (roleObj, nsList = namespaces) => {
    if (!roleObj) return;
    setSelectedRoleName(roleObj.name);
    const bound = Array.isArray(roleObj.bound_namespaces)
      ? roleObj.bound_namespaces
      : ['default'];
    const isWildcard =
      bound.includes('*') ||
      roleObj.role_type === 'admin' ||
      roleObj.name === 'tekton-devops-admin';

    setRoleDraft({
      isUpdate: true,
      name: roleObj.name,
      display_name: roleObj.display_name || roleObj.name,
      role_type: roleObj.role_type || 'worker',
      description: getConciseRoleDescription(roleObj),
      is_builtin: isBuiltinRole(roleObj),
      allNamespaces: isWildcard,
      selectedNamespaces: isWildcard
        ? nsList.map(n => n.name)
        : bound.filter(Boolean),
      permissions: structuredClone(roleObj.permissions || DEFAULT_MATRIX)
    });
    setIsRoleConfigOpen(true);
  };

  const openUserConfigModal = userObj => {
    if (!userObj) return;
    setSelectedUserId(String(userObj.id));
    setUserDraft({
      id: userObj.id,
      username: userObj.username,
      role: userObj.role || 'worker',
      role_name:
        userObj.k8s_role_name || `tekton-devops-${userObj.role || 'worker'}`
    });
    setIsUserConfigOpen(true);
  };

  const fetchAll = async (isManualRefresh = false) => {
    try {
      if (isManualRefresh) {
        setRefreshing(true);
      } else {
        setLoading(true);
      }
      setError(null);
      const [rList, uList, nsList] = await Promise.all([
        getRBACRoles(),
        getRBACUsers(),
        getK8sNamespaces()
      ]);
      const safeRoles = Array.isArray(rList) ? rList : [];
      const safeUsers = Array.isArray(uList) ? uList : [];
      const safeNS = Array.isArray(nsList) ? nsList : [];

      setRoles(safeRoles);
      setUsers(safeUsers);
      setNamespaces(safeNS);

      if (isManualRefresh) {
        setSuccessMsg('已重新加载最新权限列表');
      }
    } catch (err) {
      setError(err.message || '加载权限数据失败');
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };

  useEffect(() => {
    fetchAll(false);
  }, []);

  // --- Role Config Modal & Delete Handlers ---
  const toggleRolePerm = (modKey, verbKey, checked) => {
    if (!roleDraft) return;
    setRoleDraft(prev => ({
      ...prev,
      permissions: {
        ...prev.permissions,
        [modKey]: {
          ...(prev.permissions[modKey] || {}),
          [verbKey]: checked,
          ...(verbKey === 'get' ? { list: checked } : {})
        }
      }
    }));
  };

  const setModulePreset = (modKey, mode) => {
    if (!roleDraft) return;
    const isFull = mode === 'full';
    setRoleDraft(prev => ({
      ...prev,
      permissions: {
        ...prev.permissions,
        [modKey]: {
          get: true,
          list: true,
          create: isFull,
          update: isFull,
          delete: isFull,
          run: isFull
        }
      }
    }));
  };

  const toggleRoleDraftNS = (nsName, checked) => {
    if (!roleDraft) return;
    const set = new Set(roleDraft.selectedNamespaces);
    if (checked) {
      set.add(nsName);
    } else {
      set.delete(nsName);
    }
    setRoleDraft(prev => ({
      ...prev,
      selectedNamespaces: Array.from(set)
    }));
  };

  const handleSaveRoleDetail = async () => {
    if (!roleDraft) return;
    const targetNS =
      roleDraft.role_type === 'admin' ||
      roleDraft.name === 'tekton-devops-admin' ||
      roleDraft.allNamespaces
        ? ['*']
        : roleDraft.selectedNamespaces.length > 0
          ? roleDraft.selectedNamespaces
          : ['default'];
    try {
      setSaving(true);
      setError(null);
      await saveRBACRole({
        ...roleDraft,
        bound_namespaces: targetNS
      });
      setIsRoleConfigOpen(false);
      setSuccessMsg(`角色「${roleDraft.display_name}」权限与命名空间 RoleBinding 已保存`);
      await fetchAll(false);
      window.dispatchEvent(new CustomEvent('tkn-auth-change'));
    } catch (err) {
      setError(err.message || '保存角色配置失败');
    } finally {
      setSaving(false);
    }
  };

  const handleDeleteRole = async (roleObj, e) => {
    if (e) {
      e.stopPropagation();
    }
    if (!roleObj || isBuiltinRole(roleObj)) return;
    if (!window.confirm(`确定要删除角色「${roleObj.display_name || roleObj.name}」吗？`)) {
      return;
    }
    try {
      setSaving(true);
      setError(null);
      await deleteRBACRole(roleObj.name);
      setIsRoleConfigOpen(false);
      setSuccessMsg(`已删除角色「${roleObj.display_name || roleObj.name}」`);
      if (selectedRoleName === roleObj.name) {
        setSelectedRoleName('');
      }
      await fetchAll(false);
    } catch (err) {
      setError(err.message || '删除角色失败');
    } finally {
      setSaving(false);
    }
  };

  const handleCreateNewRole = async () => {
    if (!newRoleForm.name.trim() || !newRoleForm.display_name.trim()) {
      setError('请填写角色标识与角色名称');
      return;
    }
    const isReadonly = newRoleForm.role_type === 'readonly';
    const initMatrix = {};
    PERMISSION_MODULES.forEach(m => {
      initMatrix[m.key] = {
        get: true,
        list: true,
        create: !isReadonly,
        update: !isReadonly,
        delete: !isReadonly,
        run: !isReadonly
      };
    });
    const targetNS = newRoleForm.allNamespaces
      ? ['*']
      : newRoleForm.selectedNamespaces.length > 0
        ? newRoleForm.selectedNamespaces
        : ['default'];
    try {
      setSaving(true);
      setError(null);
      const created = await saveRBACRole({
        isUpdate: false,
        name: newRoleForm.name.trim().toLowerCase(),
        display_name: newRoleForm.display_name.trim(),
        role_type: newRoleForm.role_type,
        description: newRoleForm.description.trim() || '自定义角色',
        bound_namespaces: targetNS,
        permissions: initMatrix
      });
      setIsNewRoleOpen(false);
      setNewRoleForm({
        name: '',
        display_name: '',
        role_type: 'worker',
        description: '',
        allNamespaces: false,
        selectedNamespaces: ['default']
      });
      setSuccessMsg(`已新建角色「${created.display_name || newRoleForm.display_name}」并下发目标 Namespace RoleBinding`);
      await fetchAll(false);
    } catch (err) {
      setError(err.message || '新建角色失败');
    } finally {
      setSaving(false);
    }
  };

  // --- User Config Modal & Delete Handlers ---
  const handleUserDraftRoleChange = newRoleName => {
    if (!userDraft) return;
    const matchedRole = roles.find(r => r.name === newRoleName);
    const rType = matchedRole?.role_type || 'worker';
    setUserDraft(prev => ({
      ...prev,
      role: rType,
      role_name: newRoleName
    }));
  };

  const handleSaveUserDetail = async () => {
    if (!userDraft) return;
    try {
      setSaving(true);
      setError(null);
      await updateRBACUser(userDraft.id, {
        role: userDraft.role,
        role_name: userDraft.role_name
      });
      setIsUserConfigOpen(false);
      setSuccessMsg(`用户「${userDraft.username}」角色绑定已保存`);
      await fetchAll(false);
      window.dispatchEvent(new CustomEvent('tkn-auth-change'));
    } catch (err) {
      setError(err.message || '保存用户配置失败');
    } finally {
      setSaving(false);
    }
  };

  const handleDeleteUser = async (userObj, e) => {
    if (e) {
      e.stopPropagation();
    }
    if (!userObj || userObj.username === 'admin') return;
    if (!window.confirm(`确定要删除用户「${userObj.username}」吗？`)) {
      return;
    }
    try {
      setSaving(true);
      setError(null);
      await deleteRBACUser(userObj.id);
      setIsUserConfigOpen(false);
      setSuccessMsg(`已删除用户「${userObj.username}」`);
      if (String(selectedUserId) === String(userObj.id)) {
        setSelectedUserId('');
      }
      await fetchAll(false);
    } catch (err) {
      setError(err.message || '删除用户失败');
    } finally {
      setSaving(false);
    }
  };

  const handleCreateNewUser = async () => {
    if (!newUserForm.username.trim() || !newUserForm.password.trim()) {
      setError('请填写账号和初始密码');
      return;
    }
    const matchedRole = roles.find(r => r.name === newUserForm.role_name);
    const rType = matchedRole?.role_type || newUserForm.role || 'worker';
    try {
      setSaving(true);
      setError(null);
      const created = await createRBACUser({
        username: newUserForm.username.trim(),
        password: newUserForm.password,
        role: rType,
        role_name: newUserForm.role_name
      });
      setIsNewUserOpen(false);
      setNewUserForm({
        username: '',
        password: '',
        role: 'worker',
        role_name: 'tekton-devops-worker'
      });
      setSuccessMsg(`已创建用户「${created.username}」并关联至角色 RoleBinding`);
      await fetchAll(false);
    } catch (err) {
      setError(err.message || '新建用户失败');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="platform-container">
      {/* Top Header with View Switcher & Refresh Feedback */}
      <div className="platform-header rbac-clean-header">
        <div className="rbac-header-left">
          <h1>权限管理</h1>
          <div className="rbac-view-switcher">
            <button
              className={`rbac-switch-btn ${
                activeTab === 'roles' ? 'active' : ''
              }`}
              onClick={() => setActiveTab('roles')}
              type="button"
            >
              <ShieldIcon size={16} />
              <span>角色配置 ({roles.length})</span>
            </button>
            <button
              className={`rbac-switch-btn ${
                activeTab === 'users' ? 'active' : ''
              }`}
              onClick={() => setActiveTab('users')}
              type="button"
            >
              <UserIcon size={16} />
              <span>用户配置 ({users.length})</span>
            </button>
          </div>
        </div>

        <div
          className="header-actions"
          style={{ display: 'flex', alignItems: 'center', gap: '10px' }}
        >
          <Button
            className={refreshing ? 'is-spinning-icon' : ''}
            disabled={refreshing || loading}
            kind="ghost"
            onClick={() => fetchAll(true)}
            renderIcon={RefreshIcon}
            size="md"
          >
            {refreshing ? '刷新中...' : '刷新'}
          </Button>
          {perm.isAdmin &&
            (activeTab === 'roles' ? (
              <Button
                kind="primary"
                onClick={() => setIsNewRoleOpen(true)}
                renderIcon={AddIcon}
                size="md"
              >
                新建角色
              </Button>
            ) : (
              <Button
                kind="primary"
                onClick={() => setIsNewUserOpen(true)}
                renderIcon={AddIcon}
                size="md"
              >
                新建用户
              </Button>
            ))}
        </div>
      </div>

      {error && (
        <InlineNotification
          kind="error"
          lowContrast
          onCloseButtonClick={() => setError(null)}
          style={{ marginBottom: '1rem', maxWidth: '100%' }}
          subtitle={error}
          title="操作失败"
        />
      )}

      {successMsg && (
        <InlineNotification
          kind="success"
          lowContrast
          onCloseButtonClick={() => setSuccessMsg(null)}
          style={{ marginBottom: '1rem', maxWidth: '100%' }}
          subtitle={successMsg}
          title="提示"
        />
      )}

      {loading ? (
        <Loading description="正在加载权限配置..." withOverlay={false} />
      ) : activeTab === 'roles' ? (
        /* ==================== VIEW 1: 角色配置列表 ==================== */
        <div className="rbac-master-detail-layout">
          <div className="rbac-horizontal-list-card">
            <table className="rbac-horizontal-list-table">
              <thead>
                <tr>
                  <th>角色名称</th>
                  <th>ClusterRole 标识</th>
                  <th>权限级别</th>
                  <th>生效命名空间 (RoleBinding)</th>
                  <th>说明</th>
                  <th style={{ width: '180px', textAlign: 'right' }}>操作</th>
                </tr>
              </thead>
              <tbody>
                {roles.map(r => {
                  const builtin = isBuiltinRole(r);
                  const isSelected =
                    isRoleConfigOpen && r.name === selectedRoleName;
                  const boundNS = Array.isArray(r.bound_namespaces)
                    ? r.bound_namespaces
                    : r.role_type === 'admin'
                      ? ['*']
                      : ['default'];
                  return (
                    <tr
                      className={`rbac-list-row ${
                        isSelected ? 'selected' : ''
                      }`}
                      key={r.name}
                      onClick={() => openRoleConfigModal(r)}
                    >
                      <td>
                        <strong>{r.display_name}</strong>
                      </td>
                      <td>
                        <code className="row-code-tag">{r.name}</code>
                      </td>
                      <td>
                        <span className={`role-pill role-${r.role_type}`}>
                          {r.role_type === 'admin'
                            ? '平台管理员'
                            : r.role_type === 'readonly'
                              ? '只读角色'
                              : '工作人员'}
                        </span>
                      </td>
                      <td>
                        <div style={{ display: 'flex', flexWrap: 'wrap', gap: '4px' }}>
                          {boundNS.includes('*') ? (
                            <code className="row-code-tag">全部命名空间 (*)</code>
                          ) : (
                            boundNS.map(ns => (
                              <code className="row-code-tag" key={ns}>
                                {ns}
                              </code>
                            ))
                          )}
                        </div>
                      </td>
                      <td>
                        <span className="row-desc-text">
                          {getConciseRoleDescription(r)}
                        </span>
                      </td>
                      <td
                        onClick={e => e.stopPropagation()}
                        style={{ textAlign: 'right' }}
                      >
                        <div className="row-action-group">
                          <Button
                            kind="ghost"
                            onClick={() => openRoleConfigModal(r)}
                            renderIcon={SettingsIcon}
                            size="sm"
                          >
                            权限配置
                          </Button>
                          {perm.isAdmin && !builtin ? (
                            <Button
                              disabled={saving}
                              hasIconOnly
                              iconDescription="删除角色"
                              kind="danger--ghost"
                              onClick={e => handleDeleteRole(r, e)}
                              renderIcon={DeleteIcon}
                              size="sm"
                            />
                          ) : (
                            <span aria-hidden="true" className="row-action-placeholder" />
                          )}
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      ) : (
        /* ==================== VIEW 2: 用户配置列表 ==================== */
        <div className="rbac-master-detail-layout">
          <div className="rbac-horizontal-list-card">
            <table className="rbac-horizontal-list-table">
              <thead>
                <tr>
                  <th>用户账号</th>
                  <th>绑定角色</th>
                  <th>关联 ClusterRole</th>
                  <th>角色生效命名空间 (RoleBinding)</th>
                  <th style={{ width: '180px', textAlign: 'right' }}>操作</th>
                </tr>
              </thead>
              <tbody>
                {users.map(u => {
                  const isAdminUser = u.username === 'admin';
                  const isSelected =
                    isUserConfigOpen &&
                    String(u.id) === String(selectedUserId);
                  return (
                    <tr
                      className={`rbac-list-row ${
                        isSelected ? 'selected' : ''
                      }`}
                      key={u.id}
                      onClick={() => openUserConfigModal(u)}
                    >
                      <td>
                        <strong>{u.username}</strong>
                      </td>
                      <td>
                        <span
                          className={`role-pill role-${u.role || 'worker'}`}
                        >
                          {u.role_display || '工作人员'}
                        </span>
                      </td>
                      <td>
                        <code className="row-code-tag">
                          {u.k8s_role_name ||
                            `tekton-devops-${u.role || 'worker'}`}
                        </code>
                      </td>
                      <td>
                        <div style={{ display: 'flex', flexWrap: 'wrap', gap: '4px' }}>
                          {Array.isArray(u.bound_namespaces) &&
                          u.bound_namespaces.includes('*') ? (
                            <code className="row-code-tag">全部命名空间 (*)</code>
                          ) : (
                            (u.bound_namespaces || ['default']).map(ns => (
                              <code className="row-code-tag" key={ns}>
                                {ns}
                              </code>
                            ))
                          )}
                        </div>
                      </td>
                      <td
                        onClick={e => e.stopPropagation()}
                        style={{ textAlign: 'right' }}
                      >
                        <div className="row-action-group">
                          <Button
                            kind="ghost"
                            onClick={() => openUserConfigModal(u)}
                            renderIcon={SettingsIcon}
                            size="sm"
                          >
                            角色绑定
                          </Button>
                          {perm.isAdmin && !isAdminUser ? (
                            <Button
                              disabled={saving}
                              hasIconOnly
                              iconDescription="删除用户"
                              kind="danger--ghost"
                              onClick={e => handleDeleteUser(u, e)}
                              renderIcon={DeleteIcon}
                              size="sm"
                            />
                          ) : (
                            <span aria-hidden="true" className="row-action-placeholder" />
                          )}
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* ==================== MODAL 1: 角色详细权限与 Namespace RoleBinding 配置弹窗 ==================== */}
      <Modal
        className="platform-modal rbac-config-modal-wide"
        modalHeading={
          roleDraft ? `${roleDraft.display_name} · 角色与命名空间配置` : '角色权限配置'
        }
        onRequestClose={() => setIsRoleConfigOpen(false)}
        onRequestSubmit={handleSaveRoleDetail}
        open={isRoleConfigOpen}
        primaryButtonDisabled={saving}
        primaryButtonText={saving ? '保存中...' : '保存配置'}
        secondaryButtonText="取消"
      >
        {roleDraft && (
          <div className="platform-modal-form">
            <div className="form-grid-2col">
              <TextInput
                id="modal-role-display"
                labelText="角色名称"
                onChange={e =>
                  setRoleDraft({ ...roleDraft, display_name: e.target.value })
                }
                value={roleDraft.display_name}
              />
              <TextInput
                id="modal-role-desc"
                labelText="简要说明"
                onChange={e =>
                  setRoleDraft({ ...roleDraft, description: e.target.value })
                }
                value={roleDraft.description}
              />
            </div>

            {/* Namespace-level RoleBinding selector on the Role */}
            <div className="user-ns-config-box">
              <div className="ns-box-head">
                <strong>生效命名空间 (自动在目标 Namespace 创建 RoleBinding)</strong>
                {roleDraft.role_type !== 'admin' &&
                  roleDraft.name !== 'tekton-devops-admin' && (
                    <Checkbox
                      checked={roleDraft.allNamespaces}
                      id="modal-role-all-ns"
                      labelText="全部命名空间 (ClusterRoleBinding)"
                      onChange={(_, { checked }) =>
                        setRoleDraft({ ...roleDraft, allNamespaces: checked })
                      }
                    />
                  )}
              </div>

              {roleDraft.role_type === 'admin' ||
              roleDraft.name === 'tekton-devops-admin' ||
              roleDraft.allNamespaces ? (
                <p className="ns-all-granted-note">
                  通过 ClusterRoleBinding 对集群内全部命名空间生效。
                </p>
              ) : (
                <div className="ns-checkbox-grid">
                  {namespaces.map(ns => (
                    <Checkbox
                      checked={roleDraft.selectedNamespaces.includes(ns.name)}
                      id={`modal-role-ns-${ns.name}`}
                      key={ns.name}
                      labelText={ns.name}
                      onChange={(_, { checked }) =>
                        toggleRoleDraftNS(ns.name, checked)
                      }
                    />
                  ))}
                </div>
              )}
            </div>

            <div className="rbac-matrix-table-wrap">
              <table className="rbac-matrix-table">
                <thead>
                  <tr>
                    <th>功能模块 (ClusterRole 规则)</th>
                    <th>查看</th>
                    <th>创建</th>
                    <th>编辑</th>
                    <th>删除</th>
                    <th>触发运行</th>
                    <th>快捷设置</th>
                  </tr>
                </thead>
                <tbody>
                  {PERMISSION_MODULES.map(mod => {
                    const mp = roleDraft.permissions?.[mod.key] || {};
                    return (
                      <tr key={mod.key}>
                        <td className="mod-name-cell">
                          <strong>{mod.label}</strong>
                        </td>
                        <td>
                          <Checkbox
                            checked={!!mp.get}
                            id={`modal-role-perm-${mod.key}-get`}
                            labelText="允许"
                            onChange={(_, { checked }) =>
                              toggleRolePerm(mod.key, 'get', checked)
                            }
                          />
                        </td>
                        <td>
                          <Checkbox
                            checked={!!mp.create}
                            id={`modal-role-perm-${mod.key}-create`}
                            labelText="允许"
                            onChange={(_, { checked }) =>
                              toggleRolePerm(mod.key, 'create', checked)
                            }
                          />
                        </td>
                        <td>
                          <Checkbox
                            checked={!!mp.update}
                            id={`modal-role-perm-${mod.key}-update`}
                            labelText="允许"
                            onChange={(_, { checked }) =>
                              toggleRolePerm(mod.key, 'update', checked)
                            }
                          />
                        </td>
                        <td>
                          <Checkbox
                            checked={!!mp.delete}
                            id={`modal-role-perm-${mod.key}-delete`}
                            labelText="允许"
                            onChange={(_, { checked }) =>
                              toggleRolePerm(mod.key, 'delete', checked)
                            }
                          />
                        </td>
                        <td>
                          {mod.supportsRun ? (
                            <Checkbox
                              checked={!!mp.run}
                              id={`modal-role-perm-${mod.key}-run`}
                              labelText="允许"
                              onChange={(_, { checked }) =>
                                toggleRolePerm(mod.key, 'run', checked)
                              }
                            />
                          ) : (
                            <span style={{ color: '#94a3b8' }}>-</span>
                          )}
                        </td>
                        <td>
                          <div className="matrix-quick-btns">
                            <button
                              onClick={() => setModulePreset(mod.key, 'full')}
                              type="button"
                            >
                              全权
                            </button>
                            <button
                              onClick={() =>
                                setModulePreset(mod.key, 'readonly')
                              }
                              type="button"
                            >
                              只读
                            </button>
                          </div>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>

            {perm.isAdmin && !roleDraft.is_builtin && (
              <div className="modal-danger-zone">
                <span>自定义角色支持从集群中移除</span>
                <Button
                  disabled={saving}
                  kind="danger--ghost"
                  onClick={() => handleDeleteRole(roleDraft)}
                  renderIcon={DeleteIcon}
                  size="sm"
                >
                  删除此角色
                </Button>
              </div>
            )}
          </div>
        )}
      </Modal>

      {/* ==================== MODAL 2: 用户角色绑定配置弹窗 ==================== */}
      <Modal
        className="platform-modal"
        modalHeading={
          userDraft ? `用户「${userDraft.username}」 · 角色绑定` : '用户角色绑定'
        }
        onRequestClose={() => setIsUserConfigOpen(false)}
        onRequestSubmit={handleSaveUserDetail}
        open={isUserConfigOpen}
        primaryButtonDisabled={saving || userDraft?.username === 'admin'}
        primaryButtonText={saving ? '保存中...' : '保存配置'}
        secondaryButtonText="取消"
      >
        {userDraft && (
          <div className="platform-modal-form">
            <div className="form-grid-2col">
              <TextInput
                disabled
                id="modal-user-name"
                labelText="用户账号"
                value={userDraft.username}
              />
              <Select
                disabled={userDraft.username === 'admin'}
                id="modal-user-role"
                labelText="绑定角色"
                onChange={e => handleUserDraftRoleChange(e.target.value)}
                value={userDraft.role_name}
              >
                {roles.map(r => (
                  <SelectItem
                    key={r.name}
                    text={`${r.display_name} (${r.name})`}
                    value={r.name}
                  />
                ))}
              </Select>
            </div>

            {(() => {
              const selectedRoleObj = roles.find(
                r => r.name === userDraft.role_name
              );
              const inheritedNS = Array.isArray(
                selectedRoleObj?.bound_namespaces
              )
                ? selectedRoleObj.bound_namespaces
                : userDraft.role === 'admin'
                  ? ['*']
                  : ['default'];
              return (
                <div className="user-ns-config-box">
                  <div className="ns-box-head">
                    <strong>该角色已绑定的命名空间 (RoleBinding)</strong>
                  </div>
                  <div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px' }}>
                    {inheritedNS.includes('*') ? (
                      <code className="row-code-tag">全部命名空间 (*)</code>
                    ) : (
                      inheritedNS.map(ns => (
                        <code className="row-code-tag" key={ns}>
                          {ns}
                        </code>
                      ))
                    )}
                  </div>
                </div>
              );
            })()}

            {perm.isAdmin && userDraft.username !== 'admin' && (
              <div className="modal-danger-zone">
                <span>删除后该用户将无法登录平台</span>
                <Button
                  disabled={saving}
                  kind="danger--ghost"
                  onClick={() => handleDeleteUser(userDraft)}
                  renderIcon={DeleteIcon}
                  size="sm"
                >
                  删除此用户
                </Button>
              </div>
            )}
          </div>
        )}
      </Modal>

      {/* ==================== MODAL 3: 新建角色弹窗 ==================== */}
      <Modal
        className="platform-modal"
        modalHeading="新建角色"
        onRequestClose={() => setIsNewRoleOpen(false)}
        onRequestSubmit={handleCreateNewRole}
        open={isNewRoleOpen}
        primaryButtonDisabled={saving}
        primaryButtonText={saving ? '创建中...' : '确定创建'}
        secondaryButtonText="取消"
      >
        <div className="platform-modal-form">
          <div className="form-grid-2col">
            <TextInput
              id="new-role-id"
              labelText="角色标识 (ClusterRole) *"
              onChange={e =>
                setNewRoleForm({ ...newRoleForm, name: e.target.value })
              }
              placeholder="例如: release-operator"
              value={newRoleForm.name}
            />
            <TextInput
              id="new-role-display"
              labelText="角色名称 *"
              onChange={e =>
                setNewRoleForm({
                  ...newRoleForm,
                  display_name: e.target.value
                })
              }
              placeholder="例如: 发布专员"
              value={newRoleForm.display_name}
            />
          </div>
          <Select
            id="new-role-type"
            labelText="权限模板"
            onChange={e =>
              setNewRoleForm({ ...newRoleForm, role_type: e.target.value })
            }
            value={newRoleForm.role_type}
          >
            <SelectItem text="工作人员 (默认读写权限)" value="worker" />
            <SelectItem text="只读角色 (默认只读权限)" value="readonly" />
          </Select>
          <TextInput
            id="new-role-desc"
            labelText="简要说明"
            onChange={e =>
              setNewRoleForm({ ...newRoleForm, description: e.target.value })
            }
            placeholder="选填"
            value={newRoleForm.description}
          />

          <div className="user-ns-config-box">
            <div className="ns-box-head">
              <strong>生效命名空间 (创建对应 Namespace RoleBinding)</strong>
              <Checkbox
                checked={newRoleForm.allNamespaces}
                id="new-role-all-ns"
                labelText="全部命名空间"
                onChange={(_, { checked }) =>
                  setNewRoleForm({ ...newRoleForm, allNamespaces: checked })
                }
              />
            </div>
            {!newRoleForm.allNamespaces && (
              <div className="ns-checkbox-grid">
                {namespaces.map(ns => (
                  <Checkbox
                    checked={newRoleForm.selectedNamespaces.includes(ns.name)}
                    id={`new-role-ns-${ns.name}`}
                    key={ns.name}
                    labelText={ns.name}
                    onChange={(_, { checked }) => {
                      const set = new Set(newRoleForm.selectedNamespaces);
                      if (checked) set.add(ns.name);
                      else set.delete(ns.name);
                      setNewRoleForm({
                        ...newRoleForm,
                        selectedNamespaces: Array.from(set)
                      });
                    }}
                  />
                ))}
              </div>
            )}
          </div>
        </div>
      </Modal>

      {/* ==================== MODAL 4: 新建用户弹窗 ==================== */}
      <Modal
        className="platform-modal"
        modalHeading="新建用户"
        onRequestClose={() => setIsNewUserOpen(false)}
        onRequestSubmit={handleCreateNewUser}
        open={isNewUserOpen}
        primaryButtonDisabled={saving}
        primaryButtonText={saving ? '创建中...' : '确定创建'}
        secondaryButtonText="取消"
      >
        <div className="platform-modal-form">
          <div className="form-grid-2col">
            <TextInput
              id="new-user-username"
              labelText="账号 *"
              onChange={e =>
                setNewUserForm({ ...newUserForm, username: e.target.value })
              }
              placeholder="请输入账号"
              value={newUserForm.username}
            />
            <TextInput
              id="new-user-password"
              labelText="初始密码 *"
              onChange={e =>
                setNewUserForm({ ...newUserForm, password: e.target.value })
              }
              placeholder="请输入初始密码"
              type="password"
              value={newUserForm.password}
            />
          </div>

          <Select
            id="new-user-role"
            labelText="分配角色"
            onChange={e => {
              const rName = e.target.value;
              const matched = roles.find(r => r.name === rName);
              const rType = matched?.role_type || 'worker';
              setNewUserForm({
                ...newUserForm,
                role: rType,
                role_name: rName
              });
            }}
            value={newUserForm.role_name}
          >
            {roles.map(r => (
              <SelectItem
                key={r.name}
                text={`${r.display_name} (${r.name})`}
                value={r.name}
              />
            ))}
          </Select>
        </div>
      </Modal>
    </div>
  );
}

export default PlatformRBAC;
