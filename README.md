# 🦞 claw-mesh

> One mesh, many claws — orchestrate OpenClaw across machines.

claw-mesh is a multi-gateway orchestrator for [OpenClaw](https://github.com/openclaw/openclaw). Run OpenClaw on multiple machines and let claw-mesh handle node discovery, capability-based routing, and message forwarding — all from a single binary.

## Why?

Your AI assistant shouldn't be trapped on one machine. Mac has Xcode, Linux has GPU, VPS has a public IP — claw-mesh makes them work together.

- **Cross-machine capabilities** — route tasks to the right node automatically
- **Load balancing** — busy node? messages flow to idle ones
- **Failover** — node goes down, traffic reroutes
- **Web Dashboard** — see everything at a glance

## Quick Start

```bash
# Install
go install github.com/SallyKAN/claw-mesh/cmd/claw-mesh@latest

# Or build from source
git clone https://github.com/SallyKAN/claw-mesh.git
cd claw-mesh && make build

# Start coordinator
./bin/claw-mesh up --port 9180 --token mysecret

# Join from another machine (or another terminal)
./bin/claw-mesh join http://coordinator:9180 --name mac-mini --tags xcode,local --token mysecret
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
