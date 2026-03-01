# 🦞 claw-mesh

[中文](README-zh.md)

> One mesh, many claws — orchestrate OpenClaw across machines.

claw-mesh is a multi-gateway orchestrator for [OpenClaw](https://github.com/openclaw/openclaw). Run OpenClaw on multiple machines and let claw-mesh handle node discovery, capability-based routing, and message forwarding — all from a single binary.

## Why?

Your AI assistant shouldn't be trapped on one machine. Mac has Xcode, Linux has GPU, VPS has a public IP — claw-mesh makes them work together.

- **Cross-machine capabilities** — route tasks to the right node automatically
- **Load balancing** — busy node? messages flow to idle ones
- **Failover** — node goes down, traffic reroutes
- **Web Dashboard** — see everything at a glance

## Prerequisites

Each machine joining the mesh needs an AI runtime (the gateway that actually talks to your AI provider). claw-mesh supports two runtimes:

| | [OpenClaw](https://github.com/openclaw/openclaw) | [ZeroClaw](https://github.com/zeroclaw-labs/zeroclaw) |
|---|---|---|
| Language | Node.js / TypeScript | Rust |
| Binary size | ~200 MB (with node_modules) | ~5 MB |
| Memory | 512 MB+ recommended | < 50 MB |
| Requires | Node.js ≥ 22 | Nothing (static binary) |
| Channels | Telegram, WhatsApp, Slack, Discord, etc. | CLI, HTTP API |
| Best for | Full-featured desktop setups | Headless servers, ARM/embedded, low-resource devices |

**Easiest path — let claw-mesh decide:**

```bash
# Auto-detect hardware and install the best runtime
claw-mesh join <coordinator-url> --auto-install
```

`--auto-install` checks your system (memory, Node.js availability) and picks the right runtime. On a beefy Mac with Node.js it installs OpenClaw; on a headless Linux box without Node.js it installs ZeroClaw.

**Manual install (if you prefer):**

```bash
# OpenClaw (Node ≥22 required)
npm install -g openclaw@latest
openclaw onboard --install-daemon

# Or ZeroClaw (no dependencies)
curl -fsSL https://github.com/zeroclaw-labs/zeroclaw/releases/latest/download/zeroclaw-$(uname -m)-unknown-linux-gnu.tar.gz | tar xz -C ~/.local/bin/
```

**Community runtimes:** The Claw ecosystem also includes community ports like [TinyClaw](https://github.com/suislanchez/tinyclaw) (Rust, ultra-light), [MobClaw](https://github.com/wamynobe/mobclaw) (Android/Kotlin), [NetClaw](https://github.com/Aisht669/NetClaw) (.NET), and others. claw-mesh currently orchestrates OpenClaw and ZeroClaw; community runtimes can join via `--no-gateway` (echo mode) or manual gateway endpoint configuration.

## Quick Start

```bash
# Install claw-mesh
go install github.com/SallyKAN/claw-mesh/cmd/claw-mesh@latest

# Or build from source
git clone https://github.com/SallyKAN/claw-mesh.git
cd claw-mesh && make build

# Start coordinator (add --allow-private for LAN setups)
./bin/claw-mesh up --port 9180 --token mysecret --allow-private

# Join from another machine (or another terminal)
./bin/claw-mesh join http://<coordinator-ip>:9180 --name mac-mini --tags xcode,local --token mysecret --auto-install
```

Open `http://localhost:9180` for the web dashboard.

## Architecture

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

## CLI

```bash
claw-mesh up                    # Start coordinator
claw-mesh join <url>            # Join as a node
claw-mesh join <url> --auto-install          # Join + auto-install runtime
claw-mesh join <url> --runtime zeroclaw      # Join with specific runtime
claw-mesh join <url> --no-gateway            # Join in echo mode (no AI runtime)
claw-mesh status                # Mesh overview
claw-mesh nodes                 # List all nodes
claw-mesh send --auto "msg"     # Auto-route a message
claw-mesh send --node mac "msg" # Send to specific node
claw-mesh route list            # View routing rules
claw-mesh route add --match "gpu:true" --target linux-gpu
```

## Routing

Messages are routed by matching rules against node capabilities:

```yaml
# Route GPU tasks to Linux
- match: { requires_gpu: true }
  target: linux-gpu

# Route macOS tasks to Mac
- match: { requires_os: darwin }
  target: mac-nodes

# Default: least busy node
- match: { wildcard: true }
  strategy: least-busy
```

## Configuration

```yaml
# claw-mesh.yaml
coordinator:
  port: 9180
  token: "your-secret-token"
  allow_private: true  # allow private/loopback IPs

node:
  name: "my-node"
  tags: ["gpu", "docker"]
```

## Security

- Bearer token auth on all mutating endpoints
- Per-node tokens (generated on registration)
- Endpoint validation (SSRF protection)
- Private IP blocking (configurable)

## Troubleshooting

**`yaml: invalid trailing UTF-8 octet` on startup**

Don't build the binary to the project root (`go build -o claw-mesh`). Viper searches for `claw-mesh.*` config files and will try to parse the binary as YAML. Always build to `bin/`:

```bash
make build   # outputs to bin/claw-mesh
```

**`registration failed (502)` when joining**

Two common causes:

1. **HTTP proxy interference** — If the joining machine has `http_proxy` set (e.g. Clash), requests to the coordinator go through the proxy and fail. Bypass it:
   ```bash
   no_proxy=<coordinator-ip> ./bin/claw-mesh join http://<coordinator-ip>:9180 ...
   ```

2. **Private IP rejected** — By default, the coordinator blocks private/loopback IPs (SSRF protection). If the joining node is on the same LAN (e.g. `192.168.x.x`, `10.x.x.x`), start the coordinator with `--allow-private`. For nodes with public IPs this is not needed:
   ```bash
   # LAN setup — nodes on private network
   ./bin/claw-mesh up --port 9180 --token mysecret --allow-private

   # Public setup — nodes have public IPs, no flag needed
   ./bin/claw-mesh up --port 9180 --token mysecret
   ```

**`invalid go version` when building**

The `go.mod` specifies Go 1.25. If your machine has an older Go version, either upgrade Go or lower the version in `go.mod`.

## Scripts

Helper scripts for multi-machine development (configure IPs at the top of each script):

```bash
./scripts/e2e-deploy.sh   # Build, deploy to remote, start, test, cleanup
./scripts/start.sh        # Start coordinator + remote node in background
./scripts/stop.sh         # Stop all processes
```

## Development

```bash
make build          # Build binary
make test           # Run tests
make lint           # Lint (requires golangci-lint)
make run-coordinator # Start coordinator locally
make run-node       # Join as local node
```

## Roadmap

- [x] CLI single binary
- [x] Node registration + heartbeat
- [x] Capability detection
- [x] Manual + auto routing
- [x] Web Dashboard
- [x] Token auth + SSRF protection
- [x] GoReleaser + CI
- [ ] Memory/config sync (git-based)
- [ ] Task queue + retry + timeout
- [ ] Node groups
- [ ] Prometheus metrics
- [ ] Gateway Federation

## License

MIT — see [LICENSE](LICENSE)
