// Copyright 2024 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package xmp

import "testing"

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
