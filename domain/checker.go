// domain/checker.go
package domain

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/phi42/ad-enforcement-tool/rule"
)

// CheckResult holds the outcome of running all file checks for one rule.
type CheckResult struct {
	RuleName string
	Failures []string
	Warnings []string
}

// RunFileChecks executes all rule.CheckIR entries in spec against the filesystem,
// rooted at rootDir. Rules with rule.Severity warning produce non-fatal warnings;
// all other rules produce hard failures.
func RunFileChecks(spec *rule.SpecIR, rootDir string) ([]CheckResult, error) {
	// Build selector name → pattern map for path resolution.
	selMap := make(map[string]string, len(spec.Selectors))
	for _, s := range spec.Selectors {
		selMap[s.Name] = s.Pattern
	}

	var results []CheckResult

	for _, r := range spec.Rules {
		if !r.GetIsFileRule() {
			continue
		}

		result := CheckResult{RuleName: r.Name}
		isWarning := r.GetSeverity() == rule.Severity_SEVERITY_WARNING

		for _, c := range r.Checks {
			path := stripScheme(c.Path)
			pattern := stripScheme(c.Pattern)

			// Resolve selector reference to its pattern value.
			if resolved, ok := selMap[path]; ok {
				path = stripScheme(resolved)
			}

			var msgs []string
			var err error

			switch c.Kind {
			case rule.CheckKind_CHECK_FS_MUST_EXIST:
				msgs, err = checkMustExist(rootDir, path)
			case rule.CheckKind_CHECK_FS_MUST_NOT_EXIST:
				msgs, err = checkMustNotExist(rootDir, path)
			case rule.CheckKind_CHECK_FS_MUST_CONTAIN:
				msgs, err = checkMustContain(rootDir, path, pattern)
			case rule.CheckKind_CHECK_FS_MUST_NOT_CONTAIN:
				msgs, err = checkMustNotContain(rootDir, path, pattern)
			default:
				return nil, fmt.Errorf("rule %q: unsupported check kind %v", r.Name, c.Kind)
			}
			if err != nil {
				return nil, fmt.Errorf("rule %q: %w", r.Name, err)
			}

			if isWarning {
				result.Warnings = append(result.Warnings, msgs...)
			} else {
				result.Failures = append(result.Failures, msgs...)
			}
		}

		results = append(results, result)
	}

	return results, nil
}

// stripScheme strips the "glob:" or "regex:" prefix that the DSL may embed in paths/patterns.
func stripScheme(s string) string {
	if after, ok := strings.CutPrefix(s, "glob:"); ok {
		return after
	}
	if after, ok := strings.CutPrefix(s, "regex:"); ok {
		return after
	}
	return s
}

// glob returns all filesystem paths (files and directories) that match the
// glob pattern rooted at rootDir. The pattern is a slash-separated string
// supporting * (single segment wildcard), ? (single char), and ** (any depth).
func glob(rootDir, pattern string) ([]string, error) {
	pattern = filepath.ToSlash(pattern)
	rawParts := strings.Split(pattern, "/")

	// Filter out empty segments (from leading/trailing slashes or double slashes).
	var parts []string
	for _, p := range rawParts {
		if p != "" {
			parts = append(parts, p)
		}
	}

	matches, err := walkGlob(rootDir, parts)
	if err != nil {
		return nil, err
	}
	return matches, nil
}

func walkGlob(current string, parts []string) ([]string, error) {
	if len(parts) == 0 {
		// Reached end of pattern — the current path itself is a match.
		if _, err := os.Stat(current); err == nil {
			return []string{current}, nil
		}
		return nil, nil
	}

	part := parts[0]
	rest := parts[1:]

	// "." means "current directory" — skip the segment and stay where we are.
	if part == "." {
		return walkGlob(current, rest)
	}

	if part == "**" {
		// Match zero or more directory levels.
		var matches []string
		// Try matching rest from current directory (zero levels).
		sub, err := walkGlob(current, rest)
		if err != nil {
			return nil, err
		}
		matches = append(matches, sub...)

		// Recurse into all subdirectories.
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

	// Literal or wildcard segment — use filepath.Match for globbing.
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

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// checkMustExist returns a failure message if the glob matches nothing.
func checkMustExist(rootDir, pattern string) ([]string, error) {
	matches, err := glob(rootDir, pattern)
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return []string{fmt.Sprintf("expected path %q to exist, but no match found", pattern)}, nil
	}
	anyExists := false
	for _, m := range matches {
		if _, statErr := os.Stat(m); statErr == nil {
			anyExists = true
			break
		}
	}
	if !anyExists {
		return []string{fmt.Sprintf("expected path %q to exist, but no match found", pattern)}, nil
	}
	return nil, nil
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
func checkMustContain(rootDir, pathPattern, regexPattern string) ([]string, error) {
	matches, err := glob(rootDir, pathPattern)
	if err != nil {
		return nil, err
	}

	re, err := regexp.Compile(regexPattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex %q: %w", regexPattern, err)
	}

	// If no files match the path pattern, that is itself a failure.
	fileMatches := filterFiles(matches)
	if len(fileMatches) == 0 {
		return []string{fmt.Sprintf("no files matched %q for must-contain check", pathPattern)}, nil
	}

	var msgs []string
	for _, f := range fileMatches {
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
