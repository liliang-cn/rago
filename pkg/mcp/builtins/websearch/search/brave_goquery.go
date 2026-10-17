package search

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// Brave's result page, as served 2026-10-17: every organic hit is a
// `div.snippet[data-type="web"]`; the first `a[href]` inside it is the
// destination, `.title` is the title, and the description sits in
// `.generic-snippet .content` (older markup: `.snippet-description`). Other
// `.snippet` blocks on the page are clusters, news carousels and a divider,
// which is what the previous scraper was reading when it returned
// "Disclaimer | https://./help/finance-disclaimer" as the first result and
// breadcrumbs glued to titles for the rest.

const braveDefaultBaseURL = "https://search.brave.com"

type braveGoQueryEngine struct {
	baseURL string
	client  *http.Client
}

func NewBraveGoQueryEngine() SearchEngine {
	return &braveGoQueryEngine{
		baseURL: braveDefaultBaseURL,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (b *braveGoQueryEngine) Name() string {
	return "brave"
}

func (b *braveGoQueryEngine) Search(ctx context.Context, query string, maxResults int) ([]SearchResult, error) {
	if maxResults <= 0 {
		maxResults = 10
	}
	searchURL := fmt.Sprintf("%s/search?q=%s&source=web", strings.TrimRight(b.baseURL, "/"), url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Brave results: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("brave returned %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var results []SearchResult
	doc.Find(`.snippet[data-type="web"]`).EachWithBreak(func(_ int, s *goquery.Selection) bool {
		if len(results) >= maxResults {
			return false
		}
		link, _ := s.Find("a[href]").First().Attr("href")
		link = strings.TrimSpace(link)
		title := strings.TrimSpace(s.Find(".title").First().Text())
		if title == "" {
			title = strings.TrimSpace(s.Find("a[href]").First().AttrOr("title", ""))
		}
		snippet := strings.TrimSpace(s.Find(".generic-snippet .content").First().Text())
		if snippet == "" {
			snippet = strings.TrimSpace(s.Find(".snippet-description").First().Text())
		}
		if !strings.HasPrefix(link, "http") || title == "" {
			return true
		}
		results = append(results, SearchResult{
			Title:   title,
			URL:     link,
			Snippet: collapseSpaces(snippet),
			Engine:  b.Name(),
		})
		return true
	})
	return results, nil
}

// collapseSpaces folds the whitespace runs that fall out of nested markup.
func collapseSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
