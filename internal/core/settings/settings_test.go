package settings

import (
	"testing"
)

// TestKeyHint pins the hint contract the save hook enforces: the hint is the key's LAST FOUR
// characters, and an unset key has no hint at all. Short keys still yield only what is there,
// so the hint can never reconstruct a secret it describes.
func TestKeyHint(t *testing.T) {
	cases := []struct{ key, want string }{
		{"", ""},
		{"ab", "ab"},
		{"abcd", "abcd"},
		{"sk-ant-0123456789WXYZ", "WXYZ"},
	}
	for _, tc := range cases {
		if got := KeyHint(tc.key); got != tc.want {
			t.Errorf("KeyHint(%q) = %q, want %q", tc.key, got, tc.want)
		}
	}
}
