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
	"fmt"
	"os"
	"testing"
)

func TestXMP(t *testing.T) {
	fd, err := os.Open("sample2.xml")
	if err != nil {
		t.Fatal(err)
	}
	defer fd.Close()

	p, err := Read(fd)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("%#v\n", p)
}
