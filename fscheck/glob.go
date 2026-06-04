package fscheck

import (
	"os"
	"path/filepath"
	"strings"
)

// glob returns all filesystem paths (files and directories) that match the
// glob pattern rooted at rootDir. The pattern is a slash-separated string
// supporting "*" (single segment wildcard), "?" (single character), and "**"
// (any depth).
func glob(rootDir, pattern string) ([]string, error) {
	pattern = filepath.ToSlash(pattern)
	rawParts := strings.Split(pattern, "/")

	var parts []string
	for _, p := range rawParts {
		if p != "" {
			parts = append(parts, p)
		}
	}

	return walkGlob(rootDir, parts)
}

// walkGlob recursively expands a sequence of pattern segments starting from
// the current directory. The "**" segment matches zero or more directory
// levels.
func walkGlob(current string, parts []string) ([]string, error) {
	if len(parts) == 0 {
		if _, err := os.Stat(current); err == nil {
			return []string{current}, nil
		}
		return nil, nil
	}

	part := parts[0]
	rest := parts[1:]

	if part == "." {
		return walkGlob(current, rest)
	}

	if part == "**" {
		var matches []string

		// Zero levels: try to match rest from the current directory.
		sub, err := walkGlob(current, rest)
		if err != nil {
			return nil, err
		}
		matches = append(matches, sub...)

		// One or more levels: recurse into each subdirectory.
		entries, err := os.ReadDir(current)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, nil
			}
			return nil, err
		}
		for _, e := range entries {
			if e.IsDir() {
				sub, err := walkGlob(filepath.Join(current, e.Name()), parts)
				if err != nil {
					return nil, err
				}
				matches = append(matches, sub...)
			}
		}
		return matches, nil
	}

	entries, err := os.ReadDir(current)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var matches []string
	for _, e := range entries {
		matched, err := filepath.Match(part, e.Name())
		if err != nil {
			return nil, err
		}
		if matched {
			sub, err := walkGlob(filepath.Join(current, e.Name()), rest)
			if err != nil {
				return nil, err
			}
			matches = append(matches, sub...)
		}
	}
	return matches, nil
}

// filterFiles returns only the paths that are regular files.
func filterFiles(paths []string) []string {
	var out []string
	for _, p := range paths {
		info, err := os.Stat(p)
		if err == nil && info.Mode().IsRegular() {
			out = append(out, p)
		}
	}
	return out
}
