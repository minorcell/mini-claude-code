package tui

import "testing"

func TestQueueComposerDraft(t *testing.T) {
	m := newModel(App{MaxSteps: 24})
	m.busy = true
	m.activePane = paneComposer
	m.input.SetValue("queued draft")

	m.queueComposerDraft()

	if got := m.queuedPrompt; got != "queued draft" {
		t.Fatalf("queuedPrompt = %q, want %q", got, "queued draft")
	}
	if got := m.input.Value(); got != "" {
		t.Fatalf("input = %q, want empty", got)
	}
	if m.composerEditable() {
		t.Fatal("composerEditable() = true, want false while a queued prompt exists")
	}
}

func TestRestoreQueuedPrompt(t *testing.T) {
	m := newModel(App{MaxSteps: 24})
	m.busy = true
	m.activePane = paneConversation
	m.queuedPrompt = "queued draft"

	m.restoreQueuedPrompt()

	if got := m.queuedPrompt; got != "" {
		t.Fatalf("queuedPrompt = %q, want empty", got)
	}
	if got := m.input.Value(); got != "queued draft" {
		t.Fatalf("input = %q, want %q", got, "queued draft")
	}
	if m.activePane != paneComposer {
		t.Fatalf("activePane = %v, want composer", m.activePane)
	}
}

func TestHandleEscapeArmsThenInterrupts(t *testing.T) {
	m := newModel(App{MaxSteps: 24})
	m.busy = true

	m.handleEscape()
	if m.escArmedUntil.IsZero() {
		t.Fatal("escArmedUntil was not set on first Esc")
	}

	m.handleEscape()
	if !m.escArmedUntil.IsZero() {
		t.Fatal("escArmedUntil was not cleared on second Esc")
	}
	if got := m.status; got != "Interrupt requested for the current run" {
		t.Fatalf("status = %q, want interrupt requested", got)
	}
}

func TestAcceptSelectedFileCandidateInMultilineComposer(t *testing.T) {
	m := newModel(App{MaxSteps: 24})
	value := "outline plan\ninspect @tui/mod"
	cursor := len([]rune(value))

	m.input.SetValue(value)
	m.setComposerCursorOffset(cursor)

	context := detectFilePickerContext(m.input.Value(), m.composerCursorOffset())
	if !context.Active {
		t.Fatal("expected active file picker context")
	}

	m.filePicker = context
	m.filePicker.Results = []fileCandidate{{Path: "tui/model.go"}}

	if !m.acceptSelectedFileCandidate() {
		t.Fatal("acceptSelectedFileCandidate() = false, want true")
	}

	want := "outline plan\ninspect @tui/model.go "
	if got := m.input.Value(); got != want {
		t.Fatalf("input = %q, want %q", got, want)
	}
	if got := m.composerCursorOffset(); got != len([]rune(want)) {
		t.Fatalf("cursor offset = %d, want %d", got, len([]rune(want)))
	}
}
