# 参与 DevOps 平台贡献指南 (Contributing Guide)

感谢你关注并愿意为 DevOps 平台贡献代码或完善文档！我们非常欢迎来自社区与开发者的每一份 Pull Request (PR) 和建议。

在开始提交代码之前，请阅读以下开发规范与流程指引。

---

## 1. 行为准则 (Code of Conduct)

所有参与本项目的贡献者、维护者均需遵守我们的 [行为准则 (code-of-conduct.md)](./code-of-conduct.md)。请尊重所有参与者，保持友好、专业与协作的沟通氛围。

---

## 2. 准备开发环境

在开始编写代码前，请确保本地已配置好开发环境并阅读核心指引：
- **开发快速入门**：阅读 [DEVELOPMENT.md](./DEVELOPMENT.md)
- **后端架构设计**：阅读 [docs/platform/DEVELOPMENT.md](./docs/platform/DEVELOPMENT.md)
- **后端 API 规范**：阅读 [docs/dev/api.md](./docs/dev/api.md)
- **原生编译与构建**：阅读 [docs/dev/native-build.md](./docs/dev/native-build.md)

---

## 3. 分支与提交规范

### 3.1 分支拉取规范
1. 从 `main` 主分支检出最新的开发分支：
   ```bash
   git checkout main
   git pull origin main
   git checkout -b <type>/<short-description>
   ```
   *分支命名规范*：
   - 功能开发：`feat/m1-pipeline-crud`、`feat/add-github-oauth`
   - 问题修复：`fix/sse-log-reconnect`、`fix/db-lock-timeout`
   - 文档修改：`docs/update-native-build-guide`
   - 重构优化：`refactor/repository-interface`

### 3.2 Commit 提交信息规范
我们遵循标准语义化提交规范（Conventional Commits）：
```
<type>(<scope>): <subject>
```

- **`type`** 类别：
  - `feat`：新增功能特性
  - `fix`：修复 Bug
  - `docs`：仅文档修改
  - `refactor`：重构（既不新增功能也不修复 bug 的代码变动）
  - `test`：增加或修正测试用例
  - `chore`：构建流程、辅助工具或依赖项变动
- **`scope`** 范围（可选）：如 `api`, `server`, `compiler`, `executor`, `frontend`, `ui`
- **`subject`** 简述：动宾短语或简要中文说明。

*示例*：
```bash
git commit -m "feat(api): 新增流水线 DSL 实时语法校验接口"
git commit -m "fix(executor): 修复本地无 K8s 集群时 FakeExecutor 状态未更新问题"
git commit -m "docs: 完善 npm 与 go 原生编译指南"
```

---

## 4. 代码编写与分层原则

### 4.1 后端代码规范 (Go)
1. **严格单向依赖**：
   $$\text{api (Handler)} \longrightarrow \text{service} \longrightarrow \text{repository / executor}$$
   Handler 严禁越过 Service 直接访问数据库连接。
2. **错误处理**：
   - 严禁忽略任何返回的 `err`；
   - 使用 `fmt.Errorf("...: %w", err)` 包装错误上下文；
   - 敏感信息（如密码明文、用户 JWT、凭据 Secret）**严禁记录在 Zap 日志中**。
3. **格式化与静态检查**：
   提交前必须执行：
   ```bash
   cd backend
   go vet ./...
   ```

### 4.2 前端代码规范 (React 18)
1. 组件命名采用 PascalCase，包含 JSX 语法的代码文件统一使用 `.jsx` 后缀。
2. 保持与既有 IBM Carbon Design System 设计语言的一致性。
3. 提交前必须执行 Lint 检查：
   ```bash
   cd frontend
   npm run lint
   # 自动修复
   npm run lint:fix
   ```

---

## 5. 本地测试与验收 (Definition of Done)

在向仓库发起 Pull Request 前，请确保在本地完整执行并通过以下测试套件：

### 5.1 后端单元测试
```bash
cd backend
go test -v ./...
```
*所有测试用例必须均为 PASS。*

### 5.2 前端单元测试
```bash
cd frontend
npx vitest run
```

---

## 6. 发起 Pull Request (PR)

1. 将你的本地分支推送到远程仓库：
   ```bash
   git push origin <your-branch-name>
   ```
2. 在代码仓库中发起 Pull Request 指向 `main` 分支。
3. 在 PR 说明中清楚阐述：
   - **变更背景**：解决了什么问题或提供了什么新功能；
   - **核心改动**：涉及哪些模块（前端、后端 API、数据库迁移等）；
   - **验证结果**：附带本地测试通过的截图或日志输出。
4. 维护者进行 Code Review 后，符合要求的 PR 将被 Squash Merge 合并入主分支。
