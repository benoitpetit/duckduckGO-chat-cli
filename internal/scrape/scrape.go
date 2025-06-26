package scrape

import (
    "context"
    "encoding/json"
    "fmt"
    "regexp"
    "strings"
    "time"
    "log"

    "github.com/chromedp/chromedp"
)

type WebContentResult struct {
    URL     string `json:"url"`
    Content string `json:"content"`
}

func WebContent(url string) (*WebContentResult, error) {
    ctx, cancel := chromedp.NewContext(context.Background())
    defer cancel()
    ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    var content string
    err := chromedp.Run(ctx,
        chromedp.Navigate(url),
        chromedp.Sleep(2*time.Second),
        chromedp.Evaluate(extractionScript, &content),
    )
    if err != nil {
        log.Printf("Error fetching content for URL %s: %v\n", url, err)
        return nil, err
    }

    cleaned := cleanContent(content)
    result := &WebContentResult{
        URL:     url,
        Content: cleaned,
    }
    return result, nil
}

func (wcr *WebContentResult) ToJSON() (string, error) {
    jsonData, err := json.MarshalIndent(wcr, "", "  ")
    if err != nil {
        return "", fmt.Errorf("error marshalling JSON: %v", err)
    }
    return string(jsonData), nil
}

const extractionScript = `(function() {
	[...document.querySelectorAll('script, style')].forEach(e => e.remove());
	return document.body.innerText;
})()`

func cleanContent(text string) string {
    cleaned := regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
    cleaned = strings.TrimSpace(cleaned)
    return cleaned
}
