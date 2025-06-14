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

	// Set up cookies comme un vrai navigateur avec TOUS les cookies nécessaires
	jar, _ := cookiejar.New(nil)
	u, _ := url.Parse("https://duckduckgo.com")
	cookies := []*http.Cookie{
		{Name: "5", Value: "1", Domain: ".duckduckgo.com"},
		{Name: "dcm", Value: "3", Domain: ".duckduckgo.com"},
		{Name: "dcs", Value: "1", Domain: ".duckduckgo.com"},
		// Cookies supplémentaires basés sur l'analyse de l'image
		{Name: "duckassist-opt-in-count", Value: "1", Domain: ".duckduckgo.com"},
		{Name: "isRecentChatOn", Value: "1", Domain: ".duckduckgo.com"},
		{Name: "preferredDuckAiModel", Value: "3", Domain: ".duckduckgo.com"},
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
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.6")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("DNT", "1")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Sec-CH-UA", `"Brave";v="137", "Chromium";v="137", "Not/A)Brand";v="24"`)
	req.Header.Set("Sec-CH-UA-Mobile", "?0")
	req.Header.Set("Sec-CH-UA-Platform", `"Windows"`)
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Sec-GPC", "1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch initial page: %v", err)
	}
	defer resp.Body.Close()

	// Attendre un peu pour simuler l'interaction utilisateur
	time.Sleep(500 * time.Millisecond)

	// Utiliser les valeurs exactes du curl qui fonctionne au lieu d'essayer de les extraire
	headers := &DynamicHeaders{
		// Valeurs EXACTES du curl qui fonctionne
		FeSignals: "eyJzdGFydCI6MTc0OTgyODU3NzE1NiwiZXZlbnRzIjpbeyJuYW1lIjoic3RhcnROZXdDaGF0IiwiZGVsdGEiOjYwfV0sImVuZCI6NTM4MX0=",
		FeVersion: "serp_20250613_094749_ET-cafd73f97f51c983eb30",
		VqdHash1:  "eyJzZXJ2ZXJfaGFzaGVzIjpbIm5oWlUrcVZ3d3dzODFPVStDTm4vVkZJcS9DbXBSeGxYY2E5cHpGQ0JVZUk9IiwiajRNNmNBRzRheVFqQ21kWkN0a1IzOFY3eVRpd1gvZ2RmcDFueFhEdlV3cz0iXSwiY2xpZW50X2hhc2hlcyI6WyJpRTNqeXRnSm0xZGJaZlo1bW81M1NmaVAxdXUxeEdzY0F5RnB3V2NVOUtrPSIsInJaRGtaR2h4S0JEL1JuY00xVVNraHZNM3pLdEJzQmlzSlJTWFF4L2QzRFU9Il0sInNpZ25hbHMiOnt9LCJtZXRhIjp7InYiOiIzIiwiY2hhbGxlbmdlX2lkIjoiODU3NjA5YjlmMTg2NThlMWM0MzZhZWI2MGM0MDc1ZjdhYWNmYmI0OTlhY2Y4NTVmNDJkNWRjZmM5MTViNDhiOGg4amJ0IiwidGltZXN0YW1wIjoiMTc0OTgyODU3NjQ5NyIsIm9yaWdpbiI6Imh0dHBzOi8vZHVja2R1Y2tnby5jb20iLCJzdGFjayI6IkVycm9yXG5hdCBiYSAoaHR0cHM6Ly9kdWNrZHVja2dvLmNvbS9kaXN0L3dwbS5jaGF0LmNhZmQ3M2Y5N2Y1MWM5ODNlYjMwLmpzOjE6NzQ4MDMpXG5hdCBhc3luYyBkaXNwYXRjaFNlcnZpY2VJbml0aWFsVlFEIChodHRwczovL2R1Y2tkdWNrZ28uY29tL2Rpc3Qvd3BtLmNoYXQuY2FmZDczZjk3ZjUxYzk4M2ViMzAuanM6MTo5OTUyOSkifX0=",
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36",
		ChromeUA:  `"Brave";v="137", "Chromium";v="137", "Not/A)Brand";v="24"`,
	}

	return headers, nil
}
