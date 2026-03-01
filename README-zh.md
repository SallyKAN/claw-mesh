# 🦞 claw-mesh

[English](README.md)

> One mesh, many claws — 跨机器编排 OpenClaw。

claw-mesh 是 [OpenClaw](https://github.com/openclaw/openclaw) 的多网关编排工具。在多台机器上运行 OpenClaw，claw-mesh 负责节点发现、基于能力的路由和消息转发——全部集成在一个二进制文件中。

## 为什么需要？

你的 AI 助手不应该被困在一台机器上。Mac 有 Xcode，Linux 有 GPU，VPS 有公网 IP——claw-mesh 让它们协同工作。

- **跨机器能力互补** — 自动将任务路由到合适的节点
- **负载均衡** — 节点繁忙？消息自动流向空闲节点
- **故障转移** — 节点宕机，流量自动重路由
- **Web 控制台** — 一目了然

## 前置条件

每台加入 mesh 的机器需要一个 AI 运行时（负责与 AI 提供商通信的 Gateway）。claw-mesh 支持两种运行时：

| | [OpenClaw](https://github.com/openclaw/openclaw) | [ZeroClaw](https://github.com/zeroclaw-labs/zeroclaw) |
|---|---|---|
| 语言 | Node.js / TypeScript | Rust |
| 二进制大小 | ~200 MB（含 node_modules） | ~5 MB |
| 内存占用 | 建议 512 MB+ | < 50 MB |
| 依赖 | Node.js ≥ 22 | 无（静态二进制） |
| 频道支持 | Telegram、WhatsApp、Slack、Discord 等 | CLI、HTTP API |
| 适用场景 | 全功能桌面环境 | 无头服务器、ARM/嵌入式、低资源设备 |

**最简方式 — 让 claw-mesh 自动选择：**

```bash
# 自动检测硬件环境并安装最合适的运行时
claw-mesh join <coordinator-url> --auto-install
```

`--auto-install` 会检测系统环境（内存、Node.js 是否可用）并选择合适的运行时。有 Node.js 的 Mac 上会安装 OpenClaw；没有 Node.js 的 Linux 服务器上会安装 ZeroClaw。

**手动安装：**

```bash
# OpenClaw（需要 Node ≥22）
npm install -g openclaw@latest
openclaw onboard --install-daemon

# 或 ZeroClaw（无依赖）
curl -fsSL https://github.com/zeroclaw-labs/zeroclaw/releases/latest/download/zeroclaw-$(uname -m)-unknown-linux-gnu.tar.gz | tar xz -C ~/.local/bin/
```

**社区运行时：** Claw 生态还有社区移植版本，如 [TinyClaw](https://github.com/suislanchez/tinyclaw)（Rust 超轻量）、[MobClaw](https://github.com/wamynobe/mobclaw)（Android/Kotlin）、[NetClaw](https://github.com/Aisht669/NetClaw)（.NET）等。claw-mesh 目前编排 OpenClaw 和 ZeroClaw；社区运行时可通过 `--no-gateway`（echo 模式）或手动配置 gateway endpoint 加入。

## 快速开始

```bash
# 安装 claw-mesh
go install github.com/SallyKAN/claw-mesh/cmd/claw-mesh@latest

# 或从源码构建
git clone https://github.com/SallyKAN/claw-mesh.git
cd claw-mesh && make build

# 启动 coordinator（局域网加 --allow-private）
./bin/claw-mesh up --port 9180 --token mysecret --allow-private

# 从另一台机器（或另一个终端）加入
./bin/claw-mesh join http://<coordinator-ip>:9180 --name mac-mini --tags xcode,local --token mysecret --auto-install
```

打开 `http://localhost:9180` 查看 Web 控制台。

## 架构

```
                ┌─────────────────────┐
                │   claw-mesh coord   │
                │  Router · Registry  │
                │  Health · Dashboard │
                └──────┬──────┬───────┘
                       │      │
          ┌────────────┘      └────────────┐
          ▼                                ▼
 ┌─────────────────┐             ┌─────────────────┐
 │  Node A (Mac)   │             │  Node B (Linux)  │
 │  OpenClaw GW    │             │  OpenClaw GW     │
 │  xcode, notes   │             │  gpu, docker     │
 └─────────────────┘             └──────────────────┘
```

## CLI 速查

```bash
claw-mesh up                    # 启动 coordinator
claw-mesh join <url>            # 作为节点加入
claw-mesh join <url> --auto-install          # 加入 + 自动安装运行时
claw-mesh join <url> --runtime zeroclaw      # 指定运行时加入
claw-mesh join <url> --no-gateway            # echo 模式加入（无 AI 运行时）
claw-mesh status                # 查看 mesh 概览
claw-mesh nodes                 # 列出所有节点
claw-mesh send --auto "msg"     # 自动路由消息
claw-mesh send --node mac "msg" # 发送到指定节点
claw-mesh route list            # 查看路由规则
claw-mesh route add --match "gpu:true" --target linux-gpu
```

## 路由

消息通过匹配规则路由到对应能力的节点：

```yaml
# GPU 任务路由到 Linux
- match: { requires_gpu: true }
  target: linux-gpu

# macOS 任务路由到 Mac
- match: { requires_os: darwin }
  target: mac-nodes

# 默认：最空闲的节点
- match: { wildcard: true }
  strategy: least-busy
```

## 配置

```yaml
# claw-mesh.yaml
coordinator:
  port: 9180
  token: "your-secret-token"
  allow_private: true  # 允许私有/回环 IP

node:
  name: "my-node"
  tags: ["gpu", "docker"]
```

## 安全

- 所有写操作端点使用 Bearer token 认证
- 每个节点独立 token（注册时生成）
- 端点验证（SSRF 防护）
- 私有 IP 拦截（可配置）

## 故障排查

**启动时报 `yaml: invalid trailing UTF-8 octet`**

不要把二进制构建到项目根目录（`go build -o claw-mesh`）。Viper 会搜索 `claw-mesh.*` 配置文件并尝试把二进制当 YAML 解析。始终构建到 `bin/`：

```bash
make build   # 输出到 bin/claw-mesh
```

**加入时报 `registration failed (502)`**

两个常见原因：

1. **HTTP 代理干扰** — 如果机器设置了 `http_proxy`（如 Clash），请求会经过代理导致失败。绕过它：
   ```bash
   no_proxy=<coordinator-ip> ./bin/claw-mesh join http://<coordinator-ip>:9180 ...
   ```

2. **私有 IP 被拒绝** — 默认情况下 coordinator 会拦截私有/回环 IP（SSRF 防护）。如果节点在同一局域网（如 `192.168.x.x`、`10.x.x.x`），启动 coordinator 时加 `--allow-private`：
   ```bash
   # 局域网 — 节点在私有网络
   ./bin/claw-mesh up --port 9180 --token mysecret --allow-private

   # 公网 — 节点有公网 IP，无需此参数
   ./bin/claw-mesh up --port 9180 --token mysecret
   ```

**构建时报 `invalid go version`**

`go.mod` 指定了 Go 1.25。如果你的 Go 版本较旧，升级 Go 或降低 `go.mod` 中的版本号。

## 脚本

多机器开发辅助脚本（在每个脚本顶部配置 IP）：

```bash
./scripts/e2e-deploy.sh   # 构建、部署到远端、启动、测试、清理
./scripts/start.sh        # 后台启动 coordinator + 远端节点
./scripts/stop.sh         # 停止所有进程
```

## 开发

```bash
make build          # 构建二进制
make test           # 运行测试
make lint           # 代码检查（需要 golangci-lint）
make run-coordinator # 本地启动 coordinator
make run-node       # 作为本地节点加入
```

## Roadmap

- [x] CLI 单二进制
- [x] 节点注册 + 心跳
- [x] 能力检测
- [x] 手动 + 自动路由
- [x] Web 控制台
- [x] Token 认证 + SSRF 防护
- [x] GoReleaser + CI
- [ ] 记忆/配置同步（git-based）
- [ ] 任务队列 + 重试 + 超时
- [ ] 节点分组
- [ ] Prometheus metrics
- [ ] Gateway Federation

## License

MIT — 见 [LICENSE](LICENSE)
