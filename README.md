# 🦆 DuckDuckGo AI Chat CLI

**A powerful CLI tool to interact with DuckDuckGo's AI**  
_Advanced context integration, multi-models and enhanced productivity_

## ✨ Key Features

| **Smart Chat**        | **Context Management**  | **Integrations**   |
| -------------------- | --------------------- | ----------------- |
| ▶️ Real-time streaming | 🔍 Integrated web search | 📂 Local files     |
| 🤖 4 AI models        | 🌐 Web extraction       | 📦 Markdown export |
| 🔄 Regeneration (in progress..)     | 🧹 Smart cleanup        | 🙊 Show Context    |
| 🎨 Colored output     | ⏳ History              | 🤖 Models switch |

## 🧠 Supported Models

### `GPT-4o mini` (_Recommended_)

- **Optimized for**: Quick, general-purpose responses
- **Use cases**: Common discussions, brainstorming

### `Claude 3 Haiku`

- **Specialty**: Structured data analysis
- **Strength**: Deep contextual understanding
- **Bonus**: Supports complex prompts

### `Llama 3.1 70B`

- **For**: Developers/Data Scientists
- **Asset**: Code generation/technical analysis

### `Mixtral 8x7B`

- **Expertise**: Specialized topics (medicine, law)
- **Advantage**: Multi-source synthesis
- **Performance**: Slightly higher latency

## 🛠️ Installation

## Platform-Specific Downloads

**Linux (64-bit)**
```bash
curl -LO $(curl -s https://api.github.com/repos/benoitpetit/duckduckGO-chat-cli/releases/latest | grep -oP 'https.*linux_amd64' | head -1)
```

**Windows (64-bit)**
```powershell
Invoke-WebRequest -Uri ((Invoke-RestMethod -Uri "https://api.github.com/repos/benoitpetit/duckduckGO-chat-cli/releases/latest").assets | Where-Object name -like "*windows_amd64.exe").browser_download_url -OutFile duckduckgo-chat-cli_windows_amd64.exe
```

**2. Build from source:**

Prerequisites:
- Go 1.21+ (`go version`)
- Chrome/Chromium 115+ (`chromium-browser --version`)
- 500MB disk space


```bash
git clone https://github.com/benoitpetit/duckduckGO-chat-cli
cd duckduckGO-chat-cli
go build -ldflags "-s -w" -o ddg-chat
```

## 🚀 Advanced Usage

### Typical Workflow

```bash
./ddg-chat
> Accept terms? [yes/no] yes
> Choose model (1-4): 2

[Claude 3 Haiku activated]
/user : /search Rust best practices 2025
[+] 10 results added
/user : /file ~/project/src/lib.rs
[+] File analyzed (1.2KB)
/user : How can I improve this implementation?
AI : █ Generating...
```

### Essential Commands

| Command           | Example                          | Result                |
| ---------------- | -------------------------------- | --------------------- |
| `/search <query>`| `/search GPT-5 speculations`     | Injects 10 results   |
| `/file <path>`   | `/file /tmp/notes.md`           | Adds content |
| `/url <link>`    | `/url https://arxiv.org/abs/123`| Extracts content |
| `/clear`         | `/clear`                         | Resets context  |
| `/markdown`      | `/markdown`                      | Generates MD export  |
| `/extract`       | `/extract`                       | extract latest AI message |


### Markdown Export Format

````markdown
# Conversation from 03/15/2024

## Search context (03/15 14:30)

```rust
▸ Rust Security Audit Guide
    "Best practices for unsafe code..."
    https://rustsec.org
```

## User message (03/15 14:32)

How to secure this unsafe block?

## AI Response (03/15 14:33)

1. Use `SafeWrapper` for raw pointers...

````

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

**Issue**: High latency  
**Solution**:

- Switch model (`/model` then choose GPT-4o mini)
- Reduce context size (`export MAX_CONTEXT=3000`) use (`/clear` for clean context)

## 📜 License & Ethics

- **Data collection**: No personal data stored
- **Caution**: AI outputs may contain errors - always verify critical facts

_This project is not affiliated with DuckDuckGo - use at your own risk_

> Made with ♥ by Benoit Petit

