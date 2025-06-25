# 🦆 DuckDuckGo AI Chat CLI

<p align="center">
  <img src="logo.png" width="200" alt="DuckDuckGo AI Chat CLI Logo">
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
  <a href="REVERSE_ENGINEERING_COMPLETE.md">🔬 Reverse Engineering</a>
</p>

---

## ✨ Key Features

<table>
<tr>
<td>

### 💬 Chat Experience
- Streaming responses
- Multiple AI models
- Terminal integration
- Auto token refresh

</td>
<td>

### 🧠 Context Enhancement
- Web search integration
- File content importing
- URL content scraping
- Session management

</td>
<td>

### 🛠️ Productivity Tools
- Clipboard integration
- Flexible export options
- Conversation history
- Customizable settings

</td>
</tr>
</table>

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
<summary><strong>:apple: MacOS (curl)</strong></summary>

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

You: /search Go concurrency patterns
[+] Search results added

You: /file main.go
[+] File content processed
File analyzed (2.3KB)

You: How can I improve this implementation?
GPT-4 Mini: Analyzing your code...
```

</details>

<details>
<summary><strong>🧪 Example 2: Research Assistant</strong></summary>

```bash
You: /url https://en.wikipedia.org/wiki/Quantum_computing
[+] URL content processed
Data extracted (42KB)

You: /search recent quantum computing breakthroughs
[+] Search results added (10 entries)

You: Can you provide a summary of the latest advancements?
GPT-4 Mini: Sure! Here are the key points...
```

</details>

### 📝 Command Reference

| Command           | Example                  | Description                     |
| ----------------- | ------------------------ | ------------------------------- |
| 🔍 `/search <query>` | `/search Go tutorials`   | Add search results as context   |
| 📁 `/file <path>`    | `/file src/main.go`      | Import file content as context  |
| 🌐 `/url <link>`     | `/url github.com/golang` | Add webpage content as context  |
| 🤖 `/model`          | `/model` or `/model 2`   | Change AI model (interactive)   |
| 🧹 `/clear`          | `/clear`                 | Reset conversation context      |
| 📤 `/export`         | `/export`                | Export content (interactive)    |
| 📋 `/copy`           | `/copy`                  | Copy to clipboard (interactive) |
| 📚 `/history`        | `/history`               | Display conversation history    |
| ⚙️ `/config`         | `/config`                | Modify configuration settings   |
| 🏷️ `/version`        | `/version`               | Show version and system info    |
| ❓ `/help`           | `/help`                  | Show available commands         |
| 🚪 `/exit`           | `/exit`                  | Exit application                |

## ⚙️ Configuration

### 🎛️ Application Settings

| Option           | Description               | Default              | Range              |
| ---------------- | ------------------------- | -------------------- | ------------------ |
| `DefaultModel`   | Starting AI model         | gpt-4o-mini          | 5 models available |
| `GlobalPrompt`   | System prompt always sent | ""                   | Any text           |
| `ExportDir`      | Export directory          | ~/Documents/duckchat | Any valid path     |
| `ShowMenu`       | Display commands on start | true                 | true/false         |
| `SearchSettings` | Search behavior config    | N/A                  | See below          |

### 🔍 Search Settings

| Option           | Description               | Default | Range      |
| ---------------- | ------------------------- | ------- | ---------- |
| `MaxResults`     | Results per search        | 10      | 1-20       |
| `IncludeSnippet` | Show result descriptions  | true    | true/false |
| `MaxRetries`     | Connection retry attempts | 3       | 1-5        |
| `RetryDelay`     | Seconds between retries   | 1       | 1-10       |

> 💡 **Tip:** Use `/config` to modify these settings interactively.

### 📁 Configuration Files

- **Windows:** `%APPDATA%\duckduckgo-chat-cli\config.json`
- **Linux/macOS:** `~/.config/duckduckgo-chat-cli/config.json`

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

## 🔬 Reverse Engineering

> **📋 Technical Implementation Details**  
> Discover the technical details of DuckDuckGo API reverse engineering, including the anti-418 solution and complete system architecture.
> 
> **[🔍 View complete reverse engineering documentation](REVERSE_ENGINEERING_COMPLETE.md)**
>
> - Anti-error 418 solution (98.3% reduction)
> - Automatic VQD token system
> - Dynamic headers and cookie management
> - Functional auto-recovery

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

## �📜 License & Ethics

### 🛡️ Privacy & Responsibility

- **Privacy First:** This tool respects your privacy and stores no personal data
- **Verify Information:** Always verify critical information from AI responses
- **Responsible Use:** Use responsibly and in accordance with DuckDuckGo's terms

---

*🔧 This is an unofficial client and not affiliated with or endorsed by DuckDuckGo*

> **Made with ♥ for the community**
