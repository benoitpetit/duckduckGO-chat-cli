package command

import (
	"fmt"
	"regexp"
	"strings"
)

// GetSupportedCommands returns a list of all supported commands
func GetSupportedCommands() []string {
	return []string{
		"/help", "/exit", "/clear", "/history", "/search", "/file",
		"/library", "/url", "/pmp", "/export", "/copy", "/config",
		"/model", "/version", "/api", "/stats",
	}
}

// IsChainableCommand checks if a command can be used in a chain
func IsChainableCommand(cmdType string) bool {
	chainableCommands := map[string]bool{
		"/search":  true,
		"/file":    true,
		"/url":     true,
		"/library": true,
	}

	return chainableCommands[cmdType]
}

// ExtractArguments extracts arguments from a command using regex patterns
func ExtractArguments(cmd *Command) map[string]string {
	args := make(map[string]string)

	switch cmd.Type {
	case "/search":
		// Extract search query
		if cmd.Args != "" {
			args["query"] = cmd.Args
		}

	case "/file":
		// Extract file path
		if cmd.Args != "" {
			args["path"] = cmd.Args
		}

	case "/url":
		// Extract URL
		if cmd.Args != "" {
			args["url"] = cmd.Args
		}

	case "/library":
		// Extract library subcommand and arguments
		parts := strings.Fields(cmd.Args)
		if len(parts) > 0 {
			args["subcommand"] = parts[0]
			if len(parts) > 1 {
				args["argument"] = strings.Join(parts[1:], " ")
			}
		}

	case "/model":
		// Extract model name/number
		if cmd.Args != "" {
			args["model"] = cmd.Args
		}

	case "/api":
		// Extract port number
		if cmd.Args != "" {
			args["port"] = cmd.Args
		}

	case "/export":
		// Extract export type
		if cmd.Args != "" {
			args["type"] = cmd.Args
		}

	case "/stats":
		// Stats command doesn't need arguments
		break
	}

	return args
}

// ValidateCommand performs additional validation on a parsed command
func ValidateCommand(cmd *Command) error {
	switch cmd.Type {
	case "/file":
		if cmd.Args == "" {
			return fmt.Errorf("/file command requires a file path")
		}

	case "/url":
		if cmd.Args == "" {
			return fmt.Errorf("/url command requires a URL")
		}
		// Basic URL validation
		urlPattern := regexp.MustCompile(`^https?://|^[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
		if !urlPattern.MatchString(cmd.Args) {
			return fmt.Errorf("invalid URL format: %s", cmd.Args)
		}

	case "/search":
		if cmd.Args == "" {
			return fmt.Errorf("/search command requires a search query")
		}

	case "/model":
		// Model validation can be added here if needed

	case "/api":
		// Port validation can be added here if needed

	case "/stats":
		// Stats command doesn't need validation
		break
	}

	return nil
}

// FormatCommand formats a command for display
func FormatCommand(cmd *Command) string {
	if cmd.Args != "" {
		return fmt.Sprintf("%s %s", cmd.Type, cmd.Args)
	}
	return cmd.Type
}

// FormatChainedCommand formats a chained command for display
func FormatChainedCommand(chainedCmd *ChainedCommand) string {
	var parts []string

	for _, cmd := range chainedCmd.Commands {
		parts = append(parts, FormatCommand(cmd))
	}

	result := strings.Join(parts, " && ")

	if chainedCmd.Prompt != "" {
		if result != "" {
			result += " -- " + chainedCmd.Prompt
		} else {
			result = chainedCmd.Prompt
		}
	}

	return result
}

// IsValidCommand checks if a command type is valid
func IsValidCommand(cmdType string) bool {
	validCommands := map[string]bool{
		"/help":    true,
		"/exit":    true,
		"/clear":   true,
		"/history": true,
		"/search":  true,
		"/file":    true,
		"/library": true,
		"/url":     true,
		"/pmp":     true,
		"/export":  true,
		"/copy":    true,
		"/config":  true,
		"/model":   true,
		"/version": true,
		"/api":     true,
		"/stats":   true, // New command for analytics
	}

	return validCommands[cmdType]
}
