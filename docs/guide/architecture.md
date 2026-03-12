# Architecture

## Overview

claw-mesh uses a hub-and-spoke model: one coordinator, many nodes.

```
                          ┌──────────────────────────┐
                          │     claw-mesh coord       │
                          │                           │
                          │  Router · Registry        │
                          │  Health · Dashboard       │
                          │  Seed API                 │
                          └──┬───────┬───────┬────┬───┘
                             │       │       │    │
           ┌─────────────────┘       │       │    └──────────────────┐
           │          ┌──────────────┘       └──────────┐            │
           ▼          ▼                                 ▼            ▼
  ┌──────────────┐ ┌──────────────┐ ┌────────────────┐ ┌──────────────┐
  │ mac-mini      │ │ linux-gpu    │ │ vps-tokyo       │ │ pi-home       │
  │ darwin/arm64  │ │ linux/amd64  │ │ linux/amd64     │ │ linux/arm64   │
  │ 16GB, Metal   │ │ 64GB, A100   │ │ 4GB, public IP  │ │ 4GB           │
  │ xcode, golang │ │ docker, k8s  │ │ docker, nginx   │ │ python        │
  └──────────────┘ └──────────────┘ └────────────────┘ └──────────────┘
    Local (LAN)      Local (LAN)      Remote (WAN)       Local (LAN)
```

## Components

| Component | Role |
|-----------|------|
| **Coordinator** | Central server. Runs HTTP/WebSocket server, manages node registry, health checks, message routing, web dashboard. |
| **Node Agent** | Lightweight sidecar on each Gateway machine. Handles registration, heartbeat, capability reporting, message forwarding. |
| **Web Dashboard** | SPA embedded in the coordinator binary. Shows node topology, routing rules, message flow. |
| **CLI** | Single binary that acts as coordinator, node agent, and management tool depending on the subcommand. |

## Identity & sync model

All nodes share one AI identity but have independent local capabilities:

| Layer | Sync | Contents |
|-------|------|----------|
| Identity | Shared | SOUL.md, IDENTITY.md, agent skills |
| Memory | Auto-sync | MEMORY.md, `memory/*.md` |
| Config | Independent | `openclaw.json`, `skills.yaml` |
| Capability | Independent | Hardware detection + tool availability |

Your AI knows the same things everywhere. But each machine contributes its own strengths.

## Communication

- **Coordinator ↔ Node**: HTTP REST + WebSocket for real-time events
- **Authentication**: Bearer token on all mutating endpoints, per-node tokens generated on registration
- **Discovery**: Nodes register on join, coordinator tracks via heartbeat (default 30s interval, 90s timeout)
