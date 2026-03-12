# Routing

## How routing works

When a message arrives at the coordinator, it evaluates routing rules top-to-bottom. The first matching rule determines the target node(s).

## Routing rules

Rules are defined in `claw-mesh.yaml`. All conditions within a rule are AND:

```yaml
routing_rules:
  # GPU tasks → specific node
  - match: { requires_gpu: true }
    target: linux-gpu

  # Need both docker AND python → least busy node that has both
  - match: { requires_skills: [docker, python] }
    strategy: least-busy

  # Need xcode OR docker → least busy node that has either
  - match: { requires_any_skill: [xcode, docker] }
    strategy: least-busy

  # Default fallback
  - match: { wildcard: true }
    strategy: least-busy
```

## Strategies

| Strategy | Behavior |
|----------|----------|
| `least-busy` | Routes to the node with the fewest active tasks among matching nodes |
| `round-robin` | Cycles through matching nodes sequentially |
| `target: <name>` | Always routes to a specific named node |

## Managing rules via CLI

```bash
claw-mesh route list
claw-mesh route add --match "gpu:true" --target linux-gpu
claw-mesh route add --match "skills:docker,python" --strategy least-busy
claw-mesh route add --match "any-skill:xcode,docker" --strategy least-busy
```

## Failover

If the target node is offline, the coordinator automatically tries the next matching node. If no nodes match, the message is queued until a capable node comes online (with a configurable timeout).
