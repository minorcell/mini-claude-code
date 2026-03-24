package tui

import "testing"

func TestDetectFilePickerContext(t *testing.T) {
	context := detectFilePickerContext("inspect @tui/mod", len([]rune("inspect @tui/mod")))
	if !context.Active {
		t.Fatalf("expected active file picker context")
	}
	if context.Query != "tui/mod" {
		t.Fatalf("expected query %q, got %q", "tui/mod", context.Query)
	}
	if context.Start != len([]rune("inspect ")) {
		t.Fatalf("unexpected start position %d", context.Start)
	}
}

func TestDetectFilePickerContextIgnoresEmails(t *testing.T) {
	context := detectFilePickerContext("mail me at user@example.com", len([]rune("mail me at user@example.com")))
	if context.Active {
		t.Fatalf("expected inactive file picker context for email-like string")
	}
}

func TestSearchFileCandidatesPrefersBasenamePrefix(t *testing.T) {
	results := searchFileCandidates([]string{
		"docs/model-notes.md",
		"tui/model.go",
		"core/agent.go",
		"cmd/model_runner.go",
	}, "mod", 4)

	if len(results) == 0 {
		t.Fatalf("expected search results")
	}
	if results[0].Path != "tui/model.go" {
		t.Fatalf("expected best match %q, got %q", "tui/model.go", results[0].Path)
	}
}
