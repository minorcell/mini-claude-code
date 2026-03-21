# 最小 Agent：天气查询（Bun + TypeScript）

一个最小可运行的 Agent Loop 示例，演示：模型输出 XML 动作、调用工具、接收 observation、再产出最终回答。

## 文件结构

```txt
projects/agent-loop
├── main.ts
├── package.json
├── prompt.md
├── README.md
├── tools.ts
└── tsconfig.json
```

## 快速开始

1. 安装

```bash
cd projects/agent-loop
bun install
```

2. 环境变量（推荐 .env）

```bash
cp .env.example .env
```

将 `.env` 中的以下字段替换为真实值：

- `DEEPSEEK_API_KEY`（必填）

可选：仅当前终端临时注入

```bash
export DEEPSEEK_API_KEY="your_api_key"
```

3. 启动

```bash
bun run start
```

## 运行示例

```bash
bun run start "杭州昨天天气怎么样？"
```

不传问题时，默认：`上海现在天气如何？`

## 运行输出说明

- `[LLM 第N轮输出]`：模型原始输出
- `<observation>...</observation>`：当本轮有工具调用时，紧跟在 `<action>` 后输出工具返回结果
- `=== 最终回答 ===`：最终回答

## XML 协议

- 工具调用：`<action tool="getWeather">{"city":"上海","time":"2026-02-27 10:00"}</action>`
- 最终回答：`<final>...</final>`

## 说明

- 当前仅包含 2 个工具：`getWeather`（模拟天气）和 `getTime`（当前时间）。
- 天气工具返回本地 mock 数据，不调用真实天气 API。
