# DevOps 平台开发与构建环境指南 (Dev Docs)

本文档面向 DevOps 平台的日常开发人员，提供本地构建、联调、测试、代码规范及国际化（i18n）的详细操作指南。

---

## 目录
- [1. 开发环境要求](#1-开发环境要求)
- [2. 代码库工程结构](#2-代码库工程结构)
- [3. 本地构建与快速启动](#3-本地构建与快速启动)
  - [3.1 前端构建与运行](#31-前端构建与运行)
  - [3.2 后端构建与运行](#32-后端构建与运行)
- [4. 测试执行规范](#4-测试执行规范)
  - [4.1 后端单元测试](#41-后端单元测试)
  - [4.2 前端单元测试 (Vitest)](#42-前端单元测试-vitest)
  - [4.3 E2E 端到端测试 (Cypress)](#43-e2e-端到端测试-cypress)
- [5. 代码风格与规范检查 (Linter)](#5-代码风格与规范检查-linter)
- [6. 国际化多语言支持 (i18n)](#6-国际化多语言支持-i18n)
- [7. 组件库独立预览 (Storybook)](#7-组件库独立预览-storybook)
- [8. 相关文档索引](#8-相关文档索引)

---

## 1. 开发环境要求

开发与构建 DevOps 平台需要以下基础工具：

1. **Go 语言环境**：Go ≥ 1.22（当前开发与验证推荐 Go 1.24+ 或 1.27.x），请确认 `go version` 可用。
2. **Node.js & npm**：Node.js ≥ 18（推荐 20.x 或 24.x），npm ≥ 9。请确认 `node -v` 与 `npm -v` 可用。
3. **Git**：用于版本管理。
4. **可选工具**：
   - `make`：用于执行根目录及模块 Makefile 脚本。
   - `PowerShell`：Windows 下可直接使用 `build.ps1` 脚本。
   - 若不使用任何脚本包装，可直接阅读 [原生编译指南 (docs/dev/native-build.md)](./native-build.md)。

---

## 2. 代码库工程结构

项目采用标准 Monorepo 组织结构：
- **`frontend/`**：包含 React 18 前端主应用、Carbon UI 库、TanStack Query 以及共享组件包。构建产物输出至 `frontend/dist`。
- **`backend/`**：包含基于 Gin 框架构建的平台核心服务、内置轻量级 DSL 编译器、K8s / FakeExecutor 执行引擎以及多方言数据库访问层。
- **`Makefile`** & **`build.ps1`**：提供一键式编译与运行任务编排。
- **`docs/`**：系统全量技术文档与 API 规范。

---

## 3. 本地构建与快速启动

### 3.1 前端构建与运行

所有前端依赖与脚本均位于 `frontend/` 目录中：

```bash
cd frontend

# 安装依赖
npm ci

# 启动纯前端 Mock 开发模式 (端口 8000，无须启动后端)
npm run dev

# 启动联调模式 (Vite 将 /api/platform 请求代理至本地 8080 端口)
npm run prod

# 生产环境打包构建 (产物输出至 frontend/dist)
npm run build
```

### 3.2 后端构建与运行

后端服务位于 `backend/` 目录中：

```bash
cd backend

# 安装并整理依赖
go mod download
go mod tidy

# 启动开发环境服务 (默认加载 configs/dev.yaml，端口 8080)
go run -tags dev ./cmd/platform -env dev

# 编译 Windows 平台二进制文件
set CGO_ENABLED=0&& set GOOS=windows&& set GOARCH=amd64&& go build -tags dev -ldflags "-s -w" -o bin/platform.exe ./cmd/platform

# 交叉编译 Linux 静态无依赖二进制文件
set CGO_ENABLED=0&& set GOOS=linux&& set GOARCH=amd64&& go build -tags prod -ldflags "-s -w" -o bin/platform-linux-amd64 ./cmd/platform
```

> **静态资源自动托管说明**：当执行 `npm run build` 生成 `frontend/dist` 后，启动后端二进制程序会自动检测并托管该目录下的静态网页与资源文件，访问 `http://localhost:8080/` 即可直接使用完整的前后端一体化平台。

---

## 4. 测试执行规范

### 4.1 后端单元测试

后端单元测试覆盖了配置加载、K8s 客户端绑定、多方言数据库仓储以及 M1-M5 核心功能流水线与 API 校验：

```bash
# 根目录执行：
make test-backend

# 或在 backend/ 目录下执行：
cd backend
go test -v ./...
```

### 4.2 前端单元测试 (Vitest)

前端单元测试基于 [Vitest](https://vitest.dev/) 和 React Testing Library 构建：

```bash
# 交互式监听模式：
cd frontend
npm test

# 单次运行所有测试 (适用于 CI 流水线)：
cd frontend
npx vitest run
```

### 4.3 E2E 端到端测试 (Cypress)

针对前端界面的全流程回归测试：

```bash
cd frontend

# 首次运行前安装 Cypress 依赖：
npm run e2e:install

# 启动交互式 Cypress 测试控制台：
npm run e2e
```

---

## 5. 代码风格与规范检查 (Linter)

在提交代码前，请确保前端和后端代码均通过质量检查：

```bash
# 前端 ESLint & Prettier 格式校验
cd frontend
npm run lint

# 自动修复前端格式问题
npm run lint:fix

# 后端代码格式化与规范检查
cd backend
go vet ./...
```

---

## 6. 国际化多语言支持 (i18n)

项目使用 `@formatjs/intl` (`react-intl`) 负责前端界面的国际化多语言渲染：

- **语言包目录**：`frontend/src/nls/`
- **默认语言**：英文 (`messages_en.json`) 与简体中文 (`messages_zh-Hans.json`)。
- **提取代码中的新词条**：
  ```bash
  cd frontend
  npm run i18n:extract
  ```
  > **注意**：`messages_en.json` 为脚本自动扫描生成，请勿手动编辑，应直接修改源码组件中 `defaultMessage` 的文本内容并重新运行提取脚本。
- **启动多语言调试模式**：
  ```bash
  cd frontend
  npm run i18n:start
  ```

---

## 7. 组件库独立预览 (Storybook)

共享 UI 组件（如流水线状态图标、图表看板、步骤抽屉等）支持通过 Storybook 独立运行与预览：

```bash
cd frontend

# 启动 Storybook 独立开发服务器 (默认端口 5000)
npm run storybook

# 构建静态 Storybook 文档包
npm run storybook:build
```

---

## 8. 相关文档索引

- 📘 **[后端接口规范文档 (api.md)](./api.md)**：包含全部 REST API 详细参数、JSON 样例与 SSE 日志流定义。
- 🛠️ **[原生编译与运行指南 (native-build.md)](./native-build.md)**：脱离 Makefile 的纯原生 `npm` 与 `go` 交叉构建全指南。
- 🏛️ **[平台开发规范与架构设计 (../platform/DEVELOPMENT.md)](../platform/DEVELOPMENT.md)**：后端依赖注入分层模型、数据库迁移原则。
- 🚀 **[平台部署与安装手册 (../install.md)](../install.md)**：单体安装、容器打包及 K8s 部署。
