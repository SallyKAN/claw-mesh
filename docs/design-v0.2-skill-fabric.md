# claw-mesh v0.2 设计文档：Skill-Aware Routing + Task Plan + Personal AI Fabric

> 版本: v0.2-draft | 更新日期: 2026-03-01

## 1. 背景与目标

claw-mesh v0.1 实现了多机编排的基础能力：节点注册、心跳、能力检测、消息路由。但路由只支持基础匹配（GPU/OS/单个 skill），skill 发现仅限 PATH 中的二进制工具。

v0.2 的三个目标：

1. **Skill-Aware Routing** — 扩展 skill 发现范围，支持 agent skill 文件和自定义 manifest，实现基于 skill 的智能路由
2. **Cross-Node Task Plan** — 支持跨节点多步任务编排，coordinator 调用 LLM 自动拆解任务
3. **Personal AI Fabric 定位** — 从"多机编排工具"升级为"个人 AI 能力网络"

## 2. 设计决策：Skill 与四层同步模型的兼容

### 2.1 现有四层模型

| 层 | 策略 | 内容 |
|----|------|------|
| 身份层 | 共享同步 | SOUL.md、IDENTITY.md、AGENTS.md |
| 记忆层 | 自动同步 | MEMORY.md、memory/*.md |
| 配置层 | 各自独立 | openclaw.json |
| 能力层 | 各自独立 | 本地 skills、工具链、硬件 |

### 2.2 核心原则：区分"skill 定义"和"skill 可执行性"

Skill 引入了一个跨层的概念：

- `.claude/skills/*.md` 是 AI 人格的一部分（"这个 AI 会什么"）→ 属于**身份层** → 通过 git 同步到所有节点
- 但 skill 能否在某节点执行，取决于该节点的硬件和工具链 → 属于**能力层** → 各节点独立判断

**示例**：`ios-build.md` 所有节点都有文件（git 同步），但只有 Mac 节点标记为 `executable=true`（本地判断需要 xcode + darwin）。

### 2.3 更新后的四层模型

| 层 | 策略 | 内容 |
|----|------|------|
| 身份层 | 共享同步 | SOUL.md、IDENTITY.md、**`.claude/skills/*.md`** |
| 记忆层 | 自动同步 | MEMORY.md、memory/*.md |
| 配置层 | 各自独立 | openclaw.json、**skills.yaml** |
| 能力层 | 各自独立 | 硬件检测 + **skill 可执行性判断** |

### 2.4 同步职责分离

- **v0.2 当前**：节点启动时扫描本地目录，不实现同步机制
- **v0.2 git-based sync（后续）**：负责同步 skill 定义文件，和记忆同步走同一套
- **skills.yaml** 是节点独立的配置，不参与同步
- **路由只考虑 `executable=true` 的 skill**

---

## 3. Skill 类型系统

### 3.1 Skill 分类

| 类型 | 来源 | 示例 | 同步 |
|------|------|------|------|
| `tool` | PATH 中的二进制工具自动检测 | docker, xcode, python | 不同步（各节点独立） |
| `agent-skill` | `.claude/skills/*.md` 文件 | ios-build, code-review | 身份层 git 同步 |
| `custom` | `skills.yaml` 手动声明 | stable-diffusion, data-pipeline | 不同步（配置层独立） |

### 3.2 数据结构

```go
type SkillType string

const (
    SkillTypeTool       SkillType = "tool"
    SkillTypeAgentSkill SkillType = "agent-skill"
    SkillTypeCustom     SkillType = "custom"
)

type Skill struct {
    Name        string    `json:"name" yaml:"name"`
    Type        SkillType `json:"type" yaml:"type"`
    Description string    `json:"description,omitempty" yaml:"description,omitempty"`
    Category    string    `json:"category,omitempty" yaml:"category,omitempty"`
    Executable  bool      `json:"executable" yaml:"executable"`
}
```

### 3.3 Capabilities 扩展

```go
type Capabilities struct {
    OS             string   `json:"os"`
    Arch           string   `json:"arch"`
    GPU            bool     `json:"gpu"`
    MemoryGB       int      `json:"memory_gb"`
    Tags           []string `json:"tags"`
    Skills         []string `json:"skills"`                              // 扁平列表（向后兼容，仅 executable=true）
    DetailedSkills []Skill  `json:"detailed_skills,omitempty"`           // 完整 skill 元数据
}
```

`Skills []string` 保留向后兼容，只包含 `executable=true` 的 skill 名称。`DetailedSkills` 包含完整元数据。

---

## 4. Skill 发现机制

### 4.1 Tool Skills（二进制工具检测）

沿用 v0.1 逻辑，检测 PATH 中的已知工具：

| 二进制 | Skill 名称 |
|--------|-----------|
| docker | docker |
| xcodebuild | xcode |
| python3 | python |
| node | nodejs |
| go | golang |
| rustc | rust |
| kubectl | kubernetes |

Tool skills 始终 `executable=true`（检测到就说明能用）。

### 4.2 Agent Skills（`.claude/skills/*.md`）

扫描目录：`~/.claude/skills/`

- 文件名去 `.md` 后缀作为 skill name（如 `code-review.md` → `code-review`）
- 解析文件头部 YAML frontmatter 中的 `requires` 字段
- 对比本节点能力判断 `executable`

**Agent skill 文件格式**：

```markdown
---
requires:
  os: darwin
  tools: [xcode, cocoapods]
  tags: [local]
---
# iOS Build Skill

这个 skill 负责 iOS 项目的构建和打包...
```

**可执行性判断规则**：

| requires 字段 | 判断逻辑 |
|--------------|---------|
| `os` | `runtime.GOOS` 必须匹配 |
| `arch` | `runtime.GOARCH` 必须匹配 |
| `tools` | 每个工具必须在本节点的 tool skills 中 |
| `tags` | 每个 tag 必须在本节点的 tags 中 |
| `gpu` | 如果为 true，本节点必须有 GPU |
| （无 requires）| 默认 `executable=true` |

所有条件为 AND 关系。

### 4.3 Custom Skills（`skills.yaml`）

用户在节点上手动声明的能力。搜索路径：

1. `./skills.yaml`（当前目录）
2. `~/.claw-mesh/skills.yaml`
3. `--skills-manifest` flag 指定的路径

```yaml
# skills.yaml — 本节点的自定义能力声明
skills:
  - name: "stable-diffusion"
    description: "Generate images using Stable Diffusion XL"
    category: "image-gen"

  - name: "code-review"
    description: "Deep code review with security analysis"
    category: "coding"

  - name: "data-pipeline"
    description: "ETL pipeline management with Airflow"
    category: "devops"
```

Custom skills 默认 `executable=true`（用户声明了就说明能用）。

### 4.4 Skill 合并与去重

三个来源的 skill 合并时：
- 同名 skill 以更高优先级来源为准：custom > agent-skill > tool
- 扁平 `Skills []string` 只包含 `executable=true` 的 skill 名称
- `DetailedSkills []Skill` 包含所有 skill（含 `executable=false` 的）

---

## 5. 路由增强

### 5.1 扩展 MatchCriteria

```go
type MatchCriteria struct {
    RequiresGPU       *bool     `json:"requires_gpu,omitempty"`
    RequiresOS        string    `json:"requires_os,omitempty"`
    RequiresArch      string    `json:"requires_arch,omitempty"`       // 新增
    RequiresSkill     string    `json:"requires_skill,omitempty"`      // 保留（向后兼容）
    RequiresSkills    []string  `json:"requires_skills,omitempty"`     // 新增：AND 匹配
    RequiresAnySkill  []string  `json:"requires_any_skill,omitempty"`  // 新增：OR 匹配
    RequiresSkillType SkillType `json:"requires_skill_type,omitempty"` // 新增
    Wildcard          *bool     `json:"wildcard,omitempty"`
}
```

### 5.2 匹配逻辑

所有非空条件为 AND 关系：

| 条件 | 匹配逻辑 |
|------|---------|
| `RequiresGPU` | 节点 `GPU == true` |
| `RequiresOS` | 节点 `OS` 精确匹配 |
| `RequiresArch` | 节点 `Arch` 精确匹配 |
| `RequiresSkill` | 节点 Skills 或 Tags 包含该 skill |
| `RequiresSkills` | 节点必须拥有**所有**列出的 skill（AND） |
| `RequiresAnySkill` | 节点拥有**任一**列出的 skill 即可（OR） |
| `RequiresSkillType` | 节点至少有一个该类型的 executable skill |

### 5.3 CLI 路由规则语法

```bash
# 按架构匹配
claw-mesh route add --match "arch:arm64" --target "mac-mini"

# 多 skill AND 匹配
claw-mesh route add --match "skills:docker,python" --strategy least-busy

# 多 skill OR 匹配
claw-mesh route add --match "any-skill:xcode,docker" --strategy least-busy

# 按 skill 类型匹配
claw-mesh route add --match "skill-type:agent-skill" --target "main-node"
```

### 5.4 Skill 聚合 API

新增 `GET /api/v1/skills` — 返回 mesh 中所有节点的 skill 聚合视图：

```json
[
  {
    "name": "docker",
    "type": "tool",
    "node_ids": ["node-a1b2", "node-c3d4"],
    "node_names": ["mac-mini", "linux-gpu"]
  },
  {
    "name": "stable-diffusion",
    "type": "custom",
    "category": "image-gen",
    "node_ids": ["node-c3d4"],
    "node_names": ["linux-gpu"]
  }
]
```

### 5.5 心跳 Capabilities 刷新

节点每 10 次心跳（约 2.5 分钟）附带刷新后的 capabilities，coordinator 更新 registry。

```go
type HeartbeatRequest struct {
    Status       NodeStatus    `json:"status"`
    Capabilities *Capabilities `json:"capabilities,omitempty"` // 可选刷新
}
```

---

## 6. 统一路由 + 多步任务规划（Task Plan）

### 6.1 设计原则：统一入口，后端自动判断

用户不应该区分"单步路由"和"多步规划"。无论从 Web UI 还是 CLI 发消息，都走同一个入口 `POST /api/v1/route`。Coordinator 内部自动判断：

```
用户发消息 → POST /api/v1/route
                ↓
         Planner 已配置？
           ├── 否 → 直接走现有路由逻辑（单步）
           └── 是 → 调用 LLM Planner 分析
                      ↓
                Planner 返回几步？
                  ├── 1 步 → 走现有路由逻辑（单步，同步返回）
                  └── N 步 → 创建 TaskPlan，异步执行，返回 plan_id
```

**对前端的影响**：Web UI 的 `sendMessage()` 不需要改 API 调用。响应格式扩展：
- 单步：和现在一样，同步返回 `MessageResponse`
- 多步：返回 `{ "plan_id": "...", "steps": N }`，前端轮询 plan 状态

**对 CLI 的影响**：`claw-mesh send` 行为不变。不再需要单独的 `chain` 命令。保留 `claw-mesh plan status <id>` 用于查看多步任务进度。

### 6.2 命名：TaskPlan / PlanStep

"Chain" 是技术实现视角（链式执行），用户视角应该是"任务规划"。内部类型命名：

| 旧名 | 新名 | 理由 |
|------|------|------|
| Chain | TaskPlan | 用户视角：AI 把任务拆成了一个计划 |
| ChainStep | PlanStep | 计划中的一步 |
| ChainExecutor | PlanExecutor | 执行计划 |
| ChainStore | PlanStore | 存储计划 |
| /api/v1/chain | 不需要 | 统一走 /api/v1/route |
| /api/v1/chain/{id} | /api/v1/plans/{id} | 查询计划状态 |

### 6.3 数据结构

```go
type PlanStatus string // pending / running / completed / failed
type StepStatus string // pending / running / completed / failed / skipped

type PlanStep struct {
    ID            string     `json:"id"`
    Skill         string     `json:"skill"`
    NodeID        string     `json:"node_id,omitempty"`
    Prompt        string     `json:"prompt"`
    DependsOnPrev bool       `json:"depends_on_prev"`
    TimeoutSec    int        `json:"timeout_sec,omitempty"`    // 默认 120
    Status        StepStatus `json:"status"`
    Response      string     `json:"response,omitempty"`
    NodeUsed      string     `json:"node_used,omitempty"`
    Error         string     `json:"error,omitempty"`
}

type TaskPlan struct {
    ID            string      `json:"id"`
    UserPrompt    string      `json:"user_prompt"`
    Steps         []PlanStep  `json:"steps"`
    Status        PlanStatus  `json:"status"`
    TimeoutSec    int         `json:"timeout_sec,omitempty"`   // 默认 600
    CreatedAt     time.Time   `json:"created_at"`
    CompletedAt   *time.Time  `json:"completed_at,omitempty"`
    FinalResponse string      `json:"final_response,omitempty"`
    Error         string      `json:"error,omitempty"`
}
```

### 6.4 统一路由流程（修改 `POST /api/v1/route`）

现有 `handleRouteAuto` 的行为扩展：

```go
func (s *Server) handleRouteAuto(w, r) {
    msg := parseMessage(r)

    // 1. 如果指定了 target node，直接转发（不走 planner）
    if msg.TargetNode != "" {
        // 现有逻辑不变
    }

    // 2. 如果 planner 已配置，先尝试规划
    if s.planner != nil {
        skills := s.registry.AggregateSkills()
        plan, err := s.planner.Plan(ctx, msg.Content, skills)

        if err == nil && len(plan.Steps) > 1 {
            // 多步任务 → 异步执行，返回 plan_id
            s.planStore.Save(plan)
            go s.planExecutor.Execute(ctx, plan)
            respondJSON(w, PlanCreatedResponse{
                PlanID: plan.ID,
                Steps:  len(plan.Steps),
            })
            return
        }
        // planner 失败或单步 → 继续走普通路由
    }

    // 3. 普通单步路由（现有逻辑）
    node, err := s.router.Route(msg)
    resp, err := s.forwarder.ForwardMessage(ctx, node, msg, token)
    respondJSON(w, resp)
}
```

### 6.5 响应格式扩展

**单步响应**（和现在一样）：
```json
{
  "message_id": "msg-123",
  "node_id": "node-a1b2",
  "response": "Docker compose 已启动..."
}
```

**多步响应**（新增）：
```json
{
  "plan_id": "plan-x7y8z9",
  "steps": 3,
  "status": "running"
}
```

前端通过检查响应中是否有 `plan_id` 字段来区分：有则切换到 plan 轮询模式，无则直接显示结果。

### 6.6 Web UI 适配

Web UI 的 `sendMessage()` 改动最小：

```javascript
async function sendMessage() {
    const resp = await fetch(API + url, { ... });
    const data = await resp.json();

    if (data.plan_id) {
        // 多步任务：显示规划中状态，开始轮询
        addPlanMessage(data.plan_id, data.steps);
        pollPlan(data.plan_id);
    } else {
        // 单步：和现在一样显示结果
        addMessage('ai', data.response, data.node_id);
    }
}

async function pollPlan(planId) {
    const interval = setInterval(async () => {
        const plan = await fetch(API + '/api/v1/plans/' + planId).then(r => r.json());
        updatePlanDisplay(plan);  // 更新每个 step 的状态
        if (plan.status === 'completed' || plan.status === 'failed') {
            clearInterval(interval);
        }
    }, 2000);
}
```

UI 展示多步任务时，显示为可折叠的步骤列表：

```
┌─────────────────────────────────────────────┐
│ 🤖 AI 正在执行任务规划（3 步）               │
│                                             │
│  ✅ Step 1: python → linux-gpu              │
│     "训练模型完成，准确率 95%"                │
│                                             │
│  ⏳ Step 2: xcode → mac-mini                │
│     正在执行...                              │
│                                             │
│  ⬜ Step 3: docker → linux-gpu              │
│     等待中                                   │
└─────────────────────────────────────────────┘
```

### 6.7 LLM Planner

Coordinator 内置一个 LLM planner，通过 OpenAI 兼容 API 调用外部模型。

**配置**：

```yaml
coordinator:
  planner:
    endpoint: "https://api.openai.com/v1"
    token: "sk-..."
    model: "gpt-4o"
    timeout_sec: 30
```

**System Prompt**：

```
You are a task planner for a Personal AI Fabric — a network of devices
that each contribute unique capabilities to a unified AI assistant.

Your job: decompose the user's request into a sequence of steps,
each targeting a specific skill available in the mesh.

Available skills across the mesh:
- "docker" (tool) on nodes: linux-gpu
- "xcode" (tool) on nodes: mac-mini
- "stable-diffusion" (custom, image-gen) on nodes: linux-gpu
- "python" (tool) on nodes: linux-gpu, mac-mini
...

Rules:
1. If the task can be handled by a single skill on one node, return exactly one step.
2. Only use skills listed above. Do not invent skills.
3. Set depends_on_prev=true when a step needs the output of the previous step.
4. Keep the plan as short as possible. Fewer steps = better.
5. Return ONLY valid JSON matching this schema:

{"steps": [{"skill": "...", "prompt": "...", "depends_on_prev": bool, "timeout_sec": int}]}
```

**回退策略**：
- Planner 未配置 → 所有消息走普通单步路由（完全透明）
- LLM 返回无法解析的 JSON → 回退到单步普通路由
- LLM 返回单步 → 走普通路由（同步返回，不创建 plan）
- LLM 超时 → 回退到单步普通路由

### 6.8 Plan 执行引擎

顺序执行每个 step：

```
for each step in plan.Steps:
    1. 构造 Message，设置 RequiresSkill = step.Skill
    2. 如果 step.DependsOnPrev:
       step.Prompt = step.Prompt + "\n\n--- Previous step output ---\n" + prevStep.Response
    3. Router.Route(msg) → 选择节点
    4. Forwarder.ForwardMessage → 发送到节点
    5. 记录 step.Response, step.NodeUsed, step.Status
    6. 失败处理：
       - 重试一次（复用 Forwarder 的重试逻辑）
       - 仍失败 → plan.Status = failed, 中止
```

**异步执行**：`handleRouteAuto` 检测到多步 plan 后在 goroutine 中执行，立即返回 plan ID。

**超时**：整体超时通过 `context.WithTimeout` 控制，默认 600 秒。

### 6.9 API

| 方法 | 路径 | 认证 | 说明 |
|------|------|------|------|
| POST | /api/v1/route | Bearer Token | 统一入口（自动判断单步/多步） |
| GET | /api/v1/plans/{id} | Bearer Token | 查询 plan 执行状态 |
| GET | /api/v1/skills | 无 | 聚合 skill 列表 |

**GET /api/v1/plans/{id} 响应**：

```json
{
  "id": "plan-x7y8z9",
  "user_prompt": "在 Linux 上训练模型，然后在 Mac 上做 iOS 集成",
  "status": "running",
  "steps": [
    {
      "id": "step-1",
      "skill": "python",
      "status": "completed",
      "node_used": "linux-gpu",
      "response": "模型训练完成，准确率 95%..."
    },
    {
      "id": "step-2",
      "skill": "xcode",
      "status": "running",
      "node_used": "mac-mini"
    }
  ]
}
```

### 6.10 CLI

```bash
# 发消息（自动判断单步/多步，和以前一样）
claw-mesh send --auto "在 Linux 上训练模型，然后在 Mac 上做 iOS 集成"
# 如果是多步，输出：Plan plan-x7y8z9 created (2 steps), executing...

# 查看 plan 状态
claw-mesh plan status plan-x7y8z9

# 查看 mesh 中所有 skill
claw-mesh skills
```

---

## 7. 端到端场景

以下场景基于一个典型的三节点 mesh：

```
┌─────────────────────────────────────────────────────────────┐
│                    Coordinator (:9180)                       │
└──────────┬──────────────┬──────────────┬────────────────────┘
           │              │              │
   ┌───────▼──────┐ ┌────▼───────┐ ┌────▼──────────────┐
   │  mac-mini    │ │ linux-gpu  │ │ pi-home            │
   │  darwin/arm64│ │ linux/amd64│ │ linux/arm64         │
   │  16GB, Metal │ │ 64GB, A100 │ │ 4GB, no GPU         │
   │              │ │            │ │                     │
   │  skills:     │ │  skills:   │ │  skills:            │
   │  - xcode     │ │  - docker  │ │  - python           │
   │  - golang    │ │  - python  │ │  - home-automation  │
   │  - ios-build │ │  - golang  │ │    (custom)         │
   │    (agent)   │ │  - sd-xl   │ │  - sensor-reader    │
   │  - cocoapods │ │    (custom)│ │    (agent)          │
   │    (tool)    │ │  - k8s     │ │                     │
   └──────────────┘ └────────────┘ └─────────────────────┘
```

### 场景 1：Skill 感知路由 — 单步自动路由

**用户操作**：
```bash
claw-mesh send --auto "帮我跑一下 docker compose up"
```

**系统行为**：
1. Coordinator 收到消息，进入 Router
2. 已有规则 `match: { requires_skill: "docker" }` → strategy: least-busy
3. Router 扫描在线节点，只有 `linux-gpu` 有 docker skill
4. Forwarder 转发消息到 `linux-gpu` 的 OpenClaw Gateway
5. Gateway 执行 `docker compose up`，返回结果
6. 用户收到响应

**关键点**：用户不需要知道哪台机器有 docker，路由自动处理。

### 场景 2：Agent Skill 可执行性判断

**节点配置**：三个节点都通过 git 同步了 `.claude/skills/ios-build.md`：

```markdown
---
requires:
  os: darwin
  tools: [xcode, cocoapods]
---
# iOS Build Skill
帮用户构建 iOS 项目，运行 xcodebuild...
```

**各节点判断**：
- `mac-mini`：os=darwin ✓, xcode ✓, cocoapods ✓ → `executable=true`
- `linux-gpu`：os=linux ✗ → `executable=false`
- `pi-home`：os=linux ✗ → `executable=false`

**结果**：三个节点的 `DetailedSkills` 都包含 `ios-build`，但只有 `mac-mini` 的 `Skills` 扁平列表包含它。路由 `requires_skill: "ios-build"` 只会匹配到 `mac-mini`。

### 场景 3：多步任务规划 — 跨机器 AI 图片生成 + iOS 集成

**用户操作**（Web UI 或 CLI，同一个入口）：
```bash
claw-mesh send --auto "生成一张赛博朋克风格的 App 启动图，然后集成到我的 iOS 项目里"
```

**系统行为**：

Step 1 — `handleRouteAuto` 检测到 planner 已配置，调用 LLM：
```
Coordinator → POST planner LLM
System prompt 包含可用 skills:
  - sd-xl (custom, image-gen) on linux-gpu
  - ios-build (agent-skill) on mac-mini
  - xcode (tool) on mac-mini
  ...

LLM 返回:
{
  "steps": [
    {"skill": "sd-xl", "prompt": "生成赛博朋克风格的 App 启动图，1242x2688 分辨率", "depends_on_prev": false, "timeout_sec": 180},
    {"skill": "ios-build", "prompt": "将上一步生成的图片设置为 LaunchScreen，更新 Assets.xcassets", "depends_on_prev": true, "timeout_sec": 120}
  ]
}
```

Step 2 — 多步 → 创建 TaskPlan，异步执行，立即返回：
```json
{"plan_id": "plan-x7y8z9", "steps": 2, "status": "running"}
```

Step 3 — 执行 step-1：
```
Router: requires_skill="sd-xl" → 匹配 linux-gpu
Forwarder → linux-gpu OpenClaw Gateway
Gateway 调用 Stable Diffusion，生成图片
Response: "图片已生成，保存在 /tmp/launch-cyberpunk.png，base64: ..."
```

Step 4 — 执行 step-2（depends_on_prev=true）：
```
Prompt 注入上一步输出:
  原始 prompt + "\n\n--- Previous step output ---\n图片已生成..."
Router: requires_skill="ios-build" → 匹配 mac-mini
Forwarder → mac-mini OpenClaw Gateway
Gateway 读取 ios-build.md skill，执行 xcodebuild 集成
Response: "已将启动图添加到 Assets.xcassets，build succeeded"
```

Step 5 — plan 完成：
```
plan.FinalResponse = step-2 的 response
plan.Status = completed
```

**Web UI 展示**：
```
┌─────────────────────────────────────────────┐
│ 🤖 任务规划执行完成（2 步）                   │
│                                             │
│  ✅ Step 1: sd-xl → linux-gpu (23s)         │
│     "图片已生成，保存在 /tmp/launch-..."      │
│                                             │
│  ✅ Step 2: ios-build → mac-mini (45s)      │
│     "已将启动图添加到 Assets.xcassets"        │
└─────────────────────────────────────────────┘
```

**CLI 查看**：
```bash
$ claw-mesh plan status plan-x7y8z9
Plan: plan-x7y8z9 [completed]
  Step 1: sd-xl → linux-gpu ✓ (23s)
  Step 2: ios-build → mac-mini ✓ (45s)
Result: 已将启动图添加到 Assets.xcassets，build succeeded
```

### 场景 4：多步任务规划 — 失败与回退

**用户操作**：
```bash
claw-mesh send --auto "在 GPU 服务器上训练模型，然后部署到 K8s"
```

**Planner 拆解为 2 步，异步执行**：

```
Step 1: python → linux-gpu ✓ "模型训练完成，保存在 /models/resnet-v2.pt"
Step 2: k8s → linux-gpu ... 发送失败（kubectl 连接超时）
  → Forwarder 重试 1 次 ... 仍然失败
  → step-2.Status = failed, step-2.Error = "kubectl: connection timed out"
  → plan.Status = failed
```

**Web UI 展示**：
```
┌─────────────────────────────────────────────┐
│ ⚠️ 任务规划执行失败（2 步，1 步失败）         │
│                                             │
│  ✅ Step 1: python → linux-gpu (180s)       │
│     "模型训练完成，保存在 /models/resnet..."  │
│                                             │
│  ❌ Step 2: k8s → linux-gpu                 │
│     "kubectl: connection timed out"          │
└─────────────────────────────────────────────┘
```

用户可以修复 K8s 连接后手动重试：
```bash
claw-mesh send --node linux-gpu "部署 /models/resnet-v2.pt 到 K8s 集群"
```

### 场景 5：边缘设备 — 树莓派 IoT 数据采集 + 云端分析

**pi-home 的 skills.yaml**：
```yaml
skills:
  - name: "home-automation"
    description: "控制智能家居设备，读取传感器数据"
    category: "iot"
```

**pi-home 的 `.claude/skills/sensor-reader.md`**：
```markdown
---
requires:
  os: linux
  tools: [python]
  tags: [iot]
---
# Sensor Reader
读取 GPIO 传感器数据，支持温湿度、光照、空气质量...
```

**用户操作**：
```bash
claw-mesh send --auto "读取家里的温湿度数据，分析趋势并生成可视化图表"
```

**Planner 拆解为 2 步**：
```json
{
  "steps": [
    {"skill": "sensor-reader", "prompt": "读取最近 24 小时的温湿度传感器数据，输出 CSV 格式", "depends_on_prev": false, "timeout_sec": 30},
    {"skill": "python", "prompt": "分析温湿度 CSV 数据，用 matplotlib 生成趋势图，标注异常值", "depends_on_prev": true, "timeout_sec": 60}
  ]
}
```

**执行**：
```
Step 1: sensor-reader → pi-home ✓ (读取 GPIO，输出 CSV)
Step 2: python → linux-gpu ✓ (pi-home 也有 python，但 linux-gpu 更空闲且内存大，least-busy 选中)
```

**关键点**：Pi 负责数据采集（只有它连着传感器），计算密集的分析任务自动路由到更强的机器。

### 场景 6：Skill 聚合视图 — 运维可观测

**用户操作**：
```bash
$ claw-mesh skills
SKILL              TYPE          CATEGORY    NODES
docker             tool          -           linux-gpu
golang             tool          -           mac-mini, linux-gpu
home-automation    custom        iot         pi-home
ios-build          agent-skill   -           mac-mini
k8s                tool          -           linux-gpu
python             tool          -           linux-gpu, mac-mini, pi-home
sd-xl              custom        image-gen   linux-gpu
sensor-reader      agent-skill   iot         pi-home
xcode              tool          -           mac-mini

$ claw-mesh nodes
NAME         STATUS   OS      ARCH    GPU    MEM    SKILLS
mac-mini     online   darwin  arm64   yes    16GB   5 (2 agent, 3 tool)
linux-gpu    online   linux   amd64   yes    64GB   5 (1 custom, 4 tool)
pi-home      online   linux   arm64   no     4GB    3 (1 agent, 1 custom, 1 tool)
```

### 场景 7：节点下线 Failover + Skill 路由

**初始状态**：`linux-gpu` 和 `mac-mini` 都有 `python` skill。

**事件**：`linux-gpu` 断电下线。

```
Health checker: linux-gpu 心跳超时 → 标记 offline
```

**用户操作**：
```bash
claw-mesh send --auto "跑一下 Python 单元测试"
```

**路由行为**：
```
Router: requires_skill="python"
  → linux-gpu: offline ✗
  → mac-mini: online, has python ✓
  → pi-home: online, has python ✓
  → least-busy 选择 mac-mini（16GB > 4GB，更适合跑测试）
```

**关键点**：用户无感知。GPU 机器挂了，Python 任务自动切到 Mac。如果任务需要 `sd-xl`（只有 linux-gpu 有），则返回错误"no online node with required skill"。

### 场景 8：心跳 Capabilities 刷新 — 动态 Skill 变更

**事件**：运维在 `linux-gpu` 上安装了 `rustc`。

```
T+0:    linux-gpu skills = [docker, python, golang, k8s, sd-xl]
T+2.5m: 第 10 次心跳，触发 capabilities 刷新
        DetectCapabilities() 重新扫描 → 发现 rustc
        HeartbeatRequest.Capabilities = { ..., skills: [..., "rust"] }
        Coordinator 更新 registry
T+2.5m: linux-gpu skills = [docker, python, golang, k8s, sd-xl, rust]
```

**用户操作**：
```bash
$ claw-mesh skills | grep rust
rust               tool          -           linux-gpu
```

无需重启节点，skill 自动刷新。

---

## 8. Personal AI Fabric 定位

### 7.1 叙事升级

从：
> "Multi-Gateway orchestrator for OpenClaw. One mesh, many claws."

到：
> "Your Personal AI Fabric — one AI, all your devices."

核心叙事：你的所有设备组成一个私有能力网络。每个设备贡献自己的独特能力（GPU、Xcode、Docker、公网 IP）。AI 助手看到的是一个统一的能力平面，不关心能力在哪台机器上。

### 7.2 三个核心概念

1. **Unified Capability Plane** — 所有设备的能力汇聚为一个平面，AI 按需调用
2. **Skill-Aware Routing** — 消息自动路由到拥有所需 skill 的节点
3. **Task Plan** — 复杂任务自动拆解为跨节点的多步执行

### 7.3 文档更新范围

- `README.md` — 重写，Personal AI Fabric 叙事
- `README-zh.md` — 中文版
- `docs/usage-zh.md` — 加入 skills、plan status 命令说明
- CLI `Short`/`Long` 描述 — 更新为 fabric 叙事

---

## 8. 文件变更清单

### 修改文件

| 文件 | 变更 |
|------|------|
| `internal/types/types.go` | 新增 Skill/SkillType/TaskPlan/PlanStep 等类型，修改 Capabilities/MatchCriteria/HeartbeatRequest |
| `internal/node/capabilities.go` | 集成新 skill 发现，填充 DetailedSkills |
| `internal/coordinator/router.go` | 新增 RequiresArch/RequiresSkills/RequiresAnySkill/RequiresSkillType 匹配 |
| `internal/coordinator/api.go` | validateRule 加入新字段检查 |
| `internal/coordinator/registry.go` | 新增 UpdateCapabilities、AggregateSkills、深拷贝 DetailedSkills |
| `internal/coordinator/server.go` | 修改 handleRouteAuto 集成 planner 判断、新增 plan 路由、heartbeat capabilities 刷新 |
| `internal/node/agent.go` | 心跳定期刷新 capabilities |
| `internal/config/config.go` | 新增 PlannerConfig、NodeConfig.SkillsManifest |
| `cmd/claw-mesh/main.go` | 新增 skills/plan status 子命令，更新 route add 语法，更新 CLI 描述 |
| `README.md` | Personal AI Fabric 叙事重写 |
| `README-zh.md` | 中文版 |
| `docs/usage-zh.md` | 新功能使用说明 |

### 新建文件

| 文件 | 职责 |
|------|------|
| `internal/node/skills.go` | Agent skill 发现、OpenClaw skill 发现、skills.yaml 解析、合并去重 |
| `internal/coordinator/planner.go` | LLM planner（OpenAI 兼容 client + prompt 构建） |
| `internal/coordinator/plan.go` | PlanExecutor + PlanStore |
| `internal/coordinator/plan_api.go` | Plan HTTP handlers（GET /api/v1/plans/{id}、GET /api/v1/skills） |

---

## 9. Config Seed — 新节点配置同步

### 9.1 问题

新节点 `join --auto-install` 时，安装完 OpenClaw 后需要手动配置 API key、model、provider 等。这些配置和主节点完全一样，重复配置既麻烦又容易出错。

同时，根据四层同步模型，身份层（SOUL.md、IDENTITY.md、AGENTS.md、`.claude/skills/*.md`）和记忆层（MEMORY.md、`memory/*.md`）也应该在新节点加入时同步过去。

### 9.2 设计

Coordinator 本地读取主节点的 OpenClaw 配置和 workspace 文件，通过 API 分发给新节点。

**CoordinatorConfig 新增字段**：

```yaml
coordinator:
  port: 9180
  token: "..."
  workspace_dir: "/home/user/clawd"              # 主节点 workspace 路径
  openclaw_config: "~/.config/openclaw/openclaw.json"  # 主节点 OpenClaw 配置路径
```

**新增 API**：

| 方法 | 路径 | 认证 | 说明 |
|------|------|------|------|
| GET | /api/v1/seed/config | Bearer Token | 返回 OpenClaw 配置（去掉 channel/port 等本地字段） |
| GET | /api/v1/seed/workspace | Bearer Token | 返回身份层 + 记忆层文件（JSON 打包） |

### 9.3 Config Seed 过滤规则

从主节点 `openclaw.json` 中读取完整配置，去掉以下本地字段后返回：

**排除字段**（节点独立，不同步）：
- `channels` — Telegram/Discord 等通道配置
- `port` / `listen` — 监听端口
- `hostname` — 主机名
- `daemon` — 守护进程配置
- `node_name` — 节点名称

**保留字段**（所有节点共享）：
- `apiProvider` / `apiKey` / `apiBase` — AI 提供商配置
- `model` / `defaultModel` — 模型配置
- `thinking` — 推理配置
- `persona` / `soul` — 人格配置
- 其他非本地字段

### 9.4 Workspace Seed 内容

按四层模型同步身份层和记忆层文件：

```json
{
  "files": [
    {"path": "SOUL.md", "content": "..."},
    {"path": "IDENTITY.md", "content": "..."},
    {"path": "AGENTS.md", "content": "..."},
    {"path": "MEMORY.md", "content": "..."},
    {"path": "memory/2026-03-01.md", "content": "..."},
    {"path": "memory/2026-03-02.md", "content": "..."}
  ]
}
```

不同步的文件：
- `HEARTBEAT.md` — 各节点独立
- `TOOLS.md` — 各节点独立（本地工具笔记）
- `USER.md` — 包含隐私信息，不通过 API 分发
- `openclaw.json` — 走 seed/config 单独处理
- `skills/` — v0.2 后续通过 git-based sync 处理

### 9.5 新节点 Join 流程

```
claw-mesh join <coordinator> --auto-install
  │
  ├─ 1. 检测/安装 OpenClaw runtime
  │
  ├─ 2. GET /api/v1/seed/config
  │     → 写入本地 ~/.config/openclaw/openclaw.json
  │     → 跳过 `openclaw onboard`，直接 `openclaw gateway start`
  │
  ├─ 3. GET /api/v1/seed/workspace
  │     → 写入本地 workspace 目录
  │     → 身份层 + 记忆层文件就位
  │
  └─ 4. 启动 gateway → 注册到 coordinator → 正常运行
```

**CLI flag**：
- `--sync-config` — 从 coordinator 拉取 OpenClaw 配置（默认 true when --auto-install）
- `--no-sync-config` — 跳过配置同步（使用本地已有配置）

### 9.6 安全考虑

- 所有 seed API 需要 Bearer token 认证（复用 mesh token）
- API key 在传输中通过 HTTPS 保护（生产环境应启用 TLS）
- Coordinator 只读取本地文件，不缓存敏感信息

---

## 10. 实施顺序

**Phase 0: Config Seed（新节点配置同步）**
1. config/config.go — CoordinatorConfig 新增 WorkspaceDir、OpenClawConfig
2. coordinator/seed.go — seed API handlers（config + workspace）
3. coordinator/server.go — 注册 seed 路由
4. node/runtime.go — 新增 FetchSeedConfig、FetchSeedWorkspace
5. cmd/claw-mesh/main.go — join 流程集成 seed 拉取
6. 测试

**Phase 1: Skill-Aware Routing**
1. types.go — 新类型
2. node/skills.go — skill 发现
3. node/capabilities.go — 集成
4. coordinator/router.go — 路由增强
5. coordinator/api.go — 验证更新
6. coordinator/registry.go — 聚合 + 刷新
7. coordinator/server.go — 新路由 + heartbeat
8. node/agent.go — 定期刷新
9. config/config.go — 新配置
10. main.go — CLI
11. 测试

**Phase 2: 多步任务规划（Task Plan）**（依赖 Phase 1）
1. types.go — TaskPlan/PlanStep 类型
2. coordinator/planner.go
3. coordinator/plan.go
4. coordinator/plan_api.go
5. coordinator/server.go — handleRouteAuto 集成 planner + plan 路由
6. config/config.go — planner 配置
7. main.go — plan status 命令
8. web UI — plan 轮询展示
9. 测试

**Phase 3: Personal AI Fabric 定位**（可并行）
1. README.md
2. README-zh.md
3. docs/usage-zh.md
4. CLI 描述

---

## 10. 注意事项

1. **向后兼容**：`server.go` 的 `decodeJSON` 使用 `DisallowUnknownFields()`，新增字段后旧 coordinator 会拒绝新 node。这是 minor version bump 可接受的 breaking change。

2. **Plan 异步执行**：`handleRouteAuto` 检测到多步 plan 后立即返回 plan ID，避免 30s WriteTimeout 限制。

3. **Planner 未配置**：所有消息走普通单步路由，用户完全无感知。

4. **心跳 payload**：capabilities 刷新用计数器（每 10 次心跳），不是每次都发。

5. **Skill 文件同步**：v0.2 不实现同步机制，依赖用户手动或后续 git-based sync。
