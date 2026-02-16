# claw-mesh 设计文档

> 版本: v0.2-dev | 更新日期: 2026-02-16

## 1. 项目定位

claw-mesh 是一个面向 OpenClaw 的多网关编排器（Multi-Gateway Orchestrator）。它解决的核心问题是：当你有多台机器运行 OpenClaw Gateway 时，如何统一管理、智能路由、健康监控。

### 1.1 核心场景

- 家里有 Mac Mini + Linux 服务器，想让 GPU 任务跑 Linux，日常任务跑 Mac
- 团队有多台开发机，想按能力（Python/Go/GPU）自动分配任务
- 需要高可用：一台机器挂了，任务自动切到其他节点

### 1.2 架构概览

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   用户/CLI   │────▶│ Coordinator │────▶│   Node A    │
│             │     │  (调度中心)   │     │ (OpenClaw)  │
└─────────────┘     │             │     └─────────────┘
                    │  - 节点注册   │     ┌─────────────┐
                    │  - 能力路由   │────▶│   Node B    │
                    │  - 健康检查   │     │ (OpenClaw)  │
                    │  - Web 面板   │     └─────────────┘
                    └─────────────┘
```

**Coordinator（调度中心）**：接收消息，根据路由规则分发到合适的节点。
**Node（节点）**：运行 OpenClaw Gateway 的机器，向 Coordinator 注册并接收任务。

## 2. 核心模块设计

### 2.1 节点注册与发现

节点通过 `claw-mesh join` 命令注册到 Coordinator：

1. 节点启动本地 HTTP Handler（接收转发的消息）
2. 节点向 Coordinator 发送注册请求（名称、端点、能力）
3. Coordinator 生成 per-node token 返回给节点
4. 节点开始定期心跳（每 10 秒）

**能力自动检测**：节点注册时自动检测 OS、架构、GPU、内存、已安装的技能（Go/Python/Node.js 等）。

### 2.2 路由引擎

路由规则按优先级顺序匹配：

```yaml
# 规则示例
- match: { requires_gpu: true }
  target: linux-gpu        # 指定节点

- match: { requires_os: darwin }
  strategy: least-busy     # 最空闲的 macOS 节点

- match: { wildcard: true }
  strategy: least-busy     # 兜底：最空闲的任意节点
```

**匹配条件**：`requires_gpu`、`requires_os`、`requires_skill`、`wildcard`
**路由策略**：`least-busy`（当前唯一，P1 将增加 round-robin、weighted）

### 2.3 消息转发

```
CLI send → Coordinator → Router 选节点 → Forwarder 转发 → Node Handler
                                                              ↓
CLI ← Coordinator ← 响应 ←──────────────────────── Node Handler
```

- 转发失败自动重试（3 次，指数退避：100ms/200ms/400ms）
- 502/503 视为瞬态错误，自动重试
- 网络错误（连接拒绝、超时）也自动重试

### 2.4 健康检查（双模式）

**被动模式**：节点每 10 秒发心跳，30 秒无心跳标记 offline。
**主动模式**（P0-4 新增）：Coordinator 每 10 秒 GET 节点的 `/healthz`，连续 2 次失败标记 offline。

两种模式同时运行，任一检测到异常都会标记节点离线。

### 2.5 节点自动重连（P0-3）

当 Coordinator 重启后：
1. 节点心跳开始失败
2. 连续 3 次失败后触发重连
3. 节点重置 token 为初始 admin token
4. 重新发送注册请求
5. 获取新的 per-node token，恢复正常心跳

### 2.6 配置持久化（P0-2）

- 路由规则持久化到 `~/.claw-mesh/rules.json`（可通过 `--data-dir` 自定义）
- 原子写入：写临时文件 → fsync → rename，防止崩溃损坏
- 节点信息不持久化（节点是动态的，重启后需重新注册）

### 2.7 认证模型

```
Admin Token（全局）
  ├── 用于 CLI → Coordinator 的 API 调用
  ├── 用于节点注册
  └── 由 claw-mesh init 随机生成

Per-Node Token（每节点独立）
  ├── 注册时由 Coordinator 生成并返回
  ├── 用于 Coordinator → Node 的消息转发
  └── 防止节点间互相冒充
```

## 3. API 参考

### 3.1 节点管理

| 方法 | 路径 | 认证 | 说明 |
|------|------|------|------|
| POST | /api/v1/nodes/register | Bearer Token | 注册节点 |
| DELETE | /api/v1/nodes/{id} | Bearer Token | 注销节点 |
| GET | /api/v1/nodes | 无 | 列出所有节点 |
| GET | /api/v1/nodes/{id} | 无 | 获取单个节点 |
| POST | /api/v1/nodes/{id}/heartbeat | Bearer Token | 心跳 |

### 3.2 消息路由

| 方法 | 路径 | 认证 | 说明 |
|------|------|------|------|
| POST | /api/v1/route | Bearer Token | 自动路由消息 |
| POST | /api/v1/route/{nodeId} | Bearer Token | 指定节点路由 |

### 3.3 路由规则

| 方法 | 路径 | 认证 | 说明 |
|------|------|------|------|
| GET | /api/v1/rules | 无 | 列出路由规则 |
| POST | /api/v1/rules | Bearer Token | 添加规则 |
| DELETE | /api/v1/rules/{id} | Bearer Token | 删除规则 |

### 3.4 节点端

| 方法 | 路径 | 认证 | 说明 |
|------|------|------|------|
| POST | /api/v1/messages | Per-Node Token | 接收转发消息 |
| GET | /healthz | 无 | 健康检查探针 |
