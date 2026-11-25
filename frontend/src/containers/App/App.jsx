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

import { useEffect, useMemo, useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import {
  createHashRouter,
  Link,
  Navigate,
  Outlet,
  RouterProvider,
  useLocation,
  useNavigate
} from 'react-router';
import { IntlProvider, useIntl } from 'react-intl';
import { Content, HeaderContainer, InlineNotification } from '@carbon/react';
import {
  PageErrorBoundary,
  ScrollButtons
} from '@tektoncd/dashboard-components';
import {
  ALL_NAMESPACES,
  getErrorMessage,
  useWebSocketReconnected
} from '@tektoncd/dashboard-utils';

import {
  ErrorPage,
  Header,
  HeaderBarContent,
  LoadingShell,
  NotFound
} from '..';
import { PlatformLogin } from '../Platform';

import {
  NamespaceContext,
  useDefaultNamespace,
  useExtensions,
  useNamespaces,
  useProperties,
  useSelectedNamespace,
  useTenantNamespaces
} from '../../api';
import { defaultLocale, getLocale } from '../../utils';
import { getToken } from '../../api/platform';
import routes from '../../routes';

import '../../scss/App.scss';

/* istanbul ignore next */
const ConfigErrorComponent = ({ loadingConfigError }) => {
  const intl = useIntl();
  if (!loadingConfigError) {
    return null;
  }

  return (
    <InlineNotification
      kind="error"
      lowContrast
      subtitle={getErrorMessage(loadingConfigError)}
      title={intl.formatMessage({
        id: 'dashboard.app.loadingConfigError',
        defaultMessage: 'Error loading configuration'
      })}
    />
  );
};

const ConfigError = ConfigErrorComponent;

async function loadMessages(lang) {
  const loadedMessages = (await import(`../../nls/messages_${lang}.json`))
    .default;
  /* istanbul ignore next */
  if (import.meta.env.MODE === 'i18n:pseudo') {
    const startBoundary = '[[%';
    const endBoundary = '%]]';
    Object.keys(loadedMessages).forEach(messageId => {
      if (loadedMessages[messageId].startsWith(startBoundary)) {
        return;
      }
      loadedMessages[messageId] =
        `${startBoundary}${loadedMessages[messageId]}${endBoundary}`;
    });
  }

  return loadedMessages;
}

function HeaderNameLink(props) {
  return <Link {...props} to="/platform/dashboard" />;
}

function RequireAuth({ children }) {
  const location = useLocation();
  const [, setAuthTick] = useState(0);

  useEffect(() => {
    const onAuthChange = () => setAuthTick(t => t + 1);
    window.addEventListener('tkn-auth-change', onAuthChange);
    return () => window.removeEventListener('tkn-auth-change', onAuthChange);
  }, []);

  const isTest =
    (typeof process !== 'undefined' && process.env?.NODE_ENV === 'test') ||
    import.meta.env?.MODE === 'test';

  if (!isTest && !getToken()) {
    return <Navigate replace state={{ from: location }} to="/login" />;
  }

  return children;
}

function Root() {
  const intl = useIntl();
  const lang = intl.locale;
  const location = useLocation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { error: propertiesError } =
    queryClient.getQueryState(['properties']) || {};
  const { error: messagesError } =
    queryClient.getQueryState(['i18n', lang]) || {};
  const loadingConfigError = propertiesError || messagesError;
  const defaultNamespace = useDefaultNamespace();
  const { selectNamespace } = useSelectedNamespace();

  /* istanbul ignore next */
  if (location.state?.fromDefaultRoute && defaultNamespace) {
    setTimeout(() => {
      selectNamespace(defaultNamespace);
      navigate(location.pathname, { replace: true, state: {} });
    }, 0);
  }

  return (
    <RequireAuth>
      <HeaderContainer
        isSideNavExpanded
        render={({ isSideNavExpanded, onClickSideNavExpand }) => (
          <Header
            headerNameProps={{
              as: HeaderNameLink
            }}
            isSideNavExpanded={isSideNavExpanded}
            onHeaderMenuButtonClick={onClickSideNavExpand}
          >
            <HeaderBarContent />
          </Header>
        )}
      />

      <Content
        aria-labelledby="main-content-header"
        className="tkn--main-content"
        id="main-content"
      >
        <ConfigError loadingConfigError={loadingConfigError} />
        <PageErrorBoundary>
          <Outlet />
        </PageErrorBoundary>
      </Content>
      <ScrollButtons />
    </RequireAuth>
  );
}

const router = createHashRouter([
  {
    path: '/login',
    element: <PlatformLogin />,
    errorElement: <ErrorPage />
  },
  {
    path: '/',
    element: <Root />,
    errorElement: <ErrorPage />,
    children: [
      {
        errorElement: <ErrorPage />,
        children: [
          {
            index: true,
            element: (
              <Navigate
                replace
                state={{ fromDefaultRoute: true }}
                to="/platform/dashboard"
              />
            )
          },
          ...routes.platform,
          ...routes.dashboard,
          ...routes.pipelines,
          ...routes.triggers,
          {
            path: '*',
            element: <NotFound />
          }
        ]
      }
    ]
  }
]);

/* istanbul ignore next */
export function App() {
  const [lang, setLang] = useState(() => getLocale());
  const [hasToken, setHasToken] = useState(() => !!getToken());

  useEffect(() => {
    const handleLocaleChange = () => {
      setLang(getLocale());
    };
    const handleAuthChange = () => {
      setHasToken(!!getToken());
    };
    window.addEventListener('tkn-locale-change', handleLocaleChange);
    window.addEventListener('tkn-auth-change', handleAuthChange);
    return () => {
      window.removeEventListener('tkn-locale-change', handleLocaleChange);
      window.removeEventListener('tkn-auth-change', handleAuthChange);
    };
  }, []);

  const {
    isFetching: isFetchingProperties,
    isPlaceholderData: isPropertiesPlaceholder
  } = useProperties();
  const tenantNamespaces = useTenantNamespaces();

  const [selectedNamespace, setSelectedNamespace] = useState(
    tenantNamespaces[0] || ALL_NAMESPACES
  );

  const {
    data: messages,
    isFetching: isFetchingMessages,
    isPlaceholderData: isMessagesPlaceholder
  } = useQuery({
    queryKey: ['i18n', lang],
    queryFn: () => loadMessages(lang),
    placeholderData: {}
  });

  const showLoadingState = isPropertiesPlaceholder || isMessagesPlaceholder;
  const isFetchingConfig = isFetchingProperties || isFetchingMessages;

  const { isWebSocketConnected } = useExtensions(
    { namespace: tenantNamespaces[0] || ALL_NAMESPACES },
    { enabled: !isFetchingConfig && hasToken }
  );

  const queryClient = useQueryClient();

  useNamespaces({
    enabled: !isFetchingConfig && !tenantNamespaces.length && hasToken
  });
  useWebSocketReconnected(
    () => queryClient.invalidateQueries(),
    isWebSocketConnected
  );

  const namespaceContext = useMemo(
    () => ({
      selectedNamespace,
      selectNamespace: setSelectedNamespace
    }),
    [selectedNamespace]
  );

  return (
    <NamespaceContext.Provider value={namespaceContext}>
      <IntlProvider
        defaultLocale={defaultLocale}
        locale={messages ? lang : defaultLocale}
        messages={messages}
      >
        {showLoadingState && <LoadingShell />}
        {!showLoadingState && (
          <RouterProvider router={router} useTransitions={false} />
        )}
      </IntlProvider>
    </NamespaceContext.Provider>
  );
}

export default App;
