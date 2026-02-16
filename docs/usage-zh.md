# claw-mesh 使用指南

> 版本: v0.2-dev | 更新日期: 2026-02-16

## 前置条件

claw-mesh 编排的是 [OpenClaw](https://github.com/openclaw/openclaw) Gateway。每台要加入 mesh 的机器都需要先安装 OpenClaw。

```bash
# 安装 OpenClaw（需要 Node ≥22）
npm install -g openclaw@latest

# 运行配置向导（自动配置 Gateway、工作区、频道）
openclaw onboard --install-daemon
```

向导会引导你完成所有配置。详细指南：[Getting Started](https://docs.openclaw.ai/start/getting-started)

## 快速开始

### 1. 安装

```bash
# 从源码构建
git clone https://github.com/SallyKAN/claw-mesh.git
cd claw-mesh
go build -o claw-mesh ./cmd/claw-mesh
```

### 2. 初始化配置

```bash
claw-mesh init
# 输出: Config written to claw-mesh.yaml
```

生成的 `claw-mesh.yaml`：

```yaml
coordinator:
  port: 9180
  token: a1b2c3d4...  # 随机生成的 32 位 hex token
  allow_private: true
node:
  name: your-hostname
  tags: []
tls:
  enabled: false
```

### 3. 启动 Coordinator

```bash
# 在调度中心机器上运行
claw-mesh up
# 输出: coordinator listening on :9180
```

常用参数：
- `--port 9180` — 监听端口
- `--token <token>` — 覆盖配置文件中的 token
- `--allow-private` — 允许私有 IP 注册（局域网必须）
- `--data-dir <path>` — 数据目录（默认 ~/.claw-mesh）

### 4. 加入节点

```bash
# 在每台 OpenClaw 机器上运行
claw-mesh join http://coordinator-ip:9180 \
  --name my-mac \
  --token <同一个token>
```

常用参数：
- `--name <名称>` — 节点名称（默认主机名）
- `--token <token>` — Coordinator 的 admin token
- `--listen :9121` — 本地 handler 监听地址
- `--tags gpu,python` — 自定义标签
- `--no-gateway` — 禁用 OpenClaw Gateway 自动发现（测试用）

### 5. 发送消息

```bash
# 自动路由（选最空闲的节点）
claw-mesh send --auto "你好，帮我写个脚本"

# 指定节点
claw-mesh send --node my-mac "在 Mac 上编译这个项目"
```

### 6. 管理节点

```bash
# 查看所有节点
claw-mesh nodes

# 输出示例：
# ID                     NAME    STATUS  ENDPOINT         OS/ARCH       GPU  SKILLS
# node-a1b2c3d4e5f6g7h8  my-mac  online  192.168.1.10:9121  darwin/arm64  yes  golang,python,nodejs

# 查看 Coordinator 状态
claw-mesh status
```

### 7. 管理路由规则

```bash
# 添加规则：GPU 任务路由到 linux-gpu 节点
claw-mesh route add --match "gpu:true" --target linux-gpu

# 添加规则：Python 任务路由到有 python 技能的节点
claw-mesh route add --match "skill:python" --strategy least-busy

# 查看规则
claw-mesh route list

# 删除规则
claw-mesh route delete <rule-id>
```

## 进阶用法

### 多机器部署示例

假设你有 3 台机器：

| 机器 | IP | 角色 | 特点 |
|------|-----|------|------|
| Mac Mini | 192.168.1.10 | Coordinator + Node | M2 芯片，日常任务 |
| Linux Server | 192.168.1.20 | Node | RTX 4090，GPU 任务 |
| Raspberry Pi | 192.168.1.30 | Node | 轻量任务，监控 |

**Mac Mini（调度中心 + 节点）：**
```bash
# 初始化
claw-mesh init
# 编辑 claw-mesh.yaml，记下 token

# 启动 Coordinator
claw-mesh up --allow-private &

# 自己也作为节点加入
claw-mesh join http://127.0.0.1:9180 --name mac-mini
```

**Linux Server：**
```bash
claw-mesh join http://192.168.1.10:9180 \
  --name linux-gpu \
  --token <mac-mini的token> \
  --tags gpu,cuda
```

**Raspberry Pi：**
```bash
claw-mesh join http://192.168.1.10:9180 \
  --name rpi-monitor \
  --token <mac-mini的token> \
  --tags lightweight
```

**配置路由规则：**
```bash
# GPU 任务 → Linux
claw-mesh route add --match "gpu:true" --target linux-gpu

# 轻量任务 → Pi
claw-mesh route add --match "skill:lightweight" --target rpi-monitor

# 其他 → 最空闲节点
claw-mesh route add --match "wildcard:true" --strategy least-busy
```

### Web Dashboard

启动 Coordinator 后访问 `http://coordinator-ip:9180/`，可以看到：
- 节点状态概览
- 在线/离线节点列表
- 路由规则管理
- 消息发送测试

### 故障恢复

**Coordinator 重启：**
- 路由规则自动恢复（持久化在 ~/.claw-mesh/rules.json）
- 节点会在 ~30 秒内自动重连
- 无需手动干预

**节点崩溃：**
- Coordinator 在 ~20 秒内通过主动探测检测到
- 自动标记为 offline，不再路由消息
- 节点恢复后重新 `claw-mesh join` 即可

**节点优雅退出（Ctrl+C）：**
- 自动向 Coordinator 发送注销请求
- 立即从节点列表移除

## 配置文件参考

```yaml
# claw-mesh.yaml 完整配置

coordinator:
  port: 9180                    # 监听端口
  token: "your-secret-token"    # Admin token
  allow_private: true           # 允许私有 IP
  data_dir: "~/.claw-mesh"     # 数据目录

node:
  name: "my-node"              # 节点名称
  tags: ["gpu", "python"]      # 自定义标签
  endpoint: ""                 # 注册端点（自动检测）
  gateway:
    endpoint: ""               # OpenClaw Gateway 地址（自动发现）
    token: ""                  # Gateway token
    timeout: 120               # Gateway 超时（秒）
    auto_discover: true        # 自动发现本地 Gateway

tls:
  enabled: false               # TLS（P1 功能，暂未实现）
  cert_file: ""
  key_file: ""
```

## 常见问题

**Q: 节点注册失败，提示 "endpoint validation failed"**
A: 确保 Coordinator 启动时加了 `--allow-private`（局域网环境必须）。

**Q: 消息发送返回 502**
A: 节点可能自动发现了本地 OpenClaw Gateway 但 Gateway 未运行。用 `--no-gateway` 测试，或确保 Gateway 正常运行。

**Q: Coordinator 重启后规则丢失**
A: 检查 `~/.claw-mesh/rules.json` 是否存在。如果用了 `--data-dir`，确保路径一致。

**Q: 节点显示 offline 但实际在运行**
A: 检查网络连通性。Coordinator 需要能访问节点的 handler 端口（默认 9121）。
