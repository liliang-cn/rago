package search

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Pages saved from the live engines on 2026-10-17. A selector test against a
// real page is the only kind that can tell a scraper has stopped matching:
// the mock-engine tests cannot, and the previous selectors were green while
// returning nothing.

func serveFixture(t *testing.T, name, contentType string) *httptest.Server {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("q") == "" {
			t.Errorf("no q parameter in %s", r.URL)
		}
		w.Header().Set("Content-Type", contentType)
		_, _ = w.Write(body)
	}))
}

func TestBingParsesRSS(t *testing.T) {
	srv := serveFixture(t, "bing_results.rss", "text/xml; charset=utf-8")
	defer srv.Close()
	eng := &bingGoQueryEngine{baseURL: srv.URL, client: srv.Client()}

	got, err := eng.Search(context.Background(), "golang goquery", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("want 3 results, got %d", len(got))
	}
	first := got[0]
	if first.Title != "The Go Programming Language" || first.URL != "https://go.dev/" {
		t.Fatalf("first result parsed wrong: %+v", first)
	}
	if !strings.Contains(first.Snippet, "open source programming language") {
		t.Fatalf("snippet missing: %q", first.Snippet)
	}
	for _, r := range got {
		if strings.Contains(r.URL, "bing.com/ck/") {
			t.Fatalf("redirect URL leaked: %s", r.URL)
		}
		if r.Engine != "bing" {
			t.Fatalf("engine label %q", r.Engine)
		}
	}
}

func TestBingChallengePageIsAnError(t *testing.T) {
	challenge := `<html><body><div class="captcha"><div class="captcha_header">One last step</div>` +
		`<div class="captcha_text">Please solve the challenge below to continue</div><div id="turnstile-widget"></div></div></body></html>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(challenge))
	}))
	defer srv.Close()
	eng := &bingGoQueryEngine{baseURL: srv.URL, client: srv.Client()}

	got, err := eng.Search(context.Background(), "anything", 5)
	if err == nil {
		t.Fatalf("challenge page must be an error, got %d results and nil", len(got))
	}
	if !strings.Contains(err.Error(), "challenge") {
		t.Fatalf("error should say what happened: %v", err)
	}
}

func TestBraveParsesWebResultsOnly(t *testing.T) {
	srv := serveFixture(t, "brave_results.html", "text/html; charset=utf-8")
	defer srv.Close()
	eng := &braveGoQueryEngine{baseURL: srv.URL, client: srv.Client()}

	got, err := eng.Search(context.Background(), "NVIDIA stock price today", 10)
	if err != nil {
		t.Fatal(err)
	}
	// The fixture holds three web results and one cluster block; the cluster
	// must not become a result.
	if len(got) != 3 {
		t.Fatalf("want 3 results, got %d: %+v", len(got), got)
	}
	first := got[0]
	if first.URL != "https://finance.yahoo.com/quote/NVDA/" {
		t.Fatalf("first URL: %q", first.URL)
	}
	if first.Title != "NVIDIA Corporation (NVDA) Stock Price, News, Quote & History - Yahoo Finance" {
		t.Fatalf("first title carries site name or breadcrumbs: %q", first.Title)
	}
	if !strings.Contains(first.Snippet, "Compute & Networking segment") {
		t.Fatalf("first snippet: %q", first.Snippet)
	}
	for _, r := range got {
		if strings.Contains(r.Title, "›") || strings.Contains(r.URL, "https://./") {
			t.Fatalf("old breadcrumb bug is back: %+v", r)
		}
	}
}

func TestBraveMaxResultsCapsOutput(t *testing.T) {
	srv := serveFixture(t, "brave_results.html", "text/html; charset=utf-8")
	defer srv.Close()
	eng := &braveGoQueryEngine{baseURL: srv.URL, client: srv.Client()}
	got, err := eng.Search(context.Background(), "q", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2, got %d", len(got))
	}
}
