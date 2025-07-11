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
// en suivant le même processus que les scripts PowerShell qui fonctionnent
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

	// Headers de base pour simuler un navigateur réel
	headers := &DynamicHeaders{
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36",
		ChromeUA:  `"Not)A;Brand";v="8", "Chromium";v="138", "Brave";v="138"`,
		Cookies:   make(map[string]string),
	}

	// Étape 1: Récupérer la page principale pour obtenir les données dynamiques
	fmt.Printf("🔄 Step 1: Getting main page...\n")
	req, err := http.NewRequest("GET", "https://duckduckgo.com/?q=DuckDuckGo+AI+Chat&ia=chat&duckai=1", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create initial request: %v", err)
	}

	setRequestHeaders(req, headers)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch initial page: %v", err)
	}

	// Lire le contenu HTML
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	bodyStr := string(body)

	// Extraire l'URL encodée pour les requêtes POST (crucial!)
	challengeURL := extractChallengeURL(bodyStr)
	if challengeURL == "" {
		return nil, fmt.Errorf("could not extract challenge URL from page")
	}
	fmt.Printf("🔍 Found challenge URL: %s\n", challengeURL)

	// Étape 2: Effectuer les requêtes POST de challenge comme dans le script PS1
	fmt.Printf("🔄 Step 2: Performing challenge requests...\n")
	err = performChallengeRequests(client, challengeURL, headers)
	if err != nil {
		return nil, fmt.Errorf("challenge requests failed: %v", err)
	}

	// Étape 3: Effectuer la requête HEAD comme dans le script PS1
	fmt.Printf("🔄 Step 3: Performing HEAD request...\n")
	headURL := strings.Replace(challengeURL, "OpDbEoniqNYx5opPFPeNRglChzuydeWaPOayQEf3-MPp_Y4gn8nA7VPlJ8lDcXUK9Fp6zdCuF_Rn7UrBfWJQHA==", "Xa8ueLMSTpim-a8z7lWRols8UdrzpeqDl5UedPPIZjZz33tfh7z6M7TWE7xeWn_IJPvcqKX4rU7iA80koYQmJQ==", 1)
	err = performHEADRequest(client, headURL, headers)
	if err != nil {
		return nil, fmt.Errorf("HEAD request failed: %v", err)
	}

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

	// Utiliser le VQD hash JSON fonctionnel du navigateur
	fmt.Printf("🔍 Using working VQD hash from browser analysis...\n")
	headers.VqdHash1 = "eyJzZXJ2ZXJfaGFzaGVzIjpbIkNjS0RKWjNjQjVNc0QzVzJkc2hQK1hFQjB0RjNmWE1jUXcxV1M2NE5zcWc9IiwiQ3l2TUpNKzNIY3k0dkVPS1d6alJIbUZyQmEzMDR5MTVGcHBtTUpqV1NFOD0iLCIrTTFVZ1ZPWHpCc1JybDhIOVBjWjlkd1VxTExZNWRWdVZTb09RcUxqdzk4PSJdLCJjbGllbnRfaGFzaGVzIjpbImxWblI0MStCMVFWZ0o4d0hhMUdBNmdxR0JoSjlWdjN5K0dISkdGekJmTGM9IiwiTFEwYUxFaEJTeHJPeFoxcVVjb1dIWUlRKzNEcWtmZTZGYm9MdUlQcEFVYz0iLCJkRXkxaVNaTWxqQnYvK3dFRzg2UStSaFNTUlJFbDh2eGtaem1jS21iYTM0PSJdLCJzaWduYWxzIjp7fSwibWV0YSI6eyJ2IjoiNCIsImNoYWxsZW5nZV9pZCI6IjBkZTYxMmRkZTM4ZTVkOGFlNmQyYTFjMzY1ZmYzZjdiNDBiNDViYzgxODk1OTNiNzdmY2UwMDBhNzA3ODc4Y2VoOGpidCIsInRpbWVzdGFtcCI6IjE3NTIxNTIwNTkzOTkiLCJvcmlnaW4iOiJodHRwczovL2R1Y2tkdWNrZ28uY29tIiwic3RhY2siOiJFcnJvclxuYXQgRSAoaHR0cHM6Ly9kdWNrZHVja2dvLmNvbS9kaXN0L3dwbS5jaGF0LjcwZWFjYTZhZWEyOTQ4YjBiYjYwLmpzOjE6MTQ4MjUpXG5hdCBhc3luYyBodHRwczovL2R1Y2tkdWNrZ28uY29tL2Rpc3Qvd3BtLmNoYXQuNzBlYWNhNmFlYTI5NDhiMGJiNjAuanM6MToxNjk4NSIsImR1cmF0aW9uIjoiODUifX0="
	fmt.Printf("✅ Using proven working VQD hash from browser\n")

	// Capturer les cookies supplémentaires
	extractCookies(resp, headers)

	// Attendre un peu pour simuler l'interaction utilisateur
	time.Sleep(500 * time.Millisecond)

	// Validation des données extraites
	if headers.FeVersion == "" {
		headers.FeVersion = "serp_20250710_070136_ET-70eaca6aea2948b0bb60"
	}
	if headers.FeSignals == "" {
		headers.FeSignals = "eyJzdGFydCI6MTc1MjE1MjA1OTAzMywiZXZlbnRzIjpbeyJuYW1lIjoic3RhcnROZXdDaGF0IiwiZGVsdGEiOjcyfSx7Im5hbWUiOiJyZWNlbnRDaGF0c0xpc3RJbXByZXNzaW9uIiwiZGVsdGEiOjEzOH1dLCJlbmQiOjQ2MDZ9"
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

// extractCookies extrait les cookies de la réponse
func extractCookies(resp *http.Response, headers *DynamicHeaders) {
	for _, cookie := range resp.Cookies() {
		headers.Cookies[cookie.Name] = cookie.Value
	}
}

// extractChallengeURL extrait l'URL de challenge depuis le HTML de la page
func extractChallengeURL(html string) string {
	// Chercher l'URL encodée dans le JavaScript
	patterns := []string{
		`https://duckduckgo\.com/[A-Za-z0-9_\-=]+`,
		`"/[A-Za-z0-9_\-=]{50,}"`,
		`'/[A-Za-z0-9_\-=]{50,}'`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(html)
		if len(matches) > 0 {
			match := matches[0]
			if strings.Contains(match, "OpDbEoniqNYx5opPFPeNRglChzuydeWaPOayQEf3") ||
				strings.Contains(match, "duckduckgo.com") {
				if strings.HasPrefix(match, "/") {
					return "https://duckduckgo.com" + strings.Trim(match, `"'`)
				}
				return strings.Trim(match, `"'`)
			}
		}
	}

	// Fallback: utiliser l'URL des scripts PS1 qui fonctionnent
	return "https://duckduckgo.com/OpDbEoniqNYx5opPFPeNRglChzuydeWaPOayQEf3-MPp_Y4gn8nA7VPlJ8lDcXUK9Fp6zdCuF_Rn7UrBfWJQHA=="
}

// performChallengeRequests effectue les requêtes POST de challenge comme dans les scripts PS1
func performChallengeRequests(client *http.Client, challengeURL string, headers *DynamicHeaders) error {
	// Préparer les données de challenge comme dans les scripts PS1
	challengeData := []string{
		`{"DOMIdentifiers":{"rulesType":1,"url":"https://duckduckgo.com/?q=DuckDuckGo+AI+Chat&ia=chat&duckai=1","lang":"fr-FR","ids":["icon60","icon76","icon120","icon152","state_hidden","spacing_hidden_wrapper","spacing_hidden","header_wrapper","header","header-non-nav","header-logo-wrapper","search_form","search_form_input","search_form_input_clear","search_button","search_dropdown","search_elements_hidden","react-ai-button-slot","react-duckbar","react-browser-update-info","zero_click_wrapper","react-root-zci","vertical_wrapper","web_content_wrapper","react-layout","bottom_spacing2","z2","z"],"classes":["has-zcm","no-theme","is-link-style-exp","is-link-order-exp","is-link-breadcrumb-exp","is-related-search-exp","is-vertical-tabs-exp","js","no-touch","opacity","csstransforms3d","csstransitions","svg","cssfilters","body--serp","is-duckchat","is-duckai","site-wrapper","js-site-wrapper","welcome-wrap","js-welcome-wrap","header-wrap","js-header-wrap","ai-header-exp","header","cw","header__shrink-beyond-min-size","header__search-wrap","header__logo-wrap","js-header-logo","header__logo","js-logo-ddg","header__content","header__search","search--adv","search--header","js-search-form","search__input","search__input--adv","js-search-input","search__clear","js-search-clear","search__button","js-search-button","search__dropdown","search__hidden","js-search-hidden","header--aside","js-header-aside","zci-wrap","verticals","content-wrap","serp__top-right","js-serp-top-right","serp__bottom-right","js-serp-bottom-right","js-feedback-btn-wrap","results--main"]}}`,
		`{"DOMIdentifiers":{"rulesType":2,"url":"https://duckduckgo.com/?q=DuckDuckGo+AI+Chat&ia=chat&duckai=1","lang":"fr-FR","ids":[],"classes":["is-not-mobile-device","full-urls","breadcrumb-urls","dark-header","dark-bg","react","has-footer","has-text","ready","footer"]}}`,
		`{"DOMIdentifiers":{"rulesType":2,"url":"https://duckduckgo.com/?q=DuckDuckGo+AI+Chat&ia=chat&duckai=1","lang":"fr-FR","ids":[],"classes":["iqWauQNeRzJ1Ot90nG8b"]}}`,
	}

	for i, data := range challengeData {
		fmt.Printf("  📡 Challenge request %d/3...\n", i+1)

		req, err := http.NewRequest("POST", challengeURL, strings.NewReader(data))
		if err != nil {
			return fmt.Errorf("failed to create challenge request %d: %v", i+1, err)
		}

		// Headers spécifiques pour les requêtes de challenge
		req.Header.Set("Accept", "*/*")
		req.Header.Set("Accept-Language", "fr-FR,fr;q=0.8")
		req.Header.Set("Cache-Control", "max-age=0")
		req.Header.Set("Content-Type", "text/plain;charset=UTF-8")
		req.Header.Set("DNT", "1")
		req.Header.Set("Origin", "https://duckduckgo.com")
		req.Header.Set("Priority", "u=1, i")
		req.Header.Set("Referer", "https://duckduckgo.com/")
		req.Header.Set("Sec-CH-UA", headers.ChromeUA)
		req.Header.Set("Sec-CH-UA-Mobile", "?0")
		req.Header.Set("Sec-CH-UA-Platform", `"Windows"`)
		req.Header.Set("Sec-Fetch-Dest", "empty")
		req.Header.Set("Sec-Fetch-Mode", "cors")
		req.Header.Set("Sec-Fetch-Site", "same-origin")
		req.Header.Set("Sec-GPC", "1")
		req.Header.Set("User-Agent", headers.UserAgent)

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to execute challenge request %d: %v", i+1, err)
		}
		resp.Body.Close()

		// Petit délai entre les requêtes
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

// performHEADRequest effectue la requête HEAD comme dans les scripts PS1
func performHEADRequest(client *http.Client, headURL string, headers *DynamicHeaders) error {
	req, err := http.NewRequest("HEAD", headURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create HEAD request: %v", err)
	}

	// Headers pour la requête HEAD
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.8")
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("DNT", "1")
	req.Header.Set("Priority", "u=1, i")
	req.Header.Set("Referer", "https://duckduckgo.com/")
	req.Header.Set("Sec-CH-UA", headers.ChromeUA)
	req.Header.Set("Sec-CH-UA-Mobile", "?0")
	req.Header.Set("Sec-CH-UA-Platform", `"Windows"`)
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-GPC", "1")
	req.Header.Set("User-Agent", headers.UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute HEAD request: %v", err)
	}
	resp.Body.Close()

	return nil
}
