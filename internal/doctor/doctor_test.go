package doctor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"duckduckgo-chat-cli/internal/config"
)

func dependenciesForRun(t *testing.T) (Dependencies, string) {
	t.Helper()
	root := t.TempDir()
	configPath := filepath.Join(root, "config.json")
	if err := os.WriteFile(configPath, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	exportDir := filepath.Join(root, "exports")
	if err := os.Mkdir(exportDir, 0o700); err != nil {
		t.Fatal(err)
	}
	deps := Dependencies{
		ReadConfig: func() (config.Config, string, error) {
			return config.Config{TOSAccepted: true, ExportDir: exportDir}, configPath, nil
		},
		BrowserPath: func() (string, error) { return "/usr/bin/chromium", nil },
		BrowserVersion: func(context.Context, string) (string, error) {
			return "Chromium 130.0.0.0", nil
		},
		NetworkProbe: func(context.Context) error { return nil },
	}
	return deps, exportDir
}

func findCheck(result Result, id string) (Check, bool) {
	for _, check := range result.Checks {
		if check.ID == id {
			return check, true
		}
	}
	return Check{}, false
}

func TestRunReportsMissingConfig(t *testing.T) {
	deps, exportDir := dependenciesForRun(t)
	missingPath := filepath.Join(t.TempDir(), "not-created", "config.json")
	deps.ReadConfig = func() (config.Config, string, error) {
		return config.Config{TOSAccepted: true, ExportDir: exportDir}, missingPath, nil
	}

	result := Run(context.Background(), Options{}, deps)
	check, ok := findCheck(result, "config_file")
	if !ok || check.Status != StatusWarning {
		t.Fatalf("config_file check = %+v, present %v; want warning", check, ok)
	}
	if result.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0 for an uninitialized config", result.ExitCode)
	}
}

func TestRunReportsValidConfig(t *testing.T) {
	deps, _ := dependenciesForRun(t)
	result := Run(context.Background(), Options{}, deps)
	for _, id := range []string{"platform", "config_file", "terms", "browser", "browser_version", "export_directory", "network"} {
		check, ok := findCheck(result, id)
		if !ok {
			t.Errorf("missing stable check ID %q", id)
			continue
		}
		if id != "network" && check.Status != StatusPass {
			t.Errorf("%s status = %q, want pass (%s)", id, check.Status, check.Message)
		}
	}
	if result.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0", result.ExitCode)
	}
}

func TestRunReportsMissingBrowserWithoutNetworkProbe(t *testing.T) {
	deps, _ := dependenciesForRun(t)
	networkCalled := false
	deps.BrowserPath = func() (string, error) { return "", errors.New("not found") }
	deps.NetworkProbe = func(context.Context) error {
		networkCalled = true
		return nil
	}

	result := Run(context.Background(), Options{}, deps)
	check, ok := findCheck(result, "browser")
	if !ok || check.Status != StatusFail {
		t.Fatalf("browser check = %+v, present %v; want fail", check, ok)
	}
	if result.ExitCode != 1 {
		t.Fatalf("ExitCode = %d, want 1", result.ExitCode)
	}
	if networkCalled {
		t.Fatal("local-only run called the network probe")
	}
}

func TestRunWarnsOnOldBrowserVersion(t *testing.T) {
	deps, _ := dependenciesForRun(t)
	deps.BrowserVersion = func(context.Context, string) (string, error) {
		return "Chromium 119.0.0.0", nil
	}

	result := Run(context.Background(), Options{}, deps)
	check, ok := findCheck(result, "browser_version")
	if !ok || check.Status != StatusWarning {
		t.Fatalf("browser_version check = %+v, present %v; want warning", check, ok)
	}
	if result.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0 for a recommended-version warning", result.ExitCode)
	}
}

func TestRunReportsMissingExportDirectory(t *testing.T) {
	deps, _ := dependenciesForRun(t)
	missing := filepath.Join(t.TempDir(), "exports")
	deps.ReadConfig = func() (config.Config, string, error) {
		return config.Config{TOSAccepted: true, ExportDir: missing}, "", nil
	}

	result := Run(context.Background(), Options{}, deps)
	check, ok := findCheck(result, "export_directory")
	if !ok || check.Status != StatusWarning {
		t.Fatalf("export_directory check = %+v, present %v; want warning", check, ok)
	}
}

func TestRunSkipsNetworkProbeByDefault(t *testing.T) {
	deps, _ := dependenciesForRun(t)
	networkCalled := false
	deps.NetworkProbe = func(context.Context) error {
		networkCalled = true
		return nil
	}

	result := Run(context.Background(), Options{}, deps)
	check, ok := findCheck(result, "network")
	if !ok || check.Status != StatusSkipped {
		t.Fatalf("network check = %+v, present %v; want skipped", check, ok)
	}
	if networkCalled {
		t.Fatal("network probe ran without opt-in")
	}
}

func TestRunReportsNetworkProbeFailure(t *testing.T) {
	deps, _ := dependenciesForRun(t)
	deps.NetworkProbe = func(context.Context) error {
		return errors.New("captured VQD=private-value")
	}

	result := Run(context.Background(), Options{Network: true}, deps)
	check, ok := findCheck(result, "network")
	if !ok || check.Status != StatusWarning {
		t.Fatalf("network check = %+v, present %v; want warning", check, ok)
	}
	if result.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0 for a transient network failure", result.ExitCode)
	}
	if check.Message == "captured VQD=private-value" {
		t.Fatal("network check exposed the raw probe error")
	}
}

func TestRunDoesNotAcceptTermsOrPrompt(t *testing.T) {
	deps, _ := dependenciesForRun(t)
	accepted := false
	deps.ReadConfig = func() (config.Config, string, error) {
		return config.Config{TOSAccepted: accepted, ExportDir: t.TempDir()}, "", nil
	}

	result := Run(context.Background(), Options{}, deps)
	check, ok := findCheck(result, "terms")
	if !ok || check.Status != StatusFail {
		t.Fatalf("terms check = %+v, present %v; want fail", check, ok)
	}
	if accepted {
		t.Fatal("diagnostic changed terms acceptance")
	}
	if result.ExitCode != 1 {
		t.Fatalf("ExitCode = %d, want 1", result.ExitCode)
	}
}

func TestRunSkipsUnavailableBrowserVersionCheck(t *testing.T) {
	deps, _ := dependenciesForRun(t)
	deps.BrowserVersion = nil

	result := Run(context.Background(), Options{}, deps)
	check, ok := findCheck(result, "browser_version")
	if !ok || check.Status != StatusSkipped {
		t.Fatalf("browser_version check = %+v, present %v; want skipped", check, ok)
	}
}

func TestRunNetworkProbeIsOptInAndBounded(t *testing.T) {
	deps, _ := dependenciesForRun(t)
	called := false
	deps.NetworkProbe = func(ctx context.Context) error {
		called = true
		if _, ok := ctx.Deadline(); !ok {
			t.Error("opt-in network probe has no deadline")
		}
		return nil
	}

	Run(context.Background(), Options{Network: true}, deps)
	if !called {
		t.Fatal("opt-in network probe was not called")
	}
}
