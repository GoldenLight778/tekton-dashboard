# DevOps 平台开发指南 (Development Guide)

欢迎参与 DevOps 平台的开发！本项目已全面升级为以 **Go + Gin + SQLite/MySQL/Postgres** 为核心后端，以 **React 18 + Carbon Design + Vite** 为前端的现代化云原生 CI/CD 与多租户项目管理平台。

---

## 1. 架构总览与工程布局

项目采用前后端解耦的 Monorepo 单代码库结构：

```
tekton-dashboard/
├── frontend/               # 前端工程 (React 18, Vite, Carbon, Vitest)
│   ├── src/                # 平台与仪表盘业务源码
│   ├── packages/           # 共享组件库与工具包
│   └── dist/               # 生产编译输出目录
├── backend/                # 后端工程 (Go 1.22+, Gin, SQLite/MySQL/Postgres)
│   ├── cmd/platform/       # 平台主服务入口
│   ├── configs/            # 多环境配置 (dev.yaml, test.yaml, prod.yaml)
│   ├── internal/           # 业务逻辑、DSL编译器、K8s适配器、数据仓储
│   └── bin/                # 编译二进制产物目录
├── Makefile                # 跨平台 GNU Make 脚本封装
├── build.ps1               # 专为 Windows PowerShell 优化的构建脚本
└── docs/                   # 完整设计文档与 API 规范
```

---

## 2. 快速开始 (Quick Start)

### 2.1 依赖工具
- **Go**：≥ 1.22（推荐 1.24+ 或 1.27.x）
- **Node.js**：≥ 18（推荐 20.x 或 24.x）+ **npm**
- **可选工具**：`make` (Linux/macOS) 或 PowerShell (Windows)；若均不使用，请参阅 [原生编译指南](./docs/dev/native-build.md)。

---

### 2.2 启动本地开发

#### 方式一：使用 `Makefile` (推荐 Linux / macOS 或装有 make 的 Windows)

```bash
# 1. 终端一：启动前端开发服务器 (Mock 数据模式，端口 8000)
make dev-frontend

# 2. 终端二：启动后端 Go 服务 (默认 Dev 环境，端口 8080)
make run-backend
```

#### 方式二：使用 Windows PowerShell `build.ps1` 脚本

```powershell
# 编译前端并编译 Windows 版后端
.\build.ps1 -Frontend -Target windows

# 运行后端
.\backend\bin\platform.exe
```

#### 方式三：使用 `npm` 与 `go` 原生命令 (无须任何脚本包装)
详见专项文档：👉 **[原生编译与运行指南 (docs/dev/native-build.md)](./docs/dev/native-build.md)**

```bash
# 前端：
cd frontend
npm install
npm run dev        # Mock 模式，访问 http://localhost:8000/
# 或
npm run prod       # 联调代理模式，将 /api 代理至 8080

# 后端：
cd backend
go run -tags dev ./cmd/platform -env dev
```

---

## 3. 多环境配置体系

系统配置采用统一层级覆盖策略（外部命令行参数 > 环境变量 > 外部配置文件 > 内嵌环境配置）：

| 环境标识 (`ENV`) | 默认加载配置 | 默认数据库 | 日志级别 | K8s 运行模式 |
|:---|:---|:---|:---|:---|
| **`dev`** (开发) | `configs/dev.yaml` | `./data/platform-dev.db` (SQLite) | `debug` | 自动探测 (minikube / fake) |
| **`test`** (测试) | `configs/test.yaml` | 内存数据库或临时 SQLite | `debug` | `fake` 模式 |
| **`prod`** (生产) | `configs/prod.yaml` | 环境变量指定 SQLite/MySQL/PG | `info` | `in-cluster` 或 `kubeconfig` |

### 常用环境变量覆盖：
- `PLATFORM_ENV`：指定运行环境 (`dev` / `test` / `prod`)。
- `PLATFORM_ADDR`：指定服务监听地址，默认 `:8080`。
- `PLATFORM_DB_DRIVER`：指定数据库驱动 (`sqlite` / `mysql` / `postgres`)。
- `PLATFORM_DATASOURCE`：指定数据库连接数据源（如 SQLite 文件路径、MySQL/Postgres 连接串，兼容 `PLATFORM_DB_DSN`）。
- `PLATFORM_JWT_SECRET`：JWT 签名私钥（生产环境**必须**通过安全密钥注入）。
- `PLATFORM_SECRET_KEY`：用于凭据对称加密的 32 字节 AES-GCM 密钥。
- `PLATFORM_K8S_MODE`：Kubernetes 连接模式 (`minikube` / `kubeconfig` / `in-cluster` / `fake`)。若未连接集群，系统自动切换为 `fake` 模式，完整支持前端调试与流水线状态模拟。

---

## 4. 测试与代码质量

### 4.1 后端测试
```bash
# 使用 Makefile
make test-backend

# 或原生 Go 测试
cd backend
go test -v ./...
```

### 4.2 前端测试与 Lint
```bash
# 前端单元测试 (Vitest)
make test-frontend
# 或
cd frontend && npx vitest run

# 前端代码检查 (ESLint & Prettier)
make lint-frontend
# 或
cd frontend && npm run lint:fix
```

---

## 5. 核心文档导航

- 📘 **[后端接口规范文档 (docs/dev/api.md)](./docs/dev/api.md)**：包含全部 RESTful API、统一错误码、SSE 实时日志流、K8s 反向代理与 Webhook 规范。
- 🛠️ **[npm & go 原生编译指南 (docs/dev/native-build.md)](./docs/dev/native-build.md)**：无 Makefile 环境下的前端 Vite 原生打包、后端交叉编译与静态资源托管步骤。
- 🏛️ **[平台开发规范与架构设计 (docs/platform/DEVELOPMENT.md)](./docs/platform/DEVELOPMENT.md)**：分层依赖原则（API → Service → Repository）、数据库迁移流程与设计标准。
- 🚀 **[平台部署与安装手册 (docs/install.md)](./docs/install.md)**：单体二进制部署、容器化部署与 Kubernetes 安装指南。
- 👥 **[贡献者指南 (CONTRIBUTING.md)](./CONTRIBUTING.md)**：提交规范、分支管理与 PR 验收标准。
