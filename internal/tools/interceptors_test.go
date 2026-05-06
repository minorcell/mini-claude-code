package tools

import "testing"

func TestSafeJoinRejectsEscape(t *testing.T) {
	root := t.TempDir()

	if _, err := SafeJoin(root, "../outside"); err == nil {
		t.Fatalf("expected SafeJoin to reject path escape")
	}
}
