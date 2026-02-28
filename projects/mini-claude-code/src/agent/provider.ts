import { createOpenAI } from "@ai-sdk/openai"

if (!process.env.QINIU_API_KEY) {
  throw new Error("缺少环境变量 QINIU_API_KEY，请参考 .env.example 配置")
}

// 七牛大模型推理服务，兼容 OpenAI 协议
// 文档：https://www.qiniu.com/ai/chat
const qiniu = createOpenAI({
  apiKey: process.env.QINIU_API_KEY,
  baseURL: "https://api.qnaigc.com/v1",
  // 兼容模式：保持 system role，不转成 OpenAI 新规范的 developer role
  // 所有非官方 OpenAI 兼容 API 都应设此项，否则会报 "role developer not supported"
  compatibility: "compatible",
})

const modelName = process.env.QINIU_MODEL ?? "claude-4.6-sonnet"

export const model = qiniu(modelName)
