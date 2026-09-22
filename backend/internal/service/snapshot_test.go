package service

import (
	"strings"
	"testing"
)

func TestSnapshotRecursivelyRedactsSecrets(t *testing.T) {
	got := snapshot(map[string]any{
		"name":        "safe",
		"auth":        map[string]any{"access_token": "token-value", "nested": []any{map[string]any{"password": "password-value"}}},
		"private-key": "key-value",
	})
	for _, secret := range []string{"token-value", "password-value", "key-value"} {
		if strings.Contains(got, secret) {
			t.Fatalf("snapshot leaked %q: %s", secret, got)
		}
	}
	if !strings.Contains(got, `"name":"safe"`) || strings.Count(got, "[REDACTED]") != 3 {
		t.Fatalf("unexpected redacted snapshot: %s", got)
	}
}
