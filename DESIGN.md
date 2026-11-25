---
name: Tekton DevOps 平台 (Tekton DevOps Dashboard)
description: Cloud-native CI/CD control deck for Kubernetes-native Tekton Pipelines, Triggers, Secrets, and label-scoped RBAC.
colors:
  core-violet-indigo: "#4f46e5"
  core-indigo-hover: "#4338ca"
  core-indigo-soft: "#eef2ff"
  pipeline-cyan: "#0ea5e9"
  status-green: "#10b981"
  status-green-deep: "#059669"
  alert-crimson: "#dc2626"
  carbon-graphite: "#0f172a"
  slate-ink: "#1e293b"
  steel-muted: "#64748b"
  surface-pure: "#ffffff"
  surface-panel: "#f8fafc"
  surface-recessed: "#f1f5f9"
  border-structural: "#e2e8f0"
  border-control: "#cbd5e1"
typography:
  display:
    fontFamily: "'IBM Plex Sans', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif"
    fontSize: "1.75rem"
    fontWeight: 700
    lineHeight: 1.25
    letterSpacing: "-0.02em"
  headline:
    fontFamily: "'IBM Plex Sans', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif"
    fontSize: "1.35rem"
    fontWeight: 700
    lineHeight: 1.3
  title:
    fontFamily: "'IBM Plex Sans', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif"
    fontSize: "1.15rem"
    fontWeight: 700
    lineHeight: 1.35
  body:
    fontFamily: "'IBM Plex Sans', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif"
    fontSize: "0.875rem"
    fontWeight: 400
    lineHeight: 1.5
  label:
    fontFamily: "'JetBrains Mono', 'IBM Plex Mono', ui-monospace, SFMono-Regular, monospace"
    fontSize: "0.78rem"
    fontWeight: 600
    lineHeight: 1.4
    letterSpacing: "0.02em"
rounded:
  sm: "6px"
  md: "8px"
  lg: "10px"
  xl: "12px"
  pill: "999px"
spacing:
  xs: "6px"
  sm: "10px"
  md: "16px"
  lg: "20px"
  xl: "28px"
  xxl: "36px"
components:
  button-primary:
    backgroundColor: "{colors.core-violet-indigo}"
    textColor: "{colors.surface-pure}"
    rounded: "{rounded.sm}"
    padding: "0 16px"
    height: "36px"
  button-primary-hover:
    backgroundColor: "{colors.core-indigo-hover}"
  button-secondary:
    backgroundColor: "{colors.surface-pure}"
    textColor: "{colors.slate-ink}"
    rounded: "{rounded.sm}"
    padding: "0 16px"
    height: "36px"
  button-ghost:
    backgroundColor: "{colors.surface-pure}"
    textColor: "{colors.core-violet-indigo}"
    rounded: "{rounded.sm}"
    padding: "0 14px"
    height: "36px"
  input-field:
    backgroundColor: "{colors.surface-pure}"
    textColor: "{colors.carbon-graphite}"
    rounded: "{rounded.sm}"
    padding: "0 12px"
    height: "36px"
  card-surface:
    backgroundColor: "{colors.surface-pure}"
    textColor: "{colors.carbon-graphite}"
    rounded: "{rounded.lg}"
    padding: "24px 28px"
  role-pill:
    backgroundColor: "{colors.core-indigo-soft}"
    textColor: "{colors.core-indigo-hover}"
    rounded: "{rounded.pill}"
    padding: "2px 10px"
---

# Design System: Tekton DevOps 平台 (Tekton DevOps Dashboard)

## Overview

**Creative North Star: "The Cloud-Native Control Deck"**

Tekton DevOps 平台的设计语言定位为**「云原生精密操控台」(The Cloud-Native Control Deck)**——面向高频操作 Kubernetes 与 Tekton CI/CD 资源的研发工程师、运维/SRE 与平台管理员。界面以高信噪比、结构化的工业级克制感为核心，摒弃一切冗余的营销式装饰、繁琐的底层机制说教文案与花哨的悬浮阴影，让工程师的视线第一时间锁定在命名空间作用域、流水线运行状态、触发器拓扑以及横向展开的 RBAC 权限矩阵上。

整个视觉系统采用**全面扁平化分层 (Tonal & Structural Flatness)**：深色碳素石墨顶栏固定全局导航与紧凑的 `30px` 用户菜单，主工作区则通过纯白工作面、浅灰面板色阶与 `1px` 结构化边框构建严密的秩序感。无论是独立极简的登录卡片，还是「角色配置 / 用户配置」双视图下的横向列表与下方详情联动面板，均遵循一致的紧凑间距与直接反馈。

**Key Characteristics:**
- **高信噪比与文案极简**：拒绝在界面上堆砌冗长的标签规则解释或演示账号提示，表单、列表与弹窗只保留直接服务于运维决策的字段与状态。
- **全面扁平的色阶与边框分层**：不依赖悬浮阴影制造虚假立体感，完全通过背景色阶（纯白工作面、浅灰次级面板、内嵌槽位色）与 `1px` 结构线界定信息边界。
- **主从联动的横向列表范式**：在 RBAC 等配置场景采用上方全宽横向数据列表（选中行左侧内嵌主色指示条 + 浅靛蓝底色高亮）与下方详细配置面板的上下联动结构。
- **硬朗工业级双字体体系**：UI 标题与正文采用清晰利落的 `IBM Plex Sans`，所有 Kubernetes 资源名、`ClusterRole` 标识、命名空间与 YAML/日志代码统一使用等宽字体 `JetBrains Mono` / `IBM Plex Mono`。

## Colors

调色板采用硬朗工业风格的高对比度配色：以深邃的碳素石墨色锚定文字与全局顶栏，用高辨识度的核心紫靛蓝驱动主操作与激活态，并辅以管线青与状态绿传达实时集群遥测信号。

### Primary
- **Core Violet-Indigo** (`#4f46e5`): 全局核心主色。用于主操作按钮（新建角色、新建用户、保存配置、创建流水线）、横向列表选中行左侧指示条、分段切换器激活文字以及表单聚焦边框。
- **Core Indigo Hover** (`#4338ca`): 主按钮与交互控件的悬停加深态，以及角色徽章的高对比文字色。
- **Core Indigo Soft** (`#eef2ff`): 主色低饱和度浅底。专用于横向列表当前选中行（`.rbac-list-row.selected`）的整行高亮背景与角色标签底色。

### Secondary
- **Pipeline Cyan** (`#0ea5e9`): 流水线与集群遥测辅助色。用于任务/触发器统计标识、深色终端代码高亮以及架构拓扑的次级流向指示。

### Tertiary
- **Status Green** (`#10b981`) & **Status Green Deep** (`#059669`): 集群与 PipelineRun 成功/活跃状态色（如 Namespace `Active`、`Succeeded` 运行状态点）。
- **Alert Crimson** (`#dc2626`): 危险操作与失败状态色（删除角色、删除用户、退出登录、`Failed` 构建状态）。

### Neutral
- **Carbon Graphite** (`#0f172a`): 最高层级标题文字、关键资源名称、登录页深色背景基调以及实时日志终端底色。
- **Slate Ink** (`#1e293b`): 列表正文、表格主体文字与次级按钮文本色。
- **Steel Muted** (`#64748b`): 表头说明、辅助备注文字、空状态提示与未激活图标色。
- **Surface Pure** (`#ffffff`): 核心工作卡片、横向列表容器、详情面板与输入框的纯白主表面。
- **Surface Panel** (`#f8fafc`): 列表表头（`thead th`）、内嵌命名空间配置框（`.user-ns-config-box`）、个人信息行与表格行悬停态背景。
- **Surface Recessed** (`#f1f5f9`): 代码标签（`.row-code-tag`）底色、内部细微分隔线与分段控制器槽位背景。
- **Border Structural** (`#e2e8f0`): 卡片外框、表格容器外框与面板区块分隔线的标准结构边框色。
- **Border Control** (`#cbd5e1`): 输入框、下拉选择框、快捷操作小按钮与行状态胶囊的控件边框色。

### Named Rules
**The Industrial Restraint Rule.** 核心紫靛蓝（`#4f46e5`）仅用于当前可交互的主动作、当前选中的列表行以及聚焦态控件，绝不作为大面积装饰背景铺满工作区。
**The Semantic Status Rule.** 绿色、青色与红色严格绑定 Kubernetes 资源与流水线的真实运行或权限语义，严禁用于无状态含义的纯视觉点缀。

## Typography

**Display & Body Font:** `IBM Plex Sans` (fallback: `-apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif`)
**Label & Mono Font:** `JetBrains Mono`, `IBM Plex Mono` (fallback: `ui-monospace, SFMono-Regular, Menlo, Consolas, monospace`)

**Character:** 严谨的工程控制台排版——无衬线字体保证中文与英文标签在紧凑表单中的极佳可读性，等宽字体则赋予所有 Kubernetes 资源标识符（如 `tekton-devops-worker`、`Namespace` 名称、YAML 字段）一目了然的代码级精确感。

### Hierarchy
- **Display** (`700`, `1.75rem`, `line-height: 1.25`, `letter-spacing: -0.02em`): 页面主标题（如「权限管理」「命名空间」）。
- **Headline** (`700`, `1.35rem`, `line-height: 1.3`): 独立登录页卡片标题与一级弹窗标题。
- **Title** (`700`, `1.15rem`, `line-height: 1.35`): 下方详情配置面板标题（如「工作人员 · 详细权限配置」）。
- **Body** (`400` / `500`, `0.85rem–0.875rem`, `line-height: 1.5`): 数据列表单元格、表单输入值与配置项说明文本。
- **Label / Mono** (`600`, `0.78rem`, `line-height: 1.4`): 表头字段（`0.78rem` 字重 `600`）、K8s `ClusterRole` 标识标签（`.row-code-tag`）以及状态胶囊标签。

### Named Rules
**The Native Identifier Mono Rule.** 凡是对应 Kubernetes 集群真实对象名称的文本（`ClusterRole` 名称、`Namespace` 名称、`Secret` 名称、CRD Kind），必须使用等宽字体加浅灰底框（`.row-code-tag`）渲染，与普通中文业务名称形成明确区分。

## Layout

整体采用左侧固定导航栏 + 顶部 `48px` 全局黑石顶栏 + 主工作区（`.platform-container`）的经典控制台框架：

- **容器宽度与内边距**：主内容区最大宽度 `1560px`，水平居中，四周留白 `28px 36px 64px`，确保高密度表格与 5×5 权限矩阵拥有充足的横向呼吸空间。
- **紧凑顶栏右区（Header Bar Right）**：顶栏右侧高度严格收敛，用户触发器高度固定为 `30px`（内边距 `0 10px`，字号 `0.78rem`），点击展开 `170px` 最小宽度的「个人信息 / 退出登录」下拉菜单。
- **主从上下联动布局（`.rbac-master-detail-layout`）**：
  - 顶部标题栏左侧将页面标题与分段视图切换器（`角色配置` / `用户配置`）水平并置，右侧放置带旋转反馈的「刷新」按钮与主创建按钮。
  - 上半区为全宽横向数据列表（`.rbac-horizontal-list-table`），单行紧凑展示核心属性；下半区紧跟当前选中项的详细配置面板（`.rbac-detail-panel`），上下间距固定为 `18px`。
- **响应式断点**：表单双列网格（`.form-grid-2col`）在 `640px` 以下自动折叠为单列；统计卡片四列网格在 `1200px` 折为双列，`640px` 折为单列。

## Elevation & Depth

本设计系统遵循**全面扁平化（Flat & Tonal Layering）**原则。界面不依赖弥散投影或浮夸的悬浮抬升来区分层级，而是纯粹依靠**三层背景色阶**（`#ffffff` 纯白主表面、`#f8fafc` 浅灰面板表面、`#f1f5f9` 内嵌凹槽表面）配合 **`1px solid #e2e8f0` 结构线**来建立清晰、克制的工业级空间秩序。

### Named Rules
**The Zero-Shadow Flat Rule.** 所有常规卡片、横向数据列表、统计模块与表单容器在默认与悬停状态下均保持扁平（无装饰性阴影、无悬浮位移），仅依靠边框色与背景色阶的变化反馈交互状态。
**The Tonal Row Selection Rule.** 在横向列表中标记当前选中行时，不使用外发光、浮动卡片或粗侧边条，而是采用整行 `#eef2ff` 浅靛蓝底色配合行末激活态勾选徽章（`配置中`）呈现纯净的色阶选中态。

## Shapes

- **容器与面板圆角**：横向列表外框、详情面板与内嵌命名空间配置框统一采用 `10px`–`12px` 的干净圆角，搭配 `1px solid #e2e8f0` 实线边框。
- **交互控件圆角**：所有按钮（高度 `36px`）、文本输入框（高度 `36px`）、下拉选择器与顶栏用户菜单触发器（高度 `30px`）统一使用干练的 `6px` 圆角。
- **胶囊与代码标签**：角色类别标签（`平台管理员`、`工作人员`、`只读角色`）与行状态指示器（`配置中` / `点击配置`）采用 `999px` 全圆角胶囊；K8s 资源代码标签采用 `5px` 微圆角。

## Components

所有组件均呈现**干练、直接、无冗余装饰**的工程操控感。

### Buttons
- **Shape:** `6px` 圆角，标准操作高度固定为 `36px`，内部水平内边距 `16px`，图标与文字间距 `7px`。
- **Primary:** 核心紫靛蓝背景（`#4f46e5`）配纯白文字（`#ffffff`），字号 `0.8125rem`，字重 `500`。
- **Hover / Focus:** 悬停时背景平滑切换为 `#4338ca`，聚焦时显示 `1px` 核心紫靛蓝外框，无位移变形。
- **Secondary & Ghost:** 次级按钮采用纯白背景配 `#cbd5e1` 边框与 `#1e293b` 文字；幽灵按钮（如顶栏「刷新」）在点击加载时触发图标 `360deg` 连续旋转动画（`.is-spinning-icon`）并切换文案为 `刷新中...`。

### Chips & Role Pills
- **Style:** 胶囊造型（`999px` 圆角），字号 `0.75rem`，字重 `600`，内边距 `2px 10px`。
- **State:** 依据角色语义区分底色——管理员与核心角色采用 `#eef2ff` 浅靛蓝底配 `#4338ca` 文字；行末状态指示器在未选中时为白底 `#cbd5e1` 边框，选中时边框变为 `#818cf8` 且文字转为 `#4f46e5` 并附带勾选图标。

### Cards / Containers
- **Corner Style:** `10px`–`12px` 圆角。
- **Background:** 主卡片与列表容器为 `#ffffff`；内嵌配置块（如 `.user-ns-config-box`）为 `#f8fafc`。
- **Shadow Strategy:** 全面扁平化，依靠 `1px solid #e2e8f0` 界定边界。
- **Internal Padding:** 详情配置面板为 `28px 32px`；内嵌配置子框为 `16px 18px`。

### Inputs / Fields
- **Style:** 高度 `36px`，纯白背景（`#ffffff`），`1px solid #cbd5e1` 实线边框，`6px` 圆角，字号 `0.875rem`。
- **Focus:** 边框切换为 `#6366f1` 并附带 `0 0 0 1px #6366f1` 清晰焦点环。
- **Code / YAML Editor (`.dsl-code-input`):** 采用 `#0f172a` 碳素石墨深色背景、`#38bdf8` 亮青色等宽代码字体与 `#334155` 边框。

### Navigation & View Switchers
- **Top Header User Menu (`.tkn--header-user-menu`):** 位于深色顶栏右侧，高 `30px` 的半透明扁平触发器（`rgba(255,255,255,0.08)`），点击展开包含角色徽章、「个人信息」弹窗入口与红色「退出登录」项的紧凑下拉菜单。
- **Segmented View Switcher (`.rbac-view-switcher`):** `#e2e8f0` 凹槽底框内嵌两个切换按钮（`角色配置` / `用户配置`），激活项以纯白背景（`#ffffff`）与 `#4f46e5` 粗体文字高亮。

### Horizontal Master-Detail List & Matrix (`.rbac-horizontal-list-table` & `.rbac-matrix-table`)
- **Horizontal List Table:** 位于视图上方，表头采用 `#f8fafc` 底色与 `0.78rem` 加粗文字；数据行高紧凑（单元格内边距 `12px 16px`），整行可点击切换下方详情面板；当前选中行以 `#eef2ff` 底色与左侧 `3px` 靛蓝内嵌边条明确指示。
- **5×5 Permission Matrix Table:** 位于角色详情面板下方，按行展示「流水线」「任务库」「执行记录」「触发器与事件」「构建凭据」5 大模块，按列提供 `查看`、`创建`、`编辑`、`删除`、`触发运行` 复选框及行末 `全权 / 只读` 快捷设置按钮。

## Do's and Don'ts

### Do:
- **Do** 保持全面扁平化的层级表达：使用 `#ffffff`、`#f8fafc`、`#f1f5f9` 三级背景色阶与 `1px solid #e2e8f0` 边框区分容器层级。
- **Do** 在配置管理页面（如 RBAC）坚持「上方横向列表 + 下方选中项详情面板」的联动范式，并用左侧 `3px` 内嵌靛蓝边条标记当前正在配置的行。
- **Do** 对所有 Kubernetes 原生资源标识符（`ClusterRole`、`Namespace`、`Secret`）使用等宽字体代码标签（`.row-code-tag`）。
- **Do** 让每一个异步刷新或保存按钮提供直接的状态反馈（如刷新图标旋转动画 + `刷新中...` / `保存中...` 文案）。

### Don't:
- **Don't** 在登录页、新建角色/用户弹窗或页面头部堆砌冗长的底层技术解释（如解释标签名称、`release.yaml` 映射原理或预置账号体验框）。
- **Don't** 在页面上展示「同步于 HH:mm:ss」等暗示本地数据库缓存同步的文案——所有 K8s 资源与 RBAC 状态均表现为集群实时直连。
- **Don't** 在命名空间视图或任何代理入口提供「创建命名空间」或「删除命名空间」按钮（命名空间在平台上严格只读）。
- **Don't** 为卡片或列表行添加浮夸的悬浮阴影或 `translateY` 跳动效果，保持工业级控制台的稳定与克制。
