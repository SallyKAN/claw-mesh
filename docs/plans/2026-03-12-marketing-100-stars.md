# claw-mesh 开源营销计划：GitHub 100+ Stars

## Context

claw-mesh 是 OpenClaw 的多机编排工具。OpenClaw 是 2026 年增长最快的开源项目（247K+ stars，30-40 万活跃用户，5700+ ClawHub skills）。claw-mesh 解决的是 OpenClaw 用户的真实痛点：多台机器上的 OpenClaw 实例各自为政，无法协同。

核心策略：**深度绑定 OpenClaw 生态，把 claw-mesh 定位为 "OpenClaw 的多机方案"，而不是一个独立项目。** 所有内容、SEO、社区活动都围绕 OpenClaw 关键词展开，最大化借势 OpenClaw 的流量池。

---

## 第一阶段：发布前准备（1-2 天）

### 1.1 定位重塑 — 绑定 OpenClaw

- [x] **README 标题改为**：`claw-mesh — Multi-machine orchestration for OpenClaw`
- [x] **Tagline**：`Run OpenClaw on multiple machines. One AI, all your devices.`
- [ ] **GitHub About**：`Multi-machine orchestration for OpenClaw — route messages to the right node based on capabilities`
- [ ] **GitHub Topics**：`openclaw`, `openclaw-extension`, `openclaw-multi-node`, `ai-agent`, `self-hosted`, `golang`, `mesh`, `orchestration`
- [x] **README 开头第一句话提到 OpenClaw**

### 1.2 视觉素材

- [ ] **Demo GIF**（30 秒，用 VHS 或 asciinema 录制）
  - 展示：`claw-mesh up` → 两个 OpenClaw 节点 join → `claw-mesh send --auto` 自动路由
  - 放在 README 最顶部
- [ ] **Dashboard 截图**：节点拓扑 + 聊天面板，放在 README
- [ ] **Social Preview 图片**（1280x640）：项目名 + "for OpenClaw" + 架构简图

### 1.3 GitHub 仓库打磨

- [x] 添加 badges（build status, Go version, license, stars, release）
- [ ] 确保 v0.1 有正式 GitHub Release + changelog
- [x] 一键安装脚本 `install.sh`
- [ ] 开启 GitHub Discussions（需手动：Settings → Features → 勾选）
- [x] GitHub Issue/PR 模板

### 1.4 内容预写（发布日集中投放）

所有内容的标题和正文都要包含 "OpenClaw" 关键词：
- [ ] Show HN 帖子
- [ ] r/openclaw 帖子（最关键渠道）
- [ ] Dev.to 博客
- [ ] V2EX 帖子
- [ ] 掘金文章
- [ ] X/Twitter thread

---

## 第二阶段：OpenClaw 生态渗透（核心，与发布日同步）

> 这是蹭热度的主战场，优先级高于通用社区

### 2.1 r/openclaw 发帖（最高优先级）

r/openclaw 是 OpenClaw 用户聚集地，精准度远超 HN/r/selfhosted。

帖子角度：**解决 OpenClaw 用户的真实痛点**，不是推销项目。

```
标题：I built a tool to connect multiple OpenClaw instances into one mesh

I run OpenClaw on my Mac (Xcode, Apple Notes) and a Linux VPS (GPU, Docker).
The problem: they're completely isolated. If I ask "generate an image", the Mac
can't use the GPU. If I ask "check my notes", the VPS can't access Apple Notes.

So I built claw-mesh — a lightweight coordinator that connects multiple OpenClaw
gateways into one mesh. Messages auto-route to the right machine based on
capabilities.

How it works:
1. Start coordinator: `claw-mesh up`
2. Join nodes: `claw-mesh join <coordinator-url>` (on each machine)
3. Send messages: they auto-route to the best node

It auto-detects each node's OS, GPU, memory, and installed tools. There's a web
dashboard to monitor everything. Single Go binary, ~13MB, token auth.

Would love feedback from anyone running OpenClaw on multiple machines.

GitHub: https://github.com/SallyKAN/claw-mesh
```

### 2.2 OpenClaw Discord 社区

策略：
1. **先做贡献者，再做推广者** — 先在 Discord 里帮人解答问题，建立信誉（1-2 天）
2. **在合适的频道分享** — 找 #showcase / #projects / #community-tools 频道
3. **用"我遇到了这个问题，所以做了这个"的叙事** — 不要硬推
4. **回答 "how to run OpenClaw on multiple machines" 类问题时自然提及**

### 2.3 OpenClaw GitHub 生态

- [ ] 向 OpenClaw 主仓库提 Issue/Discussion：标题如 "Multi-machine orchestration support — claw-mesh"
- [ ] 向 OpenClaw 的 awesome list / ecosystem 页面提 PR（如果存在）
- [ ] 在 OpenClaw 相关的 GitHub Issues 中自然提及

### 2.4 ClawHub 技能发布

- [ ] 发布一个 claw-mesh 相关的 ClawHub skill（比如 "mesh-status" skill）

### 2.5 OpenClaw 教程/博客蹭流量

- [ ] 写一篇 "How to run OpenClaw on multiple machines with claw-mesh" 教程
- [ ] 在现有 OpenClaw 教程的评论区自然提及

---

## 第三阶段：通用社区投放（发布日 D-Day）

### 时间线

| 时间 | 平台 | 内容 | 备注 |
|------|------|------|------|
| D-Day 早上 | r/openclaw | 帖子（见 2.1 模板） | 最精准渠道，第一个发 |
| 同时 | OpenClaw Discord | #showcase 分享 | 简短，附 GIF |
| 美西 8AM | Hacker News | Show HN 帖子 | 标题带 OpenClaw |
| 同时 | r/selfhosted, r/golang | Reddit 帖子 | 各调整角度 |
| 同时 | Dev.to | 英文博客 | "How to orchestrate OpenClaw across machines" |
| 北京 9AM | V2EX | 分享创造 | 中文，讲 OpenClaw 多机痛点 |
| 同时 | 掘金 | 技术文章 | 偏 Go 实现 + OpenClaw 集成 |
| 全天 | X/Twitter | Thread | @OpenClaw 官方账号 |
| 全天 | 即刻 | 动态 | 中文简短版 |

### Show HN 帖子模板

```
Show HN: claw-mesh – connect multiple OpenClaw instances into one mesh

OpenClaw is great on one machine, but I run it on three — Mac (Xcode),
Linux (GPU), VPS (public IP). They can't share capabilities.

claw-mesh is a lightweight Go coordinator that connects them. Messages
auto-route based on what each machine can do. "Generate an image" → GPU
node. "Build the iOS app" → Mac.

- Single binary, ~13MB
- Auto-detects OS, GPU, memory, tools
- Web dashboard
- Token auth, self-hosted

GitHub: https://github.com/SallyKAN/claw-mesh
```

### V2EX 帖子模板

```
标题：[分享创造] claw-mesh — OpenClaw 多机编排，让 AI 助手跨机器协同

用 OpenClaw 的朋友应该都有这个痛点：Mac 上跑一个实例，Linux 上跑一个，
但它们各自为政。Mac 有 Xcode 但没 GPU，Linux 有 GPU 但没 Apple Notes。

claw-mesh 把多个 OpenClaw Gateway 组成一个 mesh，消息自动路由到合适的节点。

- Go 单二进制，~13MB
- 自动检测节点能力
- 内置 Web Dashboard
- 支持记忆/身份跨机器同步（v0.2）

如果你也在多台机器上跑 OpenClaw，欢迎试用反馈。

GitHub: https://github.com/SallyKAN/claw-mesh
```

---

## 第四阶段：持续推广（D+1 到 D+14）

### 4.1 社区互动（每天 15-30 分钟）

- 回复所有平台的评论
- 在 r/openclaw 和 OpenClaw Discord 持续活跃
- 把反馈转化为 GitHub Issues

### 4.2 二次内容

| 时间 | 平台 | 角度 |
|------|------|------|
| D+3 | 知乎 | 回答 "OpenClaw 如何多机部署" 类问题 |
| D+3 | 少数派 | "我的 OpenClaw 多机工作流" 体验文 |
| D+5 | Reddit r/devops | DevOps/infra 角度 |
| D+7 | Awesome Lists PR | awesome-selfhosted, awesome-go, awesome-openclaw |

### 4.3 v0.2 发布 = 第二波推广

v0.2 的 sync 功能是杀手级卖点：
- 新 Demo GIF 展示 sync
- 新帖子角度："Your OpenClaw now remembers across machines"
- 再发一轮 r/openclaw + Discord + V2EX

---

## 第五阶段：长期 OpenClaw 生态绑定

### 5.1 SEO 蹭流量

写 SEO 文章，标题全部包含 "OpenClaw"：
- "How to run OpenClaw on multiple machines"
- "OpenClaw multi-node setup guide"
- "OpenClaw + GPU server: route AI tasks to the right machine"

### 5.2 成为 OpenClaw 生态的"官方"多机方案

- 持续在 OpenClaw 社区提供多机相关的技术支持
- 争取被 OpenClaw 官方文档/生态页面收录
- 长期目标：当有人问 "how to run OpenClaw on multiple machines" 时，claw-mesh 是第一个被提到的答案

### 5.3 Skool 社区

- 加入 OpenClaw Lab, OpenClaw Users 社区
- 分享多机使用经验，自然引流

---

## 关键指标

| 指标 | 目标 | 追踪 |
|------|------|------|
| GitHub Stars | 100+ (4 周) | GitHub |
| r/openclaw 帖子 upvotes | 50+ | Reddit |
| OpenClaw Discord 提及 | 被 pin 或被官方认可 | Discord |
| "openclaw multi machine" 搜索排名 | 首页 | Google |
| ClawHub skill 安装量 | 100+ | ClawHub |

---

## 优先级排序（按 ROI）

1. **r/openclaw 发帖** — 最精准的用户群，转化率最高
2. **OpenClaw Discord 分享** — 活跃社区，即时反馈
3. **Demo GIF** — 影响所有渠道转化率
4. **SEO 博客（标题含 OpenClaw）** — 长尾被动流量
5. **Show HN** — 大流量但不够精准
6. **V2EX 分享创造** — 中文最大流量
7. **ClawHub skill 发布** — 被动流量入口
8. **Awesome lists PR** — 长期被动流量

---

## 注意事项

- 所有内容标题和正文都要自然包含 "OpenClaw" 关键词
- 不要在 OpenClaw 社区硬推 — 先贡献价值，再提项目
- 在 r/openclaw 发帖要遵守版规，用 "I built" 叙事而非广告
- OpenClaw Discord 先混脸熟再分享项目
- 不要贬低 OpenClaw 单机能力，定位是"增强"而非"替代"
- HN 标题可以不带 OpenClaw（避免被认为是蹭热度），但正文要提
