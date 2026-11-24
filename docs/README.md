# DevOps 平台文档中心 (Documentation Center)

欢迎查阅 DevOps 平台文档库。本目录汇总了平台架构设计、开发联调、原生编译、后端 API 规范、部署运维以及原生 Tekton 功能的使用指南。

---

## 📌 核心技术与开发文档

- 📘 **[后端接口规范文档 (dev/api.md)](./dev/api.md)**
  详细阐述系统全部 RESTful API 规范、统一错误码、JWT 认证鉴权、SSE 实时流式日志接口、项目管理、流水线编排、凭据托管及 Kubernetes 反向代理。

- 🛠️ **[npm & go 原生编译指南 (dev/native-build.md)](./dev/native-build.md)**
  **专为无 Makefile 环境定制**。手把手指导如何在纯 Windows / Linux / macOS 环境下使用原生 `npm` 打包前端，使用原生 `go` 编译无 CGO 依赖的静态二进制文件及前后端联合打包流程。

- 💻 **[开发者快速入门指南 (../DEVELOPMENT.md)](../DEVELOPMENT.md)**
  快速启动本地开发、前后端联动、Makefile 与 PowerShell 构建指令速查。

- 🏛️ **[平台开发规范与架构设计 (platform/DEVELOPMENT.md)](./platform/DEVELOPMENT.md)**
  分层依赖原则（API → Service → Repository）、Goose 数据库版本迁移规范、单容器部署原则及 Definition of Done。

- ⚙️ **[开发环境与构建指南 (dev/README.md)](./dev/README.md)**
  单测运行（Vitest / Go test）、代码风格格式化检查（ESLint / Prettier）、国际化多语言支持（i18n）以及 Storybook 组件库预览。

---

## 🚀 部署与运维指南

- 📦 **[平台部署与安装手册 (install.md)](./install.md)**
  涵盖本地单可执行程序独立部署、Docker 单容器打包部署以及 Kubernetes 集群生产级安装步骤。

- 🖥️ **[日志查看与流式分析 (logs.md)](./logs.md)**
  详细说明任务运行日志的高级查看能力：ANSI 终端着色渲染、日志级别解析（error/warn/info/debug）、分组折叠与自动链接识别。

- 🧩 **[扩展机制指南 (extensions.md)](./extensions.md)**
  如何通过 Extension CRD 向仪表盘注入第三方 Kubernetes 自定义资源视图。

- 🎓 **[入门演练教程 (tutorial.md)](./tutorial.md)**
  基于本地集群快速体验流水线编排与运行的端到端演练。

- 🚶‍♂️ **[专题操作指导 (walkthrough/README.md)](./walkthrough/README.md)**
  - [基于 Kind 本地集群搭建环境 (walkthrough/walkthrough-kind.md)](./walkthrough/walkthrough-kind.md)
  - [持久化日志配置与外部存储接入 (walkthrough/walkthrough-logs.md)](./walkthrough/walkthrough-logs.md)
  - [集成 OAuth2-Proxy 进行统一身份鉴权 (walkthrough/walkthrough-oauth2-proxy.md)](./walkthrough/walkthrough-oauth2-proxy.md)

---

## 🧭 项目工程架构一览

```
tekton-dashboard/
├── frontend/               # 前端代码库 (React 18, Vite, Carbon Design)
├── backend/                # 后端代码库 (Go 1.22+, Gin, SQLite/MySQL/PG)
├── deploy/                 # 容器构建与集群清单
├── docs/                   # 本文档目录
│   ├── dev/                # 开发者指南、API 规范与原生编译文档
│   ├── platform/           # 平台架构与工程约定
│   └── walkthrough/        # 专题实战教程
├── Makefile                # 统一 GNU Make 编译任务
├── build.ps1               # Windows PowerShell 编译任务
└── README.md               # 仓库主说明文件
```
