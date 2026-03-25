# mini-opencode 架构设计文档

## 1. 模块划分

当前项目按 6 个模块组织：

```txt
cmd/          启动编排与依赖组装
config/       配置加载、默认值、系统提示词
provider/     统一 LLM 接口与多厂商适配器
core/         agent loop、session、turn 结果与进度事件
tools/        工具注册中心、内置工具、安全拦截
tui/          Bubble Tea 终端界面
```

启动路径如下：

```txt
config.Load
  -> config.EffectiveWorkspace
  -> buildClient
  -> tools.DefaultRegistry
  -> core.NewAgent
  -> core.NewSession
  -> tui.Run
```

## 2. config 模块

### 2.1 职责

- 加载 `~/.mini-opencode/config.yaml`
- 在配置不存在时自动写入默认文件
- 规范化 `provider.type`
- 解析工作区与环境变量
- 通过 `//go:embed` 提供内置系统提示词

### 2.2 核心类型

```go
type Config struct {
    Provider    ProviderConfig `yaml:"provider"`
    MaxTokens   int            `yaml:"max_tokens"`
    MaxSteps    int            `yaml:"max_steps"`
    Temperature float64        `yaml:"temperature"`
    Workspace   string         `yaml:"workspace"`
}

type ProviderConfig struct {
    Name      string `yaml:"name"`
    Type      string `yaml:"type"`
    URL       string `yaml:"url"`
    EnvAPIKey string `yaml:"env_api_key"`
    ModelID   string `yaml:"model_id"`
}
```

### 2.3 关键接口

| 方法 | 说明 |
| --- | --- |
| `Default()` | 返回默认配置 |
| `DefaultPath()` | 返回默认配置文件路径 |
| `Load(path)` | 读取配置，不存在时自动初始化 |
| `SystemPrompt()` | 返回嵌入的系统提示词 |
| `EffectiveModel()` | 返回生效模型 ID |
| `EffectiveProviderType()` | 返回规范化后的 provider 类型 |
| `ProviderAPIKey()` | 按 `env_api_key` 读取真实 API Key |
| `EffectiveWorkspace(cwd)` | 解析最终工作区绝对路径 |

### 2.4 默认值

- `openai -> gpt-4.1-mini`
- `anthropic -> claude-3-7-sonnet-latest`
- `gemini -> gemini-2.0-flash`
- `openai-compatible -> 不提供默认 url / env_api_key / model_id`
- `max_tokens -> 1024`
- `max_steps -> 24`
- `temperature -> 0.2`

## 3. provider 模块

### 3.1 职责

`provider` 定义统一请求/响应结构，并分别适配 OpenAI、Anthropic、Gemini 三种协议。

### 3.2 统一接口

```go
type Client interface {
    Name() string
    Complete(ctx context.Context, req Request) (Response, error)
}
```

### 3.3 统一数据模型

```go
type Message struct {
    Role       Role       `json:"role"`
    Content    string     `json:"content,omitempty"`
    ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
    ToolCallID string     `json:"tool_call_id,omitempty"`
    Name       string     `json:"name,omitempty"`
}

type ToolCall struct {
    ID        string          `json:"id"`
    Name      string          `json:"name"`
    Arguments json.RawMessage `json:"arguments"`
}

type ToolDefinition struct {
    Name        string         `json:"name"`
    Description string         `json:"description"`
    InputSchema map[string]any `json:"input_schema"`
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

### 3.4 实现文件

| 文件 | 作用 |
| --- | --- |
| `openai.go` | OpenAI 与 OpenAI-compatible |
| `anthropic.go` | Anthropic Claude |
| `gemini.go` | Gemini |
| `http.go` | 共享 HTTP POST 与错误处理 |

当前没有引入外部 LLM SDK，全部 provider 调用都是手写 HTTP 客户端。

## 4. tools 模块

### 4.1 核心职责

- 导出模型可见的工具定义
- 执行工具调用
- 在工具执行前后串联拦截器
- 把结果统一渲染成 JSON 文本回写给模型

### 4.2 核心接口

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

```go
type State struct {
    WorkingDir string
    SessionID  string
}

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

### 4.3 Registry

```go
type Registry struct { ... }

func NewRegistry(interceptors ...Interceptor) *Registry
func (r *Registry) Register(tool Tool) error
func (r *Registry) MustRegister(tool Tool)
func (r *Registry) Definitions() []provider.ToolDefinition
func (r *Registry) Execute(ctx context.Context, call provider.ToolCall, state State) (Result, error)
```

`Execute()` 的顺序：

1. 解析 `provider.ToolCall`
2. 执行所有 `Before()`
3. 执行目标工具
4. 逆序执行所有 `After()`
5. 返回 `Result`

### 4.4 当前默认工具

`tools.DefaultRegistry()` 当前注册以下工具：

| 工具名 | 文件 | 说明 |
| --- | --- | --- |
| `bash` | `bash.go` | `/bin/sh -lc` 执行命令 |
| `read` | `read_tool.go` | 读取文件并返回带行号内容 |
| `write` | `write_tool.go` | 覆盖或追加写入文件 |
| `edit` | `edit_tool.go` | 精确替换旧内容 |
| `list` | `list_tool.go` | 列目录，可递归 |
| `glob` | `glob_tool.go` | 基于 Go 标准 glob 匹配路径 |
| `grep` | `grep_tool.go` | 搜索文件内容 |
| `todo` | `todo_tool.go` | 维护完整 todo 列表 |
| `webfetch` | `webfetch.go` | 抓取网页或接口 |

### 4.5 当前默认拦截器

| 拦截器 | 说明 |
| --- | --- |
| `WorkspaceInterceptor` | 校验 `bash.working_dir` 不逃逸工作区 |
| `ShellSafetyInterceptor` | 拦截明显危险的 shell 命令片段 |

补充说明：

- 文件工具本身也会使用 `SafeJoin()` 校验路径
- `Result.Render()` 会把工具结果序列化成 JSON，作为 tool message 回写给模型

## 5. core 模块

### 5.1 职责

- 维护单轮 turn 的执行闭环
- 管理会话消息历史
- 汇总事件和 token 使用量
- 向 TUI 发出逐步执行事件

### 5.2 核心类型

```go
type AgentConfig struct {
    Model       string
    MaxTokens   int
    Temperature float64
    MaxSteps    int
    WorkingDir  string
}

type TurnResult struct {
    Events []Event
    Usage  provider.Usage
    Steps  int
}

type Session struct {
    ID        string
    StartedAt time.Time
    Messages  []provider.Message
}
```

### 5.3 Progress 事件

```go
const (
    ProgressEventStepStarted
    ProgressEventStepCompleted
    ProgressEventAssistantMessage
    ProgressEventToolStarted
    ProgressEventToolFinished
)
```

这些事件通过 `TurnObserver` 推给上层 UI，用于实时刷新状态。

### 5.4 turn 循环

当前 `Agent.runTurn()` 的逻辑是：

1. 把用户消息追加到 `Session.Messages`
2. 在 `max_steps` 范围内循环
3. 调用 `provider.Client.Complete()`
4. 记录 usage、assistant 消息和进度事件
5. 如果响应里没有 `ToolCalls`，turn 结束
6. 如果有工具调用，逐个执行 `registry.Execute()`
7. 把工具结果作为 `provider.RoleTool` 消息回写到 session
8. 超出 `max_steps` 时返回错误

注意：当前 turn 是否继续，实际由“本轮是否返回工具调用”决定；`StopReason` 会保留在统一响应里，但主循环当前不单独依赖它做分支。

## 6. tui 模块

### 6.1 职责

- 提供终端交互界面
- 管理 Composer、Conversation、Context 三个区域
- 把 `core` 的进度事件渲染成用户可见状态
- 支持排队消息、恢复草稿、文件候选补全

### 6.2 启动接口

```go
type App struct {
    Agent        *core.Agent
    Session      *core.Session
    ConfigPath   string
    ProviderName string
    ProviderType string
    ModelName    string
    MaxSteps     int
    Workspace    string
}

func Run(app App) error
```

`Run()` 会以 Alt Screen + mouse cell motion 模式启动 Bubble Tea 程序。

### 6.3 主要组件

```go
type model struct {
    input      textarea.Model
    transcript viewport.Model
    activity   viewport.Model
    spinner    spinner.Model
    progress   progress.Model
    help       help.Model

    queuedPrompt string
    filePicker   filePickerState
    todoItems    []todoSidebarItem
    // ...
}
```

关键点：

- 输入组件是 `textarea.Model`，不是单行 `textinput`
- `transcript` 对应 `Conversation`
- `activity` 当前渲染的是 `Context` 侧栏，不是旧设计中的独立 trace viewport
- `filePickerState` 支持在 Composer 中输入 `@` 后做文件候选补全
- `queuedPrompt` 允许在当前运行中排队 1 条下一条消息

### 6.4 交互行为

- `Enter` 发送；运行中再按 `Enter` 则排队消息
- `Ctrl+J` 插入换行
- `Esc` 恢复已排队草稿
- `Esc Esc` 中断当前 turn
- `@` 打开文件候选
- 鼠标滚轮滚动对话记录
- 终端宽度小于 `120` 时，`Context` 会堆叠到 `Conversation` 下方

## 7. cmd 模块

`cmd/mini-opencode/main.go` 负责唯一的组装入口：

1. `config.DefaultPath()`
2. `config.Load()`
3. `cfg.EffectiveWorkspace(cwd)`
4. `buildClient(cfg)`
5. `tools.DefaultRegistry(workspace)`
6. `core.NewAgent(...)`
7. `core.NewSession(config.SystemPrompt())`
8. `tui.Run(app)`

`buildClient()` 根据 `provider.type` 分发到：

- `provider.NewOpenAIClient`
- `provider.NewAnthropicClient`
- `provider.NewGeminiClient`

## 8. 当前实现边界

- `glob` 使用 Go `filepath.Glob`，不支持把 `**` 当成递归匹配
- `grep` 当前返回匹配行文本，`context` 参数还没有展开成上下文块输出
- 右侧 `Context` 面板当前聚焦状态和 todo，不单独显示逐步 trace 列表
- provider 调用当前是 request/response 式，没有 token 级 streaming
