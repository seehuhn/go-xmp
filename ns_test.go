// seehuhn.de/go/xmp - Extensible Metadata Platform in Go
// Copyright (C) 2024  Jochen Voss <voss@seehuhn.de>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

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
