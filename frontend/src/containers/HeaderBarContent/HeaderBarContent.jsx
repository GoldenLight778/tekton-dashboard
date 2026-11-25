/*
Copyright 2022-2026 The Tekton Authors
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
import { useEffect, useRef, useState } from 'react';
import {
  generatePath,
  useLocation,
  useMatches,
  useNavigate,
  useParams
} from 'react-router';
import { Modal } from '@carbon/react';
import {
  ChevronDown as ChevronDownIcon,
  Logout as LogoutIcon,
  UserAvatar as UserAvatarIcon
} from '@carbon/react/icons';
import { ALL_NAMESPACES, paths, urls } from '@tektoncd/dashboard-utils';

import LanguageSwitcher from '../LanguageSwitcher';
import NamespacesDropdown from '../NamespacesDropdown';
import { useSelectedNamespace, useTenantNamespaces } from '../../api';
import { clearToken, usePermissions } from '../../api/platform';

export default function HeaderBarContent() {
  const location = useLocation();
  const matches = useMatches();
  const match = matches.at(-1);

  const navigate = useNavigate();
  const params = useParams();
  const tenantNamespaces = useTenantNamespaces();

  const { selectedNamespace: namespace, selectNamespace } =
    useSelectedNamespace();
  const perm = usePermissions(namespace);

  const [menuOpen, setMenuOpen] = useState(false);
  const [profileModalOpen, setProfileModalOpen] = useState(false);
  const menuRef = useRef(null);

  useEffect(() => {
    if (params.namespace) {
      selectNamespace(params.namespace);
    } else if (tenantNamespaces.length) {
      selectNamespace(tenantNamespaces[0]);
    }
  }, [params.namespace, JSON.stringify(tenantNamespaces)]);

  useEffect(() => {
    const handleClickOutside = e => {
      if (menuRef.current && !menuRef.current.contains(e.target)) {
        setMenuOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  function setPath(path, { dropQueryParams } = {}) {
    navigate(`${path}${dropQueryParams ? '' : location.search}`);
  }

  function handleNamespaceSelected(event) {
    const newNamespace =
      event.selectedItem?.id || tenantNamespaces[0] || ALL_NAMESPACES;
    selectNamespace(newNamespace);

    if (!match.handle?.isNamespaced) {
      return;
    }

    if (newNamespace === ALL_NAMESPACES) {
      if (match.handle?.isResourceDetails) {
        setPath(
          location.pathname
            .replace(urls.byNamespace({ namespace }), '')
            .split('/')
            .slice(0, -1)
            .join('/'),
          {
            dropQueryParams: true
          }
        );
      } else {
        setPath(
          generatePath(match.handle.path.replace(paths.byNamespace(), ''), {
            ...match.params
          })
        );
      }
      return;
    }

    const prefix = match.params?.namespace ? '' : paths.byNamespace();

    const newURL = generatePath(`${prefix}${match.handle.path}`, {
      ...match.params,
      namespace: newNamespace
    });
    setPath(newURL);
  }

  const roleLabel =
    perm.effective_role === 'readonly' || perm.effective_role === 'viewer'
      ? '只读角色'
      : perm.effective_role === 'worker' || perm.effective_role === 'editor'
        ? '工作人员'
        : '平台管理员';

  const handleLogout = () => {
    setMenuOpen(false);
    clearToken();
    navigate('/login', { replace: true });
  };

  return (
    <div className="tkn--header-bar-right">
      <NamespacesDropdown
        id="header-namespace-dropdown"
        onChange={handleNamespaceSelected}
        selectedItem={{ id: namespace, text: namespace }}
        showAllNamespaces
        size="sm"
        titleText=""
      />
      <LanguageSwitcher />

      {/* Compact User Dropdown Menu */}
      <div className="tkn--header-user-menu" ref={menuRef}>
        <button
          aria-expanded={menuOpen}
          className="tkn--header-user-trigger"
          onClick={() => setMenuOpen(prev => !prev)}
          type="button"
        >
          <UserAvatarIcon size={16} />
          <span className="user-trigger-name">{perm.username || 'admin'}</span>
          <ChevronDownIcon
            className={`user-trigger-chevron ${menuOpen ? 'open' : ''}`}
            size={12}
          />
        </button>

        {menuOpen && (
          <div className="tkn--header-user-dropdown">
            <div className="dropdown-user-summary">
              <strong>{perm.username || 'admin'}</strong>
              <span className="dropdown-role-badge">{roleLabel}</span>
            </div>
            <button
              className="dropdown-action-item"
              onClick={() => {
                setMenuOpen(false);
                setProfileModalOpen(true);
              }}
              type="button"
            >
              <UserAvatarIcon size={15} />
              <span>个人信息</span>
            </button>
            <button
              className="dropdown-action-item danger"
              onClick={handleLogout}
              type="button"
            >
              <LogoutIcon size={15} />
              <span>退出登录</span>
            </button>
          </div>
        )}
      </div>

      {/* Personal Profile Modal */}
      <Modal
        className="platform-modal"
        modalHeading="个人信息"
        onRequestClose={() => setProfileModalOpen(false)}
        onRequestSubmit={() => setProfileModalOpen(false)}
        open={profileModalOpen}
        passiveModal
      >
        <div className="user-profile-modal-body">
          <div className="profile-info-row">
            <span className="profile-label">登录账号</span>
            <strong>{perm.username || 'admin'}</strong>
          </div>
          <div className="profile-info-row">
            <span className="profile-label">所属角色</span>
            <span
              className={`role-pill role-${perm.effective_role || 'admin'}`}
            >
              {roleLabel}
            </span>
          </div>
          <div className="profile-info-row">
            <span className="profile-label">当前命名空间</span>
            <code className="row-code-tag">
              {namespace === ALL_NAMESPACES || !namespace
                ? 'All Namespaces'
                : namespace}
            </code>
          </div>
          <div className="profile-info-row">
            <span className="profile-label">可访问命名空间</span>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: '4px', justifyContent: 'flex-end' }}>
              {Array.isArray(perm.allowed_namespaces) &&
              perm.allowed_namespaces.length > 0 ? (
                perm.allowed_namespaces.map(ns => (
                  <code className="row-code-tag" key={ns}>
                    {ns}
                  </code>
                ))
              ) : (
                <code className="row-code-tag">全部命名空间 (*)</code>
              )}
            </div>
          </div>
          <div className="profile-info-row">
            <span className="profile-label">操作权限状态</span>
            <span>{perm.canWrite ? '读写与执行权限' : '只读权限'}</span>
          </div>
        </div>
      </Modal>
    </div>
  );
}
