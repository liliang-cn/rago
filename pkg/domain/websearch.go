package domain

import "strings"

type WebSearchMode string

const (
	WebSearchModeAuto   WebSearchMode = "auto"
	WebSearchModeNative WebSearchMode = "native"
	WebSearchModeMCP    WebSearchMode = "mcp"
	WebSearchModeOff    WebSearchMode = "off"
)

func NormalizeWebSearchMode(mode WebSearchMode) WebSearchMode {
	switch strings.ToLower(strings.TrimSpace(string(mode))) {
	case string(WebSearchModeAuto):
		return WebSearchModeAuto
	case string(WebSearchModeNative):
		return WebSearchModeNative
	case string(WebSearchModeOff):
		return WebSearchModeOff
	case "", string(WebSearchModeMCP):
		return WebSearchModeMCP
	default:
		return WebSearchModeMCP
	}
}

func UsesNativeWebSearch(mode WebSearchMode) bool {
	normalized := NormalizeWebSearchMode(mode)
	return normalized == WebSearchModeAuto || normalized == WebSearchModeNative
}

func NormalizeWebSearchContextSize(size string) string {
	switch strings.ToLower(strings.TrimSpace(size)) {
	case "low", "medium", "high":
		return strings.ToLower(strings.TrimSpace(size))
	default:
		return "medium"
	}
}
