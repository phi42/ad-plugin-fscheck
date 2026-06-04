package fscheck

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// checkMustExist returns a failure message if the glob matches no existing path.
func checkMustExist(rootDir, pattern string) ([]string, error) {
	matches, err := glob(rootDir, pattern)
	if err != nil {
		return nil, err
	}
	for _, m := range matches {
		if _, statErr := os.Stat(m); statErr == nil {
			return nil, nil
		}
	}
	return []string{fmt.Sprintf("expected path %q to exist, but no match found", pattern)}, nil
}

// checkMustNotExist returns a failure message for each match that actually exists.
func checkMustNotExist(rootDir, pattern string) ([]string, error) {
	matches, err := glob(rootDir, pattern)
	if err != nil {
		return nil, err
	}
	var msgs []string
	for _, m := range matches {
		if _, statErr := os.Stat(m); statErr == nil {
			msgs = append(msgs, fmt.Sprintf("expected path %q to NOT exist, but it does", m))
		}
	}
	return msgs, nil
}

// checkMustContain verifies each matching file contains the given regex pattern.
// If the path pattern matches no files, that is itself reported as a failure.
func checkMustContain(rootDir, pathPattern, regexPattern string) ([]string, error) {
	matches, err := glob(rootDir, pathPattern)
	if err != nil {
		return nil, err
	}

	re, err := regexp.Compile(regexPattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex %q: %w", regexPattern, err)
	}

	files := filterFiles(matches)
	if len(files) == 0 {
		return []string{fmt.Sprintf("no files matched %q for must-contain check", pathPattern)}, nil
	}

	var msgs []string
	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("reading %q: %w", f, err)
		}
		if !re.Match(content) {
			rel, _ := filepath.Rel(rootDir, f)
			msgs = append(msgs, fmt.Sprintf("file %q must contain pattern %q but does not", rel, regexPattern))
		}
	}
	return msgs, nil
}

// checkMustNotContain verifies each matching file does not contain the given regex pattern.
func checkMustNotContain(rootDir, pathPattern, regexPattern string) ([]string, error) {
	matches, err := glob(rootDir, pathPattern)
	if err != nil {
		return nil, err
	}

	re, err := regexp.Compile(regexPattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex %q: %w", regexPattern, err)
	}

	var msgs []string
	for _, f := range filterFiles(matches) {
		content, err := os.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("reading %q: %w", f, err)
		}
		if re.Match(content) {
			rel, _ := filepath.Rel(rootDir, f)
			msgs = append(msgs, fmt.Sprintf("file %q must NOT contain pattern %q but it does", rel, regexPattern))
		}
	}
	return msgs, nil
}
