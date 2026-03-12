# What is claw-mesh?

claw-mesh is a lightweight orchestrator that connects multiple [OpenClaw](https://github.com/openclaw/openclaw) instances into a single mesh network. Messages auto-route to the right machine based on what each node can do.

## The problem

You run OpenClaw on multiple machines, but they're completely isolated:

- Your **Mac** has Xcode, Apple Notes, and local files — but no GPU.
- Your **Linux server** has an A100 GPU and Docker — but can't access your Mac apps.
- Your **VPS** has a public IP — but limited compute.

When you ask your AI assistant to "generate an image", the Mac can't use the GPU. When you ask "check my notes", the Linux server can't access Apple Notes.

## The solution

claw-mesh adds a coordinator layer that connects all your OpenClaw gateways:

```
              ┌──────────────────────┐
              │   claw-mesh coord    │
              │                      │
              │  Router · Registry   │
              │  Health · Dashboard  │
              └───┬──────────┬───────┘
                  │          │
       ┌──────────┘          └──────────┐
       ▼                                ▼
┌──────────────┐              ┌──────────────┐
│  Mac Mini    │              │  Linux GPU   │
│  xcode, swift│              │  docker, k8s │
└──────────────┘              └──────────────┘
```

- Each node registers its capabilities (OS, GPU, tools, skills) with the coordinator.
- The coordinator routes incoming messages to the best-fit node.
- If a node goes down, traffic fails over to another capable node.
- New nodes can auto-provision with `join --auto-install`.

## What is OpenClaw?

[OpenClaw](https://github.com/openclaw/openclaw) is an open-source personal AI assistant platform with 247K+ stars. It runs a Gateway on your machine that connects to LLMs (GPT, Claude, etc.) and messaging channels (Telegram, Discord, Slack). claw-mesh doesn't replace OpenClaw — it connects multiple OpenClaw instances together.

## Design goals

- **Single binary** — one `claw-mesh` binary (~13 MB) handles coordinator, node agent, and CLI.
- **Zero config on new nodes** — `claw-mesh join <url> --auto-install` does everything.
- **Non-invasive** — claw-mesh sits alongside OpenClaw, not inside it. No patches or forks needed.
- **Self-hosted** — your data stays on your machines. No cloud dependency.
