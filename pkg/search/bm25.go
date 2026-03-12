package search

import (
	"math"
	"sort"
	"strings"
)

// BM25Config holds configuration for BM25 ranking.
type BM25Config struct {
	K1 float64
	B  float64
}

// Document is a generic text document for ranking.
type Document struct {
	ID   string
	Text string
}

// Result is one scored BM25 match.
type Result struct {
	ID    string
	Score float64
}

// DefaultBM25Config returns the default BM25 parameters.
func DefaultBM25Config() *BM25Config {
	return &BM25Config{
		K1: 1.5,
		B:  0.75,
	}
}

// ExpandKeywords expands a natural-language query into deduplicated BM25 terms.
func ExpandKeywords(query string) []string {
	seen := make(map[string]struct{})
	terms := make([]string, 0)

	for _, raw := range strings.Fields(strings.ToLower(query)) {
		raw = strings.TrimSpace(raw)
		if len(raw) <= 1 {
			continue
		}
		if _, ok := seen[raw]; !ok {
			seen[raw] = struct{}{}
			terms = append(terms, raw)
		}

		for _, normalized := range Tokenize(raw) {
			if _, ok := seen[normalized]; ok {
				continue
			}
			seen[normalized] = struct{}{}
			terms = append(terms, normalized)
		}
	}

	for _, term := range Tokenize(query) {
		if _, ok := seen[term]; !ok {
			seen[term] = struct{}{}
			terms = append(terms, term)
		}

		for _, part := range strings.FieldsFunc(term, func(r rune) bool {
			return r == '_' || r == '-' || r == '/'
		}) {
			if len(part) <= 1 {
				continue
			}
			if _, ok := seen[part]; ok {
				continue
			}
			seen[part] = struct{}{}
			terms = append(terms, part)
		}
	}

	return terms
}

// Tokenize splits text into normalized searchable terms.
func Tokenize(text string) []string {
	text = strings.ToLower(text)
	replacer := strings.NewReplacer(
		".", " ",
		",", " ",
		"!", " ",
		"?", " ",
		";", " ",
		":", " ",
		"(", " ",
		")", " ",
		"[", " ",
		"]", " ",
		"{", " ",
		"}", " ",
		"\"", " ",
		"'", " ",
		"-", " ",
		"_", " ",
		"/", " ",
		"\\", " ",
		"\n", " ",
		"\t", " ",
	)
	text = replacer.Replace(text)

	words := strings.Fields(text)
	terms := make([]string, 0, len(words))
	for _, word := range words {
		if len(word) > 1 {
			terms = append(terms, word)
		}
	}
	return terms
}

// Rank ranks generic documents by BM25 score for the given query.
func Rank(query string, documents []Document, topK int, config *BM25Config) []Result {
	if config == nil {
		config = DefaultBM25Config()
	}
	if len(documents) == 0 {
		return nil
	}

	queryTerms := ExpandKeywords(query)
	if len(queryTerms) == 0 {
		return nil
	}

	totalLength := 0
	docTerms := make([][]string, 0, len(documents))
	for _, doc := range documents {
		terms := Tokenize(doc.Text)
		docTerms = append(docTerms, terms)
		totalLength += len(terms)
	}
	if totalLength == 0 {
		return nil
	}

	avgDocLength := float64(totalLength) / float64(len(documents))
	docFreq := make(map[string]int)
	for _, terms := range docTerms {
		seen := make(map[string]bool, len(terms))
		for _, term := range terms {
			if !seen[term] {
				docFreq[term]++
				seen[term] = true
			}
		}
	}

	results := make([]Result, 0, len(documents))
	for idx, doc := range documents {
		score := calculateBM25(docTerms[idx], queryTerms, docFreq, len(documents), avgDocLength, config)
		if score <= 0 {
			continue
		}
		results = append(results, Result{
			ID:    doc.ID,
			Score: score,
		})
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].ID < results[j].ID
		}
		return results[i].Score > results[j].Score
	})

	if topK > 0 && len(results) > topK {
		return results[:topK]
	}
	return results
}

func calculateBM25(docTerms []string, queryTerms []string, docFreq map[string]int, totalDocs int, avgDocLength float64, config *BM25Config) float64 {
	docLength := float64(len(docTerms))
	termFreq := make(map[string]int, len(docTerms))
	for _, term := range docTerms {
		termFreq[term]++
	}

	var score float64
	for _, queryTerm := range queryTerms {
		tf := float64(termFreq[queryTerm])
		if tf == 0 {
			continue
		}

		df := float64(docFreq[queryTerm])
		if df == 0 {
			continue
		}

		idf := math.Log((float64(totalDocs)-df+0.5)/(df+0.5) + 1)
		numerator := tf * (config.K1 + 1)
		denominator := tf + config.K1*(1-config.B+config.B*(docLength/avgDocLength))
		score += idf * (numerator / denominator)
	}

	return score
}
