# 🦞 claw-mesh

[中文](README-zh.md)

> Your personal AI mesh — one AI, all your devices.

Your AI assistant shouldn't be trapped on one machine. Mac has Xcode, Linux has a GPU, your VPS has a public IP. claw-mesh connects them into a single capability plane — messages route to the right device automatically, tasks span across machines, and if one node goes down, traffic fails over.

Built for [OpenClaw](https://github.com/openclaw/openclaw). Single binary. Zero config on new nodes.

## What it does

```
You: "Build the iOS app, then deploy the backend with Docker"
                              │
                    ┌─────────▼──────────┐
                    │   claw-mesh coord   │
                    │                     │
                    │  1. LLM planner     │
                    │     splits into     │
                    │     2 steps         │
                    │                     │
                    │  2. Skill-aware     │
                    │     routing         │
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

- **Skill-aware routing** — each node reports what it can do (tools, agent skills, custom capabilities). Messages auto-route to the node with the right skill.
- **Cross-node task plans** — complex requests are split into multi-step plans by an LLM planner, each step dispatched to the best node.
- **Failover** — node goes down? tasks reroute to another node with the same skill.
- **Config seed** — new nodes pull AI provider config and identity files from the coordinator. One `join --auto-install` and they're ready.
- **Web dashboard** — real-time node status, skill map, routing rules, message flow.

## Quick start

```bash
# Install
go install github.com/SallyKAN/claw-mesh/cmd/claw-mesh@latest

# Or build from source
git clone https://github.com/SallyKAN/claw-mesh.git
cd claw-mesh && make build

# Initialize config (generates claw-mesh.yaml with token)
./bin/claw-mesh init

# Start coordinator
./bin/claw-mesh up --port 9180 --token mysecret --allow-private

# Join from another machine
./bin/claw-mesh join http://<coordinator-ip>:9180 \
  --name mac-mini --tags xcode,local \
  --token mysecret --auto-install
```

Open `http://localhost:9180` for the dashboard.

## How it works

### Nodes and capabilities

Every machine running an OpenClaw Gateway is a node. When a node joins, it auto-detects its capabilities:

| Detected | Examples |
|----------|---------|
| OS & arch | `darwin/arm64`, `linux/amd64` |
| Hardware | GPU, memory |
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

### Routing

Messages match against rules. All conditions are AND:

```yaml
routing_rules:
  # GPU tasks → Linux
  - match: { requires_gpu: true }
    target: linux-gpu

  # Need both docker AND python
  - match: { requires_skills: [docker, python] }
    strategy: least-busy

  # Need xcode OR docker
  - match: { requires_any_skill: [xcode, docker] }
    strategy: least-busy

  # Default
  - match: { wildcard: true }
    strategy: least-busy
```

### Task plans

When an LLM planner is configured, complex requests are automatically decomposed:

```bash
$ claw-mesh send --auto "Train a model on GPU, then deploy to K8s"
Plan plan-x7y8z9 created (2 steps), executing...

$ claw-mesh plan status plan-x7y8z9
Plan: plan-x7y8z9 [completed]
  Step 1: python → linux-gpu ✓ (180s)
  Step 2: k8s → linux-gpu ✓ (45s)
```

Single-step requests go through normal routing — no planner overhead. The planner is optional; without it, everything works as before.

### Config seed

New nodes can pull shared config from the coordinator:

```bash
# Auto-install runtime + sync AI provider config + sync identity/memory files
claw-mesh join <coordinator-url> --auto-install

# Skip config sync if you have local config
claw-mesh join <coordinator-url> --auto-install --no-sync-config
```

The coordinator shares API keys, model config, and identity-layer files (SOUL.md, IDENTITY.md, MEMORY.md). Node-local settings (channels, ports, hostname) are excluded.

## Prerequisites

Each node needs an AI runtime. claw-mesh supports two:

| | [OpenClaw](https://github.com/openclaw/openclaw) | [ZeroClaw](https://github.com/zeroclaw-labs/zeroclaw) |
|---|---|---|
| Language | Node.js / TypeScript | Rust |
| Size | ~200 MB (with node_modules) | ~5 MB |
| Memory | 512 MB+ | < 50 MB |
| Requires | Node.js ≥ 22 | Nothing (static binary) |
| Channels | Telegram, WhatsApp, Slack, Discord, etc. | CLI, HTTP API |
| Best for | Full-featured desktops | Headless servers, ARM/embedded |

`--auto-install` picks the right one based on your hardware. Or install manually:

```bash
# OpenClaw
npm install -g openclaw@latest && openclaw onboard --install-daemon

# ZeroClaw
curl -fsSL https://github.com/zeroclaw-labs/zeroclaw/releases/latest/download/zeroclaw-$(uname -m)-unknown-linux-gnu.tar.gz | tar xz -C ~/.local/bin/
```

Community runtimes ([TinyClaw](https://github.com/suislanchez/tinyclaw), [MobClaw](https://github.com/wamynobe/mobclaw), [NetClaw](https://github.com/Aisht669/NetClaw), etc.) can join via `--no-gateway` or manual endpoint config.

## Architecture

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

### Identity & sync model

All nodes share one AI identity but have independent local capabilities:

| Layer | Sync | Contents |
|-------|------|----------|
| Identity | Shared | SOUL.md, IDENTITY.md, `.claude/skills/*.md` |
| Memory | Auto-sync | MEMORY.md, `memory/*.md` |
| Config | Independent | `openclaw.json`, `skills.yaml` |
| Capability | Independent | Hardware detection + skill executability |

Your AI knows the same things everywhere. But each machine contributes its own strengths.

## CLI

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
claw-mesh plan status <plan-id>           # Check multi-step plan

# Routing rules
claw-mesh route list
claw-mesh route add --match "gpu:true" --target linux-gpu
claw-mesh route add --match "skills:docker,python" --strategy least-busy
claw-mesh route add --match "any-skill:xcode,docker" --strategy least-busy
```

## Configuration

```yaml
# claw-mesh.yaml
coordinator:
  port: 9180
  token: "your-secret-token"
  allow_private: true
  workspace_dir: "/home/user/clawd"                    # For config seed
  openclaw_config: "~/.config/openclaw/openclaw.json"  # For config seed
  planner:                                             # Optional LLM planner
    endpoint: "https://api.openai.com/v1"
    token: "sk-..."
    model: "gpt-4o"

node:
  name: "my-node"
  tags: ["gpu", "docker"]
  skills_manifest: "./skills.yaml"  # Custom skill declarations
```

## Security

- Bearer token auth on all mutating endpoints
- Per-node tokens (generated on registration)
- Endpoint validation (SSRF protection)
- Private IP blocking (configurable with `--allow-private`)
- Config seed API requires auth; API keys protected via HTTPS in production

## Troubleshooting

**`yaml: invalid trailing UTF-8 octet` on startup** — Don't build the binary to the project root. Viper tries to parse `claw-mesh.*` files. Always use `make build` (outputs to `bin/`).

**`registration failed (502)` when joining** — Either HTTP proxy interference (bypass with `no_proxy=<ip>`) or private IP rejected (start coordinator with `--allow-private` for LAN setups).

**`invalid go version` when building** — `go.mod` specifies Go 1.25. Upgrade Go or lower the version.

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

## Roadmap

- [x] CLI single binary (coordinator + node + management)
- [x] Node registration + heartbeat + auto-offline
- [x] Capability detection (OS, arch, GPU, memory, tools)
- [x] Manual + auto routing (least-busy strategy)
- [x] Web dashboard
- [x] Token auth + SSRF protection
- [x] GoReleaser + GitHub Actions CI
- [x] Config seed (new node auto-provisioning)
- [ ] Skill-aware routing (agent skills, custom skills, skill types)
- [ ] Cross-node task plans (LLM planner)
- [ ] Memory/identity sync (git-based)
- [ ] Task queue + retry + timeout
- [ ] Node groups
- [ ] Prometheus metrics

## License

MIT — see [LICENSE](LICENSE)
