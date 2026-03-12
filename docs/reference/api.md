# REST API

The coordinator exposes a REST API on the configured port (default `:9180`).

All mutating endpoints require a `Authorization: Bearer <token>` header.

## Nodes

### Register node

```
POST /api/v1/nodes/register
```

Body:
```json
{
  "name": "mac-mini",
  "endpoint": "192.168.1.100:9120",
  "capabilities": {
    "os": "darwin",
    "arch": "arm64",
    "gpu": false,
    "memory_gb": 16,
    "tags": ["xcode", "homebrew"],
    "skills": ["coding-agent"]
  }
}
```

### List nodes

```
GET /api/v1/nodes
```

### Get node

```
GET /api/v1/nodes/:id
```

### Remove node

```
DELETE /api/v1/nodes/:id
```

### Heartbeat

```
POST /api/v1/nodes/:id/heartbeat
```

## Routing

### Send message (auto-route)

```
POST /api/v1/route
```

Body:
```json
{
  "message": "Generate an image of a sunset",
  "requires_gpu": true
}
```

### Send message (target node)

```
POST /api/v1/route/:nodeId
```

## Rules

### List rules

```
GET /api/v1/rules
```

### Add rule

```
POST /api/v1/rules
```

### Delete rule

```
DELETE /api/v1/rules/:id
```

## Dashboard

### Stats

```
GET /api/v1/dashboard/stats
```

## WebSocket

### Event stream

```
ws://coordinator:9180/ws/events
```

Real-time events: node online/offline, message routing, heartbeat status.
