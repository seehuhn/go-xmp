package xmp

import "testing"

// TestDefaultPrefix ensures that the prefixes in the defaultPrefix table are
// unique and non-empty.
func TestDefaultPrefix(t *testing.T) {
	seen := make(map[string]bool)
	for _, p := range defaultPrefix {
		if seen[p] {
			t.Errorf("prefix %q is not unique", p)
		}
		if p == "" {
			t.Errorf("prefix %q is empty", p)
		}
		seen[p] = true
	}
}
