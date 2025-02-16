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
	Mixtral  Model = "mistralai/Mixtral-8x7B-Instruct-v0.1"
	o3mini   Model = "o3-mini"

	GPT4MiniAlias ModelAlias = "gpt-4o-mini"
	Claude3Alias  ModelAlias = "claude-3-haiku"
	LlamaAlias    ModelAlias = "llama"
	MixtralAlias  ModelAlias = "mixtral"
	o3miniAlias   ModelAlias = "o3mini"
)

var modelMap = map[ModelAlias]Model{
	GPT4MiniAlias: GPT4Mini,
	Claude3Alias:  Claude3,
	LlamaAlias:    Llama,
	MixtralAlias:  Mixtral,
	o3miniAlias:   o3mini,
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
		panic(fmt.Sprintf("Chrome version check failed: %v", err))
	}

	result, err := compareVersions(version, MinChromeVersion)
	if err != nil {
		panic(fmt.Sprintf("Version comparison failed: %v", err))
	}

	if result < 0 {
		panic(fmt.Sprintf("Chrome %s+ required, found %s", MinChromeVersion, version))
	}
}

func compareVersions(v1, v2 string) (int, error) {
	// Clean version string
	v1 = strings.TrimPrefix(strings.TrimSpace(v1), "Google Chrome ")
	v1parts := strings.Split(v1, ".")
	v2parts := strings.Split(v2, ".")

	if len(v1parts) == 0 || len(v2parts) == 0 {
		return 0, fmt.Errorf("invalid version format")
	}

	v1num, err := strconv.Atoi(v1parts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid version: %s", v1)
	}

	v2num, err := strconv.Atoi(v2parts[0])
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
		cmd := exec.Command("reg", "query", chromeRegistryPath, "/v", "version")
		output, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("chrome not found: %v", err)
		}

		// Parse Windows registry output
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "REG_SZ") {
				fields := strings.Fields(line)
				if len(fields) >= 3 {
					version := fields[len(fields)-1]
					return version, nil
				}
			}
		}
		return "", fmt.Errorf("version not found in registry")

	case "linux":
		cmd := exec.Command("chromium-browser", "--version")
		output, err := cmd.Output()
		if err != nil {
			cmd = exec.Command("google-chrome", "--version")
			output, err = cmd.Output()
			if err != nil {
				return "", fmt.Errorf("Chrome/Chromium not found: %v", err)
			}
		}
		return strings.TrimSpace(string(output)), nil

	case "darwin":
		cmd := exec.Command("/Applications/Google Chrome.app/Contents/MacOS/Google Chrome", "--version")
		output, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("chrome not found: %v", err)
		}
		return strings.TrimSpace(string(output)), nil

	default:
		return "", fmt.Errorf("unsupported operating system")
	}
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
	color.White("4) Mixtral 8x7B")
	color.White("5) o3-mini")
	color.White("6) Cancel")

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\nEnter your choice (1-6): ")
	choice, _ := reader.ReadString('\n')
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
	case "5", "o3mini":
		return o3miniAlias
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
