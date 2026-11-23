/*
Copyright 2026 The Tekton Authors
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

import {
  mockLogs,
  mockNamespaces,
  mockPipelineRuns,
  mockPipelines,
  mockPlatformCredentials,
  mockPlatformMetrics,
  mockPlatformPipelines,
  mockPlatformProjects,
  mockProperties,
  mockTaskRuns,
  mockTasks
} from './data';

export function isMockEnabled() {
  return (
    import.meta.env.MODE === 'mock' || import.meta.env.VITE_APP_MOCK === 'true'
  );
}

function filterByNamespace(items, namespace) {
  if (!namespace) {
    return items;
  }
  return items.filter(
    item => !item.metadata?.namespace || item.metadata.namespace === namespace
  );
}

export function handleMockRequest(uri, _options = {}) {
  if (!isMockEnabled()) {
    return undefined;
  }

  const urlObj = new URL(uri, window.location.origin);
  const path = urlObj.pathname;

  // 1. Properties
  if (path.endsWith('/properties') || path.includes('/v1/properties')) {
    return Promise.resolve(mockProperties);
  }

  // 2. Namespaces
  if (path.endsWith('/api/v1/namespaces') || path === '/api/v1/namespaces/') {
    return Promise.resolve(mockNamespaces);
  }

  // 3. Extensions
  if (path.includes('/apis/dashboard.tekton.dev/v1alpha1/extensions')) {
    return Promise.resolve({ items: [], metadata: { resourceVersion: '1' } });
  }

  // 4. Pod Logs
  const logMatch = path.match(
    /\/api\/v1\/namespaces\/([^/]+)\/pods\/([^/]+)\/log/
  );
  if (logMatch) {
    const podName = logMatch[2];
    const logContent =
      mockLogs[podName] ||
      `[2026-11-20T03:30:00Z] Step executed successfully. Pod ${podName} completed.`;
    return Promise.resolve(logContent);
  }

  // 5. PipelineRuns
  const prDetailMatch = path.match(
    /\/apis\/tekton\.dev\/v1\/(?:namespaces\/([^/]+)\/)?pipelineruns\/([^/?#]+)/
  );
  if (prDetailMatch) {
    const [, , prName] = prDetailMatch;
    const found = mockPipelineRuns.items.find(
      item => item.metadata.name === prName
    );
    return Promise.resolve(found || mockPipelineRuns.items[0]);
  }
  const prListMatch = path.match(
    /\/apis\/tekton\.dev\/v1\/(?:namespaces\/([^/]+)\/)?pipelineruns\/?$/
  );
  if (prListMatch) {
    const ns = prListMatch[1];
    const filtered = filterByNamespace(mockPipelineRuns.items, ns);
    return Promise.resolve({
      ...mockPipelineRuns,
      items: filtered,
      metadata: { resourceVersion: '100' }
    });
  }

  // 6. Pipelines
  const pipeDetailMatch = path.match(
    /\/apis\/tekton\.dev\/v1\/(?:namespaces\/([^/]+)\/)?pipelines\/([^/?#]+)/
  );
  if (pipeDetailMatch) {
    const [, , pipeName] = pipeDetailMatch;
    const found = mockPipelines.items.find(
      item => item.metadata.name === pipeName
    );
    return Promise.resolve(found || mockPipelines.items[0]);
  }
  const pipeListMatch = path.match(
    /\/apis\/tekton\.dev\/v1\/(?:namespaces\/([^/]+)\/)?pipelines\/?$/
  );
  if (pipeListMatch) {
    const ns = pipeListMatch[1];
    const filtered = filterByNamespace(mockPipelines.items, ns);
    return Promise.resolve({
      ...mockPipelines,
      items: filtered,
      metadata: { resourceVersion: '100' }
    });
  }

  // 7. TaskRuns
  const trDetailMatch = path.match(
    /\/apis\/tekton\.dev\/v1\/(?:namespaces\/([^/]+)\/)?taskruns\/([^/?#]+)/
  );
  if (trDetailMatch) {
    const [, , trName] = trDetailMatch;
    const found = mockTaskRuns.items.find(
      item => item.metadata.name === trName
    );
    return Promise.resolve(found || mockTaskRuns.items[0]);
  }
  const trListMatch = path.match(
    /\/apis\/tekton\.dev\/v1\/(?:namespaces\/([^/]+)\/)?taskruns\/?$/
  );
  if (trListMatch) {
    const ns = trListMatch[1];
    const filtered = filterByNamespace(mockTaskRuns.items, ns);
    return Promise.resolve({
      ...mockTaskRuns,
      items: filtered,
      metadata: { resourceVersion: '100' }
    });
  }

  // 8. Tasks & ClusterTasks
  const taskDetailMatch = path.match(
    /\/apis\/tekton\.dev\/v1\/(?:namespaces\/([^/]+)\/)?tasks\/([^/?#]+)/
  );
  if (taskDetailMatch) {
    const [, , taskName] = taskDetailMatch;
    const found = mockTasks.items.find(item => item.metadata.name === taskName);
    return Promise.resolve(found || mockTasks.items[0]);
  }
  const taskListMatch = path.match(
    /\/apis\/tekton\.dev\/v1\/(?:namespaces\/([^/]+)\/)?tasks\/?$/
  );
  if (taskListMatch) {
    const ns = taskListMatch[1];
    const filtered = filterByNamespace(mockTasks.items, ns);
    return Promise.resolve({
      ...mockTasks,
      items: filtered,
      metadata: { resourceVersion: '100' }
    });
  }
  if (path.includes('/apis/tekton.dev/v1/clustertasks')) {
    return Promise.resolve({ items: [], metadata: { resourceVersion: '100' } });
  }

  // 9. Triggers
  if (path.includes('/apis/triggers.tekton.dev/')) {
    return Promise.resolve({ items: [], metadata: { resourceVersion: '100' } });
  }

  // 10. ServiceAccounts
  if (path.includes('/api/v1/') && path.includes('/serviceaccounts')) {
    return Promise.resolve({
      items: [
        { metadata: { name: 'default' } },
        { metadata: { name: 'tekton-pipeline-sa' } }
      ],
      metadata: { resourceVersion: '100' }
    });
  }

  // 11. Platform APIs
  if (path.includes('/api/platform/v1/metrics/dashboard')) {
    return Promise.resolve(mockPlatformMetrics);
  }
  if (path.includes('/api/platform/v1/projects')) {
    const singleMatch = path.match(/\/api\/platform\/v1\/projects\/([^/]+)$/);
    if (singleMatch) {
      const proj =
        mockPlatformProjects.find(p => p.id === singleMatch[1]) ||
        mockPlatformProjects[0];
      return Promise.resolve(proj);
    }
    return Promise.resolve(mockPlatformProjects);
  }
  if (
    path.includes('/api/platform/v1/pipelines') ||
    /\/api\/platform\/v1\/projects\/[^/]+\/pipelines/.test(path)
  ) {
    return Promise.resolve({
      items: mockPlatformPipelines,
      total: mockPlatformPipelines.length
    });
  }
  if (
    path.includes('/api/platform/v1/credentials') ||
    /\/api\/platform\/v1\/projects\/[^/]+\/credentials/.test(path)
  ) {
    return Promise.resolve(mockPlatformCredentials);
  }
  if (path.includes('/api/platform/v1/auth/me')) {
    return Promise.resolve({
      id: 'user-admin',
      username: 'admin',
      role: 'admin'
    });
  }
  if (path.includes('/api/platform/v1/version')) {
    return Promise.resolve({
      version: 'v1.0.0',
      git_commit: 'release',
      build_time: '2026-11-20'
    });
  }

  // Fallback for mock mode to prevent 404 or uncaught exceptions
  return Promise.resolve({ items: [], metadata: { resourceVersion: '100' } });
}

export function createMockWebSocket() {
  const listeners = {};
  return {
    addEventListener: (type, fn) => {
      listeners[type] = listeners[type] || [];
      listeners[type].push(fn);
      if (type === 'open') {
        setTimeout(() => fn({ type: 'open' }), 50);
      }
    },
    removeEventListener: (type, fn) => {
      if (!listeners[type]) {
        return;
      }
      listeners[type] = listeners[type].filter(l => l !== fn);
    },
    close: () => {},
    send: () => {}
  };
}
