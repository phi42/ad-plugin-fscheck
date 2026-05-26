package cmd

import (
	"fmt"
	"io"
	"log/slog"
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
		setupPluginLogger()
		if err := run(); err != nil {
			slog.Error("plugin failed", "error", err)
			os.Exit(1)
		}
	},
}

func setupPluginLogger() {
	level := slog.LevelInfo
	skipWarn := false
	switch os.Getenv("ADE_LOG_LEVEL") {
	case "debug":
		level = slog.LevelDebug
	case "quiet":
		level = slog.LevelError
	case "no-warnings":
		skipWarn = true
	}
	slog.SetDefault(slog.New(newCLIHandler(os.Stderr, level, skipWarn)))
}

func Execute() {
	if len(os.Args) == 2 && os.Args[1] == "--info" {
		fmt.Println(`{"modes":["verify"]}`)
		os.Exit(0)
	}
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func run() error {
	// Read protobuf SpecIR from stdin.
	payload, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}

	var spec rule.SpecIR
	if err := proto.Unmarshal(payload, &spec); err != nil {
		return fmt.Errorf("unmarshal SpecIR protobuf: %w", err)
	}

	// Warn about any rules this plugin does not handle. filecheck only
	// executes file rules; code and custom rules are skipped.
	for _, r := range spec.Rules {
		if !r.GetIsFileRule() {
			slog.Warn(fmt.Sprintf("rule %q skipped (filecheck handles file rules only)", r.GetName()))
		}
	}

	// Use the root dir provided by ade enforce verify (via OutputDir field), falling
	// back to the current working directory.
	rootDir := spec.GetOutputDir()
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
			slog.Warn(fmt.Sprintf("failed [%s] with warning: %s", res.RuleName, w))
		}
		for _, f := range res.Failures {
			slog.Error(fmt.Sprintf("failed [%s] with error: %s", res.RuleName, f))
			hasFailures = true
		}
		if len(res.Failures) == 0 && len(res.Warnings) == 0 {
			slog.Info(fmt.Sprintf("passed [%s]", res.RuleName))
		}
	}

	if hasFailures {
		return fmt.Errorf("one or more file checks failed")
	}
	return nil
}
