# Nodes & Capabilities

## What is a node?

Every machine running an OpenClaw Gateway is a node. Nodes join the mesh via `claw-mesh join` and automatically report their capabilities to the coordinator.

## Auto-detection

When a node joins, claw-mesh detects:

| Category | Examples |
|----------|---------|
| OS & arch | `darwin/arm64`, `linux/amd64` |
| Hardware | GPU presence, memory size |
| Tool skills | `docker`, `xcode`, `python`, `kubectl` — detected from PATH |
| Agent skills | `.claude/skills/*.md` — parsed for `requires` to determine executability |
| Custom skills | Declared in `skills.yaml` |

```bash
$ claw-mesh skills
SKILL              TYPE          CATEGORY    NODES
docker             tool          -           linux-gpu
golang             tool          -           mac-mini, linux-gpu
ios-build          agent-skill   -           mac-mini
python             tool          -           linux-gpu, mac-mini, pi-home
sd-xl              custom        image-gen   linux-gpu
```

## Node status

Nodes have three states:

- **online** — healthy, accepting messages
- **busy** — processing a task (still accepts messages but deprioritized in routing)
- **offline** — missed heartbeat threshold (default 90s)

## Custom skills

Declare additional capabilities in `skills.yaml`:

```yaml
skills:
  - name: stable-diffusion
    type: custom
    category: image-gen
    description: "SDXL model loaded and ready"
  - name: data-pipeline
    type: custom
    category: etl
```

Pass it on join: `claw-mesh join <url> --skills-manifest ./skills.yaml`
