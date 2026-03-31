# Issue: TUI Textarea 输入域交互设计方案

## 问题描述

需要为 mini-opencode-go 设计一个友好的多行输入域（textarea），支持：

1. 多行文本输入
2. 换行与发送的快捷键区分
3. 符合用户习惯的交互方式

---

## 设计方案

### 1. 技术选型

使用 **Bubble Tea** 框架的 `bubbles/textarea` 组件：

- 成熟的 TUI 组件库
- 内置光标管理、滚动、样式系统
- 与 Bubble Tea 架构无缝集成

### 2. 核心实现

#### 2.1 初始化配置

```go
import "github.com/charmbracelet/bubbles/textarea"

input := textarea.New()
input.Placeholder = "Ask mini-opencode to inspect, edit, or debug..."
input.Prompt = ">> "
input.ShowLineNumbers = false
input.CharLimit = 4096
input.SetHeight(3)

// 关键：添加 Ctrl+J 作为换行快捷键（Shift+Enter 也能换行）
input.KeyMap.InsertNewline.SetKeys("ctrl+j")
input.KeyMap.InsertNewline.SetHelp("ctrl+j", "newline")
input.Focus()
```

#### 2.2 快捷键绑定

```go
type keyMap struct {
    Send            key.Binding
    CandidateAccept key.Binding
    Up              key.Binding
    Down            key.Binding
    Quit            key.Binding
}

func newKeyMap() keyMap {
    return keyMap{
        Send: key.NewBinding(
            key.WithKeys("enter"),
            key.WithHelp("enter", "send/queue"),
        ),
        // ... 其他绑定
    }
}
```

#### 2.3 消息处理

```go
case tea.KeyMsg:
    if keyMatches(message, m.keys.Send) {
        if m.activePane == paneComposer {
            if m.busy {
                return m.queueComposerDraft()  // 忙碌时排队
            }
            return m.startTurn()  // 空闲时发送
        }
    }
```

### 3. 交互设计

| 快捷键                   | 功能      | 说明                                 |
| ------------------------ | --------- | ------------------------------------ |
| `Enter`                  | 发送/排队 | 空闲时直接发送，忙碌时将消息加入队列 |
| `Ctrl+J` / `Shift+Enter` | 插入换行  | 多行文本输入                         |
| `Esc`                    | 召回草稿  | 当有排队消息时，按 Esc 可以召回编辑  |
| `Esc Esc`                | 中断      | 双按 Esc 中断当前运行                |

### 4. UI 提示

在界面底部显示当前可用的快捷键：

```go
func (m model) renderComposer() string {
    hint := "Enter sends | Ctrl+J/Shift+Enter newline"
    if m.busy && m.queuedPrompt != "" {
        hint = "Esc recalls queued draft | Esc Esc interrupts"
    } else if m.busy {
        hint = "Enter queues | Ctrl+J/Shift+Enter newline | Esc Esc interrupts"
    }
    // ... 渲染逻辑
}
```

---

## 实现要点

1. **快捷键冲突处理**：`textarea` 组件默认 `InsertNewline` 绑定 `enter` 和 `ctrl+m`。代码通过 `SetKeys("ctrl+j")` 添加 `Ctrl+J` 作为换行快捷键。此外，`Shift+Enter` 也能换行（由终端或 bubbles 底层处理）

2. **状态管理**：
   - `busy`: 表示当前是否有运行中的任务
   - `queuedPrompt`: 存储排队的消息草稿

3. **消息队列机制**：
   - 忙碌时发送的消息不会丢失
   - 当前任务完成后自动发送队列中的消息

4. **文件选择器支持**：
   - 在输入 `@` 时激活文件选择器
   - `Tab` 或 `Enter` 接受候选
   - `Up/Down` 切换候选

---

## 参考代码位置

- `tui/model.go:145-165` - 输入框初始化
- `tui/keys.go:17-21` - 快捷键绑定
- `tui/model.go:263-271` - 发送消息处理
- `tui/components.go:101-108` - UI 提示文本

---

## 可能的改进

1. 明确绑定 `Shift+Enter` 到 `InsertNewline`（如果终端支持）
2. 可配置的快捷键映射
3. 输入历史记录（上下键浏览历史）
