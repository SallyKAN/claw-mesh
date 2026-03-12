# Config Seed

Config seed lets new nodes pull shared configuration from the coordinator, so you don't have to manually set up each machine.

## What gets synced

| Synced | Not synced |
|--------|-----------|
| AI provider API keys | Node-local ports/hostname |
| Model configuration | Channel config (Telegram tokens, etc.) |
| Identity files (SOUL.md, IDENTITY.md) | Local skills/tools |
| Memory files (MEMORY.md) | `openclaw.json` node-specific settings |

## Usage

```bash
# Join + auto-install runtime + sync config
claw-mesh join <coordinator-url> --auto-install

# Join + auto-install but skip config sync
claw-mesh join <coordinator-url> --auto-install --no-sync-config
```

## How it works

1. Node sends a registration request to the coordinator
2. Coordinator responds with a per-node token
3. Node requests seed config via `GET /api/v1/seed` (authenticated)
4. Coordinator sends shared config (API keys, identity files)
5. Node writes config to the appropriate local paths

## Security

- The seed API requires bearer token authentication
- API keys are transmitted over the network — **use HTTPS in production**
- The coordinator only shares config explicitly marked as shared in `claw-mesh.yaml`
