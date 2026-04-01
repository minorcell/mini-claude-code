package tools

import "testing"

// TestSafeJoinRejectsEscape 验证 SafeJoin 会阻止路径逃逸出工作区。
func TestSafeJoinRejectsEscape(t *testing.T) {
	root := t.TempDir()

	if _, err := SafeJoin(root, "../outside"); err == nil {
		t.Fatalf("expected SafeJoin to reject path escape")
	}
}
