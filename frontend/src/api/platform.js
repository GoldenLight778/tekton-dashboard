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

import { handleMockRequest, isMockEnabled } from '../mock';

const TOKEN_KEY = 'devops_platform_token';
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

export function clearToken() {
  setToken('');
}

async function request(endpoint, options = {}) {
  const fullUri = `${API_BASE}${endpoint}`;

  if (isMockEnabled()) {
    const mockResponse = handleMockRequest(fullUri, options);
    if (mockResponse !== undefined) {
      return mockResponse;
    }
  }

  const headers = {
    'Content-Type': 'application/json',
    ...(options.headers || {})
  };

  const token = getToken();
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }

  try {
    const response = await fetch(fullUri, {
      ...options,
      headers
    });

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

// 1. Auth API
export async function login(username, password) {
  const data = await request('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ username, password })
  });
  if (data?.token) {
    setToken(data.token);
  }
  return data;
}

export async function getMe() {
  return request('/auth/me');
}

export async function getVersion() {
  return request('/version');
}

// 2. Project API
export async function getProjects() {
  return request('/projects');
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
export async function getDashboardMetrics() {
  return request('/metrics/dashboard');
}

// 7. Audit Log API
export async function getAuditLogs({ page = 1, pageSize = 20 } = {}) {
  return request(`/audit-logs?page=${page}&pageSize=${pageSize}`);
}
