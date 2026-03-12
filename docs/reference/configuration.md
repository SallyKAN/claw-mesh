# Configuration

claw-mesh uses a YAML config file (`claw-mesh.yaml`) in the working directory.

Generate one with `claw-mesh init`.

## Full example

```yaml
coordinator:
  port: 9180
  token: "your-secret-token"
  allow_private: true
  workspace_dir: "/home/user/clawd"
  openclaw_config: "~/.config/openclaw/openclaw.json"
  planner:
    endpoint: "https://api.openai.com/v1"
    token: "sk-..."
    model: "gpt-4o"

node:
  name: "my-node"
  tags: ["gpu", "docker"]
  skills_manifest: "./skills.yaml"

routing_rules:
  - match: { requires_gpu: true }
    target: linux-gpu
  - match: { requires_skills: [docker, python] }
    strategy: least-busy
  - match: { wildcard: true }
    strategy: least-busy
```

## Coordinator settings

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `coordinator.port` | int | `9180` | HTTP listen port |
| `coordinator.token` | string | generated | Auth token for node registration |
| `coordinator.allow_private` | bool | `false` | Accept nodes with private IPs |
| `coordinator.workspace_dir` | string | none | Workspace path for config seed |
| `coordinator.openclaw_config` | string | none | OpenClaw config path for seed |
| `coordinator.planner.endpoint` | string | none | LLM API endpoint for task planning |
| `coordinator.planner.token` | string | none | LLM API token |
| `coordinator.planner.model` | string | none | LLM model name |

## Node settings

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `node.name` | string | hostname | Display name |
| `node.tags` | []string | none | Capability tags |
| `node.skills_manifest` | string | none | Path to custom skills YAML |

## Environment variables

| Variable | Overrides |
|----------|-----------|
| `CLAW_MESH_TOKEN` | `coordinator.token` |
| `CLAW_MESH_PORT` | `coordinator.port` |
