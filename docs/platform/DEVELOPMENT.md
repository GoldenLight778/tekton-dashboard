# DevOps 平台开发与流程规范文档

> 本文档描述 DevOps 平台（Go + Gin + SQLite/MySQL/Postgres 后端，React 18 + Carbon Design 前端，支持单二进制运行、单容器及 Kubernetes 混合部署）的工程架构、开发约定与日常发布流程。

---

## 1. 核心技术选型

| 分层 | 技术选型 | 说明 |
|:---|:---|:---|
| **后端框架** | **Gin (Go 1.22+, 验证 1.27.x)** | 高性能 REST API、中间件流水线、SSE 实时日志流推送 |
| **数据库** | **SQLite / MySQL / PostgreSQL** | 默认采用纯 Go SQLite (`modernc.org/sqlite`，无须 CGO)；通过方言驱动隔离支持切换外部 MySQL/Postgres |
| **数据访问** | **`database/sql` + Repository 模式** | 数据仓储接口隔离，业务 Service 零数据库方言感知，便于单测 Fake 注入 |
| **数据库迁移** | **Goose SQL 迁移** | 迁移脚本位于 `backend/db/migrations/`，系统启动时自动检测版本并平滑升级 |
| **认证与加密** | **JWT (golang-jwt/jwt/v5) + bcrypt + AES-256-GCM** | 内置管理员与多角色 RBAC；凭据敏感信息高强度加密存储并脱敏展示 |
| **Kubernetes 适配** | **client-go + Dynamic Client + FakeExecutor** | 真实集群环境下动态提交/监听 Tekton CRD；本地无集群时自动降级为 Fake 模式 |
| **结构化日志** | **Zap (`go.uber.org/zap`)** | 开发环境彩色控制台输出，生产环境 JSON 结构化流式输出 |
| **前端体系** | **React 18 + Carbon Design + Vite + Vitest** | 支持 `npm run dev` 纯 Mock 独立开发与生产级代理联调 |

---

## 2. 统一代码仓库目录结构

代码仓库采用模块清晰的 Monorepo 单仓结构：

```
tekton-dashboard/
├── backend/                        # 后端 Go 平台服务根目录
│   ├── cmd/
│   │   ├── platform/               # 平台主服务入口 main.go
│   │   └── dashboard/              # 兼容与历史独立入口
│   ├── configs/                    # 内嵌多环境配置文件 (embed.FS)
│   │   ├── dev.yaml                # 开发环境默认配置
│   │   ├── test.yaml               # 测试环境配置
│   │   └── prod.yaml               # 生产环境配置
│   ├── data/                       # 本地开发 SQLite 数据文件存放目录 (git ignore)
│   ├── db/
│   │   └── migrations/             # Goose SQL 数据库版本迁移脚本
│   ├── dist/                       # 后端独立打包时的静态资源候选目录
│   ├── internal/                   # 后端内部包 (禁止外部仓库直接 import)
│   │   ├── api/                    # HTTP Handler 接口层：请求参数校验、标准化响应
│   │   ├── compiler/               # 流水线 DSL 编译器：DSL → Tekton CRD 转换
│   │   ├── config/                 # 统一配置加载层：Env / Flag / Yaml 优先级合并
│   │   ├── crypto/                 # JWT 令牌签发、密码散列与 AES 凭据加解密
│   │   ├── executor/               # 流水线执行引擎：K8sExecutor 与 FakeExecutor
│   │   ├── k8s/                    # Kubernetes 客户端 Bundle 构建与代理初始化
│   │   ├── repository/             # 仓储接口与 SQLite/MySQL/PG 跨方言实现
│   │   ├── server/                 # Gin 路由注册、中间件装配、静态资源与 SPA 托管
│   │   ├── service/                # 核心业务逻辑层 (事务、审计、生命周期流转)
│   │   └── webhook/                # GitHub / GitLab / 通用 Webhook 事件解析适配
│   ├── pkg/                        # 跨模块通用工具库 (CSRF、Logging、Proxy 等)
│   ├── Makefile                    # 后端独立编译 Makefile
│   └── go.mod / go.sum             # Go 依赖配置
├── frontend/                       # 前端 React 18 模块根目录
│   ├── src/                        # 前端主应用源码
│   │   ├── api/                    # 后端 REST API 请求客户端封装
│   │   ├── containers/             # 页面与主要功能容器 (PlatformDashboard, Pipelines 等)
│   │   ├── mock/                   # 本地独立开发的完整 Mock 数据集
│   │   └── routes/                 # 页面路由声明
│   ├── packages/                   # 独立共享 npm 包 (components, utils, graph)
│   ├── vite.config.js              # Vite 配置文件 (代理规则、构建规则)
│   └── package.json                # 前端依赖与构建脚本
├── deploy/                         # 容器与 Kubernetes 部署物料
├── docs/                           # 官方文档库
│   ├── dev/
│   │   ├── api.md                  # 后端接口完整规范文档
│   │   ├── native-build.md         # 原生 npm 与 go 编译指南
│   │   └── README.md               # 开发者环境指南
│   └── platform/
│       └── DEVELOPMENT.md          # 本文档
├── Makefile                        # 根目录全局 Makefile
├── build.ps1                       # Windows 专属构建脚本
└── DEVELOPMENT.md                  # 根目录快速上手指南
```

---

## 3. 架构分层依赖原则

为保证代码可维护性与可测试性，后端代码必须严格遵守**单向依赖**原则：

$$\text{api (HTTP Handler)} \longrightarrow \text{service (业务逻辑)} \longrightarrow \text{repository / executor / compiler (数据访问与外部抽象)}$$

- **Handler 层 (`internal/api`)**：仅负责解析 HTTP 请求参数、调用 Service 层方法、输出统一的 JSON 包装响应，**严禁直接访问数据库连接或 K8s 客户端**。
- **Service 层 (`internal/service`)**：封装具体业务规则、编排事务与审计日志。Service 仅依赖仓储的 **Interface 接口**，绝不依赖具体 SQL 实现，确保单元测试时可 100% 注入 Mock。
- **Repository 层 (`internal/repository`)**：实现 SQL 增删改查。使用 `Rebind()` 自动适配 SQLite/MySQL（`?` 占位符）与 PostgreSQL（`$1, $2` 占位符）。
- **Executor 层 (`internal/executor`)**：封装对流水线执行生命周期的抽象，提供 `K8sExecutor`（面向真实集群）与 `FakeExecutor`（本地离线运行与自动化测试）。

---

## 4. 本地日常开发模式

### 4.1 模式一：纯前端 Mock 独立开发（无须启动后端）
前端包含完整的数据 Mock 层，脱离后端即可完成页面设计与组件开发。
```bash
# 方式 A：使用根目录 Makefile
make dev-frontend

# 方式 B：使用原生 npm
cd frontend && npm run dev
```
服务将在 `http://localhost:8000/` 启动，所有数据请求均由本地 Mock 拦截器响应。

### 4.2 模式二：前后端联合联调开发
```bash
# 终端 1：启动后端服务 (监听 8080 端口，自动初始化 SQLite 数据库)
make run-backend
# 或原生命令：cd backend && go run -tags dev ./cmd/platform -env dev

# 终端 2：启动前端联调模式 (Vite 将 /api/platform 与 /apis/ 代理至 8080)
cd frontend && npm run prod
```
浏览器打开 `http://localhost:8000/`，默认管理员账号密码为：`admin` / `admin123`。

---

## 5. 统一编译命令总览

### 5.1 使用根目录 Makefile
```bash
make help                   # 查看所有支持的开发与编译指令
make dev-frontend           # 启动前端 Mock 开发服务器
make build-frontend         # 编译前端静态产物至 frontend/dist
make test-frontend          # 运行前端单元测试
make lint-frontend          # 前端代码检查与校验
make run-backend            # 运行后端开发服务
make build-backend          # 编译后端 Windows 二进制产物
make build-backend-linux    # 静态交叉编译 Linux amd64 二进制产物
make build-backend-all      # 同时编译 Windows 与 Linux 二进制
make test-backend           # 运行后端单元测试套件
```

### 5.2 使用 Windows PowerShell `build.ps1`
```powershell
# 编译全套（前端 + 后端 Windows 版）
.\build.ps1 -Frontend -Target windows

# 交叉编译 Linux 版后端
.\build.ps1 -Target linux

# 运行测试并编译全平台产物
.\build.ps1 -Test -Target all
```

### 5.3 使用原生 `npm` 与 `go` 原生编译
完整细节请参阅：👉 **[docs/dev/native-build.md](../dev/native-build.md)**。

---

## 6. 数据库设计与迁移管理

1. **自动迁移机制**：
   后端服务启动时会自动检查数据库版本（基于 Goose 规范），并在初始化连接池后执行安全平滑升级，无需手动执行 DDL。
2. **连接池与锁控制**：
   - SQLite 默认开启 WAL 日志模式（`PRAGMA journal_mode=WAL;`）并设置 `busy_timeout=5000`，有效避免并发写争用。
   - 生产环境建议根据 QPS 配置连接池大小（`max_open_conns` 与 `max_idle_conns`）。
3. **新增数据表或字段规范**：
   - 迁移脚本放置于 `backend/db/migrations/`。
   - 命名遵循序号规范：`000X_description.sql`，必须同时包含 `-- +goose Up` 与 `-- +goose Down`。
   - 遵循“只增不改”原则，历史已合入的迁移脚本严禁修改。

---

## 7. 质量保障与测试规范

| 测试层级 | 测试目标 | 执行命令 | 说明 |
|:---|:---|:---|:---|
| **后端单测** | Repository、Config、Server API 流程 | `cd backend && go test -v ./...` | 使用内存 SQLite 或临时文件运行真实事务与接口断言 |
| **前端单测** | 组件渲染、工具函数、状态流转 | `cd frontend && npm test` | 基于 Vitest 与 Testing Library 进行断言 |
| **代码规范** | Go 语法与依赖健康 | `cd backend && go vet ./...` | 遵循标准 Go 编码格式 |
| **代码格式** | 前端 ESLint & Prettier | `cd frontend && npm run lint` | 统一团队样式与代码规范 |

---

## 8. 单容器与生产发布流程

1. **静态产物内嵌托管**：
   前端执行 `npm run build` 生成 `frontend/dist` 后，后端启动时会自动发现并注册静态路由，任意 SPA 页面刷新均通过 `NoRoute` 自动回退至 `index.html`，无需额外配置 Nginx 即可独立提供完整的 Web 服务。
2. **纯静态可执行文件**：
   通过 `CGO_ENABLED=0 go build -ldflags "-s -w"` 编译出的后端二进制文件不依赖任何系统的动态链接库（glibc 等），可直接在 Alpine Linux、Ubuntu 或无基础运行时的 Scratch/Distroless 容器镜像中极简运行。
3. **版本号与提交追踪**：
   在构建流程中通过 `-X main.version=$(VERSION)` 注入版本信息，运维可通过执行 `./platform -version` 或调用 `GET /api/platform/v1/version` 实时核对。
