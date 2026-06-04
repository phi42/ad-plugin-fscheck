package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/phi42/ad-enforcement-tool/rule"
	"github.com/phi42/ad-plugin-FScheck/fscheck"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

type pluginInfo struct {
	Modes        []string `json:"modes"`
	ConfigPrefix string   `json:"config_prefix"`
}

var info = pluginInfo{
	Modes:        []string{"verify"},
	ConfigPrefix: "fscheck",
}

var rootCmd = &cobra.Command{
	Use:   "Install this plugin using `ade plugin install` and then run it via `ade verify`",
	Short: "Filesystem rule executor for ADR rules (file rules only)",
	Run: func(cmd *cobra.Command, args []string) {
		if err := run(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	},
}

func Execute() {
	if len(os.Args) == 2 && os.Args[1] == "--info" {
		out, err := json.Marshal(info)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: marshaling plugin info: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(out))
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
	// read protobuf Spec from stdin
	payload, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}

	var spec rule.Spec
	if err := proto.Unmarshal(payload, &spec); err != nil {
		return fmt.Errorf("unmarshal Spec protobuf: %w", err)
	}

	var skipped int
	for _, r := range spec.Rules {
		if !r.GetIsFileRule() {
			skipped++
		}
	}
	if skipped > 0 {
		fmt.Fprintf(os.Stderr, "warn: %d rule(s) skipped (plugin can only handle file rules)\n", skipped)
	}

	rootDir := spec.GetPluginConfig()["root-dir"]
	if rootDir == "" {
		rootDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("getting working directory: %w", err)
		}
	}

	results, err := fscheck.RunFileChecks(&spec, rootDir)
	if err != nil {
		return fmt.Errorf("running file checks: %w", err)
	}

	hasFailures := false
	for _, res := range results {
		for _, w := range res.Warnings {
			fmt.Fprintf(os.Stderr, "warn: failed [%s]: %s\n", res.RuleName, w)
		}
		for _, f := range res.Failures {
			fmt.Fprintf(os.Stderr, "error: failed [%s]: %s\n", res.RuleName, f)
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
