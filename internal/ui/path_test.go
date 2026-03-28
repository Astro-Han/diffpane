package ui

import "testing"

// TestShortestUniquePaths verifies path display uses the shortest unique suffix.
func TestShortestUniquePaths(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "unique basenames",
			input: []string{"src/auth.ts", "src/utils.ts", "package.json"},
			want:  []string{"auth.ts", "utils.ts", "package.json"},
		},
		{
			name:  "duplicate basenames",
			input: []string{"src/auth.ts", "pkg/auth.ts", "lib/utils.ts"},
			want:  []string{"src/auth.ts", "pkg/auth.ts", "utils.ts"},
		},
		{
			name:  "single file",
			input: []string{"src/deep/file.go"},
			want:  []string{"file.go"},
		},
		{
			name:  "root files",
			input: []string{"README.md", "go.mod"},
			want:  []string{"README.md", "go.mod"},
		},
		{
			name:  "deeper dups",
			input: []string{"a/b/auth.ts", "a/c/auth.ts", "x/c/auth.ts"},
			want:  []string{"b/auth.ts", "a/c/auth.ts", "x/c/auth.ts"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ShortestUniquePaths(tc.input)
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("[%d] got %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}
