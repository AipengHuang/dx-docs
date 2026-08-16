# 帝显知识引擎 (WeKnora) 开发文档

> **版本**: v0.7.2 | **最后更新**: 2026-08-16 | **适用对象**: 帝显公司开发团队
> **原始项目**: 腾讯 WeKnora | **许可证**: MIT
> **GitHub**: https://github.com/Tencent/WeKnora

---

## 目录

- [1. 项目概述](#1-项目概述)
- [2. 技术栈](#2-技术栈)
- [3. 系统架构](#3-系统架构)
- [4. 目录结构](#4-目录结构)
- [5. 核心功能模块](#5-核心功能模块)
- [6. 前端页面](#6-前端页面)
- [7. 后端 API](#7-后端-api)
- [8. 开发环境搭建](#8-开发环境搭建)
- [9. 环境配置详解](#9-环境配置详解)
- [10. 部署方案](#10-部署方案)
- [11. 与帝显平台的关系](#11-与帝显平台的关系)

---

## 1. 项目概述

**WeKnora**（维娜拉）是一个**企业级、LLM 驱动的知识管理与 RAG 框架**，由腾讯开源。核心使命：将分散的企业文档转化为可查询、可推理、持续进化的知识资产。

### 三大核心能力

| 能力 | 说明 |
|------|------|
| **RAG 快速问答** | 基于检索增强生成的知识库快速问答 |
| **ReAct Agent** | 自主多步推理，编排检索、MCP 工具和网络搜索 |
| **Wiki 模式** | Agent 自动生成结构化、互联的 Markdown Wiki 页面 |

### 产品信息

| 项目 | 地址 |
|------|------|
| 官方网站 | https://weknora.weixin.qq.com |
| 微信对话开放平台 | https://chatbot.weixin.qq.com |
| GitHub | https://github.com/Tencent/WeKnora |

### 在帝显平台中的角色

```
┌──────────────────────────────────────────────┐
│              帝显 AI 平台                      │
│                                              │
│  dixian-platform → dixian-app → dx-docs      │
│  (控制面/网关)    (运行时/UI)    (WeKnora)    │
│  NestJS 11        FastAPI+React   Go+Vue     │
│                                  知识引擎     │
│                                  RAG/Agent   │
│                                  Wiki/图谱    │
└──────────────────────────────────────────────┘

通信链路:
  - dixian-platform → WeKnora: 内网健康探针 (服务身份)
  - dixian-app → WeKnora: 知识检索、Agent 执行
  - WeKnora docreader: gRPC 文档解析服务
```

---

## 2. 技术栈

### 2.1 后端 (Go)

| 技术 | 版本 | 用途 |
|------|------|------|
| **Go** | 1.26.0 | 主语言 |
| **Gin** | v1.12.0 | Web 框架 |
| **dig** | — | 依赖注入 |
| **Viper** | — | 配置管理 (YAML + 环境变量) |
| **pgx** | v5 | PostgreSQL 驱动 |
| **go-sql-driver/mysql** | — | MySQL 驱动 |
| **sqlite + sqlite-vec** | — | SQLite + 向量搜索 (Lite 模式) |
| **DuckDB** | — | 分析型查询 |
| **go-redis** | v9 | Redis 客户端 |
| **Asynq** | — | 异步任务队列 (基于 Redis) |
| **golang-jwt** | v5 | JWT 认证 |
| **AES-256-GCM** | — | 加密 |
| **gin-swagger** | — | API 文档 |
| **gRPC + Protobuf** | — | docreader 通信 |
| **Langfuse** | — | 追踪与可观测性 |
| **mcp-go** | mark3labs | MCP SDK |
| **chromedp / goquery** | — | 网页抓取 |
| **Neo4j driver** | — | 知识图谱 |
| **AWS S3 SDK** | — | 对象存储 |
| **cron** | robfig/v3 | 定时任务 |

### 2.2 前端 (Vue 3)

| 技术 | 版本 | 用途 |
|------|------|------|
| **Vue** | 3.5+ | UI 框架 (Composition API) |
| **TypeScript** | 6.0+ | 开发语言 |
| **Vite** | 7.3+ | 构建工具 |
| **TDesign Vue Next** | v1.19.2 | UI 组件库 (腾讯设计系统) |
| **Pinia** | 3.0+ | 状态管理 |
| **Vue Router** | 4.5+ | 路由 |
| **vue-i18n** | 11.4+ | 国际化 |
| **Axios** | — | HTTP 客户端 |
| **marked** | — | Markdown 渲染 |
| **KaTeX** | — | 数学公式 |
| **Mermaid** | — | 图表渲染 |
| **Less** | — | CSS 预处理 |
| **DOMPurify** | — | HTML 净化 |
| **highlight.js** | — | 代码高亮 |

### 2.3 文档解析微服务 (Python)

| 技术 | 用途 |
|------|------|
| Python 3.10+ | 运行时 |
| gRPC server | 与主服务通信 |
| PyPDF | PDF 解析 |
| python-docx | Word 解析 |
| openpyxl | Excel 解析 |
| Pillow | 图片处理 |
| Playwright | 浏览器自动化 |
| lxml / BeautifulSoup | HTML/XML 解析 |
| trafilatura | 网页内容提取 |
| markitdown | Markdown 转换 |
| uv | 包管理 |

### 2.4 MCP Server (Python)

| 技术 | 用途 |
|------|------|
| tencent-weknora-mcp | PyPI 发布包 |
| 29 个工具 | 覆盖知识库、文档、聊天等操作 |
| stdio / SSE / HTTP | 三种传输方式 |

### 2.5 其他组件

| 组件 | 技术 | 说明 |
|------|------|------|
| **CLI** | Go (独立模块) | Agent 优先设计，JSON 信封输出 |
| **文档站点** | VitePress 1.6+ | ~50 页文档，Mermaid 支持 |
| **微信小程序** | 微信开发者工具 | 移动端入口 |
| **桌面应用** | Wails | Go + Vue 桌面壳 |

---

## 3. 系统架构

### 3.1 整体架构

```
┌─────────────────────────────────────────────────────────┐
│                      浏览器 / 客户端                      │
└──────────────────────┬──────────────────────────────────┘
                       │ HTTPS
┌──────────────────────▼──────────────────────────────────┐
│  Vue 3 SPA (Nginx 托管, 端口 80)                        │
│  TDesign UI | Pinia | Axios | marked + KaTeX + Mermaid  │
└──────────────────────┬──────────────────────────────────┘
                       │ HTTP
┌──────────────────────▼──────────────────────────────────┐
│  Go 后端 (Gin, 端口 8080)                               │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐              │
│  │ ReAct    │  │ RAG      │  │ Wiki     │              │
│  │ Agent    │  │ Pipeline │  │ Engine   │              │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘              │
│       │              │              │                    │
│  ┌────▼──────────────▼──────────────▼─────┐             │
│  │  Models (LLM/Embedding/Rerank/VLM)     │             │
│  │  20+ 提供商 | MCP Tools | Web Search    │             │
│  └────────────────────────────────────────┘             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐              │
│  │ Auth     │  │ IM       │  │ Audit    │              │
│  │ JWT/OIDC │  │ 10+ 渠道  │  │ RBAC     │              │
│  └──────────┘  └──────────┘  └──────────┘              │
└──┬────────────┬──────────────┬──────────────┬──────────┘
   │ gRPC       │ Asynq+Redis  │ HTTP         │
┌──▼──────┐ ┌───▼────┐  ┌─────▼───┐  ┌──────▼──────┐
│Docreader│ │ 任务队列│  │向量数据库│  │ Neo4j       │
│(Python) │ │(Redis) │  │8+ 后端  │  │(知识图谱)   │
│文档解析  │ │        │  │         │  │             │
└─────────┘ └────────┘  └─────────┘  └─────────────┘
```

### 3.2 数据库支持

| 类型 | 支持的数据库 | 用途 |
|------|-------------|------|
| **关系型** | PostgreSQL (pgx), MySQL, SQLite, DuckDB | 主数据存储 |
| **向量数据库** | pgvector, ES/OpenSearch, Milvus, Weaviate, Qdrant, Apache Doris, Tencent VectorDB | 向量检索 |
| **图数据库** | Neo4j | 知识图谱 |
| **缓存** | Redis | 缓存 + 任务队列 |

### 3.3 异步任务处理

```
文档上传 → Asynq 任务入队 → Redis 队列
  → Worker 消费 → gRPC 调用 Docreader 解析
  → 文本切分 → 向量化 → 存入向量数据库
  → 完成 → 通知前端
```

---

## 4. 目录结构

```
dx-docs/
├── cmd/server/                      # Go 后端入口
│   ├── main.go                      # 启动入口
│   ├── bootstrap.go                 # 初始化引导
│   └── signals.go                   # 信号处理
│
├── internal/                       # Go 后端内部包
│   ├── agent/                       # ReAct Agent 引擎
│   │   ├── act.go                   # Agent 执行动作
│   │   ├── think.go                # Agent 思考推理
│   │   ├── finalize.go             # Agent 结果输出
│   │   ├── tools.go                # 工具编排
│   │   ├── skills.go               # 技能系统
│   │   └── prompts.go              # Prompt 模板
│   ├── handler/                     # HTTP 处理器 (70+ 文件)
│   │   ├── auth_*.go               # 认证相关
│   │   ├── knowledge_*.go          # 知识库相关
│   │   ├── chat_*.go               # 聊天相关
│   │   ├── agent_*.go              # Agent 相关
│   │   ├── mcp_*.go                # MCP 相关
│   │   └── ...                     # 其他 60+ 处理器
│   ├── router/                      # Gin 路由定义
│   │   ├── routes_agent.go         # Agent 路由
│   │   ├── routes_auth_tenant.go   # 认证/租户路由
│   │   ├── routes_chat.go          # 聊天路由
│   │   ├── routes_infra.go         # 基础设施路由
│   │   └── routes_knowledge.go      # 知识库路由
│   ├── models/                      # LLM/Embedding/Rerank/VLM 模型集成
│   ├── im/                          # IM 集成 (WeCom, Feishu, Slack, Telegram, DingTalk, QQ 等)
│   ├── mcp/                         # MCP 集成
│   ├── middleware/                   # 认证, CORS, 限流中间件
│   ├── config/                      # 配置类型和加载
│   ├── container/                   # DI 容器
│   ├── database/                    # 数据库访问层
│   ├── infrastructure/               # 共享基础设施
│   ├── datasource/                   # 外部数据源连接器
│   ├── tracing/                     # Langfuse 可观测性
│   ├── runtime/                     # 启动环境日志
│   ├── sandbox/                     # Agent 技能沙箱
│   ├── ratelimit/                   # 限流
│   ├── stream/                      # 流式响应处理
│   └── types/interfaces/            # 服务接口定义
│
├── frontend/                       # Vue 3 SPA
│   ├── src/
│   │   ├── views/                   # 页面组件 (14 个视图目录)
│   │   ├── components/              # 可复用组件 (60+ 文件)
│   │   ├── api/                     # API 客户端模块 (20+ 模块)
│   │   ├── stores/                  # Pinia 状态管理
│   │   ├── router/                  # Vue Router 配置
│   │   ├── i18n/                    # 国际化
│   │   ├── composables/             # Vue 组合式函数
│   │   ├── hooks/                   # 自定义 Hooks
│   │   └── wailsjs/                # Wails 桌面应用绑定
│   ├── vite.config.ts
│   └── package.json
│
├── docreader/                       # Python gRPC 文档解析微服务
│   ├── parser/                      # 文档解析器
│   ├── splitter/                    # 文本切分器
│   ├── proto/                       # gRPC protobuf 定义
│   └── pyproject.toml
│
├── mcp-server/                       # Python MCP 服务
│   ├── weknora_mcp_server.py
│   ├── pyproject.toml
│   └── tests/
│
├── cli/                              # Go CLI 工具 (独立模块)
│   ├── cmd/
│   ├── internal/
│   └── skills/
│
├── website-docs/                     # VitePress 文档站点 (~50 页)
│   ├── 01-getting-started/           # 快速开始 (4 页)
│   ├── 02-architecture/             # 架构设计 (5 页)
│   ├── 03-features/                 # 功能特性 (21 页)
│   ├── 04-api/                      # API 文档 (12 页, ~360 端点)
│   ├── 05-clients/                  # 客户端 (7 页)
│   ├── 06-development/              # 开发指南 (3 页)
│   └── .vitepress/                  # VitePress 配置
│
├── config/                           # YAML 配置文件
│   ├── config.yaml                  # 主配置 (服务器、对话、知识库、抽取)
│   ├── builtin_agents.yaml           # 内置 Agent 定义
│   ├── builtin_models.yaml           # 声明式内置模型
│   └── prompt_templates/             # Prompt 模板 YAML
│
├── migrations/                       # 数据库迁移
│   ├── mysql/                       # MySQL 迁移
│   ├── paradedb/                    # ParadeDB (PostgreSQL + pgvector) 迁移
│   ├── sqlite/                      # SQLite 迁移
│   └── versioned/                    # 版本化迁移
│
├── docker/                           # Docker 镜像定义
│   ├── Dockerfile.app               # Go 后端多阶段构建
│   ├── Dockerfile.docreader         # Python 文档解析服务
│   ├── Dockerfile.sandbox           # Agent 技能沙箱
│   └── Dockerfile.odl-hybrid        # ODL 混合模式
│
├── helm/                             # Kubernetes Helm Chart
│   ├── Chart.yaml
│   ├── values.yaml
│   └── templates/
│
├── miniprogram/                      # 微信小程序
├── scripts/                          # 构建/部署脚本 (18 个)
├── skills/                           # 预加载 Agent 技能
├── tests/                            # 集成/e2e 测试
├── dataset/                          # 示例数据集
├── knowledge_sources/               # 知识源文件
├── deploy/                           # 部署配置
├── railway/                          # Railway 部署配置
├── Formula/                          # Homebrew 安装包
├── examples/                         # 示例代码
├── docker-compose.yml                # 主 Docker Compose (48KB)
├── docker-compose.dev.yml            # 开发模式 Docker Compose
├── Makefile                           # 构建目标
├── .air.toml                          # Air 热重载配置
├── .env.example                       # 环境变量参考 (~40KB, ~150 变量)
├── go.mod / go.sum                    # Go 依赖管理
├── CHANGELOG.md                       # 更新日志 (142KB)
├── README.md / README_CN.md           # 英文/中文 README
├── LICENSE                            # MIT 许可证
└── VERSION                            # 0.7.2
```

---

## 5. 核心功能模块

### 5.1 知识库管理

| 功能 | 说明 |
|------|------|
| **知识库类型** | FAQ 知识库、文档知识库、Wiki 知识库 |
| **文件夹树** | 多级文件夹组织 |
| **Chunk 编辑** | 文本块编辑，带版本历史 |
| **批量操作** | 批量上传、删除、移动 |

### 5.2 文档解析

支持 **12+ 格式**的文档解析：

| 格式 | 解析器 | 说明 |
|------|--------|------|
| PDF | PyPDF, opendataloader | 文本/表格/图片提取 |
| Word | python-docx | .docx 文档 |
| Excel | openpyxl | .xlsx 表格 |
| PPT | python-pptx | .pptx 演示文稿 |
| 图片 | Pillow + VLM | OCR + 视觉理解 |
| HTML | lxml, BeautifulSoup | 网页解析 |
| EPUB | ebooklib | 电子书解析 |
| Markdown | — | 直接解析 |
| 纯文本 | — | 直接使用 |
| 音频 | ASR | 语音转文本 (VLM) |
| 视频 | ASR | 视频转文本 (VLM) |
| 网页 | Playwright + trafilatura | 动态网页抓取 |

**解析流程**: 通过 gRPC 调用 docreader 微服务 → 文本提取 → 结构化输出 → 传入切分器

### 5.3 RAG 管道

```
用户提问 → 查询预处理 → BM25 稀疏检索 + 密集向量检索
  → 结果融合 → Rerank 重排序 → 父子 Chunk 扩展
  → 上下文组装 → LLM 生成回答
```

| 检索策略 | 说明 |
|----------|------|
| **BM25** | 基于关键词的稀疏检索 |
| **Dense Retrieval** | 基于向量相似度的密集检索 |
| **Reranking** | 交叉编码器重排序 |
| **Parent-Child Chunking** | 父子分块，检索子块返回父块 |
| **GraphRAG** | 基于知识图谱的检索增强 |
| **HNSW** | pgvector HNSW 索引加速 |

### 5.4 Agent 引擎 (ReAct)

```
用户提问
  → Think: 分析问题，决定下一步
  → Act: 调用工具 (检索/MCP/Web Search)
  → Observe: 获取结果
  → Think: 继续推理或得出结论
  → Finalize: 输出最终回答
```

| 能力 | 说明 |
|------|------|
| **多步推理** | Think-Act-Observe 循环 |
| **工具编排** | 知识库检索、MCP 工具、网络搜索 |
| **Wiki 生成** | 自动生成结构化 Wiki 页面 |
| **技能沙箱** | 在隔离沙箱中执行技能代码 |

### 5.5 Wiki 模式

| 功能 | 说明 |
|------|------|
| **自动生成** | Agent 从知识库自动生成交互链接的 Markdown 页面 |
| **知识图谱** | 页面间的关联可视化 |
| **版本历史** | 页面修改的完整历史 |
| **Diff/回滚** | 版本差异对比和一键回滚 |

### 5.6 IM 渠道集成

| 渠道 | SDK/协议 |
|------|----------|
| 企业微信 (WeCom) | WeCom API |
| 飞书 (Lark/Feishu) | Lark API |
| Slack | Slack API |
| Telegram | Telegram Bot API |
| 钉钉 (DingTalk) | DingTalk API |
| Mattermost | Mattermost API |
| 微信 | WeChat API |
| QQ 机器人 | QQ Bot API |
| 云之家 (Yunzhijia) | Yunzhijia API |

### 5.7 数据源连接

| 数据源 | 说明 |
|--------|------|
| 飞书 Wiki/文档 | 飞书云文档同步 |
| Lark 文档 | Lark 文档同步 |
| Notion | Notion 数据库/页面同步 |
| 语雀 | 语雀知识库同步 |
| RSS | RSS 订阅源同步 |

### 5.8 存储后端

| 后端 | SDK | 说明 |
|------|-----|------|
| 本地存储 | — | 文件系统 |
| MinIO | MinIO SDK | 自建对象存储 |
| AWS S3 | AWS SDK | Amazon S3 |
| 腾讯云 TOS | TOS SDK | 腾讯对象存储 |
| 阿里云 OSS | OSS SDK | 阿里云对象存储 |
| 金山云 KS3 | KS3 SDK | 金山云存储 |
| 华为云 OBS | OBS SDK | 华为云存储 |

支持每个工作空间配置多个存储实例。

### 5.9 RBAC 权限

| 角色 | 权限范围 |
|------|----------|
| **Owner** | 工作空间最高权限 |
| **Admin** | 管理员权限 |
| **Contributor** | 内容贡献权限 |
| **Viewer** | 只读权限 |

附加功能：每个知识库独立所有者、审计日志、作用域 API Key。

### 5.10 可观测性

| 工具 | 监控内容 |
|------|----------|
| **Langfuse** | LLM 调用追踪、Agent 推理链、成本统计 |
| **文档解析时间线** | 每个文档的解析进度和耗时 |
| **任务队列面板** | Asynq 任务状态、失败重试 |

---

## 6. 前端页面

| 路由 | 视图组件 | 用途 |
|------|----------|------|
| `/login` | `auth/Login.vue` | 登录/邀请注册 |
| `/register` | `auth/Login.vue` | 分享链接邀请注册 |
| `/onboarding/workspace` | `auth/WorkspaceOnboarding.vue` | 首次工作空间创建 |
| `/platform/knowledge-bases` | `knowledge/KnowledgeBaseList.vue` | **主知识库列表** (默认着陆页) |
| `/platform/knowledge-bases/:kbId` | `knowledge/KnowledgeBase.vue` | 知识库详情 + 文档列表 |
| `/platform/settings` | `settings/Settings.vue` | 工作空间/组织设置 |
| `/platform/agents` | `agent/AgentList.vue` | Agent 管理 |
| `/platform/creatChat` | `creatChat/creatChat.vue` | 创建新对话 |
| `/platform/chat/:chatid` | `chat/index.vue` | 对话界面 |
| `/platform/organizations` | `organization/OrganizationList.vue` | 多组织管理 |
| `/embed` (独立入口) | `embed/` | 网站嵌入组件 |

---

## 7. 后端 API

### 7.1 概览

- **约 150+ 路由组** 在 `/api/v1/` 下
- **Swagger 文档** 自动生成，访问 `/swagger/*`
- **约 360 个 API 端点**，覆盖 11 个类别

### 7.2 路由文件

| 文件 | 路由范围 | 说明 |
|------|----------|------|
| `routes_agent.go` | Agent 相关 | Agent 管理、执行 |
| `routes_auth_tenant.go` | 认证/租户 | 登录、注册、工作空间 |
| `routes_chat.go` | 聊天 | 对话、消息、会话 |
| `routes_infra.go` | 基础设施 | 模型、存储、系统管理 |
| `routes_knowledge.go` | 知识库 | 知识库、文档、Chunk |

### 7.3 Handler 类别

| 类别 | 说明 |
|------|------|
| Auth | 认证/登录/注册 |
| Knowledge Base | 知识库 CRUD |
| Knowledge | 知识点管理 |
| Chat/Session/Message | 对话系统 |
| Model | LLM 模型管理 |
| Agent/MCP | Agent 和 MCP 工具 |
| WebSearch | 网络搜索 |
| DataSource | 外部数据源 |
| Embed | 网站嵌入 |
| Wiki/FAQ | Wiki 和 FAQ |
| Storage/VectorStore | 存储和向量库 |
| IM | IM 渠道管理 |
| System Admin | 系统管理 |
| Audit | 审计日志 |
| Tenant/Org | 租户和组织 |
| Evaluation | 评估 |
| Tags/Skills | 标签和技能 |

---

## 8. 开发环境搭建

### 8.1 前置要求

| 工具 | 版本 | 用途 |
|------|------|------|
| Go | 1.26.0 | 后端编译 |
| Node.js | 20+ | 前端构建 |
| Python | 3.10+ | docreader |
| PostgreSQL | 15+ | 主数据库 (推荐 ParadeDB) |
| Redis | 7+ | 缓存 + 任务队列 |
| Docker + Compose | 最新 | 容器编排 |

### 8.2 Docker Compose 快速启动 (推荐)

```bash
# 1. 克隆并配置
cp .env.example .env
# 编辑 .env (见第 9 节环境配置)

# 2. 启动核心服务
docker compose up -d
# 启动: frontend (Nginx:80), app (Go:8080), ollama (:11434), PostgreSQL, Redis

# 3. 启动可选服务 (按需)
docker compose --profile full up -d        # 所有可选服务
docker compose --profile neo4j up -d        # Neo4j 知识图谱
docker compose --profile minio up -d        # MinIO 对象存储
docker compose --profile langfuse up -d    # Langfuse 可观测性

# 4. 访问
# 前端: http://localhost
# 后端: http://localhost:8080
# Swagger: http://localhost:8080/swagger/
```

### 8.3 本地开发

```bash
# 1. 安装 Go 依赖
go mod download

# 2. 安装前端依赖
cd frontend && npm install

# 3. 安装 docreader 依赖
cd docreader && uv sync

# 4. 启动后端 (热重载)
make dev-start
# 或使用 Air:
air -c .air.toml

# 5. 启动前端 (另一个终端)
cd frontend && npm run dev

# 6. 启动 docreader (另一个终端)
cd docreader && uv run python -m docreader.server

# 7. 数据库迁移 (自动执行)
# 应用启动时 AUTO_MIGRATE=true (默认) 自动运行迁移
# 手动执行:
make migrate-up
```

### 8.4 Lite 模式 (零外部依赖)

```bash
# 构建嵌入式单二进制 (内嵌前端 + SQLite + sqlite-vec)
make build-lite

# 运行
make run-lite
# 或
./weknora-lite --config config.yaml
```

适合快速体验和轻量部署，无需 PostgreSQL/Redis/向量数据库。

---

## 9. 环境配置详解

`.env.example` 包含约 **150 个环境变量**，按类别分组：

### 9.1 配置类别

| 类别 | 关键变量 | 说明 |
|------|----------|------|
| **A. 部署基础** | `IMAGE_TAG`, `RUNTIME` | 镜像版本、运行时配置 |
| **B. 数据与存储** | `DB_TYPE`, `DB_HOST`, `DB_PORT`, `DB_NAME`, `REDIS_URL` | 数据库和缓存连接 |
| **C. 检索与图谱** | `VECTOR_DB_TYPE`, `NEO4J_URI` | 向量数据库、图数据库 |
| **D. 模型** | `LLM_PROVIDER`, `LLM_MODEL`, `OPENAI_API_KEY`, `OLLAMA_BASE_URL` | LLM、VLM、本地模型 |
| **E. 文档解析** | `DOCREADER_URL`, `PARSER_TIMEOUT` | docreader 配置、超时设置 |
| **F. 认证与工作空间** | `JWT_SECRET`, `AES_KEY`, `RBAC_ENABLED`, `OIDC_*` | JWT、加密、RBAC、OIDC |
| **G. Agent 与沙箱** | `AGENT_SKILLS_DIR`, `SANDBOX_TIMEOUT` | 技能目录、沙箱超时 |
| **H. 可选集成** | `WEB_SEARCH_*`, `MCP_*` | 网络搜索、MCP 服务 |
| **I. 可观测性** | `LANGFUSE_*` | Langfuse 追踪 |
| **J. 安全与调优** | `SSRF_PROXY`, `MAX_CONCURRENCY`, `RATE_LIMIT_*` | SSRF 防护、并发控制、限流 |

### 9.2 最小化配置

以下为启动所需的最小环境变量：

```bash
# .env (最小配置)
DB_TYPE=paradedb
DB_HOST=localhost
DB_PORT=5432
DB_NAME=weknora
DB_USER=weknora
DB_PASSWORD=your_password
REDIS_URL=redis://localhost:6379
JWT_SECRET=your_jwt_secret_min_32_chars
AES_KEY=your_aes_key_min_32_chars
```

---

## 10. 部署方案

### 10.1 Docker Compose (主部署)

```yaml
# docker-compose.yml (48KB, 完整生产部署)
services:
  frontend:       # Vue SPA (Nginx, 端口 80)
  app:            # Go 后端 (端口 8080, 健康检查)
  ollama:         # 本地 Embedding 模型 (端口 11434)
  postgres:       # PostgreSQL + ParadeDB 扩展
  redis:          # Redis 7
  # 可选:
  minio:          # MinIO 对象存储
  neo4j:          # Neo4j 知识图谱
  langfuse:       # Langfuse 可观测性
  docreader:      # 文档解析服务
  searxng:        # 网络搜索
```

### 10.2 Docker 镜像

| 镜像 | 说明 |
|------|------|
| `wechatopenai/weknora-app` | Go 后端 (多阶段 Alpine 构建) |
| `wechatopenai/weknora-docreader` | Python 文档解析服务 |
| `wechatopenai/weknora-ui` | Vue 前端 (Nginx) |

### 10.3 Kubernetes (Helm)

```bash
# 使用 Helm Chart 部署
cd helm
helm install weknora . -f values.yaml
```

Chart 配置项在 `helm/values.yaml` 中定义。

### 10.4 桌面应用

```bash
# macOS 打包
make package-mac-app
```

基于 Wails 的桌面应用，支持 macOS 打包。

### 10.5 数据库迁移

```bash
# 自动迁移 (启动时默认执行, AUTO_MIGRATE=true)
# 手动执行:
make migrate-up          # 执行迁移
make migrate-down        # 回滚迁移

# 支持的迁移后端:
# migrations/mysql/       MySQL 迁移
# migrations/paradedb/    ParadeDB (PostgreSQL + pgvector) 迁移
# migrations/sqlite/      SQLite 迁移
# migrations/versioned/   版本化迁移
```

### 10.6 Makefile 命令

```bash
make build                 # 构建 Go 后端
make test                  # 运行测试
make lint                  # 代码检查
make fmt                   # 代码格式化
make docker-build-all      # 构建所有 Docker 镜像
make dev-start             # 启动开发环境基础设施
make dev-app               # 启动 Go 后端 (Air 热重载)
make dev-frontend          # 启动 Vue 前端 (Vite 热重载)
make build-lite            # 构建嵌入式单二进制
make run-lite              # 运行 Lite 模式
make package-lite          # 打包 Lite 版本
make package-mac-app        # 打包 macOS 桌面应用
```

---

## 11. 与帝显平台的关系

### 11.1 三仓库协作

```
┌────────────────────────────────────────────────────────────┐
│                     帝显 AI 平台                            │
│                                                            │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐    │
│  │ dixian-      │  │ dixian-app   │  │ dx-docs      │    │
│  │ platform     │  │ (LambChat)   │  │ (WeKnora)    │    │
│  │              │  │              │  │              │    │
│  │ 身份/组织/   │──│── Agent UI   │──│── 知识引擎   │    │
│  │ RBAC/门户    │  │ 运行时       │  │ RAG/Agent    │    │
│  └──────────────┘  └──────────────┘  └──────────────┘    │
│                                                            │
│  浏览器 ──HTTPS──→ platform ──内网──→ app ──内网──→ WeKnora│
│  (唯一入口)      (控制面)         (运行时)    (知识引擎)  │
└────────────────────────────────────────────────────────────┘
```

### 11.2 WeKnora 在帝显平台中的职责

| 职责 | 说明 |
|------|------|
| **知识检索** | 为 LambChat Agent 提供知识库检索能力 |
| **Agent 执行** | ReAct Agent 在知识库上进行多步推理 |
| **文档解析** | 通过 gRPC docreader 微服务解析企业文档 |
| **Wiki 生成** | 自动构建企业知识 Wiki |
| **知识图谱** | Neo4j 存储和查询知识关联 |

### 11.3 通信方式

| 方向 | 协议 | 认证 |
|------|------|------|
| dixian-platform → WeKnora | 内网 HTTP (健康探针) | 服务身份标识 Header |
| LambChat → WeKnora | 内网 HTTP | 服务令牌 |
| app → docreader | gRPC | gRPC 凭据 |

---

## 附录：开发命令速查

```bash
# ===== 开发 =====
make dev-start               # 启动基础设施
make dev-app                 # 启动 Go 后端 (热重载)
make dev-frontend            # 启动 Vue 前端 (热重载)

# ===== 构建 =====
make build                   # 构建 Go 后端
make docker-build-all        # 构建所有 Docker 镜像
make build-lite              # 构建嵌入式单二进制

# ===== 测试 =====
make test                    # 运行测试
make lint                    # 代码检查
make fmt                     # 代码格式化

# ===== 运行 =====
make run-lite                # 运行 Lite 模式 (零依赖)

# ===== 部署 =====
docker compose up -d                         # 核心服务
docker compose --profile full up -d          # 全部服务
helm install weknora ./helm                 # Kubernetes

# ===== 数据库 =====
make migrate-up             # 执行迁移
make migrate-down           # 回滚迁移

# ===== 打包 =====
make package-lite           # 打包 Lite 版本
make package-mac-app         # 打包 macOS 桌面
```
