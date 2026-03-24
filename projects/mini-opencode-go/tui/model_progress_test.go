package tui

import (
	"testing"
	"time"

	"github.com/minorcell/mini-claude-code/projects/mini-claude-code-go-whit-library/core"
	"github.com/minorcell/mini-claude-code/projects/mini-claude-code-go-whit-library/provider"
)

func TestHandleProgressUpdatesUsagePerStep(t *testing.T) {
	m := newModel(App{MaxSteps: 24})
	m.sessionUsage = provider.Usage{
		InputTokens:  100,
		OutputTokens: 40,
	}
	m.sessionUsageBase = m.sessionUsage

	m.handleProgress(core.ProgressEvent{
		Kind: core.ProgressEventStepCompleted,
		Step: 1,
		Usage: provider.Usage{
			InputTokens:  12,
			OutputTokens: 5,
		},
		TotalUsage: provider.Usage{
			InputTokens:  12,
			OutputTokens: 5,
		},
	})

	if got := m.currentTurnUsage; got.InputTokens != 12 || got.OutputTokens != 5 {
		t.Fatalf("currentTurnUsage = %+v, want input=12 output=5", got)
	}
	if got := m.sessionUsage; got.InputTokens != 112 || got.OutputTokens != 45 {
		t.Fatalf("sessionUsage = %+v, want input=112 output=45", got)
	}

	m.handleProgress(core.ProgressEvent{
		Kind: core.ProgressEventStepCompleted,
		Step: 2,
		Usage: provider.Usage{
			InputTokens:  9,
			OutputTokens: 3,
		},
		TotalUsage: provider.Usage{
			InputTokens:  21,
			OutputTokens: 8,
		},
	})

	if got := m.currentTurnUsage; got.InputTokens != 21 || got.OutputTokens != 8 {
		t.Fatalf("currentTurnUsage after step 2 = %+v, want input=21 output=8", got)
	}
	if got := m.sessionUsage; got.InputTokens != 121 || got.OutputTokens != 48 {
		t.Fatalf("sessionUsage after step 2 = %+v, want input=121 output=48", got)
	}
}

func TestHandleTurnFinishedReplaysMissingAssistantEvent(t *testing.T) {
	m := newModel(App{MaxSteps: 24})
	m.busy = true
	m.turnStarted = time.Now().Add(-time.Second)
	initialEntries := len(m.entries)

	m.handleTurnFinished(turnFinishedMsg{
		result: core.TurnResult{
			Events: []core.Event{
				{
					Kind:    core.EventAssistant,
					Content: "final answer",
				},
			},
			Steps: 1,
		},
	})

	if len(m.entries) != initialEntries+1 {
		t.Fatalf("entries = %d, want %d", len(m.entries), initialEntries+1)
	}
	last := m.entries[len(m.entries)-1]
	if last.Role != "assistant" || last.Body != "final answer" {
		t.Fatalf("unexpected replayed entry: %+v", last)
	}
}

func TestHandleTurnFinishedDoesNotDuplicateObservedAssistantEvent(t *testing.T) {
	m := newModel(App{MaxSteps: 24})
	m.busy = true
	m.turnStarted = time.Now().Add(-time.Second)
	m.recordTurnEvent(core.Event{
		Kind:    core.EventAssistant,
		Content: "final answer",
	})
	m.entries = append(m.entries, transcriptEntry{
		Role:  "assistant",
		Title: "ASSISTANT",
		Body:  "final answer",
		Meta:  "step 01",
	})
	initialEntries := len(m.entries)

	m.handleTurnFinished(turnFinishedMsg{
		result: core.TurnResult{
			Events: []core.Event{
				{
					Kind:    core.EventAssistant,
					Content: "final answer",
				},
			},
			Steps: 1,
		},
	})

	if len(m.entries) != initialEntries {
		t.Fatalf("entries = %d, want %d", len(m.entries), initialEntries)
	}
}
