# mini-opencode 产品设计文档

## 1. 产品概述

mini-opencode 是一个运行在终端里的 Code Agent。用户用自然语言发出任务，模型在同一工作区内读文件、改文件、执行命令、维护 todo，并把每一步产物写回会话上下文。

当前实现强调三点：

- 终端内完成完整闭环，不依赖外部 GUI
- 工具能力尽量贴近真实开发工作流
- 所有操作都围绕工作区约束和最小安全防护展开

## 2. 目标用户

- 习惯在终端内工作、希望获得 AI 编码辅助的开发者
- 需要一个轻量、可配置、可本地运行的 Code Agent 原型的用户
- 想研究 Agent loop、工具调用和终端交互的工程实现者

## 3. 核心能力

### 3.1 对话与回合执行

用户在 Composer 中输入消息后，系统会启动一次 turn：

1. 将用户消息写入 session
2. 调用当前 provider
3. 如果模型请求工具，执行工具并把结果回写给模型
4. 直到模型不再请求工具，或达到 `max_steps`

当前交互行为：

- `Enter` 发送消息
- 当前 turn 运行中再次按 `Enter`，会排队 1 条后续消息
- `Esc` 可把已排队草稿恢复回 Composer
- `Esc Esc` 可请求中断当前 turn
- `Ctrl+C` 退出程序

### 3.2 工作区文件工具

mini-opencode 当前内置的文件相关工具如下：

| 工具 | 能力 | 说明 |
| --- | --- | --- |
| `read` | 读取文件 | 返回带行号内容，支持 `offset` / `limit` / `max_bytes` |
| `write` | 写入文件 | 自动创建父目录，支持覆盖与追加 |
| `edit` | 精确修改 | 用 `old_content -> new_content` 做定点替换 |
| `list` | 列目录 | 支持递归和显示隐藏文件 |
| `glob` | 名称匹配 | 基于 Go 标准 glob 规则匹配路径 |
| `grep` | 内容搜索 | 支持普通文本或正则搜索，返回 `file:line: content` |

这里不再是旧文档里的 `mkdir` / `stat` 接口；当前实现已经演进为更贴近代码编辑场景的 `edit`、`glob`、`grep` 组合。

### 3.3 Shell 命令执行

`bash` 工具通过 `/bin/sh -lc` 在工作区内执行命令，主要约束如下：

- 默认超时 20 秒
- 最大超时 2 分钟
- 输出上限 64KB
- 支持 `working_dir`
- 对明显危险的命令片段做拦截，例如 `rm -rf /`、`mkfs`、`shutdown`

### 3.4 Todo 与过程状态

`todo` 工具维护当前任务的完整 todo 列表，而不是只追加单项。右侧 `Context` 侧栏会把 todo 状态渲染出来，帮助用户看到当前任务分解情况。

约束：

- 每次更新都应替换整张列表
- 最多允许 1 个 `in_progress`
- 任务结束时可以传空数组清空

### 3.5 网页抓取

`webfetch` 支持抓取 HTTP(S) 页面或接口：

- 只允许 `http` / `https`
- HTML 会被做最小清洗，剥离脚本、样式和标签
- 内容上限 64KB
- User-Agent 为 `mini-opencode/0.1`

### 3.6 多 Provider 支持

系统支持多种模型后端，按统一接口接入：

| 提供商类型 | 默认模型 | 说明 |
| --- | --- | --- |
| `openai` | `gpt-4.1-mini` | 官方 OpenAI API |
| `openai-compatible` | 无默认值 | DeepSeek、MiniMax 等兼容网关 |
| `anthropic` | `claude-3-7-sonnet-latest` | Anthropic Claude API |
| `gemini` | `gemini-2.0-flash` | Google Gemini API |

## 4. 终端界面

### 4.1 布局

当前界面由三个主要区域组成：

- 左侧 `Conversation`
  展示 user / assistant / tool 的完整记录
- 右侧 `Context`
  展示 token 统计、step 进度、状态文案和 todo 列表
- 底部 `Composer`
  输入新消息、排队下一条消息、触发 `@文件` 候选

当终端宽度较小时，`Context` 会堆叠到 `Conversation` 下方，而不是维持固定双栏。

### 4.2 交互细节

- `Conversation` 是主滚动区域，鼠标滚轮会滚动这里
- `Context` 当前是被动信息侧栏，不是独立的 trace 操作面板
- 在 Composer 中输入 `@` 会触发文件候选搜索
- 文件候选来自工作区索引，默认跳过 `.git`、`node_modules`、`vendor`、`dist`、`.cache` 等重目录
- 窗口过小会直接提示最小尺寸要求：`72x18`

## 5. 配置管理

### 5.1 配置文件

默认配置文件路径：

```txt
~/.mini-opencode/config.yaml
```

首次启动时，如果文件不存在，会自动创建目录并写入默认配置。

### 5.2 核心配置项

| 配置项 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `provider.name` | string | `openai` | 仅用于界面显示 |
| `provider.type` | string | `openai` | 协议类型 |
| `provider.url` | string | 依 provider 而定 | API 端点 |
| `provider.env_api_key` | string | 依 provider 而定 | API Key 环境变量名 |
| `provider.model_id` | string | 依 provider 而定 | 模型 ID |
| `max_tokens` | int | `1024` | 单次模型调用的最大生成 token |
| `max_steps` | int | `24` | 单轮 agent 最大步数 |
| `temperature` | float | `0.2` | 采样温度 |
| `workspace` | string | 启动时当前目录 | 工作区根目录 |

### 5.3 规范化行为

- `provider.type` 支持别名折叠，例如 `compatible` 会转成 `openai-compatible`
- `workspace` 支持 `~` 展开
- `workspace`、`provider.url`、`provider.model_id` 支持环境变量展开

## 6. 安全设计

### 6.1 工作区约束

所有文件路径都会经过 `SafeJoin()` 校验，防止逃逸出工作区。`bash` 的 `working_dir` 也会经过同样约束。

### 6.2 危险命令拦截

Shell 安全拦截器会在执行前阻断明显危险的命令片段，例如：

- `rm -rf /`
- `mkfs`
- `shutdown`
- `reboot`
- `poweroff`
- fork bomb `:(){:|:&};:`

## 7. 当前边界

| 项目 | 当前行为 |
| --- | --- |
| `glob` 语义 | 使用 Go `filepath.Glob`，不把 `**` 视为递归 doublestar |
| `Context` 面板 | 当前主要展示状态和 todo，不单独渲染逐步 trace 列表 |
| `grep` 输出 | 返回匹配行，不展开上下文块 |
| 内容上限 | `read` / `bash` / `webfetch` 默认都有限长保护 |
