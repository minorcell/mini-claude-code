package core

import (
	"fmt"
	"time"

	"github.com/minorcell/mini-claude-code/projects/mini-claude-code-go-whit-library/provider"
)

type Session struct {
	ID        string
	StartedAt time.Time
	Messages  []provider.Message
}

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

func (s *Session) Add(message provider.Message) {
	s.Messages = append(s.Messages, message)
}
