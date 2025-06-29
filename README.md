# 🦆 DuckDuckGo AI Chat CLI

<p align="center">
  <img src="logo.png" width="220" alt="DuckDuckGo AI Chat CLI Logo">
  <br>
  <strong>🚀 A powerful CLI tool to interact with DuckDuckGo's AI</strong><br>
  <em>Advanced context integration, multi-models and enhanced productivity</em>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go Version">
  <img src="https://img.shields.io/badge/Platform-Linux%20%7C%20Windows%20%7C%20MacOS-blue?style=for-the-badge" alt="Platform">
  <img src="https://img.shields.io/badge/License-Open%20Source-green?style=for-the-badge" alt="License">
  <img src="https://img.shields.io/github/v/release/benoitpetit/duckduckGO-chat-cli?style=for-the-badge" alt="Latest Release">
</p>

<p align="center">
  <a href="#-key-features">Features</a> •
  <a href="#-installation">Installation</a> •
  <a href="#-usage">Usage</a> •
  <a href="#-configuration">Configuration</a> •
  <a href="REVERSE_ENGINEERING_COMPLETE.md">Reverse Engineering</a>
</p>

---

## ✨ Key Features

### 💬 Chat Experience
- **🔄 Real-time streaming** - Live response display with smooth markdown formatting
- **🤖 Multiple AI models** - GPT-4o mini, Claude 3 Haiku, Llama 3.3, Mistral Small, o4-mini & more
- **💻 Terminal-native** - Optimized for command-line workflows with interactive menus
- **⌨️ Smart autocompletion** - Interactive command menus and context-aware suggestions
- **🔑 Auto-authentication** - Seamless session management with dynamic header refresh
- **🔄 Model switching** - Interactive model selection during conversations

### 🧠 Context Integration
- **🔍 Web search** - Integrate DuckDuckGo search results into conversations
- **📄 File processing** - Add local file content (15+ formats: Go, Python, JS, TS, JSON, MD, etc.)
- **🌐 URL scraping** - Extract and analyze webpage content with Chrome-based scraping
- **🚀 Project analysis** - Generate comprehensive project prompts with PMP auto-installation
- **💾 Session persistence** - Maintain conversation history across sessions
- **📚 Library management** - Organize and search through document collections

### 🛠️ Productivity Tools
- **📋 Smart clipboard** - Copy responses, code blocks, or full conversations with interactive selection
- **📤 Advanced export** - Save conversations in multiple formats with search-based filtering
- **📝 History management** - Browse, search, and restore previous conversations with timestamps
- **🔍 Content search** - Search within conversations and document libraries
- **⚙️ Interactive config** - Visual configuration menus for all settings

### 🌐 API Server
- **🚀 REST API** - Built-in HTTP server for external integrations
- **📡 Real-time endpoints** - Chat, history, and status endpoints
- **🔧 Request logging** - Configurable API request/response logging
- **📖 Auto-documentation** - Interactive API documentation at root endpoint

### 📚 Library System
- **📁 Document collections** - Organize files into searchable libraries
- **🔍 Advanced search** - Search across all libraries with pattern matching
- **📊 Library stats** - File counts, sizes, and modification dates
- **🎯 Selective loading** - Load specific libraries or files into context
- **📁 Multi-format support** - 15+ file formats automatically recognized

### 🔧 Advanced Features
- **🛠️ PMP Integration** - Auto-install and use Prompt My Project for code analysis
- **🔄 Dynamic headers** - Automatic browser session management
- **📱 Cross-platform** - Linux, Windows, macOS support
- **🎨 Rich formatting** - Colored output with markdown rendering
- **⚡ Performance** - Efficient memory usage and fast response times

## 🤖 Available Models

| Model Name         | Integration ID                            | Alias          | Strength         | Best For             | Characteristics              |
| :----------------- | :---------------------------------------- | :------------- | :------------------- | :----------------------- | :---------------------------------- |
| **GPT-4o mini**    | gpt-4o-mini                               | gpt-4o-mini    | General purpose      | Everyday questions       | • Fast<br>• Well-balanced           |
| **Claude 3 Haiku** | claude-3-haiku-20240307                   | claude-3-haiku | Creative writing     | Explanations & summaries | • Clear responses<br>• Concise      |
| **Llama 3.3 70B**  | meta-llama/Llama-3.3-70B-Instruct-Turbo   | llama          | Programming          | Code-related tasks       | • Technical precision<br>• Detailed |
| **Mistral Small**  | mistralai/Mistral-Small-24B-Instruct-2501 | mixtral        | Knowledge & analysis | Complex topics           | • Reasoning<br>• Logic-focused      |
| **o4-mini**        | o4-mini                                   | o4mini         | Speed                | Quick answers            | • Very fast<br>• Compact responses  |

## 📦 Installation

> [📥 **Download Latest Release**](https://github.com/benoitpetit/duckduckGO-chat-cli/releases/latest)

### 🚀 1. Direct Download & Run

<details>
<summary><strong>🪟 Windows (PowerShell)</strong></summary>

```powershell
$exe="duckduckgo-chat-cli_windows_amd64.exe"; Invoke-WebRequest -Uri ((Invoke-RestMethod "https://api.github.com/repos/benoitpetit/duckduckGO-chat-cli/releases/latest").assets | Where-Object name -like "*windows_amd64.exe").browser_download_url -OutFile $exe; Start-Process -Wait -NoNewWindow -FilePath ".\$exe"
```

</details>

<details>
<summary><strong>🐧 Linux (curl)</strong></summary>

```bash
curl -LO $(curl -s https://api.github.com/repos/benoitpetit/duckduckGO-chat-cli/releases/latest | grep -oP 'https.*linux_amd64' | grep -oP 'https.*v[0-9]+\.[0-9]+\.[0-9]+_linux_amd64' | head -1) && chmod +x duckduckgo-chat-cli_v*_linux_amd64 && ./duckduckgo-chat-cli_v*_linux_amd64
```

</details>

<details>
<summary><strong>🍎 MacOS (curl)</strong></summary>

```bash
curl -LO $(curl -s https://api.github.com/repos/benoitpetit/duckduckGO-chat-cli/releases/latest | grep -oP 'https.*darwin_arm64' | grep -oP 'https.*v[0-9]+\.[0-9]+\.[0-9]+_darwin_arm64' | head -1) && chmod +x duckduckgo-chat-cli_v*_darwin_arm64 && ./duckduckgo-chat-cli_v*_darwin_arm64
```

</details>

### 🔨 2. Build from source

**📋 Prerequisites:**
- Go 1.21+ (`go version`)
- Chrome/Chromium 115+ (`chromium-browser --version`)

```sh
git clone https://github.com/benoitpetit/duckduckGO-chat-cli
cd duckduckGO-chat-cli
./scripts/build.sh
```

## 🎯 Usage

### 📖 Typical Workflow

<details>
<summary><strong>🔍 Example 1: Code Analysis</strong></summary>

```bash
./duckduckgo-chat-cli_linux_amd64
Accept terms? [yes/no] yes
Type /help to show available commands

You: /search Go concurrency patterns -- What are the best practices?
🔍 Searching for: Go concurrency patterns
✅ Added 10 search results to the context
Processing your request about the search results...

You: /file main.go -- Explain this code and suggest improvements
📄 Adding file content: main.go
✅ Successfully added content from file: main.go
Processing your request about the file...

GPT-4o mini: Based on the search results about Go concurrency patterns and your code...
[Detailed analysis follows]

You: /copy
Choose what to copy:
1) Last Q&A exchange
2) Largest code block
3) Cancel
Enter your choice: 2
✅ Content copied to clipboard
```

</details>

<details>
<summary><strong>🧪 Example 2: Research Assistant</strong></summary>

```bash
You: /url https://en.wikipedia.org/wiki/Quantum_computing -- Summarize the key concepts
[+] URL content processed and summarized
Data extracted (42KB)

You: /search recent quantum computing breakthroughs -- How do these relate to the Wikipedia content?
[+] Search results added and analyzed (10 entries)

GPT-4 Mini: Based on the Wikipedia content and recent breakthroughs...
```

</details>

<details>
<summary><strong>📦 Example 3: Project Analysis with PMP</strong></summary>

```bash
You: /pmp ./src -i "*.go" -e "test/*" -- analyze this Go codebase and suggest improvements
⚠️ PMP (Prompt My Project) is not installed.
Would you like to install PMP automatically? (y/n): y
📦 Installing PMP...
✅ PMP installed successfully!
🔄 Generating project prompt for: ./src
✅ Project prompt added to context (15.2KB)
Processing your request about the project...

GPT-4 Mini: Analyzing your Go codebase structure and code...
Based on the project analysis, here are my suggestions for improvements:
1. Code organization: Consider implementing...
2. Error handling: I notice some patterns that could be improved...
```

</details>

<details>
<summary><strong>📚 Example 4: Library Management Workflow</strong></summary>

```bash
You: /library add ~/projects/myapp
✅ Library added: myapp
   Path: /home/user/projects/myapp

You: /library add ~/documents/api-docs  
✅ Library added: api-docs
   Path: /home/user/documents/api-docs

You: /library list
📚 Configured Libraries:
  1. myapp
     Path: /home/user/projects/myapp
     Files: 47 (125.3 KB)
  2. api-docs
     Path: /home/user/documents/api-docs
     Files: 23 (89.1 KB)

You: /library search config
🔍 Searching for files matching: config
Found 3 matching files:
  1. [myapp] src/config.go (2.1 KB, 2024-01-15 14:30)
  2. [myapp] docker/config.yml (856 B, 2024-01-15 10:22)
  3. [api-docs] configuration.md (4.2 KB, 2024-01-14 16:45)

You: /library load myapp -- analyze the architecture of this project
📚 Loading library: myapp
📄 Loading 47 files (125.3KB)...
✅ Successfully loaded 47 files from library: myapp
Processing your request about the library...

GPT-4o mini: Based on the 47 files from your project, I can see this is a Go-based application with the following architecture:
[Detailed architectural analysis follows]

You: /export
Export options:
1. Full conversation
2. Last AI response
3. Largest code block
4. Search in conversation
5. Cancel

Enter your choice (1-5): 1
✅ Saved to: /home/user/Documents/duckchat/conversation_20240127_143022.md
```

</details>


### 📝 Command Reference

| Command           | Example                  | Description                     |
| ----------------- | ------------------------ | ------------------------------- |
| 🔍 `/search <query> [-- prompt]` | `/search machine learning -- What are the best practices?`   | Add search results as context and optionally process them with a prompt   |
| 📁 `/file <path> [-- prompt]`    | `/file src/main.go -- Explain this code`      | Import file content as context and optionally analyze it with a prompt  |
| 📚 `/library [command] [args]`   | `/library add /path/to/docs` | Manage library directories for bulk file operations |
| 🌐 `/url <link> [-- prompt]`     | `/url github.com/golang -- Summarize this page` | Add webpage content as context and optionally process it with a prompt  |
| 📦 `/pmp [path] [options] [-- prompt]` | `/pmp . -i "*.go" -- analyze this codebase` | Generate structured project prompts with automatic PMP installation |
| 📡 `/api [port]`         | `/api` or `/api 8080`    | Start or stop the API server    |
| 🤖 `/model`          | `/model` or `/model 2`   | Change AI model (interactive)   |
| 🧹 `/clear`          | `/clear`                 | Reset conversation context      |
| 📤 `/export`         | `/export`                | Export content (interactive)    |
| 📋 `/copy`           | `/copy`                  | Copy to clipboard (interactive) |
| 📚 `/history`        | `/history`               | Display conversation history    |
| ⚙️ `/config`         | `/config`                | Modify configuration settings   |
| 🏷️ `/version`        | `/version`               | Show version and system info    |
| ❓ `/help`           | `/help`                  | Show available commands         |
| 🚪 `/exit`           | `/exit`                  | Exit application                |

#### 📚 Library Command Details

The `/library` command provides advanced file management capabilities:

| Subcommand | Example | Description |
|------------|---------|-------------|
| `/library list` | `/library` or `/library list` | List all configured library directories |
| `/library add <path>` | `/library add /home/user/docs` | Add a directory as a library |
| `/library remove <n>` | `/library remove 1` | Remove library by number or name |
| `/library search <pattern>` | `/library search readme` | Search for files across all libraries |
| `/library search <pattern> <lib>` | `/library search config myproject` | Search in specific library |
| `/library load <lib> [-- request]` | `/library load docs -- summarize all files` | Load all files from library into context |

**Supported file types:** `.txt`, `.md`, `.json`, `.yaml`, `.yml`, `.xml`, `.csv`, `.log`, `.ini`, `.conf`, `.cfg`, `.py`, `.go`, `.js`, `.ts`, `.html`, `.css`, `.sql`, `.sh`, `.bat`, `.ps1`, `.php`, `.java`, `.cpp`, `.c`, `.h`, `.hpp`, `.rs`, `.rb`, `.pl`, `.r`

#### 📦 PMP (Prompt My Project) Integration

The `/pmp` command integrates with [Prompt My Project](https://github.com/benoitpetit/prompt-my-project) for advanced codebase analysis:

| Usage | Example | Description |
|-------|---------|-------------|
| `/pmp` | `/pmp` | Generate prompt for current directory |
| `/pmp <path>` | `/pmp ./src` | Generate prompt for specific directory |
| `/pmp <path> [options]` | `/pmp . -i "*.go" -e "test/*"` | Filter files with include/exclude patterns |
| `/pmp help` | `/pmp help` | Show detailed PMP usage and options |

**Key Features:**
- 🚀 **Auto-installation**: Automatically installs PMP if not found
- 🎯 **Smart filtering**: Include/exclude files by patterns
- 📊 **Project analysis**: Comprehensive code structure and content
- 🔧 **Cross-platform**: Works on Linux, macOS, and Windows

**Common Options:**
- `-i "*.ext"` - Include only files matching pattern
- `-e "pattern"` - Exclude files matching pattern  
- `--max-files <n>` - Limit number of files (default: 500)
- `--max-size <size>` - Maximum file size (default: 100MB)

#### 📤 Export Command Details

The `/export` command provides multiple export options:

| Export Type | Description | Output |
|-------------|-------------|---------|
| **Full conversation** | Complete chat history with metadata | Markdown file with timestamps |
| **Last AI response** | Most recent AI answer only | Formatted response with context |
| **Largest code block** | Biggest code snippet from last response | Clean code extraction |
| **Search in conversation** | Find and export specific content | Filtered conversation matching search terms |

**Features:**
- 📝 **Markdown format**: Well-structured output with metadata
- 🕒 **Timestamps**: All exports include timing information
- 🎯 **Smart filtering**: Context-aware content organization
- 📁 **Auto-naming**: Files named with type and timestamp

#### 📋 Copy Command Details

The `/copy` command offers quick clipboard operations:

| Copy Option | Description | Use Case |
|-------------|-------------|----------|
| **Last Q&A exchange** | Previous question and answer pair | Quick sharing of solutions |
| **Largest code block** | Biggest code snippet from response | Copying code for implementation |

**Features:**
- ⚡ **Instant access**: Direct clipboard integration
- 🧠 **Smart detection**: Automatically finds code blocks
- 🔍 **Context preservation**: Maintains question-answer relationships

## ⚙️ Configuration

### 🎛️ Application Settings

| Option           | Description               | Default              | Range              |
| ---------------- | ------------------------- | -------------------- | ------------------ |
| `DefaultModel`   | Starting AI model         | gpt-4o-mini          | 5 models available |
| `GlobalPrompt`   | System prompt always sent | ""                   | Any text           |
| `ExportDir`      | Export directory          | ~/Documents/duckchat | Any valid path     |
| `ShowMenu`       | Display commands on start | true                 | true/false         |
| `SearchSettings` | Search behavior config    | N/A                  | See below          |
| `LibrarySettings`| Library behavior config   | N/A                  | See below          |
| `APISettings`    | API server behavior config| N/A                  | See below          |

### 🔍 Search Settings

| Option           | Description               | Default | Range      |
| ---------------- | ------------------------- | ------- | ---------- |
| `MaxResults`     | Results per search        | 10      | 1-20       |
| `IncludeSnippet` | Show result descriptions  | true    | true/false |
| `MaxRetries`     | Connection retry attempts | 3       | 1-5        |
| `RetryDelay`     | Seconds between retries   | 1       | 1-10       |

### 📚 Library Settings

| Option           | Description               | Default | Range      |
| ---------------- | ------------------------- | ------- | ---------- |
| `Enabled`        | Enable library system     | true    | true/false |
| `Directories`    | List of library paths     | []      | Array of paths |

### 📡 API Settings

| Option        | Description               | Default | Range           |
|---------------|---------------------------|---------|-----------------|
| `Enabled`     | Enable API server         | `false` | `true`/`false`  |
| `Port`        | API server port           | `8080`  | Any valid port  |
| `Autostart`   | Start API on app launch   | `false` | `true`/`false`  |
| `LogRequests` | Log incoming API requests | `true`  | `true`/`false`  |

> 💡 **Tip:** Use `/config` to modify these settings interactively.

### 📁 Configuration Files

- **Windows:** `%APPDATA%\duckduckgo-chat-cli\config.json`
- **Linux/macOS:** `~/.config/duckduckgo-chat-cli/config.json`

### 🛠️ Configuration Structure

```json
{
  "tos_accepted": true,
  "default_model": "gpt-4o-mini",
  "export_dir": "~/Documents/duckchat",
  "show_menu": true,
  "global_prompt": "",
  "search": {
    "max_results": 10,
    "include_snippet": true,
    "max_retries": 3,
    "retry_delay": 1
  },
  "library": {
    "enabled": true,
    "directories": [
      "/path/to/docs",
      "/path/to/projects"
    ]
  },
  "api": {
    "enabled": false,
    "port": 8080,
    "autostart": false,
    "log_requests": true
  }
}
```

## 📤 Export Features

### 🗂️ Export Options

1. **Complete conversation** (`/export` → 1)
2. **Last AI response only** (`/export` → 2)
3. **Code blocks only** (`/export` → 3)
4. **Search by keyword** (`/export` → 4)

### 📋 Clipboard Functions

- **Copy last Q&A exchange** (`/copy` → 1)
- **Copy largest code block** (`/copy` → 2)

## 🔧 Technical Details

### 📊 **Content Limits**
- **Files:** 5MB recommended max
- **URL content:** ~100KB max extraction
- **Search results:** Limited by config (default 10)

### 🔒 **Security**
- Auto token refresh
- Persistent cookie handling
- Automatic retry on API errors

### 📦 **Dependencies**
- Chrome/Chromium 115+ (for web scraping)
- Go 1.21+ (for building from source)

## 🛠️ Development & Contributing

### 🚀 Automated Release Process (beta)

This project uses GitHub Actions for automated building and releasing:

- **Development:** Work on the `master` branch
- **Release:** Create PR to `prod` branch to trigger automatic release
- **CI/CD:** Automated testing, building, and publishing

#### 📋 Release Workflow

1. **Create a release branch:**
   ```bash
   ./scripts/release.sh          # Interactive mode
   ./scripts/release.sh 1.2.0    # Specific version
   ```

2. **Or manually:**
   ```bash
   git checkout -b release/v1.2.0
   git push origin release/v1.2.0
   ```

3. **Create PR from `release/v1.2.0` to `prod`**
   - Automatic version detection
   - Cross-platform builds (Linux, Windows, macOS)
   - Release notes generation
   - Asset upload with SHA256 checksums

#### 🧪 Testing

> comming soon

### 📚 Development Documentation

- **[🔧 CI/CD & Release Process](.github/README.md)** - Complete GitHub Actions documentation
- **[🔬 Reverse Engineering](REVERSE_ENGINEERING_COMPLETE.md)** - Technical implementation details

## 🚨 Troubleshooting

### 🔧 Connection Issues

If you encounter connection errors:

```bash
# Try clearing the conversation context to refresh security tokens
/clear

# Check your Chrome/Chromium installation
chromium-browser --version

# Enable debug mode
DEBUG=true ./duckduckgo-chat-cli_linux_amd64
```

### 🩺 Persistent Issues

Persistent connection issues may require:

- Waiting a few minutes between attempts
- Using a different network connection
- A VPN service like [Cloudflare WARP](https://1.1.1.1/)

## 🚀🚀 Related Projects

This project is part of a suite of DuckDuckGo AI Chat tools:

### 🌐 **DuckDuckGo Chat Web Interface**
**Repository:** [github.com/benoitpetit/duckduckGO-chat-interface](https://github.com/benoitpetit/duckduckGO-chat-interface)

A modern web-based interface for DuckDuckGo AI Chat featuring:
- Clean, responsive design
- Real-time streaming responses
- Multi-model support
- Context management tools
- Export and sharing capabilities

### 🚀 **DuckDuckGo Chat API**
**Repository:** [github.com/benoitpetit/duckduckGO-chat-api](https://github.com/benoitpetit/duckduckGO-chat-api)

A RESTful API server for DuckDuckGo AI Chat integration:
- HTTP/HTTPS API endpoints
- Authentication handling
- Request/response management
- Perfect for integrating into existing applications
- Supports all available AI models

> 💡 **Choose your preferred interface:** Command-line (this project), web browser, or API integration!

## 📜 License & Ethics

### 🛡️ Privacy & Responsibility

- **Privacy First:** This tool respects your privacy and stores no personal data
- **Verify Information:** Always verify critical information from AI responses
- **Responsible Use:** Use responsibly and in accordance with DuckDuckGo's terms

---

*🔧 This is an unofficial client and not affiliated with or endorsed by DuckDuckGo*

> **Made with ♥ for the community**
