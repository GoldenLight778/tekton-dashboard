# DevOps 平台原生编译与运行指南 (npm & go 原生编译)

本文档面向需要在**无 `make` 环境**（如纯净 Windows PowerShell / CMD、轻量级 Docker CI 构建镜像、独立开发机器）下直接使用系统原生工具链（`npm` 和 `go`）进行编译、测试、跨平台构建及本地联调的开发者与运维工程师。

---

## 1. 编译体系与目录结构总览

DevOps 平台采用前后端工程解耦架构：
- **前端工程**：位于 `frontend/`，基于 **Node.js + React 18 + Vite + Carbon Design**。
- **后端工程**：位于 `backend/`，基于 **Go 1.22+ (标准 1.27.x) + Gin + SQLite/MySQL/Postgres**，集成内置轻量级 DSL 编译器与 Tekton K8s 调度适配器。
- **构建产物联动**：前端编译产物输出至 `frontend/dist`；后端启动时会自动检测并托管前端静态产物，实现单可执行文件或单容器的完整平台服务。

```
tekton-dashboard/
├── frontend/               # 前端工程根目录 (npm/vite)
│   ├── package.json        # 前端依赖与脚本定义
│   ├── vite.config.js      # Vite 构建与开发代理配置
│   ├── src/                # 平台与仪表盘 React 源码
│   └── dist/               # 前端编译输出目录 (静态 html/js/css)
├── backend/                # 后端工程根目录 (go)
│   ├── go.mod / go.sum     # Go 模块依赖定义
│   ├── cmd/platform/       # 平台主启动入口 (main.go)
│   ├── configs/            # 多环境内嵌配置文件 (dev.yaml, prod.yaml 等)
│   ├── internal/           # 核心业务、API、数据仓储与 K8s 交互包
│   └── bin/                # 编译二进制输出目录
├── Makefile                # 统一 Make 包装脚本 (可选)
├── build.ps1               # 统一 PowerShell 编译脚本 (可选)
└── docs/                   # 项目完整技术与使用文档
```

---

## 2. 环境前置准备与检查

在开始编译前，请确保开发环境已安装以下基础工具链：

### 2.1 Go 编译环境
- **版本要求**：Go ≥ 1.22.0（推荐 1.24.x 或更高，当前环境验证通过 Go 1.27.4）。
- **验证命令**：
  ```bash
  go version
  ```
- **国内环境建议设置 Go Proxy 加速**：
  ```bash
  go env -w GOPROXY=https://goproxy.cn,direct
  ```

### 2.2 Node.js & npm 环境
- **版本要求**：Node.js ≥ 18.0.0（推荐 Node.js 20.x 或 24.x，当前环境验证通过 Node.js v24.18.0，npm 11.16.0）。
- **验证命令**：
  ```bash
  node -v
  npm -v
  ```
- **国内环境建议设置 npm 镜像加速**（可选）：
  ```bash
  npm config set registry https://registry.npmmirror.com
  ```

---

## 3. 前端原生编译指南 (`npm`)

所有前端操作均在 `frontend/` 目录下执行。

### 3.1 进入前端目录并安装依赖

```bash
cd frontend

# 本地首次或更新依赖安装：
npm install

# 或在 CI/CD 纯净构建环境安装（严格对齐 package-lock.json）：
npm ci
```

### 3.2 本地开发服务器启动

前端提供了两种本地运行模式：

#### 模式 A：纯 Mock 数据独立开发模式（无需启动后端）
使用内置的 Mock 数据层拦截 API，可直接进行 UI 页面、组件开发与调试。
- **命令**：
  ```bash
  npm run dev
  ```
- **实际执行**：`vite --mode mock`
- **默认监听地址**：`http://localhost:8000/`

#### 模式 B：前后端联调代理模式（对接真实后端）
Vite 将 `/api/platform` 与 `/apis/` 请求代理到运行在 `http://localhost:8080` 的后端服务。
- **命令**：
  ```bash
  npm run prod
  # 或
  npm start
  ```
- **实际执行**：`vite`
- **默认监听地址**：`http://localhost:8000/`

### 3.3 生产环境静态产物编译 (`build`)

将 React 前端项目编译打包为生产级别的静态 Web 资源（HTML、CSS、JS、字体与图片）。

- **命令**：
  ```bash
  npm run build
  ```
- **实际执行**：`vite build`
- **输出目录**：`frontend/dist/`
  - `frontend/dist/index.html`：单页应用入口 HTML
  - `frontend/dist/assets/`：压缩与哈希命名的 JavaScript 和 CSS 资源

### 3.4 产物包大小分析 (Bundle Analyze)
如果需要分析打包后各模块体积占用，可运行：
```bash
npm run build:analyze
```

### 3.5 前端单元测试与代码检查
- **运行单元测试 (Vitest)**：
  ```bash
  # 交互式监听测试：
  npm test

  # 单次运行所有测试（适用于 CI）：
  npx vitest run
  ```
- **执行代码风格检查与修复 (ESLint & Prettier)**：
  ```bash
  # 检查代码风格：
  npm run lint

  # 自动修复格式问题：
  npm run lint:fix
  ```

### 3.6 本地预览编译后的静态产物
在本地直接启动轻量级静态服务器预览 `dist` 目录产物：
```bash
npm run preview
```

---

## 4. 后端原生编译指南 (`go`)

所有后端操作均在 `backend/` 目录下执行。

### 4.1 进入后端目录并下载依赖

```bash
cd backend

# 下载并校验 go.mod 中的所有依赖项：
go mod download
go mod tidy
```

### 4.2 本地直接运行 (`go run`)

通过 `go run` 可以直接编译并在内存中启动后端服务，适合本地日常开发调试。

#### 构建标签 (`-tags`) 与环境参数 (`-env`)
系统支持通过 Go 构建标签注入默认配置：
- `dev`：开发环境，加载 `configs/dev.yaml`，使用开发 SQLite (`./data/platform-dev.db`)，日志级别为 Debug。
- `testenv`：测试环境，加载 `configs/test.yaml`，日志级别为 Debug。
- `prod`：生产环境，加载 `configs/prod.yaml`，日志级别为 Info。

#### 运行命令示例：
```bash
# 启动开发环境（默认）：
go run -tags dev ./cmd/platform -env dev

# 启动测试环境：
go run -tags testenv ./cmd/platform -env test

# 启动生产环境：
go run -tags prod ./cmd/platform -env prod
```

#### 常用启动参数覆盖 (CLI Flags)：
可直接在 `go run` 结尾附加命令行参数覆盖默认配置：
```bash
# 自定义监听端口（如 9090）：
go run ./cmd/platform -port 9090 -env dev

# 指定外部 YAML 配置文件路径：
go run ./cmd/platform -config ../custom-config.yaml

# 覆盖数据库驱动与数据源连接串（使用外部 MySQL 或 Postgres）：
go run ./cmd/platform -db-driver mysql -datasource "root:password@tcp(127.0.0.1:3306)/devops_prod?parseTime=true"

# 查看编译版本与信息后退出：
go run ./cmd/platform -version
```

### 4.3 原生二进制文件编译 (`go build`)

#### 关键编译参数说明：
1. **`CGO_ENABLED=0`**：
   - 平台采用纯 Go 实现的 SQLite 驱动（`modernc.org/sqlite`），因此**完全不需要 C 编译器（gcc / MinGW）**。
   - 显式设置 `CGO_ENABLED=0` 可生成纯静态二进制文件，跨平台移植性极佳，能在任意精简 Linux 容器（如 Alpine、Distroless）或裸机系统上无依赖运行。
2. **构建标签 `-tags <env>`**：
   - 编译开发版本传 `-tags dev`
   - 编译测试版本传 `-tags testenv`
   - 编译生产版本传 `-tags prod`
3. **连接器参数 `-ldflags "<flags>"`**：
   - `-s -w`：裁剪调试符号表与 DWARF 信息，可使二进制体积大幅缩减约 40%（从约 65MB 缩减至 38MB 左右）。
   - `-X main.version=0.1.0-alpha`：动态注入应用主版本号。
   - `-X devops-platform/internal/config.BuildEnv=prod`：动态注入编译环境变量。

---

### 4.4 跨操作系统原生编译命令详解

请根据你当前的操作系统终端选择对应命令：

#### 选项 A：在 Windows 环境编译 (PowerShell)

```powershell
cd backend

# 确保输出目录存在
if (-not (Test-Path "bin")) { New-Item -ItemType Directory -Path "bin" | Out-Null }

# 1. 编译 Windows 本地二进制文件 (platform.exe)
$env:CGO_ENABLED="0"
$env:GOOS="windows"
$env:GOARCH="amd64"
go build -tags dev -ldflags "-s -w -X main.version=0.1.0-alpha -X devops-platform/internal/config.BuildEnv=dev" -o bin/platform-windows-amd64.exe ./cmd/platform
Copy-Item -Path "bin/platform-windows-amd64.exe" -Destination "bin/platform.exe" -Force

# 2. 交叉编译 Linux 64位无依赖静态二进制文件
$env:CGO_ENABLED="0"
$env:GOOS="linux"
$env:GOARCH="amd64"
go build -tags prod -ldflags "-s -w -X main.version=0.1.0-alpha -X devops-platform/internal/config.BuildEnv=prod" -o bin/platform-linux-amd64 ./cmd/platform
```

#### 选项 B：在 Windows 环境编译 (经典 CMD)

```cmd
cd backend
if not exist bin mkdir bin

REM 1. 编译 Windows 可执行程序
set CGO_ENABLED=0
set GOOS=windows
set GOARCH=amd64
go build -tags dev -ldflags "-s -w -X main.version=0.1.0-alpha -X devops-platform/internal/config.BuildEnv=dev" -o bin\platform-windows-amd64.exe .\cmd\platform
copy /Y bin\platform-windows-amd64.exe bin\platform.exe

REM 2. 交叉编译 Linux 可执行程序
set CGO_ENABLED=0
set GOOS=linux
set GOARCH=amd64
go build -tags prod -ldflags "-s -w -X main.version=0.1.0-alpha -X devops-platform/internal/config.BuildEnv=prod" -o bin\platform-linux-amd64 .\cmd\platform
```

#### 选项 C：在 Linux 或 macOS (Bash / Zsh) 环境编译

```bash
cd backend
mkdir -p bin

# 1. 编译当前平台 (Linux amd64) 生产环境二进制
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -tags prod \
  -ldflags "-s -w -X main.version=0.1.0-alpha -X devops-platform/internal/config.BuildEnv=prod" \
  -o bin/platform-linux-amd64 ./cmd/platform

# 2. 交叉编译 Windows amd64 二进制
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build \
  -tags dev \
  -ldflags "-s -w -X main.version=0.1.0-alpha -X devops-platform/internal/config.BuildEnv=dev" \
  -o bin/platform-windows-amd64.exe ./cmd/platform

# 3. 交叉编译 macOS (Apple Silicon ARM64) 二进制
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build \
  -tags prod \
  -ldflags "-s -w -X main.version=0.1.0-alpha -X devops-platform/internal/config.BuildEnv=prod" \
  -o bin/platform-darwin-arm64 ./cmd/platform
```

### 4.5 后端单元测试执行

后端单元测试覆盖了系统配置加载、K8s 客户端绑定、多方言数据库与 SQLite 仓储、认证与核心 API 流程（M1-M5）。

```bash
cd backend

# 运行全量单元测试：
go test -v ./...

# 包含特定标签的测试：
go test -tags dev -v ./...

# 运行指定包的单测（如 server 模块 API 级测试）：
go test -v ./internal/server
```

---

## 5. 前后端联合原生打包与部署全流程 (End-to-End Workflow)

当需要将前后端打包为一个自包含的单体服务直接运行时，按照以下四步即可完成：

```
[步骤 1: 原生 npm 构建前端]  --> 输出静态产物 frontend/dist/
                                    │
                                    ▼
[步骤 2: 后端静态路由自动感知] <-- 检测 ./dist 或 ../frontend/dist
                                    │
                                    ▼
[步骤 3: 原生 go build 编译]  --> 输出后端二进制 backend/bin/platform.exe
                                    │
                                    ▼
[步骤 4: 启动后端服务验证]    --> 浏览器访问 http://localhost:8080/
```

### 步骤 1：原生编译前端资源
```bash
# 1. 进入前端目录编译静态包
cd frontend
npm ci
npm run build
cd ..
```
*此时静态产物已保存在 `frontend/dist/`。*

### 步骤 2：静态资源目录托管机制
后端服务启动时会依次检测以下目录并自动托管前端静态文件及 SPA 路由重定向：
1. `./dist`（当前工作目录下的 dist）
2. `../frontend/dist`（上级目录的 frontend/dist）
3. `./frontend/dist`（子目录下的 frontend/dist）

若将二进制与静态产物分发至生产服务器，只需将 `frontend/dist` 内容拷贝至二进制同级目录的 `dist/` 下即可。

### 步骤 3：原生编译后端二进制
```bash
# 2. 进入后端目录进行静态编译
cd backend
# Windows:
$env:CGO_ENABLED="0"; $env:GOOS="windows"; $env:GOARCH="amd64"
go build -tags prod -ldflags "-s -w -X main.version=0.1.0-alpha -X devops-platform/internal/config.BuildEnv=prod" -o bin/platform.exe ./cmd/platform
cd ..
```

### 步骤 4：运行与验证
```bash
# 3. 运行平台服务
./backend/bin/platform.exe
```

打开浏览器访问 `http://localhost:8080/`：
- 可以直接看到 React 生产界面。
- 输入默认管理员账号：`admin`，密码：`admin123` 即可登录并使用全套流水线与项目管理能力。

---

## 6. 构建命令对照速查表 (Cheatsheet)

| 操作目标 | `Makefile` 命令 | `build.ps1` 命令 | 原生 `npm` / `go` 命令 |
|:---|:---|:---|:---|
| **启动前端 Mock 开发** | `make dev-frontend` | - | `cd frontend && npm run dev` |
| **启动前端联调开发** | - | - | `cd frontend && npm run prod` |
| **编译前端静态资源** | `make build-frontend` | `.\build.ps1 -Frontend` | `cd frontend && npm run build` |
| **运行前端单元测试** | `make test-frontend` | - | `cd frontend && npx vitest run` |
| **前端代码格式检查** | `make lint-frontend` | - | `cd frontend && npm run lint` |
| **运行后端 (Dev)** | `make run-backend` | - | `cd backend && go run -tags dev ./cmd/platform -env dev` |
| **运行后端 (Prod)** | `make run-backend ENV=prod` | - | `cd backend && go run -tags prod ./cmd/platform -env prod` |
| **编译 Windows 二进制** | `make build-backend-windows` | `.\build.ps1 -Target windows` | `cd backend && set CGO_ENABLED=0&& set GOOS=windows&& set GOARCH=amd64&& go build -tags dev -ldflags "-s -w" -o bin/platform.exe ./cmd/platform` |
| **交叉编译 Linux 二进制** | `make build-backend-linux` | `.\build.ps1 -Target linux` | `cd backend && set CGO_ENABLED=0&& set GOOS=linux&& set GOARCH=amd64&& go build -tags prod -ldflags "-s -w" -o bin/platform-linux-amd64 ./cmd/platform` |
| **编译全平台二进制** | `make build-backend-all` | `.\build.ps1 -Target all` | 依次执行 Windows 与 Linux 的 `go build` |
| **后端单元测试** | `make test-backend` | `.\build.ps1 -Test` | `cd backend && go test -v ./...` |
| **全流程联合构建** | - | `.\build.ps1 -Frontend -Target windows` | 依次执行 `frontend npm run build` 和 `backend go build` |

---

## 7. 常见问题排查与 FAQ

### Q1: 在纯 Windows 环境下执行 `go build` 报找不到 gcc 或 clang？
- **解答**：这是因为默认开启了 CGO。必须显式设置环境变量 `CGO_ENABLED=0`。后端使用的 SQLite 驱动为纯 Go 实现的 `modernc.org/sqlite`，在 `CGO_ENABLED=0` 下能完美运行，无需安装任何 gcc/MinGW。

### Q2: 启动前端 `npm run dev` 报端口 8000 已被占用？
- **解答**：可在 `frontend/package.json` 中的 `scripts.dev` 后附加参数指定端口：
  ```bash
  npx vite --mode mock --port 8001
  ```

### Q3: 启动后端报 `database is locked` 错误？
- **解答**：SQLite 数据库在单进程中会自动配置 WAL 模式（`PRAGMA journal_mode=WAL;`）并设置 `busy_timeout=5000`。如果在本地同时运行了多个平台实例，或者使用外部 SQLite 浏览器锁定了 `.db` 文件，可能出现该提示。请关闭外部工具或多余进程，确保单一后端写操作连接。

### Q4: 独立运行后端二进制时访问浏览器显示 404？
- **解答**：请检查前端是否已编译生成 `dist` 目录。后端控制台启动时会输出：
  - 若找到产物：`serving frontend static assets from directory ...`
  - 若未找到产物：`frontend dist directory not found, running in API-only mode`
  执行 `cd frontend && npm run build` 生成 `frontend/dist` 后重新启动后端即可。
