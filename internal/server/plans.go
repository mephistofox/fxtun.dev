package server

// IsUnlimited returns true if a plan limit value means "unlimited" (negative).
func IsUnlimited(v int) bool { return v < 0 }
