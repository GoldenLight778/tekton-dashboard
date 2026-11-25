/*
Copyright 2020-2026 The Tekton Authors
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
/* istanbul ignore file */

import { Fragment, useState } from 'react';
import { useIntl } from 'react-intl';
import {
  InlineNotification,
  SkeletonText,
  Tile,
  usePrefix
} from '@carbon/react';
import {
  ArrowRight as ArrowIcon,
  Checkmark as CheckIcon,
  Copy as CopyIcon
} from '@carbon/react/icons';
import { getErrorMessage, useTitleSync } from '@tektoncd/dashboard-utils';
import { Icons } from '@tektoncd/dashboard-components';

import { useProperties } from '../../api';

const {
  Assets: DocsPictogram,
  TektonDashboardColor: TektonDashboardLogo,
  UserInterface: HubPictogram
} = Icons;

const STANDARD_TEKTON_PIPELINE_YAML = `apiVersion: tekton.dev/v1
kind: Pipeline
metadata:
  name: fullstack-ci-pipeline
  namespace: default
spec:
  description: 标准云原生 Tekton 多阶段 CI/CD 交付流水线
  workspaces:
    - name: shared-workspace
      description: 各 Task 阶段共享源码与构建产物的工作空间
  tasks:
    - name: fetch-source
      taskSpec:
        steps:
          - name: git-clone
            image: alpine/git:latest
            script: |
              #!/bin/sh
              set -e
              echo "[Stage 1] Checking out repository..."
              git --version
    - name: unit-test
      runAfter:
        - fetch-source
      taskSpec:
        steps:
          - name: go-test
            image: golang:1.22-alpine
            script: |
              #!/bin/sh
              set -e
              echo "[Stage 2] Running unit tests & static analysis..."
              go version
    - name: build-release
      runAfter:
        - unit-test
      taskSpec:
        steps:
          - name: package-artifact
            image: alpine:latest
            script: |
              #!/bin/sh
              set -e
              echo "[Stage 3] Packaging production container artifact... Done!"`;

const DOC_RESOURCES = [
  {
    href: 'https://tekton.dev/docs/concepts/overview/',
    tag: 'CORE ARCHITECTURE',
    title: 'Tekton 核心架构概览 (Overview)',
    desc: '深入理解 Tekton 云原生 CI/CD 设计哲学、核心组件协作拓扑与最佳实践',
    icon: DocsPictogram
  },
  {
    href: 'https://tekton.dev/docs/concepts/concept-model/',
    tag: 'CONCEPT MODEL',
    title: '标准概念与数据模型 (Concept Model)',
    desc: '掌握 Step、Task、Pipeline、TaskRun、PipelineRun 与 Workspace 的声明式关系',
    icon: DocsPictogram
  },
  {
    href: 'https://tekton.dev/docs/pipelines/',
    tag: 'PIPELINES SPEC',
    title: '流水线与任务编排参考 (Pipelines API)',
    desc: '查阅 tekton.dev/v1 完整字段规范：runAfter 依赖、Params、Results、Matrix 与 Finally',
    icon: DocsPictogram
  },
  {
    href: 'https://artifacthub.io/packages/search?kind=23&kind=7&kind=11',
    tag: 'ECOSYSTEM HUB',
    title: 'Artifact Hub 官方任务市场',
    desc: '检索并导入社区维护的标准化 StepActions、Tasks（如 git-clone、kaniko）与 Pipelines',
    icon: HubPictogram
  }
];

export function About() {
  const intl = useIntl();
  const carbonPrefix = usePrefix();
  const [copied, setCopied] = useState(false);

  useTitleSync({
    page: intl.formatMessage({
      id: 'dashboard.about.title',
      defaultMessage: 'About Tekton'
    })
  });

  const getDisplayValue = value => {
    switch (value) {
      case true:
        return intl.formatMessage({
          id: 'dashboard.about.true',
          defaultMessage: 'True'
        });
      default:
        return value;
    }
  };

  const { data, isPlaceholderData } = useProperties();
  const dashboardNamespace =
    data?.dashboardNamespace || data?.DashboardNamespace;
  const dashboardVersion = data?.dashboardVersion || data?.DashboardVersion;
  const isReadOnly = data?.isReadOnly ?? data?.ReadOnly;
  const logoutURL = data?.logoutURL || data?.LogoutURL;
  const pipelinesNamespace =
    data?.pipelinesNamespace || data?.PipelineNamespace;
  const pipelinesVersion = data?.pipelinesVersion || data?.PipelineVersion;
  const triggersNamespace = data?.triggersNamespace || data?.TriggersNamespace;
  const triggersVersion = data?.triggersVersion || data?.TriggersVersion;

  const isTriggersInstalled = !!(triggersNamespace && triggersVersion);

  const checkMissingProperties = () => {
    if (isPlaceholderData) {
      return null;
    }

    const propertiesToCheck = {
      dashboardNamespace,
      dashboardVersion,
      pipelinesNamespace,
      pipelinesVersion
    };

    const errorsFound = Object.keys(propertiesToCheck)
      .map(key => (propertiesToCheck[key] ? null : key))
      .filter(Boolean);

    return errorsFound.length
      ? intl.formatMessage(
          {
            id: 'dashboard.about.missingProperties',
            defaultMessage: 'Could not find: {errorsFound}'
          },
          { errorsFound: errorsFound.join(', ') }
        )
      : null;
  };

  const getField = (property, value) => {
    const displayValue = getDisplayValue(value);
    return (
      displayValue && (
        <Fragment key={property}>
          <dt className={`${carbonPrefix}--label`}>{property}</dt>
          <dd>{displayValue}</dd>
        </Fragment>
      )
    );
  };

  const handleCopyYaml = () => {
    if (navigator?.clipboard?.writeText) {
      navigator.clipboard.writeText(STANDARD_TEKTON_PIPELINE_YAML);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  const error = checkMissingProperties();

  const versionLabel = intl.formatMessage({
    id: 'dashboard.about.version',
    defaultMessage: 'Version'
  });
  const isReadOnlyLabel = intl.formatMessage({
    id: 'dashboard.about.isReadOnly',
    defaultMessage: 'ReadOnly'
  });
  const logoutURLLabel = intl.formatMessage({
    id: 'dashboard.about.logoutURL',
    defaultMessage: 'LogoutURL'
  });

  return (
    <div className="tkn--about">
      {/* 1. Modern Hero Card Header */}
      <div className="tkn--about-hero">
        <div className="about-hero-text">
          <div className="about-hero-badges">
            <span className="hero-pill pill-primary">
              CLOUD NATIVE CI/CD ENGINE
            </span>
            <span className="hero-pill pill-rbac">
              K8s Native RBAC & Secret Integrated
            </span>
          </div>
          <h1 id="main-content-header" tabIndex={-1}>
            {intl.formatMessage({
              id: 'dashboard.about.title',
              defaultMessage: '关于 Tekton 云原生交付平台'
            })}
          </h1>
          <p>
            {intl.formatMessage({
              id: 'dashboard.about.description',
              defaultMessage:
                'Tekton 是基于 Kubernetes 原生 CRD 的标准化开源 CI/CD 框架。本控制台 100% 遵循 Tekton 原生资源模型，提供多命名空间 RBAC 鉴权、Secret 凭据自动挂载、可视化 DAG 流水线编排与容器实时日志观测。'
            })}
          </p>
        </div>
        <div className="about-hero-logo-box">
          <TektonDashboardLogo className="tkn--logo">
            <title>
              {intl.formatMessage({
                id: 'dashboard.logo.tooltip',
                defaultMessage: 'Meow'
              })}
            </title>
          </TektonDashboardLogo>
        </div>
      </div>

      {error && (
        <InlineNotification
          kind="error"
          lowContrast
          style={{ marginBottom: '1.25rem', maxWidth: '100%' }}
          subtitle={getErrorMessage(error)}
          title={intl.formatMessage({
            id: 'dashboard.about.error',
            defaultMessage: 'Error getting data'
          })}
        />
      )}

      {/* 2. Modern Environment & Engine Components Matrix */}
      <section className="tkn--about-section">
        <header className="about-section-header">
          <div>
            <h2 className="tkn--section-title">
              {intl.formatMessage({
                id: 'dashboard.about.environmentDetails',
                defaultMessage: '集群环境与核心引擎状态 (Environment Details)'
              })}
            </h2>
            <p className="about-section-subtitle">
              实时探测当前 Kubernetes 集群内 Tekton 核心控制器组件版本与运行命名空间
            </p>
          </div>
        </header>

        <div className="about-env-cards-grid">
          <Tile
            className="about-env-card card-indigo"
            id="tkn--about--dashboard-tile"
          >
            <div className="env-card-top">
              <div>
                <span className="env-component-tag">CONTROL PLANE UI</span>
                <h3>Dashboard</h3>
              </div>
              <span className="env-status-pill">● 运行中</span>
            </div>
            {isPlaceholderData ? (
              <SkeletonText paragraph />
            ) : (
              <dl className="env-meta-list">
                {[
                  getField(isReadOnlyLabel, isReadOnly),
                  getField(logoutURLLabel, logoutURL),
                  getField('Namespace', dashboardNamespace),
                  getField(versionLabel, dashboardVersion)
                ].filter(Boolean)}
              </dl>
            )}
          </Tile>

          <Tile
            className="about-env-card card-violet"
            id="tkn--about--pipelines-tile"
          >
            <div className="env-card-top">
              <div>
                <span className="env-component-tag">EXECUTION ENGINE</span>
                <h3>Pipelines</h3>
              </div>
              <span className="env-status-pill">● 核心就绪</span>
            </div>
            {isPlaceholderData ? (
              <SkeletonText paragraph />
            ) : (
              <dl className="env-meta-list">
                {[
                  getField('Namespace', pipelinesNamespace),
                  getField(versionLabel, pipelinesVersion)
                ].filter(Boolean)}
              </dl>
            )}
          </Tile>

          {isTriggersInstalled && (
            <Tile
              className="about-env-card card-emerald"
              id="tkn--about--triggers-tile"
            >
              <div className="env-card-top">
                <div>
                  <span className="env-component-tag">WEBHOOK & EVENTS</span>
                  <h3>Triggers</h3>
                </div>
                <span className="env-status-pill">● 已安装</span>
              </div>
              {isPlaceholderData ? (
                <SkeletonText paragraph />
              ) : (
                <dl className="env-meta-list">
                  {[
                    getField('Namespace', triggersNamespace),
                    getField(versionLabel, triggersVersion)
                  ].filter(Boolean)}
                </dl>
              )}
            </Tile>
          )}
        </div>
      </section>

      {/* 3. Pure Tekton Native Pipeline Mind Map & 4-Step Tutorial */}
      <section className="tkn--about-section tkn--about-mindmap">
        <header className="about-section-header">
          <div>
            <h2 className="tkn--section-title">
              构建完整可执行 Tekton Pipeline 思维导图与实战教程
            </h2>
            <p className="about-section-subtitle">
              100% 遵循 Kubernetes 与 <code>tekton.dev/v1</code> 原生设计理念，从命名空间隔离、Secret 自动挂载到 DAG 流水线触发的四大核心阶段：
            </p>
          </div>
        </header>

        <div className="about-mindmap-board">
          <div className="mindmap-root-col">
            <div className="mindmap-root-node">
              <span className="node-badge">TEKTON.DEV/V1</span>
              <h3>原生可执行 Pipeline</h3>
              <p>Namespace 隔离 + Secret 自动注入 + 多 Task DAG 编排闭环</p>
            </div>
          </div>

          <div aria-hidden="true" className="mindmap-trunk-connector" />

          <div className="mindmap-branches-col">
            <div className="mindmap-branch branch-indigo">
              <div className="branch-main-node">
                <span className="branch-step">阶段 1</span>
                <h4>命名空间与 K8s Secret</h4>
              </div>
              <div className="branch-children">
                <div className="leaf-node">
                  <strong>选择或新建 K8s Namespace</strong>
                  <span>在顶栏或「命名空间」页按团队/环境隔离 Tekton 资源与 RBAC 权限</span>
                </div>
                <div className="leaf-node">
                  <strong>托管 Secret 并自动挂载 SA</strong>
                  <span>配置 Git/Harbor 密钥，自动注入 tekton.dev/git-0 注解并挂载至 ServiceAccount</span>
                </div>
              </div>
            </div>

            <div className="mindmap-branch branch-sky">
              <div className="branch-main-node">
                <span className="branch-step">阶段 2</span>
                <h4>原子任务定义 (Task)</h4>
              </div>
              <div className="branch-children">
                <div className="leaf-node">
                  <strong>在「任务库」新建 Tekton Task</strong>
                  <span>一键选用 Git Clone、Go 编译、Node 打包等模板或自定义可复用 Task</span>
                </div>
                <div className="leaf-node">
                  <strong>声明容器步骤 (Steps & Image)</strong>
                  <span>定义每个 Step 的基础容器镜像、脚本与输入输出参数</span>
                </div>
              </div>
            </div>

            <div className="mindmap-branch branch-violet">
              <div className="branch-main-node">
                <span className="branch-step">阶段 3</span>
                <h4>流水线编排 (Pipeline)</h4>
              </div>
              <div className="branch-children">
                <div className="leaf-node">
                  <strong>可视化组装多阶段 DAG 拓扑</strong>
                  <span>在「Tekton 流水线」添加多个任务节点，引用任务库 Task 或内联脚本</span>
                </div>
                <div className="leaf-node">
                  <strong>配置 runAfter 与共享 Workspace</strong>
                  <span>通过 runAfter 控制串并行顺序，通过 Workspace 在任务间共享源码产物</span>
                </div>
              </div>
            </div>

            <div className="mindmap-branch branch-emerald">
              <div className="branch-main-node">
                <span className="branch-step">阶段 4</span>
                <h4>触发 PipelineRun 与观测</h4>
              </div>
              <div className="branch-children">
                <div className="leaf-node">
                  <strong>一键触发 PipelineRun 实例</strong>
                  <span>在流水线列表点击运行图标填写参数与运行 ServiceAccount 即刻启动</span>
                </div>
                <div className="leaf-node">
                  <strong>流式观测容器实时日志</strong>
                  <span>在「流水线执行」或「平台大盘」实时追踪各容器步骤输出日志与耗时</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div className="about-tutorial-grid">
          <div className="about-tutorial-steps">
            <h3>快速上手：4 步在当前命名空间跑通第一条 Tekton 流水线</h3>
            <ol className="tutorial-step-list">
              <li>
                <span className="step-num">1</span>
                <div className="step-body">
                  <strong>
                    第一步：确认顶栏「Namespace」并配置「K8s 凭据中心 (Secrets)」
                  </strong>
                  <p>
                    在顶栏选择目标命名空间（如 <code>default</code>），如需拉取私有仓库，可在「安全与访问控制 → K8s 凭据中心」新建凭据并勾选自动挂载至{' '}
                    <code>default</code> ServiceAccount。
                  </p>
                </div>
              </li>
              <li>
                <span className="step-num">2</span>
                <div className="step-body">
                  <strong>
                    第二步：进入「任务管理 → 任务库」创建可复用 Tekton Task
                  </strong>
                  <p>
                    点击「新建基础任务」，选择预设模板（如 Git 检出、Go 测试构建、Node 打包）或自定义容器镜像与 Shell 脚本，保存为标准{' '}
                    <code>tekton.dev/v1 Task</code>。
                  </p>
                </div>
              </li>
              <li>
                <span className="step-num">3</span>
                <div className="step-body">
                  <strong>
                    第三步：进入「流水线中心 → Tekton 流水线」可视化组装或导入 YAML
                  </strong>
                  <p>
                    点击「新建 Tekton 流水线」，使用「可视化任务编排」添加多个阶段并设置{' '}
                    <code>runAfter</code> 前置依赖，或直接切换至「原生 YAML 规格」粘贴右侧标准示例。
                  </p>
                </div>
              </li>
              <li>
                <span className="step-num">4</span>
                <div className="step-body">
                  <strong>
                    第四步：点击列表「一键运行」触发 PipelineRun 并查看实时日志
                  </strong>
                  <p>
                    在流水线行右侧点击三角形运行按钮，确认运行{' '}
                    <code>ServiceAccount</code> 后立即生成{' '}
                    <code>PipelineRun</code>，随后在「流水线执行」或「平台大盘」实时查看容器终端日志。
                  </p>
                </div>
              </li>
            </ol>
          </div>

          <div className="about-tutorial-yaml">
            <div className="yaml-card-header">
              <span>标准可执行 Tekton Pipeline 示例 (tekton.dev/v1)</span>
              <button
                className="yaml-copy-btn"
                onClick={handleCopyYaml}
                type="button"
              >
                {copied ? <CheckIcon size={14} /> : <CopyIcon size={14} />}
                <span>{copied ? '已复制' : '一键复制 YAML'}</span>
              </button>
            </div>
            <pre className="yaml-code-block">
              <code>{STANDARD_TEKTON_PIPELINE_YAML}</code>
            </pre>
          </div>
        </div>
      </section>

      {/* 4. Modern Documentation & Ecosystem Resource Cards */}
      <section className="tkn--about-section tkn--about-docs">
        <header className="about-section-header">
          <div>
            <h2 className="tkn--section-title">
              {intl.formatMessage({
                id: 'dashboard.about.documentation',
                defaultMessage: '官方文档与云原生生态资源 (Documentation & Resources)'
              })}
            </h2>
            <p className="about-section-subtitle">
              快速查阅 Tekton 官方技术规范与社区可复用 Task / Pipeline 模板库
            </p>
          </div>
        </header>

        <div className="about-docs-cards-grid">
          {DOC_RESOURCES.map(item => {
            const IconComp = item.icon;
            return (
              <a
                className="about-doc-card"
                href={item.href}
                key={item.href}
                rel="noopener noreferrer"
                target="_blank"
              >
                <div className="doc-card-top">
                  <div className="doc-icon-badge">
                    <IconComp />
                  </div>
                  <span className="doc-tag">{item.tag}</span>
                </div>
                <div className="doc-card-body">
                  <h3>{item.title}</h3>
                  <p>{item.desc}</p>
                </div>
                <div className="doc-card-footer">
                  <span>访问官方文档</span>
                  <ArrowIcon className="doc-arrow" size={18} />
                </div>
              </a>
            );
          })}
        </div>
      </section>
    </div>
  );
}

export default About;
