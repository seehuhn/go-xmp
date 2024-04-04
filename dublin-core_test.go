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
	"encoding/xml"
	"fmt"
	"testing"
)

func TestDublinCore(t *testing.T) {
	dc := DublinCore{
		Contributor: UnorderedArray[ProperName]{
			Values: []ProperName{
				{Text: Text{Value: "Alice"}},
				{Text: Text{Value: "Bob"}},
			},
		},
		Coverage: Text{Value: "Earth"},
	}

	e := NewEncoder()
	e.Encoder.Indent("", "  ")

	name := xml.Name{
		// Space: rdfNS,
		Local: "rdf:RDF",
	}
	err := e.EncodeToken(xml.StartElement{
		Name: name,
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "xmlns:rdf"}, Value: rdfNS},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	err = dc.EncodeProperties(e, "dc")
	if err != nil {
		t.Errorf("EncodeProperties failed: %v", err)
	}

	err = e.EncodeToken(xml.EndElement{Name: name})
	if err != nil {
		t.Fatal(err)
	}

	err = e.Encoder.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	fmt.Println(e.buf.String())
}
