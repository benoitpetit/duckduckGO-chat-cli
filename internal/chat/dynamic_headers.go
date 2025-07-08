package chat

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"
)

type DynamicHeaders struct {
	FeSignals string
	FeVersion string
	VqdHash1  string
	UserAgent string
	ChromeUA  string
}

// ExtractDynamicHeaders utilise maintenant les valeurs exactes des curl qui fonctionnent
func ExtractDynamicHeaders() (*DynamicHeaders, error) {
	client := &http.Client{Timeout: 15 * time.Second}

	// Set up cookies comme dans les nouvelles requêtes curl
	jar, _ := cookiejar.New(nil)
	u, _ := url.Parse("https://duckduckgo.com")
	cookies := []*http.Cookie{
		{Name: "5", Value: "1", Domain: ".duckduckgo.com"},
		{Name: "dcm", Value: "3", Domain: ".duckduckgo.com"},
		{Name: "dcs", Value: "1", Domain: ".duckduckgo.com"},
	}
	jar.SetCookies(u, cookies)
	client.Jar = jar

	// Étape 1: Visiter la page principale pour établir une session
	req, err := http.NewRequest("GET", "https://duckduckgo.com/?q=DuckDuckGo+AI+Chat&ia=chat&duckai=1", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create initial request: %v", err)
	}

	// Headers exactement comme dans le curl qui fonctionne
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.5")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("DNT", "1")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Sec-CH-UA", `"Not)A;Brand";v="8", "Chromium";v="138", "Brave";v="138"`)
	req.Header.Set("Sec-CH-UA-Mobile", "?0")
	req.Header.Set("Sec-CH-UA-Platform", `"Windows"`)
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Sec-GPC", "1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch initial page: %v", err)
	}
	defer resp.Body.Close()

	// Attendre un peu pour simuler l'interaction utilisateur
	time.Sleep(500 * time.Millisecond)

	// Utiliser les valeurs exactes du curl qui fonctionne au lieu d'essayer de les extraire
	headers := &DynamicHeaders{
		// Valeurs EXACTES du curl qui fonctionne - mises à jour pour Chrome 138
		FeSignals: "eyJzdGFydCI6MTc1MTc1MTg4NTc3MiwiZXZlbnRzIjpbeyJuYW1lIjoic3RhcnROZXdDaGF0IiwiZGVsdGEiOjk1fSx7Im5hbWUiOiJyZWNlbnRDaGF0c0xpc3RJbXByZXNzaW9uIiwiZGVsdGEiOjIxOX0seyJuYW1lIjoiaW5pdFN3aXRjaE1vZGVsIiwiZGVsdGEiOjI2OTd9LHsibmFtZSI6InN0YXJ0TmV3Q2hhdCIsImRlbHRhIjo4MTYxfV0sImVuZCI6MjU0ODN9",
		FeVersion: "serp_20250704_184539_ET-8bee6051143b0c382099",
		VqdHash1:  "eyJzZXJ2ZXJfaGFzaGVzIjpbIjdYbEtTdFJxbkRDbVV6dEh2TkVBMm9kYXB5S3NKR21WSVYxZG4xWHpHbFk9Iiwic3pKR05nSytIV3pHWXVIR0taU1NjVXhOU2EyQmhJMy9XbExvalNzUDZRZz0iLCJhNzFZL05QM2RnMGoyUEEzK2p6S1ovLytnL01HWU1VZjd4ZXlIbkdVMDhFPSJdLCJjbGllbnRfaGFzaGVzIjpbImxWblI0MStCMVFWZ0o4d0hhMUdBNmdxR0JoSjlWdjN5K0dISkdGekJmTGM9IiwiakNoZUlFNUVKUjJlMUlURy9zQzd0N250QnVTQm9qdDY5MVVGNk1BK01pZz0iLCJFczV0akh6VjVTKzNCSEdVTnZ6Z1pZeVAvU3JBa3JETWVBSzlKVUlReDBjPSJdLCJzaWduYWxzIjp7fSwibWV0YSI6eyJ2IjoiNCIsImNoYWxsZW5nZV9pZCI6IjRmZmJhYzliNmIxMGM4MWVmODE0YzgxZTdmMmE4MDkxZDc5ODI0OGI2MDYxMmE0ZTViOGNhYjFhNDRkZjQ0OTRoOGpidCIsInRpbWVzdGFtcCI6IjE3NTE3NTE4ODU0ODMiLCJvcmlnaW4iOiJodHRwczovL2R1Y2tkdWNrZ28uY29tIiwic3RhY2siOiJFcnJvclxuYXQgdWUgKGh0dHBzOi8vZHVja2R1Y2tnby5jb20vZGlzdC93cG0uY2hhdC44YmVlNjA1MTE0M2IwYzM4MjA5OS5qczoxOjI2MTU4KVxuYXQgYXN5bmMgaHR0cHM6Ly9kdWNrZHVja2dvLmNvbS9kaXN0L3dwbS5jaGF0LjhiZWU2MDUxMTQzYjBjMzgyMDk5LmpzOjE6MjgzNDUiLCJkdXJhdGlvbiI6Ijg4In19",
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36",
		ChromeUA:  `"Not)A;Brand";v="8", "Chromium";v="138", "Brave";v="138"`,
	}

	return headers, nil
}
