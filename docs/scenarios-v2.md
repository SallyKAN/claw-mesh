### 场景 1：「线上出问题了，帮我查」— 分布式 AI 推理

**日常频率：每周数次 | SSH 做不到的原因：远程数据量超出 context window**

这是 claw-mesh 最核心的差异化场景。SSH 模式下，AI 要把远程日志、数据库查询结果、系统状态全部拉回本地塞进 context window。日志量一大就爆了。claw-mesh 模式下，远程节点的 AI 本地分析，只传结论。

**用户操作**（Mac 上）：
```
用户说下单失败了，帮我查查怎么回事
```

**系统行为**：
```
POST /api/v1/route → handleRouteAuto
  → Planner 分析：需要在服务器上做深度排障

  → linux-dev 节点的 AI 大脑本地执行：
    1. docker logs app --since 1h (可能几万行，本地 AI 直接读)
    2. docker logs redis --since 1h
    3. psql -c "SELECT * FROM orders WHERE status='failed' ORDER BY created_at DESC LIMIT 20"
    4. redis-cli info clients
    5. df -h && free -m
    → 本地 AI 分析所有数据，提炼出结论

  → 只传回结论（不是原始日志）：
    "根因：Redis 连接池耗尽。15 分钟前 cache-warmer 任务
     新建了 98 个连接未释放，maxclients=100 已满。
     导致最近 23 个订单写入超时。
     建议：cache-warmer.ts:42 加连接池限制。"
```

**vs SSH 模式**：
```
SSH: AI 在 Mac 上 → ssh linux-dev "docker logs app --since 1h"
     → 10MB 日志拉回本地 → context window 爆了 → 只能 grep 关键词
     → 丢失了日志之间的时间关联和上下文

claw-mesh: linux-dev 上的 AI 本地读 10MB 日志
     → 本地分析时间线、关联多个服务的日志
     → 只传回 200 字的根因分析
```

**关键差异**：不是"方便不方便"的问题，是"能不能做"的问题。当数据量超过 context window 时，SSH 模式根本做不了深度分析。

---

### 场景 2：「换个参数重新跑」— 算力亲和（数据亲和）

**日常频率：AI/ML 工程师每天多次 | SSH 做不到的原因：数据在远程，搬不动**

训练数据 50GB 在 gpu-vps 上，模型 checkpoint 也在那。你说"换个 learning rate 重新跑"，数据不需要动，任务应该路由到数据所在的机器。

**用户操作**（Mac 上）：
```
刚才那个训练 loss 降不下去，把 learning rate 从 2e-5 调到 1e-5 重新跑
```

**系统行为**：
```
POST /api/v1/route → handleRouteAuto
  → Planner 分析：单步任务，需要 cuda skill
  → Router: requires_skill="cuda"
    ├─ gpu-vps: online, has cuda ✓
    │  且上一次训练的数据和 checkpoint 都在这台机器上（状态亲和）
    └─ 其他节点: no cuda ✗
  → 选中 gpu-vps

  → gpu-vps 节点的 AI 大脑本地执行：
    修改 configs/finetune.yaml 中的 lr: 1e-5
    数据已在本地 /workspace/data/ (50GB，不需要重传)
    checkpoint 已在本地 /workspace/checkpoints/ (从上次断点继续)
    accelerate launch train.py --resume-from latest
```

**vs SSH 模式**：
```
SSH: AI 在 Mac 上 → ssh gpu-vps "vim configs/finetune.yaml" → 改参数 → 重新跑
     这个 SSH 确实能做。但：
     - AI 不知道上次训练的状态（loss 曲线、最佳 checkpoint）
     - AI 不知道数据在哪个路径
     - AI 不知道该 resume 还是从头跑

claw-mesh: gpu-vps 上的 AI 有完整的本地上下文
     - 它知道上次训练跑到哪了（本地读 tensorboard logs）
     - 它知道数据和 checkpoint 的路径
     - 它能智能决定 resume from checkpoint 而不是从头跑
```

**关键差异**：数据亲和 + 状态亲和。50GB 数据搬不动，训练状态也搬不动。远程 AI 有完整的本地上下文，能做出更智能的决策。

---

### 场景 3：「模型已经加载了，再试几个 prompt」— 算力亲和（缓存亲和）

**日常频率：AI 开发者每天多次 | SSH 做不到的原因：GPU 显存状态无法远程感知**

7B 模型加载到 GPU 显存需要 2 分钟。加载完之后，后续推理请求应该路由到已加载模型的节点，而不是重新加载。

**用户操作**（Mac 上，连续对话）：
```
第一条：帮我用本地的 Qwen-7B 测试一下这个 prompt
第二条：效果不好，换个 system prompt 再试
第三条：这次好多了，再试 10 个 test case
```

**系统行为**：
```
第一条：
  → gpu-vps: 加载模型到显存 (2min) → 推理 → 返回结果

第二条：
  → Router 识别到 gpu-vps 上模型已加载（缓存亲和）
  → 直接路由到 gpu-vps，跳过加载，直接推理 (<1s)

第三条：
  → 同样路由到 gpu-vps（模型还在显存里）
  → 批量跑 10 个 test case，本地 AI 直接循环调用
  → 返回汇总结果
```

**vs SSH 模式**：
```
SSH: AI 在 Mac 上 → ssh gpu-vps "python inference.py --prompt '...'"
     每次调用都是独立的 SSH 命令
     AI 不知道模型是否已加载
     可能每次都重新加载模型（2min × 3 = 6min 浪费）

claw-mesh: gpu-vps 上的 AI 知道模型已在显存中
     后续请求直接推理，无需重新加载
     10 个 test case 本地批量跑，不用 10 次 SSH 往返
```

**关键差异**：缓存亲和。远程 AI 知道本地 GPU 的状态（哪个模型已加载、显存占用多少），能做出最优调度。SSH 模式下 AI 对远程 GPU 状态完全无感知。

---

### 场景 4：「帮我盯着服务器」— 自治 Agent（SSH 根本做不到）

**日常频率：设置一次，持续受益 | SSH 根本做不到：需要 always-on 主动 agent**

这是 claw-mesh 的杀手场景。SSH 是被动的（你连过去才能做事），claw-mesh 的节点是主动的（它自己决定什么时候做什么）。

**用户操作**：
```
帮我盯着 linux-dev 上跑的服务，有异常直接处理
```

**pi-home 节点持续运行**（always-on，5W 功耗）：
```
每 30 秒:
  curl http://linux-dev:3000/healthz
  curl http://linux-dev:5432 (PostgreSQL)
  curl http://linux-dev:6379 (Redis)
每 5 分钟:
  ssh linux-dev "df -h && docker stats --no-stream"
每小时:
  检查 SSL 证书过期时间
  分析磁盘空间趋势
```

**凌晨 3 点，用户在睡觉**：
```
pi-home 检测到 linux-dev:5432 连接失败
  → pi-home 的 AI 大脑自主决策：
    ssh linux-dev "docker logs postgres --tail 50"
    → 分析：PostgreSQL OOM killed
    → 决定：自动重启
    ssh linux-dev "docker compose restart postgres"
    → 等待恢复 → health check 通过
    → 记录事件到 coordinator
```

**用户早上打开 Web UI**：
```
┌─────────────────────────────────────────────────────┐
│ ⚠️ 夜间事件报告（pi-home 哨兵）                     │
│                                                     │
│  03:12 PostgreSQL 容器 OOM killed                   │
│  03:12 自动重启 → 03:14 恢复正常                     │
│  这是本周第 3 次 OOM，建议调整 shm_size              │
│                                                     │
│  当前状态：所有服务正常 ✓                             │
│  要我改 docker-compose.yml 的 shm_size 吗？          │
└─────────────────────────────────────────────────────┘
```

**vs SSH**：SSH 根本做不到这个。没有人在凌晨 3 点发起 SSH 连接。即使用 cron + 脚本做监控，也只能做固定规则的检查，不能像 AI agent 一样分析日志、判断根因、决定是否重启。

**Pi 的不可替代性**：
- always-on，功耗极低（5W），Mac 关机了它还在
- 在家庭内网，能监控内网服务（不需要公网暴露）
- 有自己的 AI 大脑，能自主决策而不只是报警

---

### 场景 5：「帮我在三台机器上同时准备」— 真正的并行执行

**日常频率：每周数次 | SSH 做不到的原因：AI agent 框架串行执行工具调用**

现有 AI agent（Claude Code、Cursor）执行工具调用是串行的。SSH 到 3 台机器依次操作，总时间是三者之和。claw-mesh 的 coordinator 天然支持并行分发。

**用户操作**（Mac 上）：
```
明天要 demo，帮我同时准备好后端、AI 服务和前端
```

**系统行为**：
```
POST /api/v1/route → handleRouteAuto
  → Planner 拆解为 3 步，前 2 步可并行：

  → Step 1 (并行): linux-dev (requires_skill="docker")
    git pull → docker compose build → docker compose up -d
    重置 demo 数据库 → 跑 E2E 测试
    耗时：3 分钟

  → Step 2 (并行): gpu-vps (requires_skill="cuda")
    加载模型到显存 → 预热推理缓存
    跑 5 个 sample request 确认响应时间
    耗时：2 分钟

  → Step 3 (等前两步完成): mac-mini
    汇总所有检查结果
    耗时：5 秒

  总耗时：3 分钟（并行）而不是 5 分钟（串行）
```

**vs SSH 模式**：
```
SSH (串行):
  ssh linux-dev "docker compose up..." → 等 3 分钟
  ssh gpu-vps "python load_model.py..." → 等 2 分钟
  汇总 → 5 秒
  总耗时：5 分钟 + 网络往返开销

claw-mesh (并行):
  coordinator 同时分发到 linux-dev 和 gpu-vps
  两台机器各自的 AI 独立执行
  总耗时：3 分钟（取最慢的那个）
```

**关键差异**：不只是快了 2 分钟。当涉及 4-8 台机器时，串行 vs 并行的差距会指数级放大。而且每个节点的 AI 能独立处理异常（比如 build 失败自动重试），不需要等 Mac 上的 AI 一个个处理。

---

### 场景 6：「继续刚才的调试」— 会话亲和（状态亲和）

**日常频率：每天多次 | SSH 做不到的原因：远程 AI 有持续的调试上下文**

你在 linux-dev 上 debug 一个问题，AI 已经读了代码、查了日志、设了断点。你说"继续"，应该回到同一台机器的同一个 AI session，而不是重新建立上下文。

**用户操作**（连续对话）：
```
第一条：linux-dev 上的 payment 服务有个 race condition，帮我查
第二条：（10 分钟后）找到了吗？
第三条：试试加个 mutex 看看能不能修
第四条：改完了帮我跑一下相关测试
```

**系统行为**：
```
第一条：
  → 路由到 linux-dev
  → linux-dev AI 开始排查：读代码、加日志、复现问题
  → 建立了完整的调试上下文（哪些文件看过、哪些假设排除了）

第二条：
  → Router 识别到这是同一个调试会话（会话亲和）
  → 路由到同一个 linux-dev 节点
  → AI 继续之前的上下文："我缩小了范围，问题在 payment/processor.go:127..."

第三条：
  → 继续路由到 linux-dev（状态亲和）
  → AI 直接改代码（它已经知道文件在哪、问题在哪）

第四条：
  → 继续路由到 linux-dev
  → AI 跑测试（它知道该跑哪些测试，因为它理解改动范围）
```

**vs SSH 模式**：
```
SSH: AI 在 Mac 上，每次 ssh linux-dev 执行命令
     AI 的调试上下文在 Mac 的 context window 里
     如果对话太长，context window 溢出，之前的分析全丢了
     AI 要重新读文件、重新分析

claw-mesh: linux-dev 上的 AI 有自己的 session
     调试上下文在本地，不受 Mac 端 context window 限制
     即使用户中间去做了别的事，回来说"继续调试"
     linux-dev 的 AI 还记得之前的分析
```

**关键差异**：会话亲和 + 状态亲和。远程 AI 维护了持续的调试上下文，不会因为 Mac 端 context window 溢出而丢失。这在复杂调试场景（可能持续几小时）中是决定性的优势。

---

### 场景 7：「节点挂了，自动切换」— Failover（SSH 没有）

**日常频率：每月数次，但每次都很关键 | SSH 做不到的原因：没有健康检查和自动切换**

gpu-vps 是按需付费的云实例，可能被回收。linux-dev 可能断电。claw-mesh 有实时健康检查和自动 failover。

**事件**：gpu-vps 云实例被回收
```
Health checker: gpu-vps 心跳超时 → 标记 offline
```

**用户操作**（不知道 gpu-vps 挂了）：
```
帮我跑一下 Python 数据分析脚本
```

**系统行为**：
```
Router: requires_skill="python"
  → gpu-vps: offline ✗ (已被回收)
  → linux-dev: online, has python ✓
  → mac-mini: online, has python ✓
  → pi-home: online, has python ✓
  → least-busy 选择 linux-dev (32GB 内存最适合数据分析)
```

**用户无感知**。如果任务需要 GPU（比如推理），则返回明确错误："gpu-vps 离线，需要 cuda skill 的任务暂时无法执行。要我在 gpu-vps 恢复后自动重试吗？"

**vs SSH 模式**：
```
SSH: AI 在 Mac 上 → ssh gpu-vps "python analyze.py"
     → Connection refused
     → AI 报错，用户手动决定换哪台机器
     → ssh linux-dev "python analyze.py"
     → 用户要自己记住哪台机器有 python

claw-mesh: 自动 failover，用户无感知
     能力注册表实时更新，路由器自动选择在线节点
```

---

### 场景评估总结

| 场景 | 核心差异 | SSH 能替代吗？ | 杀伤力 |
|------|---------|--------------|--------|
| 1. 深度排障 | 分布式 AI 推理，本地分析大量数据 | 简单排障能，复杂排障不能 | **S 级** |
| 2. 换参数重跑 | 数据亲和，50GB 数据不用搬 | 能跑命令，但缺本地上下文 | **A 级** |
| 3. 模型推理迭代 | 缓存亲和，模型已在显存 | 不知道 GPU 状态 | **A 级** |
| 4. 智能哨兵 | 自治 agent，主动监控修复 | **根本做不到** | **S 级** |
| 5. 多机并行 | 真正的并行执行 | 只能串行 | **A 级** |
| 6. 持续调试 | 会话亲和，远程 AI 保持上下文 | context window 会溢出 | **A 级** |
| 7. 自动 failover | 健康检查 + 自动切换 | 没有 | **B+ 级** |

**核心叙事**：

> SSH 让你远程执行命令。claw-mesh 让你的每台机器都有一个 AI 大脑，它们协同思考。
>
> 数据不用搬，AI 过去分析。状态不用传，AI 就在本地。你睡着了，AI 还在盯着。
