# sub2api 半自动发布方案设计

日期：2026-05-11

## 目标

将当前 VPS 上 `xrouter.uk` 的 `sub2api` 部署，从“直接使用上游公共镜像 `weishaw/sub2api:latest`”改造成“基于 `yelog/sub2api` 仓库半自动发布”的方式。

目标发布流程：

1. 本地在 `/data/workspace/sub2api` 开发、测试、提交
2. 推送到 GitHub 仓库 `yelog/sub2api`
3. VPS 执行一次部署脚本
4. VPS 拉取 `origin/main` 最新代码并本地构建镜像
5. 使用 Docker Compose 更新 `sub2api` 服务
6. 通过健康检查验证发布结果

## 当前现状

当前生产环境运行方式如下：

- 域名：`xrouter.uk`
- 入口：Nginx
- 反代目标：`127.0.0.1:18080`
- 应用容器：`sub2api`
- 数据依赖：`sub2api-postgres`、`sub2api-redis`
- 当前应用镜像：`weishaw/sub2api:latest`
- 部署目录：`/opt/sub2api/deploy`

现状问题：

- 生产环境未接入 `yelog/sub2api`
- 本地修改和线上运行版本脱节
- 使用 `latest` 标签不利于追踪、回滚和稳定发布

## 方案对比

### 方案 A：继续使用上游公共镜像

做法：继续拉取 `weishaw/sub2api:latest`。

优点：
- 最省事

缺点：
- 本地 fork 的修改无法上线
- 无法准确控制发布内容
- 回滚能力弱

结论：不采用。

### 方案 B：VPS 拉取 fork 源码并本地构建（推荐）

做法：
- VPS 保留 git 工作区 `/opt/sub2api/repo`
- 部署脚本拉取 `yelog/sub2api` 最新代码
- 在 VPS 本地 `docker compose build`
- 再 `docker compose up -d`

优点：
- 与当前开发方式最匹配
- 易于理解和维护
- 便于按 commit 回滚
- 不依赖额外镜像仓库流程

缺点：
- VPS 本地构建会增加发布时间
- 服务器需要保留构建环境

结论：采用。

### 方案 C：先构建私有镜像，再由 VPS 拉取镜像发布

优点：
- 发布速度快
- 生产环境更干净

缺点：
- 需要额外镜像仓库和构建链路
- 当前阶段复杂度偏高

结论：后续可升级，不作为第一阶段方案。

## 最终设计

## 1. 目录结构

VPS 上使用如下结构：

- `/opt/sub2api/repo`：`yelog/sub2api` 的 git 工作区
- `/opt/sub2api/deploy`：运行配置、`.env`、compose 覆盖文件、数据目录
- `/opt/sub2api/scripts/deploy.sh`：标准部署脚本
- `/opt/sub2api/scripts/rollback.sh`：回滚脚本
- `/opt/sub2api/scripts/current-successful-release`：记录上次成功发布 commit

本地开发目录保持不变：

- `/data/workspace/sub2api`

## 2. 运行架构

保留现有生产链路：

- Nginx
- Docker Compose
- Postgres / Redis 数据卷
- `xrouter.uk -> Nginx -> 127.0.0.1:18080 -> sub2api container:8080`

只替换应用来源：

- 从 `image: weishaw/sub2api:latest`
- 改为 `build: /opt/sub2api/repo`

这样可以在不改变外部访问路径和不调整数据库的前提下，完成应用升级。

## 3. 发布流程

标准发布流程：

1. 在本地修改 `/data/workspace/sub2api`
2. 本地自测
3. `git commit`
4. `git push origin main`
5. 登录 VPS 并执行 `deploy.sh`

`deploy.sh` 逻辑：

1. `set -euo pipefail`
2. 进入 `/opt/sub2api/repo`
3. 记录当前 commit 作为候选回滚点
4. `git fetch origin`
5. `git reset --hard origin/main`
6. 进入 `/opt/sub2api/deploy`
7. `docker compose build sub2api`
8. `docker compose up -d sub2api`
9. 循环检查 `http://127.0.0.1:18080/health`
10. 成功后记录本次成功 commit
11. 失败时退出并提示回滚

## 4. 配置与数据策略

以下内容继续保留在 VPS：

- `/opt/sub2api/deploy/.env`
- Postgres 数据卷
- Redis 数据卷
- Nginx 站点配置
- 证书

原则：

- 代码跟随 git 更新
- 配置和数据留在服务器本地
- 不把生产密钥提交进仓库

这样可避免因为代码更新误伤业务数据。

## 5. 回滚设计

提供 `rollback.sh <commit>`。

回滚逻辑：

1. 进入 `/opt/sub2api/repo`
2. `git fetch origin`
3. `git reset --hard <commit>`
4. 重新构建 `sub2api`
5. `docker compose up -d sub2api`
6. 重新做健康检查

同时，`deploy.sh` 每次成功部署后写入：

- 当前成功 commit
- 可选记录部署时间

这样出现线上问题时，可以快速回到上一版。

## 6. 错误处理

部署失败判定包括：

- `git fetch/reset` 失败
- `docker compose build` 失败
- `docker compose up -d` 失败
- `/health` 在限定时间内未恢复正常

处理原则：

- 任一步失败立即终止
- 不自动做高风险回滚
- 明确打印当前失败阶段
- 保留手动执行 `rollback.sh` 的入口

原因：第一阶段优先保证可控和可观察，不做过度自动化。

## 7. 测试与验证

最小验证要求：

1. 本地仓库存在可推送提交
2. VPS 能成功拉到 `yelog/sub2api`
3. `docker compose build sub2api` 成功
4. 新容器成功启动
5. `http://127.0.0.1:18080/health` 返回成功
6. `https://xrouter.uk` 外部访问正常

建议增加的验证：

- 检查 `docker ps` 状态
- 检查容器日志是否有迁移/配置错误
- 检查关键 API 是否能正常返回

## 8. 第一阶段范围

本阶段包含：

- 建立 VPS git 工作区
- 调整 Compose 使其从本地源码构建
- 增加 `deploy.sh`
- 增加 `rollback.sh`
- 保留现有 Nginx、数据库、Redis 和域名入口

本阶段不包含：

- GitHub Actions 自动部署
- 蓝绿发布
- 私有镜像仓库
- 自动数据库迁移编排优化
- 多环境发布体系

## 9. 后续升级路径

当第一阶段稳定后，可继续演进到：

1. GitHub Actions 构建镜像
2. 推送到私有/公开镜像仓库
3. VPS 仅做 `pull + up -d`
4. 再进一步做发布审计、版本标签和更快回滚

## 结论

推荐采用：

- `yelog/sub2api` 作为唯一代码来源
- VPS 保留本地工作区并本地构建镜像
- 使用部署脚本进行半自动发布
- 保留现有数据和反代链路
- 增加明确的健康检查和回滚能力

这是当前复杂度、稳定性和可维护性之间最平衡的方案。
