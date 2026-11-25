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

import { Navigate } from 'react-router';
import {
  PlatformCredentials,
  PlatformDashboard,
  PlatformLogin,
  PlatformNamespaces,
  PlatformRBAC
} from '../containers/Platform';

export default [
  {
    path: '/login',
    element: <PlatformLogin />
  },
  {
    path: '/platform/dashboard',
    element: <PlatformDashboard />
  },
  {
    path: '/platform/namespaces',
    element: <PlatformNamespaces />
  },
  {
    path: '/platform/credentials',
    element: <PlatformCredentials />
  },
  {
    path: '/platform/rbac',
    element: <PlatformRBAC />
  },
  // Redirect legacy dual-track routes to unified Tekton/K8s native views
  {
    path: '/platform/projects',
    element: <Navigate replace to="/platform/namespaces" />
  },
  {
    path: '/platform/pipelines',
    element: <Navigate replace to="/pipelines" />
  }
];
