# CLI Reference

## Coordinator commands

### `claw-mesh init`

Generate a config file (`claw-mesh.yaml`) with a random token.

```bash
claw-mesh init
claw-mesh init --rotate-token   # regenerate token
```

### `claw-mesh up`

Start the coordinator server.

```bash
claw-mesh up                              # defaults: port 9180
claw-mesh up --port 9180 --token secret   # explicit options
claw-mesh up --allow-private              # allow LAN nodes
```

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | `9180` | HTTP listen port |
| `--token` | from config | Coordinator auth token |
| `--allow-private` | `false` | Accept nodes with private IPs |

## Node commands

### `claw-mesh join`

Join a mesh as a node.

```bash
claw-mesh join <coordinator-url>
claw-mesh join <url> --name mac-mini --tags xcode,local
claw-mesh join <url> --auto-install
claw-mesh join <url> --runtime zeroclaw
claw-mesh join <url> --no-gateway          # echo mode, no AI runtime
```

| Flag | Default | Description |
|------|---------|-------------|
| `--name` | hostname | Node display name |
| `--tags` | none | Comma-separated capability tags |
| `--token` | from config | Auth token |
| `--auto-install` | `false` | Auto-detect and install OpenClaw/ZeroClaw |
| `--runtime` | auto | Force specific runtime (`openclaw` or `zeroclaw`) |
| `--no-gateway` | `false` | Join without an AI runtime (echo mode) |
| `--no-sync-config` | `false` | Skip config seed sync |
| `--skills-manifest` | none | Path to custom `skills.yaml` |

## Status commands

### `claw-mesh status`

Show mesh overview (coordinator info, node count, health).

### `claw-mesh nodes`

List all registered nodes with status, capabilities, and last heartbeat.

### `claw-mesh skills`

List all skills across the mesh, grouped by type and showing which nodes have each skill.

## Messaging commands

### `claw-mesh send`

Send a message through the mesh.

```bash
claw-mesh send --auto "message"           # auto-route
claw-mesh send --node mac-mini "message"  # target specific node
```

## Routing commands

### `claw-mesh route`

Manage routing rules.

```bash
claw-mesh route list
claw-mesh route add --match "gpu:true" --target linux-gpu
claw-mesh route add --match "skills:docker,python" --strategy least-busy
claw-mesh route add --match "any-skill:xcode,docker" --strategy least-busy
```
