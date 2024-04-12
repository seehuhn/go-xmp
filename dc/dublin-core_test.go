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

package dc

import (
	"fmt"
	"testing"

	"seehuhn.de/go/xmp"
)

func TestDublinCore(t *testing.T) {
	dc := DublinCore{
		Contributor: xmp.UnorderedArray[xmp.ProperName]{
			Values: []xmp.ProperName{
				{Text: xmp.Text{Value: "Alice"}},
				{Text: xmp.Text{Value: "Bob"}},
			},
		},
		Coverage: xmp.Text{Value: "Earth"},
	}

	packet := &xmp.Packet{
		Properties: map[string]xmp.Model{
			"http://purl.org/dc/elements/1.1/": &dc,
		},
	}

	body, err := packet.Encode()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	fmt.Println(string(body))
}
