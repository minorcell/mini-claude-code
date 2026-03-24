package tools

import (
	"context"
	"testing"
)

func TestTodoToolStoresItems(t *testing.T) {
	tool := NewTodoTool()

	result, err := tool.Execute(context.Background(), Invocation{
		SessionID: "session-1",
		Arguments: []byte(`{"items":[{"content":"inspect code","status":"completed"},{"content":"patch tui","status":"in_progress"}]}`),
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if got := result.Output; got != "updated 2 todo item(s)" {
		t.Fatalf("unexpected output %q", got)
	}
	if got := metadataInt(result.Metadata["count"]); got != 2 {
		t.Fatalf("count = %d, want 2", got)
	}
	if got := metadataInt(result.Metadata["completed"]); got != 1 {
		t.Fatalf("completed = %d, want 1", got)
	}
	if got := metadataInt(result.Metadata["in_progress"]); got != 1 {
		t.Fatalf("in_progress = %d, want 1", got)
	}
}

func TestTodoToolRejectsMultipleInProgressItems(t *testing.T) {
	tool := NewTodoTool()

	_, err := tool.Execute(context.Background(), Invocation{
		SessionID: "session-1",
		Arguments: []byte(`{"items":[{"content":"a","status":"in_progress"},{"content":"b","status":"in_progress"}]}`),
	})
	if err == nil {
		t.Fatal("expected Execute() to reject multiple in_progress items")
	}
}

func metadataInt(value any) int {
	switch actual := value.(type) {
	case int:
		return actual
	case float64:
		return int(actual)
	default:
		return 0
	}
}
