package agent

import (
	"regexp"
	"strings"
)

var (
	toolCamelBoundaryRe = regexp.MustCompile(`([a-z0-9])([A-Z])`)
	toolNonWordRe       = regexp.MustCompile(`[^a-z0-9]+`)
)

func normalizeToolLookupName(name string) string {
	name = toolCamelBoundaryRe.ReplaceAllString(name, `${1}_${2}`)
	name = strings.ToLower(name)
	name = toolNonWordRe.ReplaceAllString(name, "_")
	name = strings.Trim(name, "_")
	name = strings.ReplaceAll(name, "__", "_")
	return name
}

func toolTokens(name string) []string {
	normalized := normalizeToolLookupName(name)
	if normalized == "" {
		return nil
	}
	return strings.Split(normalized, "_")
}

func resolveClosestToolName(name string, candidates []string) string {
	if name == "" || len(candidates) == 0 {
		return ""
	}

	normalizedTarget := normalizeToolLookupName(name)
	if normalizedTarget == "" {
		return ""
	}

	bestName := ""
	bestScore := 0
	targetTokens := toolTokens(name)

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if candidate == name {
			return candidate
		}

		normalizedCandidate := normalizeToolLookupName(candidate)
		if normalizedCandidate == normalizedTarget {
			return candidate
		}

		score := scoreToolNameMatch(targetTokens, toolTokens(candidate))
		if score > bestScore {
			bestScore = score
			bestName = candidate
		}
	}

	if bestScore < 4 {
		return ""
	}
	return bestName
}

func scoreToolNameMatch(target, candidate []string) int {
	if len(target) == 0 || len(candidate) == 0 {
		return 0
	}

	score := 0
	for i := 0; i < len(target) && i < len(candidate); i++ {
		if target[i] != candidate[i] {
			break
		}
		score += 3
	}

	candidateSet := make(map[string]struct{}, len(candidate))
	for _, token := range candidate {
		candidateSet[token] = struct{}{}
	}
	for _, token := range target {
		if _, ok := candidateSet[token]; ok {
			score++
		}
	}

	return score
}
