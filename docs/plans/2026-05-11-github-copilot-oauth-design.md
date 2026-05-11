# GitHub Copilot 登录最小实现设计

日期：2026-05-11
项目：`/data/workspace/sub2api`

## 背景

目标是在不合并 `Wei-Shaw/sub2api` 的 `PR #771` 的前提下，为现有账号体系增加 GitHub Copilot 登录支持。`PR #771` 改动过大（`1443 files changed`、`344,327 deletions`），风险超出本次需求，因此采用最小自实现。

现有账号体系已可复用：`Account` 包含 `Platform`、`Type`、`Credentials`、`Extra` 等字段；平台常量集中在 `backend/internal/domain/constants.go`，service 层通过 `backend/internal/service/domain_constants.go` 复用；admin OAuth 路由已有 OpenAI、Gemini、Antigravity 模式可参考。

## 目标

1. 新增 `copilot` 平台常量。
2. 提供 GitHub Device OAuth 最小流程：
   - `POST /api/v1/admin/copilot/oauth/start`
   - `POST /api/v1/admin/copilot/oauth/poll`
3. `start` 返回 GitHub Device Flow 所需字段：`device_code`、`user_code`、`verification_uri`、`expires_in`、`interval`。
4. `poll` 用 `device_code` 换 GitHub access token，再调用 GitHub Copilot token exchange 验证 Copilot 可用。
5. `poll` 成功时返回可保存到账号 `Credentials` 的数据，最小包含 GitHub access token、短期 Copilot token、过期时间、GitHub 用户信息。
6. 不改数据库结构，不接入完整 Copilot 网关，不做前端大改。

## 非目标

- 不合并 `PR #771`。
- 不实现 Copilot API 转发、模型映射、额度统计或 token 自动刷新。
- 不引入新的迁移。
- 不要求本次创建账号，只返回可保存的 credentials payload。

## 方案

采用“后端最小 OAuth 服务 + admin handler + 路由注册”：

- `backend/internal/pkg/copilot`：封装 GitHub Device OAuth 与 Copilot token exchange HTTP 调用。
- `backend/internal/service/copilot_oauth_service.go`：编排流程、代理解析、返回 credentials payload。
- `backend/internal/handler/admin/copilot_oauth_handler.go`：暴露 admin API。
- `backend/internal/server/routes/admin.go`：注册 `/copilot/oauth/start` 和 `/copilot/oauth/poll`。
- DI：在 `service/wire.go`、`handler/wire.go`、`cmd/server/wire_gen.go` 中接入。

## 数据流

1. 管理端调用 `start`，可选传 `proxy_id`。
2. 服务端调用 `https://github.com/login/device/code` 获取 device code。
3. 用户在 GitHub 验证页输入 `user_code`。
4. 管理端按 `interval` 调用 `poll`，传 `device_code`。
5. 服务端调用 `https://github.com/login/oauth/access_token`。
6. 成功后调用 `https://api.github.com/user` 获取用户信息。
7. 再调用 `https://api.github.com/copilot_internal/v2/token` 验证 Copilot 权限并获取短期 Copilot token。
8. 返回 `credentials`，由管理端或后续账号接口保存。

## 错误处理

- `authorization_pending`、`slow_down` 返回 200 + `status=pending`，避免前端当作失败。
- `expired_token`、`access_denied`、Copilot 不可用返回 400。
- GitHub/Copilot 网络失败返回网关类错误。
- 代理 ID 不存在或代理 URL 异常直接返回明确错误，不静默直连回退。

## 测试

- service 单测：start、pending、slow_down、success credentials、Copilot exchange failure。
- handler 单测：参数校验、pending 响应、success 响应。
- route/contract 测试：确认 admin routes 包含 Copilot OAuth 端点，同时保留既有 `/dashboard/openai-token-stats`、`/antigravity/default-model-mapping`、`/internal/server/routes/gateway.go` 相关行为不被影响。

## 风险与取舍

- Device OAuth 本身不依赖 secret，适合最小接入。
- 返回 GitHub access token 属于敏感数据，handler 不写日志，不在错误中回显 token。
- 当前不做持久化会话，简化实现；由 `device_code` 直接驱动 poll。
- 不先做完整 Copilot 网关，避免把本次登录需求扩大成平台接入重构。
