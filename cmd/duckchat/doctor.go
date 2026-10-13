package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"duckduckgo-chat-cli/internal/chat"
	"duckduckgo-chat-cli/internal/config"
	"duckduckgo-chat-cli/internal/doctor"
)

func runDoctor(args []string, stdout, stderr io.Writer) int {
	deps := doctor.Dependencies{
		ReadConfig:  config.ReadOnlyDiagnosticsConfig,
		BrowserPath: chat.BrowserExecutable,
		BrowserVersion: func(ctx context.Context, path string) (string, error) {
			output, err := exec.CommandContext(ctx, path, "--version").CombinedOutput()
			if err != nil {
				return "", errors.New("browser version command failed")
			}
			return strings.TrimSpace(string(output)), nil
		},
		NetworkProbe: func(ctx context.Context) error {
			_, err := chat.CaptureDynamicHeaders(ctx)
			return err
		},
	}
	return runDoctorWithDependencies(args, stdout, stderr, deps)
}

func runDoctorWithDependencies(args []string, stdout, stderr io.Writer, deps doctor.Dependencies) int {
	flags := flag.NewFlagSet("duckchat doctor", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: duckchat doctor [--network] [--json]")
	}
	var options doctor.Options
	var jsonOutput, help bool
	flags.BoolVar(&options.Network, "network", false, "Check Duck.ai connectivity without sending a prompt")
	flags.BoolVar(&jsonOutput, "json", false, "Output a stable JSON report")
	flags.BoolVar(&help, "help", false, "Show this help message")
	flags.BoolVar(&help, "h", false, "Show this help message")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, fmt.Sprintf("duckchat doctor: unexpected argument %q", flags.Arg(0)))
		flags.Usage()
		return 2
	}
	if help {
		printDoctorUsage(stdout)
		return 0
	}
	if options.Network {
		defer func() { _ = chat.ShutdownBrowser() }()
	}

	result := doctor.Run(context.Background(), options, deps)
	if jsonOutput {
		if err := json.NewEncoder(stdout).Encode(result); err != nil {
			fmt.Fprintln(stderr, "duckchat doctor: could not write JSON report")
			return 1
		}
		return result.ExitCode
	}

	for _, check := range result.Checks {
		fmt.Fprintln(stdout, fmt.Sprintf("%s [%s] %s", check.ID, check.Status, check.Message))
	}
	if result.ExitCode == 0 {
		fmt.Fprintln(stdout, "Result: no required checks failed.")
	} else {
		fmt.Fprintln(stdout, "Result: one or more required checks failed.")
	}
	return result.ExitCode
}

func printDoctorUsage(out io.Writer) {
	fmt.Fprintln(out, "Usage: duckchat doctor [--network] [--json]")
	fmt.Fprintln(out, "Check local setup and Duck.ai connectivity without sending a prompt.")
	fmt.Fprintln(out, "  --network  Run a bounded Duck.ai browser-capture check")
	fmt.Fprintln(out, "  --json     Output stable machine-readable JSON")
}
