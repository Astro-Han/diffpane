package ui

import (
	"strings"
	"testing"
)

// TestWrapLineShort verifies lines shorter than the viewport stay untouched.
func TestWrapLineShort(t *testing.T) {
	result := wrapLine("+hello", 40)
	if result != "+hello" {
		t.Fatalf("wrapLine returned %q, want +hello", result)
	}
}

// TestWrapLineLong verifies continuation lines use indentation instead of a diff prefix.
func TestWrapLineLong(t *testing.T) {
	line := "+" + strings.Repeat("a", 50)
	result := wrapLine(line, 30)
	lines := strings.Split(result, "\n")
	if len(lines) < 2 {
		t.Fatal("expected wrapped output")
	}
	if !strings.HasPrefix(lines[1], "  ") {
		t.Fatalf("continuation = %q, want 2-space indent", lines[1])
	}
}

// TestWrapLineCJK verifies wrapping never cuts through a rune.
func TestWrapLineCJK(t *testing.T) {
	line := "+你好世界测试"
	result := wrapLine(line, 8)
	for _, r := range result {
		if r == '\uFFFD' {
			t.Fatal("wrapLine cut a rune and produced replacement character")
		}
	}
}
