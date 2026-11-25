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
import { handleMockRequest, isMockEnabled } from '../mock';

const TOKEN_KEY = 'devops_platform_token';
const USER_KEY = 'devops_platform_user';
const API_BASE = '/api/platform/v1';

export function getToken() {
  try {
    return localStorage.getItem(TOKEN_KEY) || '';
  } catch {
    return '';
  }
}

export function setToken(token) {
  try {
    if (token) {
      localStorage.setItem(TOKEN_KEY, token);
    } else {
      localStorage.removeItem(TOKEN_KEY);
    }
  } catch {
    // ignore local storage errors
  }
}

export function getCurrentUser() {
  try {
    const raw = localStorage.getItem(USER_KEY);
    if (raw) {
      return JSON.parse(raw);
    }
  } catch {
    // ignore
  }
  return { username: 'admin', role: 'admin', auth_mode: 'password' };
}

export function setCurrentUser(user) {
  try {
    if (user) {
      localStorage.setItem(USER_KEY, JSON.stringify(user));
    } else {
      localStorage.removeItem(USER_KEY);
    }
    if (typeof window !== 'undefined') {
      window.dispatchEvent(new CustomEvent('tkn-auth-change'));
    }
  } catch {
    // ignore
  }
}

export function clearToken() {
  setToken('');
  setCurrentUser(null);
}

let loginPromise = null;

export async function autoLogin() {
  if (getToken()) {
    return getToken();
  }
  if (!loginPromise) {
    loginPromise = (async () => {
      try {
        const response = await fetch(`${API_BASE}/auth/login`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ username: 'admin', password: 'admin123' })
        });
        const resJson = await response.json().catch(() => null);
        if (resJson?.data?.token) {
          setToken(resJson.data.token);
          return resJson.data.token;
        }
      } catch {
        // ignore network error
      }
      return '';
    })().finally(() => {
      loginPromise = null;
    });
  }
  return loginPromise;
}

async function request(endpoint, options = {}, isRetry = false) {
  const fullUri = `${API_BASE}${endpoint}`;

  if (isMockEnabled()) {
    const mockResponse = handleMockRequest(fullUri, options);
    if (mockResponse !== undefined) {
      return mockResponse;
    }
  }

  const isTest =
    (typeof process !== 'undefined' && process.env?.NODE_ENV === 'test') ||
    import.meta.env?.MODE === 'test';
  const token = getToken();

  const headers = {
    'Content-Type': 'application/json',
    ...(options.headers || {})
  };

  if (token) {
    headers.Authorization = `Bearer ${token}`;
    headers['X-Platform-Token'] = token;
  }

  try {
    const response = await fetch(fullUri, {
      ...options,
      headers
    });

    if (response.status === 401 && !isTest && endpoint !== '/auth/login') {
      clearToken();
      if (
        typeof window !== 'undefined' &&
        !window.location.hash.startsWith('#/login')
      ) {
        window.location.hash = '#/login';
      }
    }

    const data = await response.json().catch(() => null);

    if (!response.ok) {
      const errorMsg =
        data?.message || `HTTP ${response.status} ${response.statusText}`;
      const err = new Error(errorMsg);
      err.status = response.status;
      err.code = data?.code;
      throw err;
    }

    return data?.data !== undefined ? data.data : data;
  } catch (err) {
    if (isMockEnabled()) {
      const fallbackMock = handleMockRequest(fullUri, options);
      if (fallbackMock !== undefined) {
        return fallbackMock;
      }
    }
    throw err;
  }
}

// 1. Auth & K8s RBAC API
export async function login(username, password) {
  const data = await request('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ mode: 'password', username, password })
  });
  if (data?.token) {
    setToken(data.token);
    if (data.user) {
      setCurrentUser(data.user);
    }
  }
  return data;
}

export async function loginWithK8sToken(token) {
  const data = await request('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ mode: 'k8s_token', token })
  });
  if (data?.token) {
    setToken(data.token);
    if (data.user) {
      setCurrentUser(data.user);
    }
  }
  return data;
}

export async function getMe() {
  return request('/auth/me');
}

export async function getPermissions(namespace = 'ALL_NAMESPACES') {
  const nsParam = encodeURIComponent(namespace || 'ALL_NAMESPACES');
  return request(`/auth/permissions?namespace=${nsParam}`);
}

export function usePermissions(namespace = 'ALL_NAMESPACES') {
  const [perm, setPerm] = useState(() => {
    const u = getCurrentUser();
    const isAdmin = !u || u.role === 'admin';
    const isViewer = u?.role === 'viewer';
    return {
      username: u?.username || 'admin',
      auth_mode: u?.auth_mode || 'password',
      global_role: u?.role || 'admin',
      effective_role: u?.role || 'admin',
      k8s_cluster_role: `tekton-dashboard-${u?.role || 'admin'}`,
      read_only: isViewer,
      canWrite: !isViewer,
      isAdmin,
      rules: {
        pipelines: { create: !isViewer, delete: !isViewer, run: !isViewer },
        tasks: { create: !isViewer, delete: !isViewer, run: !isViewer },
        secrets: { create: !isViewer, delete: !isViewer },
        namespaces: { create: isAdmin, delete: isAdmin },
        rbac: { manage: isAdmin }
      }
    };
  });

  useEffect(() => {
    let mounted = true;
    const fetchPerm = async () => {
      try {
        const res = await getPermissions(namespace);
        if (mounted && res) {
          setPerm({
            ...res,
            canWrite: !res.read_only,
            isAdmin:
              res.effective_role === 'admin' || res.global_role === 'admin'
          });
        }
      } catch {
        // fallback to current user stored role
      }
    };
    fetchPerm();

    const onAuthChange = () => {
      fetchPerm();
    };
    if (typeof window !== 'undefined') {
      window.addEventListener('tkn-auth-change', onAuthChange);
    }
    return () => {
      mounted = false;
      if (typeof window !== 'undefined') {
        window.removeEventListener('tkn-auth-change', onAuthChange);
      }
    };
  }, [namespace]);

  return perm;
}

export async function getVersion() {
  return request('/version');
}

// 1b. Native K8s Namespaces API
export async function getK8sNamespaces() {
  const data = await request('/namespaces');
  return data?.items !== undefined ? data.items : data || [];
}

export async function createK8sNamespace({ name, alias, description }) {
  return request('/namespaces', {
    method: 'POST',
    body: JSON.stringify({ name, alias, description })
  });
}

export async function deleteK8sNamespace(name) {
  return request(`/namespaces/${encodeURIComponent(name)}`, {
    method: 'DELETE'
  });
}

// 1c. Native K8s Secrets + ServiceAccount Auto-Binding API
export async function getK8sSecrets(namespace = 'ALL_NAMESPACES') {
  const ns =
    !namespace || namespace === '*' ? 'ALL_NAMESPACES' : encodeURIComponent(namespace);
  const data = await request(`/namespaces/${ns}/secrets`);
  return data?.items !== undefined ? data.items : data || [];
}

export async function createK8sSecret(namespace, payload) {
  const ns =
    !namespace || namespace === '*' || namespace === 'ALL_NAMESPACES'
      ? payload.namespace || 'default'
      : namespace;
  return request(`/namespaces/${encodeURIComponent(ns)}/secrets`, {
    method: 'POST',
    body: JSON.stringify({ ...payload, namespace: ns })
  });
}

export async function deleteK8sSecret(namespace, name) {
  const ns =
    !namespace || namespace === '*' || namespace === 'ALL_NAMESPACES'
      ? 'default'
      : namespace;
  return request(
    `/namespaces/${encodeURIComponent(ns)}/secrets/${encodeURIComponent(name)}`,
    {
      method: 'DELETE'
    }
  );
}

// 1d. K8s RBAC Users, Labeled ClusterRoles & RoleBindings API (Scoped to label `tekton-devops-dashboard=true`)
export async function getRBACRoles() {
  const data = await request('/rbac/roles');
  return data?.items !== undefined ? data.items : data || [];
}

export async function saveRBACRole(rolePayload) {
  const isUpdate = !!rolePayload.isUpdate && !!rolePayload.name;
  const endpoint = isUpdate
    ? `/rbac/roles/${encodeURIComponent(rolePayload.name)}`
    : '/rbac/roles';
  return request(endpoint, {
    method: isUpdate ? 'PUT' : 'POST',
    body: JSON.stringify(rolePayload)
  });
}

export async function deleteRBACRole(name) {
  return request(`/rbac/roles/${encodeURIComponent(name)}`, {
    method: 'DELETE'
  });
}

export async function getRBACUsers() {
  const data = await request('/rbac/users');
  return data?.items !== undefined ? data.items : data || [];
}

export async function createRBACUser({
  username,
  password,
  role,
  role_name,
  namespace,
  namespaces
}) {
  return request('/rbac/users', {
    method: 'POST',
    body: JSON.stringify({
      username,
      password,
      role,
      role_name,
      namespace,
      namespaces
    })
  });
}

export async function updateRBACUser(id, { role, role_name, namespaces }) {
  return request(`/rbac/users/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: JSON.stringify({ role, role_name, namespaces })
  });
}

export async function deleteRBACUser(id) {
  return request(`/rbac/users/${encodeURIComponent(id)}`, {
    method: 'DELETE'
  });
}

export async function getRBACBindings() {
  const data = await request('/rbac/bindings');
  return data?.items !== undefined ? data.items : data || [];
}

export async function createRBACBinding({ username, namespace, role_name }) {
  return request('/rbac/bindings', {
    method: 'POST',
    body: JSON.stringify({ username, namespace, role_name })
  });
}

export async function deleteRBACBinding(id) {
  return request(`/rbac/bindings/${encodeURIComponent(id)}`, {
    method: 'DELETE'
  });
}

// 2. Project API
export async function getProjects() {
  const data = await request('/projects');
  return data?.items !== undefined ? data.items : data;
}

export async function getProject(id) {
  return request(`/projects/${id}`);
}

export async function createProject({
  name,
  description,
  git_url,
  default_branch,
  namespace
}) {
  return request('/projects', {
    method: 'POST',
    body: JSON.stringify({
      name,
      description,
      git_url,
      default_branch,
      namespace
    })
  });
}

export async function updateProject(
  id,
  { name, description, git_url, default_branch, namespace }
) {
  return request(`/projects/${id}`, {
    method: 'PUT',
    body: JSON.stringify({
      name,
      description,
      git_url,
      default_branch,
      namespace
    })
  });
}

export async function deleteProject(id) {
  return request(`/projects/${id}`, {
    method: 'DELETE'
  });
}

// 3. Pipeline API
export async function getPipelines(
  projectId,
  { page = 1, pageSize = 20 } = {}
) {
  return request(
    `/projects/${projectId}/pipelines?page=${page}&pageSize=${pageSize}`
  );
}

export async function getPipeline(id) {
  return request(`/pipelines/${id}`);
}

export async function createPipeline(projectId, { name, dsl_content }) {
  return request(`/projects/${projectId}/pipelines`, {
    method: 'POST',
    body: JSON.stringify({ name, dsl_content })
  });
}

export async function updatePipeline(id, { name, dsl_content }) {
  return request(`/pipelines/${id}`, {
    method: 'PUT',
    body: JSON.stringify({ name, dsl_content })
  });
}

export async function deletePipeline(id) {
  return request(`/pipelines/${id}`, {
    method: 'DELETE'
  });
}

// 4. Pipeline Run API
export async function triggerPipelineRun(pipelineId) {
  return request(`/pipelines/${pipelineId}/runs`, {
    method: 'POST'
  });
}

export async function getPipelineRuns(
  pipelineId,
  { page = 1, pageSize = 20 } = {}
) {
  return request(
    `/pipelines/${pipelineId}/runs?page=${page}&pageSize=${pageSize}`
  );
}

export async function getProjectRuns(
  projectId,
  { page = 1, pageSize = 20 } = {}
) {
  return request(
    `/projects/${projectId}/runs?page=${page}&pageSize=${pageSize}`
  );
}

export async function getRun(id) {
  return request(`/runs/${id}`);
}

export async function cancelRun(id) {
  return request(`/runs/${id}/cancel`, {
    method: 'POST'
  });
}

// 5. Credential API
export async function getCredentials(projectId) {
  return request(`/projects/${projectId}/credentials`);
}

export async function createCredential(projectId, { name, type, secret }) {
  return request(`/projects/${projectId}/credentials`, {
    method: 'POST',
    body: JSON.stringify({ name, type, secret })
  });
}

export async function deleteCredential(id) {
  return request(`/credentials/${id}`, {
    method: 'DELETE'
  });
}

// 6. Metrics API
export async function getDashboardMetrics(namespace = '') {
  const q =
    namespace && namespace !== '*' && namespace !== 'ALL_NAMESPACES'
      ? `?namespace=${encodeURIComponent(namespace)}`
      : '';
  return request(`/metrics/dashboard${q}`);
}

// 7. Audit Log API
export async function getAuditLogs({ page = 1, pageSize = 20 } = {}) {
  return request(`/audit-logs?page=${page}&pageSize=${pageSize}`);
}
