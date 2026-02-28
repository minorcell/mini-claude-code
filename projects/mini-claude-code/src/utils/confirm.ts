import readline from "readline"

// bash 工具在执行危险命令前调用，等待用户明确输入 y 确认
// 创建临时 rl 实例，不影响外层 CLI 的 readline
export async function confirmFromUser(command: string): Promise<boolean> {
  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
  })

  return new Promise((resolve) => {
    console.log("\n\x1b[33m⚠️  检测到潜在危险命令：\x1b[0m")
    console.log(`   \x1b[90m${command}\x1b[0m`)

    rl.question("\n确认执行? (y/N) ", (answer) => {
      rl.close()
      resolve(answer.trim().toLowerCase() === "y")
    })
  })
}
