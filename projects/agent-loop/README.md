# 最小 Agent：天气查询（Bun + TypeScript）

## 文件

```txt
projects
├── bun.lock
├── tools.ts
├── prompt.md
├── package.json
├── tsconfig.json
└── main.ts
```

## 安装

```bash
cd projects
bun install
```

## 环境变量

由外部注入：

- `DEEPSEEK_API_KEY`（必填）

## 运行

```bash
bun run main.ts "上海现在天气如何？"
```

不传问题会使用默认问题：`上海现在天气如何？`

运行时会打印每一轮：

- `[LLM 第N轮输出]`：模型原始输出
- `<observation>...</observation>`：当本轮有工具调用时，紧跟在 `<action>` 后输出工具返回结果
- `=== 最终回答 ===`：最终回答

## XML 协议

- 工具调用：`<action tool="getWeather">{"city":"上海","time":"2026-02-27 10:00"}</action>`
- 最终回答：`<final>...</final>`

## 说明

- 当前仅包含 2 个工具：`getWeather`（模拟天气）和 `getTime`（当前时间）。
- 天气工具返回本地 mock 数据，不调用真实天气 API。
