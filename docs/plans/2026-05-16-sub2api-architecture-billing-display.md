# Sub2API 架构识别与余额/倍率展示计划

日期：2026-05-16
状态：Implemented

## 背景

当前后台账号管理已支持多平台、多账号类型、多分组、多倍率和余额/订阅两种计费模式，但 UI 对“一个账号/请求为什么按这个倍率扣费、余额和配额分别按什么口径消耗”的解释不足。管理员在排查用户扣费、账号配额、分组倍率、用户专属倍率时，需要跨多个页面和日志推断。

本计划目标是在不改变现有计费语义的前提下，补充后端结构化识别结果与前端可解释展示，让管理员能在账号管理界面快速理解：

- 账号添加后进入哪个平台/类型/分组链路。
- 当前账号的账号倍率是多少。
- 用户侧扣费倍率来自系统默认、分组默认还是用户专属覆盖。
- 余额模式和订阅模式分别消耗什么字段。
- 账号配额、API Key 配额、余额/订阅额度分别按哪个 cost 口径消耗。

## 已确认的当前架构

### 添加账号链路

前端入口：

- `frontend/src/components/account/CreateAccountModal.vue`
- `frontend/src/api/admin/accounts.ts`
- `frontend/src/types/index.ts`

核心表单字段：

- `name`
- `notes`
- `platform`
- `type`
- `credentials`
- `extra`
- `proxy_id`
- `concurrency`
- `load_factor`
- `priority`
- `rate_multiplier`
- `group_ids`
- `expires_at`
- `auto_pause_on_expired`

前端创建分支：

- OAuth：先进入授权步骤，换取 token 后创建。
- Bedrock：构建 AWS/Bedrock credentials 后直接创建。
- Antigravity upstream：用 `base_url + api_key` 创建。
- Gemini/Anthropic service account：用 service account json 创建。
- 普通 API key：用 `base_url + api_key` 创建。

后端路由：

- `backend/internal/server/routes/admin.go`
- `POST /api/v1/admin/accounts` → `h.Admin.Account.Create`

后端服务：

- `backend/internal/handler/admin/account_handler.go`
- `backend/internal/service/admin_service.go`

创建逻辑要点：

1. 无 `group_ids` 时自动绑定 `${platform}-default` 分组。
2. 有分组绑定时执行 mixed channel 风险检查，除非用户显式确认跳过。
3. 创建 `service.Account`，默认 `Status=active`、`Schedulable=true`。
4. `rate_multiplier` 允许为 `0`，不允许负数。
5. `load_factor` 仅接受 `> 0` 且 `<= 10000`。
6. OAuth 账号创建后对 OpenAI/Antigravity 异步设置隐私。

### 倍率来源

#### 系统默认倍率

来源：`cfg.Default.RateMultiplier`。

#### 分组倍率

来源：`apiKey.Group.RateMultiplier`。

#### 用户专属分组倍率

来源：`user_group_rate_multipliers.rate_multiplier`。

仓储：

- `backend/internal/repository/user_group_rate_repo.go`

查询：

- `GetByUserAndGroup(ctx, userID, groupID)`

Gateway 解析顺序：

```text
用户专属分组倍率 > 分组默认倍率 > 系统默认倍率
```

相关文件：

- `backend/internal/service/gateway_service.go`

#### 账号倍率

来源：`Account.RateMultiplier`。

相关文件：

- `backend/internal/service/account.go`

语义：

- `nil`：按 `1.0`。
- `0`：合法，账号口径计费为 0。
- `<0`：非法数据，安全回退为 `1.0`。

账号倍率主要影响账号口径统计和账号配额，用户余额/订阅扣费使用的是用户侧倍率后的 `ActualCost`。

### Cost 语义

核心字段：

- `TotalCost`：标准费用，倍率前。
- `ActualCost`：用户/API Key 口径费用，受分组倍率或用户专属倍率影响。
- `AccountQuotaCost`：账号配额口径，使用 `TotalCost * AccountRateMultiplier`。

扣费命令：

- `backend/internal/service/usage_billing.go`
- `UsageBillingCommand`

字段：

- `BalanceCost`
- `SubscriptionCost`
- `APIKeyQuotaCost`
- `APIKeyRateLimitCost`
- `AccountQuotaCost`

规则：

- 余额模式：`BalanceCost = ActualCost`。
- 订阅模式：`SubscriptionCost = ActualCost`。
- API Key quota/rate limit：使用 `ActualCost`。
- 账号 quota：使用 `TotalCost * AccountRateMultiplier`。

## 需求分析

### 用户故事

作为管理员，我希望在账号管理页面看到某个账号的计费架构摘要，这样我能快速判断它的余额扣费、订阅扣费、账号配额和倍率来源。

作为管理员，我希望在排查用户/API Key 请求时能看到有效倍率来源，这样我能解释为什么某个用户按特定倍率扣费。

作为管理员，我希望添加/编辑账号时已有的账号倍率、分组绑定、配额设置能被明确说明，避免误以为账号倍率会影响用户余额。

### 非目标

本阶段不改变现有计费规则：

- 不改变 `ActualCost` 计算方式。
- 不改变余额/订阅扣费逻辑。
- 不改变账号配额逻辑。
- 不改变默认分组绑定逻辑。
- 不改变用户专属倍率配置入口。

## UI/UX/UE 设计

### 页面位置

优先选择账号管理列表/详情的轻量展示：

1. 账号列表增加“计费架构”入口或 tooltip。
2. 账号详情/展开面板增加 `BillingInfo` 信息块。
3. 创建/编辑弹窗中在账号倍率字段旁增加说明：
   - “账号倍率影响账号口径统计/账号配额，不直接影响用户余额；用户余额按分组/用户专属倍率后的 ActualCost 扣减。”

### 推荐组件

新增组件：

- `frontend/src/components/account/AccountBillingInfo.vue`

职责：

- 展示账号倍率。
- 展示分组倍率摘要。
- 展示用户侧倍率来源说明。
- 展示余额/订阅/账号配额/API Key 配额的 cost 口径。

现有可复用位置：

- `AccountQuotaInfo.vue`
- `AccountStatusIndicator.vue`
- `AccountCapacityCell.vue`

### 信息架构

建议展示字段：

- 账号倍率：`account_rate_multiplier`
- 用户侧默认倍率：系统默认 or 分组默认
- 用户专属倍率：是否存在覆盖
- 余额扣费：`ActualCost`
- 订阅扣费：`ActualCost`
- API Key 配额：`ActualCost`
- 账号配额：`TotalCost × AccountRateMultiplier`
- 计费模式：余额 / 订阅 / 简单模式不扣费

## 后端接口设计

### 方案 A：扩展账号详情/列表响应

优点：前端改动少。

风险：账号列表可能变重，涉及分组和用户专属倍率时需要避免 N+1。

建议：列表只返回账号倍率和静态 cost 语义；详情接口返回完整解释。

### 方案 B：新增账号计费架构接口

推荐接口：

```text
GET /api/v1/admin/accounts/:id/billing-architecture
```

响应示例：

```json
{
  "account_id": 123,
  "platform": "anthropic",
  "type": "oauth",
  "account_rate_multiplier": 1.0,
  "effective_account_rate_multiplier": 1.0,
  "groups": [
    {
      "id": 1,
      "name": "anthropic-default",
      "rate_multiplier": 1.0,
      "subscription_type": "balance"
    }
  ],
  "cost_semantics": {
    "total_cost": "standard_cost_before_user_multiplier",
    "actual_cost": "cost_after_group_or_user_group_multiplier",
    "balance_cost": "actual_cost",
    "subscription_cost": "actual_cost",
    "api_key_quota_cost": "actual_cost",
    "api_key_rate_limit_cost": "actual_cost",
    "account_quota_cost": "total_cost * account_rate_multiplier"
  }
}
```

后续如果要解释“某个用户”的有效倍率，可扩展 query：

```text
GET /api/v1/admin/accounts/:id/billing-architecture?user_id=123&group_id=456
```

返回：

```json
{
  "effective_user_rate_multiplier": 1.2,
  "user_rate_source": "user_group_override"
}
```

### 服务层建议

新增 service 方法：

```go
GetAccountBillingArchitecture(ctx context.Context, accountID int64, opts BillingArchitectureOptions) (*AccountBillingArchitecture, error)
```

需要读取：

- account repo：账号、分组绑定。
- group repo：分组倍率、订阅类型。
- user group rate repo：可选，按 user_id/group_id 获取专属倍率。
- config：系统默认倍率。

## 测试计划（TDD）

### 后端测试

先写失败测试：

1. `TestGetAccountBillingArchitecture_AccountRateMultiplierNilDefaultsToOne`
2. `TestGetAccountBillingArchitecture_AccountRateMultiplierZeroAllowed`
3. `TestGetAccountBillingArchitecture_UserGroupOverrideWins`
4. `TestGetAccountBillingArchitecture_GroupMultiplierFallback`
5. `TestGetAccountBillingArchitecture_CostSemanticsStable`
6. `TestBuildUsageBillingCommand_BalanceUsesActualCost`
7. `TestBuildUsageBillingCommand_SubscriptionUsesActualCost`
8. `TestBuildUsageBillingCommand_AccountQuotaUsesTotalCostTimesAccountMultiplier`

已有相关测试可参考：

- `backend/internal/service/account_billing_rate_multiplier_test.go`
- `backend/internal/service/gateway_service_subscription_billing_test.go`
- `backend/internal/service/billing_service_rate_multiplier_test.go`
- `backend/internal/service/gateway_hotpath_optimization_test.go`

### 前端测试

建议覆盖：

1. `AccountBillingInfo` 渲染账号倍率。
2. `AccountBillingInfo` 渲染余额/订阅/API Key/账号配额 cost 语义。
3. `AccountBillingInfo` 对用户专属倍率覆盖显示来源。
4. API 类型能解析后端响应。
5. 创建/编辑账号倍率说明文案存在。

## 实施步骤

1. 后端新增 DTO / service 方法 / handler / route。
2. 后端补测试，先 RED，再实现到 GREEN。
3. 前端新增 API 方法与类型。
4. 新增 `AccountBillingInfo.vue`。
5. 在账号详情或展开面板接入组件。
6. 在创建/编辑弹窗账号倍率字段旁补充说明。
7. 跑后端测试。
8. 跑前端测试和构建。
9. 提交到 main。
10. push main。
11. 按固定 VPS 流程构建镜像 tar、上传、docker load、compose up、健康检查。

## 验收标准

- 管理员能在账号 UI 中看到账号倍率和 cost 口径说明。
- 不改变现有扣费、配额、订阅行为。
- 后端接口返回稳定、无敏感凭据。
- 前端不展示 credentials/token/api_key/password 等敏感字段。
- 后端相关测试通过。
- 前端测试和构建通过。
- 发布后 `/health` 正常。

## 风险与注意事项

- 账号列表不能引入明显 N+1 查询。
- 不要在接口中返回 `credentials` 内的敏感字段。
- 用户专属倍率是用户+分组维度；账号本身没有单一“用户有效倍率”，必须在传入 user_id/group_id 时才可精确解释。
- 账号倍率和用户扣费倍率语义不同，UI 文案必须明确区分。
- 简单模式 `RunModeSimple` 下记录 usage 但不扣费，UI 需要避免误导。
