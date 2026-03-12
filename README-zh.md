# 🦞 claw-mesh

[English](README.md)

> Your personal AI mesh — 一个 AI，所有设备。

你的 AI 助手不应该被困在一台机器上。Mac 有 Xcode，Linux 有 GPU，VPS 有公网 IP。claw-mesh 把它们连成一个统一的能力平面——消息自动路由到对的设备，任务跨机器执行，节点挂了流量自动切换。

基于 [OpenClaw](https://github.com/openclaw/openclaw) 构建。单二进制。新节点零配置。

## 它做什么

```
你: "构建 iOS 应用，然后用 Docker 部署后端"
                              │
                    ┌─────────▼──────────┐
                    │   claw-mesh coord   │
                    │                     │
                    │  1. LLM planner     │
                    │     拆解为 2 步      │
                    │                     │
                    │  2. Skill 感知路由   │
                    └──────┬──────┬───────┘
                           │      │
              Step 1       │      │       Step 2
              xcode ───────┘      └────── docker
              ▼                                ▼
     ┌─────────────────┐             ┌─────────────────┐
     │  Mac Mini        │             │  Linux GPU       │
     │  xcode, cocoapods│             │  docker, k8s     │
     │  ios-build skill │             │  python, rust    │
     └─────────────────┘             └──────────────────┘
```

- **Skill 感知路由** — 每个节点上报自己的能力（工具、agent skill、自定义能力），消息自动路由到拥有所需 skill 的节点。
- **跨节点任务规划** — 复杂请求由 LLM planner 自动拆解为多步计划，每步分发到最合适的节点。
- **故障转移** — 节点宕机？任务自动切到拥有相同 skill 的其他节点。
- **Config Seed** — 新节点从 coordinator 拉取 AI 提供商配置和身份文件。一条 `join --auto-install` 即可就绪。
- **Web 控制台** — 实时节点状态、skill 地图、路由规则、消息流。

## 快速开始

```bash
# 安装
go install github.com/SallyKAN/claw-mesh/cmd/claw-mesh@latest

# 或从源码构建
git clone https://github.com/SallyKAN/claw-mesh.git
cd claw-mesh && make build

# 初始化配置（生成 claw-mesh.yaml 和 token）
./bin/claw-mesh init

# 启动 coordinator
./bin/claw-mesh up --port 9180 --token mysecret --allow-private

# 从另一台机器加入
./bin/claw-mesh join http://<coordinator-ip>:9180 \
  --name mac-mini --tags xcode,local \
  --token mysecret --auto-install
```

打开 `http://localhost:9180` 查看控制台。

## 工作原理

### 节点与能力

每台运行 OpenClaw Gateway 的机器就是一个节点。加入时自动检测能力：

| 检测项 | 示例 |
|--------|------|
| OS 与架构 | `darwin/arm64`, `linux/amd64` |
| 硬件 | GPU、内存 |
| Tool skills | `docker`, `xcode`, `python`, `kubectl` — 从 PATH 自动检测 |
| Agent skills | `.claude/skills/*.md` — 解析 `requires` 字段（OS、工具、标签）判断各节点可执行性 |
| Custom skills | 在 `skills.yaml` 中声明 — 如 `stable-diffusion`, `data-pipeline` |

```bash
$ claw-mesh skills
SKILL              TYPE          CATEGORY    NODES
docker             tool          -           linux-gpu
golang             tool          -           mac-mini, linux-gpu
ios-build          agent-skill   -           mac-mini
python             tool          -           linux-gpu, mac-mini, pi-home
sd-xl              custom        image-gen   linux-gpu
sensor-reader      agent-skill   iot         pi-home
```

### 路由

消息通过规则匹配路由，所有条件为 AND 关系：

```yaml
routing_rules:
  # GPU 任务 → Linux
  - match: { requires_gpu: true }
    target: linux-gpu

  # 同时需要 docker 和 python
  - match: { requires_skills: [docker, python] }
    strategy: least-busy

  # 需要 xcode 或 docker 之一
  - match: { requires_any_skill: [xcode, docker] }
    strategy: least-busy

  # 默认
  - match: { wildcard: true }
    strategy: least-busy
```

### 任务规划

配置 LLM planner 后，复杂请求自动拆解：

```bash
$ claw-mesh send --auto "在 GPU 上训练模型，然后部署到 K8s"
Plan plan-x7y8z9 created (2 steps), executing...

$ claw-mesh plan status plan-x7y8z9
Plan: plan-x7y8z9 [completed]
  Step 1: python → linux-gpu ✓ (180s)
  Step 2: k8s → linux-gpu ✓ (45s)
```

单步请求走普通路由，没有 planner 开销。Planner 是可选的，不配置时一切照旧。

### Config Seed

新节点可从 coordinator 拉取共享配置：

```bash
# 自动安装运行时 + 同步 AI 提供商配置 + 同步身份/记忆文件
claw-mesh join <coordinator-url> --auto-install

# 跳过配置同步（使用本地已有配置）
claw-mesh join <coordinator-url> --auto-install --no-sync-config
```

Coordinator 分发 API key、模型配置和身份层文件（SOUL.md、IDENTITY.md、MEMORY.md）。节点本地设置（频道、端口、主机名）不会同步。

## 前置条件

每个节点需要一个 AI 运行时，claw-mesh 支持两种：

| | [OpenClaw](https://github.com/openclaw/openclaw) | [ZeroClaw](https://github.com/zeroclaw-labs/zeroclaw) |
|---|---|---|
| 语言 | Node.js / TypeScript | Rust |
| 大小 | ~200 MB（含 node_modules） | ~5 MB |
| 内存 | 建议 512 MB+ | < 50 MB |
| 依赖 | Node.js ≥ 22 | 无（静态二进制） |
| 频道 | Telegram、WhatsApp、Slack、Discord 等 | CLI、HTTP API |
| 适用 | 全功能桌面环境 | 无头服务器、ARM/嵌入式 |

`--auto-install` 根据硬件自动选择。也可手动安装：

```bash
# OpenClaw
npm install -g openclaw@latest && openclaw onboard --install-daemon

# ZeroClaw
curl -fsSL https://github.com/zeroclaw-labs/zeroclaw/releases/latest/download/zeroclaw-$(uname -m)-unknown-linux-gnu.tar.gz | tar xz -C ~/.local/bin/
```

社区运行时（[TinyClaw](https://github.com/suislanchez/tinyclaw)、[MobClaw](https://github.com/wamynobe/mobclaw)、[NetClaw](https://github.com/Aisht669/NetClaw) 等）可通过 `--no-gateway` 或手动配置 endpoint 加入。

## 架构

```
                          ┌──────────────────────────┐
                          │     claw-mesh coord       │
                          │                           │
                          │  Router · Registry        │
                          │  Planner · Health         │
                          │  Dashboard · Seed API     │
                          └──┬───────┬───────┬────┬───┘
                             │       │       │    │
           ┌─────────────────┘       │       │    └──────────────────┐
           │          ┌──────────────┘       └──────────┐            │
           ▼          ▼                                 ▼            ▼
  ┌──────────────┐ ┌──────────────┐ ┌────────────────┐ ┌──────────────┐
  │ mac-mini      │ │ linux-gpu    │ │ vps-tokyo       │ │ pi-home       │
  │ darwin/arm64  │ │ linux/amd64  │ │ linux/amd64     │ │ linux/arm64   │
  │ 16GB, Metal   │ │ 64GB, A100   │ │ 4GB, public IP  │ │ 4GB           │
  │               │ │              │ │                 │ │               │
  │ xcode, golang │ │ docker, k8s  │ │ docker, nginx   │ │ python        │
  │ ios-build     │ │ python, rust │ │ certbot         │ │ sensor-reader │
  │ cocoapods     │ │ sd-xl        │ │ deploy (agent)  │ │ home-auto     │
  └──────────────┘ └──────────────┘ └────────────────┘ └──────────────┘
    Local (LAN)      Local (LAN)      Remote (WAN)       Local (LAN)
```

### 身份与同步模型

所有节点共享同一个 AI 身份，但各自保留独立的本地能力：

| 层 | 同步策略 | 内容 |
|----|----------|------|
| 身份层 | 共享同步 | SOUL.md、IDENTITY.md、`.claude/skills/*.md` |
| 记忆层 | 自动同步 | MEMORY.md、`memory/*.md` |
| 配置层 | 各自独立 | `openclaw.json`、`skills.yaml` |
| 能力层 | 各自独立 | 硬件检测 + skill 可执行性判断 |

你的 AI 在每台机器上知道同样的事。但每台机器贡献自己的独特能力。

## CLI

```bash
# Coordinator
claw-mesh init                            # 生成配置文件和 token
claw-mesh up                              # 启动 coordinator
claw-mesh up --port 9180 --token secret   # 带参数启动

# 节点
claw-mesh join <url>                      # 加入 mesh
claw-mesh join <url> --auto-install       # 加入 + 安装运行时
claw-mesh join <url> --runtime zeroclaw   # 指定运行时
claw-mesh join <url> --no-gateway         # Echo 模式（无 AI 运行时）

# 状态
claw-mesh status                          # Mesh 概览
claw-mesh nodes                           # 节点列表
claw-mesh skills                          # 查看 mesh 中所有 skill

# 消息
claw-mesh send --auto "消息"              # 自动路由
claw-mesh send --node mac-mini "消息"     # 发送到指定节点
claw-mesh plan status <plan-id>           # 查看多步任务进度

# 路由规则
claw-mesh route list
claw-mesh route add --match "gpu:true" --target linux-gpu
claw-mesh route add --match "skills:docker,python" --strategy least-busy
claw-mesh route add --match "any-skill:xcode,docker" --strategy least-busy
```

## 配置

```yaml
# claw-mesh.yaml
coordinator:
  port: 9180
  token: "your-secret-token"
  allow_private: true
  workspace_dir: "/home/user/clawd"                    # Config seed 用
  openclaw_config: "~/.config/openclaw/openclaw.json"  # Config seed 用
  planner:                                             # 可选 LLM planner
    endpoint: "https://api.openai.com/v1"
    token: "sk-..."
    model: "gpt-4o"

node:
  name: "my-node"
  tags: ["gpu", "docker"]
  skills_manifest: "./skills.yaml"  # 自定义 skill 声明
```

## 安全

- 所有写操作端点使用 Bearer token 认证
- 每个节点独立 token（注册时生成）
- 端点验证（SSRF 防护）
- 私有 IP 拦截（可通过 `--allow-private` 配置）
- Config seed API 需要认证；生产环境通过 HTTPS 保护 API key

## 故障排查

**启动时报 `yaml: invalid trailing UTF-8 octet`** — 不要把二进制构建到项目根目录。Viper 会尝试解析 `claw-mesh.*` 文件。始终用 `make build`（输出到 `bin/`）。

**加入时报 `registration failed (502)`** — HTTP 代理干扰（用 `no_proxy=<ip>` 绕过）或私有 IP 被拒绝（coordinator 启动时加 `--allow-private`）。

**构建时报 `invalid go version`** — `go.mod` 指定了 Go 1.25。升级 Go 或降低版本号。

## 开发

```bash
make build           # 构建二进制
make test            # 运行测试
make lint            # 代码检查（需要 golangci-lint）
make run-coordinator # 本地启动 coordinator
make run-node        # 作为本地节点加入
```

多机器开发辅助脚本（在脚本顶部配置 IP）：

```bash
./scripts/e2e-deploy.sh   # 构建、部署、测试、清理
./scripts/start.sh        # 启动 coordinator + 远端节点
./scripts/stop.sh         # 停止所有进程
```

## Roadmap

- [x] CLI 单二进制（coordinator + node + 管理命令）
- [x] 节点注册 + 心跳 + 自动下线检测
- [x] 能力检测（OS、架构、GPU、内存、工具）
- [x] 手动 + 自动路由（least-busy 策略）
- [x] Web 控制台
- [x] Token 认证 + SSRF 防护
- [x] GoReleaser + GitHub Actions CI
- [x] Config seed（新节点自动配置同步）
- [ ] Skill 感知路由（agent skill、custom skill、skill 类型）
- [ ] 跨节点任务规划（LLM planner）
- [ ] 记忆/身份同步（git-based）
- [ ] 任务队列 + 重试 + 超时
- [ ] 节点分组
- [ ] Prometheus metrics

## License

MIT — 见 [LICENSE](LICENSE)
