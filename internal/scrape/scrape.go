package scrape

import (
	"context"
	"regexp"
	"time"

	"github.com/chromedp/chromedp"
)

func WebContent(url string) (string, error) {
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
	return cleanContent(content), err
}

const extractionScript = `(function() {
	[...document.querySelectorAll('script, style')].forEach(e => e.remove());
	return document.body.innerText;
})()`

func cleanContent(text string) string {
	return regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
}
