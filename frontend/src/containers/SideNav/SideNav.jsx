/*
Copyright 2019-2026 The Tekton Authors
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

import { matchPath, NavLink, useLocation } from 'react-router';
import { useIntl } from 'react-intl';
import {
  SideNav as CarbonSideNav,
  SideNavItems,
  SideNavLink,
  SideNavMenu,
  SideNavMenuItem,
  Theme
} from '@carbon/react';
import {
  Information as AboutIcon,
  Dashboard as DashboardIcon,
  Events as EventsIcon,
  Chip as ExtensionsIcon,
  DocumentImport as ImportResourcesIcon,
  ApiKey as KeyIcon,
  Rocket as RocketIcon,
  Settings as SettingsIcon,
  Task as TaskIcon
} from '@carbon/react/icons';
import { ALL_NAMESPACES, urls } from '@tektoncd/dashboard-utils';
import { Icons } from '@tektoncd/dashboard-components';

import {
  useExtensions,
  useIsReadOnly,
  useIsTriggersInstalled,
  useSelectedNamespace,
  useTenantNamespaces
} from '../../api';
import { usePermissions } from '../../api/platform';

const { Kubernetes: KubernetesIcon } = Icons;

function SideNav({ expanded, showKubernetesResources = false }) {
  const intl = useIntl();
  if (!expanded) {
    return null;
  }

  const location = useLocation();

  const { selectedNamespace } = useSelectedNamespace();
  const perm = usePermissions(selectedNamespace);
  const tenantNamespaces = useTenantNamespaces();
  const { data: extensions = [] } = useExtensions(
    {
      namespace: tenantNamespaces[0] || ALL_NAMESPACES
    },
    { disableWebSocket: true }
  );

  function getMenuItemProps(to) {
    return {
      as: NavLink,
      isActive: !!matchPath(
        {
          path: to
        },
        location.pathname
      ),
      to
    };
  }

  function getPath(path, namespaced = true) {
    if (
      namespaced &&
      selectedNamespace &&
      selectedNamespace !== ALL_NAMESPACES
    ) {
      return urls.byNamespace({
        namespace: selectedNamespace,
        path
      });
    }

    return path;
  }

  const isReadOnly = useIsReadOnly();
  const isTriggersInstalled = useIsTriggersInstalled();
  // eslint-disable-next-line formatjs/enforce-default-message
  const t = (id, defaultMessage) => intl.formatMessage({ id, defaultMessage });

  return (
    <Theme
      aria-label="Main navigation"
      as={CarbonSideNav}
      expanded={expanded}
      isFixedNav
      theme="g100"
    >
      <SideNavItems>
        {/* 1. 工作空间与大盘 */}
        <SideNavMenu
          defaultExpanded
          renderIcon={DashboardIcon}
          title={t('platform.nav.workspace', '工作台')}
        >
          <SideNavMenuItem {...getMenuItemProps('/platform/dashboard')}>
            {t('platform.nav.overview', '平台大盘')}
          </SideNavMenuItem>
          <SideNavMenuItem {...getMenuItemProps('/platform/namespaces')}>
            {t('platform.nav.namespaces', '命名空间 (Namespaces)')}
          </SideNavMenuItem>
        </SideNavMenu>

        {/* 2. 任务管理 - 整合 Tekton Tasks & StepActions */}
        <SideNavMenu
          defaultExpanded
          renderIcon={TaskIcon}
          title={t('platform.nav.tasksActions', '任务管理')}
        >
          <SideNavMenuItem {...getMenuItemProps(getPath(urls.tasks.all()))}>
            {t('platform.nav.tasks', '任务库')}
          </SideNavMenuItem>
          <SideNavMenuItem {...getMenuItemProps(getPath(urls.taskRuns.all()))}>
            {t('platform.nav.taskRuns', '任务执行')}
          </SideNavMenuItem>
          <SideNavMenuItem
            {...getMenuItemProps(getPath(urls.stepActions.all()))}
          >
            {t('platform.nav.stepActions', '步骤动作')}
          </SideNavMenuItem>
          <SideNavMenuItem
            {...getMenuItemProps(getPath(urls.customRuns.all()))}
          >
            {t('platform.nav.customRuns', '自定义执行')}
          </SideNavMenuItem>
        </SideNavMenu>

        {/* 3. 流水线中心 - 纯正 Tekton 原生流水线与执行 */}
        <SideNavMenu
          defaultExpanded
          renderIcon={RocketIcon}
          title={t('platform.nav.pipelines', '流水线中心')}
        >
          <SideNavMenuItem {...getMenuItemProps(getPath(urls.pipelines.all()))}>
            {t('platform.nav.tektonPipelines', 'Tekton 流水线')}
          </SideNavMenuItem>
          <SideNavMenuItem
            {...getMenuItemProps(getPath(urls.pipelineRuns.all()))}
          >
            {t('platform.nav.pipelineRuns', '流水线执行')}
          </SideNavMenuItem>
        </SideNavMenu>

        {/* 4. 自动化与触发器 - 整合 Tekton Triggers */}
        {isTriggersInstalled && (
          <SideNavMenu
            defaultExpanded
            renderIcon={EventsIcon}
            title={t('platform.nav.triggers', '触发器与事件')}
          >
            <SideNavMenuItem
              {...getMenuItemProps(getPath(urls.eventListeners.all()))}
            >
              {t('platform.nav.eventListeners', '事件监听')}
            </SideNavMenuItem>
            <SideNavMenuItem
              {...getMenuItemProps(getPath(urls.triggers.all()))}
            >
              {t('platform.nav.triggersItem', '触发器')}
            </SideNavMenuItem>
            <SideNavMenuItem
              {...getMenuItemProps(getPath(urls.triggerBindings.all()))}
            >
              {t('platform.nav.triggerBindings', '触发绑定')}
            </SideNavMenuItem>
            <SideNavMenuItem
              {...getMenuItemProps(urls.clusterTriggerBindings.all())}
            >
              {t('platform.nav.clusterBindings', '集群触发绑定')}
            </SideNavMenuItem>
            <SideNavMenuItem
              {...getMenuItemProps(getPath(urls.triggerTemplates.all()))}
            >
              {t('platform.nav.triggerTemplates', '触发模板')}
            </SideNavMenuItem>
            <SideNavMenuItem
              {...getMenuItemProps(getPath(urls.interceptors.all()))}
            >
              {t('platform.nav.interceptors', '拦截器')}
            </SideNavMenuItem>
          </SideNavMenu>
        )}

        {/* 5. 安全与 K8s RBAC 权限 */}
        <SideNavMenu
          defaultExpanded
          renderIcon={KeyIcon}
          title={t('platform.nav.security', '安全与访问控制')}
        >
          <SideNavMenuItem {...getMenuItemProps('/platform/credentials')}>
            {t('platform.nav.credentials', 'K8s 凭据中心 (Secrets)')}
          </SideNavMenuItem>
          {perm.isAdmin && (
            <SideNavMenuItem {...getMenuItemProps('/platform/rbac')}>
              {t('platform.nav.rbac', 'RBAC 权限管理')}
            </SideNavMenuItem>
          )}
        </SideNavMenu>

        {showKubernetesResources && (
          <SideNavMenu
            defaultExpanded
            renderIcon={KubernetesIcon}
            title={intl.formatMessage({
              id: 'dashboard.sideNav.kubernetesResources',
              defaultMessage: 'Kubernetes resources'
            })}
          >
            placeholder
          </SideNavMenu>
        )}

        {extensions.length > 0 && (
          <SideNavMenu
            defaultExpanded
            renderIcon={props => <ExtensionsIcon size={20} {...props} />}
            title={intl.formatMessage({
              id: 'dashboard.extensions.title',
              defaultMessage: 'Extensions'
            })}
          >
            {extensions.map(
              ({ apiGroup, apiVersion, displayName, name, namespaced }) => {
                const to = getPath(
                  urls.kubernetesResources.all({
                    group: apiGroup,
                    kind: name,
                    version: apiVersion
                  }),
                  namespaced
                );
                return (
                  <SideNavMenuItem
                    {...getMenuItemProps(to)}
                    key={name}
                    title={displayName}
                  >
                    {displayName}
                  </SideNavMenuItem>
                );
              }
            )}
          </SideNavMenu>
        )}

        {!isReadOnly && (
          <SideNavLink
            as={NavLink}
            renderIcon={props => <ImportResourcesIcon size={20} {...props} />}
            to={urls.importResources()}
          >
            {intl.formatMessage({
              id: 'dashboard.importResources.title',
              defaultMessage: 'Import resources'
            })}
          </SideNavLink>
        )}

        <SideNavLink
          as={NavLink}
          renderIcon={props => <AboutIcon size={20} {...props} />}
          to={urls.about()}
        >
          {intl.formatMessage({
            id: 'dashboard.about.title',
            defaultMessage: 'About Tekton'
          })}
        </SideNavLink>

        <SideNavLink
          as={NavLink}
          renderIcon={props => <SettingsIcon size={20} {...props} />}
          to={urls.settings()}
        >
          {intl.formatMessage({
            id: 'dashboard.settings.title',
            defaultMessage: 'Settings'
          })}
        </SideNavLink>
      </SideNavItems>
    </Theme>
  );
}

export default SideNav;
