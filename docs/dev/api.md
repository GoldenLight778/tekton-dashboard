# DevOps 平台后端接口规范文档 (Backend API Reference)

本文档详细描述 DevOps 平台（基于 Go + Gin 构建）所提供的全部 HTTP RESTful API、Server-Sent Events (SSE) 实时日志流以及 Kubernetes API 反向代理与兼容层接口。

---

## 1. 基础规范与设计约定

### 1.1 基础路径 (Base URL)
- **平台核心 API 路径前缀**：`/api/platform/v1`
- **系统基础与健康检查端点**：`/healthz`, `/health`, `/readiness`, `/v1/properties`
- **Kubernetes 反向代理端点**：`/api/v1/*`, `/apis/*`

### 1.2 统一响应结构 (Standard Response Envelope)
所有 `/api/platform/v1` 接口均采用标准的 JSON 响应包装结构：

```json
{
  "code": 0,
  "message": "ok",
  "data": {}
}
```

- **`code`**：业务状态码。`0` 表示调用成功；非 `0` 表示发生业务或校验异常。
- **`message`**：状态描述文本。成功时固定为 `"ok"`，失败时返回具体的人性化错误说明。
- **`data`**：响应载荷（对象、数组或布尔结果），若无载荷或请求失败则为空。

### 1.3 统一错误码对照表 (Error Codes)

| HTTP 状态码 | 业务 Code | 说明 | 典型触发场景 |
|:---|:---|:---|:---|
| `200 OK` | `0` | 成功 | 请求处理正常 |
| `400 Bad Request` | `40001` | 请求参数无效 | 缺少必填参数（如项目名、DSL、凭据名称或密钥） |
| `400 Bad Request` | `40002` | 资源命名或格式非法 | 项目名/流水线名含有非法字符（仅允许英文字母、数字、短横线、下划线） |
| `400 Bad Request` | `40010` | 资源状态不允许此操作 | 取消流水线运行时，运行已结束（Succeeded/Failed/Cancelled） |
| `401 Unauthorized` | `40100` | 未认证 | 请求上下文缺少有效用户标识 |
| `401 Unauthorized` | `40101` | 缺少认证头 | 请求未携带 `Authorization` 头 |
| `401 Unauthorized` | `40102` | 认证头格式错误 | 未使用 `Bearer <token>` 格式 |
| `401 Unauthorized` | `40103` | Token 无效或已过期 | JWT 签名错误、Token 过期或格式损坏 |
| `401 Unauthorized` | `40110` | 账号或密码错误 | 登录验证失败 |
| `403 Forbidden` | `40301` | 禁止访问 | 上下文角色解析失败 |
| `403 Forbidden` | `40302` | 权限不足 | 用户角色不满足操作所需权限（如 viewer 尝试写操作） |
| `404 Not Found` | `40400` | 路由不存在 | 访问了不存在的 API 路径 |
| `404 Not Found` | `40401` | 资源不存在 | 指定的项目、流水线、运行实例、凭据或用户未找到 |
| `409 Conflict` | `40901` | 资源冲突已存在 | 同名项目、同名流水线或同名凭据已存在 |
| `500 Internal Error` | `50000` | 内部服务器错误 | 数据库异常、DSL 编译异常、K8s 交互异常 |
| `500 Internal Error` | `50001` | 凭据加解密或统计异常 | 凭据加密/解密失败、聚合统计查询失败 |

### 1.4 身份认证与 RBAC 权限控制
除公开端点（健康检查、登录接口、Webhook 回调、平台版本）外，所有平台接口均需携带 JWT Bearer Token：
```http
Authorization: Bearer <your_jwt_token>
```

#### 角色与权限模型 (RBAC)
系统内置三种权限角色（默认系统初始管理员为 `admin` / `admin123`）：
- **`admin` (超级管理员)**：拥有项目增删改查、凭据管理、流水线编排、运行触发/取消、审计日志查看所有权限。
- **`dev` (开发者)**：拥有项目查看、流水线增删改、运行触发/取消、凭据创建等日常开发权限。
- **`viewer` (只读访客)**：仅允许查看项目、流水线、运行历史与指标看板，禁止触发构建与修改配置。

### 1.5 通用分页规范
对于所有支持分页的查询列表接口，统一通过 URL Query 参数传递：
- **`page`** (int，可选，默认 `1`)：当前查询页码（从 1 开始）。
- **`pageSize`** (int，可选，默认 `20`)：每页条数。

响应结构中的 `data` 统一为：
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "items": [ ... ],
    "total": 42,
    "page": 1,
    "pageSize": 20
  }
}
```

---

## 2. 系统通用与健康检查接口

### 2.1 服务健康检查 (Health Check)
用于负载均衡探测、K8s Liveness 探针或人工健康探测。

- **请求方式**：`GET /healthz` 或 `GET /health`
- **鉴权要求**：公开，无须认证
- **响应示例** (`200 OK`)：
```json
{
  "status": "ok",
  "time": "2026-11-24T02:15:00Z"
}
```

### 2.2 服务就绪检查 (Readiness Probe)
用于 K8s Readiness 探针探测服务是否已加载配置并连接就绪。

- **请求方式**：`GET /readiness`
- **鉴权要求**：公开，无须认证
- **响应示例** (`200 OK`)：
```json
{
  "status": "ok"
}
```

### 2.3 平台版本与环境信息
获取平台后端版本与 Go 运行环境。

- **请求方式**：`GET /api/platform/v1/version`
- **鉴权要求**：公开，无须认证
- **响应示例** (`200 OK`)：
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "framework": "gin",
    "go": "1.27.x",
    "version": "0.1.0-alpha"
  }
}
```

### 2.4 Tekton Dashboard 属性兼容接口
提供原 Tekton Dashboard 前端所需的系统级环境配置，保证前端兼容无报错运行。

- **请求方式**：`GET /v1/properties`
- **鉴权要求**：公开，无须认证
- **响应示例** (`200 OK`)：
```json
{
  "DashboardNamespace": "default",
  "DashboardVersion": "0.1.0-alpha",
  "DefaultNamespace": "default",
  "LogoutURL": "",
  "PipelineNamespace": "default",
  "PipelineVersion": "v0.62.0",
  "ReadOnly": false,
  "StreamLogs": true,
  "TenantNamespaces": [
    "default"
  ],
  "TriggersNamespace": "default",
  "TriggersVersion": "v0.28.0"
}
```

---

## 3. 认证管理接口 (`/auth`)

### 3.1 用户账号登录
通过账号密码进行认证，验证成功后返回 JWT Token 及用户基本信息。

- **请求方式**：`POST /api/platform/v1/auth/login`
- **鉴权要求**：公开，无须认证
- **请求头**：`Content-Type: application/json`
- **请求参数**：

| 字段名 | 类型 | 必填 | 校验说明 |
|:---|:---|:---|:---|
| `username` | string | 是 | 账号名，默认初始管理员账号为 `admin` |
| `password` | string | 是 | 账号密码，默认初始密码为 `admin123` |

- **请求体示例**：
```json
{
  "username": "admin",
  "password": "admin123"
}
```

- **响应体示例** (`200 OK`)：
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": "usr-admin-01",
      "username": "admin",
      "role": "admin",
      "created_at": "2026-11-20T07:00:00Z",
      "updated_at": "2026-11-20T07:00:00Z"
    }
  }
}
```

- **失败响应示例** (`401 Unauthorized`)：
```json
{
  "code": 40110,
  "message": "incorrect username or password"
}
```

### 3.2 获取当前登录用户画像 (Profile)
通过 JWT Token 获取当前登录用户的基本信息与角色。

- **请求方式**：`GET /api/platform/v1/auth/me`
- **鉴权要求**：需要携带 `Authorization: Bearer <token>`
- **响应体示例** (`200 OK`)：
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "id": "usr-admin-01",
    "username": "admin",
    "role": "admin",
    "created_at": "2026-11-20T07:00:00Z",
    "updated_at": "2026-11-20T07:00:00Z"
  }
}
```

---

## 4. 项目空间管理接口 (`/projects`)

### 4.1 创建项目
新建一个多租户项目或代码工程空间，自动记录审计日志。

- **请求方式**：`POST /api/platform/v1/projects`
- **鉴权要求**：需要 `Bearer <token>` (角色: `admin` / `dev`)
- **请求体参数**：

| 字段名 | 类型 | 必填 | 说明 |
|:---|:---|:---|:---|
| `name` | string | 是 | 项目唯一名称，仅限小写字母、数字与 `-`、`_`，长度 3~50 |
| `description` | string | 否 | 项目详细描述或用途说明 |

- **请求体示例**：
```json
{
  "name": "payment-service",
  "description": "核心在线支付网关服务"
}
```

- **响应体示例** (`200 OK`)：
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "id": "prj_9b2e8a10",
    "name": "payment-service",
    "description": "核心在线支付网关服务",
    "created_by": "usr-admin-01",
    "created_at": "2026-11-24T02:20:00Z",
    "updated_at": "2026-11-24T02:20:00Z"
  }
}
```

- **重名冲突响应** (`409 Conflict`)：
```json
{
  "code": 40901,
  "message": "project name already exists"
}
```

### 4.2 获取项目列表 (分页)
查询平台所有项目空间列表。

- **请求方式**：`GET /api/platform/v1/projects`
- **查询参数**：
  - `page`: 页码，默认 `1`
  - `pageSize`: 每页条数，默认 `20`
- **响应体示例** (`200 OK`)：
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "items": [
      {
        "id": "prj_9b2e8a10",
        "name": "payment-service",
        "description": "核心在线支付网关服务",
        "created_by": "usr-admin-01",
        "created_at": "2026-11-24T02:20:00Z",
        "updated_at": "2026-11-24T02:20:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "pageSize": 20
  }
}
```

### 4.3 获取指定项目详情
- **请求方式**：`GET /api/platform/v1/projects/:id`
- **路径参数**：`id` (string，项目 ID)
- **响应体示例** (`200 OK`)：
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "id": "prj_9b2e8a10",
    "name": "payment-service",
    "description": "核心在线支付网关服务",
    "created_by": "usr-admin-01",
    "created_at": "2026-11-24T02:20:00Z",
    "updated_at": "2026-11-24T02:20:00Z"
  }
}
```

### 4.4 更新项目信息
修改指定项目的名称与描述，自动记录审计日志。

- **请求方式**：`PUT /api/platform/v1/projects/:id`
- **路径参数**：`id` (string，项目 ID)
- **请求体参数**：
```json
{
  "name": "payment-service-v2",
  "description": "重构后全新支付网关"
}
```
- **响应体示例** (`200 OK`)：
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "id": "prj_9b2e8a10",
    "name": "payment-service-v2",
    "description": "重构后全新支付网关",
    "created_by": "usr-admin-01",
    "created_at": "2026-11-24T02:20:00Z",
    "updated_at": "2026-11-24T02:25:00Z"
  }
}
```

### 4.5 删除项目
删除指定项目空间（关联的流水线和凭据将一并清理）。

- **请求方式**：`DELETE /api/platform/v1/projects/:id`
- **路径参数**：`id` (string，项目 ID)
- **响应体示例** (`200 OK`)：
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "deleted": true
  }
}
```

---

## 5. 流水线定义接口 (`/pipelines`)

平台支持轻量级声明式 DSL 语法（YAML 或 JSON），内置编译器会自动将 DSL 转换为标准的 Kubernetes Tekton `PipelineRun` / `Pipeline` CRD 规范。

### 5.1 流水线 DSL 格式规范说明

平台 DSL 支持按阶段（Stages）和任务（Tasks）顺序串联：
```yaml
name: build-and-deploy
stages:
  - name: test-stage
    tasks:
      - name: unit-test
        image: golang:1.24
        script: go test -v ./...
        env:
          CGO_ENABLED: "0"
  - name: build-stage
    tasks:
      - name: compile
        image: golang:1.24
        script: go build -o bin/app ./cmd/main.go
  - name: deploy-stage
    tasks:
      - name: deploy-k8s
        image: bitnami/kubectl:latest
        script: kubectl apply -f deploy/
```

### 5.2 在项目中创建流水线
- **请求方式**：`POST /api/platform/v1/projects/:id/pipelines`
- **路径参数**：`id` (string，所属项目 ID)
- **请求体参数**：

| 字段名 | 类型 | 必填 | 说明 |
|:---|:---|:---|:---|
| `name` | string | 是 | 流水线名称，同一项目下唯一 |
| `dsl_content` | string | 是 | 声明式 DSL 文本（YAML 或 JSON 格式） |

- **请求体示例**：
```json
{
  "name": "ci-pipeline",
  "dsl_content": "name: ci-pipeline\nstages:\n  - name: build\n    tasks:\n      - name: compile\n        image: golang:1.24\n        script: go build ./..."
}
```

- **响应体示例** (`200 OK`)：
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "id": "pip_3fa85f64",
    "project_id": "prj_9b2e8a10",
    "name": "ci-pipeline",
    "dsl_content": "name: ci-pipeline\nstages:\n  - name: build\n    tasks:\n      - name: compile\n        image: golang:1.24\n        script: go build ./...",
    "created_at": "2026-11-24T02:30:00Z",
    "updated_at": "2026-11-24T02:30:00Z"
  }
}
```

### 5.3 获取指定项目下的所有流水线
- **请求方式**：`GET /api/platform/v1/projects/:id/pipelines`
- **路径参数**：`id` (string，项目 ID)
- **查询参数**：`page` (默认 1), `pageSize` (默认 20)
- **响应体示例** (`200 OK`)：
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "items": [
      {
        "id": "pip_3fa85f64",
        "project_id": "prj_9b2e8a10",
        "name": "ci-pipeline",
        "dsl_content": "name: ci-pipeline...",
        "created_at": "2026-11-24T02:30:00Z",
        "updated_at": "2026-11-24T02:30:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "pageSize": 20
  }
}
```

### 5.4 获取流水线详情
- **请求方式**：`GET /api/platform/v1/pipelines/:id`
- **路径参数**：`id` (string，流水线 ID)
- **响应体示例** (`200 OK`)：
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "id": "pip_3fa85f64",
    "project_id": "prj_9b2e8a10",
    "name": "ci-pipeline",
    "dsl_content": "name: ci-pipeline\nstages:\n  - name: build\n    tasks:\n      - name: compile\n        image: golang:1.24\n        script: go build ./...",
    "created_at": "2026-11-24T02:30:00Z",
    "updated_at": "2026-11-24T02:30:00Z"
  }
}
```

### 5.5 更新流水线
修改流水线名称或更新 DSL 内容。

- **请求方式**：`PUT /api/platform/v1/pipelines/:id`
- **路径参数**：`id` (string，流水线 ID)
- **请求体参数**：
```json
{
  "name": "ci-pipeline-updated",
  "dsl_content": "name: ci-pipeline-updated\nstages:\n  - name: test\n    tasks:\n      - name: test\n        image: golang:1.24\n        script: go test ./..."
}
```
- **响应体示例** (`200 OK`)：
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "id": "pip_3fa85f64",
    "project_id": "prj_9b2e8a10",
    "name": "ci-pipeline-updated",
    "dsl_content": "name: ci-pipeline-updated...",
    "created_at": "2026-11-24T02:30:00Z",
    "updated_at": "2026-11-24T02:35:00Z"
  }
}
```

### 5.6 删除流水线
- **请求方式**：`DELETE /api/platform/v1/pipelines/:id`
- **路径参数**：`id` (string，流水线 ID)
- **响应体示例** (`200 OK`)：
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "deleted": true
  }
}
```

---

## 6. 流水线执行与运行历史 (`/runs`)

### 6.1 手动触发流水线执行
触发指定流水线运行。后台编译器自动将 DSL 转换为 Tekton CRD，并由执行引擎（真实 K8s 集群或 FakeExecutor）执行。

- **请求方式**：`POST /api/platform/v1/pipelines/:id/runs`
- **路径参数**：`id` (string，流水线 ID)
- **鉴权要求**：需要 `Bearer <token>`
- **响应体示例** (`200 OK`)：
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "id": "run_a8b21c90",
    "pipeline_id": "pip_3fa85f64",
    "k8s_pipelinerun_name": "ci-pipeline-run-1700000000",
    "namespace": "default",
    "status": "Running",
    "triggered_by": "admin",
    "started_at": "2026-11-24T02:40:00Z",
    "finished_at": null,
    "created_at": "2026-11-24T02:40:00Z"
  }
}
```

### 6.2 获取流水线运行实例详情
- **请求方式**：`GET /api/platform/v1/runs/:id`
- **路径参数**：`id` (string，运行实例 ID)
- **响应体示例** (`200 OK`)：
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "id": "run_a8b21c90",
    "pipeline_id": "pip_3fa85f64",
    "k8s_pipelinerun_name": "ci-pipeline-run-1700000000",
    "namespace": "default",
    "status": "Succeeded",
    "triggered_by": "admin",
    "started_at": "2026-11-24T02:40:00Z",
    "finished_at": "2026-11-24T02:41:15Z",
    "created_at": "2026-11-24T02:40:00Z"
  }
}
```

> **运行状态枚举 (`status`)**：
> - `Pending`：等待调度排队
> - `Running`：容器执行中
> - `Succeeded`：所有任务执行完毕且退出码为 0
> - `Failed`：某一阶段任务执行失败或容器异常
> - `Cancelled`：用户手动取消或终止

### 6.3 按流水线查询运行记录 (分页)
- **请求方式**：`GET /api/platform/v1/pipelines/:id/runs`
- **路径参数**：`id` (string，流水线 ID)
- **查询参数**：`page` (默认 1), `pageSize` (默认 20)
- **响应体示例** (`200 OK`)：
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "items": [
      {
        "id": "run_a8b21c90",
        "pipeline_id": "pip_3fa85f64",
        "k8s_pipelinerun_name": "ci-pipeline-run-1700000000",
        "namespace": "default",
        "status": "Succeeded",
        "triggered_by": "admin",
        "started_at": "2026-11-24T02:40:00Z",
        "finished_at": "2026-11-24T02:41:15Z",
        "created_at": "2026-11-24T02:40:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "pageSize": 20
  }
}
```

### 6.4 按项目查询运行记录 (分页)
- **请求方式**：`GET /api/platform/v1/projects/:id/runs`
- **路径参数**：`id` (string，项目 ID)
- **查询参数**：`page` (默认 1), `pageSize` (默认 20)
- **响应体示例**：同 6.3。

### 6.5 取消正在执行的流水线
- **请求方式**：`POST /api/platform/v1/runs/:id/cancel`
- **路径参数**：`id` (string，运行实例 ID)
- **响应体示例** (`200 OK`)：
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "cancelled": true
  }
}
```

### 6.6 SSE 实时日志流推送 (Server-Sent Events)
通过长连接流式接收任务执行过程中的实时日志，适用于前端实时日志看板展示。

- **请求方式**：`GET /api/platform/v1/runs/:id/logs/sse`
- **路径参数**：`id` (string，运行实例 ID)
- **响应头**：
  ```http
  Content-Type: text/event-stream
  Cache-Control: no-cache
  Connection: keep-alive
  Transfer-Encoding: chunked
  ```
- **SSE 事件格式**：
```
event: message
data: [2026-11-24T02:40:01Z] [compile] Cloning repository...

event: message
data: [2026-11-24T02:40:05Z] [compile] go: downloading modules...

event: message
data: [2026-11-24T02:40:15Z] [compile] Build completed successfully.
```
- **前端接入示例**：
```javascript
const eventSource = new EventSource(`/api/platform/v1/runs/${runId}/logs/sse`, {
  headers: { Authorization: `Bearer ${token}` }
});

eventSource.onmessage = (event) => {
  console.log("Log chunk:", event.data);
};

eventSource.onerror = (err) => {
  console.error("SSE connection error", err);
  eventSource.close();
};
```

---

## 7. 凭据密钥管理接口 (`/credentials`)

为保护敏感密钥（如 Git 访问令牌、SSH 私钥、Kubernetes Kubeconfig），所有写入的数据均采用 **AES-256-GCM** 高强度对称加密后落库。查询接口**永远不会返回密文或明文**，仅提供前端掩码字符（Masked Value）。

### 7.1 支持的凭据类型 (`type`)
- `git_token`：Git 平台 Personal Access Token (如 GitHub / GitLab / Gitee Token)
- `ssh_key`：用于拉取代码的 SSH Private Key
- `kubeconfig`：Kubernetes 集群访问配置文本
- `secret_text`：通用的机密环境变量或 API 密钥（默认）

### 7.2 在项目中创建凭据
- **请求方式**：`POST /api/platform/v1/projects/:id/credentials`
- **路径参数**：`id` (string，项目 ID)
- **请求体参数**：

| 字段名 | 类型 | 必填 | 说明 |
|:---|:---|:---|:---|
| `name` | string | 是 | 凭据名称，同项目下不可重复 |
| `type` | string | 否 | 凭据类型，默认为 `secret_text` |
| `secret` | string | 是 | 明文机密内容，后台自动加密存储 |

- **请求体示例**：
```json
{
  "name": "github-ci-token",
  "type": "git_token",
  "secret": "ghp_1234567890abcdefghijklmnopqrstuvwxyz"
}
```

- **响应体示例** (`200 OK`)：
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "id": "crd_8192a0e2",
    "project_id": "prj_9b2e8a10",
    "name": "github-ci-token",
    "type": "git_token",
    "masked_value": "ghp_****************************wxyz",
    "created_at": "2026-11-24T02:45:00Z",
    "updated_at": "2026-11-24T02:45:00Z"
  }
}
```

### 7.3 获取项目下的凭据列表
- **请求方式**：`GET /api/platform/v1/projects/:id/credentials`
- **路径参数**：`id` (string，项目 ID)
- **响应体示例** (`200 OK`)：
```json
{
  "code": 0,
  "message": "ok",
  "data": [
    {
      "id": "crd_8192a0e2",
      "project_id": "prj_9b2e8a10",
      "name": "github-ci-token",
      "type": "git_token",
      "masked_value": "ghp_****************************wxyz",
      "created_at": "2026-11-24T02:45:00Z",
      "updated_at": "2026-11-24T02:45:00Z"
    }
  ]
}
```

### 7.4 删除指定凭据
- **请求方式**：`DELETE /api/platform/v1/credentials/:id`
- **路径参数**：`id` (string，凭据 ID)
- **响应体示例** (`200 OK`)：
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "deleted": true
  }
}
```

---

## 8. 观测度量与监控大盘接口 (`/metrics`)

### 8.1 平台全局大盘指标 (Dashboard Metrics)
为平台首页看板提供全局统计分析数据，包括项目数、流水线数、构建成功率、平均耗时、状态分布以及近期运行记录。

- **请求方式**：`GET /api/platform/v1/metrics/dashboard`
- **鉴权要求**：需要 `Bearer <token>`
- **响应体示例** (`200 OK`)：
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "total_projects": 8,
    "total_pipelines": 24,
    "total_runs": 150,
    "succeeded_runs": 138,
    "failed_runs": 9,
    "running_runs": 2,
    "cancelled_runs": 1,
    "success_rate": 92.0,
    "avg_duration_seconds": 68.5,
    "status_counts": {
      "Succeeded": 138,
      "Failed": 9,
      "Running": 2,
      "Cancelled": 1
    },
    "recent_runs": [
      {
        "id": "run_a8b21c90",
        "pipeline_id": "pip_3fa85f64",
        "pipeline_name": "ci-pipeline",
        "project_name": "payment-service",
        "status": "Succeeded",
        "triggered_by": "admin",
        "duration_seconds": 75,
        "started_at": "2026-11-24T02:40:00Z",
        "finished_at": "2026-11-24T02:41:15Z",
        "created_at": "2026-11-24T02:40:00Z"
      }
    ]
  }
}
```

---

## 9. 审计日志查询接口 (`/audit-logs`)

所有关键写操作（包括创建/修改/删除项目、创建/修改流水线、触发与取消运行、创建/删除凭据）均自动通过事务记录到审计日志表中。

### 9.1 查询平台审计日志 (分页)
- **请求方式**：`GET /api/platform/v1/audit-logs`
- **鉴权要求**：需要 `Bearer <token>` (角色: `admin` / `dev`)
- **查询参数**：
  - `page`: 当前页码，默认 `1`
  - `pageSize`: 每页条数，默认 `20`
- **响应体示例** (`200 OK`)：
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "items": [
      {
        "id": 1,
        "user_id": "usr-admin-01",
        "action": "TRIGGER_PIPELINE",
        "resource_type": "pipeline",
        "resource_id": "pip_3fa85f64",
        "detail": "triggered run run_a8b21c90 for pipeline ci-pipeline",
        "ip": "127.0.0.1",
        "created_at": "2026-11-24T02:40:00Z"
      },
      {
        "id": 2,
        "user_id": "usr-admin-01",
        "action": "CREATE_PROJECT",
        "resource_type": "project",
        "resource_id": "prj_9b2e8a10",
        "detail": "created project payment-service",
        "ip": "127.0.0.1",
        "created_at": "2026-11-24T02:20:00Z"
      }
    ],
    "total": 2,
    "page": 1,
    "pageSize": 20
  }
}
```

---

## 10. Webhook 自动化触发接口 (`/webhooks`)

支持接收主流 Git 托管平台（GitHub、GitLab、Gitee 或通用 HTTP Webhook）的 Push 或 Merge 事件，自动解析分支与提交信息并触发目标项目的主流水线构建。

### 10.1 接收 Git Webhook 事件
- **请求方式**：`POST /api/platform/v1/webhooks/:id`
- **路径参数**：`id` (string，目标项目 ID)
- **鉴权要求**：公开，由 Webhook 签名或项目 ID 路由
- **支持平台识别与请求头**：
  - **GitHub**：自动识别 `X-GitHub-Event: push`，解析 `ref` (`refs/heads/main`)、`head_commit.id`、`head_commit.message`、`head_commit.author.name`
  - **GitLab**：自动识别 `X-Gitlab-Event: Push Hook`，解析 `ref`、`user_name`、`commits[0]`
  - **通用 JSON**：支持 `{ "branch": "main", "commit": "8f407a", "author": "dev", "message": "feat: xxx" }`

- **cURL 模拟触发示例**：
```bash
curl -X POST http://localhost:8080/api/platform/v1/webhooks/prj_9b2e8a10 \
  -H "Content-Type: application/json" \
  -H "X-GitHub-Event: push" \
  -d '{
    "ref": "refs/heads/release-1.0",
    "head_commit": {
      "id": "e4d909c290d0fb1ca068ffaddf22cbd0add91718",
      "message": "fix: resolve memory leak in worker pool",
      "author": { "name": "alice" }
    }
  }'
```

- **响应体示例** (`200 OK`)：
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "triggered": true,
    "pipeline_id": "pip_3fa85f64",
    "pipeline_name": "ci-pipeline",
    "run_id": "run_b7c32d18",
    "branch": "release-1.0",
    "commit": "e4d909c290d0fb1ca068ffaddf22cbd0add91718"
  }
}
```

---

## 11. Kubernetes API 反向代理与 Fallback 兼容层

为保持与原生 Tekton Dashboard 前端及扩展模块的 100% 兼容性，后端服务内置了 Kubernetes REST 反向代理与无集群时的智能 Mock 兜底逻辑：

### 11.1 在线模式（已连接 K8s 集群）
- **代理路径**：
  - `/api/v1/*` 自动反向代理至 Kubernetes API Server 的 `/api/v1/*`
  - `/apis/*` 自动反向代理至 Kubernetes API Server 的 `/apis/*`（如 `apis/tekton.dev/v1/pipelineruns`）
- **特性**：透明转发身份认证、支持长连接与流式输出，保持超时时间 30 秒。

### 11.2 离线 / 本地开发模式（Fake / 离线兜底）
当未配置 Kubeconfig 或集群离线时，系统自动提供兜底 Mock 响应，防止前端报 502/504 错误：
- `GET /api/v1/namespaces`：返回默认命名空间列表（`default`, `tekton-pipelines`）。
- `GET /apis/*`：返回空的 Tekton 资源列表，支持页面平滑浏览与无集群纯前端展示。
