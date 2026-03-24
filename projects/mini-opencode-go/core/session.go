package core

import (
	"fmt"
	"time"

	"github.com/minorcell/mini-claude-code/projects/mini-opencode-go/provider"
)

// Session 表示一次对话会话及其消息历史。
type Session struct {
	ID        string
	StartedAt time.Time
	Messages  []provider.Message
}

// NewSession 创建一个新会话，并在存在系统提示时自动写入首条 system 消息。
func NewSession(systemPrompt string) *Session {
	session := &Session{
		ID:        fmt.Sprintf("session-%d", time.Now().UnixNano()),
		StartedAt: time.Now(),
	}
	if systemPrompt != "" {
		session.Messages = append(session.Messages, provider.Message{
			Role:    provider.RoleSystem,
			Content: systemPrompt,
		})
	}
	return session
}

// Add 向当前会话追加一条消息。
func (s *Session) Add(message provider.Message) {
	s.Messages = append(s.Messages, message)
}
