# OpenClaw Gateway 集成设计

## 1. 概述

### 1.1 背景

claw-mesh 的 mesh 网络层已经可以工作：节点注册、心跳、能力上报、消息路由均已实现。但当前节点收到消息后只是 echo 回来（`handler.go:77`），并没有真正调用本地的 OpenClaw Gateway。

本文档设计完整的 Gateway 集成方案，打通端到端消息流，让 AI 助手真正实现跨机器协作。

### 1.2 目标

- 节点收到 coordinator 转发的消息后，调用本地 OpenClaw Gateway 处理
- 支持两阶段集成：Phase 1 用 HTTP API（简单可靠），Phase 2 用 WebSocket RPC（功能完整）
- 分析 Telegram 消息入口架构，推荐最优方案

### 1.3 OpenClaw Gateway 关键信息

| 项目 | 值 |
|------|-----|
| 默认端口 | **18789**（非 claw-mesh 当前代码中的 9120） |
| HTTP API | `/v1/chat/completions`（OpenAI 兼容），Bearer token 认证 |
| WebSocket RPC | `agent` 方法（message + idempotencyKey 必填）、`health` 方法 |
| HTTP Health 端点 | **无**（health 只能通过 WebSocket RPC 访问） |
| Auth token 来源 | `gateway.auth.token` > `OPENCLAW_GATEWAY_TOKEN` env > `CLAWDBOT_GATEWAY_TOKEN` env |

## 2. 集成架构

### 2.1 消息流总览

```
用户消息
  │
  ▼
Coordinator (路由决策)
  │
  ▼
Node Agent (handler.go)
  │
  ▼
OpenClaw Gateway (/v1/chat/completions 或 WebSocket RPC)
  │
  ▼
AI 处理 + 响应
  │
  ▼
Node Agent → Coordinator → 用户
```

### 2.2 两阶段方案

| | Phase 1: HTTP API | Phase 2: WebSocket RPC |
|---|---|---|
| 端点 | `/v1/chat/completions` | WebSocket `agent` 方法 |
| 复杂度 | 低，Go 原生 HTTP 客户端 | 中，需要 WebSocket 连接管理 |
| 功能 | 单轮对话，无状态 | Session 管理、流式事件、工具执行可见性 |
| 适用场景 | MVP / 简单消息转发 | 完整 agent 交互 |
| 优先级 | **v0.1 实现** | v0.2 实现 |

## 3. Phase 1: HTTP API 集成

### 3.1 请求/响应格式映射

claw-mesh 的 `types.Message` 需要转换为 OpenAI ChatCompletion 格式：

**请求映射：claw-mesh Message → OpenAI ChatCompletionRequest**

```go
// claw-mesh 侧
type Message struct {
    ID         string    `json:"id"`
    Content    string    `json:"content"`
    Source     string    `json:"source"`
    TargetNode string    `json:"target_node,omitempty"`
    CreatedAt  time.Time `json:"created_at"`
}

// 转换为 OpenAI 格式
type ChatCompletionRequest struct {
    Model    string              `json:"model"`
    Messages []ChatMessage       `json:"messages"`
    Stream   bool                `json:"stream,omitempty"`
}

type ChatMessage struct {
    Role    string `json:"role"`    // "user", "assistant", "system"
    Content string `json:"content"`
}
```

转换逻辑：

```go
func messageToCompletionRequest(msg *types.Message) *ChatCompletionRequest {
    return &ChatCompletionRequest{
        Model: "default",  // OpenClaw 会使用配置的默认模型
        Messages: []ChatMessage{
            {Role: "user", Content: msg.Content},
        },
    }
}
```

**响应映射：OpenAI ChatCompletionResponse → claw-mesh MessageResponse**

```go
type ChatCompletionResponse struct {
    ID      string   `json:"id"`
    Choices []Choice `json:"choices"`
}

type Choice struct {
    Index        int         `json:"index"`
    Message      ChatMessage `json:"message"`
    FinishReason string      `json:"finish_reason"`
}
```

转换逻辑：

```go
func completionResponseToMessage(msgID string, resp *ChatCompletionResponse) *types.MessageResponse {
    content := ""
    if len(resp.Choices) > 0 {
        content = resp.Choices[0].Message.Content
    }
    return &types.MessageResponse{
        MessageID: msgID,
        Response:  content,
    }
}
```

### 3.2 认证机制

OpenClaw Gateway 使用 Bearer token 认证，token 来源优先级：

1. CLI flag `--gateway-token`
2. 环境变量 `OPENCLAW_GATEWAY_TOKEN`（fallback: `CLAWDBOT_GATEWAY_TOKEN`）
3. `openclaw.json` 配置文件中的 `gateway.auth.token`
4. claw-mesh.yaml 中的 `node.gateway.token`

```go
func resolveGatewayToken(cliToken string) string {
    if cliToken != "" {
        return cliToken
    }
    if t := os.Getenv("OPENCLAW_GATEWAY_TOKEN"); t != "" {
        return t
    }
    if t := os.Getenv("CLAWDBOT_GATEWAY_TOKEN"); t != "" {
        return t
    }
    // 从 openclaw.json 读取（discovery 阶段提取）
    return ""
}
```

请求示例：

```
POST /v1/chat/completions HTTP/1.1
Host: 127.0.0.1:18789
Authorization: Bearer <token>
Content-Type: application/json

{
  "model": "default",
  "messages": [{"role": "user", "content": "帮我跑一下 Xcode build"}]
}
```

### 3.3 健康检查

OpenClaw Gateway **没有** HTTP health 端点（`/__openclaw__/health` 不存在于 HTTP 层）。

**Phase 1 方案：TCP 连接检测 + 轻量 completion 探测**

```go
// 第一层：TCP 可达性（快速）
func (c *HTTPGatewayClient) tcpHealthCheck() bool {
    conn, err := net.DialTimeout("tcp", c.endpoint, 2*time.Second)
    if err != nil {
        return false
    }
    conn.Close()
    return true
}

// 第二层：API 可用性（可选，较慢）
func (c *HTTPGatewayClient) apiHealthCheck() bool {
    req := &ChatCompletionRequest{
        Model:    "default",
        Messages: []ChatMessage{{Role: "user", Content: "ping"}},
    }
    // 设置短超时，只验证 API 能响应
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    _, err := c.sendCompletion(ctx, req)
    return err == nil
}
```

**Phase 2 方案：** 使用 WebSocket RPC 的 `health` 方法，提供真正的健康状态。

### 3.4 错误处理

| 场景 | HTTP 状态码 | 处理策略 |
|------|------------|---------|
| Gateway 不可达 | — | 返回 503，日志警告，心跳上报 status=degraded |
| 连接超时 | — | 重试 1 次（指数退避），超时后返回 504 |
| 认证失败 | 401 | 返回 502，日志错误，不重试 |
| 模型不可用 | 404 | 返回 502，透传错误信息 |
| Gateway 内部错误 | 500 | 重试 1 次，失败后返回 502 |
| 响应格式错误 | 200 但解析失败 | 返回 502，日志错误 |
| 请求体过大 | 413 | 返回 413，透传 |

重试策略：

```go
const (
    maxRetries     = 1
    retryBaseDelay = 500 * time.Millisecond
)
```

### 3.5 优雅降级

当 Gateway 未发现或不可达时，节点不应完全不可用：

```go
func (h *Handler) handleMessage(w http.ResponseWriter, r *http.Request) {
    // ... 解析消息 ...

    if h.gatewayClient == nil {
        // Gateway 未配置，echo fallback + 警告
        log.Printf("WARN: no gateway client configured, echoing message %s", msg.ID)
        resp := types.MessageResponse{
            MessageID: msg.ID,
            Response:  "[claw-mesh] Gateway not available. Message: " + msg.Content,
        }
        writeNodeJSON(w, http.StatusOK, resp)
        return
    }

    // 正常转发到 Gateway
    gwResp, err := h.gatewayClient.SendMessage(r.Context(), &msg)
    if err != nil {
        log.Printf("gateway forwarding failed for message %s: %v", msg.ID, err)
        writeNodeJSON(w, http.StatusBadGateway, map[string]string{
            "error": fmt.Sprintf("gateway error: %v", err),
        })
        return
    }

    writeNodeJSON(w, http.StatusOK, gwResp)
}
```

## 4. Phase 2: WebSocket RPC 集成

### 4.1 连接管理

OpenClaw Gateway 的 WebSocket RPC 端点位于 `ws://127.0.0.1:18789/ws`。

**连接建立流程：**

```
1. WebSocket 握手 → ws://gateway:18789/ws
2. 发送 connect frame:
   {
     "method": "connect",
     "params": {
       "token": "<auth-token>",
       "clientId": "claw-mesh-node-<nodeID>"
     }
   }
3. 收到 connected 确认
4. 开始收发 RPC 消息
```

**心跳保活：**

Gateway 会定期发送 `tick` 事件，客户端需要响应以保持连接。如果超过一定时间没有收到 tick，应视为连接断开。

**断线重连策略：**

```go
type WSGatewayClient struct {
    endpoint    string
    token       string
    conn        *websocket.Conn
    mu          sync.Mutex
    reconnectCh chan struct{}
}

// 指数退避重连，最大间隔 30s
const (
    wsReconnectBase = 1 * time.Second
    wsReconnectMax  = 30 * time.Second
)
```

### 4.2 agent RPC 方法

`agent` 是 OpenClaw Gateway 的核心 RPC 方法，用于发送消息给 AI agent 处理。

**请求参数：**

```json
{
  "method": "agent",
  "params": {
    "message": "帮我跑一下 Xcode build",
    "idempotencyKey": "msg-<uuid>",
    "agentId": "default",
    "sessionKey": "agent:default:telegram:user:12345",
    "options": {
      "stream": true
    }
  }
}
```

| 参数 | 必填 | 说明 |
|------|------|------|
| message | 是 | 用户消息内容 |
| idempotencyKey | 是 | 幂等键，防止重复处理。建议用 `msg-<claw-mesh-message-id>` |
| agentId | 否 | 指定 agent，默认 "default" |
| sessionKey | 否 | 会话标识，用于上下文连续性 |
| options.stream | 否 | 是否流式返回事件 |

**响应格式：**

```json
{
  "result": {
    "runId": "run-abc123",
    "status": "completed",
    "summary": "已执行 Xcode build，结果如下..."
  }
}
```

**流式事件（stream=true 时）：**

Gateway 会推送多个事件，每个事件有 `phase` 字段：

| phase | 说明 |
|-------|------|
| assistant | AI 助手的文本输出 |
| tool | 工具调用（执行命令、读文件等） |
| reasoning | 推理过程（思考链） |
| debug | 调试信息 |

claw-mesh Phase 2 可以选择性地将这些事件转发给 coordinator，实现实时进度展示。

### 4.3 Session 管理

OpenClaw 的 session key 格式：

```
agent:{agentId}:{provider}:{scope}:{identifier}
```

示例：
- `agent:default:telegram:user:12345` — Telegram 用户 12345 与 default agent 的会话
- `agent:coding:claw-mesh:node:mac-mini` — claw-mesh 节点 mac-mini 的 coding agent 会话

**claw-mesh 的 session key 生成策略：**

```go
func buildSessionKey(msg *types.Message, agentID string) string {
    // 使用消息来源作为 provider + scope
    // 例如 source="telegram:user:12345" → provider=telegram, scope=user, id=12345
    return fmt.Sprintf("agent:%s:%s", agentID, msg.Source)
}
```

这样可以保证同一用户在同一 agent 上的对话具有上下文连续性，即使消息经过 claw-mesh 路由到不同节点。

### 4.4 health RPC 方法

```json
// 请求
{"method": "health", "params": {}}

// 响应
{
  "result": {
    "status": "ok",
    "version": "0.5.2",
    "uptime": 3600,
    "agents": ["default", "coding"]
  }
}
```

Phase 2 中，节点心跳可以集成 Gateway health 检查结果，上报更精确的节点状态。

## 5. Telegram 消息入口架构

当前 claw-mesh 架构图中，用户消息（Telegram/Discord/CLI）直接到达 coordinator。但实际部署中，Telegram Bot 的消息入口有两种可行方案。

### 5.1 方案 A: Gateway 作为入口（推荐）

```
Telegram Bot API
      │
      ▼
OpenClaw Gateway (主节点)
  ├── 本地能力匹配 → 本地处理
  └── 需要远程能力 → claw-mesh Coordinator API → 路由到其他节点 Gateway
                                                        │
                                                        ▼
                                                   远程节点 Gateway
                                                        │
                                                        ▼
                                                   响应回传 → 主节点 → Telegram
```

**实现方式：**

OpenClaw 已有完整的 Telegram channel 适配器（基于 grammY 框架）。在主节点的 OpenClaw 中添加一个 skill/plugin，当检测到消息需要远程能力时，调用 claw-mesh coordinator 的 `POST /api/v1/route` API 转发消息。

```typescript
// OpenClaw skill 伪代码（TypeScript，运行在 OpenClaw 侧）
async function handleMessage(message: string, context: Context) {
    // 判断是否需要远程能力
    if (needsRemoteCapability(message)) {
        const resp = await fetch('http://coordinator:9180/api/v1/route', {
            method: 'POST',
            headers: { 'Authorization': 'Bearer <token>' },
            body: JSON.stringify({ content: message, source: context.source })
        });
        return resp.json();
    }
    // 否则本地处理
    return localProcess(message);
}
```

**优势：**
- 复用 OpenClaw 已有的 Telegram 适配器，零额外开发
- Telegram Bot token 管理、消息格式化、富文本渲染等全部复用
- 路由决策可以结合 AI 理解（OpenClaw 侧可以用 LLM 判断消息需要什么能力）
- claw-mesh 保持纯粹的 mesh 编排职责，不涉及 channel 适配

**劣势：**
- 主节点是单点（但可以通过 failover 缓解）
- 路由决策分散在 OpenClaw skill 和 coordinator 两处

### 5.2 方案 B: Coordinator 作为入口

```
Telegram Bot API
      │
      ▼
claw-mesh Coordinator (Telegram 适配层)
      │
      ▼
路由决策
  ├── 节点 A Gateway
  └── 节点 B Gateway
```

**实现方式：**

在 coordinator 中新增 Telegram Bot 适配层，直接接收 Telegram webhook/polling，解析消息后路由到合适的节点。

**优势：**
- 统一入口，路由逻辑完全集中在 coordinator
- 无单点问题（coordinator 本身可以做 HA）
- 消息流更简洁

**劣势：**
- 需要在 Go 侧重新实现 Telegram Bot 适配器（grammY 是 TypeScript/Deno）
- 需要处理 Telegram 消息格式、富文本、inline keyboard 等复杂逻辑
- 与 OpenClaw 的 channel 系统重复建设
- 增加 coordinator 的复杂度，偏离"纯编排"定位

### 5.3 推荐与对比分析

| 维度 | 方案 A: Gateway 入口 | 方案 B: Coordinator 入口 |
|------|---------------------|------------------------|
| 开发量 | 低（写一个 OpenClaw skill） | 高（Go 实现 Telegram 适配器） |
| 复用性 | 高（复用 OpenClaw channel 生态） | 低（重新实现） |
| 架构纯粹性 | 高（claw-mesh 只做编排） | 中（coordinator 承担 channel 职责） |
| 单点风险 | 主节点是单点 | coordinator 是单点（但更容易 HA） |
| 路由智能性 | 高（可用 LLM 辅助判断） | 中（基于规则匹配） |
| 扩展到其他 channel | 自动支持（OpenClaw 支持的都支持） | 每个 channel 都要重新实现 |

**推荐方案 A**，理由：

1. OpenClaw 已经有成熟的 Telegram 适配器，没必要在 Go 侧重写
2. claw-mesh 的定位是 mesh 编排层，不应该承担 channel 适配职责
3. 方案 A 天然支持 OpenClaw 未来新增的任何 channel（Discord、Slack、Web 等）
4. 主节点单点问题可以通过 v0.2 的 failover 机制解决（备用节点自动接管 Telegram Bot）

## 6. 配置设计

### 6.1 claw-mesh.yaml 新增字段

```yaml
node:
  name: "mac-mini"
  tags: ["xcode", "homebrew"]
  endpoint: "192.168.1.100:9121"

  # 新增：Gateway 配置
  gateway:
    endpoint: "127.0.0.1:18789"   # OpenClaw Gateway 地址（默认 127.0.0.1:18789）
    token: ""                      # Gateway auth token（优先级低于 env var）
    timeout: 120                   # 请求超时秒数（默认 120，AI 处理可能较慢）
    auto_discover: true            # 是否自动发现本地 Gateway（默认 true）
```

对应 Go 结构体：

```go
// config.go 新增
type GatewayConfig struct {
    Endpoint     string `json:"endpoint" yaml:"endpoint" mapstructure:"endpoint"`
    Token        string `json:"token" yaml:"token" mapstructure:"token"`
    Timeout      int    `json:"timeout" yaml:"timeout" mapstructure:"timeout"`
    AutoDiscover bool   `json:"auto_discover" yaml:"auto_discover" mapstructure:"auto_discover"`
}

// NodeConfig 修改
type NodeConfig struct {
    Name     string        `json:"name" yaml:"name" mapstructure:"name"`
    Tags     []string      `json:"tags" yaml:"tags" mapstructure:"tags"`
    Endpoint string        `json:"endpoint" yaml:"endpoint" mapstructure:"endpoint"`
    Gateway  GatewayConfig `json:"gateway" yaml:"gateway" mapstructure:"gateway"`
}
```

### 6.2 自动发现增强

当前 `discovery.go` 存在的问题和修复方案：

| 问题 | 修复 |
|------|------|
| 默认端口 9120（错误） | 改为 18789 |
| 不提取 auth token | 从 `openclaw.json` 的 `gateway.auth.token` 提取 |
| 只做 TCP 检测 | 保留 TCP 检测（Phase 1 足够），Phase 2 加 WebSocket health |

`GatewayInfo` 结构体增强：

```go
type GatewayInfo struct {
    Endpoint string `json:"endpoint"`
    Version  string `json:"version"`
    Token    string `json:"token"`  // 新增：从 openclaw.json 提取的 auth token
}
```

`discoverFromConfig` 修改要点：

```go
var cfg struct {
    Gateway struct {
        Host string `json:"host"`
        Port int    `json:"port"`
        Auth struct {
            Token string `json:"token"`
        } `json:"auth"`
    } `json:"gateway"`
    Version string `json:"version"`
}

// 默认端口修复
port := cfg.Gateway.Port
if port == 0 {
    port = 18789  // 修复：9120 → 18789
}

return &GatewayInfo{
    Endpoint: fmt.Sprintf("%s:%d", host, port),
    Version:  cfg.Version,
    Token:    cfg.Gateway.Auth.Token,  // 新增
}, nil
```

### 6.3 CLI 新增 flags

`claw-mesh join` 命令新增：

```
--gateway-endpoint string   OpenClaw Gateway endpoint (default: auto-discover)
--gateway-token string      OpenClaw Gateway auth token (default: from env/config)
--gateway-timeout int       Gateway request timeout in seconds (default: 120)
```

## 7. 代码变更清单

### 新增文件

| 文件 | 说明 |
|------|------|
| `internal/node/gateway_client.go` | GatewayClient 接口 + HTTPGatewayClient 实现 |

```go
// gateway_client.go 核心接口
type GatewayClient interface {
    SendMessage(ctx context.Context, msg *types.Message) (*types.MessageResponse, error)
    HealthCheck(ctx context.Context) bool
    Close() error
}

type HTTPGatewayClient struct {
    endpoint string
    token    string
    timeout  time.Duration
    client   *http.Client
}
```

### 修改文件

| 文件 | 变更 |
|------|------|
| `internal/node/handler.go` | Handler 新增 `gatewayClient` 字段；`handleMessage` 替换 echo 为 gateway 转发；NewHandler 接受 GatewayClient 参数 |
| `internal/node/agent.go` | NewAgent 创建 GatewayClient 并传入 Handler；AgentConfig 新增 Gateway 相关字段 |
| `internal/node/discovery.go` | 默认端口 9120→18789；GatewayInfo 新增 Token 字段；从 openclaw.json 提取 auth token |
| `internal/config/config.go` | NodeConfig 新增 GatewayConfig 嵌套结构体 |
| `internal/types/types.go` | 新增 OpenAI 兼容类型（ChatCompletionRequest/Response 等） |
| `cmd/claw-mesh/main.go` | `join` 命令新增 `--gateway-endpoint`、`--gateway-token`、`--gateway-timeout` flags |

### handler.go 关键变更

```go
// Before
type Handler struct {
    token *string
    mux   *http.ServeMux
}

func NewHandler(token *string) *Handler { ... }

// After
type Handler struct {
    token         *string
    gatewayClient GatewayClient  // 新增
    mux           *http.ServeMux
}

func NewHandler(token *string, gw GatewayClient) *Handler { ... }
```

### agent.go 关键变更

```go
// AgentConfig 新增
type AgentConfig struct {
    // ... 现有字段 ...
    GatewayEndpoint string
    GatewayToken    string
    GatewayTimeout  int
}

// StartHandler 修改
func (a *Agent) StartHandler() error {
    var gw GatewayClient
    if a.gatewayEndpoint != "" {
        gw = NewHTTPGatewayClient(a.gatewayEndpoint, a.gatewayToken, a.gatewayTimeout)
    }
    handler := NewHandler(&a.token, gw)
    // ...
}
```

## 8. 测试验证

### 8.1 单元测试

**gateway_client_test.go** — mock HTTP server 模拟 OpenAI 响应：

```go
func TestHTTPGatewayClient_SendMessage(t *testing.T) {
    // 启动 mock server，返回 OpenAI 格式响应
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 验证 Authorization header
        // 验证请求体格式
        // 返回 ChatCompletionResponse
        json.NewEncoder(w).Encode(ChatCompletionResponse{
            ID: "chatcmpl-123",
            Choices: []Choice{{
                Message:      ChatMessage{Role: "assistant", Content: "Build succeeded"},
                FinishReason: "stop",
            }},
        })
    }))
    defer srv.Close()

    client := NewHTTPGatewayClient(srv.URL, "test-token", 30)
    resp, err := client.SendMessage(context.Background(), &types.Message{
        ID:      "msg-1",
        Content: "run xcode build",
    })
    assert.NoError(t, err)
    assert.Equal(t, "Build succeeded", resp.Response)
}
```

### 8.2 集成测试

**handler_test.go** — mock GatewayClient 测试 handler 完整流程：

```go
type mockGatewayClient struct {
    response *types.MessageResponse
    err      error
}

func (m *mockGatewayClient) SendMessage(ctx context.Context, msg *types.Message) (*types.MessageResponse, error) {
    return m.response, m.err
}

func TestHandler_WithGateway(t *testing.T) {
    mock := &mockGatewayClient{
        response: &types.MessageResponse{MessageID: "msg-1", Response: "done"},
    }
    handler := NewHandler(nil, mock)
    // 发送请求，验证响应来自 gateway 而非 echo
}

func TestHandler_WithoutGateway_Fallback(t *testing.T) {
    handler := NewHandler(nil, nil)
    // 发送请求，验证 echo fallback 行为
}
```

### 8.3 端到端验证步骤

```bash
# 1. 启动 OpenClaw Gateway（确认端口 18789）
cd ~/openclaw && npm start

# 2. 启动 coordinator
claw-mesh up --token mysecret

# 3. 加入节点（自动发现 Gateway）
claw-mesh join http://localhost:9180 --name "local-dev" --token mysecret

# 4. 验证节点状态（应显示 gateway: connected）
claw-mesh status

# 5. 发送测试消息
claw-mesh send --auto "hello, what can you do?"

# 6. 验证响应来自 AI（非 echo）
# 预期：收到 AI 助手的实际回复，而非原文回显
```
