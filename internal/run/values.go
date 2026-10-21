package run

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode/utf8"
)

// maxRecordBytes bounds one recorded observation, such as a check's output.
const maxRecordBytes = 64 << 20

func text(s string) bool {
	return utf8.ValidString(s) && strings.TrimSpace(s) != "" && !strings.ContainsRune(s, 0)
}

func identifier(s string) bool { return text(s) && len(s) <= 256 }

func imageIdentity(s string) bool {
	if len(s) != 71 || !strings.HasPrefix(s, "sha256:") || s != strings.ToLower(s) {
		return false
	}
	_, err := hex.DecodeString(s[7:])
	return err == nil
}

// textDigest is the sha256 digest of a text an Agent received, for evidence
// that ties the recorded payload to its transcript.
func textDigest(text string) string {
	return fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(text)))
}
