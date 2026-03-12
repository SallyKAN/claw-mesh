# 🦞 claw-mesh — Multi-machine orchestration for OpenClaw

<p align="center">
  <a href="https://github.com/SallyKAN/claw-mesh/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/SallyKAN/claw-mesh/ci.yml?branch=main&style=for-the-badge" alt="CI status"></a>
  <a href="https://github.com/SallyKAN/claw-mesh/releases"><img src="https://img.shields.io/github/v/release/SallyKAN/claw-mesh?include_prereleases&style=for-the-badge" alt="GitHub release"></a>
  <a href="https://goreportcard.com/report/github.com/SallyKAN/claw-mesh"><img src="https://goreportcard.com/badge/github.com/SallyKAN/claw-mesh?style=for-the-badge" alt="Go Report Card"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-blue.svg?style=for-the-badge" alt="MIT License"></a>
</p>

**claw-mesh** connects multiple [OpenClaw](https://github.com/openclaw/openclaw) instances into a single mesh. Your AI assistant shouldn't be trapped on one machine — Mac has Xcode, Linux has a GPU, your VPS has a public IP. claw-mesh routes messages to the right device automatically, syncs identity and memory across nodes, and fails over when a node goes down.

If you run OpenClaw on more than one machine, this is the missing piece.

[Docs](https://SallyKAN.github.io/claw-mesh) · [Quick Start](#quick-start) · [中文](README-zh.md) · [Architecture](#architecture) · [Roadmap](#roadmap)

<!-- TODO: Demo GIF — record with asciinema/VHS: claw-mesh up → two nodes join → send --auto routes to correct node -->
<!-- <p align="center"><img src="docs/assets/demo.gif" width="720" alt="claw-mesh demo"></p> -->

## Great for

- **Cross-machine AI workflows** — Mac has Xcode, Linux has a GPU, VPS has a public IP. Use them all as one unified assistant.
- **Task offloading** — Mac is busy with a coding agent? New messages auto-route to an idle node.
- **Remote dev machines** — `claw-mesh join` from home, operate your office machine's codebase through your AI assistant.
- **Failover** — A node goes down, traffic reroutes to another node with the same capabilities.
- **Shared identity** — your AI remembers the same things on every machine. Memory and identity files sync automatically.

## Install

Single Go binary. ~13 MB. No runtime dependencies.

```bash
# One-liner (macOS / Linux)
curl -fsSL https://raw.githubusercontent.com/SallyKAN/claw-mesh/main/install.sh | sh

# Homebrew
brew install SallyKAN/tap/claw-mesh

# Go
go install github.com/SallyKAN/claw-mesh/cmd/claw-mesh@latest

# From source
git clone https://github.com/SallyKAN/claw-mesh.git
cd claw-mesh && make build
```

## Quick start

```bash
# 1. Initialize config (generates claw-mesh.yaml with auth token)
claw-mesh init

# 2. Start coordinator
claw-mesh up --allow-private

# 3. Join from another machine
claw-mesh join http://<coordinator-ip>:9180 \
  --name mac-mini --tags xcode,local \
  --token <your-token> --auto-install

# 4. Check mesh status
claw-mesh status
claw-mesh nodes
```

Open `http://localhost:9180` for the web dashboard.

`--auto-install` detects and installs the best runtime for the machine. See [Runtimes](#runtimes) below.

> **What is OpenClaw?** [OpenClaw](https://github.com/openclaw/openclaw) is an open-source personal AI assistant platform (247K+ stars). Each machine runs an OpenClaw Gateway that connects to LLMs and messaging channels (Telegram, WhatsApp, Slack, Discord, etc.). claw-mesh orchestrates multiple gateways into one unified mesh.

## Highlights

- **[Capability-aware routing](#routing)** — each node auto-detects OS, arch, GPU, memory, and installed tools. Messages route to the node that can handle them.
- **[Config seed](#config-seed)** — new nodes pull AI provider config and identity files from the coordinator. One `join --auto-install` and they're ready.
- **[File sync](#file-sync-v02)** — identity files (SOUL.md, IDENTITY.md) and memory files sync across all nodes automatically.
- **[Web dashboard](#web-dashboard)** — real-time node topology, status monitoring, interactive chat panel, and add-node wizard at `:9180`.
- **[Multi-runtime](#runtimes)** — supports [OpenClaw](https://github.com/openclaw/openclaw), [ZeroClaw](https://github.com/zeroclaw-labs/zeroclaw), and community runtimes.
- **[OpenClaw version management](#openclaw-version-management)** — auto-detects OpenClaw version on node registration, prompts upgrade if outdated.
- **[Security](#security)** — bearer token auth, per-node tokens, SSRF protection, private IP blocking.

## Architecture

```
                          ┌──────────────────────────┐
                          │     claw-mesh coord       │
                          │                           │
                          │  Router · Registry        │
                          │  Health · Dashboard       │
                          │  Seed API · File Sync     │
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

| Component | Role |
|-----------|------|
| **Coordinator** | Central server — HTTP/WS server, node registry, health checks, message routing, web dashboard, file sync hub. |
| **Node Agent** | Lightweight sidecar on each Gateway machine — registration, heartbeat, capability reporting, message forwarding. |
| **Web Dashboard** | SPA embedded in the coordinator binary — node topology, status, chat panel, add-node wizard. |
| **CLI** | Single binary that acts as coordinator, node agent, and management tool depending on the subcommand. |

## Nodes and capabilities

Every machine running an OpenClaw Gateway is a node. When a node joins, it auto-detects:

| Detected | Examples |
|----------|---------|
| OS & arch | `darwin/arm64`, `linux/amd64` |
| Hardware | GPU presence, memory size |
| Tool skills | `docker`, `xcode`, `python`, `kubectl` — auto-detected from PATH |
| Agent skills | `.claude/skills/*.md` — parsed for `requires` (OS, tools, tags) to determine executability per node |
| Custom skills | Declared in `skills.yaml` — e.g. `stable-diffusion`, `data-pipeline` |

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

Node status: **online** (healthy, accepting messages) · **busy** (processing, deprioritized) · **offline** (missed heartbeat, default 90s timeout).

## Routing

Messages match against routing rules top-to-bottom. All conditions within a rule are AND:

```yaml
routing_rules:
  # GPU tasks → specific node
  - match: { requires_gpu: true }
    target: linux-gpu

  # Need both docker AND python
  - match: { requires_skills: [docker, python] }
    strategy: least-busy

  # Need xcode OR docker
  - match: { requires_any_skill: [xcode, docker] }
    strategy: least-busy

  # Default fallback
  - match: { wildcard: true }
    strategy: least-busy
```

Strategies: `least-busy` (fewest active tasks) · `round-robin` · `target: <name>` (pinned).

If the target node is offline, the coordinator automatically tries the next matching node.

CLI:
```bash
claw-mesh route list
claw-mesh route add --match "gpu:true" --target linux-gpu
claw-mesh route add --match "skills:docker,python" --strategy least-busy
```

## Config seed

New nodes can pull shared config from the coordinator so you don't have to set up each machine manually:

```bash
# Auto-install runtime + sync AI provider config + sync identity/memory files
claw-mesh join <coordinator-url> --auto-install

# Skip config sync if you have local config
claw-mesh join <coordinator-url> --auto-install --no-sync-config
```

The coordinator shares API keys, model config, and identity-layer files (SOUL.md, IDENTITY.md, MEMORY.md). Node-local settings (channels, ports, hostname) are excluded.

## File sync (v0.2)

Identity and memory files sync across all mesh nodes automatically:

| Layer | Sync | Contents |
|-------|------|----------|
| Identity | Shared | SOUL.md, IDENTITY.md, agent skills |
| Memory | Auto-sync | MEMORY.md, `memory/*.md` |
| Config | Independent | `openclaw.json`, `skills.yaml` |
| Capability | Independent | Hardware detection + tool availability |

Your AI knows the same things everywhere. But each machine contributes its own strengths.

Sync is file-based — the coordinator acts as the hub, nodes push/pull on join and periodically. Conflicts are resolved by last-write-wins at the file level.

## Web dashboard

<!-- TODO: replace with actual screenshot -->
<!-- <p align="center"><img src="docs/assets/dashboard.png" width="720" alt="claw-mesh dashboard"></p> -->

The dashboard is embedded in the coordinator binary and served at `http://coordinator:9180/`. Features:

- **Node topology** — visual map of all nodes, their status, capabilities, and last heartbeat.
- **Chat panel** — send messages through the mesh directly from the browser, with auto-routing or node targeting.
- **Add Node wizard** — interactive form to generate the `claw-mesh join` command for a new machine.
- **Real-time updates** — node online/offline events stream via WebSocket.

Dark theme. Built with vanilla HTML/CSS/JS. No build step required.

## OpenClaw version management

When a node registers, claw-mesh auto-detects the installed OpenClaw version. If the version is outdated compared to the coordinator's known latest, the dashboard shows an upgrade prompt. The node can be upgraded via:

```bash
claw-mesh join <url> --auto-install  # re-run to upgrade
```

## Runtimes

Each node needs an AI runtime. claw-mesh supports:

| | [OpenClaw](https://github.com/openclaw/openclaw) | [ZeroClaw](https://github.com/zeroclaw-labs/zeroclaw) |
|---|---|---|
| Language | Node.js / TypeScript | Rust |
| Size | ~200 MB (with node_modules) | ~5 MB |
| Memory | 512 MB+ | < 50 MB |
| Requires | Node.js ≥ 22 | Nothing (static binary) |
| Channels | Telegram, WhatsApp, Slack, Discord, etc. | CLI, HTTP API |
| Best for | Full-featured desktops | Headless servers, ARM/embedded |

`--auto-install` picks the right one based on hardware. Or install manually:

```bash
# OpenClaw
npm install -g openclaw@latest && openclaw onboard --install-daemon

# ZeroClaw
curl -fsSL https://github.com/zeroclaw-labs/zeroclaw/releases/latest/download/zeroclaw-$(uname -m)-unknown-linux-gnu.tar.gz | tar xz -C ~/.local/bin/
```

Community runtimes ([TinyClaw](https://github.com/suislanchez/tinyclaw), [MobClaw](https://github.com/wamynobe/mobclaw), [NetClaw](https://github.com/Aisht669/NetClaw), etc.) can join via `--no-gateway` or manual endpoint config.

## Security

- **Bearer token auth** on all mutating endpoints
- **Per-node tokens** generated on registration
- **Endpoint validation** (SSRF protection)
- **Private IP blocking** (configurable with `--allow-private` for LAN setups)
- **Config seed API** requires auth; API keys protected via HTTPS in production

| Environment | Recommended setup |
|-------------|-------------------|
| LAN / home lab | `claw-mesh up --allow-private` |
| Public internet | HTTPS (reverse proxy) + strong token |
| Mixed | WireGuard / Tailscale tunnel, then `--allow-private` |

## CLI reference

```bash
# Coordinator
claw-mesh init                            # Generate config file with token
claw-mesh up                              # Start coordinator
claw-mesh up --port 9180 --token secret   # With options

# Nodes
claw-mesh join <url>                      # Join mesh
claw-mesh join <url> --auto-install       # Join + install runtime
claw-mesh join <url> --runtime zeroclaw   # Specific runtime
claw-mesh join <url> --no-gateway         # Echo mode (no AI runtime)

# Status
claw-mesh status                          # Mesh overview
claw-mesh nodes                           # List nodes
claw-mesh skills                          # List all skills across mesh

# Messaging
claw-mesh send --auto "message"           # Auto-route
claw-mesh send --node mac-mini "message"  # Target specific node

# Routing rules
claw-mesh route list
claw-mesh route add --match "gpu:true" --target linux-gpu
claw-mesh route add --match "skills:docker,python" --strategy least-busy
```

Full reference: [CLI docs](https://SallyKAN.github.io/claw-mesh/reference/cli)

## Configuration

```yaml
# claw-mesh.yaml
coordinator:
  port: 9180
  token: "your-secret-token"
  allow_private: true
  workspace_dir: "/home/user/clawd"
  openclaw_config: "~/.config/openclaw/openclaw.json"

node:
  name: "my-node"
  tags: ["gpu", "docker"]
  skills_manifest: "./skills.yaml"
```

Full reference: [Configuration docs](https://SallyKAN.github.io/claw-mesh/reference/configuration)

## Troubleshooting

**`yaml: invalid trailing UTF-8 octet` on startup** — Don't build the binary to the project root. Viper tries to parse `claw-mesh.*` files. Always use `make build` (outputs to `bin/`).

**`registration failed (502)` when joining** — Either HTTP proxy interference (bypass with `no_proxy=<ip>`) or private IP rejected (start coordinator with `--allow-private` for LAN setups).

**`invalid go version` when building** — `go.mod` specifies Go 1.25. Upgrade Go or lower the version.

## Roadmap

- [x] CLI single binary (coordinator + node + management)
- [x] Node registration + heartbeat + auto-offline detection
- [x] Capability detection (OS, arch, GPU, memory, tools)
- [x] Manual + auto routing (least-busy strategy)
- [x] Web dashboard with chat panel and add-node wizard
- [x] Token auth + SSRF protection
- [x] Config seed (new node auto-provisioning)
- [x] OpenClaw version detection + upgrade prompt
- [x] File sync across mesh nodes (v0.2)
- [ ] Skill-aware routing — agent skills, custom skills, skill types (TODO)
- [ ] Cross-node task plans — LLM planner decomposes complex requests (TODO)
- [ ] Task queue + retry + timeout (TODO)
- [ ] Node groups (TODO)
- [ ] Prometheus metrics (TODO)

## Development

```bash
make build           # Build binary
make test            # Run tests
make lint            # Lint (requires golangci-lint)
make run-coordinator # Start coordinator locally
make run-node        # Join as local node
```

Helper scripts for multi-machine dev (configure IPs at the top):

```bash
./scripts/e2e-deploy.sh   # Build, deploy, test, cleanup
./scripts/start.sh        # Start coordinator + remote node
./scripts/stop.sh         # Stop all
```

## License

MIT — see [LICENSE](LICENSE)

---

<p align="center">
  <a href="https://github.com/SallyKAN/claw-mesh/graphs/contributors">
    <img src="https://contrib.rocks/image?repo=SallyKAN/claw-mesh" alt="Contributors">
  </a>
</p>
