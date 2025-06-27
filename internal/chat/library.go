package chat

import (
	"duckduckgo-chat-cli/internal/config"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/fatih/color"
)

// SupportedExtensions contains the file extensions that can be read as text
var SupportedExtensions = map[string]bool{
	".txt":  true,
	".md":   true,
	".json": true,
	".yaml": true,
	".yml":  true,
	".xml":  true,
	".csv":  true,
	".log":  true,
	".ini":  true,
	".conf": true,
	".cfg":  true,
	".py":   true,
	".go":   true,
	".js":   true,
	".ts":   true,
	".html": true,
	".css":  true,
	".sql":  true,
	".sh":   true,
	".bat":  true,
	".ps1":  true,
	".php":  true,
	".java": true,
	".cpp":  true,
	".c":    true,
	".h":    true,
	".hpp":  true,
	".rs":   true,
	".rb":   true,
	".pl":   true,
	".r":    true,
}

type LibraryInfo struct {
	Name      string
	Path      string
	FileCount int
	TotalSize int64
}

type FileInfo struct {
	Path    string
	Name    string
	Size    int64
	ModTime string
}

// HandleLibraryCommand processes the /library command
func HandleLibraryCommand(c *Chat, input string, cfg *config.Config) {
	// Parse the command: /library [list|add <path>|remove <name>|search <pattern> [library]|load <library>] [-- <request>]
	commandInput := strings.TrimPrefix(input, "/library ")
	commandInput = strings.TrimSpace(commandInput)

	if commandInput == "" || commandInput == "list" {
		listLibraries(cfg)
		return
	}

	var subCommand, argument, userRequest string

	// Check if there's a -- separator for user request
	if strings.Contains(commandInput, " -- ") {
		parts := strings.SplitN(commandInput, " -- ", 2)
		commandInput = strings.TrimSpace(parts[0])
		if len(parts) > 1 {
			userRequest = strings.TrimSpace(parts[1])
		}
	}

	// Parse subcommand and argument
	parts := strings.SplitN(commandInput, " ", 2)
	subCommand = parts[0]
	if len(parts) > 1 {
		argument = strings.TrimSpace(parts[1])
	}

	switch subCommand {
	case "add":
		if argument == "" {
			color.Red("Usage: /library add <directory_path>")
			return
		}
		handleLibraryAdd(cfg, argument)

	case "remove", "rm":
		if argument == "" {
			color.Red("Usage: /library remove <library_number_or_name>")
			return
		}
		handleLibraryRemove(cfg, argument)

	case "search":
		if argument == "" {
			color.Red("Usage: /library search <pattern> [library_name]")
			return
		}
		handleLibrarySearch(cfg, argument)

	case "load":
		if argument == "" {
			color.Red("Usage: /library load <library_number_or_name> [-- request]")
			return
		}
		handleLibraryLoad(c, cfg, argument, userRequest)

	default:
		showLibraryHelp()
	}
}

// showLibraryHelp displays usage information for the library command
func showLibraryHelp() {
	color.Red("Usage: /library [list|add <path>|remove <name>|search <pattern> [library]|load <library>] [-- request]")
	color.White("Commands:")
	color.White("  /library list                              - List all configured libraries")
	color.White("  /library add /path/to/docs                 - Add a directory as a library")
	color.White("  /library remove 1                          - Remove library by number")
	color.White("  /library search readme                     - Search for files in all libraries")
	color.White("  /library search readme my_docs             - Search in specific library")
	color.White("  /library load my_docs -- summarize files  - Load all files from library into context")
}

// listLibraries displays all configured libraries
func listLibraries(cfg *config.Config) {
	if !cfg.Library.Enabled {
		color.Yellow("⚠️ Library system is disabled. Enable it in /config → Library Settings")
		return
	}

	color.Yellow("📚 Configured Libraries:")

	if len(cfg.Library.Directories) == 0 {
		color.Yellow("No libraries configured. Use '/library add <path>' to add one.")
		return
	}

	libraries := make([]LibraryInfo, 0, len(cfg.Library.Directories))

	for i, dir := range cfg.Library.Directories {
		libInfo := LibraryInfo{
			Name: getLibraryName(dir),
			Path: dir,
		}

		// Count files and calculate total size
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil // Continue despite errors
			}

			if !d.IsDir() {
				ext := strings.ToLower(filepath.Ext(path))
				if SupportedExtensions[ext] {
					if info, err := d.Info(); err == nil {
						libInfo.FileCount++
						libInfo.TotalSize += info.Size()
					}
				}
			}
			return nil
		})

		if err != nil {
			color.Yellow("  %d. %s (error reading directory)", i+1, libInfo.Name)
		} else {
			libraries = append(libraries, libInfo)
			sizeStr := formatFileSize(libInfo.TotalSize)
			color.White("  %d. %s", i+1, libInfo.Name)
			color.White("     Path: %s", libInfo.Path)
			color.White("     Files: %d (%s)", libInfo.FileCount, sizeStr)
		}
	}

	if len(libraries) > 0 {
		color.White("\nUse '/library load <number>' to load a library into context")
		color.White("Use '/library search <pattern>' to search in all libraries")
	}
}

// handleLibraryAdd adds a new directory as a library
func handleLibraryAdd(cfg *config.Config, path string) {
	// Expand tilde and resolve path
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, path[2:])
		}
	}

	// Convert to absolute path
	if absPath, err := filepath.Abs(path); err == nil {
		path = absPath
	}

	// Check if directory exists
	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		color.Red("❌ Directory does not exist or is not accessible: %s", path)
		return
	}

	// Check if already exists
	for _, existing := range cfg.Library.Directories {
		if existing == path {
			color.Yellow("⚠️ Library already exists: %s", getLibraryName(path))
			return
		}
	}

	// Add to configuration
	cfg.Library.Directories = append(cfg.Library.Directories, path)
	if err := config.SaveConfig(cfg); err != nil {
		color.Red("❌ Error saving config: %v", err)
		return
	}

	color.Green("✅ Library added: %s", getLibraryName(path))
	color.White("   Path: %s", path)
}

// handleLibraryRemove removes a library from configuration
func handleLibraryRemove(cfg *config.Config, argument string) {
	if len(cfg.Library.Directories) == 0 {
		color.Yellow("No libraries to remove")
		return
	}

	var indexToRemove int = -1

	// Try to parse as number first
	if num, err := strconv.Atoi(argument); err == nil {
		if num > 0 && num <= len(cfg.Library.Directories) {
			indexToRemove = num - 1
		} else {
			color.Red("❌ Invalid library number. Use '/library list' to see available libraries")
			return
		}
	} else {
		// Try to find by name
		argument = strings.ToLower(argument)
		for i, dir := range cfg.Library.Directories {
			if strings.Contains(strings.ToLower(getLibraryName(dir)), argument) {
				indexToRemove = i
				break
			}
		}

		if indexToRemove == -1 {
			color.Red("❌ Library not found: %s", argument)
			return
		}
	}

	// Remove from configuration
	removed := cfg.Library.Directories[indexToRemove]
	cfg.Library.Directories = append(cfg.Library.Directories[:indexToRemove], cfg.Library.Directories[indexToRemove+1:]...)

	if err := config.SaveConfig(cfg); err != nil {
		color.Red("❌ Error saving config: %v", err)
		return
	}

	color.Green("✅ Library removed: %s", getLibraryName(removed))
	color.White("   Path: %s", removed)
	color.Yellow("   (Files were not deleted, only removed from library list)")
}

// handleLibrarySearch searches for files in libraries
func handleLibrarySearch(cfg *config.Config, argument string) {
	if !cfg.Library.Enabled {
		color.Yellow("⚠️ Library system is disabled")
		return
	}

	if len(cfg.Library.Directories) == 0 {
		color.Yellow("⚠️ No libraries configured")
		return
	}

	// Parse search argument: "pattern [library_name]"
	parts := strings.SplitN(argument, " ", 2)
	pattern := strings.ToLower(parts[0])
	var targetLibrary string
	if len(parts) > 1 {
		targetLibrary = strings.ToLower(parts[1])
	}

	color.Yellow("🔍 Searching for files matching: %s", pattern)
	if targetLibrary != "" {
		color.Yellow("   In library: %s", targetLibrary)
	}

	var searchDirs []string
	if targetLibrary != "" {
		// Search in specific library
		for _, dir := range cfg.Library.Directories {
			if strings.Contains(strings.ToLower(getLibraryName(dir)), targetLibrary) {
				searchDirs = append(searchDirs, dir)
				break
			}
		}
		if len(searchDirs) == 0 {
			color.Red("❌ Library not found: %s", targetLibrary)
			return
		}
	} else {
		// Search in all libraries
		searchDirs = cfg.Library.Directories
	}

	matchingFiles := []FileInfo{}

	for _, dir := range searchDirs {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}

			if !d.IsDir() {
				ext := strings.ToLower(filepath.Ext(path))
				if SupportedExtensions[ext] {
					fileName := strings.ToLower(filepath.Base(path))
					if strings.Contains(fileName, pattern) {
						info, err := d.Info()
						if err != nil {
							return nil
						}

						relPath, _ := filepath.Rel(dir, path)
						fileInfo := FileInfo{
							Path:    path,
							Name:    fmt.Sprintf("[%s] %s", getLibraryName(dir), relPath),
							Size:    info.Size(),
							ModTime: info.ModTime().Format("2006-01-02 15:04"),
						}
						matchingFiles = append(matchingFiles, fileInfo)
					}
				}
			}
			return nil
		})

		if err != nil {
			color.Red("Error searching in library %s: %v", getLibraryName(dir), err)
		}
	}

	if len(matchingFiles) == 0 {
		color.Yellow("No files found matching pattern: %s", pattern)
		return
	}

	sort.Slice(matchingFiles, func(i, j int) bool {
		return matchingFiles[i].Name < matchingFiles[j].Name
	})

	color.Green("Found %d matching files:", len(matchingFiles))
	for i, file := range matchingFiles {
		sizeStr := formatFileSize(file.Size)
		color.White("  %d. %s (%s, %s)", i+1, file.Name, sizeStr, file.ModTime)
	}
}

// handleLibraryLoad loads all files from a library into context
func handleLibraryLoad(c *Chat, cfg *config.Config, argument string, userRequest string) {
	if !cfg.Library.Enabled {
		color.Yellow("⚠️ Library system is disabled")
		return
	}

	if len(cfg.Library.Directories) == 0 {
		color.Yellow("⚠️ No libraries configured")
		return
	}

	var targetDir string

	// Try to parse as number first
	if num, err := strconv.Atoi(argument); err == nil {
		if num > 0 && num <= len(cfg.Library.Directories) {
			targetDir = cfg.Library.Directories[num-1]
		} else {
			color.Red("❌ Invalid library number. Use '/library list' to see available libraries")
			return
		}
	} else {
		// Try to find by name
		argument = strings.ToLower(argument)
		for _, dir := range cfg.Library.Directories {
			if strings.Contains(strings.ToLower(getLibraryName(dir)), argument) {
				targetDir = dir
				break
			}
		}

		if targetDir == "" {
			color.Red("❌ Library not found: %s", argument)
			return
		}
	}

	color.Yellow("📚 Loading library: %s", getLibraryName(targetDir))

	// Collect all files
	var allFiles []FileInfo
	var totalSize int64

	err := filepath.WalkDir(targetDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if !d.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			if SupportedExtensions[ext] {
				info, err := d.Info()
				if err != nil {
					return nil
				}

				relPath, _ := filepath.Rel(targetDir, path)
				fileInfo := FileInfo{
					Path:    path,
					Name:    relPath,
					Size:    info.Size(),
					ModTime: info.ModTime().Format("2006-01-02 15:04"),
				}
				allFiles = append(allFiles, fileInfo)
				totalSize += info.Size()
			}
		}
		return nil
	})

	if err != nil {
		color.Red("❌ Error reading library: %v", err)
		return
	}

	if len(allFiles) == 0 {
		color.Yellow("⚠️ No supported files found in library")
		return
	}

	// Sort files by name
	sort.Slice(allFiles, func(i, j int) bool {
		return allFiles[i].Name < allFiles[j].Name
	})

	color.Yellow("📄 Loading %d files (%s)...", len(allFiles), formatFileSize(totalSize))

	// Add all files to context
	var contextBuilder strings.Builder
	contextBuilder.WriteString(fmt.Sprintf("[Library Context: %s]\n", getLibraryName(targetDir)))
	contextBuilder.WriteString(fmt.Sprintf("Path: %s\n", targetDir))
	contextBuilder.WriteString(fmt.Sprintf("Files: %d\n\n", len(allFiles)))

	filesAdded := 0
	for _, file := range allFiles {
		content, err := os.ReadFile(file.Path)
		if err != nil {
			color.Yellow("⚠️ Could not read file: %s (%v)", file.Name, err)
			continue
		}

		contextBuilder.WriteString(fmt.Sprintf("=== File: %s ===\n", file.Name))
		contextBuilder.WriteString(string(content))
		contextBuilder.WriteString("\n\n")
		filesAdded++
	}

	if filesAdded == 0 {
		color.Red("❌ No files could be loaded")
		return
	}

	// Add to chat context
	c.Messages = append(c.Messages, Message{
		Role:    "user",
		Content: contextBuilder.String(),
	})

	color.Green("✅ Successfully loaded %d files from library: %s", filesAdded, getLibraryName(targetDir))

	// If user provided a specific request, process it with the library context
	if userRequest != "" {
		color.Cyan("Processing your request about the library...")
		ProcessInput(c, userRequest, cfg)
	} else {
		color.White("Library files added to context. You can now ask questions about them.")
	}
}

// getLibraryName extracts a readable name from a directory path
func getLibraryName(path string) string {
	return filepath.Base(path)
}

// formatFileSize formats file size in human readable format
func formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}
