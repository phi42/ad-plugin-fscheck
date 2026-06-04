package fscheck

import (
	"fmt"
	"strings"

	"github.com/phi42/ad-enforcement-tool/rule"
)

// RunFileChecks executes all file rules in spec against the filesystem rooted
// at rootDir. Rules with severity warning produce non-fatal warnings; all
// other rules produce hard failures. Non-file rules are ignored by this
// function (the caller is expected to warn about them separately).
func RunFileChecks(spec *rule.Spec, rootDir string) ([]CheckResult, error) {
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

// stripScheme strips the "glob:" or "regex:" prefix that the DSL may embed in
// path or pattern values.
func stripScheme(s string) string {
	if after, ok := strings.CutPrefix(s, "glob:"); ok {
		return after
	}
	if after, ok := strings.CutPrefix(s, "regex:"); ok {
		return after
	}
	return s
}
