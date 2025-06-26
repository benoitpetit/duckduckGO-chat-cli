package models

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"github.com/fatih/color"
)

const (
	StatusURL        = "https://duckduckgo.com/duckchat/v1/status"
	ChatURL          = "https://duckduckgo.com/duckchat/v1/chat"
	StatusHeaders    = "1"
	MinChromeVersion = "115.0.5790.110"
)

const chromeRegistryPath = `HKEY_CURRENT_USER\Software\Google\Chrome\BLBeacon`

type Model string
type ModelAlias string

const (
	GPT4Mini Model = "gpt-4o-mini"
	Claude3  Model = "claude-3-haiku-20240307"
	Llama    Model = "meta-llama/Llama-3.3-70B-Instruct-Turbo"
	Mixtral  Model = "mistralai/Mistral-Small-24B-Instruct-2501"
	o4mini   Model = "o4-mini"

	GPT4MiniAlias ModelAlias = "gpt-4o-mini"
	Claude3Alias  ModelAlias = "claude-3-haiku"
	LlamaAlias    ModelAlias = "llama"
	MixtralAlias  ModelAlias = "mixtral"
	o4miniAlias   ModelAlias = "o4mini"
)

var modelMap = map[ModelAlias]Model{
	GPT4MiniAlias: GPT4Mini,
	Claude3Alias:  Claude3,
	LlamaAlias:    Llama,
	MixtralAlias:  Mixtral,
	o4miniAlias:   o4mini,
}

func GetModel(alias string) Model {
	if model, ok := modelMap[ModelAlias(alias)]; ok {
		return model
	}
	return GPT4Mini // default model
}

func CheckChromeVersion() {
	version, err := getChromeVersion()
	if err != nil {
		color.Yellow("Warning: %v", err)
		color.Yellow("Continuing without Chrome version check...")
		return // Continue instead of panic
	}

	result, err := compareVersions(version, MinChromeVersion)
	if err != nil {
		color.Yellow("Warning: Chrome version check failed: %v", err)
		color.Yellow("Continuing without version verification...")
		return // Continue instead of panic
	}

	if result < 0 {
		color.Yellow("Warning: Chrome %s+ recommended, found %s", MinChromeVersion, version)
		color.Yellow("The application might still work, but it's recommended to upgrade Chrome")
	}
}

func compareVersions(v1, v2 string) (int, error) {
	// Clean version strings
	v1 = strings.TrimSpace(v1)
	v1 = strings.TrimPrefix(v1, "Google Chrome ")
	v1 = strings.TrimPrefix(v1, "Chromium ")

	// Remove "snap" suffix if present
	if idx := strings.Index(v1, " snap"); idx != -1 {
		v1 = v1[:idx]
	}

	v1parts := strings.Split(v1, ".")
	v2parts := strings.Split(v2, ".")

	if len(v1parts) == 0 || len(v2parts) == 0 {
		return 0, fmt.Errorf("invalid version format")
	}

	// Compare major version numbers
	v1num, err := strconv.Atoi(strings.TrimSpace(v1parts[0]))
	if err != nil {
		return 0, fmt.Errorf("invalid version: %s", v1)
	}

	v2num, err := strconv.Atoi(strings.TrimSpace(v2parts[0]))
	if err != nil {
		return 0, fmt.Errorf("invalid version: %s", v2)
	}

	if v1num > v2num {
		return 1, nil
	} else if v1num < v2num {
		return -1, nil
	}
	return 0, nil
}

func getChromeVersion() (string, error) {
	switch runtime.GOOS {
	case "windows":
		paths := []string{
			"reg query \"HKEY_CURRENT_USER\\Software\\Google\\Chrome\\BLBeacon\" /v version",
			"reg query \"HKLM\\SOFTWARE\\Wow6432Node\\Microsoft\\Windows\\CurrentVersion\\Uninstall\\Google Chrome\" /v Version",
		}

		for _, cmd := range paths {
			if output, err := exec.Command("cmd", "/C", cmd).Output(); err == nil {
				if version := extractWindowsVersion(string(output)); version != "" {
					return version, nil
				}
			}
		}
		return "", fmt.Errorf("chrome not found in Windows registry")

	case "linux":
		browsers := []string{
			"google-chrome",
			"google-chrome-stable",
			"chromium",
			"chromium-browser",
		}

		for _, browser := range browsers {
			if output, err := exec.Command(browser, "--version").Output(); err == nil {
				return strings.TrimSpace(string(output)), nil
			}
		}

		// Vérifier les chemins communs
		paths := []string{
			"/usr/bin/google-chrome",
			"/usr/bin/chromium",
			"/snap/bin/chromium",
		}

		for _, path := range paths {
			if _, err := os.Stat(path); err == nil {
				if output, err := exec.Command(path, "--version").Output(); err == nil {
					return strings.TrimSpace(string(output)), nil
				}
			}
		}

		return "", fmt.Errorf("neither Chrome nor Chromium found on system")

	case "darwin":
		paths := []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
		}

		for _, path := range paths {
			if output, err := exec.Command(path, "--version").Output(); err == nil {
				return strings.TrimSpace(string(output)), nil
			}
		}
		return "", fmt.Errorf("chrome not found on macOS")

	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func extractWindowsVersion(output string) string {
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "REG_SZ") {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				return fields[len(fields)-1]
			}
		}
	}
	return ""
}

func HandleModelChange(chat interface{}, modelArg string) ModelAlias {
	currentModel := GetCurrentModel(chat)

	// Si un argument est fourni, essayer de le traiter directement
	if modelArg != "" {
		if modelAlias := validateModelChoice(modelArg); modelAlias != "" {
			return modelAlias
		}
		color.Red("Invalid model choice: %s", modelArg)
		return ""
	}

	// Afficher le menu interactif si aucun argument n'est fourni
	color.Yellow("\nAvailable models:")
	color.White("Current model: %s", currentModel)
	color.White("1) GPT-4o-mini")
	color.White("2) Claude-3-haiku")
	color.White("3) Llama 3.3")
	color.White("4) Mistral Small 3")
	color.White("5) o4-mini")
	color.White("6) Cancel")

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\nEnter your choice (1-6): ")
	choice, err := reader.ReadString('\n')
	if err != nil {
		color.Red("Error reading input: %v", err)
		return ""
	}
	return validateModelChoice(strings.TrimSpace(choice))
}

func validateModelChoice(choice string) ModelAlias {
	switch strings.ToLower(choice) {
	case "1", "gpt4mini", "gpt-4o-mini":
		return GPT4MiniAlias
	case "2", "claude3", "claude-3-haiku":
		return Claude3Alias
	case "3", "llama":
		return LlamaAlias
	case "4", "mixtral":
		return MixtralAlias
	case "5", "o4mini":
		return o4miniAlias
	case "6", "cancel":
		color.Yellow("Model change canceled")
		return ""
	default:
		return ""
	}
}

func GetCurrentModel(chat interface{}) Model {
	v := reflect.ValueOf(chat)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if modelField := v.FieldByName("Model"); modelField.IsValid() {
		return Model(modelField.String())
	}
	return ""
}
