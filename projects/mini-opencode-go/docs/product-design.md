# mini-opencode 产品设计文档

## 1. 产品概述

mini-opencode 是一个基于终端的 AI 编程助手。它提供一个交互式命令行界面，用户通过自然语言与大语言模型（LLM）对话，LLM 以高级软件工程师的身份工作——读代码、写代码、执行命令、调试问题，所有操作在工作区内沙箱化运行。

## 2. 目标用户

- 需要在终端环境中快速获得 AI 编码、调试、文件操作辅助的开发者
- 偏好命令行工作流的技术用户
- 需要轻量级、可配置的 AI 编程助手的用户

## 3. 核心功能

### 3.1 自然语言对话

用户在终端中输入自然语言消息，系统将消息发送给配置的 LLM，获取响应并展示在终端界面中。

**交互方式：**
- 输入消息后按 Enter 发送
- Esc 或 Ctrl+C 退出
- 对话过程中输入区域锁定，但仍可切换 pane 浏览对话和步骤轨迹
- 顶部实时显示 step 进度条，右侧轨迹区逐步展示 assistant 摘要与工具状态

### 3.2 文件系统操作

LLM 可以通过文件系统工具对工作区内的文件进行操作：

| 操作 | 说明 |
|------|------|
| read | 读取文件内容（限制 64KB） |
| write | 写入文件内容，支持追加模式 |
| list | 列出目录下的文件和子目录 |
| mkdir | 创建目录 |
| stat | 获取文件或目录的元信息 |

### 3.3 Shell 命令执行

LLM 可以执行 Shell 命令（通过 `/bin/sh -lc`）：

- 默认超时时间：20 秒
- 最大超时时间：2 分钟
- 输出限制：64KB
- 安全机制：内置危险命令拦截器，阻止 `rm -rf /`、`mkfs`、`shutdown`、`reboot`、`poweroff`、fork bomb 等命令

### 3.4 网页获取

LLM 可以通过 HTTP GET 获取网页内容：

- 自动剥离 HTML 标签，提取纯文本
- 内容限制：64KB
- User-Agent：`mini-opencode/0.1`

### 3.5 多 LLM 提供商支持

系统支持多种 LLM 提供商，用户可通过配置文件切换：

| 提供商 | 默认模型 | 说明 |
|--------|----------|------|
| OpenAI | gpt-4.1-mini | 官方 OpenAI API |
| OpenAI Compatible | gpt-4.1-mini | 兼容 OpenAI API 的第三方服务 |
| Anthropic | claude-3-7-sonnet-latest | Anthropic Claude API |
| Gemini | gemini-2.0-flash | Google Gemini API |

## 4. 用户界面

### 4.1 终端布局

```
+-------------------------------------------------------------------+
| MINI OPENCODE                       [WORKING]                  |
| provider OpenAI   type openai   model gpt-4.1-mini                |
| workspace /path/to/project                                         |
| steps 03/24 [===========-----]   focus trace                      |
+---------------------------------------------+---------------------+
| [Conversation]                              | [Live Trace]        |
| 用户 / assistant / 工具摘要                  | step / tool / 结果   |
+---------------------------------------------+---------------------+
| [Composer] > Ask mini-opencode...                              |
+-------------------------------------------------------------------+
| status / config / help                                            |
+-------------------------------------------------------------------+
```

### 4.2 消息颜色编码

| 角色 | 颜色 |
|------|------|
| 用户（user） | 青色（cyan） |
| 助手（assistant） | 橙色（orange） |
| 工具调用（tool） | 绿色（green） |
| 系统（system） | 白色（white） |

## 5. 系统提示词

系统提示词定义了 AI 的角色定位和行为规范，存储在 `config/prompt.md` 中，通过 Go 的 `//go:embed` 在编译时嵌入二进制文件。系统提示词不暴露给用户配置，由代码直接控制。

**角色定位：** 高级软件工程师。AI 以团队成员的身份工作——先读代码再动手，理解上下文再改代码，对结果负责。

**行为准则：**
- 先读再做，探索代码库后再下结论
- 像真实开发者一样思考和诊断问题
- 自信地做决策，提出方案并说明取舍
- 直接沟通，不说废话

## 6. 配置管理

### 6.1 配置文件

配置文件位于 `~/.mini-opencode/config.yaml`。首次启动时自动创建目录并写入默认配置。

### 6.2 配置项

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| provider.name | string | openai | 提供商显示名称 |
| provider.type | string | openai | 提供商类型：openai / openai-compatible / anthropic / gemini |
| provider.url | string | 依提供商而定 | API 端点地址 |
| provider.env_api_key | string | 依提供商而定 | API Key 对应的环境变量名 |
| provider.model_id | string | 依提供商而定 | 模型 ID |
| max_tokens | int | 1024 | 最大生成 token 数 |
| max_steps | int | 24 | 单轮 agent 最大循环步数 |
| temperature | float | 0.2 | 采样温度 |
| workspace | string | 当前目录 | 工作区路径 |

### 6.3 提供商别名

`provider.type` 支持别名映射，例如 `openai_compatible`、`openaicompatible`、`compatible` 均会被规范化为 `openai-compatible`。

## 7. 安全设计

### 7.1 工作区沙箱

所有文件操作和 Shell 命令均限制在配置的工作区目录内：

- 文件路径通过 `SafeJoin()` 进行规范化，防止路径穿越攻击
- Shell 命令的工作目录限制在工作区内

### 7.2 危险命令拦截

Shell 安全拦截器在命令执行前进行检查，拦截以下危险操作：

- `rm -rf /`：删除根目录
- `mkfs`：格式化磁盘
- `shutdown` / `reboot` / `poweroff`：关机或重启
- Fork bomb（`:(){ :|:& };:`）：耗尽系统资源

## 8. 约束与限制

| 项目 | 限制 |
|------|------|
| 文件读取大小 | 64KB |
| Shell 命令输出 | 64KB |
| 网页内容大小 | 64KB |
| Shell 命令默认超时 | 20 秒 |
| Shell 命令最大超时 | 2 分钟 |
| 工具调用循环最大步数 | 默认 24 步，可通过 `max_steps` 调整 |
