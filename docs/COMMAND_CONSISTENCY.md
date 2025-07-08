# 🎯 Command Consistency System

## Overview

This document describes the centralized command consistency system implemented to ensure all CLI commands are synchronized across all components of the DuckDuckGo Chat CLI.

## ✨ Problem Solved

Previously, commands were defined in multiple places throughout the codebase:
- Manual hardcoded lists in `main.go` (autocompletion)
- Separate validation functions in `internal/command/command.go`
- Manual command lists in help messages
- Documentation in README.md

This led to potential inconsistencies when adding or modifying commands.

## 🔧 Solution Implemented

### 1. Centralized Command Registry

Created a centralized `CommandRegistry` in `internal/command/command.go` that defines all commands with metadata:

```go
type CommandInfo struct {
    Name          string  // Command name (e.g., "/help")
    Description   string  // Human-readable description
    Usage         string  // Usage syntax (e.g., "/search <query> [-- prompt]")
    IsChainable   bool    // Can be used in command chains
    RequiresArgs  bool    // Requires arguments to function
    Category      string  // Command category (core, context, productivity)
}
```

### 2. Automatic Synchronization

All CLI components now automatically use the centralized registry:

- **Autocompletion** (`main.go`): Dynamically generates suggestions from registry
- **Command validation** (`command.go`): Uses registry for validation
- **Help system** (`chat.go`): Automatically organizes commands by category
- **Chainable commands**: Automatically determined from registry metadata

### 3. Consistency Validation Tool

Created `scripts/check-commands.sh` to validate command consistency:

```bash
./scripts/check-commands.sh
```

This tool:
- ✅ Validates all command metadata is complete
- ✅ Checks for duplicate commands
- ✅ Verifies consistency across all components
- 📊 Shows command statistics and categories
- 🔗 Identifies chainable commands

## 📋 Current Commands (17 total)

### Core Commands (10)
- `/help` - Show the welcome message and command list
- `/exit` - Exit the chat
- `/clear` - Clear the chat history
- `/history` - Show the chat history
- `/model` - Change the chat model
- `/config` - Open the configuration menu
- `/version` - Show version information
- `/api` - Start or stop the API server interactively
- `/stats` - Show real-time session analytics
- `/update` - Update the CLI to the latest version

### Context Commands (5)
- `/search` - Search with a query 🔗 ⚠️
- `/file` - Chat with a file 🔗
- `/library` - Chat with your library 🔗
- `/url` - Chat with a URL 🔗 ⚠️
- `/pmp` - Use a predefined prompt

### Productivity Commands (2)
- `/export` - Export the chat history
- `/copy` - Copy the last response to the clipboard

**Legend:**
- 🔗 = Chainable command (can be used with `&&`)
- ⚠️ = Requires arguments

## 🚀 Adding New Commands

To add a new command, follow these steps:

### 1. Update the Command Registry

Add the command to `internal/command/command.go`:

```go
"/newcommand": {
    Name:         "/newcommand",
    Description:  "Description of what this command does",
    Usage:        "/newcommand <arg> [-- prompt]",
    IsChainable:  true,  // If it can be used in chains
    RequiresArgs: true,  // If it requires arguments
    Category:     "context", // core, context, or productivity
},
```

### 2. Implement the Handler

Add the command handler in `cmd/duckchat/main.go`:

```go
case cmd.Type == "/newcommand":
    // Implementation here
    handleNewCommand(chatSession, cmd.Raw, cfg)
```

### 3. Add Specific Logic (if needed)

Create handler functions in `internal/chat/` if the command needs complex logic.

### 4. Validate Consistency

Run the consistency checker:

```bash
./scripts/check-commands.sh
```

### 5. Update Documentation

The help system will automatically include the new command, but update README.md if needed.

## 🎯 Benefits

1. **Single Source of Truth**: All command definitions in one place
2. **Automatic Synchronization**: No manual updates needed across components
3. **Type Safety**: Structured command metadata prevents errors
4. **Easy Maintenance**: Adding commands is now a simple, guided process
5. **Validation**: Automated checking prevents inconsistencies
6. **Better Organization**: Commands are logically categorized

## 🔍 Validation Features

The consistency checker validates:

- ✅ All commands have complete metadata
- ✅ No duplicate commands exist
- ✅ Registry matches supported commands list
- ✅ Chainable commands are properly identified
- ✅ Command categories are consistent

## 📈 Impact

- **Before**: Manual synchronization, potential inconsistencies
- **After**: Automatic synchronization, guaranteed consistency
- **Developer Experience**: Simplified command addition process
- **Maintenance**: Reduced cognitive load and error potential

This system ensures that all 17 commands remain perfectly synchronized across the entire CLI application, making maintenance easier and preventing future inconsistencies. 