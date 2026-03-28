package ui

import (
	"path/filepath"
	"strings"
)

// ShortestUniquePaths returns the shortest unique suffix for each path.
func ShortestUniquePaths(paths []string) []string {
	result := make([]string, len(paths))
	for i, path := range paths {
		parts := strings.Split(filepath.ToSlash(path), "/")
		for n := 1; n <= len(parts); n++ {
			suffix := strings.Join(parts[len(parts)-n:], "/")
			unique := true
			for j, other := range paths {
				if i == j {
					continue
				}
				otherParts := strings.Split(filepath.ToSlash(other), "/")
				otherSuffix := strings.Join(otherParts[max(0, len(otherParts)-n):], "/")
				if suffix == otherSuffix {
					unique = false
					break
				}
			}
			if unique {
				result[i] = suffix
				break
			}
		}
		if result[i] == "" {
			result[i] = path
		}
	}

	return result
}
