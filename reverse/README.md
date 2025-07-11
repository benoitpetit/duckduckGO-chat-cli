# DuckDuckGo Chat API Reverse Engineering

This document consolidates the findings and implementation details of the DuckDuckGo Chat AI API reverse engineering, focusing on how the CLI client interacts with the API without requiring a headless browser.

## API Logic Overview

The DuckDuckGo Chat AI API is designed to prevent automated access through a combination of dynamic headers, specific cookie requirements, and a unique VQD (Verification Query Data) mechanism. The key to successful interaction lies in accurately mimicking a legitimate browser's request.

The core interaction flow is as follows:
1.  **Initial VQD Acquisition**: A GET request to `/duckchat/v1/status` is made to obtain an `x-vqd-4` header. This VQD is dynamic and changes.
2.  **Static VQD Hash (`x-vqd-hash-1`)**: Crucially, the API requires a specific `x-vqd-hash-1` header for chat requests. This hash is *not* obtained from the `/status` endpoint. Instead, it's a static, base64-encoded JSON string derived from real browser traffic. This was the breakthrough in overcoming 418 "I'm a teapot" errors.
3.  **Mimicking Browser Headers**: A comprehensive set of HTTP headers, including `User-Agent`, `Sec-CH-UA`, `x-fe-signals`, and `x-fe-version`, must be precisely set to match a modern browser (e.g., Chrome 138).
4.  **Essential Cookies**: A minimal set of cookies (`5`, `dcm`, `dcs`) are required to maintain session state.
5.  **Chat Request**: A POST request is sent to `/duckchat/v1/chat` with the constructed payload, all required headers, and cookies.
6.  **Streaming Response**: The API responds with a server-sent events (SSE) stream, delivering the AI's response chunk by chunk.
7.  **VQD Refresh**: If a 418 or 429 error occurs, or if the `x-vqd-4` header changes in a successful response, the client refreshes its VQD and retries the request.

## No Headless Chrome Required

A significant achievement of this reverse engineering is the ability to interact with the DuckDuckGo Chat API *without* using a headless browser like Chrome. This is possible because:

-   **Static `x-vqd-hash-1`**: The most complex anti-bot mechanism, the `x-vqd-hash-1` header, was found to be a relatively static value that can be hardcoded (or periodically updated from real browser traffic). This eliminates the need for a browser to dynamically generate it.
-   **Mimicked Headers**: All other required headers (`User-Agent`, `Sec-CH-UA`, `x-fe-signals`, `x-fe-version`) can be directly set in the HTTP request, as they are static or follow predictable patterns.
-   **Simplified Cookies**: Only a few essential cookies are needed, which can be managed programmatically.

By understanding and replicating the exact HTTP requests a browser makes, the need for a resource-intensive headless browser is circumvented, making the CLI client lightweight and efficient.

## GetVQD() Function Deep Dive

The `GetVQD()` function (located in `internal/chat/chat.go`) plays a crucial role in initializing the chat session.

```go
func GetVQD() (string, string, string, string) {
	// Simple approach like the working PowerShell script
	// Use static headers that work, no complex challenges
	ui.Warningln("⌛ Getting VQD from status API (simple approach like working PS1 script)...")

	client := &http.Client{Timeout: 10 * time.Second}

	// Set up cookies avec les cookies minimum nécessaires (comme dans le script PS1)
	jar, _ := cookiejar.New(nil)
	u, _ := url.Parse("https://duckduckgo.com")
	cookies := []*http.Cookie{
		{Name: "5", Value: "1", Domain: ".duckduckgo.com"},
		{Name: "dcm", Value: "3", Domain: ".duckduckgo.com"},
		{Name: "dcs", Value: "1", Domain: ".duckduckgo.com"},
	}
	jar.SetCookies(u, cookies)
	client.Jar = jar

	// Direct GET to /status with exact headers from working PS1 script
	req, _ := http.NewRequest("GET", models.StatusURL, nil)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.6")
	req.Header.Set("Authority", "duckduckgo.com")
	req.Header.Set("Cache-Control", "no-store")
	req.Header.Set("DNT", "1")
	req.Header.Set("Method", "GET")
	req.Header.Set("Path", "/duckchat/v1/status")
	req.Header.Set("Priority", "u=1, i")
	req.Header.Set("Referer", "https://duckduckgo.com/")
	req.Header.Set("Scheme", "https")
	req.Header.Set("Sec-CH-UA", `"Not)A;Brand";v="8", "Chromium";v="138", "Brave";v="138"`)
	req.Header.Set("Sec-CH-UA-Mobile", "?0")
	req.Header.Set("Sec-CH-UA-Platform", `"Windows"`)
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-GPC", "1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36")
	req.Header.Set("x-vqd-accept", "1")

	resp, err := client.Do(req)
	if err != nil {
		ui.Errorln("Error fetching VQD: %v", err)
		return "", "", "", ""
	}
	defer resp.Body.Close()

	// Le VQD header de la status API pour x-vqd-4
	vqdHeader := resp.Header.Get("x-vqd-hash-1") // This is actually x-vqd-4 in practice
	if vqdHeader == "" {
		ui.Errorln("No VQD header found in response")
		return "", "", "", ""
	}

	// Return the VQD and static headers that work in PowerShell script
	vqd := vqdHeader // This is the x-vqd-4 value
	vqdHash1 := "eyJzZXJ2ZXJfaGFzaGVzIjpbImRQSlJJTWczZnFYQXIvaStaa3c2cEpFVzEwckdTdmxJVlVkNlFsOVRGWXc9IiwiMUN3Qzg3N0Q3WXE1dzlEeTc4UjhBVi9qZVZWaUlYbmV0Q0xvckx3c01QZz0iLCJQSzc3TGc2L25weDdWQ2J2UWxsTEhBR3cyenJIVmEvQUFBRFBhQTl1ekVRPSJdLCJjbGllbnRfaGFzaGVzIjpbImxWblI0MStCMVFWZ0o4d0hhMUdBNmdxR0JoSjlWdjN5K0dISkdGekJmTGM9IiwiVS9RRUc2RE1qdEU4V2hHU1FxOUU1Z0VGNmw1SWJrNk9NVlBuY01DU1licz0iLCJ6SURsYUNvZG9JUjNwbTNSVTlWOUJXaUJkZDJqenRMODAyN0VYTHhkWll3PSJdLCJzaWduYWxzIjp7fSwibWV0YSI6eyJ2IjoiNCIsImNoYWxsZW5nZV9pZCI6ImM4M2Q0ZTc5NTU2MjJmZjU3Mzc0ZDUzOTk2ZjliMmJhZGE2ZDQxZTMzNDM1ZjVlNzMyYjFmNmZjNmQ0ZTE1NzVoOGpidCIsInRpbWVzdGFtcCI6IjE3NTIxNTU3Nzc4NjYiLCJvcmlnaW4iOiJodHRwczovL2R1Y2tkdWNrZ28uY29tIiwic3RhY2siOiJFcnJvclxuYXQgRSAoaHR0cHM6Ly9kdWNrZHVja2dvLmNvbS9kaXN0L3dwbS5jaGF0LjcwZWFjYTZhZWEyOTQ4YjBiYjYwLmpzOjE6MTQ4MjUpXG5hdCBhc3luYyBodHRwczovL2R1Y2tkdWNrZ28uY29tL2Rpc3Qvd3BtLmNoYXQuNzBlYWNhNmFlYTI5NDhiMGJiNjAuanM6MToxNjk4NSIsImR1cmF0aW9uIjoiNTgifX0="
	feSignals := "eyJzdGFydCI6MTc1MjE1NTc3NzQ4MCwiZXZlbnRzIjpbeyJuYW1lIjoic3RhcnROZXdDaGF0IiwiZGVsdGEiOjc1fSx7Im5hbWUiOiJyZWNlbnRDaGF0c0xpc3RJbXByZXNzaW9uIiwiZGVsdGEiOjEyNH1dLCJlbmQiOjQzNDN9"
	feVersion := "serp_20250710_090702_ET-70eaca6aea2948b0bb60"

	ui.AIln("✅ Successfully got VQD and all required headers")
	return vqd, vqdHash1, feSignals, feVersion
}
```

**How `GetVQD()` Works:**

1.  **HTTP Client Setup**: Initializes an `http.Client` with a timeout and a `cookiejar` to manage cookies.
2.  **Essential Cookies**: Sets a minimal set of cookies (`5`, `dcm`, `dcs`) that are necessary for the DuckDuckGo domain.
3.  **Status API Request**: Constructs an HTTP GET request to `models.StatusURL` (`https://duckduckgo.com/duckchat/v1/status`).
4.  **Mimicked Headers for Status**: Sets a comprehensive list of HTTP headers (e.g., `Accept`, `User-Agent`, `Sec-CH-UA`, `Referer`, `x-vqd-accept`) to mimic a real browser's request to the status endpoint.
5.  **`x-vqd-4` Extraction**: It attempts to extract the `x-vqd-hash-1` header from the response of the status API. **However, based on the reverse engineering findings, this header actually contains the `x-vqd-4` value needed for subsequent chat requests.** The variable `vqd` is assigned this value.
6.  **Hardcoded `x-vqd-hash-1`, `x-fe-signals`, `x-fe-version`**: The critical `vqdHash1` (which is the `x-vqd-hash-1` header for chat requests), `feSignals`, and `feVersion` are **hardcoded static strings**. These values were obtained through careful analysis of real browser requests and are not dynamically generated by this function. This is the core reason why headless Chrome is not needed.
7.  **Return Values**: The function returns the `x-vqd-4` (as `vqd`), the hardcoded `x-vqd-hash-1` (as `vqdHash1`), and the hardcoded `feSignals` and `feVersion`.

**In essence, `GetVQD()` primarily fetches the dynamic `x-vqd-4` from the status endpoint, but the most critical anti-bot headers (`x-vqd-hash-1`, `x-fe-signals`, `x-fe-version`) are pre-determined static values.**

## Key Headers Explained

The following HTTP headers are crucial for successful API interaction:

-   **`x-vqd-4`**: A dynamic token obtained from the `/status` endpoint. It's used in subsequent chat requests.
-   **`x-vqd-hash-1`**: The most critical anti-bot header. This is a static, base64-encoded JSON string that contains cryptographic hashes and metadata. It's hardcoded in the client and is essential for avoiding 418 errors.
-   **`x-fe-signals`**: A static, base64-encoded JSON string representing frontend signals.
-   **`x-fe-version`**: A static string indicating the frontend version (e.g., `serp_20250710_090702_ET-70eaca6aea2948b0bb60`).
-   **`User-Agent`**: Identifies the client as a specific browser (e.g., `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36`).
-   **`Sec-CH-UA`**: Client hints for User-Agent (e.g., `"Not)A;Brand";v="8", "Chromium";v="138", "Brave";v="138"`).

## Payload Structure

The `ChatPayload` sent to the `/duckchat/v1/chat` endpoint has the following structure:

```go
type ChatPayload struct {
	Model                models.Model `json:"model"`
	Metadata             Metadata     `json:"metadata"`
	Messages             []Message    `json:"messages"`
	CanUseTools          bool         `json:"canUseTools"`
	CanUseApproxLocation bool         `json:"canUseApproxLocation"`
}

type Message struct {
	Content string `json:"content"`
	Role    string `json:"role"`
}

type Metadata struct {
	ToolChoice ToolChoice `json:"toolChoice"`
}

type ToolChoice struct {
	NewsSearch      bool `json:"NewsSearch"`
	VideosSearch    bool `json:"VideosSearch"`
	LocalSearch     bool `json:"LocalSearch"`
	WeatherForecast bool `json:"WeatherForecast"`
}
```

-   `Model`: The AI model to use (e.g., `gpt-4o-mini`).
-   `Metadata`: Contains `ToolChoice` flags for various search functionalities.
-   `Messages`: An array of `Message` objects, representing the conversation history. Each message has a `Role` (e.g., "user", "assistant") and `Content`.
-   `CanUseTools`: Boolean indicating if the AI can use external tools.
-   `CanUseApproxLocation`: Boolean indicating if approximate location can be used.

## Cookie Management

Only a few essential cookies are required for the DuckDuckGo domain:
-   `5`
-   `dcm`
-   `dcs`

These are set in the `cookiejar` associated with the HTTP client.

## Error Handling (418 / 429)

The client includes logic to handle `418 I'm a teapot` and `429 Too Many Requests` errors. Upon encountering these, or if the `x-vqd-4` header changes, the client attempts to refresh the VQD and retry the request. This provides a degree of resilience against temporary API issues or anti-bot measures.