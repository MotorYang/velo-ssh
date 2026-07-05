package transfer

func ShouldIgnoreDefault(name string) bool {
	switch name {
	case ".DS_Store", "Thumbs.db", ".git":
		return true
	default:
		return false
	}
}
