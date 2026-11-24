# Tekton DevOps Platform

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](./LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.22%2B%20(1.27.x)-00ADD8?logo=go)](./backend/go.mod)
[![Node Version](https://img.shields.io/badge/Node.js-18%2B%20(24.x)-339933?logo=node.js)](./frontend/package.json)
[![Frontend](https://img.shields.io/badge/Frontend-React%2018%20%2B%20Carbon-61DAFB?logo=react)](./frontend)

Tekton DevOps Platform 是一套现代化、高性能且高度可移植的**企业级云原生持续集成与发布（CI/CD）及多租户项目管理平台**。平台基于 **Go + Gin + SQLite/MySQL/PostgreSQL** 后端引擎与 **React 18 + Vite + IBM Carbon Design** 前端框架构建，在完整兼容原生 [Tekton Pipelines](https://github.com/tektoncd/pipeline) 与 [Tekton Triggers](https://github.com/tektoncd/triggers) 能力的同时，提供了轻量级 DSL 编排、多租户空间隔离、金融级凭据托管、实时日志流推送与全局监控大盘。

---

## ✨ 核心特性

- 🏢 **多租户项目空间隔离**：按微服务或团队划分代码工程，支持项目生命周期管理及完备的操作审计日志。
- 📝 **声明式轻量 DSL 流水线**：支持通过直观易用的阶段（Stages）与任务（Tasks）YAML/JSON 语法定义流水线，内置编译器自动无缝转换为 Kubernetes Tekton CRD 规范。
- ⚡ **双模执行引擎 (Hybrid Executor)**：
  - **Kubernetes 引擎**：依托 `client-go` 与 Dynamic Client 动态向 K8s 集群调度 Pod 与 PipelineRun；
  - **离线模拟引擎 (FakeExecutor)**：本地无需搭建 K8s 集群即可完成流水线编排、运行状态模拟与端到端调试。
- 📡 **SSE 实时流式日志看板**：采用 Server-Sent Events (SSE) 长连接技术，毫秒级流式推送构建任务输出，支持日志级别着色与分组折叠。
- 🔐 **金融级凭据与密钥托管**：对 Git Token、SSH 私钥、Kubeconfig 及环境变量采用 **AES-256-GCM** 算法加密落库，查询接口默认严格脱敏保护。
- 📊 **全局度量监控大盘**：内置构建成功率、平均构建时长、流水线状态分布及近期执行历史的多维度统计分析。
- 🔗 **多平台 Git Webhook 联动**：自动兼容 GitHub、GitLab 及通用 Git 平台的 Push 与 Merge 请求回调，实现代码提交即刻自动触发构建。
- 🔄 **Kubernetes API 反向代理与平滑兜底**：内置 K8s REST 反向代理与离线 Mock 兼容层，保证原生 Tekton 仪表盘页面在各种复杂网络或离线环境下零报错流畅运行。
- 📦 **开箱即用与极简交付**：后端采用纯 Go SQLite 驱动（无须 CGO），支持单可执行文件或单容器自包含交付运行。

---

## 🏗️ 代码库架构与目录布局

项目采用解耦的高内聚 Monorepo 组织结构：

```
tekton-dashboard/
├── frontend/               # 前端工程 (React 18, Vite, Carbon Design, Vitest)
│   ├── src/                # 平台与仪表盘业务源码
│   ├── packages/           # 共享基础组件库 (components, utils, graph)
│   ├── mock/               # 纯前端本地独立运行完整 Mock 数据集
│   └── dist/               # 前端生产打包输出目录
├── backend/                # 后端平台服务 (Go 1.22+, Gin, SQLite/MySQL/PG)
│   ├── cmd/platform/       # 平台主服务启动入口 (main.go)
│   ├── configs/            # 内嵌多环境配置 (dev.yaml, test.yaml, prod.yaml)
│   ├── data/               # 本地开发 SQLite 数据库存储目录
│   ├── db/migrations/      # Goose SQL 数据库迁移脚本
│   ├── internal/           # 核心业务包 (api, compiler, executor, repository, service)
│   └── bin/                # 二进制构建产物目录
├── deploy/                 # Dockerfile、K8s Manifests 与部署脚本
├── docs/                   # 项目官方技术与架构文档
├── Makefile                # 跨平台 GNU Make 自动化脚本
├── build.ps1               # Windows PowerShell 专属编译脚本
├── DEVELOPMENT.md          # 本地开发流程说明
└── LICENSE                 # 开源许可协议 (Apache 2.0)
```

---

## 🚀 快速启动

### 运行环境准备
- **Go**：≥ 1.22（推荐 1.24+ 或 1.27.x）
- **Node.js**：≥ 18（推荐 20.x 或 24.x）与 **npm**

---

### 方式一：使用 `Makefile` (推荐 Linux / macOS 或已安装 make 的环境)

```bash
# 1. 启动前端开发服务器 (Mock 模式，默认访问 http://localhost:8000/)
make dev-frontend

# 2. 启动后端开发服务 (默认加载 Dev 配置，监听 http://localhost:8080/)
make run-backend
```

### 方式二：使用 Windows PowerShell `build.ps1` 脚本

```powershell
# 一键编译前端静态资源并生成 Windows 版后端二进制
.\build.ps1 -Frontend -Target windows

# 运行服务
.\backend\bin\platform.exe
```

### 方式三：使用 `npm` 与 `go` 原生编译 (无须任何外部脚本)
详细步骤请参阅：👉 **[原生编译与运行指南 (docs/dev/native-build.md)](./docs/dev/native-build.md)**

```bash
# 前端：
cd frontend
npm ci
npm run dev

# 后端：
cd backend
go run -tags dev ./cmd/platform -env dev
```

---

## 🔑 默认登录认证信息

当后端服务首次启动时，系统会自动在数据库中初始化默认超级管理员账号：
- **管理员账号**：`admin`
- **默认初始密码**：`admin123`
- **角色**：`admin`

进入平台后可使用该账号管理项目、创建流水线、分配凭据及查看全局审计与监控大盘。

---

## 📚 完整文档索引导航

| 文档分类 | 推荐文档 | 说明 |
|:---|:---|:---|
| **API 文档** | 📘 **[后端接口规范文档](./docs/dev/api.md)** | 全量 RESTful API、统一错误码、SSE 实时流与请求响应格式 |
| **原生编译** | 🛠️ **[npm & go 原生编译指南](./docs/dev/native-build.md)** | 脱离 Makefile 的前后端原生构建、交叉编译与静态托管步骤 |
| **开发指南** | 💻 **[开发者快速入门](./DEVELOPMENT.md)** | 本地环境搭建、前后端联调及多环境配置切换说明 |
| **架构规范** | 🏛️ **[平台开发规范与架构设计](./docs/platform/DEVELOPMENT.md)** | 分层依赖模型、数据库自动迁移规范及测试策略 |
| **环境配置** | ⚙️ **[开发环境与构建指南](./docs/dev/README.md)** | 单测执行 (Vitest/Go test)、代码检查 (ESLint) 及 i18n 多语言 |
| **安装部署** | 🚀 **[平台部署与安装手册](./docs/install.md)** | 二进制单体运行、Docker 单容器打包与 Kubernetes 集群部署 |
| **贡献代码** | 👥 **[贡献者规范](./CONTRIBUTING.md)** | 分支管理、代码风格规范与 Pull Request 流程 |

---

## 📄 开源许可

本项目依据 [Apache License 2.0](./LICENSE) 协议开源。文档内容依据 [Creative Commons Attribution 4.0 License](https://creativecommons.org/licenses/by/4.0/) 授权。
