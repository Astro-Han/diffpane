package ui

import (
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

// getChromaStyleName returns the chroma style that best matches the terminal background.
func getChromaStyleName() string {
	chromaStyleOnce.Do(func() {
		if termenv.HasDarkBackground() {
			chromaStyleName = "monokai"
			return
		}
		chromaStyleName = "github"
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

	lexer := getLexer(filename)
	if lexer == nil {
		return code
	}

	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return code
	}

	formatter := formatters.Get("terminal256")
	style := styles.Get(getChromaStyleName())

	var builder strings.Builder
	if err := formatter.Format(&builder, style, iterator); err != nil {
		return code
	}

	// chroma appends one trailing newline for terminal output, which would
	// otherwise create an extra wrapped row in the diff view.
	return strings.TrimSuffix(builder.String(), "\n")
}
