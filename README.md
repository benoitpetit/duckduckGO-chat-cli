# 🦆 DuckDuckGo AI Chat CLI

**A powerful CLI tool to interact with DuckDuckGo's AI**  
_Advanced context integration, multi-models and enhanced productivity_

## ✨ Key Features

| **Smart Chat**        | **Context Management**  | **Integrations**   |
| -------------------- | --------------------- | ----------------- |
| ▶️ Real-time streaming | 🔍 Integrated web search | 📂 Local files     |
| 🤖 4 AI models        | 🌐 Web extraction       | 📦 Markdown export |
| 🔄 Regeneration (in progress..)     | 🧹 Smart cleanup        | 🕸️ JS rendering    |
| 🎨 Colored output     | ⏳ History              | 🔐 Models switch |

## 🧠 Supported Models

### `GPT-4o mini` (_Recommended_)

- **Optimized for**: Quick, general-purpose responses
- **Use cases**: Common discussions, brainstorming
- **Context limit**: 4K tokens

### `Claude 3 Haiku`

- **Specialty**: Structured data analysis
- **Strength**: Deep contextual understanding
- **Bonus**: Supports complex prompts

### `Llama 3.1 70B`

- **For**: Developers/Data Scientists
- **Asset**: Code generation/technical analysis
- **Configuration**: 8GB RAM minimum

### `Mixtral 8x7B`

- **Expertise**: Specialized topics (medicine, law)
- **Advantage**: Multi-source synthesis
- **Performance**: Slightly higher latency

## 🛠️ Installation

### Prerequisites

- Go 1.21+ (`go version`)
- Chrome/Chromium 115+ (`chromium-browser --version`)
- 500MB disk space

### Installation Methods

```bash
# Linux
curl -LO https://github.com/benoitpetit/duckduckGO-chat-cli/releases/latest/download/duckduckgo-chat-cli_linux_amd64
chmod +x duckduckgo-chat-cli_linux_amd64

# macOS
brew tap benoitpetit/cli && brew install duckduckgo-chat-cli
```

**2. Build from source:**

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

## 🔧 Advanced Configuration

### Environment Variables

```bash
export DDG_TIMEOUT=60        # Request timeout (seconds)
export CHROMEDP_PATH=/usr/bin/chromium  # Custom Chrome path
export MAX_CONTEXT=5000      # Contextual token limit
```

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

# Run in debug mode
DDG_DEBUG=1 ./ddg-chat
```

**Issue**: VQD Token expired  
**Solution**:

```bash
/user : /clear  # Automatically regenerates token
```

**Issue**: High latency  
**Solution**:

- Switch model (`/clear` then choose GPT-4o mini)
- Reduce context size (`export MAX_CONTEXT=3000`)

## 📜 License & Ethics

- **License**: MIT License
- **Data collection**: No personal data stored
- **Caution**: AI outputs may contain errors - always verify critical facts

_This project is not affiliated with DuckDuckGo - use at your own risk_

> Made with ♥ by Benoit Petit - [Contribution guide](CONTRIBUTING.md)

