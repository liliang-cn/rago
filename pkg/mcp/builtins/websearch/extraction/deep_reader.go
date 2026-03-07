package extraction

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
)

// LinkInfo represents a clickable element on a page
type LinkInfo struct {
	URL  string `json:"url"`
	Text string `json:"text"`
	Type string `json:"type"` // "link" or "button"
}

// SubPageResult represents content from a crawled sub-page
type SubPageResult struct {
	URL      string `json:"url"`
	Title    string `json:"title"`
	Content  string `json:"content"`
	LinkText string `json:"link_text"`
	Error    string `json:"error,omitempty"`
}

// DeepReadResult represents the complete deep read output
type DeepReadResult struct {
	MainURL      string          `json:"main_url"`
	MainTitle    string          `json:"main_title"`
	MainContent  string          `json:"main_content"`
	SubPages     []SubPageResult `json:"sub_pages"`
	TotalLinks   int             `json:"total_links"`
	CrawledLinks int             `json:"crawled_links"`
}

// DeepReader provides deep web page reading capabilities
type DeepReader struct {
	timeout      time.Duration
	maxLinks     int
	sameDomain   bool
	contentLimit int
	concurrency  int
}

// DeepReaderOption configures the DeepReader
type DeepReaderOption func(*DeepReader)

// WithMaxLinks sets the maximum number of links to crawl
func WithMaxLinks(n int) DeepReaderOption {
	return func(d *DeepReader) {
		if n > 0 && n <= 20 {
			d.maxLinks = n
		}
	}
}

// WithSameDomain sets whether to restrict crawling to same domain
func WithSameDomain(same bool) DeepReaderOption {
	return func(d *DeepReader) {
		d.sameDomain = same
	}
}

// WithContentLimit sets the maximum content length per page
func WithContentLimit(limit int) DeepReaderOption {
	return func(d *DeepReader) {
		if limit > 0 {
			d.contentLimit = limit
		}
	}
}

// WithTimeout sets the timeout for page operations
func WithTimeout(t time.Duration) DeepReaderOption {
	return func(d *DeepReader) {
		if t > 0 {
			d.timeout = t
		}
	}
}

// NewDeepReader creates a new DeepReader with default options
func NewDeepReader(opts ...DeepReaderOption) *DeepReader {
	d := &DeepReader{
		timeout:      60 * time.Second,
		maxLinks:     10,
		sameDomain:   true,
		contentLimit: 2000,
		concurrency:  3,
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// DeepRead performs deep reading of a webpage and its related pages
func (d *DeepReader) DeepRead(ctx context.Context, targetURL string) (*DeepReadResult, error) {
	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	allocCtx, cancel := chromedp.NewContext(ctx)
	defer cancel()

	var mainContent string
	var mainTitle string
	var linksJSON string

	// Extract main page content and links
	err := chromedp.Run(allocCtx,
		chromedp.Navigate(targetURL),
		chromedp.WaitReady("body"),
		chromedp.Title(&mainTitle),
		chromedp.Evaluate(`
			(function() {
				// Remove script and style elements
				var scripts = document.querySelectorAll('script, style, noscript');
				scripts.forEach(function(el) { el.remove(); });

				// Get main content
				var mainEl = document.querySelector('main, article, .content, #content, .post, .entry-content');
				var content = mainEl ? mainEl.innerText : document.body.innerText;

				// Get links
				var links = Array.from(document.querySelectorAll('a[href]')).map(function(el) {
					return {
						url: el.href,
						text: (el.innerText || el.getAttribute('aria-label') || '').trim().slice(0, 100),
						type: 'link'
					};
				}).filter(function(l) { return l.url && l.text; });

				return JSON.stringify({ content: content, links: links });
			})()
		`, &linksJSON),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to read main page %s: %w", targetURL, err)
	}

	mainContent = d.parseContentFromJSON(linksJSON)
	mainContent = CleanText(mainContent)
	if len(mainContent) > d.contentLimit {
		mainContent = mainContent[:d.contentLimit] + "..."
	}

	// Parse and filter links
	allLinks := d.parseLinksFromJSON(linksJSON)
	filteredLinks := d.filterLinks(targetURL, allLinks)

	result := &DeepReadResult{
		MainURL:     targetURL,
		MainTitle:   mainTitle,
		MainContent: mainContent,
		TotalLinks:  len(allLinks),
	}

	// Crawl sub-pages with concurrency control
	if len(filteredLinks) > 0 {
		subPages := d.crawlSubPages(ctx, filteredLinks)
		result.SubPages = subPages
		result.CrawledLinks = len(subPages)
	}

	return result, nil
}

// parseContentFromJSON extracts content from the JSON response
func (d *DeepReader) parseContentFromJSON(jsonStr string) string {
	// Simple extraction - find content field
	idx := strings.Index(jsonStr, `"content":"`)
	if idx == -1 {
		return ""
	}
	start := idx + len(`"content":"`)
	end := strings.Index(jsonStr[start:], `","links"`)
	if end == -1 {
		return ""
	}
	return jsonStr[start : start+end]
}

// parseLinksFromJSON extracts links from the JSON response
func (d *DeepReader) parseLinksFromJSON(jsonStr string) []LinkInfo {
	var links []LinkInfo

	// Find the links array
	idx := strings.Index(jsonStr, `"links":[`)
	if idx == -1 {
		return links
	}

	// Simple JSON parsing for link objects
	linkPattern := regexp.MustCompile(`\{"url":"([^"]+)","text":"([^"]+)","type":"([^"]+)"\}`)
	matches := linkPattern.FindAllStringSubmatch(jsonStr, -1)

	for _, match := range matches {
		if len(match) >= 4 {
			links = append(links, LinkInfo{
				URL:  match[1],
				Text: match[2],
				Type: match[3],
			})
		}
	}

	return links
}

// filterLinks applies smart filtering to select relevant links
func (d *DeepReader) filterLinks(baseURL string, links []LinkInfo) []LinkInfo {
	baseParsed, err := url.Parse(baseURL)
	if err != nil {
		return links[:min(d.maxLinks, len(links))]
	}

	var filtered []LinkInfo
	seen := make(map[string]bool)

	// Patterns to exclude
	excludePatterns := []string{
		"login", "signin", "sign-in", "signup", "sign-up", "register",
		"logout", "log-out", "subscribe", "unsubscribe",
		"privacy", "terms", "legal", "cookie",
		"contact", "about", "help", "support", "faq",
		"sitemap", "rss", "feed", "xml",
		"facebook.com", "twitter.com", "linkedin.com", "instagram.com",
		"youtube.com", "github.com", "medium.com",
	}

	excludeExtensions := []string{
		".pdf", ".zip", ".doc", ".docx", ".xls", ".xlsx",
		".ppt", ".pptx", ".mp3", ".mp4", ".avi", ".mov",
		".jpg", ".jpeg", ".png", ".gif", ".svg", ".webp",
	}

	for _, link := range links {
		linkURL := link.URL
		linkLower := strings.ToLower(linkURL)
		textLower := strings.ToLower(link.Text)

		// Skip empty or invalid URLs
		if linkURL == "" || linkURL == "#" ||
			strings.HasPrefix(linkURL, "javascript:") ||
			strings.HasPrefix(linkURL, "mailto:") ||
			strings.HasPrefix(linkURL, "tel:") {
			continue
		}

		// Skip already seen URLs
		if seen[linkURL] {
			continue
		}

		// Skip same page anchors
		if strings.Contains(linkURL, "#") {
			continueURL := linkURL
			if idx := strings.Index(continueURL, "#"); idx > 0 {
				continueURL = continueURL[:idx]
			}
			if continueURL == baseURL || continueURL == "" {
				continue
			}
		}

		// Check same domain restriction
		if d.sameDomain {
			linkParsed, err := url.Parse(linkURL)
			if err != nil || linkParsed.Host != baseParsed.Host {
				continue
			}
		}

		// Skip excluded patterns in URL
		skip := false
		for _, pattern := range excludePatterns {
			if strings.Contains(linkLower, pattern) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		// Skip excluded file extensions
		for _, ext := range excludeExtensions {
			if strings.HasSuffix(linkLower, ext) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		// Skip short or generic link texts
		if len(link.Text) < 3 ||
			textLower == "click here" ||
			textLower == "read more" ||
			textLower == "more" ||
			textLower == "learn more" ||
			textLower == "here" ||
			textLower == "link" {
			continue
		}

		seen[linkURL] = true
		filtered = append(filtered, link)
	}

	// Sort by text length (longer anchor text usually means more important)
	sort.Slice(filtered, func(i, j int) bool {
		return len(filtered[i].Text) > len(filtered[j].Text)
	})

	// Limit to maxLinks
	if len(filtered) > d.maxLinks {
		filtered = filtered[:d.maxLinks]
	}

	return filtered
}

// crawlSubPages crawls multiple sub-pages concurrently
func (d *DeepReader) crawlSubPages(ctx context.Context, links []LinkInfo) []SubPageResult {
	var wg sync.WaitGroup
	results := make([]SubPageResult, len(links))
	sem := make(chan struct{}, d.concurrency)

	extractor := NewHybridExtractor()

	for i, link := range links {
		wg.Add(1)
		go func(idx int, link LinkInfo) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			subCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			defer cancel()

			content, err := extractor.ExtractSummary(subCtx, link.URL, d.contentLimit)
			if err != nil {
				results[idx] = SubPageResult{
					URL:      link.URL,
					LinkText: link.Text,
					Error:    err.Error(),
				}
				return
			}

			// Extract title from content
			title := ""
			if strings.HasPrefix(content, "# ") {
				lines := strings.SplitN(content, "\n", 2)
				title = strings.TrimPrefix(lines[0], "# ")
			}

			results[idx] = SubPageResult{
				URL:      link.URL,
				Title:    title,
				Content:  content,
				LinkText: link.Text,
			}
		}(i, link)
	}

	wg.Wait()

	// Filter out empty results
	var validResults []SubPageResult
	for _, r := range results {
		if r.URL != "" {
			validResults = append(validResults, r)
		}
	}

	return validResults
}

// ToMarkdown formats the deep read result as markdown
func (r *DeepReadResult) ToMarkdown() string {
	var sb strings.Builder

	// Main page
	sb.WriteString(fmt.Sprintf("# [%s](%s)\n\n", r.MainTitle, r.MainURL))
	sb.WriteString(r.MainContent)
	sb.WriteString("\n\n---\n\n")

	// Sub pages
	if len(r.SubPages) > 0 {
		sb.WriteString("## Related Pages\n\n")
		for i, page := range r.SubPages {
			sb.WriteString(fmt.Sprintf("### %d. [%s](%s)\n", i+1, page.LinkText, page.URL))
			if page.Error != "" {
				sb.WriteString(fmt.Sprintf("*Error: %s*\n\n", page.Error))
			} else {
				if page.Title != "" && page.Title != page.LinkText {
					sb.WriteString(fmt.Sprintf("> %s\n\n", page.Title))
				}
				// Add content summary
				content := page.Content
				if len(content) > 1500 {
					content = content[:1500] + "..."
				}
				sb.WriteString(content)
				sb.WriteString("\n\n---\n\n")
			}
		}
	}

	// Summary stats
	sb.WriteString(fmt.Sprintf("*Crawled %d of %d total links*\n", r.CrawledLinks, r.TotalLinks))

	return sb.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
