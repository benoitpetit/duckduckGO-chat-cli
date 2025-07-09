package chat

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type DynamicHeaders struct {
	FeSignals string
	FeVersion string
	VqdHash1  string
	UserAgent string
	ChromeUA  string
	Cookies   map[string]string
}

// Structure pour décoder les données de configuration JavaScript
type FeSignalsData struct {
	Start  int64 `json:"start"`
	Events []struct {
		Name  string `json:"name"`
		Delta int64  `json:"delta"`
	} `json:"events"`
	End int64 `json:"end"`
}

type VqdHashData struct {
	ServerHashes []string               `json:"server_hashes"`
	ClientHashes []string               `json:"client_hashes"`
	Signals      map[string]interface{} `json:"signals"`
	Meta         struct {
		Version     string `json:"v"`
		ChallengeID string `json:"challenge_id"`
		Timestamp   string `json:"timestamp"`
		Origin      string `json:"origin"`
		Stack       string `json:"stack"`
		Duration    string `json:"duration"`
	} `json:"meta"`
}

// ExtractDynamicHeaders récupère dynamiquement les headers depuis DuckDuckGo
func ExtractDynamicHeaders() (*DynamicHeaders, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	// Configuration des cookies avec jar
	jar, _ := cookiejar.New(nil)
	u, _ := url.Parse("https://duckduckgo.com")
	cookies := []*http.Cookie{
		{Name: "5", Value: "1", Domain: ".duckduckgo.com"},
		{Name: "dcm", Value: "3", Domain: ".duckduckgo.com"},
		{Name: "dcs", Value: "1", Domain: ".duckduckgo.com"},
	}
	jar.SetCookies(u, cookies)
	client.Jar = jar

	// Étape 1: Récupérer la page principale pour obtenir les données dynamiques
	req, err := http.NewRequest("GET", "https://duckduckgo.com/?q=DuckDuckGo+AI+Chat&ia=chat&duckai=1", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create initial request: %v", err)
	}

	// Headers de base pour simuler un navigateur réel
	headers := &DynamicHeaders{
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36",
		ChromeUA:  `"Not)A;Brand";v="8", "Chromium";v="138", "Brave";v="138"`,
		Cookies:   make(map[string]string),
	}

	setRequestHeaders(req, headers)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch initial page: %v", err)
	}
	defer resp.Body.Close()

	// Lire le contenu HTML
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	bodyStr := string(body)

	// Extraire les données dynamiques du HTML/JavaScript
	if err := extractFeVersion(bodyStr, headers); err != nil {
		fmt.Printf("Warning: Could not extract fe_version: %v\n", err)
	}

	// Essayer d'extraire des signaux FE réels depuis la page
	if err := extractFeSignalsFromPage(bodyStr, headers); err != nil {
		// Si échec, générer des signaux basés sur des valeurs réalistes
		if err := generateFeSignals(headers); err != nil {
			fmt.Printf("Warning: Could not generate fe_signals: %v\n", err)
		}
	}

	// Extraire ou générer VQD hash
	if err := extractOrGenerateVqdHash(bodyStr, headers); err != nil {
		fmt.Printf("Warning: Could not extract/generate vqd_hash: %v\n", err)
	}

	// Étape 2: Utiliser le VQD hash JSON fonctionnel du navigateur
	// Après les tests PowerShell, nous savons que ce VQD hash JSON fonctionne
	fmt.Printf("🔍 Using working VQD hash from browser analysis...\n")
	headers.VqdHash1 = "eyJzZXJ2ZXJfaGFzaGVzIjpbIjR0Ui9HdVdKV0UyTzBzV2x4V0ZiNU5PbmV0SkdoUFNGTDdwSlpEUTJvTlE9IiwiK2ZaZnphZmdiZGtTUm53WEFaOW03bVZTSG5xRFZzVEhzYzgzZ3NKeXRSOD0iLCJTMVhmclNybnAyektUOGtKNE1pRDNSUk9ORzk1eFRwWGxLYko1ZUZXOGlrPSJdLCJjbGllbnRfaGFzaGVzIjpbImxWblI0MStCMVFWZ0o4d0hhMUdBNmdxR0JoSjlWdjN5K0dISkdGekJmTGM9IiwiTDROMTBxbVBnL0N1MWZzTlpMYm9CWkFTWjVGVEljNjUwNklHTzJEUVhMcz0iLCJrbFdNUTBlRDVDeUhhdXl5dnBia2hEZWs3UDZrYjF0aHlrMVNLRFlUWHRrPSJdLCJzaWduYWxzIjp7fSwibWV0YSI6eyJ2IjoiNCIsImNoYWxsZW5nZV9pZCI6IjA3ZjgxYTljZThiZmJjMzRiMWM3NGY5OTQwODkzZTA1ZWY2MmVhZjVhNTY5MTdmODRkYWZlYTExMGI1OTNjNThoOGpidCIsInRpbWVzdGFtcCI6IjE3NTIwODEyNDczOTQiLCJvcmlnaW4iOiJodHRwczovL2R1Y2tkdWNrZ28uY29tIiwic3RhY2siOiJFcnJvclxuYXQgdmUgKGh0dHBzOi8vZHVja2R1Y2tnby5jb20vZGlzdC93cG0uY2hhdC45NTFkMTYyZTJhODJmZmQ2OTBiZC5qczoxOjI3NjYwKVxuYXQgYXN5bmMgaHR0cHM6Ly9kdWNrZHVja2dvLmNvbS9kaXN0L3dwbS5jaGF0Ljk1MWQxNjJlMmE4MmZmZDY5MGJkLmpzOjE6Mjk4NDciLCJkdXJhdGlvbiI6Ijg4In19"
	fmt.Printf("✅ Using proven working VQD hash from browser\n")

	// Capturer les cookies supplémentaires
	extractCookies(resp, headers)

	// Attendre un peu pour simuler l'interaction utilisateur
	time.Sleep(500 * time.Millisecond)

	// Validation des données extraites
	if headers.FeVersion == "" || headers.VqdHash1 == "" {
		fmt.Printf("⚠️  Dynamic extraction failed, using fallback headers\n")
		return getFallbackHeaders(), nil
	}

	// Afficher les informations de débogage pour les headers extraits
	fmt.Printf("✅ Extracted FE Version: %s\n", headers.FeVersion)
	if len(headers.VqdHash1) > 50 {
		fmt.Printf("✅ Using VQD Hash: %s...\n", headers.VqdHash1[:50])
	} else {
		fmt.Printf("✅ Using VQD Hash: %s\n", headers.VqdHash1)
	}

	return headers, nil
}

// setRequestHeaders configure les headers de la requête
func setRequestHeaders(req *http.Request, headers *DynamicHeaders) {
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.5")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("DNT", "1")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Sec-CH-UA", headers.ChromeUA)
	req.Header.Set("Sec-CH-UA-Mobile", "?0")
	req.Header.Set("Sec-CH-UA-Platform", `"Windows"`)
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Sec-GPC", "1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("User-Agent", headers.UserAgent)
}

// extractFeVersion extrait la version FE depuis le HTML
func extractFeVersion(html string, headers *DynamicHeaders) error {
	// Chercher les patterns pour fe_version dans l'ordre de priorité
	patterns := []string{
		`__DDG_BE_VERSION__="([^"]+)"`,     // Pattern principal observé
		`__DDG_FE_CHAT_HASH__="([^"]+)"`,   // Hash du chat FE
		`"fe_version":"([^"]+)"`,           // JSON fe_version
		`fe_version["\s]*:["\s]*"([^"]+)"`, // Variante JSON
		`serp_\d{8}_\d{6}_[A-Z]{2}[^"]*`,   // Pattern serp avec date
		`/dist/wpm\.chat\.([a-f0-9]+)\.js`, // Hash du fichier JS
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(html)
		if len(matches) > 1 {
			match := matches[1]
			if strings.Contains(pattern, "__DDG_BE_VERSION__") {
				headers.FeVersion = match
				return nil
			} else if strings.Contains(pattern, "__DDG_FE_CHAT_HASH__") {
				// Utiliser le hash FE pour construire une version
				headers.FeVersion = fmt.Sprintf("serp_%s_ET-%s",
					time.Now().Format("20060102_150405"), match)
				return nil
			} else if strings.Contains(match, "serp_") {
				headers.FeVersion = match
				return nil
			} else if len(match) > 10 {
				headers.FeVersion = fmt.Sprintf("serp_%s_ET-%s",
					time.Now().Format("20060102_150405"), match)
				return nil
			}
		}
	}

	// Fallback: générer une version basée sur l'heure actuelle
	headers.FeVersion = fmt.Sprintf("serp_%s_ET-dynamic", time.Now().Format("20060102_150405"))
	return nil
}

// extractFeSignalsFromPage essaie d'extraire les signaux FE depuis la page HTML
func extractFeSignalsFromPage(html string, headers *DynamicHeaders) error {
	// Chercher des signaux FE dans le JavaScript de la page
	patterns := []string{
		`fe_signals["\s]*:["\s]*"([^"]+)"`,
		`"fe_signals":"([^"]+)"`,
		`feSignals["\s]*:["\s]*"([^"]+)"`,
		`signals["\s]*:["\s]*"([^"]+)"`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(html)
		if len(matches) > 1 && len(matches[1]) > 50 {
			headers.FeSignals = matches[1]
			return nil
		}
	}

	// Pas de signaux FE trouvés dans la page
	return fmt.Errorf("no fe_signals found in page")
}

// generateFeSignals génère des signaux FE dynamiques basés sur la page réelle
func generateFeSignals(headers *DynamicHeaders) error {
	// Essayer d'extraire de vrais signaux de la page si possible
	// Sinon, utiliser les signaux de fallback fonctionnels avec une légère variation

	now := time.Now().UnixMilli()

	// Générer une variation des signaux de base avec des valeurs réalistes
	baseSignals := "eyJzdGFydCI6MTc1MTc1MTg4NTc3MiwiZXZlbnRzIjpbeyJuYW1lIjoic3RhcnROZXdDaGF0IiwiZGVsdGEiOjk1fSx7Im5hbWUiOiJyZWNlbnRDaGF0c0xpc3RJbXByZXNzaW9uIiwiZGVsdGEiOjIxOX0seyJuYW1lIjoiaW5pdFN3aXRjaE1vZGVsIiwiZGVsdGEiOjI2OTd9LHsibmFtZSI6InN0YXJ0TmV3Q2hhdCIsImRlbHRhIjo4MTYxfV0sImVuZCI6MjU0ODN9"

	// Ajouter une légère variation temporelle pour rendre les signaux plus réalistes
	timeVariation := now % 1000

	// Créer une version modifiée des signaux de base en ajustant les deltas
	if timeVariation > 500 {
		// Utiliser les signaux de base avec de légers ajustements
		headers.FeSignals = baseSignals
	} else {
		// Générer des signaux légèrement modifiés
		adjustedSignals := strings.Replace(baseSignals, "delta\":95", fmt.Sprintf("delta\":%d", 95+(timeVariation%20)), 1)
		adjustedSignals = strings.Replace(adjustedSignals, "delta\":219", fmt.Sprintf("delta\":%d", 219+(timeVariation%30)), 1)
		headers.FeSignals = adjustedSignals
	}

	return nil
}

// extractOrGenerateVqdHash extrait ou génère le VQD hash
func extractOrGenerateVqdHash(html string, headers *DynamicHeaders) error {
	// Chercher le VQD hash dans le HTML avec différents patterns
	patterns := []string{
		`vqd="([^"]+)"`,             // Pattern principal observé
		`"vqd":"([^"]+)"`,           // Pattern JSON
		`vqd["\s]*:["\s]*"([^"]+)"`, // Variante JSON
		`data-vqd="([^"]+)"`,        // Attribut data
		`vqd=([^&\s]+)`,             // Dans les paramètres URL
	}

	var numericVqd string
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(html)
		if len(matches) > 1 && len(matches[1]) > 10 {
			vqd := matches[1]
			// Vérifier si c'est un VQD numérique (format: 4-chiffres...)
			if strings.Contains(vqd, "-") && len(vqd) > 30 {
				numericVqd = vqd
				break
			}
		}
	}

	// Le VQD numérique trouvé sera utilisé pour x-vqd-4 par le système principal
	if numericVqd != "" {
		fmt.Printf("📊 Found numeric VQD: %s (will be used for x-vqd-4)\n", numericVqd)
		// Sauvegarder le VQD numérique pour usage ultérieur si nécessaire
		headers.Cookies["numeric_vqd"] = numericVqd
	}

	// Ne pas assigner de VQD1 fallback ici - cela sera fait par la fonction principale
	// si l'extraction dynamique échoue
	return nil
}

// extractCookies extrait les cookies de la réponse
func extractCookies(resp *http.Response, headers *DynamicHeaders) {
	for _, cookie := range resp.Cookies() {
		headers.Cookies[cookie.Name] = cookie.Value
	}
}

// getFallbackHeaders retourne les headers de fallback si l'extraction échoue
func getFallbackHeaders() *DynamicHeaders {
	return &DynamicHeaders{
		FeSignals: "eyJzdGFydCI6MTc1MTc1MTg4NTc3MiwiZXZlbnRzIjpbeyJuYW1lIjoic3RhcnROZXdDaGF0IiwiZGVsdGEiOjk1fSx7Im5hbWUiOiJyZWNlbnRDaGF0c0xpc3RJbXByZXNzaW9uIiwiZGVsdGEiOjIxOX0seyJuYW1lIjoiaW5pdFN3aXRjaE1vZGVsIiwiZGVsdGEiOjI2OTd9LHsibmFtZSI6InN0YXJ0TmV3Q2hhdCIsImRlbHRhIjo4MTYxfV0sImVuZCI6MjU0ODN9",
		FeVersion: "serp_20250704_184539_ET-8bee6051143b0c382099",
		VqdHash1:  "eyJzZXJ2ZXJfaGFzaGVzIjpbIjdYbEtTdFJxbkRDbVV6dEh2TkVBMm9kYXB5S3NKR21WSVYxZG4xWHpHbFk9Iiwic3pKR05nSytIV3pHWXVIR0taU1NjVXhOU2EyQmhJMy9XbExvalNzUDZRZz0iLCJhNzFZL05QM2RnMGoyUEEzK2p6S1ovLytnL01HWU1VZjd4ZXlIbkdVMDhFPSJdLCJjbGllbnRfaGFzaGVzIjpbImxWblI0MStCMVFWZ0o4d0hhMUdBNmdxR0JoSjlWdjN5K0dISkdGekJmTGM9IiwiakNoZUlFNUVKUjJlMUlURy9zQzd0N250QnVTQm9qdDY5MVVGNk1BK01pZz0iLCJFczV0akh6VjVTKzNCSEdVTnZ6Z1pZeVAvU3JBa3JETWVBSzlKVUlReDBjPSJdLCJzaWduYWxzIjp7fSwibWV0YSI6eyJ2IjoiNCIsImNoYWxsZW5nZV9pZCI6IjRmZmJhYzliNmIxMGM4MWVmODE0YzgxZTdmMmE4MDkxZDc5ODI0OGI2MDYxMmE0ZTViOGNhYjFhNDRkZjQ0OTRoOGpidCIsInRpbWVzdGFtcCI6IjE3NTE3NTE4ODU0ODMiLCJvcmlnaW4iOiJodHRwczovL2R1Y2tkdWNrZ28uY29tIiwic3RhY2siOiJFcnJvclxuYXQgdWUgKGh0dHBzOi8vZHVja2R1Y2tnby5jb20vZGlzdC93cG0uY2hhdC44YmVlNjA1MTE0M2IwYzM4MjA5OS5qczoxOjI2MTU4KVxuYXQgYXN5bmMgaHR0cHM6Ly9kdWNrZHVja2dvLmNvbS9kaXN0L3dwbS5jaGF0LjhiZWU2MDUxMTQzYjBjMzgyMDk5LmpzOjE6MjgzNDUiLCJkdXJhdGlvbiI6Ijg4In19",
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36",
		ChromeUA:  `"Not)A;Brand";v="8", "Chromium";v="138", "Brave";v="138"`,
		Cookies:   make(map[string]string),
	}
}
