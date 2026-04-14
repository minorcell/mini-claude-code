package tools

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestReadToolLimitStopsScanningAtLimit(t *testing.T) {
	root := t.TempDir()
	path := writeTestFile(t, root, "sample.txt", "a\n"+strings.Repeat("b", 20)+"\n"+strings.Repeat("c", 20)+"\n")

	tool := NewReadTool(root)
	result, err := tool.readLines(path, 1, 1, 10)
	if err != nil {
		t.Fatalf("readLines() error = %v", err)
	}

	if got := metadataBool(result.Metadata["truncated"]); got {
		t.Fatalf("truncated = true, want false")
	}
	if got := metadataInt(result.Metadata["read_lines"]); got != 1 {
		t.Fatalf("read_lines = %d, want 1", got)
	}
}

func TestGrepToolContextIncludesAdjacentLines(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "file.txt", "line-1\nmatch\nline-3\n")

	tool := NewGrepTool(root)
	result, err := tool.Execute(context.Background(), Invocation{
		Arguments: []byte(`{"path":".","pattern":"match","context":1}`),
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := result.Output
	if !strings.Contains(output, "file.txt:1: line-1") {
		t.Fatalf("expected output to include previous context line, got %q", output)
	}
	if !strings.Contains(output, "file.txt:2: match") {
		t.Fatalf("expected output to include matched line, got %q", output)
	}
	if !strings.Contains(output, "file.txt:3: line-3") {
		t.Fatalf("expected output to include next context line, got %q", output)
	}
}

func TestGrepToolReturnsScanError(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "long.txt", strings.Repeat("x", 70*1024)+"\n")

	tool := NewGrepTool(root)
	_, err := tool.Execute(context.Background(), Invocation{
		Arguments: []byte(`{"path":".","pattern":"x"}`),
	})
	if err == nil {
		t.Fatal("expected Execute() to return scanner error for very long line")
	}
	if !strings.Contains(err.Error(), "token too long") {
		t.Fatalf("unexpected error %q", err)
	}
}

func writeTestFile(t *testing.T, root, relPath, content string) string {
	t.Helper()

	absPath, err := SafeJoin(root, relPath)
	if err != nil {
		t.Fatalf("SafeJoin() error = %v", err)
	}
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write file %q: %v", absPath, err)
	}
	return absPath
}

func metadataBool(value any) bool {
	switch actual := value.(type) {
	case bool:
		return actual
	default:
		return false
	}
}
