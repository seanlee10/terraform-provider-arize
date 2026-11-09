package client

import (
	"encoding/base64"
	"strings"
)

// Arize IDs are Relay global IDs: base64 of "<Type>:<number>:<suffix>",
// e.g. "LlmIntegration:227:NVbs".
const (
	GlobalIDTypeOrganization   = "AccountOrganization"
	GlobalIDTypeLLMIntegration = "LlmIntegration"
)

// GlobalIDType returns the type encoded in an Arize global ID, or false if s
// is not one (a name, a UUID, or a value base64-encoded by hand). Arize
// rejects those with "Relay global id is invalid".
func GlobalIDType(s string) (string, bool) {
	raw, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		if raw, err = base64.RawStdEncoding.DecodeString(s); err != nil {
			return "", false
		}
	}
	typ, rest, ok := strings.Cut(string(raw), ":")
	if !ok || typ == "" || rest == "" {
		return "", false
	}
	for _, r := range typ {
		if !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9') {
			return "", false
		}
	}
	return typ, true
}
