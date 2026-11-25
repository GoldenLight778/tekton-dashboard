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

import { useState } from 'react';
import { useNavigate } from 'react-router';
import {
  Button,
  InlineNotification,
  PasswordInput,
  TextInput
} from '@carbon/react';
import { Login as LoginIcon } from '@carbon/react/icons';
import { useQueryClient } from '@tanstack/react-query';
import { Icons } from '@tektoncd/dashboard-components';

import { clearToken, login } from '../../api/platform';

const { TektonDashboardColor: TektonLogo } = Icons;

export function PlatformLogin() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const handleSubmit = async e => {
    e.preventDefault();
    setError(null);
    if (!username.trim() || !password.trim()) {
      setError('请输入登录账号和密码');
      return;
    }
    setLoading(true);
    try {
      clearToken();
      await login(username.trim(), password);
      queryClient.invalidateQueries();
      navigate('/platform/dashboard', { replace: true });
    } catch (err) {
      setError(err.message || '账号或密码错误，请重新输入');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="platform-standalone-login">
      <div className="standalone-login-card">
        <div className="login-card-brand">
          <div className="login-logo-box">
            <TektonLogo />
          </div>
          <h1>Tekton DevOps 平台</h1>
          <p>请输入平台账号与密码登录</p>
        </div>

        {error && (
          <InlineNotification
            kind="error"
            lowContrast
            onCloseButtonClick={() => setError(null)}
            style={{ marginBottom: '1.25rem', maxWidth: '100%' }}
            subtitle={error}
            title="登录失败"
          />
        )}

        <form className="login-form-body" onSubmit={handleSubmit}>
          <div className="form-field-group">
            <TextInput
              autoComplete="username"
              id="login-username"
              labelText="账号"
              onChange={e => setUsername(e.target.value)}
              placeholder="请输入账号"
              value={username}
            />
          </div>

          <div className="form-field-group">
            <PasswordInput
              autoComplete="current-password"
              id="login-password"
              labelText="密码"
              onChange={e => setPassword(e.target.value)}
              placeholder="请输入密码"
              value={password}
            />
          </div>

          <div className="login-form-actions">
            <Button
              className="login-submit-btn"
              disabled={loading}
              kind="primary"
              renderIcon={LoginIcon}
              type="submit"
            >
              {loading ? '正在登录...' : '登录'}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}

export default PlatformLogin;
