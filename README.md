# DuckDuckGo AI Chat CLI

<p align="center">
  <img src="logo.png" width="200" alt="DuckDuckGo AI Chat CLI Logo">
  <br>
<strong>A powerful CLI tool to interact with DuckDuckGo's AI</strong><br>
<em>Advanced context integration, multi-models and enhanced productivity</em>
</p>

## Key Features

| Chat Experience      | Context Enhancement    | Productivity Tools      |
| :------------------- | :--------------------- | :---------------------- |
| Streaming responses  | Web search integration | Clipboard integration   |
| Multiple AI models   | File content importing | Flexible export options |
| Terminal integration | URL content scraping   | Conversation history    |
| Auto token refresh   | Session management     | Customizable settings   |

## Available Models

| Model Name         | Integration ID                            | Alias          | Strength             | Best For                 | Characteristics                     |
| :----------------- | :---------------------------------------- | :------------- | :------------------- | :----------------------- | :---------------------------------- |
| **GPT-4o mini**    | gpt-4o-mini                               | gpt-4o-mini    | General purpose      | Everyday questions       | • Fast<br>• Well-balanced           |
| **Claude 3 Haiku** | claude-3-haiku-20240307                   | claude-3-haiku | Creative writing     | Explanations & summaries | • Clear responses<br>• Concise      |
| **Llama 3.3 70B**  | meta-llama/Llama-3.3-70B-Instruct-Turbo   | llama          | Programming          | Code-related tasks       | • Technical precision<br>• Detailed |
| **Mistral Small**  | mistralai/Mistral-Small-24B-Instruct-2501 | mixtral        | Knowledge & analysis | Complex topics           | • Reasoning<br>• Logic-focused      |
| **o3-mini**        | o3-mini                                   | o3mini         | Speed                | Quick answers            | • Very fast<br>• Compact responses  |

## Installation

[Last Release version](https://github.com/benoitpetit/duckduckGO-chat-cli/releases/latest)

### 1. Direct Download & Run

**Windows (PowerShell)**

```powershell
$exe="duckduckgo-chat-cli_windows_amd64.exe"; Invoke-WebRequest -Uri ((Invoke-RestMethod "https://api.github.com/repos/benoitpetit/duckduckGO-chat-cli/releases/latest").assets | Where-Object name -like "*windows_amd64.exe").browser_download_url -OutFile $exe; Start-Process -Wait -NoNewWindow -FilePath ".\$exe"
```

**Linux (curl)**

```bash
curl -LO $(curl -s https://api.github.com/repos/benoitpetit/duckduckGO-chat-cli/releases/latest | grep -oP 'https.*linux_amd64' | grep -oP 'https.*v[0-9]+\.[0-9]+\.[0-9]+_linux_amd64' | head -1) && chmod +x duckduckgo-chat-cli_v*_linux_amd64 && ./duckduckgo-chat-cli_v*_linux_amd64
```

### 2. Build from source

Prerequisites:

- Go 1.21+ (`go version`)
- Chrome/Chromium 115+ (`chromium-browser --version`)

```sh
git clone https://github.com/benoitpetit/duckduckGO-chat-cli
cd duckduckGO-chat-cli
./scripts/build.sh
```

## Usage

### Typical Workflow

Example 1: Code Analysis

```bash
./duckduckgo-chat-cli_linux_amd64
Accept terms? [yes/no] yes
Type /help to show available commands

You: /search Go concurrency patterns
[+] Search results added

You: /file main.go
[+] File content processed
File analyzed (2.3KB)

You: How can I improve this implementation?
GPT-4 Mini: Analyzing your code...
```

Example 2: Research Assistant

```bash
You: /url https://en.wikipedia.org/wiki/Quantum_computing
[+] URL content processed
Data extracted (42KB)

You: /search recent quantum computing breakthroughs
[+] Search results added (10 entries)

You: Can you provide a summary of the latest advancements?
GPT-4 Mini: Sure! Here are the key points...
```

### Command Reference

| Command           | Example                  | Description                     |
| ----------------- | ------------------------ | ------------------------------- |
| `/search <query>` | `/search Go tutorials`   | Add search results as context   |
| `/file <path>`    | `/file src/main.go`      | Import file content as context  |
| `/url <link>`     | `/url github.com/golang` | Add webpage content as context  |
| `/model`          | `/model` or `/model 2`   | Change AI model (interactive)   |
| `/clear`          | `/clear`                 | Reset conversation context      |
| `/export`         | `/export`                | Export content (interactive)    |
| `/copy`           | `/copy`                  | Copy to clipboard (interactive) |
| `/history`        | `/history`               | Display conversation history    |
| `/config`         | `/config`                | Modify configuration settings   |
| `/help`           | `/help`                  | Show available commands         |
| `/exit`           | `/exit`                  | Exit application                |

## Configuration

### Application Settings

| Option           | Description               | Default              | Range              |
| ---------------- | ------------------------- | -------------------- | ------------------ |
| `DefaultModel`   | Starting AI model         | gpt-4o-mini          | 5 models available |
| `GlobalPrompt`   | System prompt always sent | ""                   | Any text           |
| `ExportDir`      | Export directory          | ~/Documents/duckchat | Any valid path     |
| `ShowMenu`       | Display commands on start | true                 | true/false         |
| `SearchSettings` | Search behavior config    | N/A                  | See below          |

### Search Settings

| Option           | Description               | Default | Range      |
| ---------------- | ------------------------- | ------- | ---------- |
| `MaxResults`     | Results per search        | 10      | 1-20       |
| `IncludeSnippet` | Show result descriptions  | true    | true/false |
| `MaxRetries`     | Connection retry attempts | 3       | 1-5        |
| `RetryDelay`     | Seconds between retries   | 1       | 1-10       |

Use `/config` to modify these settings interactively.

### Configuration Files

- Windows: `%APPDATA%\duckduckgo-chat-cli\config.json`
- Linux/macOS: `~/.config/duckduckgo-chat-cli/config.json`

## Export Features

### Export Options

1. Complete conversation (`/export` → 1)
2. Last AI response only (`/export` → 2)
3. Code blocks only (`/export` → 3)
4. Search by keyword (`/export` → 4)

### Clipboard Functions

- Copy last Q&A exchange (`/copy` → 1)
- Copy largest code block (`/copy` → 2)

## Technical Details

- **Content Limits**:
  - Files: 5MB recommended max
  - URL content: ~100KB max extraction
  - Search results: Limited by config (default 10)
- **Security**:
  - Auto token refresh
  - Persistent cookie handling
  - Automatic retry on API errors
- **Dependencies**:
  - Chrome/Chromium 115+ (for web scraping)
  - Go 1.21+ (for building from source)

## Troubleshooting

If you encounter connection errors:

```bash
# Try clearing the conversation context to refresh security tokens
/clear

# Check your Chrome/Chromium installation
chromium-browser --version

# Enable debug mode
DEBUG=true ./duckduckgo-chat-cli_linux_amd64
```

Persistent connection issues may require:

- Waiting a few minutes between attempts
- Using a different network connection
- A VPN service like [Cloudflare WARP](https://1.1.1.1/)

## License & Ethics

- This tool respects your privacy and stores no personal data
- Always verify critical information from AI responses
- Use responsibly and in accordance with DuckDuckGo's terms

_This is an unofficial client and not affiliated with or endorsed by DuckDuckGo_

> Made with ♥ for the community
