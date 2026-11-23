# DevOps 平台开发流程文档

> 本文档描述平台（Go + Gin + SQLite 后端，React 前端，单容器部署）的开发约定与日常流程。
> 总体方案见方案文档；本文只讲"怎么开发、怎么提交、怎么发布"。

## 1. 技术栈

| 层 | 选型 | 说明 |
|---|---|---|
| 后端框架 | **Gin** | REST API、中间件、SSE 推送日志 |
| 数据库 | SQLite（`modernc.org/sqlite`，纯 Go，无 CGO） | WAL 模式，单副本 |
| 数据访问 | `sqlc` + `database/sql` | SQL 生成类型安全代码；Repository 接口隔离，便于换库 |
| 迁移 | `goose` | SQL 文件按序号管理 |
| 认证 | `golang-jwt/jwt` + `bcrypt` | 内置账号密码 |
| 校验 | `go-playground/validator`（Gin 内置集成） | 请求参数校验 |
| K8s | `client-go`（沿用） | 提交 / 监听 Tekton 资源 |
| 日志 | `zap`（沿用） | 结构化日志 |
| 前端 | React + Carbon + Vite（沿用） | `npm run dev` 带 mock 数据 |

## 2. 环境要求

- Go ≥ `go.mod` 声明版本（当前 1.27.x），确认 `go version` 可用
- Node.js（见 `.nvmrc`）+ npm
- Docker（构建镜像）
- 可选：`kind` + `kubectl`（联调真实 Tekton，见 `scripts/prepare-kind-cluster`）
- 可选工具：`sqlc`、`goose`、`golangci-lint`

```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
go install github.com/pressly/goose/v3/cmd/goose@latest
```

## 3. 目录约定

```
cmd/platform/          # 平台入口 main.go
internal/
  config/              # 配置加载（env + flag）
  server/              # Gin 引擎、路由注册、中间件、静态资源 embed
  api/                 # Handler：只做参数解析/响应，不写业务
  service/             # 业务逻辑
  repository/          # 接口 + sqlite 实现（sqlc 生成代码放 repository/sqlc）
  compiler/            # Pipeline DSL → Tekton CRD
  executor/            # K8s 提交/取消/删除封装（接口，可 Fake）
  syncer/              # 监听 PipelineRun/TaskRun 回写数据库
  crypto/              # 凭据加解密
  webhook/             # Git 平台回调
db/
  migrations/          # goose SQL 迁移
  queries/             # sqlc 查询 SQL
deploy/                # Dockerfile、K8s manifests
docs/platform/         # 本文档与设计文档
src/ , packages/       # 前端（原有）
```

**分层依赖方向（禁止反向）**：`api → service → repository / executor / compiler`。
Handler 不直接访问数据库或 K8s；service 依赖接口而非具体实现，方便单测注入 Fake。

## 4. 本地开发

### 4.1 前端独立开发（无需后端）

```bash
npm ci
npm run dev        # mock 数据，端口 8000
```

### 4.2 前后端联调

```bash
# 终端 1：后端（端口 8080，数据库文件 ./data/platform.db）
make run           # 等价于 go run ./cmd/platform

# 终端 2：前端，Vite 将 /api/platform 代理到 8080
npm run prod
```

### 4.3 配置项（环境变量）

| 变量 | 默认 | 说明 |
|---|---|---|
| `PLATFORM_ADDR` | `:8080` | 监听地址 |
| `PLATFORM_DB_PATH` | `./data/platform.db` | SQLite 文件 |
| `PLATFORM_JWT_SECRET` | 开发默认值 | **生产必须通过 Secret 注入** |
| `PLATFORM_SECRET_KEY` | 开发默认值 | 凭据 AES-GCM 主密钥 |
| `PLATFORM_KUBECONFIG` | 空 | 空则尝试 in-cluster，再无则进入"无 K8s"模式（executor 使用 Fake） |
| `PLATFORM_LOG_LEVEL` | `info` | 日志级别 |

## 5. 数据库流程

```bash
make migrate-new name=add_projects   # 新建迁移文件 db/migrations/000X_add_projects.sql
make migrate-up                      # 应用迁移
make migrate-down                    # 回滚一步
make sqlc                            # 修改 db/queries/*.sql 后重新生成代码
```

规则：
1. **迁移只增不改**：已合并的迁移文件不得修改，需要变更就新增一个迁移。
2. 每个迁移必须含 `Up` 和 `Down`。
3. 启动时自动执行 `migrate-up`（单容器场景），失败则进程退出。
4. SQLite 连接开启 `PRAGMA journal_mode=WAL; foreign_keys=ON; busy_timeout=5000`。
5. 写操作集中走同一连接池（`SetMaxOpenConns(1)` 用于写），避免 `database is locked`。

## 6. API 开发规范

- 前缀：`/api/platform/v1`；原有 `/apis/...` K8s 代理保持不变。
- 统一响应：

```json
{ "code": 0, "message": "ok", "data": {} }
```

错误时 `code != 0`，HTTP 状态码使用语义化值（400/401/403/404/409/500）。
- 列表接口统一分页参数 `page`、`pageSize`，响应含 `total`。
- 所有写接口写入 `audit_logs`。
- 凭据类接口响应**永不返回明文**。
- 鉴权：`Authorization: Bearer <jwt>`；角色 `admin / dev / viewer` 由中间件校验。
- 实时日志用 **SSE**（`text/event-stream`，Gin 中 `c.Stream` / `c.SSEvent`），状态推送可选 WebSocket。
- 新增接口步骤：`db/queries` 加 SQL → `make sqlc` → repository → service → api handler → 路由注册 → 测试。

## 7. 测试策略

| 类型 | 范围 | 命令 |
|---|---|---|
| 单元测试 | service、compiler（DSL→CRD 用 golden file） | `make test` |
| Repository 测试 | 用 `:memory:` 或临时文件 SQLite 跑真实 SQL | `make test` |
| API 测试 | `httptest` + Gin，executor 注入 Fake | `make test` |
| 前端测试 | vitest | `npm test` |
| E2E | kind 集群 + 真实 Tekton（M2 起） | `make e2e` |

要求：新增功能必须带测试；compiler 的变更必须更新 golden 文件并在 PR 说明差异。

## 8. 代码规范

- Go：`gofmt` / `goimports`，`golangci-lint run`，错误用 `fmt.Errorf("...: %w", err)` 包装，不吞错误。
- 日志不得打印密码、token、凭据明文。
- 前端：`npm run lint`，样式沿用既有简约风格，不引入新设计体系。
- 提交信息：`feat|fix|refactor|docs|test|chore(scope): 描述`。

## 9. 分支与 PR 流程

1. 从 `main` 拉分支：`feat/<milestone>-<简述>`，如 `feat/m1-pipeline-crud`。
2. 小步提交，保持每个 PR 只做一件事、可独立回滚。
3. 提 PR 前本地通过：
   ```bash
   make lint test
   npm run lint && npm test
   ```
4. PR 必须包含：变更说明、测试说明、涉及迁移时的回滚方案。
5. 至少一人 review 后合并（squash merge）。

## 10. 构建与发布（单容器）

```bash
make build          # 先 npm run build，再 go build（embed 前端产物）
make docker         # 构建镜像 platform:<tag>
docker run -p 8080:8080 -v $(pwd)/data:/data platform:dev
```

- Dockerfile 多阶段：node 构建前端 → golang 构建（`CGO_ENABLED=0`）→ distroless 运行。
- 数据目录 `/data` 必须挂卷（K8s 使用 PVC），**仅支持单副本**。
- K8s 部署需 ServiceAccount + RBAC（对 `tekton.dev` 资源的读写与 watch）。
- 版本号通过 `-ldflags "-X main.version=..."` 注入，`GET /api/platform/v1/version` 返回。

## 11. 里程碑与 Definition of Done

每个里程碑（M0–M5）完成需满足：
- [ ] 功能按验收标准可演示
- [ ] 新增代码有测试且 CI 通过
- [ ] 迁移可正向/回滚
- [ ] 文档（本文、API 文档）同步更新
- [ ] `docker run` 单容器可启动并完成该里程碑的核心场景

### M0 任务清单（下一步）

1. 初始化 `cmd/platform`、`internal/config`、`internal/server`（Gin + 健康检查 `/healthz`）
2. 引入 SQLite + goose，建 `users`、`projects` 初始迁移，启动自动迁移
3. 登录（bcrypt + JWT）、`/me`、鉴权中间件、首次启动创建 admin
4. 前端对接框架：`src/platform-api/` + 登录页 + mock
5. `go:embed` 前端产物 + SPA 路由回退
6. Makefile、Dockerfile，单容器跑通并登录

## 12. 常见问题

- **`database is locked`**：检查是否开启 WAL 和 `busy_timeout`，写操作是否并发过多。
- **本地没有 K8s**：不设 `PLATFORM_KUBECONFIG`，executor 自动使用 Fake，可完整演示配置与界面。
- **前端 `npm run dev` 没数据**：确认使用的是 mock 模式（`.env.mock`），联调请用 `npm run prod`。
