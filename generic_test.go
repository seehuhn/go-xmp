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
	"bytes"
	"encoding/xml"
	"io"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

const testData = `<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:test="http://ns.seehuhn.de/test#">
<rdf:Description rdf:about="">
	<test:Test>hello world</test:Test>
</rdf:Description>
</rdf:RDF>`

func TestGeneric(t *testing.T) {
	var r io.Reader = strings.NewReader(testData)
	p1, err := Read(r)
	if err != nil {
		t.Fatal(err)
	}

	if len(p1.Models) != 1 {
		t.Fatalf("unexpected number of models: %d", len(p1.Models))
	}
	model, ok := p1.Models["http://ns.seehuhn.de/test#"]
	if !ok {
		t.Fatalf("property not found")
	}
	g, ok := model.(*genericModel)
	if !ok {
		t.Fatalf("property has wrong type")
	}
	if len(g.Properties) != 1 {
		t.Fatalf("unexpected number of properties: %d", len(g.Properties))
	}
	v, ok := g.Properties["Test"]
	if !ok {
		t.Fatalf("property not found")
	}
	if len(v.Tokens) != 1 {
		t.Fatalf("unexpected number of tokens: %d", len(v.Tokens))
	}
	tok, ok := v.Tokens[0].(xml.CharData)
	if !ok {
		t.Fatalf("unexpected token type")
	}
	if string(tok) != "hello world" {
		t.Fatalf("unexpected token value")
	}

	// re-encode the packet
	b, err := p1.Encode()
	if err != nil {
		t.Fatal(err)
	}

	// re-parse the packet
	r = bytes.NewReader(b)
	p2, err := Read(r)
	if err != nil {
		t.Fatal(err)
	}

	if d := cmp.Diff(p1, p2); d != "" {
		t.Fatalf("unexpected packet: %s", d)
	}
}
