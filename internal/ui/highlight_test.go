package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

// TestHighlightCodeGo verifies that Go source gets ANSI-highlighted output.
func TestHighlightCodeGo(t *testing.T) {
	prev := colorProfileFn
	colorProfileFn = func() termenv.Profile { return termenv.ANSI256 }
	defer func() { colorProfileFn = prev }()

	result := HighlightCode("func main() {", "main.go")

	if !strings.Contains(result, "\033[") {
		t.Fatalf("expected ANSI codes in output, got %q", result)
	}

	stripped := ansi.Strip(result)
	if stripped != "func main() {" {
		t.Fatalf("stripped = %q, want %q", stripped, "func main() {")
	}
}

// TestHighlightCodeUnknown verifies unknown file types return plain text.
func TestHighlightCodeUnknown(t *testing.T) {
	input := "some random content"
	result := HighlightCode(input, "data.randomext123")
	if result != input {
		t.Fatalf("unknown file type should return plain text, got %q", result)
	}
}

// TestHighlightCodePlaintext verifies that .txt files return plain text.
func TestHighlightCodePlaintext(t *testing.T) {
	input := "just plain text"
	result := HighlightCode(input, "notes.txt")
	if result != input {
		t.Fatalf("plaintext file should return plain text, got %q", result)
	}
}

// TestHighlightCodeEmpty verifies empty input returns empty output.
func TestHighlightCodeEmpty(t *testing.T) {
	if result := HighlightCode("", "main.go"); result != "" {
		t.Fatalf("expected empty, got %q", result)
	}
}

// TestHighlightCodeNoFilename verifies missing filename returns plain text.
func TestHighlightCodeNoFilename(t *testing.T) {
	input := "func main() {"
	if result := HighlightCode(input, ""); result != input {
		t.Fatalf("expected plain text for empty filename, got %q", result)
	}
}

// TestHighlightCodeMultipleLanguages verifies different extensions get different lexers.
func TestHighlightCodeMultipleLanguages(t *testing.T) {
	prev := colorProfileFn
	colorProfileFn = func() termenv.Profile { return termenv.ANSI256 }
	defer func() { colorProfileFn = prev }()

	goResult := HighlightCode("func main() {", "main.go")
	pyResult := HighlightCode("def main():", "main.py")

	if !strings.Contains(goResult, "\033[") {
		t.Fatal("Go result should contain ANSI codes")
	}
	if !strings.Contains(pyResult, "\033[") {
		t.Fatal("Python result should contain ANSI codes")
	}
	if goResult == pyResult {
		t.Fatal("Go and Python should produce different highlighting")
	}
}

// TestHighlightCodeCacheDoesNotLeakSpecialFilename verifies a lexer chosen for
// a special filename does not pollute unrelated files with the same extension.
func TestHighlightCodeCacheDoesNotLeakSpecialFilename(t *testing.T) {
	prev := colorProfileFn
	colorProfileFn = func() termenv.Profile { return termenv.ANSI256 }
	defer func() { colorProfileFn = prev }()

	special := HighlightCode("project(example)", "CMakeLists.txt")
	if !strings.Contains(special, "\033[") {
		t.Fatalf("special filename should be highlighted, got %q", special)
	}

	plain := HighlightCode("just plain text", "notes.txt")
	if plain != "just plain text" {
		t.Fatalf("plaintext file should stay plain after special filename cache, got %q", plain)
	}
}

// TestApplyBgReinjectsBackgroundAfterReset verifies whole-line background
// survives ANSI resets emitted by other formatters.
func TestApplyBgReinjectsBackgroundAfterReset(t *testing.T) {
	input := "\033[2m  42\033[0m func \033[38;2;255;0;0mmain\033[0m"
	result := applyBg(input, "#002200")

	if !strings.Contains(result, "\033[48;2;0;34;0m") {
		t.Fatalf("applyBg should inject true-color background, got %q", result)
	}
	if ansi.Strip(result) != "  42 func main" {
		t.Fatalf("applyBg changed visible content to %q", ansi.Strip(result))
	}
}

// TestHighlightCodeAsciiProfileReturnsPlainText verifies Ascii mode disables
// syntax ANSI completely.
func TestHighlightCodeAsciiProfileReturnsPlainText(t *testing.T) {
	prev := colorProfileFn
	colorProfileFn = func() termenv.Profile { return termenv.Ascii }
	defer func() { colorProfileFn = prev }()

	result := HighlightCode("func main() {", "main.go")
	if result != "func main() {" {
		t.Fatalf("Ascii profile should disable ANSI highlighting, got %q", result)
	}
}
