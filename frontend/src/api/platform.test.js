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

import { beforeEach, describe, expect, it, vi } from 'vitest';
import * as platformAPI from './platform';

describe('Platform API Client', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    platformAPI.clearToken();
  });

  it('manages JWT token storage correctly', () => {
    expect(platformAPI.getToken()).toBe('');
    platformAPI.setToken('test-token-xyz');
    expect(platformAPI.getToken()).toBe('test-token-xyz');
    platformAPI.clearToken();
    expect(platformAPI.getToken()).toBe('');
  });

  it('handles token storage exceptions gracefully', () => {
    const getItemSpy = vi
      .spyOn(Storage.prototype, 'getItem')
      .mockImplementation(() => {
        throw new Error('localStorage error');
      });
    expect(platformAPI.getToken()).toBe('');
    getItemSpy.mockRestore();

    const setItemSpy = vi
      .spyOn(Storage.prototype, 'setItem')
      .mockImplementation(() => {
        throw new Error('localStorage error');
      });
    expect(() => platformAPI.setToken('val')).not.toThrow();
    setItemSpy.mockRestore();
  });

  it('performs login and stores token on success', async () => {
    const mockToken = 'mocked-jwt-token';
    vi.spyOn(global, 'fetch').mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        code: 0,
        message: 'ok',
        data: { token: mockToken, user: { username: 'admin', role: 'admin' } }
      })
    });

    const data = await platformAPI.login('admin', 'admin123');
    expect(data.token).toBe(mockToken);
    expect(platformAPI.getToken()).toBe(mockToken);
  });

  it('handles error response by throwing descriptive error', async () => {
    vi.spyOn(global, 'fetch').mockResolvedValueOnce({
      ok: false,
      status: 401,
      statusText: 'Unauthorized',
      json: async () => ({
        code: 40103,
        message: 'invalid or expired token'
      })
    });

    await expect(platformAPI.getMe()).rejects.toThrow(
      'invalid or expired token'
    );
  });

  it('handles error response without json body gracefully', async () => {
    vi.spyOn(global, 'fetch').mockResolvedValueOnce({
      ok: false,
      status: 500,
      statusText: 'Internal Error',
      json: async () => {
        throw new Error('not json');
      }
    });

    await expect(platformAPI.getMe()).rejects.toThrow(
      'HTTP 500 Internal Error'
    );
  });

  it('calls getMe and getVersion', async () => {
    vi.spyOn(global, 'fetch')
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { username: 'admin' } })
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { version: '0.1.0' } })
      });

    const me = await platformAPI.getMe();
    expect(me).toEqual({ username: 'admin' });

    const ver = await platformAPI.getVersion();
    expect(ver).toEqual({ version: '0.1.0' });
  });

  it('covers project CRUD operations', async () => {
    platformAPI.setToken('auth-token');
    vi.spyOn(global, 'fetch')
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: [{ id: '1' }] })
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { id: '1' } })
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { id: '2' } })
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { id: '2', name: 'new' } })
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { deleted: true } })
      });

    expect(await platformAPI.getProjects()).toEqual([{ id: '1' }]);
    expect(await platformAPI.getProject('1')).toEqual({ id: '1' });
    expect(await platformAPI.createProject({ name: 'p2' })).toEqual({
      id: '2'
    });
    expect(await platformAPI.updateProject('2', { name: 'new' })).toEqual({
      id: '2',
      name: 'new'
    });
    expect(await platformAPI.deleteProject('2')).toEqual({ deleted: true });
  });

  it('covers pipeline CRUD operations', async () => {
    vi.spyOn(global, 'fetch')
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { items: [] } })
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { id: 'pipe-1' } })
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { id: 'pipe-2' } })
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { id: 'pipe-2', name: 'up' } })
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { deleted: true } })
      });

    expect(await platformAPI.getPipelines('proj-1')).toEqual({ items: [] });
    expect(await platformAPI.getPipeline('pipe-1')).toEqual({ id: 'pipe-1' });
    expect(
      await platformAPI.createPipeline('proj-1', { name: 'pipe-2' })
    ).toEqual({ id: 'pipe-2' });
    expect(await platformAPI.updatePipeline('pipe-2', { name: 'up' })).toEqual({
      id: 'pipe-2',
      name: 'up'
    });
    expect(await platformAPI.deletePipeline('pipe-2')).toEqual({
      deleted: true
    });
  });

  it('covers run trigger, list, cancel operations', async () => {
    vi.spyOn(global, 'fetch')
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { id: 'run-1' } })
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { items: [] } })
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { items: [] } })
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { id: 'run-1' } })
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { cancelled: true } })
      });

    expect(await platformAPI.triggerPipelineRun('pipe-1')).toEqual({
      id: 'run-1'
    });
    expect(await platformAPI.getPipelineRuns('pipe-1')).toEqual({ items: [] });
    expect(await platformAPI.getProjectRuns('proj-1')).toEqual({ items: [] });
    expect(await platformAPI.getRun('run-1')).toEqual({ id: 'run-1' });
    expect(await platformAPI.cancelRun('run-1')).toEqual({ cancelled: true });
  });

  it('covers credential operations and audit logs', async () => {
    vi.spyOn(global, 'fetch')
      .mockResolvedValueOnce({ ok: true, json: async () => ({ data: [] }) })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { id: 'c1' } })
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { deleted: true } })
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { items: [] } })
      });

    expect(await platformAPI.getCredentials('proj-1')).toEqual([]);
    expect(
      await platformAPI.createCredential('proj-1', {
        name: 'c1',
        type: 'git_token',
        secret: 'sec'
      })
    ).toEqual({ id: 'c1' });
    expect(await platformAPI.deleteCredential('c1')).toEqual({ deleted: true });
    expect(await platformAPI.getAuditLogs()).toEqual({ items: [] });
  });

  it('fetches dashboard metrics', async () => {
    const mockMetrics = {
      total_projects: 3,
      total_pipelines: 5,
      total_runs: 12,
      success_rate: 90.5
    };

    vi.spyOn(global, 'fetch').mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        code: 0,
        data: mockMetrics
      })
    });

    const res = await platformAPI.getDashboardMetrics();
    expect(res).toEqual(mockMetrics);
  });
});
