# DuckDuckGo AI Chat CLI

**A powerful CLI tool to interact with DuckDuckGo's AI**  
_Advanced context integration, multi-models and enhanced productivity_

## ✨ Key Features

| Chat Features | Context Management | Tools & Export |
|---------------|-------------------|----------------|
| ▶️ Live streaming responses | 🔍 Web search integration | 📋 Clipboard support |
| 🌈 Smart formatting | 📂 File content import | 💻 Code block extraction |
| 🎨 Syntax highlighting | 🌐 URL content scraping | 📜 History viewer |
| 🤖 5 AI models | 🧹 Context clearing | ⚙️ Configurable settings |

## 🧠 Available Models

| Model | Performance | Best For | Features |
|-------|------------|----------|-----------|
| **GPT-4o mini** | Fast | Quick answers & basic tasks | • Default model<br>• General-purpose |
| **Claude 3 Haiku** | Balanced | Technical discussions | • Good context handling<br>• Structured responses |
| **Llama 3.3** | Code-optimized | Programming tasks | • Documentation analysis<br>• Code generation |
| **Mixtral 8x7B** | Knowledge-focused | Complex topics | • Detailed explanations<br>• Deep analysis |
| **o3-mini** | Fastest | Simple queries | • Lightweight<br>• Quick responses |

## 🛠️ Installation

### 1. Direct Download & Run

**Windows (PowerShell)**
```powershell
Invoke-WebRequest -Uri ((Invoke-RestMethod -Uri "https://api.github.com/repos/benoitpetit/duckduckGO-chat-cli/releases/latest").assets | Where-Object name -like "*windows_amd64.exe").browser_download_url -OutFile duckduckgo-chat-cli_windows_amd64.exe
./duckduckgo-chat-cli_windows_amd64.exe
```

**Linux (curl)**
```bash
curl -LO $(curl -s https://api.github.com/repos/benoitpetit/duckduckGO-chat-cli/releases/latest | grep -oP 'https.*linux_amd64' | head -1)
chmod +x duckduckgo-chat-cli_linux_amd64
./duckduckgo-chat-cli_linux_amd64
```

### 2. Build from source

Prerequisites:
- Go 1.21+ (`go version`)
- Chrome/Chromium 115+ (`chromium-browser --version`)
- 500MB disk space

```bash
git clone https://github.com/benoitpetit/duckduckGO-chat-cli
cd duckduckGO-chat-cli
go build -ldflags "-s -w" -o ddg-chat
```

## 🚀 Usage

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

Example 2: Market Analysis
```bash
You: /url https://coinmarketcap.com/currencies/xrp/
[+] URL content processed
Data extracted (42KB)

You: /search xrp news
[+] Search results added (10 entries)

You: Can you provide a market analysis report for XRP based on current data and news?
GPT-4 Mini: Analyzing market data and recent news for XRP...
• Current price trends
• Recent developments
• Market sentiment analysis
• Key news impact
```

### Essential Commands
| Command | Example | Description |
|---------|---------|-------------|
| `/search <query>` | `/search Go tutorials` | Add search results to context |
| `/file <path>` | `/file src/main.go` | Import file content |
| `/url <link>` | `/url github.com/golang` | Add webpage content |
| `/model` | `/model` or `/model 2` | Change AI model (interactive) |
| `/clear` | `/clear` | Reset conversation |
| `/export` | `/export` | Export content (interactive) |
| `/copy` | `/copy` | Copy to clipboard (interactive) |
| `/history` | `/history` | Show chat history |
| `/config` | `/config` | Configure settings |
| `/help` | `/help` | Show available commands |
| `/exit` | `/exit` | Quit application |

## 🛠️ Configuration

### Main Settings
| Option | Description | Default | Range |
|--------|-------------|---------|--------|
| `DefaultModel` | Starting AI model | gpt-4o-mini | 5 models available |
| `ExportDir` | Export directory | ~/Documents/duckchat | Any valid path |
| `ShowMenu` | Display commands menu | true | true/false |
| `SearchSettings` | Access search configuration | N/A | See below |

### Search Settings
| Option | Description | Default | Range |
|--------|-------------|---------|--------|
| `MaxResults` | Results per search | 10 | 1-10 |
| `IncludeSnippet` | Show result descriptions | true | true/false |
| `MaxRetries` | Connection retry attempts | 3 | 1-5 |
| `RetryDelay` | Seconds between retries | 1 | 1-10 |

Use `/config` to modify these settings interactively through the CLI.

### Persistent Storage
- Configuration paths:
  - Windows: `C:\Users\<username>\AppData\Roaming\duckduckgo-chat-cli\config.json`
  - Linux/macOS: `~/.config/duckduckgo-chat-cli/config.json`
- Export paths:
  - Windows: `C:\Users\<username>\Documents\duckchat\`
  - Linux/macOS: `~/Documents/duckchat/`

## 🔄 Export Features

### Available Formats
1. Full conversation (`/export` → 1)
2. Last AI response (`/export` → 2)
3. Code blocks (`/export` → 3)
4. Search results (`/export` → 4)

### Copy Options
- `/copy` → 1: Copy last Q&A exchange
- `/copy` → 2: Copy largest code block

## ⚠️ Technical Limitations

- **File Size**: Recommended max 5MB per file
- **URL Content**: Max 100KB of extracted text
- **Search Results**: Limited to 10 results per query
- **Network**: Requires stable internet connection
- **Rate Limiting**: Automatic retry on 429 errors
- **Token Refresh**: Auto-refresh on expiration

### Markdown Export Format

```markdown
---
date: 2024-03-15 14:30:00
model: gpt-4o-mini
context_size: 5
---

# DuckDuckGo AI Chat Export

## Search Context (14:30)
...
```

## Prerequisites

- Chrome/Chromium 115.0.5790.110 or higher
- Go 1.21+
- 500MB disk space

## 🚨 Troubleshooting

**Issue**: Web extraction failure
**Solution**:
```bash
# Check Chrome version
chromium-browser --version  # Should show ≥ 115.0.5790.110
apt install chromium OR apt install chromium-browser
```

**Issue**: VQD Token expired  
**Solution**:

```bash
user : /clear  # Automatically regenerates token
```

If the issue persists:
- Wait a few minutes and try again
- Change your IP address using a VPN. 
- It appears to work with [Cloudflare WARP](https://1.1.1.1/)

## 📜 License & Ethics

- **Data collection**: No personal data stored
- **Caution**: AI outputs may contain errors - always verify critical facts

_This project is not affiliated with DuckDuckGo - use at your own risk_
> Made with ♥ for the community
