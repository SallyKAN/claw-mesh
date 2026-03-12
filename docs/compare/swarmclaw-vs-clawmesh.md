# SwarmClaw vs claw-mesh 全方位竞品对比分析

> 调研日期：2026-03-12
> SwarmClaw: https://www.swarmclaw.ai / https://github.com/swarmclawai/swarmclaw
> claw-mesh: https://github.com/SallyKAN/claw-mesh

---

## 一、项目基本面

| 维度 | SwarmClaw | claw-mesh |
|------|-----------|-----------|
| 定位 | OpenClaw 编排 Dashboard（Web 优先） | OpenClaw 多机 Mesh 编排器（CLI 优先） |
| Tagline | "The control plane for your OpenClaw swarm" | "One mesh, many claws" |
| 创建时间 | 2026-02-18 | 更早 |
| Stars | 105 | 新项目 |
| 语言 | TypeScript (968 源文件, ~1006 总文件) | Go (~28 文件, 5,541 行) |
| 技术栈 | Next.js 16 + React 19 + Tailwind v4 + shadcn/ui + LangGraph + Zustand + SQLite | Go 1.22+ + net/http + go:embed SPA |
| 依赖数 | 48 生产 + 16 开发 | 2 (Cobra + Viper) |
| 分发方式 | npm (`@swarmclawai/swarmclaw`) + install.sh + Docker | 单二进制 (GoReleaser + Homebrew + go install) |
| License | MIT | MIT |
| 当前版本 | v0.9.6 | v0.2 |
| 文档站 | https://swarmclaw.ai/docs (11 个专题页) | README + docs/ 目录 (6 篇设计文档) |

---

## 二、功能点逐项对比

### 2.1 核心编排能力

| 功能 | SwarmClaw | claw-mesh | 差距 |
|------|:---------:|:---------:|------|
| 节点注册/发现 | ✅ Gateway profiles + 外部 runtime 注册/心跳 | ✅ coordinator 注册 + 心跳 | SwarmClaw 多了 named profile 管理 |
| 心跳/健康检查 | ✅ 健康检查 + doctor 诊断 | ✅ 30s 探测 + 指数退避 + 自动下线 | claw-mesh 实现更扎实 |
| 消息路由 | ✅ 按 gateway tag/use-case 路由 | ✅ 规则引擎 (GPU/OS/skill) + least-busy | claw-mesh 路由规则更灵活 |
| 多 Gateway 管理 | ✅ 每个 agent 可连不同 gateway | ✅ 多节点各跑独立 Gateway | 思路不同，见定位分析 |
| 能力检测 | ❌ 依赖用户手动配 tag | ✅ 自动检测 OS/arch/GPU/内存/PATH 工具 | claw-mesh 明显领先 |
| 自动路由 | ⚠️ 按 tag 偏好，非自动匹配 | ✅ 能力匹配 + least-busy 策略 | claw-mesh 领先 |
| Failover | ❌ 无 | ⚠️ 健康检查自动下线，路由绕过 | claw-mesh 有基础，都需加强 |

### 2.2 Agent / 任务管理

| 功能 | SwarmClaw | claw-mesh | 差距 |
|------|:---------:|:---------:|------|
| Agent 创建/配置 | ✅ Agent Builder (人格/prompt/skill/MCP) | ❌ 无 agent 概念 | SwarmClaw 大幅领先 |
| 任务看板 (Kanban) | ✅ 7 列拖拽看板 + 全生命周期 | ❌ 无 | SwarmClaw 大幅领先 |
| 任务调度 (Cron) | ✅ Cron 调度 + webhook 触发 | ❌ 无 | SwarmClaw 领先 |
| AI Planning | ✅ 交互式 Q&A 规划流程 | ❌ 无 | SwarmClaw 领先 |
| 任务 Daemon | ✅ 长时间自治工作流 + 心跳安全 | ❌ 无 | SwarmClaw 领先 |
| LangGraph 编排 | ✅ 多 agent workflow + checkpoint + restore | ❌ 无 | SwarmClaw 领先 |
| 多 Agent 聊天室 | ✅ 房间路由 + @mention + reactions | ❌ 无 | SwarmClaw 领先 |

### 2.3 同步与记忆

| 功能 | SwarmClaw | claw-mesh | 差距 |
|------|:---------:|:---------:|------|
| 文件同步 | ⚠️ Remote history sync (细节不明) | ✅ SHA256 manifest + 增量同步 + 冲突检测 | claw-mesh 领先 |
| 身份同步 | ⚠️ 可编辑 SOUL/IDENTITY/USER.md | ✅ 自动同步身份层文件 | claw-mesh 领先 |
| 记忆系统 | ✅ hybrid search + memory graph + FTS5 + vector embeddings + auto-journaling | ⚠️ 同步 MEMORY.md 文件 | SwarmClaw 大幅领先 |
| Skill 管理 | ✅ ClawHub 安装 + 发现 + pin + runtime 选择 | ⚠️ 同步 skill 文件，无管理 UI | SwarmClaw 领先 |

### 2.4 Provider / 集成

| 功能 | SwarmClaw | claw-mesh | 差距 |
|------|:---------:|:---------:|------|
| 多 Provider 支持 | ✅ 15 个 (Claude CLI/Codex CLI/OpenAI/Anthropic/Gemini/DeepSeek/Groq/Ollama 等) | ❌ 仅 OpenClaw Gateway | SwarmClaw 大幅领先 |
| Chat Connectors | ✅ Discord/Slack/Telegram/WhatsApp/Teams/Matrix + 更多 | ❌ 无 | SwarmClaw 大幅领先 |
| MCP 集成 | ✅ MCP server 工具管理 | ❌ 无 | SwarmClaw 领先 |
| Plugin 系统 | ✅ 插件市场 + hooks/tools/UI 扩展 | ❌ 无 | SwarmClaw 领先 |
| OpenClaw 部署管理 | ✅ Smart Deploy (本地/VPS/SSH/Docker) + 生命周期控制 | ⚠️ 自动检测 + auto-install | SwarmClaw 领先 |

### 2.5 安全

| 功能 | SwarmClaw | claw-mesh | 差距 |
|------|:---------:|:---------:|------|
| 认证 | ✅ Access Key + Bearer Token | ✅ Bearer Token + 节点独立 Token | 平手 |
| 加密存储 | ✅ AES-256 加密凭证 | ❌ 无 | SwarmClaw 领先 |
| SSRF 防护 | ❌ 未提及 | ✅ 私有 IP 阻断 | claw-mesh 领先 |
| TLS | ⚠️ 建议反向代理 | ⚠️ 占位未实现 | 都需加强 |
| Rate Limiting | ✅ 登录暴力破解限速 | ❌ 无 | SwarmClaw 领先 |
| Webhook 签名 | ✅ HMAC 验证 | ❌ 无 | SwarmClaw 领先 |

---

## 三、工程与运营对比

### 3.1 代码工程

| 维度 | SwarmClaw | claw-mesh |
|------|-----------|-----------|
| 代码规模 | ~1006 文件, 重量级 | ~28 文件, 极轻量 |
| 测试 | 单元测试 (.test.ts) + 回归套件 + smoke 测试 + MCP 一致性检查 | Bash E2E 验收测试 (5 个场景) |
| CI/CD | GitHub Actions (ci.yml + release.yml) | GitHub Actions (build + GoReleaser) |
| Lint | ESLint + baseline | Go vet (Makefile) |
| Docker | ✅ Dockerfile + docker-compose + sandbox browser image | ❌ 无 |
| 安装方式 | npm/pnpm/yarn/bun + install.sh + Docker + npx one-off | Homebrew + go install + 二进制下载 |

### 3.2 Dashboard / UI

| 维度 | SwarmClaw | claw-mesh |
|------|-----------|-----------|
| 技术栈 | Next.js 16 + React 19 + Tailwind v4 + shadcn/ui + Zustand | 预构建 SPA (go:embed) |
| 页面数 | 15+ (Dashboard/Agents/Tasks/Providers/Connectors/Memory/Plugins/Skills/Schedules/Chatrooms/Activity/Settings...) | 1 页 (拓扑图 + 聊天 + 节点卡片) |
| 移动端 | ✅ 响应式 + BottomSheet 组件 | ⚠️ 基础响应式 |
| 实时更新 | SSE + WebSocket (双端口) | 5s 轮询 |
| 主题 | ✅ Midnight Glass 主题 + dark mode | ⚠️ 暗色渐变 |
| 截图展示 | 7 张高质量截图 | 无 |

### 3.3 文档与运营

| 维度 | SwarmClaw | claw-mesh |
|------|-----------|-----------|
| 官网 | ✅ swarmclaw.ai (Next.js, 精美落地页) | ❌ 无 |
| 文档站 | ✅ 11 个专题文档页 | ❌ 无独立文档站 |
| README 质量 | ✅ 极其详尽 (~800 行), badges, 架构图, 截图, troubleshooting, contributor 表格, star history | ✅ 不错 (~300 行中英双语), 架构图, CLI 参考 |
| CHANGELOG | ✅ 有 | ❌ 无 |
| Contributing 指南 | ✅ README 内含 | ⚠️ 简单提及 |
| 安装引导 | ✅ 2 分钟 Setup Wizard + CLI setup init + 多种安装方式 | ⚠️ init + join 手动流程 |
| npm 发布 | ✅ @swarmclawai/swarmclaw | ❌ N/A (Go 项目) |
| Branding | ✅ Logo (龙虾) + org avatar + 统一视觉 | ⚠️ 无 logo/品牌视觉 |
| SEO/社区 | ✅ 网站 + GitHub topics (9 个) + ClawHub skill | ⚠️ 仅 GitHub repo |
| Demo | ❌ 无在线 demo | ❌ 无 |

---

## 四、定位差异分析

### SwarmClaw 的本质

SwarmClaw 是一个 **Web-first 的 AI Agent 管理平台**。核心价值：
- 在一个漂亮的 Web UI 里管理多个 AI provider 和 agent
- 提供任务看板、调度、编排等项目管理能力
- 桥接各种聊天平台 (Discord/Slack/Telegram/WhatsApp)
- 插件生态和 skill 市场

它更像是一个 **"AI Agent 的 Jira + Slack"**，面向需要管理多个 AI agent 工作流的团队。

### claw-mesh 的本质

claw-mesh 是一个 **Infrastructure-first 的多机编排器**。核心价值：
- 让多台物理机器上的 OpenClaw Gateway 协同工作
- 自动能力检测 + 智能路由（GPU 任务去 GPU 机器，macOS 任务去 Mac）
- 跨机器身份/记忆同步（同一个 AI 助手，多台机器）
- 零依赖单二进制，运维友好

它更像是一个 **"OpenClaw 的 Kubernetes"**，面向有多台机器想统一管理的个人开发者/小团队。

### 关键差异

```
SwarmClaw:  用户 → Web UI → 多个 Agent → 多个 Provider/Gateway
claw-mesh:  用户 → CLI/消息 → Coordinator → 自动路由 → 最佳节点的 Gateway
```

SwarmClaw 是 **"多 agent 管理"**，claw-mesh 是 **"多机器编排"**。
SwarmClaw 让你管理 10 个不同的 AI agent；claw-mesh 让你的 1 个 AI 助手跨 10 台机器工作。

---

## 五、claw-mesh 的优势（不要丢掉）

1. **零依赖单二进制** — SwarmClaw 需要 Node.js 22.6+、48 个 npm 依赖、SQLite native rebuild。claw-mesh 下载即用，这是巨大的运维优势。

2. **自动能力检测** — SwarmClaw 完全依赖手动配置 tag。claw-mesh 自动检测 OS/arch/GPU/内存/PATH 工具，这是核心差异化。

3. **智能路由引擎** — 基于能力匹配 + least-busy 的自动路由，SwarmClaw 没有等价物。

4. **文件同步机制** — SHA256 manifest + 增量同步 + 冲突检测，比 SwarmClaw 的同步更成熟。

5. **身份统一模型** — "同一个 AI 助手跨多台机器" 的概念，SwarmClaw 没有。

6. **SSRF 防护** — 私有 IP 阻断，SwarmClaw 未提及。

7. **极低资源占用** — Go 二进制 ~13MB，无运行时依赖。SwarmClaw 是完整 Node.js 应用。

---

## 六、claw-mesh 需要补齐的短板

### 6.1 紧急（影响用户第一印象）

| 短板 | 建议 | 优先级 |
|------|------|--------|
| **无官网/落地页** | 做一个简单的 GitHub Pages 或 landing page，展示核心价值 | P0 |
| **无 Logo/品牌** | 设计一个简单 logo，统一 README/Dashboard 视觉 | P0 |
| **无 Docker 支持** | 添加 Dockerfile + docker-compose，降低试用门槛 | P0 |
| **Dashboard 截图缺失** | README 里加 Dashboard 截图/GIF，让人一眼看到效果 | P0 |
| **无 CHANGELOG** | 开始维护 CHANGELOG.md | P1 |

### 6.2 重要（功能差距）

| 短板 | 建议 | 优先级 |
|------|------|--------|
| **无任务队列/调度** | 实现基础任务队列 + 重试 + 超时（已在 roadmap） | P1 |
| **Dashboard 功能单薄** | 增加：节点详情页、路由规则管理 UI、同步状态可视化 | P1 |
| **无 WebSocket 实时推送** | 替换 5s 轮询为 WebSocket 事件流 | P1 |
| **无 Connector 集成** | 至少支持 Telegram/Discord 作为消息入口 | P2 |
| **无 Prometheus metrics** | 添加 /metrics 端点（已在 roadmap） | P2 |
| **记忆系统薄弱** | 当前只同步文件，考虑加 memory graph 或结构化记忆 | P2 |

### 6.3 加分项（运营层面）

| 短板 | 建议 | 优先级 |
|------|------|--------|
| **无独立文档站** | 用 Docusaurus/VitePress 搭建，至少覆盖 Getting Started + Architecture + CLI Reference | P1 |
| **无 install.sh 一行安装** | 写一个 curl 安装脚本（README 里已提到但未实现） | P1 |
| **无 Contributing 指南** | 写 CONTRIBUTING.md，吸引社区贡献 | P2 |
| **无 GitHub Topics** | 添加 openclaw/mesh/orchestration/multi-agent 等 topic | P2 |
| **无 Star History / Badges** | README 加 CI badge、Go Report Card、License badge | P2 |
| **无 ClawHub 集成** | 考虑发布 ClawHub skill 让 OpenClaw agent 操作 claw-mesh | P2 |

---

## 七、战略建议

### 不要试图成为 SwarmClaw

SwarmClaw 走的是 **"大而全的 Web 平台"** 路线 — 48 个依赖、968 个源文件、15+ 页面。这条路需要大量前端工程投入，而且已经有人在做了。

### claw-mesh 应该坚持的路线

**"OpenClaw 的基础设施层"** — 做 SwarmClaw 做不到的事：

1. **极致轻量** — 单二进制、零依赖、5 分钟部署。这是 Go 项目的天然优势。
2. **自动化优先** — 自动能力检测、自动路由、自动同步、自动 failover。用户不需要在 Web UI 里手动配置。
3. **身份统一** — "一个 AI 助手，多台机器" 是独特卖点，SwarmClaw 是 "多个 agent，一个面板"。
4. **可组合** — claw-mesh 应该能和 SwarmClaw/Mission Control 等 Dashboard 共存，作为底层编排层。

### 互补而非竞争

最理想的生态位是：

```
用户 → SwarmClaw (Web UI / 任务管理) → claw-mesh (多机路由/同步) → OpenClaw Gateways
```

claw-mesh 做 **网络层和编排层**，SwarmClaw 做 **应用层和 UI 层**。两者可以互补。

### 短期行动计划 (2 周)

1. Docker 支持 + install.sh 一行安装
2. Logo + 落地页 (GitHub Pages)
3. README 加截图/GIF + badges
4. CHANGELOG + Contributing 指南
5. Dashboard 加路由规则管理 UI
6. WebSocket 实时事件推送

### 中期行动计划 (1-2 月)

1. 任务队列 + 重试 + 超时
2. Prometheus metrics
3. 独立文档站
4. Telegram/Discord connector (作为消息入口)
5. ClawHub skill 发布
6. 节点分组

---

## 八、参考链接

- SwarmClaw 官网: https://www.swarmclaw.ai/
- SwarmClaw GitHub: https://github.com/swarmclawai/swarmclaw
- SwarmClaw 文档: https://swarmclaw.ai/docs
- Mission Control (Autensa): https://github.com/crshdn/mission-control
- OpenClaw 生态 Dashboard 评测: https://www.bitdoze.com/best-openclaw-dashboards/
- ClawMetry (监控 Dashboard): https://www.everydev.ai/tools/clawmetry
