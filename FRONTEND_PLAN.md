# claw-mesh Dashboard 前端重构计划

## Context

当前 Dashboard 是一个 48KB 的单 HTML 文件（1298 行内联 CSS/JS），功能有限（拓扑图+聊天+向导），5s 轮询更新，无模块化，无可访问性。竞品 SwarmClaw 使用 Next.js 16 + React 19 + Tailwind v4 + shadcn/ui 构建了 15+ 页面的专业 Dashboard。本次重构目标：用相同技术栈重建 claw-mesh Dashboard，采用 Terminal Brutalist 设计风格，覆盖全部 6 个页面，构建产物嵌入 Go 二进制保持单文件分发优势。

## 设计方向：Terminal Brutalist

极简终端风格，大量 monospace 字体，绿色/琥珀色调，黑客美学。

- 字体：IBM Plex Mono (display + body) + Berkeley Mono 或 JetBrains Mono (code/data)
- 主色调：深黑底 `#0a0a0a`，绿色 `#22c55e` 为主强调，琥珀 `#f59e0b` 为次强调
- 视觉特征：扫描线纹理、闪烁光标、等宽网格布局、粗边框、ASCII 装饰
- 动画：打字机效果、终端闪烁、数据流动画
- 卡片：硬边角 `rounded-none` 或极小圆角 `rounded-sm`，1px 实线边框
- 按钮：方形、大写字母、letter-spacing、hover 时反色

## 技术栈

- Next.js 16 (static export → `web/dist/`)
- React 19
- Tailwind CSS v4 (`@theme` directive)
- shadcn/ui (定制为 brutalist 风格)
- Zustand (状态管理)
- 字体自托管 (打包进 public/fonts/)

## 部署模式

- 构建：`next build` → static export → `web/dist/`
- 嵌入：Go `//go:embed all:web/dist` → 单二进制
- 开发：`next dev` + `rewrites` proxy 到 Go API (localhost:9180)
- Token 注入：Go 在 serve index.html 时注入 `window.__TOKEN__`

---

## 项目结构

```
web/
├── package.json
├── next.config.ts          # static export + dev proxy
├── tsconfig.json
├── postcss.config.mjs
├── components.json         # shadcn/ui config
├── public/
│   └── fonts/              # IBM Plex Mono + JetBrains Mono woff2
├── src/
│   ├── app/
│   │   ├── layout.tsx      # Root: fonts, providers, shell
│   │   ├── page.tsx        # Overview dashboard
│   │   ├── nodes/
│   │   │   ├── page.tsx    # Node list
│   │   │   └── [id]/page.tsx  # Node detail
│   │   ├── routing/page.tsx   # Rules management
│   │   ├── chat/page.tsx      # Messages
│   │   ├── sync/page.tsx      # Sync status
│   │   └── wizard/page.tsx    # Add Node wizard
│   ├── components/
│   │   ├── ui/             # shadcn/ui (button, card, badge, dialog, table, input, select, tabs, tooltip, separator)
│   │   ├── layout/
│   │   │   ├── sidebar.tsx
│   │   │   ├── topbar.tsx
│   │   │   └── shell.tsx
│   │   ├── topology/
│   │   │   ├── topology-canvas.tsx
│   │   │   ├── coordinator-hub.tsx
│   │   │   ├── orbit-node.tsx
│   │   │   └── connection-line.tsx
│   │   ├── nodes/
│   │   │   ├── node-card.tsx
│   │   │   ├── node-detail.tsx
│   │   │   ├── capability-badges.tsx
│   │   │   └── status-indicator.tsx
│   │   ├── routing/
│   │   │   ├── rule-list.tsx
│   │   │   └── rule-form.tsx
│   │   ├── chat/
│   │   │   ├── message-list.tsx
│   │   │   ├── message-bubble.tsx
│   │   │   ├── chat-input.tsx
│   │   │   └── typing-indicator.tsx
│   │   ├── sync/
│   │   │   ├── sync-status-card.tsx
│   │   │   └── manifest-viewer.tsx
│   │   └── wizard/
│   │       ├── wizard-stepper.tsx
│   │       ├── network-step.tsx
│   │       ├── config-step.tsx
│   │       └── exec-step.tsx
│   ├── lib/
│   │   ├── api.ts          # Typed fetch wrapper + auth
│   │   ├── ws.ts           # WebSocket stub (future)
│   │   ├── utils.ts        # cn(), formatTime, osIcon
│   │   └── types.ts        # TS types mirroring Go types
│   ├── stores/
│   │   ├── nodes.ts        # Nodes + polling
│   │   ├── chat.ts         # Messages + sending
│   │   ├── rules.ts        # Routing rules CRUD
│   │   ├── sync.ts         # Sync manifest + status
│   │   └── ui.ts           # Sidebar, wizard, theme
│   └── hooks/
│       ├── use-polling.ts
│       └── use-nodes.ts
└── dist/                   # Build output (go:embed)
```

~50 个 TSX/TS 文件 + ~10 个 shadcn/ui 组件。

---

## 设计系统详细定义

### 色彩 (Tailwind v4 @theme)

```css
@theme {
  --color-bg: #0a0a0a;
  --color-surface: #111111;
  --color-surface-2: #1a1a1a;
  --color-border: #2a2a2a;
  --color-border-bright: #3a3a3a;
  --color-text: #e0e0e0;
  --color-text-dim: #888888;
  --color-muted: #555555;
  --color-green: #22c55e;
  --color-green-dim: #16a34a;
  --color-amber: #f59e0b;
  --color-red: #ef4444;
  --color-cyan: #06b6d4;
}
```

### 字体

- IBM Plex Mono: 所有 UI 文本（导航、标题、正文、按钮）
- JetBrains Mono: 代码块、数据表格、命令预览
- 全部自托管 woff2，打包进 public/fonts/

### 视觉特征

- 背景：纯黑 + 微弱扫描线 CSS overlay（repeating-linear-gradient 2px）
- 边框：1px solid #2a2a2a，hover 时 #22c55e
- 卡片：`rounded-sm border border-border bg-surface`，无阴影
- 按钮：`uppercase tracking-widest text-xs font-bold`，绿色边框，hover 反色
- Badge：方形 `rounded-none px-2 py-0.5 text-[10px] uppercase tracking-wider`
- 光标闪烁：`@keyframes blink { 50% { opacity: 0 } }` 用于活跃状态指示
- 数据：表格用等宽字体，行间 1px border-bottom，hover 行高亮绿色

---

## 页面设计

### P1: Overview Dashboard (`/`)

布局：全屏网格，上方统计栏 + 中央拓扑图 + 下方活动日志

```
┌─────────────────────────────────────────────────────┐
│ SIDEBAR │ ┌─ STATS BAR ──────────────────────────┐  │
│         │ │ NODES: 3/5  ONLINE: 3  SYNCED: 2     │  │
│ > MESH  │ └──────────────────────────────────────┘  │
│   NODES │ ┌─ TOPOLOGY ──────────────────────────┐  │
│   ROUTE │ │                                      │  │
│   CHAT  │ │      [node-a] ── [COORD] ── [node-b]│  │
│   SYNC  │ │                    │                 │  │
│         │ │                [node-c]              │  │
│         │ └──────────────────────────────────────┘  │
│         │ ┌─ RECENT ACTIVITY ────────────────────┐  │
│         │ │ 12:03:01 node-a registered (darwin)  │  │
│         │ │ 12:03:15 msg routed → node-b (gpu)   │  │
│         │ └──────────────────────────────────────┘  │
└─────────────────────────────────────────────────────┘
```

组件：
- `topbar.tsx` — 集群统计：`NODES: 3/5 online | SYNCED: 2/3 | RULES: 4`
- `topology-canvas.tsx` — SVG 拓扑图，coordinator 居中，节点环绕，连线用绿色虚线
- `coordinator-hub.tsx` — 中心节点，🦀 emoji，绿色脉冲环
- `orbit-node.tsx` — 节点气泡，OS 图标，状态点，hover 显示详情
- 底部活动日志：终端风格滚动文本，绿色时间戳 + 白色消息

### P2: Nodes (`/nodes`)

布局：节点列表（表格视图），点击展开详情面板

```
┌─────────────────────────────────────────────────────┐
│ NODES                                    [+ ADD]    │
├─────────────────────────────────────────────────────┤
│ ID          NAME       OS      STATUS  LAST HB      │
│ node-a1b2   mac-mini   darwin  ● ON    2s ago       │
│ node-c3d4   linux-gpu  linux   ● ON    5s ago       │
│ node-e5f6   vps-01     linux   ○ OFF   3m ago       │
├─────────────────────────────────────────────────────┤
│ ▼ node-a1b2 DETAIL                                  │
│ OS: darwin/arm64  GPU: no  MEM: 16GB                │
│ TAGS: xcode, homebrew, local                        │
│ SKILLS: coding-agent, apple-notes                   │
│ OPENCLAW: v0.1.0                                    │
│ SYNC: v42 | 4 files | no conflicts                  │
│ ENDPOINT: 192.168.1.100:9120                        │
│                              [REMOVE] [SEND MSG]    │
└─────────────────────────────────────────────────────┘
```

组件：
- `node-card.tsx` — 表格行，monospace，状态用 ● (绿) ○ (灰) ◉ (琥珀)
- `node-detail.tsx` — 展开面板，capability badges，sync 状态，操作按钮
- `capability-badges.tsx` — `[DARWIN]` `[ARM64]` `[16GB]` `[GPU:NO]` 方形 badge
- `status-indicator.tsx` — 彩色圆点 + 文字

### P3: Routing (`/routing`)

布局：规则列表 + 添加规则表单

```
┌─────────────────────────────────────────────────────┐
│ ROUTING RULES                            [+ ADD]    │
├─────────────────────────────────────────────────────┤
│ #1  IF gpu=true      → node-gpu-1    [least-busy] ✕│
│ #2  IF os=darwin     → mac-nodes     [least-busy] ✕│
│ #3  IF skill=docker  → linux-nodes   [least-busy] ✕│
│ #4  IF *             → any           [least-busy] ✕│
├─────────────────────────────────────────────────────┤
│ ┌─ ADD RULE ────────────────────────────────────┐   │
│ │ MATCH:  [gpu ▼] = [true    ]                  │   │
│ │ TARGET: [node-gpu-1 ▼]                        │   │
│ │ STRATEGY: [least-busy ▼]                      │   │
│ │                              [SAVE RULE]      │   │
│ └───────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────┘
```

组件：
- `rule-list.tsx` — 规则表格，每行显示 match → target，删除按钮
- `rule-form.tsx` — 添加规则表单，下拉选择 match 类型/target 节点/strategy

### P4: Chat (`/chat`)

布局：左侧消息历史 + 底部输入框，终端聊天风格

```
┌─────────────────────────────────────────────────────┐
│ MESSAGES                          [route: auto ▼]   │
├─────────────────────────────────────────────────────┤
│ > 帮我跑一下 Xcode build                            │
│ [node-a1b2] Build succeeded. 0 errors, 2 warnings.  │
│                                                      │
│ > 生成一张 512x512 的图片                             │
│ [node-c3d4] Image generated: output.png              │
│                                                      │
│ █ (blinking cursor)                                  │
├─────────────────────────────────────────────────────┤
│ ┌──────────────────────────────────┐ [SEND]         │
│ │ Type a message...                │                 │
│ └──────────────────────────────────┘                 │
└─────────────────────────────────────────────────────┘
```

组件：
- `message-list.tsx` — 终端风格，`>` 前缀用户消息，`[node-id]` 前缀 AI 回复
- `message-bubble.tsx` — 无气泡，纯文本，绿色用户/白色 AI
- `chat-input.tsx` — 底部输入框 + route selector + send 按钮
- `typing-indicator.tsx` — `...` 闪烁动画

### P5: Sync (`/sync`)

布局：同步概览 + 文件 manifest 表格

```
┌─────────────────────────────────────────────────────┐
│ FILE SYNC                        MANIFEST v42       │
├─────────────────────────────────────────────────────┤
│ NODE SYNC STATUS                                     │
│ node-a1b2  v42  4 files  ✓ synced   2s ago          │
│ node-c3d4  v41  3 files  ⚠ behind   1m ago          │
│ node-e5f6  ---  offline                              │
├─────────────────────────────────────────────────────┤
│ MANIFEST FILES                                       │
│ PATH            SHA256 (short)  SIZE    MODIFIED     │
│ SOUL.md         abc123..        1.2KB   12:00:01     │
│ IDENTITY.md     def456..        0.8KB   11:55:30     │
│ MEMORY.md       789abc..        4.1KB   12:03:15     │
│ memory/daily.md fab012..        2.3KB   12:01:00     │
└─────────────────────────────────────────────────────┘
```

组件：
- `sync-status-card.tsx` — 每节点同步状态行
- `manifest-viewer.tsx` — 文件列表表格，SHA256 截断显示

### P6: Add Node Wizard (`/wizard`)

布局：多步骤向导，终端风格

```
┌─────────────────────────────────────────────────────┐
│ ADD NODE TO MESH                    STEP 1/3        │
├─────────────────────────────────────────────────────┤
│ NETWORK TYPE                                         │
│ ┌─────────────┐  ┌─────────────┐                    │
│ │ [x] LAN     │  │ [ ] PUBLIC  │                    │
│ │ Same network│  │ Public IP   │                    │
│ └─────────────┘  └─────────────┘                    │
│                                                      │
│ NODE NAME: [my-node          ]                       │
│ TAGS:      [gpu, docker      ]                       │
│                                                      │
│                    [BACK]  [NEXT →]                   │
├─────────────────────────────────────────────────────┤
│ $ claw-mesh join http://192.168.1.1:9180 \          │
│     --name my-node --tags gpu,docker                 │
│                                    [COPY]            │
└─────────────────────────────────────────────────────┘
```

组件：
- `wizard-stepper.tsx` — 步骤指示器 `STEP 1/3`
- `network-step.tsx` — LAN/Public 选择
- `config-step.tsx` — 名称、标签输入
- `exec-step.tsx` — 命令预览 + 复制 + SSH 部署选项

---

## 状态管理 (Zustand)

### nodes.ts
```ts
interface NodesStore {
  nodes: Node[]
  loading: boolean
  fetch: () => Promise<void>       // GET /api/v1/nodes
  getNode: (id: string) => Node | undefined
  removeNode: (id: string) => Promise<void>  // DELETE /api/v1/nodes/{id}
}
```

### chat.ts
```ts
interface ChatStore {
  messages: ChatMessage[]
  sending: boolean
  routeTarget: string | 'auto'
  send: (content: string) => Promise<void>  // POST /api/v1/route or /route/{nodeId}
  setRouteTarget: (target: string) => void
  clear: () => void
}
```

### rules.ts
```ts
interface RulesStore {
  rules: RoutingRule[]
  fetch: () => Promise<void>       // GET /api/v1/rules
  add: (rule: NewRule) => Promise<void>    // POST /api/v1/rules
  remove: (id: string) => Promise<void>   // DELETE /api/v1/rules/{id}
}
```

### sync.ts
```ts
interface SyncStore {
  manifest: SyncManifest | null
  nodeStatuses: SyncNodeStatus[]
  fetchManifest: () => Promise<void>   // GET /api/v1/sync/manifest
  fetchStatuses: () => Promise<void>   // GET /api/v1/sync/status
}
```

### ui.ts
```ts
interface UIStore {
  sidebarCollapsed: boolean
  wizardOpen: boolean
  toggleSidebar: () => void
  toggleWizard: () => void
}
```

---

## API Client (`lib/api.ts`)

```ts
function getToken(): string {
  return (window as any).__TOKEN__ || ''
}

async function api<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${getToken()}`,
      ...options?.headers,
    },
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(err.error || res.statusText)
  }
  if (res.status === 204) return undefined as T
  return res.json()
}

export const meshApi = {
  nodes: {
    list: () => api<Node[]>('/api/v1/nodes'),
    get: (id: string) => api<Node>(`/api/v1/nodes/${id}`),
    remove: (id: string) => api<void>(`/api/v1/nodes/${id}`, { method: 'DELETE' }),
  },
  route: {
    auto: (content: string, source = 'dashboard') =>
      api<RouteResponse>('/api/v1/route', { method: 'POST', body: JSON.stringify({ content, source }) }),
    toNode: (nodeId: string, content: string, source = 'dashboard') =>
      api<RouteResponse>(`/api/v1/route/${nodeId}`, { method: 'POST', body: JSON.stringify({ content, source }) }),
  },
  rules: {
    list: () => api<RoutingRule[]>('/api/v1/rules'),
    add: (rule: NewRule) => api<RoutingRule>('/api/v1/rules', { method: 'POST', body: JSON.stringify(rule) }),
    remove: (id: string) => api<void>(`/api/v1/rules/${id}`, { method: 'DELETE' }),
  },
  sync: {
    manifest: () => api<SyncManifest>('/api/v1/sync/manifest'),
    status: () => api<SyncNodeStatus[]>('/api/v1/sync/status'),
  },
}
```

---

## WebSocket 预留 (`lib/ws.ts`)

初期用 polling（use-polling.ts hook，默认 3s），预留 WebSocket 接口：

```ts
// 未来 Go 后端添加 /ws/events 后启用
export function connectWS(onEvent: (event: MeshEvent) => void): () => void {
  // stub: return no-op cleanup
  return () => {}
}
```

use-polling.ts:
```ts
function usePolling(fn: () => Promise<void>, intervalMs = 3000) {
  useEffect(() => {
    fn()
    const id = setInterval(fn, intervalMs)
    return () => clearInterval(id)
  }, [fn, intervalMs])
}
```

---

## Go 集成改动

### 需要修改的文件

1. **`embed.go`** — 保持不变，已有 `//go:embed all:web/dist`
2. **`internal/coordinator/dashboard.go`** — 需要适配 Next.js static export 的文件结构：
   - Next.js export 产出 `index.html` + `_next/` 目录 + 各页面 HTML
   - 修改 `DashboardHandler()` 支持 SPA fallback：非 API/非静态资源路径都返回 index.html
   - Token 注入逻辑保持不变（替换 index.html 中的 placeholder）
3. **`Makefile`** — 添加 `web-build` target：`cd web && npm run build`
4. **`.gitignore`** — 添加 `web/node_modules/`，保留 `web/dist/`

### Next.js Static Export 配置

```ts
// web/next.config.ts
const config = {
  output: 'export',
  distDir: 'dist',
  trailingSlash: true,        // 生成 /nodes/index.html 而非 /nodes.html
  images: { unoptimized: true },
  // 开发时 proxy 到 Go API
  async rewrites() {
    return [
      { source: '/api/:path*', destination: 'http://localhost:9180/api/:path*' },
      { source: '/healthz', destination: 'http://localhost:9180/healthz' },
    ]
  },
}
```

### Dashboard Handler 改动

```go
// dashboard.go 需要修改的逻辑：
// 1. 静态文件优先：如果 web/dist/ 中存在对应文件，直接 serve
// 2. SPA fallback：其他路径返回 index.html（带 token 注入）
// 3. 保持 /api/ 路径不被 dashboard handler 拦截
```

---

## 构建流程

### package.json scripts

```json
{
  "scripts": {
    "dev": "next dev --port 3000",
    "build": "next build",
    "lint": "next lint"
  }
}
```

### Makefile 新增

```makefile
web-install:
	cd web && npm install

web-build: web-install
	cd web && npm run build

build: web-build
	go build -o bin/claw-mesh ./cmd/claw-mesh

dev-frontend:
	cd web && npm run dev
```

### 开发工作流

1. 终端 1：`make run-coordinator`（Go API on :9180）
2. 终端 2：`make dev-frontend`（Next.js on :3000，proxy API 到 :9180）
3. 构建发布：`make build`（先 web-build 再 go build）

---

## 实施步骤（按顺序）

### Step 1: 项目脚手架
- 在 `web/` 下初始化 Next.js 16 项目
- 配置 Tailwind v4、PostCSS、TypeScript
- 初始化 shadcn/ui（定制 brutalist 主题）
- 下载并配置自托管字体 (IBM Plex Mono + JetBrains Mono)
- 配置 next.config.ts (static export + dev proxy)
- 验证 `npm run build` 产出到 `web/dist/`
- 验证 Go embed 能正确加载新的 dist 产物

### Step 2: Layout Shell + 设计系统
- 实现 `layout.tsx`（字体加载、全局样式、扫描线背景）
- 实现 `shell.tsx`（sidebar + content area）
- 实现 `sidebar.tsx`（导航：MESH / NODES / ROUTE / CHAT / SYNC）
- 实现 `topbar.tsx`（集群统计栏）
- 定义 Tailwind @theme 色彩变量
- 定制 shadcn/ui 组件为 brutalist 风格（button, card, badge, input, select, table, dialog, tabs）

### Step 3: API Client + Stores
- 实现 `lib/types.ts`（Node, RoutingRule, SyncManifest 等类型）
- 实现 `lib/api.ts`（typed fetch wrapper）
- 实现 `lib/utils.ts`（cn, formatTime, osIcon）
- 实现 5 个 Zustand stores
- 实现 `use-polling.ts` hook

### Step 4: Overview Dashboard (`/`)
- 实现 topology-canvas（SVG 拓扑图）
- 实现 coordinator-hub + orbit-node + connection-line
- 集成 nodes store + polling
- 活动日志区域（终端风格滚动文本）

### Step 5: Nodes 页面 (`/nodes`)
- 实现 node-card（表格行）
- 实现 node-detail（展开面板）
- 实现 capability-badges + status-indicator
- 节点删除操作
- 链接到 wizard

### Step 6: Routing 页面 (`/routing`)
- 实现 rule-list（规则表格）
- 实现 rule-form（添加规则表单）
- 规则删除操作

### Step 7: Chat 页面 (`/chat`)
- 实现 message-list + message-bubble（终端风格）
- 实现 chat-input + route selector
- 实现 typing-indicator
- 集成 chat store

### Step 8: Sync 页面 (`/sync`)
- 实现 sync-status-card（节点同步状态）
- 实现 manifest-viewer（文件列表）

### Step 9: Wizard (`/wizard`)
- 实现 wizard-stepper + 3 个步骤组件
- 命令生成 + 复制功能
- SSH 部署表单

### Step 10: Go 集成 + 构建验证
- 修改 dashboard.go 支持 Next.js static export SPA fallback
- 更新 Makefile
- 端到端测试：`make build` → 启动 coordinator → 浏览器访问所有页面
- 验证 token 注入正常工作
- 验证所有 API 调用正常

---

## 验证方案

1. **开发模式验证**：`npm run dev` + Go API，所有 6 个页面可访问，API 调用正常
2. **构建验证**：`npm run build` 产出到 `web/dist/`，文件结构正确
3. **嵌入验证**：`make build` → `./bin/claw-mesh up` → 浏览器访问 `http://localhost:9180/`
4. **功能验证**：
   - Overview：拓扑图渲染，节点状态实时更新
   - Nodes：列表加载，详情展开，删除操作
   - Routing：规则列表，添加/删除规则
   - Chat：发送消息，auto-route 和指定节点路由
   - Sync：manifest 加载，节点同步状态
   - Wizard：命令生成，复制功能
5. **Token 验证**：无 token 时 API 返回 401，有 token 时正常
6. **响应式验证**：移动端布局正常，sidebar 可折叠
