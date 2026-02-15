# claw-mesh — Design Document

> Multi-Gateway orchestrator for OpenClaw. One mesh, many claws.

## 1. Overview

claw-mesh 是一个面向开发者的开源工具，为 OpenClaw 提供多机编排能力。用户在多台机器上各跑一个 OpenClaw Gateway，claw-mesh 作为中心 coordinator 统一管理消息路由、节点发现和任务分发。

**核心价值：** 你的 AI 助手不再被困在一台机器上。Mac 上有 Xcode，Linux 上有 GPU，VPS 上有公网 IP——claw-mesh 让它们协同工作。

## 2. Use Cases

### 场景 1：跨机器能力互补
Mac 上有 Xcode 和 Apple Notes，Linux 上有 GPU 和 Docker。用户发"帮我生成一张图片"→ 自动路由到 GPU 节点；发"查一下我的备忘录"→ 自动路由到 Mac。用户无需关心消息去了哪台机器。

### 场景 2：任务卸载 / 负载均衡
Mac Mini 正在跑一个耗时的 coding agent 任务，新消息进来时自动分流到空闲的 Linux 节点，不排队。

### 场景 3：远程开发机
公司有台开发机，家里有台 Mac。通过 `claw-mesh join` 把公司机器加入 mesh，在家也能让 AI 助手操作公司机器上的代码仓库。

### 场景 4：高可用 / Failover
一台机器挂了或重启，消息自动 failover 到另一台在线节点，AI 助手不中断服务。

### 场景 5：专机专用
一台机器专门跑内容创作（content agent），另一台专门跑开发任务（coding），通过路由规则隔离，互不干扰。

## 3. Identity & Sync Model

所有节点共享同一个 AI 助手身份，但各自保留本地能力差异。分四层：

| 层 | 策略 | 内容 |
|----|------|------|
| 身份层 | 共享同步 | SOUL.md、USER.md、IDENTITY.md、AGENTS.md — 人格统一 |
| 记忆层 | 自动同步 | MEMORY.md、memory/*.md — git-based 同步，冲突按时间戳合并 |
| 配置层 | 各自独立 | openclaw.json — 每台机器的 channels、env、skills 路径不同 |
| 能力层 | 各自独立 | 本地 skills、工具链、硬件（GPU/Docker/Xcode 等） |

用户感知到的是"同一个 AI 助手"——Mac 上聊过的事，Linux 上也知道。但每台机器有自己的能力特长，coordinator 根据能力差异做智能路由。

记忆同步机制（v0.2）：
- 底层用 git，每个节点自动 commit + push/pull
- 身份文件变更少，直接覆盖
- 记忆文件（daily notes）冲突时按时间戳行级合并
- 配置文件不同步，各节点独立管理

## 4. Architecture

```
                    ┌─────────────────────┐
                    │   claw-mesh coord   │
                    │  (central server)   │
                    │                     │
                    │  ┌───────────────┐  │
  User ──────────►  │  │ Router        │  │
  (Telegram/        │  │ Node Registry │  │
   Discord/         │  │ Health Check  │  │
   CLI)             │  │ Web Dashboard │  │
                    │  └───────────────┘  │
                    └──────┬──────┬───────┘
                           │      │
              ┌────────────┘      └────────────┐
              ▼                                ▼
     ┌─────────────────┐             ┌─────────────────┐
     │  Node A (local) │             │  Node B (remote) │
     │  Mac Mini       │             │  Linux VPS       │
     │  OpenClaw GW    │             │  OpenClaw GW     │
     │                 │             │                  │
     │  capabilities:  │             │  capabilities:   │
     │  - macos        │             │  - linux         │
     │  - xcode        │             │  - gpu: A100     │
     │  - 16GB RAM     │             │  - docker        │
     └─────────────────┘             └──────────────────┘
```

### 组件

| 组件 | 职责 |
|------|------|
| Coordinator | 中心节点，运行 HTTP/WebSocket server，管理节点注册、健康检查、消息路由 |
| Node Agent | 轻量 sidecar，跑在每台 Gateway 机器上，负责注册、心跳、能力上报、接收转发的消息 |
| Web Dashboard | 嵌入 coordinator 的 SPA，展示节点状态、路由规则、消息流 |
| CLI | 单二进制，coordinator + node agent + 管理命令合一 |

## 5. Core Concepts

### 3.1 Node（节点）

每台运行 OpenClaw Gateway 的机器就是一个 Node。Node 通过 `claw-mesh join` 注册到 coordinator。

```yaml
node:
  id: "node-a1b2c3"          # 自动生成
  name: "mac-mini"            # 用户自定义
  endpoint: "192.168.1.100:9120"  # Node Agent 监听地址
  capabilities:
    os: "darwin"
    arch: "arm64"
    gpu: false
    memory_gb: 16
    tags: ["xcode", "homebrew", "local"]
    skills: ["coding-agent", "apple-notes"]  # 从 OpenClaw 自动发现
  status: "online"            # online / offline / busy
  last_heartbeat: "2026-02-13T12:00:00Z"
```

### 3.2 Capability Matching（能力匹配）

路由消息时，coordinator 根据消息需求匹配节点能力：

```yaml
routing_rules:
  - match:
      requires_gpu: true
    target: "node-with-gpu"
    
  - match:
      requires_os: "darwin"
    target: "mac-nodes"
    
  - match:
      requires_skill: "docker"
    target: "linux-nodes"
    
  - match: "*"               # 默认：负载最低的节点
    strategy: "least-busy"
```

### 3.3 消息流

```
1. 用户发消息 → Telegram/Discord/CLI
2. 消息到达 coordinator（作为统一入口）
3. coordinator 评估消息 → 匹配路由规则
4. 转发到目标 Node 的 OpenClaw Gateway
5. Gateway 处理 → 响应回传 coordinator
6. coordinator 回复用户
```

## 6. CLI Design

单二进制，子命令模式：

```bash
# 安装
curl -fsSL https://get.claw-mesh.dev | sh

# 初始化 coordinator
claw-mesh init
claw-mesh up                          # 启动 coordinator（默认 :9180）
claw-mesh up --port 9180 --token <secret>

# 节点加入（在 Gateway 机器上执行）
claw-mesh join <coordinator-url>      # 自动检测本机 OpenClaw Gateway
claw-mesh join https://coord.example.com --name "linux-gpu" --tags "gpu,docker"

# 管理
claw-mesh status                      # 查看所有节点
claw-mesh nodes                       # 节点列表
claw-mesh route list                  # 查看路由规则
claw-mesh route add --match "gpu:true" --target "linux-gpu"

# 手动发消息到指定节点
claw-mesh send --node "mac-mini" "帮我跑一下 Xcode build"
claw-mesh send --auto "生成一张图片"   # 自动路由
```

## 7. API Design

Coordinator 暴露 REST + WebSocket API：

### REST

```
POST   /api/v1/nodes/register     # 节点注册
DELETE /api/v1/nodes/:id           # 节点注销
GET    /api/v1/nodes               # 节点列表
GET    /api/v1/nodes/:id           # 节点详情
POST   /api/v1/nodes/:id/heartbeat # 心跳

POST   /api/v1/route               # 发送消息（自动路由）
POST   /api/v1/route/:nodeId       # 发送消息（指定节点）

GET    /api/v1/rules               # 路由规则
POST   /api/v1/rules               # 添加规则
DELETE /api/v1/rules/:id           # 删除规则

GET    /api/v1/dashboard/stats     # Dashboard 数据
```

### WebSocket

```
ws://coordinator:9180/ws/events    # 实时事件流（节点上下线、消息路由日志）
```

## 8. Web Dashboard (v0.1)

嵌入 coordinator 二进制，访问 `http://coordinator:9180/`

页面：
- **Overview** — 节点拓扑图（哪些在线、负载情况）、消息吞吐量
- **Nodes** — 节点列表，能力标签，在线状态，操作（踢出/禁用）
- **Routing** — 路由规则管理，消息路由日志
- **Messages** — 最近消息流，可以手动发消息到指定节点

技术栈：React + Tailwind + shadcn/ui，Vite 构建，`go:embed` 嵌入

## 9. Security

- Coordinator ↔ Node 通信使用 token 认证（`claw-mesh init` 时生成）
- 远端节点通过 TLS（推荐 HTTPS）或 WireGuard 隧道连接
- Dashboard 支持 basic auth
- Token rotation 支持

## 10. Tech Stack

| 层 | 技术 |
|----|------|
| Coordinator + Node Agent + CLI | Go 1.22+ |
| 通信 | HTTP/2 + WebSocket（gorilla/websocket 或 nhooyr/websocket） |
| 配置 | YAML（viper） |
| Dashboard 前端 | React 19 + Tailwind 4 + shadcn/ui |
| Dashboard 嵌入 | go:embed |
| 构建 | GoReleaser（cross-compile + GitHub Release + Homebrew tap） |
| CI | GitHub Actions |

## 11. Project Structure

```
claw-mesh/
├── cmd/
│   └── claw-mesh/
│       └── main.go              # CLI 入口
├── internal/
│   ├── coordinator/
│   │   ├── server.go            # HTTP/WS server
│   │   ├── registry.go          # 节点注册表
│   │   ├── router.go            # 消息路由引擎
│   │   ├── health.go            # 健康检查
│   │   └── api.go               # REST handlers
│   ├── node/
│   │   ├── agent.go             # Node Agent
│   │   ├── discovery.go         # OpenClaw Gateway 自动发现
│   │   └── capabilities.go      # 能力采集
│   ├── config/
│   │   └── config.go            # 配置管理
│   └── types/
│       └── types.go             # 共享类型
├── web/                         # Dashboard 前端
│   ├── src/
│   ├── package.json
│   └── vite.config.ts
├── dist/                        # 前端构建产物（go:embed）
├── go.mod
├── go.sum
├── .goreleaser.yml
├── Makefile
├── README.md
└── LICENSE
```

## 12. MVP Scope (v0.1)

### Must Have
- [ ] CLI 单二进制（coordinator + node agent + 管理命令）
- [ ] 节点注册 + 心跳 + 自动下线检测
- [ ] 能力上报（OS、arch、tags、skills 自动发现）
- [ ] 手动路由（指定节点发消息）
- [ ] 自动路由（基于能力匹配 + least-busy 策略）
- [ ] Web Dashboard（节点状态、路由规则、消息流）
- [ ] Token 认证
- [ ] 远端节点支持（公网 IP / 隧道）
- [ ] GoReleaser + GitHub Actions CI
- [ ] 一行安装脚本

### v0.2 Roadmap
- [ ] 记忆/配置同步（git-based）
- [ ] 任务队列 + 重试 + 超时
- [ ] 节点分组
- [ ] Prometheus metrics
- [ ] Gateway Federation 协议（长期愿景）

## 13. Distribution

```bash
# macOS
brew install snape/tap/claw-mesh

# Linux
curl -fsSL https://get.claw-mesh.dev | sh

# Go
go install github.com/SallyKAN/claw-mesh/cmd/claw-mesh@latest

# Docker
docker run -p 9180:9180 SallyKAN/claw-mesh coordinator
```

## 14. Naming & Branding

- **Name:** claw-mesh
- **Tagline:** "One mesh, many claws — orchestrate OpenClaw across machines"
- **Repo:** github.com/SallyKAN/claw-mesh
- **License:** MIT

## 15. OpenClaw Reference

- 源码：`~/openclaw`
- 文档：https://deepwiki.com/openclaw/openclaw
- Gateway 默认端口：18789
- Gateway API：`/v1/chat/completions` (OpenAI 兼容), WebSocket RPC (`agent`, `health` 等)
- Auth token 来源优先级：`gateway.auth.token` > `OPENCLAW_GATEWAY_TOKEN` env > `CLAWDBOT_GATEWAY_TOKEN` env
- 集成设计文档：`docs/openclaw-gateway-integration.md`
