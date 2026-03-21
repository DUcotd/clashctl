package mihomo

// GroupTypeIcon returns a stable icon for a proxy group type.
func GroupTypeIcon(t string) string {
	switch NormalizeProxyType(t) {
	case "select":
		return "🔀"
	case "url-test":
		return "⚡"
	case "fallback":
		return "🔄"
	case "load-balance":
		return "⚖️"
	default:
		return "📦"
	}
}
