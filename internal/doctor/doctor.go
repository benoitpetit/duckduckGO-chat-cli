package doctor

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"

	"duckduckgo-chat-cli/internal/config"
)

type Status string

const (
	StatusPass    Status = "pass"
	StatusWarning Status = "warning"
	StatusFail    Status = "fail"
	StatusSkipped Status = "skipped"

	minimumRecommendedBrowserMajor = 120
	browserVersionTimeout          = 3 * time.Second
	networkProbeTimeout            = 10 * time.Second
)

type Check struct {
	ID      string `json:"id"`
	Status  Status `json:"status"`
	Message string `json:"message"`
}

type Options struct {
	Network bool
}

type Result struct {
	Checks   []Check `json:"checks"`
	ExitCode int     `json:"exit_code"`
}

type Dependencies struct {
	ReadConfig     func() (config.Config, string, error)
	BrowserPath    func() (string, error)
	BrowserVersion func(context.Context, string) (string, error)
	NetworkProbe   func(context.Context) error
}

func Run(ctx context.Context, options Options, deps Dependencies) Result {
	if ctx == nil {
		ctx = context.Background()
	}
	result := Result{Checks: make([]Check, 0, 7)}
	add := func(id string, status Status, message string) {
		result.Checks = append(result.Checks, Check{ID: id, Status: status, Message: message})
		if status == StatusFail {
			result.ExitCode = 1
		}
	}

	platformStatus := StatusPass
	if runtime.GOOS != "windows" && runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		platformStatus = StatusWarning
	}
	add("platform", platformStatus, fmt.Sprintf("Detected %s/%s.", runtime.GOOS, runtime.GOARCH))

	var cfg config.Config
	configLoaded := false
	if deps.ReadConfig == nil {
		add("config_file", StatusFail, "The local configuration could not be inspected.")
	} else {
		loaded, path, err := deps.ReadConfig()
		if err != nil {
			add("config_file", StatusFail, "The local configuration could not be read.")
		} else {
			cfg = loaded
			configLoaded = true
			switch {
			case path == "":
				add("config_file", StatusSkipped, "The configuration path is unavailable.")
			default:
				_, statErr := os.Stat(path)
				switch {
				case os.IsNotExist(statErr):
					add("config_file", StatusWarning, "No saved configuration was found; default settings are active.")
				case statErr != nil:
					add("config_file", StatusFail, "The saved configuration could not be inspected.")
				default:
					add("config_file", StatusPass, "Saved configuration is readable.")
				}
			}
		}
	}

	if !configLoaded {
		add("terms", StatusSkipped, "Terms acceptance could not be checked because configuration is unavailable.")
	} else if cfg.TOSAccepted {
		add("terms", StatusPass, "Terms of service have been accepted.")
	} else {
		add("terms", StatusFail, "Terms of service have not been accepted; accept them in an interactive chat before continuing.")
	}

	browserPath := ""
	browserAvailable := false
	if deps.BrowserPath == nil {
		add("browser", StatusSkipped, "Browser discovery is unavailable on this platform.")
	} else if path, err := deps.BrowserPath(); err != nil || path == "" {
		add("browser", StatusFail, "Google Chrome or Chromium was not found; install one to use Duck.ai.")
	} else {
		browserPath = path
		browserAvailable = true
		add("browser", StatusPass, "Google Chrome or Chromium is available.")
	}

	if !browserAvailable {
		add("browser_version", StatusSkipped, "Browser version check was skipped because no browser is available.")
	} else if deps.BrowserVersion == nil {
		add("browser_version", StatusSkipped, "Browser version discovery is unavailable on this platform.")
	} else {
		versionCtx, cancel := context.WithTimeout(ctx, browserVersionTimeout)
		version, err := deps.BrowserVersion(versionCtx, browserPath)
		cancel()
		major := browserMajor(version)
		switch {
		case err != nil:
			add("browser_version", StatusWarning, "The browser version could not be checked; update it if Duck.ai fails to open.")
		case major == 0:
			add("browser_version", StatusWarning, "The browser version could not be identified; update Chrome or Chromium if Duck.ai fails to open.")
		case major < minimumRecommendedBrowserMajor:
			add("browser_version", StatusWarning, fmt.Sprintf("Browser major version %d is older than the recommended minimum %d; update Chrome or Chromium.", major, minimumRecommendedBrowserMajor))
		default:
			add("browser_version", StatusPass, fmt.Sprintf("Browser major version %d meets the recommended minimum.", major))
		}
	}

	if !configLoaded || cfg.ExportDir == "" {
		add("export_directory", StatusSkipped, "The export directory could not be checked.")
	} else if info, err := os.Stat(cfg.ExportDir); err != nil || !info.IsDir() {
		add("export_directory", StatusWarning, "The export directory is missing or unavailable; it will be created when needed.")
	} else if info.Mode().Perm()&0o222 == 0 {
		add("export_directory", StatusWarning, "The export directory has no write permission bits.")
	} else {
		add("export_directory", StatusPass, "The export directory exists and has write permission bits.")
	}

	if !options.Network {
		add("network", StatusSkipped, "Network checks were skipped; pass --network to check Duck.ai connectivity.")
	} else if deps.NetworkProbe == nil {
		add("network", StatusSkipped, "Duck.ai connectivity checks are unavailable on this platform.")
	} else {
		networkCtx, cancel := context.WithTimeout(ctx, networkProbeTimeout)
		err := deps.NetworkProbe(networkCtx)
		cancel()
		if err != nil {
			add("network", StatusWarning, "Duck.ai could not be reached or browser capture did not complete; retry when the connection is stable.")
		} else {
			add("network", StatusPass, "Duck.ai connectivity and browser capture are available.")
		}
	}

	return result
}

func browserMajor(version string) int {
	for i := 0; i < len(version); i++ {
		if version[i] < '0' || version[i] > '9' {
			continue
		}
		end := i
		for end < len(version) && version[end] >= '0' && version[end] <= '9' {
			end++
		}
		major, err := strconv.Atoi(version[i:end])
		if err == nil {
			return major
		}
		i = end - 1
	}
	return 0
}
