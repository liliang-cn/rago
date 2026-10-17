package search

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Bing is read through its RSS output (`format=rss`), not its HTML. The HTML
// result page is ~130 KB of markup whose class names move, and every result
// link in it is a bing.com/ck redirect; the RSS is ~5 KB, has carried the
// same <item><title><link><description> shape for years, and links straight
// to the page. Measured 2026-10-17: the goquery scraper over the HTML returned
// zero results and no error.
//
// Bing also rate-limits by IP. Past the limit it answers 200 with a Turnstile
// challenge page ("One last step / Please solve the challenge below") instead
// of results, for the RSS URL as well. That is reported as an error, so the
// searcher falls through to the next engine instead of returning nothing.
//
// What cannot be fixed here: to traffic it does not take for a browser, Bing
// answers long-tail queries with decoy results rather than a block. Measured
// 2026-10-17 from three different egress IPs, HTML and RSS alike: "github"
// and "python requests library documentation" came back right, while
// "PuerkitoBio goquery github" returned FedEx tracking pages and "linstor
// drbd quorum" returned Discord. Nothing in the response marks them as
// decoys. That is why bing sits last in enginePriority: it is only reached
// when duckduckgo and brave have both failed, and a popular query is the
// kind it still answers.

const bingDefaultBaseURL = "https://www.bing.com"

type bingGoQueryEngine struct {
	baseURL string
	client  *http.Client
}

func NewBingGoQueryEngine() SearchEngine {
	return &bingGoQueryEngine{
		baseURL: bingDefaultBaseURL,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (b *bingGoQueryEngine) Name() string {
	return "bing"
}

type bingRSS struct {
	Items []struct {
		Title       string `xml:"title"`
		Link        string `xml:"link"`
		Description string `xml:"description"`
	} `xml:"channel>item"`
}

func (b *bingGoQueryEngine) Search(ctx context.Context, query string, maxResults int) ([]SearchResult, error) {
	if maxResults <= 0 {
		maxResults = 10
	}
	searchURL := fmt.Sprintf("%s/search?q=%s&format=rss", strings.TrimRight(b.baseURL, "/"), url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/rss+xml,text/xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Bing results: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bing returned %d", resp.StatusCode)
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, fmt.Errorf("failed to read Bing results: %w", err)
	}
	if isBingChallenge(resp.Header.Get("Content-Type"), raw) {
		return nil, fmt.Errorf("bing served a bot challenge instead of results (rate-limited)")
	}

	var feed bingRSS
	if err := xml.Unmarshal(raw, &feed); err != nil {
		return nil, fmt.Errorf("failed to parse Bing RSS: %w", err)
	}

	results := make([]SearchResult, 0, min(len(feed.Items), maxResults))
	for _, it := range feed.Items {
		if len(results) >= maxResults {
			break
		}
		title := strings.TrimSpace(it.Title)
		link := strings.TrimSpace(it.Link)
		if title == "" || !strings.HasPrefix(link, "http") {
			continue
		}
		results = append(results, SearchResult{
			Title:   title,
			URL:     link,
			Snippet: strings.TrimSpace(it.Description),
			Engine:  b.Name(),
		})
	}
	return results, nil
}

// isBingChallenge recognises the Turnstile page Bing serves past its rate
// limit. It comes back as text/html with a 200 where RSS was asked for, and
// its visible text is fixed.
func isBingChallenge(contentType string, body []byte) bool {
	if strings.Contains(strings.ToLower(contentType), "text/html") {
		return true
	}
	s := string(body)
	return strings.Contains(s, "solve the challenge") || strings.Contains(s, `class="captcha"`) || strings.Contains(s, "turnstile-widget")
}
