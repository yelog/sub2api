# GitHub Copilot 独立平台设计

## 目标

将 GitHub Copilot 作为 `copilot` 独立平台接入后台“帐号管理 → 添加帐号”流程，支持管理员通过设备码 OAuth 创建设备账号，并在账号列表中展示 Premium requests 的 `已用 / 总量 (百分比)`。

## 背景

当前后端已经具备 Copilot OAuth 基础能力：

- `POST /api/v1/admin/copilot/oauth/start`
- `POST /api/v1/admin/copilot/oauth/poll`
- 平台常量 `PlatformCopilot = "copilot"`

但前端帐号管理仍只支持 `anthropic | openai | gemini | antigravity`，因此管理员无法在“添加帐号”入口中选择 GitHub Copilot；账号列表和用量展示链路也没有为 Copilot 预留独立分支。

## 设计原则

1. **独立平台，不挂靠 OpenAI**：鉴权、账号展示、用量拉取和错误处理都以 `copilot` 为独立边界。
2. **最小可上线闭环**：第一版聚焦后台添加账号、重授权、账号展示和 Premium requests 展示，不扩展调度、告警和自动封禁。
3. **复用外壳，独立语义**：复用现有 OAuth 弹窗和账号列表布局，但 Copilot 的状态、凭据和 usage 数据结构单独处理。
4. **优先稳定展示**：列表页不强依赖实时拉取，支持服务端缓存与降级显示。

## 用户流程

### 1. 添加帐号

1. 管理员进入“帐号管理”
2. 点击“添加帐号”
3. 平台列表出现 `GitHub Copilot`
4. 选择后进入 Copilot 专属 OAuth 设备码流程
5. 前端调用 `/admin/copilot/oauth/start` 获取：
   - `device_code`
   - `user_code`
   - `verification_uri`
   - `expires_in`
   - `interval`
6. 管理员在 GitHub 页面完成授权
7. 前端轮询 `/admin/copilot/oauth/poll`
8. 成功后创建账号：
   - `platform = copilot`
   - `type = oauth`
   - `credentials` 内含 GitHub token / Copilot token / GitHub 用户信息

### 2. 查看帐号

在账号列表中：

- 平台徽章显示 `GitHub Copilot`
- 类型显示 `OAuth`
- 使用量列显示 Premium requests：`已用 / 总量 (百分比)`
- 若无法获取 usage，则显示 `-` 或明确错误状态

### 3. 重授权

已创建的 Copilot OAuth 账号，可通过和 OpenAI/Gemini/Antigravity 一致的“重新授权”入口重新走设备码流程。

## 架构设计

### 前端

#### 平台枚举与视觉层

需要补齐以下前端基础能力：

- `AccountPlatform` 扩展为包含 `copilot`
- `platformColors.ts` 增加 Copilot 色板和 label
- `PlatformTypeBadge.vue`、`PlatformIcon.vue` 等平台展示组件增加 Copilot
- i18n 中增加 Copilot 平台名称、OAuth 文案、usage 文案

#### 添加帐号流程

在 `CreateAccountModal.vue` 中：

- 平台选择区新增 Copilot
- 选择 Copilot 时进入 `oauth-based` 流程
- 使用专属 OAuth 分支，而不是复用 OpenAI 的 token/exchange 语义
- 成功后以 `platform = copilot` 创建账号

为了降低侵入性，优先在现有 `OAuthAuthorizationFlow.vue` 中增加 `copilot` 分支，而不是额外拆新组件。该分支只支持设备码授权，不提供手输 refresh token / session token 等 OpenAI 专属模式。

#### 账号列表 usage 展示

`AccountUsageCell.vue` 增加 `copilot` 分支：

- 读取后端返回的 Copilot usage 结构
- 展示 `used / limit (ratio%)`
- 支持 loading / empty / error 三态
- 第一版不做复杂窗口图，只展示 Premium requests 摘要

### 后端

#### 账号创建与重授权

后端需要确保 Copilot OAuth 返回结果能稳定进入账号创建链路：

- `platform = copilot`
- `type = oauth`
- `credentials` 保存：
  - `github_access_token`
  - `copilot_token`
  - `copilot_token_expires_at`
  - `copilot_token_refresh_at`
  - `github_login`
  - `github_name`
  - `github_id`
  - `github_avatar_url`

如当前账号创建 DTO/校验逻辑对平台枚举有限制，需要同步补齐 `copilot`。

#### Premium requests 获取

需要新增 Copilot usage 服务，职责如下：

1. 使用账号中的 Copilot 凭据请求 Copilot usage 接口
2. 解析 Premium requests 已用量和总量
3. 归一化为统一结构返回给前端
4. 支持刷新 token 或在 token 过期时进行失败降级

建议返回结构类似：

```json
{
  "copilot_usage": {
    "premium_requests": {
      "used": 185,
      "limit": 500,
      "ratio": 0.37,
      "display": "185 / 500 (37%)",
      "fetched_at": "2026-05-11T23:59:00+08:00"
    }
  }
}
```

#### 缓存策略

第一版建议采用“读时刷新 + 短缓存”策略：

- 列表页或主动刷新时请求后端 usage 接口
- 服务端可把最近一次成功结果放入 `extra` 或统一 usage 响应缓存
- 缓存 TTL 建议 1~5 分钟

这样可以避免每次渲染表格都直接打上游，同时保持可见的新鲜度。

## 错误处理

### OAuth 相关

- 设备码过期：提示重新发起授权
- access denied：明确提示 GitHub 授权被拒绝
- Copilot token 兑换失败：提示 GitHub 已授权但 Copilot 权限不可用

### Usage 相关

- 上游 usage 接口失败：显示 `-`，并保留错误提示用于 hover/title 或日志
- 无 Premium requests 信息：视为未知，不伪造 0/0
- token 失效：标记需要重授权

## 测试策略

### 后端

1. Copilot usage 解析单测
2. Copilot usage 服务单测（成功 / token 过期 / 上游异常 / 无数据）
3. 管理端 handler 单测
4. 账号创建链路含 `platform = copilot` 的集成测试

### 前端

1. `AccountPlatform`/平台 badge/平台色板单测
2. 添加帐号弹窗中 Copilot 平台可见性和流程分支测试
3. Copilot OAuth 轮询完成后的账号创建 payload 测试
4. `AccountUsageCell` 中 Copilot usage 的展示测试
5. 重授权弹窗 Copilot 分支测试

## 非目标

第一版不包含：

- 基于 Premium requests 比例的调度策略
- Premium requests 告警/阈值通知
- 用户前台自助接入 Copilot
- 将 Copilot 视为 OpenAI 子类型

## 推荐实施顺序

1. 先补平台枚举、UI 和文案
2. 接通添加帐号与重授权的 Copilot OAuth 流程
3. 补齐后端 Copilot usage 拉取与返回
4. 在账号列表展示 Premium requests 摘要
5. 补测试、提交、发布
