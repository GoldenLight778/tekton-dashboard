/*
Copyright 2019-2024 The Tekton Authors
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

import { deleteRequest, post } from './comms';
import {
  getKubeAPI,
  getTektonPipelinesAPIVersion,
  tektonAPIGroup,
  useCollection,
  useResource
} from './utils';

export function createTask({
  name,
  namespace = 'default',
  description = '',
  stepName = 'execute',
  image = 'alpine:latest',
  script = 'echo "Hello from Tekton Task"',
  rawResource
}) {
  const resource = rawResource || {
    apiVersion: `tekton.dev/${getTektonPipelinesAPIVersion()}`,
    kind: 'Task',
    metadata: {
      name,
      namespace: namespace || 'default'
    },
    spec: {
      ...(description ? { description } : {}),
      steps: [
        {
          name: stepName || 'execute',
          image: image || 'alpine:latest',
          script: script || 'echo "Task executed"'
        }
      ]
    }
  };
  const targetNamespace =
    resource?.metadata?.namespace || namespace || 'default';
  const uri = getKubeAPI({
    group: tektonAPIGroup,
    kind: 'tasks',
    params: { namespace: targetNamespace },
    version: getTektonPipelinesAPIVersion()
  });
  return post(uri, resource).then(({ body }) => body);
}

export function deleteTask({ name, namespace }) {
  const uri = getKubeAPI({
    group: tektonAPIGroup,
    kind: 'tasks',
    params: { name, namespace },
    version: getTektonPipelinesAPIVersion()
  });
  return deleteRequest(uri);
}

export function useTasks(params) {
  return useCollection({
    group: tektonAPIGroup,
    kind: 'tasks',
    params,
    version: getTektonPipelinesAPIVersion()
  });
}

export function useTask(params, queryConfig) {
  return useResource({
    group: tektonAPIGroup,
    kind: 'tasks',
    params,
    queryConfig,
    version: getTektonPipelinesAPIVersion()
  });
}
