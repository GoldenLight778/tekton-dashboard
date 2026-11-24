# DevOps 平台安装与部署手册 (Installation & Deployment)

DevOps 平台支持多种灵活的部署形态：从**单二进制极简部署**、**Docker 单容器开箱即用运行**，到**Kubernetes 生产级集群托管**。本文档详细介绍不同形态下的部署步骤与配置规范。

---

## 目录
- [1. 部署形态选型建议](#1-部署形态选型建议)
- [2. 核心运行环境变量](#2-核心运行环境变量)
- [3. 部署方式一：单可执行文件直接部署 (裸机 / VM)](#3-部署方式一单可执行文件直接部署-裸机--vm)
- [4. 部署方式二：Docker 单容器部署 (推荐中小型团队)](#4-部署方式二docker-单容器部署-推荐中小型团队)
- [5. 部署方式三：Kubernetes 集群生产级部署](#5-部署方式三kubernetes-集群生产级部署)
- [6. 服务验证与初始化配置](#6-服务验证与初始化配置)

---

## 1. 部署形态选型建议

| 部署形态 | 适用场景 | 架构特征 | 维护复杂度 |
|:---|:---|:---|:---|
| **单二进制运行** | 本地演示、开发测试、个人服务器、离线环境 | 纯 Go 静态二进制文件 + 静态资源目录，内置 SQLite 数据库，零外部依赖 | 极低 |
| **Docker 单容器** | 中小型企业内部 DevOps 平台、私有云轻量交付 | 单镜像包含前后端产物，数据目录挂载卷持久化，支持外挂 MySQL/Postgres | 低 |
| **Kubernetes 集群** | 企业级高可用生产环境、深度对接云原生流水线 | Deployment + Service + Ingress，挂载 PVC，对接真实 Tekton Pipelines/Triggers | 中等 |

---

## 2. 核心运行环境变量

服务启动时支持通过环境变量注入或覆盖配置：

| 环境变量 | 默认值 | 说明 | 生产建议 |
|:---|:---|:---|:---|
| `PLATFORM_ENV` | `prod` | 运行环境标识 (`dev` / `test` / `prod`) | 生产环境设为 `prod` |
| `PLATFORM_ADDR` | `:8080` | HTTP 服务监听地址与端口 | 按需调整，如 `:80` 或 `:8080` |
| `PLATFORM_LOG_LEVEL` | `info` | 日志输出级别 (`debug` / `info` / `warn` / `error`) | 生产环境推荐 `info` |
| `PLATFORM_DB_DRIVER` | `sqlite` | 数据库驱动 (`sqlite` / `mysql` / `postgres`) | 可保持 `sqlite` 或切换为外部数据库 |
| `PLATFORM_DATASOURCE` | `./data/platform.db` | 数据库连接数据源（文件路径或连接串，兼容 `PLATFORM_DB_DSN`） | SQLite 必须指向持久化卷路径 |
| `PLATFORM_JWT_SECRET` | 开发默认值 | JWT 登录凭据签名密钥 | **生产环境必须通过安全随机密钥覆盖** |
| `PLATFORM_SECRET_KEY` | 开发默认值 | 凭据管理 AES-256-GCM 32 字节加密主密钥 | **生产环境必须显式注入独立保密密钥** |
| `PLATFORM_K8S_MODE` | `fake` | Kubernetes 运行模式 (`minikube` / `kubeconfig` / `in-cluster` / `fake`) | 部署在 K8s 内部时设为 `in-cluster` |
| `PLATFORM_KUBECONFIG` | 空 | K8s Kubeconfig 文件路径（当 mode 为 `kubeconfig` 时生效） | 按需挂载文件路径 |
| `PLATFORM_K8S_NAMESPACE` | `default` | 默认调度与监听的 Kubernetes 命名空间 | 推荐 `tekton-pipelines` 或专有业务空间 |

---

## 3. 部署方式一：单可执行文件直接部署 (裸机 / VM)

本方式适合在 Linux 服务器或 Windows 主机上直接托管运行。

### 3.1 编译产物准备
在编译机上完成前后端构建（可参阅 [原生编译指南 (docs/dev/native-build.md)](./dev/native-build.md)）：
```bash
# 1. 编译前端
cd frontend && npm ci && npm run build && cd ..

# 2. 静态编译 Linux amd64 二进制 (CGO_ENABLED=0 无动态依赖)
cd backend
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags prod -ldflags "-s -w -X main.version=0.1.0" -o bin/platform-linux-amd64 ./cmd/platform
cd ..
```

### 3.2 组织生产部署目录
将编译产物拷贝至目标服务器目录（例如 `/opt/devops-platform`）：
```
/opt/devops-platform/
├── platform                # 重命名后的二进制文件
├── dist/                   # 前端编译产物 (即 frontend/dist 拷贝过来)
└── data/                   # 数据持久化目录 (自动生成 platform.db)
```

### 3.3 配置 Systemd 系统守护进程 (Linux)
创建服务文件 `/etc/systemd/system/devops-platform.service`：

```ini
[Unit]
Description=Tekton DevOps Platform Service
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/devops-platform
ExecStart=/opt/devops-platform/platform
Restart=always
RestartSec=5
Environment=PLATFORM_ENV=prod
Environment=PLATFORM_ADDR=:8080
Environment=PLATFORM_DB_PATH=/opt/devops-platform/data/platform.db
Environment=PLATFORM_JWT_SECRET=c8f1e94b238a924b11f62d847c0e819a
Environment=PLATFORM_SECRET_KEY=0123456789abcdef0123456789abcdef

[Install]
WantedBy=multi-user.target
```

启动并设置开机自启：
```bash
systemctl daemon-reload
systemctl enable --now devops-platform
systemctl status devops-platform
```

---

## 4. 部署方式二：Docker 单容器部署 (推荐中小型团队)

通过 Docker 单镜像打包前端静态资源与 Go 二进制程序，直接以容器形态提供服务。

### 4.1 构建 Docker 镜像
基于项目多阶段 Dockerfile 构建极简镜像：
```bash
docker build -t devops-platform:latest -f deploy/Dockerfile .
```

### 4.2 运行 Docker 容器
```bash
# 确保宿主机持久化目录存在
mkdir -p /opt/devops-data

# 启动容器并挂载数据卷
docker run -d \
  --name devops-platform \
  -p 8080:8080 \
  -v /opt/devops-data:/data \
  -e PLATFORM_ENV=prod \
  -e PLATFORM_ADDR=:8080 \
  -e PLATFORM_DATASOURCE=/data/platform.db \
  -e PLATFORM_JWT_SECRET=your-random-jwt-secret-string \
  -e PLATFORM_SECRET_KEY=your-32-byte-aes-encryption-key-00 \
  --restart always \
  devops-platform:latest
```

---

## 5. 部署方式三：Kubernetes 集群生产级部署

将 DevOps 平台部署至已有 Kubernetes 集群，深度协同集群内的 Tekton Pipelines 与 Triggers。

### 5.1 前置条件
- Kubernetes 集群 ≥ 1.24
- 集群已安装 Tekton Pipelines（参考官方：`kubectl apply --filename https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml`）
- 目标命名空间（默认 `tekton-pipelines`）

### 5.2 部署平台清单文件
```bash
# 应用命名空间、ServiceAccount、RBAC 与 Deployment 清单
kubectl apply -k deploy/k8s
```

### 5.3 生产级存储卷配置 (PVC)
由于平台采用内嵌 SQLite 或单副本文件数据库，在 Kubernetes 中部署时需配置 `ReadWriteOnce` 的 PersistentVolumeClaim（PVC）挂载至 `/data`：
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: devops-platform-pvc
  namespace: tekton-pipelines
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
```

### 5.4 暴露服务 (Ingress 或 Port-Forward)
- **临时端口转发体验**：
  ```bash
  kubectl port-forward svc/tekton-dashboard -n tekton-pipelines 8080:8080
  ```
- **通过 Ingress 域名对外提供服务**：
  配置相应的 Ingress 规则，将请求路由至服务端口 `8080`。

---

## 6. 服务验证与初始化配置

1. 打开浏览器访问服务地址：`http://<server-ip>:8080/`
2. 使用系统默认初始化管理员账号登录：
   - **用户名**：`admin`
   - **初始密码**：`admin123`
3. 登录成功后即可创建团队项目空间、定义流水线 DSL、录入凭据并进行日常构建与监控！
