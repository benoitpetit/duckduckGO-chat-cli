# DuckDuckGO CLI v1.0.0

DuckDuckGO CLI v1.0.0 is a command-line interface for interacting with **DuckDuckGo's AI chat service**. This project is a refactored and improved version of the original [duckduckgo-ai-chat](https://github.com/mumu-lhl/duckduckgo-ai-chat) project, **rewritten in Go 🩵** with enhanced streaming capabilities in a CLI.

## Features

- Interactive command-line interface for AI chat
- Support for multiple AI models:
  - GPT-4o mini
  - Claude 3 Haiku
  - Llama 3.1 70B
  - Mixtral 8x7B
- Real-time streaming of AI responses
- Colored output for better readability
- Terms of Service acceptance prompt

## Download

You can download the latest pre-built executable from the [Releases](https://github.com/benoitpetit/duckduckGO-chat-cli/releases) page.

Alternatively, you can use the following commands to download the latest release:

# For Linux (64-bit)
```bash
curl -LO $(curl -s https://api.github.com/repos/benoitpetit/duckduckGO-chat-cli/releases/latest | grep "browser_download_url.*linux_amd64" | cut -d '"' -f 4)
```
# For macOS (64-bit)
```shell
curl -LO $(curl -s https://api.github.com/repos/benoitpetit/duckduckGO-chat-cli/releases/latest | grep "browser_download_url.*darwin_amd64" | cut -d '"' -f 4)
```
# For Windows (64-bit)
```powershell
Invoke-WebRequest -Uri ((Invoke-RestMethod -Uri "https://api.github.com/repos/benoitpetit/duckduckGO-chat-cli/releases/latest").assets | Where-Object name -like "*windows_amd64.exe").browser_download_url -OutFile duckduckgo-chat-cli_windows_amd64.exe
Start-Process -FilePath .\duckduckgo-chat-cli_windows_amd64.exe -Wait -NoNewWindow
```

After downloading, make the file executable (for Unix-based systems):

```bash
chmod +x ./duckchat-cli_*
```

## Installation from Source

If you prefer to build from source:

1. Ensure you have Go installed on your system. If not, you can download it from [golang.org](https://golang.org/).

2. Clone this repository:
   ```
   git clone https://github.com/benoitpetit/duckduckGO-chat-cli.git
   ```

3. Navigate to the project directory:
   ```
   cd duckduckGO-chat-cli
   ```

4. Install dependencies:
   ```
   go mod tidy
   ```

5. Build the project:
   ```
   go build
   ```

## Usage
   
   If you built from source, run the executable in the project directory:
   ```
   ./duckchat-cli
   ```

2. Accept the Terms of Service when prompted.

3. Choose an AI model from the available options.

4. Start chatting! Type your messages and press Enter to send them to the AI.

5. To exit the chat, type 'exit' and press Enter.

## Acknowledgements

This project is inspired by and based on [duckduckgo-ai-chat](https://github.com/mumu-lhl/duckduckgo-ai-chat) by mumu-lhl. The original concept has been refactored into Go, with improvements to the streaming functionality and overall code structure.

## License

[MIT License](LICENSE)

## Disclaimer

This project is not officially affiliated with or endorsed by DuckDuckGo. It is an independent implementation that interacts with DuckDuckGo's AI chat service. Please use responsibly and in accordance with DuckDuckGo's terms of service.
