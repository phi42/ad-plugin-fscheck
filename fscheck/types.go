package fscheck

// CheckResult holds the outcome of running all file checks for one rule.
type CheckResult struct {
	RuleName string
	Failures []string
	Warnings []string
}
