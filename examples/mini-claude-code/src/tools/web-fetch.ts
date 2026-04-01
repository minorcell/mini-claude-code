import { truncateOutput } from "../utils/truncate"

interface Params {
  url: string
}

export async function webFetch({ url }: Params): Promise<string> {
  let response: Response
  try {
    response = await fetch(url, {
      headers: { "User-Agent": "mini-claude-code/1.0" },
      signal: AbortSignal.timeout(15_000),
    })
  } catch (e) {
    return `错误：请求失败 - ${(e as Error).message}`
  }

  if (!response.ok) {
    return `错误：HTTP ${response.status} ${response.statusText} - ${url}`
  }

  const contentType = response.headers.get("content-type") ?? ""
  const text = await response.text()

  const content = contentType.includes("text/html")
    ? htmlToMarkdown(text)
    : text

  return truncateOutput("web_fetch", content)
}

// 简单的 HTML → Markdown 转换
// 目标：去除标签噪声，保留内容结构，减少 token 消耗
function htmlToMarkdown(html: string): string {
  return html
    // 去掉 <HEAD> 整块
    .replace(/<head[\s\S]*?<\/head>/gi, "")
    // 去掉 <noscript> 整块
    .replace(/<noscript[\s\S]*?<\/noscript>/gi, "")
    // 去掉 <iframe> 整块
    .replace(/<iframe[\s\S]*?<\/iframe>/gi, "")
    // 
    // 去掉 <script> 和 <style> 整块
    .replace(/<script[\s\S]*?<\/script>/gi, "")
    .replace(/<style[\s\S]*?<\/style>/gi, "")
    // 标题
    .replace(/<h1[^>]*>([\s\S]*?)<\/h1>/gi, "\n# $1\n")
    .replace(/<h2[^>]*>([\s\S]*?)<\/h2>/gi, "\n## $1\n")
    .replace(/<h3[^>]*>([\s\S]*?)<\/h3>/gi, "\n### $1\n")
    // 代码块（保留，对 LLM 最有用）
    .replace(/<pre[^>]*><code[^>]*>([\s\S]*?)<\/code><\/pre>/gi, "\n```\n$1\n```\n")
    .replace(/<code[^>]*>([\s\S]*?)<\/code>/gi, "`$1`")
    // 强调
    .replace(/<strong[^>]*>([\s\S]*?)<\/strong>/gi, "**$1**")
    .replace(/<em[^>]*>([\s\S]*?)<\/em>/gi, "*$1*")
    // 链接
    .replace(/<a[^>]*href="([^"]*)"[^>]*>([\s\S]*?)<\/a>/gi, "[$2]($1)")
    // 列表
    .replace(/<li[^>]*>([\s\S]*?)<\/li>/gi, "- $1")
    // 换行
    .replace(/<br\s*\/?>/gi, "\n")
    .replace(/<\/p>/gi, "\n\n")
    .replace(/<\/div>/gi, "\n")
    .replace(/<\/tr>/gi, "\n")
    // 去掉剩余标签
    .replace(/<[^>]+>/g, "")
    // HTML 实体
    .replace(/&amp;/g, "&")
    .replace(/&lt;/g, "<")
    .replace(/&gt;/g, ">")
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'")
    .replace(/&nbsp;/g, " ")
    // 清理多余空行
    .replace(/\n{3,}/g, "\n\n")
    .trim()
}
