# 安全设计

## 为什么 Agent 需要安全机制

ChatBot 只输出文字，最坏情况是说错话。Agent 可以执行 Shell 命令、修改文件——最坏情况是删库、泄露密钥、打爆磁盘。

安全机制不是为了防止 LLM 故意作恶，而是防止：
- LLM 误判任务范围（把清理临时文件理解成 `rm -rf /`）
- 提示词注入（用户粘贴的代码里藏了 `rm -rf ~` 的注释）
- 意外的副作用（删除了用户没想到会被影响的文件）

`mini-claude-code` 实现三道防护：危险命令检测、路径安全、工具输出沙箱。

---

## 第一道：危险命令检测

`bash` 工具执行前，先过一遍危险命令检测。分两级：

```typescript
// utils/safety.ts

export type DangerLevel = 'safe' | 'confirm' | 'block'

// block 级：直接拒绝，不询问用户
// 这些命令几乎没有合法的 Agent 使用场景
const BLOCK_PATTERNS: RegExp[] = [
  /rm\s+-rf\s+(\/|~|\$HOME)\b/,       // rm -rf / 或 rm -rf ~
  /dd\s+if=.*of=\/dev\//,             // dd 写入磁盘设备
  /mkfs\./,                            // 格式化文件系统
  />\s*\/dev\/(sda|hda|nvme)/,        // 重定向写入磁盘
  /chmod\s+-R\s+777\s+\//,           // 全盘 777
  /shutdown|reboot|halt/,              // 系统关机
]

// confirm 级：暂停，等待用户明确确认
const CONFIRM_PATTERNS: RegExp[] = [
  /rm\s+(-r|-f|-rf|-fr)\s+/,          // 任何 rm -r 或 rm -f
  />\s*[^|&>]/,                        // 重定向覆盖（> 写文件）
  /sudo\s+/,                           // sudo 命令
  /curl.*\|\s*(sh|bash|zsh)/,         // curl pipe to shell（常见攻击手法）
  /wget.*\|\s*(sh|bash|zsh)/,
  /npm\s+publish/,                     // 发布包
  /git\s+push\s+.*--force/,           // 强制推送
  /git\s+reset\s+--hard/,             // 危险的 git 操作
  /DROP\s+TABLE|DROP\s+DATABASE/i,    // SQL 删表
]

export function detectDanger(command: string): DangerLevel {
  if (BLOCK_PATTERNS.some(p => p.test(command))) return 'block'
  if (CONFIRM_PATTERNS.some(p => p.test(command))) return 'confirm'
  return 'safe'
}
```

### 用户审批流程（confirm 级）

```typescript
// tools/bash.ts（审批逻辑）

import * as readline from 'readline'

async function promptUserConfirm(command: string): Promise<boolean> {
  // 在 CLI 打印警告
  console.log('\n⚠️  检测到潜在危险命令：')
  console.log(`   ${command}`)
  console.log('\n是否继续执行？(y/N) ')

  // 暂停等待用户输入
  const answer = await readLineInput()
  return answer.trim().toLowerCase() === 'y'
}
```

审批结果也会作为工具输出返回给 LLM：

```
用户拒绝执行命令：rm -rf ./dist
```

LLM 收到这个结果后，会在下一步思考替代方案（比如先列出目录内容确认，再执行删除）。

---

## 第二道：路径安全

文件操作工具（read/write/edit）都经过路径解析，防止路径穿越攻击：

```typescript
// utils/safety.ts

export function resolveSafePath(inputPath: string): string {
  const cwd = process.cwd()
  const resolved = resolve(cwd, inputPath)

  // 确保解析后的路径在当前工作目录内
  if (!resolved.startsWith(cwd)) {
    throw new Error(
      `路径越界：${inputPath} 解析为 ${resolved}，超出工作目录 ${cwd}`
    )
  }

  return resolved
}
```

这防止了：
- `../../etc/passwd`（读取系统文件）
- `../../../home/user/.ssh/id_rsa`（读取 SSH 私钥）
- `../other-project/sensitive.ts`（读取其他项目文件）

---

## 第三道：工具输出控制

这道防线主要针对两个场景：

**1. 防止敏感信息直接出现在 LLM 上下文中**

如果 `read_file` 读取了 `.env` 文件，其中包含 API Key，这些 key 会出现在发送给 LLM 的请求里——意味着它们会经过 LLM 服务商的服务器。

简单的防护：检测工具访问路径，对敏感文件提示警告：

```typescript
const SENSITIVE_PATTERNS = [
  /\.env(\.|$)/,
  /\.aws\/credentials/,
  /\.ssh\/(id_rsa|id_ed25519)$/,
  /secrets?\.(json|yaml|yml)/i,
]

function isSensitivePath(path: string): boolean {
  return SENSITIVE_PATTERNS.some(p => p.test(path))
}
```

**2. 防止 bash 命令打印 Token/密钥**

这类问题更难完全防护（LLM 可能通过 `env` 命令打印环境变量）。在 `mini-claude-code` 的教学范围内，主要通过工具描述提示 LLM 避免此类操作：

```
bash 工具说明（系统提示词中）：
避免执行会打印敏感信息的命令（如 env、printenv）。
```

---

## 教学设计说明

### 为什么不用沙箱（Docker/e2b）？

沙箱（隔离环境）是更彻底的安全方案：给 Agent 一个独立容器，所有操作都在容器里，不影响宿主机。七牛有专用 Agent 沙箱，e2b 也是这类产品。

`mini-claude-code` 没用沙箱，原因：
1. **教学目的**：沙箱会引入额外的环境配置，分散注意力
2. **工具本身就是危险源**：mini 版工具已经有了基本防护，对教学场景足够
3. **真实感**：让学员直接看到 bash 工具在本地执行，比在沙箱里更直观

如果你要把这套代码用于生产，沙箱是必须考虑的下一步。

### 危险命令检测的局限性

正则检测有局限：

```bash
# 这些会被检测到
rm -rf ./dist
sudo npm install

# 这些正则不容易检测，但同样危险
find / -name "*.log" -delete
python3 -c "import os; os.system('rm -rf /')"
```

完整的安全方案需要更复杂的命令解析，甚至系统调用级别的拦截。`mini-claude-code` 的检测覆盖最常见的几种，足够教学演示。真实产品请参考更完整的安全方案。
