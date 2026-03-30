package ui

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/muesli/termenv"
)

var (
	// lexerCache stores resolved lexers by extension so repeated renders stay cheap.
	lexerCache   = make(map[string]chroma.Lexer)
	lexerCacheMu sync.Mutex
)

var (
	// chromaStyleOnce caches one style choice for the current terminal session.
	chromaStyleOnce sync.Once
	chromaStyleName string
)

// ThemeMode represents the resolved terminal theme.
type ThemeMode int

const (
	// ThemeDark indicates a dark terminal background.
	ThemeDark ThemeMode = iota
	// ThemeLight indicates a light terminal background.
	ThemeLight
	// ThemeUnknown means detection failed; background colors are disabled.
	ThemeUnknown
)

var (
	// colorProfileFn is overrideable in tests so rendering can force a specific terminal profile.
	colorProfileFn = termenv.ColorProfile
	// hasDarkBackgroundFn is overrideable in tests so adaptive background colors resolve deterministically.
	hasDarkBackgroundFn = termenv.HasDarkBackground
	// resolvedTheme caches the theme decision for the session.
	resolvedTheme     ThemeMode
	resolvedThemeOnce sync.Once
)

// InitTheme resolves the terminal theme once at startup. Call before
// starting the Bubble Tea program. Reads DIFFPANE_THEME env var
// (light|dark) to override auto-detection.
func InitTheme() {
	resolvedThemeOnce.Do(resolveTheme)
}

// resolveTheme determines the theme from env var or termenv detection.
func resolveTheme() {
	switch os.Getenv("DIFFPANE_THEME") {
	case "light":
		resolvedTheme = ThemeLight
	case "dark":
		resolvedTheme = ThemeDark
	default:
		// Auto-detect: termenv queries the terminal via OSC 11.
		// If the query succeeds, the result is reliable.
		// If it fails (no response), termenv defaults to dark, which
		// may be wrong. We check BackgroundColor() to see if the
		// terminal actually responded.
		bg := termenv.BackgroundColor()
		if bg == nil || bg.Sequence(false) == "" {
			// Terminal did not respond to background query.
			resolvedTheme = ThemeUnknown
		} else if hasDarkBackgroundFn() {
			resolvedTheme = ThemeDark
		} else {
			resolvedTheme = ThemeLight
		}
	}
}

// GetTheme returns the resolved theme mode.
func GetTheme() ThemeMode {
	resolvedThemeOnce.Do(resolveTheme)
	return resolvedTheme
}

// setThemeForTest overrides the resolved theme during tests.
// Returns a restore function that must be deferred.
func setThemeForTest(mode ThemeMode) func() {
	prev := resolvedTheme
	resolvedTheme = mode
	// Mark as already resolved so sync.Once doesn't run resolveTheme.
	resolvedThemeOnce.Do(func() {})
	return func() { resolvedTheme = prev }
}

// getChromaStyleName returns the chroma style that best matches the terminal background.
func getChromaStyleName() string {
	chromaStyleOnce.Do(func() {
		switch GetTheme() {
		case ThemeLight:
			chromaStyleName = "github"
		default:
			// Dark and Unknown both use monokai (safe default for dark terminals).
			chromaStyleName = "monokai"
		}
	})

	return chromaStyleName
}

// getLexer resolves and caches a chroma lexer for the given filename.
// The full filename is the cache key because chroma supports special basenames
// like CMakeLists.txt that should not leak into unrelated files sharing an extension.
// Unknown types and plaintext both return nil so callers can fall back cleanly.
func getLexer(filename string) chroma.Lexer {
	if filename == "" {
		return nil
	}

	lexerCacheMu.Lock()
	defer lexerCacheMu.Unlock()

	if lexer, ok := lexerCache[filename]; ok {
		return lexer
	}

	lexer := lexers.Match(filename)
	if lexer == nil || lexer.Config().Name == "plaintext" {
		lexerCache[filename] = nil
		return nil
	}

	lexer = chroma.Coalesce(lexer)
	lexerCache[filename] = lexer
	return lexer
}

// HighlightCode applies chroma syntax colors to one code fragment.
// It returns the original content for empty input or unsupported file types.
func HighlightCode(code, filename string) string {
	if code == "" || filename == "" {
		return code
	}
	if colorProfileFn() == termenv.Ascii {
		return code
	}

	lexer := getLexer(filename)
	if lexer == nil {
		return code
	}

	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return code
	}

	formatterName := "terminal"
	switch colorProfileFn() {
	case termenv.TrueColor:
		formatterName = "terminal16m"
	case termenv.ANSI256:
		formatterName = "terminal256"
	}

	formatter := formatters.Get(formatterName)
	style := styles.Get(getChromaStyleName())

	var builder strings.Builder
	if err := formatter.Format(&builder, style, iterator); err != nil {
		return code
	}

	// chroma appends one trailing newline for terminal output, which would
	// otherwise create an extra wrapped row in the diff view.
	return strings.TrimSuffix(builder.String(), "\n")
}

// applyBg re-injects the background after each ANSI reset so syntax-highlighted
// output keeps one continuous line background. Verified against chroma v2.23.1
// and lipgloss v1.1.0, which both emit \033[0m resets.
func applyBg(text, hexColor string) string {
	if hexColor == "" {
		return text
	}

	bgSeq := hexToBgANSI(hexColor)
	result := strings.ReplaceAll(text, "\033[0m", "\033[0m"+bgSeq)
	return bgSeq + result + "\033[0m"
}

// hexToBgANSI converts #RRGGBB to a true-color background escape sequence.
func hexToBgANSI(hex string) string {
	var r, g, b int
	fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)
	return fmt.Sprintf("\033[48;2;%d;%d;%dm", r, g, b)
}
