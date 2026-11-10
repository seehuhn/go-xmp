// Copyright 2024 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

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

func TestGetPrefix(t *testing.T) {
	m := map[string]string{
		"a": "http://ns.seehuhn.de/test/a/#",
	}
	p := getPrefix(m, "http://ns.seehuhn.de/test/b/#")
	if p != "b" {
		t.Errorf("unexpected prefix %q", p)
	}
	p = getPrefix(m, "http://ns.seehuhn.de/test/other/a/#")
	if p == "a" {
		t.Errorf("unexpected prefix %q", p)
	}
}
