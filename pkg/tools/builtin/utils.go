package builtin

// getStringWithDefault extracts a string value from args map with a default fallback
func getStringWithDefault(args map[string]interface{}, key, defaultValue string) string {
	if value, ok := args[key].(string); ok {
		return value
	}
	return defaultValue
}