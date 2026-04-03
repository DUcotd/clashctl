package core

import (
	"crypto/rand"
	"encoding/hex"
)

// GenerateControllerSecret returns a random secret for the local Mihomo controller.
func GenerateControllerSecret() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}
	return hex.EncodeToString(buf)
}
