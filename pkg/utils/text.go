package utils

// TextOrDefault returns value when it is not empty, otherwise fallback.
func TextOrDefault(value string, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
