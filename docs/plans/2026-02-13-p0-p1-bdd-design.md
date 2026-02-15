# claw-mesh 竞品对齐 & BDD 验收标准

> 日期: 2026-02-13
> 状态: 设计稿

## 1. 竞品分析

claw-mesh 定位：**AI Agent 多机编排器**。竞品分三层：

| 层级 | 竞品 | 核心能力 | claw-mesh 对标点 |
|------|------|----------|-----------------|
| 基础设施 | Consul, Traefik, Envoy | 服务发现、健康检查、负载均衡、动态路由 | 节点注册、心跳、能力路由 |
| AI 编排 | LangGraph, CrewAI, AutoGen | 多 Agent 协作、任务分发、工作流 | 跨机器 Agent 路由、能力匹配 |
| 同类直接竞品 | 无（蓝海） | — | claw-mesh 是首个面向 OpenClaw 的多机编排器 |

### 竞品核心能力 vs claw-mesh 现状

| 能力 | Consul | Traefik | claw-mesh 现状 | 差距 |
|------|--------|---------|---------------|------|
| 节点注册/发现 | ✅ Agent 自动注册 | ✅ Provider 自动发现 | ⚠️ 手动 join | 缺自动发现 |
| 健康检查 | ✅ 多种检查方式(HTTP/TCP/Script) | ✅ 后端健康检查 | ⚠️ 仅心跳超时 | 缺主动探测 |
| 负载均衡 | ✅ 多策略 | ✅ WRR/Mirror/Sticky | ⚠️ 仅 least-busy | 缺策略 |
| 动态配置 | ✅ KV Store + Watch | ✅ 热加载 | ❌ 无 | 需要 |
| TLS/mTLS | ✅ Connect + mTLS | ✅ Let's Encrypt 自动 | ❌ 仅 Bearer Token | 需要 |
| Web UI | ✅ 完整 UI | ✅ Dashboard | ⚠️ 基础 Dashboard | 需增强 |
| 配置持久化 | ✅ Raft 共识 | ✅ 文件/etcd/consul | ❌ 内存态 | 需要 |
| 优雅重启 | ✅ | ✅ 零停机 | ❌ | 需要 |
| 指标/可观测 | ✅ Prometheus/StatsD | ✅ Prometheus/OpenTelemetry | ❌ | 需要 |

## 2. P0 功能（Must Ship — 阻塞发布）

### P0-1: 端到端消息路由可用
当前状态：代码存在但未经端到端验证。coordinator 启动 → node join → 发消息 → 收到响应，这条链路必须跑通。

### P0-2: 配置持久化
当前状态：所有状态在内存，重启全丢。节点注册、路由规则必须持久化到磁盘。

### P0-3: 优雅重启 & 节点重连
当前状态：coordinator 重启后节点全部丢失。节点应自动重连，coordinator 应恢复状态。

### P0-4: 主动健康检查
当前状态：仅被动心跳超时。coordinator 应主动 HTTP 探测节点存活。

### P0-5: init 命令生成配置
当前状态：`claw-mesh init` 未实现。应生成默认配置文件 + token。

## 3. P1 功能（Should Ship — 提升竞争力）

### P1-1: 多路由策略
当前仅 least-busy。需增加：round-robin、weighted、sticky（按 source 固定节点）。

### P1-2: Prometheus 指标
暴露 /metrics 端点：节点数、消息吞吐、路由延迟、错误率。

### P1-3: WebSocket 实时事件流
Dashboard 和 CLI 可订阅实时事件（节点上下线、消息路由日志）。

### P1-4: TLS 支持
coordinator ↔ node 通信支持 TLS。`claw-mesh init` 可选生成自签证书。

### P1-5: Dashboard 增强
实时事件流、消息历史、节点拓扑图、路由规则编辑器。

### P1-6: 节点标签热更新
节点运行中可更新 tags/capabilities，无需重新 join。

## 4. BDD 验收标准

以下用 Given/When/Then 格式描述，每个场景对应一个可执行的终端操作序列。

---

### P0-1: 端到端消息路由

#### Scenario 1: 单节点消息回显
```gherkin
Feature: 端到端消息路由

  Scenario: coordinator 启动 → node join → 发消息 → 收到响应
    Given 用户在终端 A 执行:
      """
      claw-mesh up --port 9180 --token secret123 --allow-private
      """
    And 终端 A 输出包含 "coordinator listening on :9180"
    
    When 用户在终端 B 执行:
      """
      claw-mesh join http://127.0.0.1:9180 --name test-node --token secret123 --listen :9121
      """
    Then 终端 B 输出包含 "joining mesh at http://127.0.0.1:9180"
    And 终端 A 日志包含 "node registered: " 和 "(test-node)"
    
    When 用户在终端 C 执行:
      """
      claw-mesh send --auto "hello world" --coordinator http://127.0.0.1:9180 --token secret123
      """
    Then 终端 C 输出包含 "Message" 和 "routed to node"
    And 终端 C 输出包含 "Response: hello world"

  Scenario: 指定节点发送消息
    Given coordinator 运行中，test-node 已注册
    When 用户执行:
      """
      claw-mesh send --node test-node "ping" --coordinator http://127.0.0.1:9180 --token secret123
      """
    Then 输出包含 "routed to node" 和 "Response: ping"

  Scenario: 无节点时自动路由失败
    Given coordinator 运行中，无节点注册
    When 用户执行:
      """
      claw-mesh send --auto "hello" --coordinator http://127.0.0.1:9180 --token secret123
      """
    Then 退出码非 0
    And 输出包含 "503" 或 "no available nodes"
```

#### Scenario 2: 多节点能力路由
```gherkin
  Scenario: GPU 任务路由到 GPU 节点
    Given coordinator 运行中
    And node-a 已注册，tags 包含 "gpu"
    And node-b 已注册，tags 不包含 "gpu"
    And 路由规则: match=gpu:true target=node-a
    When 用户执行:
      """
      claw-mesh send --auto "render image" --coordinator http://127.0.0.1:9180 --token secret123
      """
    Then 消息被路由到 node-a（输出包含 node-a 的 ID）
```

---

### P0-2: 配置持久化

```gherkin
Feature: 配置持久化

  Scenario: 路由规则持久化
    Given coordinator 运行中
    When 用户添加路由规则:
      """
      claw-mesh route add --match "os:linux" --target linux-box --coordinator http://127.0.0.1:9180 --token secret123
      """
    And 用户 Ctrl+C 停止 coordinator
    And 用户重新启动 coordinator:
      """
      claw-mesh up --port 9180 --token secret123
      """
    When 用户查看路由规则:
      """
      claw-mesh route list --coordinator http://127.0.0.1:9180 --token secret123
      """
    Then 输出包含 "os:linux" 和 "linux-box"

  Scenario: 节点信息不持久化（节点需重新 join）
    Given coordinator 重启后
    When 用户执行:
      """
      claw-mesh nodes --coordinator http://127.0.0.1:9180
      """
    Then 输出为 "No nodes registered."
    # 节点是动态的，重启后需重新注册，这是正确行为
```

---

### P0-3: 优雅重启 & 节点重连

```gherkin
Feature: 优雅重启

  Scenario: coordinator 优雅关闭
    Given coordinator 运行中，1 个节点已注册
    When 用户向 coordinator 发送 SIGTERM
    Then coordinator 日志包含 "shutting down"
    And coordinator 进程在 5 秒内退出
    And 退出码为 0

  Scenario: 节点自动重连
    Given coordinator 运行中，node-a 已注册并心跳正常
    When coordinator 重启（SIGTERM → 重新 claw-mesh up）
    Then node-a 检测到心跳失败
    And node-a 自动重新注册到 coordinator
    And 用户执行 claw-mesh nodes 可看到 node-a 状态为 "online"

  Scenario: 节点优雅退出
    Given node-a 运行中
    When 用户向 node-a 发送 SIGTERM
    Then node-a 向 coordinator 发送 deregister 请求
    And coordinator 日志包含 "node deregistered"
    And claw-mesh nodes 不再显示 node-a
```

---

### P0-4: 主动健康检查

```gherkin
Feature: 主动健康检查

  Scenario: 节点进程崩溃被检测
    Given coordinator 运行中，node-a 已注册
    When node-a 进程被 kill -9（无法发送 deregister）
    Then 30 秒内 coordinator 将 node-a 标记为 "offline"
    And claw-mesh nodes 显示 node-a 状态为 "offline"
    And 后续消息不会路由到 node-a

  Scenario: 主动 HTTP 探测
    Given coordinator 配置 health_check.mode = "active"
    And node-a 已注册，endpoint 为 127.0.0.1:9121
    When coordinator 每 10 秒 GET http://127.0.0.1:9121/healthz
    And node-a 返回 200
    Then node-a 保持 "online" 状态
    
    When node-a 的 /healthz 开始返回 503
    Then coordinator 在 2 次失败后将 node-a 标记为 "offline"
```

---

### P0-5: init 命令

```gherkin
Feature: init 命令

  Scenario: 生成默认配置
    Given 当前目录没有 claw-mesh.yaml
    When 用户执行:
      """
      claw-mesh init
      """
    Then 当前目录生成 claw-mesh.yaml
    And 文件包含 coordinator.port: 9180
    And 文件包含一个随机生成的 coordinator.token（32 字符 hex）
    And 输出 "Config written to claw-mesh.yaml"

  Scenario: 不覆盖已有配置
    Given 当前目录已有 claw-mesh.yaml
    When 用户执行:
      """
      claw-mesh init
      """
    Then 退出码非 0
    And 输出包含 "already exists"
    And 原文件内容不变
```

---

### P1-1: 多路由策略

```gherkin
Feature: 多路由策略

  Scenario: Round-robin 策略
    Given coordinator 运行中
    And node-a 和 node-b 都在线
    And 路由规则: match=wildcard strategy=round-robin
    When 用户连续发送 4 条消息
    Then 消息交替路由到 node-a 和 node-b（各 2 条）

  Scenario: Weighted 策略
    Given coordinator 运行中
    And node-a (weight=3) 和 node-b (weight=1) 都在线
    And 路由规则: match=wildcard strategy=weighted
    When 用户发送 100 条消息
    Then 约 75% 路由到 node-a，约 25% 路由到 node-b（±10% 容差）
```

---

### P1-2: Prometheus 指标

```gherkin
Feature: Prometheus 指标

  Scenario: 指标端点可用
    Given coordinator 运行中
    When 用户执行:
      """
      curl -s http://127.0.0.1:9180/metrics
      """
    Then 响应包含 "claw_mesh_nodes_total"
    And 响应包含 "claw_mesh_messages_routed_total"
    And 响应包含 "claw_mesh_route_duration_seconds"
    And HTTP 状态码为 200
```

---

### P1-3: WebSocket 实时事件

```gherkin
Feature: WebSocket 实时事件

  Scenario: 订阅节点上下线事件
    Given coordinator 运行中
    And 用户通过 wscat 连接:
      """
      wscat -c ws://127.0.0.1:9180/ws/events
      """
    When 一个新节点 join
    Then WebSocket 收到 JSON 消息，type="node.registered"，包含节点信息
    
    When 该节点 SIGTERM 退出
    Then WebSocket 收到 JSON 消息，type="node.deregistered"
```

---

### P1-4: TLS 支持

```gherkin
Feature: TLS 支持

  Scenario: 自签证书启动
    Given 用户执行:
      """
      claw-mesh init --tls
      """
    Then 生成 claw-mesh.yaml + tls/cert.pem + tls/key.pem
    
    When 用户执行:
      """
      claw-mesh up
      """
    Then coordinator 在 HTTPS 上监听
    And 日志包含 "coordinator listening on :9180 (TLS)"

  Scenario: 节点通过 TLS 连接
    Given coordinator 以 TLS 模式运行
    When 用户执行:
      """
      claw-mesh join https://127.0.0.1:9180 --name secure-node --token secret --ca tls/cert.pem
      """
    Then 节点成功注册
    And 心跳通过 HTTPS 发送
```

---

## 5. 验收执行脚本框架

每个 BDD 场景对应一个 shell 测试脚本，放在 `tests/acceptance/` 目录：

```
tests/acceptance/
├── test_e2e_single_node.sh      # P0-1 Scenario 1
├── test_e2e_multi_node.sh       # P0-1 Scenario 2
├── test_persistence.sh          # P0-2
├── test_graceful_restart.sh     # P0-3
├── test_health_check.sh         # P0-4
├── test_init_command.sh         # P0-5
├── test_round_robin.sh          # P1-1
├── test_prometheus.sh           # P1-2
├── test_websocket.sh            # P1-3
├── test_tls.sh                  # P1-4
└── lib/
    ├── setup.sh                 # 启动 coordinator + node 的公共函数
    ├── teardown.sh              # 清理进程和临时文件
    └── assert.sh                # 断言函数 (assert_contains, assert_exit_code, etc.)
```

### 示例: test_e2e_single_node.sh

```bash
#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "$0")/lib/setup.sh"
source "$(dirname "$0")/lib/assert.sh"

COORD_PORT=19180
NODE_PORT=19121
TOKEN="test-token-$(date +%s)"
BINARY="./bin/claw-mesh"

cleanup() { kill $COORD_PID $NODE_PID 2>/dev/null; rm -f /tmp/coord.log /tmp/node.log; }
trap cleanup EXIT

# Given: coordinator 启动
$BINARY up --port $COORD_PORT --token "$TOKEN" --allow-private > /tmp/coord.log 2>&1 &
COORD_PID=$!
sleep 2
assert_contains /tmp/coord.log "coordinator listening on :$COORD_PORT"

# When: node join
$BINARY join "http://127.0.0.1:$COORD_PORT" \
  --name test-node --token "$TOKEN" --listen ":$NODE_PORT" > /tmp/node.log 2>&1 &
NODE_PID=$!
sleep 2
assert_contains /tmp/coord.log "node registered:"

# Then: 发消息并验证响应
OUTPUT=$($BINARY send --auto "hello world" \
  --coordinator "http://127.0.0.1:$COORD_PORT" --token "$TOKEN" 2>&1)
assert_contains_str "$OUTPUT" "routed to node"
assert_contains_str "$OUTPUT" "Response: hello world"

echo "✅ P0-1 Scenario 1: PASSED"
```

## 6. 实施优先级

| 优先级 | 功能 | 预估工作量 | 依赖 |
|--------|------|-----------|------|
| P0-5 | init 命令 | 0.5 天 | 无 |
| P0-1 | 端到端验证 + 修复 | 1 天 | P0-5 |
| P0-2 | 配置持久化 | 1 天 | 无 |
| P0-3 | 优雅重启 + 节点重连 | 1.5 天 | P0-2 |
| P0-4 | 主动健康检查 | 1 天 | 无 |
| P1-2 | Prometheus 指标 | 0.5 天 | 无 |
| P1-1 | 多路由策略 | 1 天 | 无 |
| P1-3 | WebSocket 事件 | 1 天 | 无 |
| P1-6 | 节点标签热更新 | 0.5 天 | 无 |
| P1-4 | TLS 支持 | 1.5 天 | P0-5 |
| P1-5 | Dashboard 增强 | 2 天 | P1-3 |

总计：P0 约 5 天，P1 约 6.5 天。

