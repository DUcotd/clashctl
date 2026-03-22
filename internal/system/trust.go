package system

import (
	"os"
	"strings"
)

const allowUntrustedMirrorEnv = "CLASHCTL_ALLOW_UNTRUSTED_MIRROR"

// AllowUntrustedMirrorDownloads reports whether binary/checksum downloads may
// fall back to third-party mirrors. This is disabled by default because a
// mirror that serves both the artifact and its checksum can satisfy integrity
// checks without providing authenticity.
func AllowUntrustedMirrorDownloads() bool {
	switch strings.TrimSpace(strings.ToLower(os.Getenv(allowUntrustedMirrorEnv))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
