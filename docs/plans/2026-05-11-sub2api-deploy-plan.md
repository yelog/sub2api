# Sub2API Semi-Automated VPS Deployment Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Switch production `xrouter.uk` from the upstream `weishaw/sub2api:latest` image to a semi-automated deployment flow that builds from `yelog/sub2api` on the VPS.

**Architecture:** Keep the current Nginx + Docker Compose + Postgres + Redis topology unchanged. Replace the application image source with a VPS-local git checkout at `/opt/sub2api/repo`, add deployment/rollback scripts under `/opt/sub2api/scripts`, and keep production secrets/data under `/opt/sub2api/deploy`.

**Tech Stack:** Git, Docker Compose, Bash, Nginx, PostgreSQL, Redis, Sub2API deploy files

---

### Task 1: Inspect current repo deployment inputs

**Files:**
- Review: `deploy/docker-compose.local.yml`
- Review: `deploy/.env.example`
- Review: `Dockerfile`
- Review: `deploy/README.md`

**Step 1: Confirm current app service uses upstream image**

Run:
```bash
cd /data/workspace/sub2api
grep -n "image:\|build:" deploy/docker-compose.local.yml
```
Expected: `sub2api` uses `image: weishaw/sub2api:latest` and has no `build:` entry.

**Step 2: Confirm Docker build context is available in repo root**

Run:
```bash
cd /data/workspace/sub2api
sed -n '1,220p' Dockerfile
```
Expected: root `Dockerfile` is usable for building the application image.

**Step 3: Commit notes only if you had to change nothing**

No commit in this task.

### Task 2: Add a VPS-oriented compose override file

**Files:**
- Create: `deploy/docker-compose.vps.yml`
- Reference: `deploy/docker-compose.local.yml`

**Step 1: Write the failing diff expectation**

Define the intended behavior in the new file:
- extend/replace the `sub2api` service image source
- use `build:` with context `../repo`
- optionally set a deterministic image name like `yelog/sub2api:local`
- keep existing ports, env file, volumes, postgres, redis behavior inherited from `docker-compose.local.yml`

**Step 2: Create minimal override file**

Example content target:
```yaml
services:
  sub2api:
    image: yelog/sub2api:local
    build:
      context: ../repo
      dockerfile: Dockerfile
```

**Step 3: Validate compose merge locally**

Run:
```bash
cd /data/workspace/sub2api/deploy
docker compose -f docker-compose.local.yml -f docker-compose.vps.yml config > /tmp/sub2api-compose-rendered.yml
sed -n '1,120p' /tmp/sub2api-compose-rendered.yml
```
Expected: rendered config shows `sub2api` with `build:` from `../repo` and no broken references.

**Step 4: Commit**

```bash
cd /data/workspace/sub2api
git add deploy/docker-compose.vps.yml
git commit -m "chore: add vps compose override for local builds"
```

### Task 3: Add deploy script

**Files:**
- Create: `deploy/scripts/deploy.sh`
- Create: `deploy/scripts/lib.sh` (optional helper; only if it keeps script simpler)
- Reference: `deploy/docker-compose.local.yml`
- Reference: `deploy/docker-compose.vps.yml`

**Step 1: Write the expected flow before coding**

The script must:
1. require repo path `/opt/sub2api/repo`
2. require deploy path `/opt/sub2api/deploy`
3. record current HEAD before update
4. fetch `origin`
5. hard reset to `origin/main`
6. render/build with compose
7. restart only `sub2api`
8. poll `http://127.0.0.1:18080/health`
9. persist successful commit metadata

**Step 2: Write minimal script implementation**

Required behaviors:
- `set -euo pipefail`
- variables for `REPO_DIR`, `DEPLOY_DIR`, `COMPOSE_FILES`, `HEALTH_URL`
- `git rev-parse HEAD` before and after reset
- `docker compose -f docker-compose.local.yml -f docker-compose.vps.yml build sub2api`
- `docker compose -f docker-compose.local.yml -f docker-compose.vps.yml up -d sub2api`
- health polling loop with timeout
- write successful commit to a state file such as `/opt/sub2api/scripts/current-successful-release`

**Step 3: Validate shell syntax**

Run:
```bash
cd /data/workspace/sub2api
bash -n deploy/scripts/deploy.sh
```
Expected: no output, exit code 0.

**Step 4: Commit**

```bash
cd /data/workspace/sub2api
git add deploy/scripts/deploy.sh
git commit -m "feat: add semi-automated deploy script"
```

### Task 4: Add rollback script

**Files:**
- Create: `deploy/scripts/rollback.sh`
- Reference: `deploy/scripts/deploy.sh`

**Step 1: Define rollback contract**

Expected input:
- explicit commit sha argument, or
- fallback to last successful commit file if you decide to support it

Expected behavior:
- reset repo to target commit
- rebuild `sub2api`
- restart `sub2api`
- run same health check

**Step 2: Implement minimal rollback script**

Required behaviors:
- `set -euo pipefail`
- usage help when no commit is supplied (unless supporting default file fallback)
- same compose command pair as deploy script
- same health polling logic

**Step 3: Validate shell syntax**

Run:
```bash
cd /data/workspace/sub2api
bash -n deploy/scripts/rollback.sh
```
Expected: no output, exit code 0.

**Step 4: Commit**

```bash
cd /data/workspace/sub2api
git add deploy/scripts/rollback.sh
git commit -m "feat: add rollback script for vps releases"
```

### Task 5: Add deployment documentation for the VPS workflow

**Files:**
- Create: `docs/plans/2026-05-11-sub2api-deploy-ops-notes.md` (optional, if you want operator notes separate)
- Or Modify: `deploy/README.md`
- Or Modify: `deploy/DOCKER.md`

**Step 1: Document only what operators need**

Document:
- required VPS directories
- first-time clone command for `/opt/sub2api/repo`
- where `.env` lives
- exact deploy command
- exact rollback command
- quick verification commands

**Step 2: Update the docs minimally**

Keep it short and operational.

**Step 3: Verify docs mention actual paths**

Run:
```bash
cd /data/workspace/sub2api
grep -R "/opt/sub2api/repo\|/opt/sub2api/deploy\|deploy.sh\|rollback.sh" deploy docs | sed -n '1,120p'
```
Expected: operator-facing docs mention the final production paths and commands.

**Step 4: Commit**

```bash
cd /data/workspace/sub2api
git add deploy/README.md deploy/DOCKER.md docs/plans/2026-05-11-sub2api-deploy-ops-notes.md 2>/dev/null || true
git commit -m "docs: add vps deployment workflow notes"
```

### Task 6: Create first-time VPS bootstrap commands

**Files:**
- Create: `deploy/scripts/bootstrap-vps.sh`
- Reference: `deploy/scripts/deploy.sh`
- Reference: `deploy/scripts/rollback.sh`

**Step 1: Define bootstrap scope**

The script should prepare only:
- `/opt/sub2api/repo`
- `/opt/sub2api/scripts`
- copy scripts from repo to VPS deployment location, or print the commands to do so
- optional initial clone of `https://github.com/yelog/sub2api.git`

It must not overwrite the existing `.env`, postgres data, redis data, or Nginx config.

**Step 2: Implement minimal bootstrap**

Suggested actions:
```bash
mkdir -p /opt/sub2api/repo /opt/sub2api/scripts
if [ ! -d /opt/sub2api/repo/.git ]; then
  git clone https://github.com/yelog/sub2api.git /opt/sub2api/repo
fi
install -m 755 deploy/scripts/deploy.sh /opt/sub2api/scripts/deploy.sh
install -m 755 deploy/scripts/rollback.sh /opt/sub2api/scripts/rollback.sh
```

**Step 3: Validate shell syntax**

Run:
```bash
cd /data/workspace/sub2api
bash -n deploy/scripts/bootstrap-vps.sh
```
Expected: no output, exit code 0.

**Step 4: Commit**

```bash
cd /data/workspace/sub2api
git add deploy/scripts/bootstrap-vps.sh
git commit -m "chore: add vps bootstrap script"
```

### Task 7: Verify compose and scripts end-to-end locally

**Files:**
- Test: `deploy/docker-compose.vps.yml`
- Test: `deploy/scripts/deploy.sh`
- Test: `deploy/scripts/rollback.sh`
- Test: `deploy/scripts/bootstrap-vps.sh`

**Step 1: Run compose config validation**

Run:
```bash
cd /data/workspace/sub2api/deploy
docker compose -f docker-compose.local.yml -f docker-compose.vps.yml config >/tmp/sub2api-vps-compose.yml
```
Expected: exit code 0.

**Step 2: Run shell syntax validation**

Run:
```bash
cd /data/workspace/sub2api
bash -n deploy/scripts/deploy.sh
bash -n deploy/scripts/rollback.sh
bash -n deploy/scripts/bootstrap-vps.sh
```
Expected: all exit 0.

**Step 3: Run a dry inspection of the deploy script**

If you implemented a dry-run flag, run:
```bash
cd /data/workspace/sub2api
bash deploy/scripts/deploy.sh --dry-run
```
Expected: prints intended commands without mutating VPS state.

If you did not implement dry-run, skip this step.

**Step 4: Commit**

```bash
cd /data/workspace/sub2api
git add deploy/docker-compose.vps.yml deploy/scripts
git commit -m "test: validate vps deployment assets"
```

### Task 8: Apply the first-time production switchover on VPS

**Files:**
- Use: `/opt/sub2api/repo`
- Use: `/opt/sub2api/deploy/.env`
- Use: `/opt/sub2api/scripts/deploy.sh`

**Step 1: Sync repo to VPS**

Run on VPS:
```bash
mkdir -p /opt/sub2api
if [ ! -d /opt/sub2api/repo/.git ]; then
  git clone https://github.com/yelog/sub2api.git /opt/sub2api/repo
else
  cd /opt/sub2api/repo && git fetch origin && git reset --hard origin/main
fi
```
Expected: VPS has the latest `yelog/sub2api` checkout.

**Step 2: Preserve existing deploy assets**

Run on VPS:
```bash
mkdir -p /opt/sub2api/scripts
cp /opt/sub2api/repo/deploy/scripts/deploy.sh /opt/sub2api/scripts/deploy.sh
cp /opt/sub2api/repo/deploy/scripts/rollback.sh /opt/sub2api/scripts/rollback.sh
chmod +x /opt/sub2api/scripts/*.sh
```
Expected: scripts installed without touching `.env` or data directories.

**Step 3: Run first deployment**

Run on VPS:
```bash
/opt/sub2api/scripts/deploy.sh
```
Expected: repo updates, image builds, `sub2api` restarts, `/health` becomes healthy.

**Step 4: Verify live service**

Run on VPS:
```bash
docker ps --format 'table {{.Names}}\t{{.Status}}\t{{.Image}}'
curl -fsS http://127.0.0.1:18080/health
curl -I https://xrouter.uk
```
Expected:
- `sub2api` is up and healthy
- local health endpoint returns success
- public HTTPS responds normally

**Step 5: Commit**

No repo commit in this task; this is an operational rollout.

### Task 9: Record rollback and release operations for future use

**Files:**
- Modify: `deploy/README.md` or `deploy/DOCKER.md`
- Optional Create: `docs/plans/2026-05-11-sub2api-release-runbook.md`

**Step 1: Add the exact routine commands**

Document:
```bash
cd /data/workspace/sub2api
git push origin main

ssh root@<vps>
/opt/sub2api/scripts/deploy.sh
```

And rollback:
```bash
ssh root@<vps>
/opt/sub2api/scripts/rollback.sh <commit>
```

**Step 2: Verify commands are copy-pastable**

Manually read the doc and ensure there are no placeholders left except the VPS hostname if intentional.

**Step 3: Commit**

```bash
cd /data/workspace/sub2api
git add deploy/README.md deploy/DOCKER.md docs/plans/2026-05-11-sub2api-release-runbook.md 2>/dev/null || true
git commit -m "docs: add release runbook for semi-automated deploys"
```
