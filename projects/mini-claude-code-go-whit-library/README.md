# mini-claude-code

一个最小可运行的 Go 版 `mini-claude-code`，按职责拆分为：

- `core`：agent loop、session、消息流转
- `provider`：统一模型接口，适配 OpenAI / Anthropic / Gemini
- `config`：`~/.mini-claude-code/config.yaml` 配置管理
- `tui`：基于 Bubble Tea 的终端界面
- `tools`：内置工具系统、注册中心、拦截器、工作区约束

## 运行

```bash
go run ./cmd/mini-claude-code
```

## 配置

默认读取 `~/.mini-claude-code/config.yaml`。`provider.name` 是用户自定义的接入名称，`provider.type` 才代表底层 API 协议格式；`url` 可以是官方地址，也可以是兼容网关。

示例：

```yaml
workspace: ~/code/my-project
max_tokens: 1024
temperature: 0.2

provider:
  name: DeepSeek
  type: openai-compatible
  url: https://api.deepseek.com/v1
  env_api_key: DEEPSEEK_API_KEY
  model_id: deepseek-chat
```

官方 OpenAI 也可以这样写：

```yaml
provider:
  name: OpenAI
  type: openai
  url: https://api.openai.com/v1
  env_api_key: OPENAI_API_KEY
  model_id: gpt-4.1-mini
```

`type` 当前支持：

- `openai`
- `openai-compatible`
- `anthropic`
- `gemini`

其中：

- `openai-compatible` 适合 DeepSeek、MiniMax 等 OpenAI 兼容接口
- `name` 可以写任意接入名，不参与协议分发

内置默认值：

- `openai -> url=https://api.openai.com/v1 env_api_key=OPENAI_API_KEY`
- `anthropic -> url=https://api.anthropic.com/v1 env_api_key=ANTHROPIC_API_KEY`
- `gemini -> url=https://generativelanguage.googleapis.com/v1beta env_api_key=GEMINI_API_KEY`
- `openai-compatible -> 不预设 url / env_api_key / model_id，需要用户显式配置`
