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
	"strings"
	"testing"
)

// TestSimple tests the parsing of a simple XMP packet.
func TestSimple(t *testing.T) {
	// This is the example from section 7.4 of ISO 16684-1:2011
	const testData = `<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:xmp="http://ns.adobe.com/xap/1.0/">
	<rdf:Description rdf:about="">
		<xmp:Rating>3</xmp:Rating>
	</rdf:Description>
 	</rdf:RDF>`

	p, err := Read(strings.NewReader(testData))
	if err != nil {
		t.Fatal(err)
	}

	if len(p.Models) != 1 {
		t.Fatalf("unexpected number of models: %d", len(p.Models))
	}
	model, ok := p.Models["http://ns.adobe.com/xap/1.0/"]
	if !ok {
		t.Fatal("model not found")
	}
	// TODO(voss): update this once the xmp namespace is implemented
	g := model.(*genericModel)
	if len(g.Properties) != 1 {
		t.Fatalf("unexpected number of properties: %d", len(g.Properties))
	}
	prop, ok := g.Properties["Rating"]
	if !ok {
		t.Fatal("property not found")
	}
	if len(prop.Tokens) != 1 {
		t.Fatalf("unexpected number of tokens: %d", len(prop.Tokens))
	}
	tok, ok := prop.Tokens[0].(xml.CharData)
	if !ok {
		t.Fatalf("unexpected token type")
	}
	if string(tok) != "3" {
		t.Fatalf("unexpected token value")
	}
	if len(prop.Q) != 0 {
		t.Fatalf("unexpected number of qualifiers: %d", len(prop.Q))
	}
}
