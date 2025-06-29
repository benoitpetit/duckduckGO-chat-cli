package command

import (
	"errors"
	"strings"
)

// Command represents a single command in a chain.
type Command struct {
	Type string
	Args string
	Raw  string
}

// ChainedCommand represents a parsed chain of commands and an optional prompt.
type ChainedCommand struct {
	Commands []*Command
	Prompt   string
}

// Parse takes a raw input string and parses it into a ChainedCommand.
func Parse(input string) (*ChainedCommand, error) {
	prompt := ""
	commandPart := input

	if strings.Contains(input, "--") {
		parts := strings.SplitN(input, "--", 2)
		if strings.Count(input, "--") > 1 {
			return nil, errors.New("only one prompt (using --) is allowed per command chain")
		}
		commandPart = strings.TrimSpace(parts[0])
		prompt = strings.TrimSpace(parts[1])
	}

	rawCommands := strings.Split(commandPart, "&&")
	if len(rawCommands) == 0 {
		return nil, errors.New("no commands found")
	}

	var commands []*Command
	for _, rawCmd := range rawCommands {
		trimmedCmd := strings.TrimSpace(rawCmd)
		if trimmedCmd == "" {
			continue
		}

		parts := strings.Fields(trimmedCmd)
		if len(parts) == 0 {
			continue
		}

		cmd := &Command{
			Type: parts[0],
			Args: strings.Join(parts[1:], " "),
			Raw:  trimmedCmd,
		}
		commands = append(commands, cmd)
	}

	return &ChainedCommand{
		Commands: commands,
		Prompt:   prompt,
	}, nil
}
