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
	"time"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/text/language"
)

func TestTag(t *testing.T) {
	dc1 := &DublinCore{}
	dc1.Date.Append(NewDate(time.Now()))
	dc1.Title.Set(language.English, "Hello, World!")
	dc1.Title.Set(language.German, "Grüß Gott!")
	dc1.Title.Default = NewText("Hello, World!")

	p := NewPacket()
	err := p.Set(dc1)
	if err != nil {
		t.Fatal(err)
	}

	dc2 := DublinCore{}
	p.Get(&dc2)

	if d := cmp.Diff(dc1, &dc2); d != "" {
		t.Errorf("dc1 and dc2 differ (-want +got):\n%s", d)
	}

	// buf := &bytes.Buffer{}
	// err = p.Write(buf, &PacketOptions{Pretty: true})
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// fmt.Println(buf.String())
}
