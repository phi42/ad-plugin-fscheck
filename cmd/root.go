package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/phi42/ad-enforcement-tool/rule"
	"github.com/phi42/ad-plugin-FScheck/domain"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

var rootCmd = &cobra.Command{
	Use:   "fscheck",
	Short: "Filesystem rule executor for ADR-based DSL rules (file checks only)",
	Run: func(cmd *cobra.Command, args []string) {
		if err := run(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	},
}

func Execute() {
	if len(os.Args) == 2 && os.Args[1] == "--info" {
		fmt.Println(`{"modes":["verify"],"config_prefix":"fscheck"}`)
		os.Exit(0)
	}
	if fi, err := os.Stdin.Stat(); err == nil && (fi.Mode()&os.ModeCharDevice) != 0 {
		_ = rootCmd.Help()
		os.Exit(0)
	}
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func run() error {
	// Read protobuf Spec from stdin.
	payload, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}

	var spec rule.Spec
	if err := proto.Unmarshal(payload, &spec); err != nil {
		return fmt.Errorf("unmarshal Spec protobuf: %w", err)
	}

	// Warn about any rules this plugin does not handle. filecheck only
	// executes file rules; code and custom rules are skipped.
	for _, r := range spec.Rules {
		if !r.GetIsFileRule() {
			fmt.Fprintf(os.Stderr, "warn: rule %q skipped (filecheck handles file rules only)\n", r.GetName())
		}
	}

	// Use the root dir from plugin_config, falling back to the current working directory.
	rootDir := spec.GetPluginConfig()["root-dir"]
	if rootDir == "" {
		rootDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("getting working directory: %w", err)
		}
	}

	results, err := domain.RunFileChecks(&spec, rootDir)
	if err != nil {
		return fmt.Errorf("running file checks: %w", err)
	}

	hasFailures := false
	for _, res := range results {
		for _, w := range res.Warnings {
			fmt.Fprintf(os.Stderr, "warn: failed [%s] with warning: %s\n", res.RuleName, w)
		}
		for _, f := range res.Failures {
			fmt.Fprintf(os.Stderr, "error: failed [%s] with error: %s\n", res.RuleName, f)
			hasFailures = true
		}
		if len(res.Failures) == 0 && len(res.Warnings) == 0 {
			fmt.Fprintf(os.Stderr, "passed [%s]\n", res.RuleName)
		}
	}

	if hasFailures {
		return fmt.Errorf("one or more file checks failed")
	}
	return nil
}
