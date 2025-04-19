package utils

// Helper function to safely dereference string pointers, returning "" if nil
func DerefString(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

// Helper function to safely create a pointer to a string, returning nil if empty
func PtrString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
