# mini-claude-code 架构设计文档

## 1. 模块划分

系统由 6 个模块组成，职责清晰、接口明确：

```
cmd/          入口与启动编排
config/       配置加载与系统提示词
provider/     LLM 适配层（统一接口，多厂商实现）
core/         智能体循环与会话管理
tools/        工具系统与安全拦截
tui/          终端用户界面
```

依赖方向：`cmd → tui → core → provider + tools → config`，无循环依赖。

## 2. 模块职责与接口

### 2.1 config — 配置管理

**职责：** 加载 YAML 配置、提供默认值、字段规范化。同时通过 `//go:embed` 提供内置系统提示词。

**对外暴露的类型：**

```go
type Config struct {
    Provider    ProviderConfig
    MaxTokens   int
    Temperature float64
    Workspace   string
}

type ProviderConfig struct {
    Name      string
    Type      string   // "openai" | "openai-compatible" | "anthropic" | "gemini"
    URL       string
    EnvAPIKey string
    ModelID   string
}
```

**对外暴露的方法：**

| 方法 | 返回值 | 说明 |
|------|--------|------|
| `Default()` | `Config` | 返回可直接启动的默认配置 |
| `Load(path)` | `(Config, error)` | 从文件加载配置；文件不存在时自动创建目录并写入默认配置 |
| `SystemPrompt()` | `string` | 返回内置的系统提示词（从嵌入的 prompt.md） |
| `EffectiveModel()` | `string` | 返回生效的模型 ID |
| `EffectiveProviderType()` | `string` | 返回规范化后的提供商类型 |
| `ProviderAPIKey()` | `string` | 从环境变量读取 API Key |
| `EffectiveWorkspace(cwd)` | `(string, error)` | 返回工作区绝对路径 |

**内部实现：**
- 系统提示词从 `config/prompt.md` 通过 `//go:embed` 编译时嵌入，不暴露给用户配置
- `provider.type` 支持别名自动规范化（如 `compatible` → `openai-compatible`）
- 首次运行时，`Load()` 自动在 `~/.mini-claude-code/` 创建目录并写入默认 `config.yaml`

---

### 2.2 provider — LLM 适配层

**职责：** 定义统一的 LLM 调用接口，屏蔽不同厂商 API 的差异。

**核心接口：**

```go
type Client interface {
    Name() string
    Complete(ctx context.Context, req Request) (Response, error)
}
```

**统一数据模型：**

```go
type Message struct {
    Role       Role        // "system" | "user" | "assistant" | "tool"
    Content    string
    ToolCalls  []ToolCall
    ToolCallID string
    Name       string
}

type ToolCall struct {
    ID        string
    Name      string
    Arguments json.RawMessage
}

type ToolDefinition struct {
    Name        string
    Description string
    InputSchema map[string]any
}

type Request struct {
    Model       string
    Messages    []Message
    Tools       []ToolDefinition
    MaxTokens   int
    Temperature float64
}

type Response struct {
    Message    Message
    StopReason StopReason
    Usage      Usage
}
```

**实现：**

| 文件 | 客户端 | 说明 |
|------|--------|------|
| `openai.go` | `OpenAIClient` | OpenAI 及兼容 API |
| `anthropic.go` | `AnthropicClient` | Anthropic Claude |
| `gemini.go` | `GeminiClient` | Google Gemini |

每个客户端在 `Complete()` 内部完成：统一格式 → 原生格式 → HTTP 请求 → 原生响应 → 统一格式的转换。

**共享 HTTP 基础设施（`http.go`）：** 提供 `postJSON()` 辅助函数，封装 HTTP POST 请求、错误处理和响应解析。

---

### 2.3 tools — 工具系统

**职责：** 提供可扩展的工具执行框架，支持拦截器管道进行安全检查。

**核心接口：**

```go
type Tool interface {
    Definition() provider.ToolDefinition
    Execute(ctx context.Context, invocation Invocation) (Result, error)
}

type Interceptor interface {
    Before(ctx context.Context, invocation *Invocation) error
    After(ctx context.Context, invocation Invocation, result *Result, err error)
}
```

**核心数据类型：**

```go
type Invocation struct {
    ToolName   string
    CallID     string
    WorkingDir string
    SessionID  string
    Arguments  json.RawMessage
    ParsedArgs map[string]any
}

type Result struct {
    ToolName string
    OK       bool
    Output   string
    Metadata map[string]any
}
```

**Registry（工具注册表）：**

```go
type Registry struct { ... }

func (r *Registry) Register(tool Tool)
func (r *Registry) RegisterInterceptor(interceptor Interceptor)
func (r *Registry) Execute(ctx context.Context, invocation Invocation) (Result, error)
func (r *Registry) Definitions() []provider.ToolDefinition
```

`Execute()` 执行管道：

```
拦截器 Before()（按注册顺序）
  → Tool.Execute()
  → 拦截器 After()（按注册逆序）
  → 返回 Result
```

**内置工具：**

| 工具 | 方法 | 说明 |
|------|------|------|
| FileSystemTool | read, write, list, mkdir, stat | 文件系统操作，read 限制 64KB |
| BashTool | execute | Shell 命令执行，默认超时 20s，最大 2min，输出 64KB |
| WebFetchTool | fetch | HTTP GET + HTML 标签剥离，内容 64KB |

**内置拦截器：**

| 拦截器 | 检查时机 | 说明 |
|--------|----------|------|
| WorkspaceInterceptor | Before | 确保所有路径在工作区内（通过 `SafeJoin()`） |
| ShellSafetyInterceptor | Before | 拦截危险命令（rm -rf /、mkfs、shutdown 等） |

---

### 2.4 core — 智能体与会话

**职责：** 编排 LLM 调用和工具执行的循环，管理对话历史。

**核心类型：**

```go
type AgentConfig struct {
    Model       string
    MaxTokens   int
    Temperature float64
    MaxSteps    int       // 最大工具循环步数，默认 24
    WorkingDir  string
}

type Agent struct { ... }

func New(client provider.Client, registry *tools.Registry, cfg AgentConfig) *Agent
func (a *Agent) RunTurn(ctx context.Context, session *Session, userMessage string) (TurnResult, error)
func (a *Agent) RunTurnWithObserver(ctx context.Context, session *Session, userMessage string, observer TurnObserver) (TurnResult, error)

type TurnResult struct {
    Events []Event
    Usage  provider.Usage
    Steps  int
}

type Event struct {
    Kind       EventKind  // "assistant" | "tool"
    Content    string
    ToolName   string
    ToolInput  string
    ToolOutput string
    IsError    bool
}

type Session struct {
    ID        string
    StartedAt time.Time
    Messages  []provider.Message
}

func NewSession(systemPrompt string) *Session
```

**Agent.RunTurn() 循环逻辑：**

```
1. 将用户消息添加到 Session.Messages
2. 循环（最多 MaxSteps 步）：
   a. 调用 provider.Client.Complete(session.Messages, toolDefinitions)
   b. 将 assistant 消息添加到 Session.Messages
   c. 如果 StopReason == "end_turn" → 返回
   d. 对每个 ToolCall：
      - 构造 Invocation，调用 registry.Execute()
      - 将 tool 结果消息添加到 Session.Messages
      - 记录 Event
3. 耗尽 MaxSteps → 返回错误
```

**增量事件流：**
- `step_started`：进入新一步
- `assistant_message`：收到可展示的 assistant 文本
- `tool_started`：即将执行工具
- `tool_finished`：工具执行完成并回写简略结果

**模块间依赖：** Agent 依赖 `provider.Client` 接口和 `tools.Registry`，不依赖任何具体实现。

---

### 2.5 tui — 终端界面

**职责：** 提供交互式终端界面，连接用户输入与 Agent 执行。

**核心类型：**

```go
type App struct { ... }

func Run(systemPrompt string, client provider.Client, registry *tools.Registry, cfg config.Config) error
```

**内部结构（Bubble Tea 架构）：**

```go
type model struct {
    input      textinput.Model
    transcript viewport.Model     // 左侧对话面板
    activity   viewport.Model     // 右侧步骤轨迹面板
    spinner    spinner.Model
    progress   progress.Model
    help       help.Model
    entries    []transcriptEntry
    trace      []activityItem
    busy       bool
    // ...
}
```

**消息类型：**
- `turnProgressMsg`：逐步透传 core 的增量事件
- `turnFinishedMsg`：Agent 回合完成时发送，携带 `core.TurnResult`

**行为：**
- Enter → 在后台 goroutine 执行 `agent.RunTurnWithObserver()`
- TUI 通过 channel 连续接收 step / tool / assistant 事件，并实时刷新双 viewport
- Agent 执行期间锁定输入，但用户仍可切换 pane 并滚动查看上下文
- `turnFinishedMsg` 到达后：更新最终状态、解锁输入、保留完整轨迹

---

### 2.6 cmd — 入口编排

**职责：** 解析配置、组装依赖、启动应用。

`main.go` 中的 `run()` 函数是唯一的组装点：

```
1. config.Load() → 加载配置（不存在则自动初始化）
2. config.EffectiveWorkspace() → 解析工作区
3. buildClient(cfg) → 根据 provider.type 创建 Client
4. tools.DefaultRegistry(workspace) → 创建工具注册表
5. core.NewAgent(client, registry, agentCfg) → 创建 Agent
6. core.NewSession(config.SystemPrompt()) → 创建会话（系统提示词从嵌入文件获取）
7. tui.Run(app) → 启动 UI
```

`buildClient()` 是一个 switch，根据配置类型实例化对应的 `provider.Client` 实现。

## 3. 模块间接口总览

```
                    config.Config
                         │
          ┌──────────────┼──────────────┐
          ▼              ▼              ▼
    config.SystemPrompt  │    config.ProviderConfig
          │              │              │
          │         ┌────┴────┐         │
          │         ▼         ▼         │
          │    provider.Client  tools.Registry
          │    .Complete()      .Execute()
          │         │               │
          └─────────┴───────┬───────┘
                            ▼
                      core.Agent
                      .RunTurn()
                            │
                            ▼
                        tui.Run()
```

**关键接口契约：**

| 上游模块 | 下游接口 | 数据流向 |
|----------|----------|----------|
| `tui` | `core.Agent.RunTurn()` | 用户消息 → TurnResult |
| `core` | `provider.Client.Complete()` | Request → Response |
| `core` | `tools.Registry.Execute()` | Invocation → Result |
| `cmd` | `config.SystemPrompt()` | 获取内置系统提示词 |
| `tui` / `core` | `config.Config` | 读取配置值（只读） |

## 4. 技术栈

| 技术 | 版本 | 用途 |
|------|------|------|
| Go | 1.24.0 | 主语言 |
| Bubble Tea | v1.3.4 | 终端 UI 框架（Elm 架构） |
| Bubbles | v0.20.0 | 预置 TUI 组件（textinput、viewport） |
| Lip Gloss | v1.1.0 | 终端样式与布局 |
| yaml.v3 | v3.0.1 | YAML 配置文件解析 |

不使用任何外部 LLM SDK，所有提供商集成均为手写 HTTP 客户端。

## 5. 可扩展性

### 5.1 新增 LLM 提供商

1. 在 `provider/` 下创建新文件（如 `mistral.go`）
2. 实现 `provider.Client` 接口
3. 在 `cmd/main.go` 的 `buildClient()` 添加 case
4. 在 `config/config.go` 添加默认配置

### 5.2 新增工具

1. 在 `tools/` 下创建新文件
2. 实现 `tools.Tool` 接口
3. 在 `tools/defaults.go` 的 `DefaultRegistry()` 中注册

### 5.3 新增拦截器

1. 实现 `tools.Interceptor` 接口
2. 在 `tools/defaults.go` 的 `DefaultRegistry()` 中注册
