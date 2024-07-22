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

import (
	"testing"

	"golang.org/x/text/language"

	"github.com/google/go-cmp/cmp"
)

func TestText(t *testing.T) {
	p := NewPacket()

	A := Text{
		V: "hello world",
		Q: Q{Language(language.English)},
	}
	p.SetValue("http://ns.seehuhn.de/test/#", "prop", A)

	B, err := PacketGetValue[Text](p, "http://ns.seehuhn.de/test/#", "prop")
	if err != nil {
		t.Fatalf("p.Get: %v", err)
	}

	if d := cmp.Diff(A, B); d != "" {
		t.Errorf("A and B are different (-want +got):\n%s", d)
	}
}

func TestUnorderedArray(t *testing.T) {
	p := NewPacket()

	A := UnorderedArray[Text]{
		V: []Text{
			{V: "Hello", Q: Q{Language(language.English)}},
			{V: "Hallo", Q: Q{Language(language.German)}},
			{V: "Bonjour", Q: Q{Language(language.French)}},
		},
	}
	p.SetValue("http://ns.seehuhn.de/test/#", "prop", A)

	B, err := PacketGetValue[UnorderedArray[Text]](p, "http://ns.seehuhn.de/test/#", "prop")
	if err != nil {
		t.Fatalf("p.Get: %v", err)
	}

	if d := cmp.Diff(A, B); d != "" {
		t.Errorf("A and B are different (-want +got):\n%s", d)
	}
}
