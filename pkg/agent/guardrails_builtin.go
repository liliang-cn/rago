package agent

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// ContentModerationGuardrail checks for potentially harmful content
func ContentModerationGuardrail() *Guardrail {
	return NewGuardrail(
		"content_moderation",
		GuardrailKindBoth,
		func(ctx context.Context, content string, kind GuardrailKind) (*GuardrailResult, error) {
			// Define patterns for potentially harmful content
			harmfulPatterns := []struct {
				name    string
				pattern string
			}{
				{"violence", `\b(kill|murder|torture|assault|attack)\b`},
				{"hate_speech", `\b(hate|discriminat|racist|sexist)\b`},
				{"self_harm", `\b(suicide|self.harm|hurt.myself)\b`},
			}

			lowerContent := strings.ToLower(content)

			for _, hp := range harmfulPatterns {
				matched, _ := regexp.MatchString(hp.pattern, lowerContent)
				if matched {
					return &GuardrailResult{
						Passed: false,
						Reason: fmt.Sprintf("content contains potentially harmful material (category: %s)", hp.name),
					}, nil
				}
			}

			return &GuardrailResult{Passed: true}, nil
		},
		WithGuardrailDescription("Checks for harmful or inappropriate content"),
	)
}

// PIIDetectionGuardrail detects potential PII (Personally Identifiable Information)
func PIIDetectionGuardrail(redact bool) *Guardrail {
	return NewGuardrail(
		"pii_detection",
		GuardrailKindBoth,
		func(ctx context.Context, content string, kind GuardrailKind) (*GuardrailResult, error) {
			// Email pattern
			emailRe := regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`)
			// Phone pattern (basic)
			phoneRe := regexp.MustCompile(`\b\d{3}[-.\s]?\d{3}[-.\s]?\d{4}\b`)
			// Credit card pattern (basic)
			ccRe := regexp.MustCompile(`\b\d{4}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b`)
			// SSN pattern
			ssnRe := regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)

			found := []string{}
			if emailRe.MatchString(content) {
				found = append(found, "email")
			}
			if phoneRe.MatchString(content) {
				found = append(found, "phone")
			}
			if ccRe.MatchString(content) {
				found = append(found, "credit card")
			}
			if ssnRe.MatchString(content) {
				found = append(found, "SSN")
			}

			if len(found) > 0 {
				if redact {
					redacted := content
					redacted = emailRe.ReplaceAllString(redacted, "[REDACTED]")
					redacted = phoneRe.ReplaceAllString(redacted, "[REDACTED]")
					redacted = ccRe.ReplaceAllString(redacted, "[REDACTED]")
					redacted = ssnRe.ReplaceAllString(redacted, "[REDACTED]")

					return &GuardrailResult{
						Passed:   true,
						Modified: true,
						Content:  redacted,
						Reason:   fmt.Sprintf("Redacted PII: %s", strings.Join(found, ", ")),
					}, nil
				}

				return &GuardrailResult{
					Passed: false,
					Reason: fmt.Sprintf("content contains PII: %s", strings.Join(found, ", ")),
				}, nil
			}

			return &GuardrailResult{Passed: true}, nil
		},
		WithGuardrailDescription("Detects and optionally redacts PII"),
	)
}

// CodeInjectionGuardrail checks for potential code injection patterns
func CodeInjectionGuardrail() *Guardrail {
	return NewGuardrail(
		"code_injection",
		GuardrailKindInput,
		func(ctx context.Context, content string, kind GuardrailKind) (*GuardrailResult, error) {
			// SQL injection patterns
			sqlPatterns := []string{
				`'(\s+or\s+|\|\|)`,
				`;\s*drop\s+`,
				`;\s*delete\s+`,
				`union\s+select`,
				`\$\{.*\}`,
			}

			// Command injection patterns
			cmdPatterns := []string{
				`;\s*rm\s+`,
				`;\s*curl\s+`,
				`;\s*wget\s+`,
				`\|\s*rm\s+`,
				"`.*`",
				`\$\([^)]*\)`,
			}

			// XSS patterns
			xssPatterns := []string{
				`<script[^>]*>`,
				`javascript:`,
				`on\w+\s*=\s*"[^"]*\("`,
			}

			allPatterns := append(append(sqlPatterns, cmdPatterns...), xssPatterns...)

			lowerContent := strings.ToLower(content)

			for _, pattern := range allPatterns {
				matched, _ := regexp.MatchString(pattern, lowerContent)
				if matched {
					return &GuardrailResult{
						Passed: false,
						Reason: fmt.Sprintf("content contains potential injection pattern: %s", pattern),
					}, nil
				}
			}

			return &GuardrailResult{Passed: true}, nil
		},
		WithGuardrailDescription("Checks for code injection patterns"),
	)
}

// PromptInjectionGuardrail checks for prompt injection attempts
func PromptInjectionGuardrail() *Guardrail {
	return NewGuardrail(
		"prompt_injection",
		GuardrailKindInput,
		func(ctx context.Context, content string, kind GuardrailKind) (*GuardrailResult, error) {
			// Common prompt injection patterns
			injectionPatterns := []string{
				"ignore previous instructions",
				"disregard all above",
				"forget everything",
				"new instructions:",
				"system prompt:",
				"jailbreak",
				"developer mode",
				"override protocol",
				"<instructions>",
				"</instructions>",
				"as an ai language model",
				"pretend you are not",
			}

			lowerContent := strings.ToLower(content)

			for _, pattern := range injectionPatterns {
				if strings.Contains(lowerContent, strings.ToLower(pattern)) {
					return &GuardrailResult{
						Passed: false,
						Reason: fmt.Sprintf("potential prompt injection detected: %s", pattern),
					}, nil
				}
			}

			return &GuardrailResult{Passed: true}, nil
		},
		WithGuardrailDescription("Checks for prompt injection attempts"),
	)
}

// ProfanityFilterGuardrail filters out profanity
func ProfanityFilterGuardrail(profanityList []string) *Guardrail {
	return NewGuardrail(
		"profanity_filter",
		GuardrailKindBoth,
		func(ctx context.Context, content string, kind GuardrailKind) (*GuardrailResult, error) {
			if len(profanityList) == 0 {
				return &GuardrailResult{Passed: true}, nil
			}

			words := strings.Fields(content)
			found := []string{}
			redactedWords := make([]string, len(words))

			for i, word := range words {
				lowerWord := strings.ToLower(strings.Trim(word, ".,!?;:'\""))
				isProfane := false

				for _, profanity := range profanityList {
					if strings.ToLower(profanity) == lowerWord {
						found = append(found, profanity)
						// Redact with asterisks
						redacted := []rune(word)
						for j, r := range redacted {
							if unicode.IsLetter(r) {
								redacted[j] = '*'
							}
						}
						redactedWords[i] = string(redacted)
						isProfane = true
						break
					}
				}

				if !isProfane {
					redactedWords[i] = words[i]
				}
			}

			if len(found) > 0 {
				redacted := strings.Join(redactedWords, " ")
				return &GuardrailResult{
					Passed:   true,
					Modified: true,
					Content:  redacted,
					Reason:   fmt.Sprintf("Filtered profanity: %s", strings.Join(found, ", ")),
				}, nil
			}

			return &GuardrailResult{Passed: true}, nil
		},
		WithGuardrailDescription("Filters out profanity"),
	)
}

// RateLimitGuardrail tracks usage and enforces rate limits
type RateLimitGuardrail struct {
	requests  map[string][]time.Time
	limit     int
	window    time.Duration
	maxTokens int
}

// NewRateLimitGuardrail creates a new rate limiter
func NewRateLimitGuardrail(limit int, window time.Duration) *Guardrail {
	rl := &RateLimitGuardrail{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}

	return NewGuardrail(
		"rate_limit",
		GuardrailKindInput,
		func(ctx context.Context, content string, kind GuardrailKind) (*GuardrailResult, error) {
			// Get session ID from context if available
			sessionID := "default"
			if sid, ok := ctx.Value("session_id").(string); ok {
				sessionID = sid
			}

			now := time.Now()
			cutoff := now.Add(-rl.window)

			// Clean old requests and count recent ones
			var recent []time.Time
			for _, t := range rl.requests[sessionID] {
				if t.After(cutoff) {
					recent = append(recent, t)
				}
			}

			if len(recent) >= rl.limit {
				return &GuardrailResult{
					Passed: false,
					Reason: fmt.Sprintf("rate limit exceeded: %d requests per %v", rl.limit, rl.window),
				}, nil
			}

			// Add current request
			recent = append(recent, now)
			rl.requests[sessionID] = recent

			return &GuardrailResult{Passed: true}, nil
		},
		WithGuardrailDescription(fmt.Sprintf("Rate limits requests to %d per %v", limit, window)),
	)
}

// AllowedDomainsGuardrail restricts URLs to allowed domains only
func AllowedDomainsGuardrail(domains []string) *Guardrail {
	return NewGuardrail(
		"allowed_domains",
		GuardrailKindInput,
		func(ctx context.Context, content string, kind GuardrailKind) (*GuardrailResult, error) {
			// Extract URLs from content
			urlRe := regexp.MustCompile(`https?://[^\s]+`)
			urls := urlRe.FindAllString(content, -1)

			if len(urls) == 0 {
				return &GuardrailResult{Passed: true}, nil
			}

			for _, url := range urls {
				allowed := false
				for _, domain := range domains {
					if strings.Contains(url, domain) {
						allowed = true
						break
					}
				}
				if !allowed {
					return &GuardrailResult{
						Passed: false,
						Reason: fmt.Sprintf("URL not from allowed domain: %s", url),
					}, nil
				}
			}

			return &GuardrailResult{Passed: true}, nil
		},
		WithGuardrailDescription("Restricts URLs to allowed domains"),
	)
}

// BannedDomainsGuardrail blocks URLs from banned domains
func BannedDomainsGuardrail(domains []string) *Guardrail {
	return NewGuardrail(
		"banned_domains",
		GuardrailKindInput,
		func(ctx context.Context, content string, kind GuardrailKind) (*GuardrailResult, error) {
			// Extract URLs from content
			urlRe := regexp.MustCompile(`https?://[^\s]+`)
			urls := urlRe.FindAllString(content, -1)

			for _, url := range urls {
				for _, domain := range domains {
					if strings.Contains(url, domain) {
						return &GuardrailResult{
							Passed: false,
							Reason: fmt.Sprintf("URL from banned domain: %s", url),
						}, nil
					}
				}
			}

			return &GuardrailResult{Passed: true}, nil
		},
		WithGuardrailDescription("Blocks URLs from banned domains"),
	)
}
