package utils

func StringPtr(s string) *string {
	return &s
}

func ContainsString(slice []string, target string) bool {
	for _, s := range slice {
		if s == target {
			return true
		}
	}
	return false
}
