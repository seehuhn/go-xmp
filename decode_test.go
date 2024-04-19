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
	"net/url"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDecodeSimple(t *testing.T) {
	// This is the example in section 7.4 of ISO 16684-1:2011.
	const in = `<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:xmp="http://ns.adobe.com/xap/1.0/">
		<rdf:Description rdf:about="">
		<xmp:Rating>3</xmp:Rating>
		</rdf:Description>
 		</rdf:RDF>`
	p, err := Read(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}

	if len(p.Properties) != 1 {
		t.Fatalf("unexpected number of properties: %d", len(p.Properties))
	}
	prop, ok := p.Properties[xml.Name{Space: "http://ns.adobe.com/xap/1.0/", Local: "Rating"}]
	if !ok {
		t.Fatalf("missing property")
	}
	if len(prop.Qualifiers()) != 0 {
		t.Fatalf("unexpected number of qualifiers: %d", len(prop.Qualifiers()))
	}
	if prop.(textValue).Value != "3" {
		t.Fatalf("unexpected value: %q", prop.(textValue).Value)
	}
}

func TestDecodeURI(t *testing.T) {
	// This is example 2 in section 7.5 of ISO 16684-1:2011.
	const in = `<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:xmp="http://ns.adobe.com/xap/1.0/">
		<rdf:Description rdf:about="">
		<xmp:BaseURL rdf:resource="http://www.adobe.com/"/>
		</rdf:Description>
		</rdf:RDF>`
	p, err := Read(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}

	if len(p.Properties) != 1 {
		t.Fatalf("unexpected number of properties: %d", len(p.Properties))
	}
	prop, ok := p.Properties[xml.Name{Space: "http://ns.adobe.com/xap/1.0/", Local: "BaseURL"}]
	if !ok {
		t.Fatalf("missing property")
	}
	if len(prop.Qualifiers()) != 0 {
		t.Fatalf("unexpected number of qualifiers: %d", len(prop.Qualifiers()))
	}
	if prop.(uriValue).Value.String() != "http://www.adobe.com/" {
		t.Fatalf("unexpected value: %q", prop.(uriValue).Value.String())
	}
}

func TestCDATA(t *testing.T) {
	// This is example 3 in section 7.5 of ISO 16684-1:2011.
	const in = `<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:xe="http://ns.adobe.com/xmp-example/">
		<rdf:Description rdf:about="">
		<xe:Entity>Embedded &lt;bold&gt;XML&lt;/bold&gt; markup</xe:Entity>
		<xe:CDATA><![CDATA[Embedded <bold>XML</bold> markup]]></xe:CDATA>
		</rdf:Description>
		</rdf:RDF>`
	p, err := Read(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}

	if len(p.Properties) != 2 {
		t.Fatalf("unexpected number of properties: %d", len(p.Properties))
	}
	p1, ok1 := p.Properties[xml.Name{Space: "http://ns.adobe.com/xmp-example/", Local: "Entity"}]
	p2, ok2 := p.Properties[xml.Name{Space: "http://ns.adobe.com/xmp-example/", Local: "CDATA"}]
	if !(ok1 && ok2) {
		t.Fatalf("missing property")
	}
	if len(p1.Qualifiers()) != 0 {
		t.Fatalf("unexpected number of qualifiers: %d", len(p1.Qualifiers()))
	}
	if p1.(textValue).Value != "Embedded <bold>XML</bold> markup" {
		t.Fatalf("unexpected value: %q", p1.(textValue).Value)
	}
	if len(p2.Qualifiers()) != 0 {
		t.Fatalf("unexpected number of qualifiers: %d", len(p2.Qualifiers()))
	}
	if p2.(textValue).Value != "Embedded <bold>XML</bold> markup" {
		t.Fatalf("unexpected value: %q", p2.(textValue).Value)
	}
}

func TestDecodeStruct(t *testing.T) {
	// This is the example in section 7.6 of ISO 16684-1:2011.
	const in = `<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:xmpTPg="http://ns.adobe.com/xap/1.0/t/pg/" xmlns:stDim="http://ns.adobe.com/xap/1.0/sType/Dimensions#">
		<rdf:Description rdf:about="">
		<xmpTPg:MaxPageSize>
		<rdf:Description>
		<stDim:h>11.0</stDim:h>
		<stDim:w>8.5</stDim:w>
		<stDim:unit>inch</stDim:unit>
		</rdf:Description>
		</xmpTPg:MaxPageSize>
		</rdf:Description>
 		</rdf:RDF>`
	p, err := Read(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}

	if len(p.Properties) != 1 {
		t.Fatalf("unexpected number of properties: %d", len(p.Properties))
	}
	prop, ok := p.Properties[xml.Name{Space: "http://ns.adobe.com/xap/1.0/t/pg/", Local: "MaxPageSize"}]
	if !ok {
		t.Fatalf("missing property")
	}

	s := prop.(structValue)
	if len(s.Value) != 3 {
		t.Fatalf("unexpected number of struct members: %d", len(s.Value))
	}
	expected := structValue{
		Value: map[xml.Name]Value{
			{Space: "http://ns.adobe.com/xap/1.0/sType/Dimensions#", Local: "h"}:    textValue{Value: "11.0"},
			{Space: "http://ns.adobe.com/xap/1.0/sType/Dimensions#", Local: "w"}:    textValue{Value: "8.5"},
			{Space: "http://ns.adobe.com/xap/1.0/sType/Dimensions#", Local: "unit"}: textValue{Value: "inch"},
		},
	}
	if d := cmp.Diff(s, expected); d != "" {
		t.Fatalf("unexpected struct value: %s", d)
	}
}

func TestDecodeArray(t *testing.T) {
	// This the example in section 7.7 of ISO 16684-1:2011.
	const in = `<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:dc="http://purl.org/dc/elements/1.1/">
		<rdf:Description rdf:about="">
		<dc:subject>
		<rdf:Bag>
		<rdf:li>XMP</rdf:li>
		<rdf:li>metadata</rdf:li>
		<rdf:li>ISO standard</rdf:li>
		</rdf:Bag>
		</dc:subject>
		</rdf:Description>
		</rdf:RDF>`
	p, err := Read(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}

	if len(p.Properties) != 1 {
		t.Fatalf("unexpected number of properties: %d", len(p.Properties))
	}
	prop, ok := p.Properties[xml.Name{Space: "http://purl.org/dc/elements/1.1/", Local: "subject"}]
	if !ok {
		t.Fatalf("missing property")
	}

	expected := arrayValue{
		Type: tpUnordered,
		Value: []Value{
			textValue{Value: "XMP"},
			textValue{Value: "metadata"},
			textValue{Value: "ISO standard"},
		},
	}
	if d := cmp.Diff(prop, expected); d != "" {
		t.Fatalf("unexpected array value: %s", d)
	}
}

func TestDecodeLang(t *testing.T) {
	// This is example 1 in section 7.8 of ISO 16684-1:2011.
	const in = `<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:xmp="http://ns.adobe.com/xap/1.0/">
		<rdf:Description rdf:about="">
		<dc:source xml:lang="en-us">Adobe XMP Specification, April 2010</dc:source>
		<xmp:BaseURL rdf:resource="http://www.adobe.com/" xml:lang="en"/>
		<dc:subject xml:lang="en">
		<rdf:Bag>
		<rdf:li>XMP</rdf:li>
		<rdf:li>metadata</rdf:li>
		<rdf:li>ISO standard</rdf:li>
		<rdf:li xml:lang="fr">Norme internationale de l’ISO</rdf:li>
		</rdf:Bag>
		</dc:subject>
		</rdf:Description>
		</rdf:RDF>`
	p, err := Read(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}

	if len(p.Properties) != 3 {
		t.Fatalf("unexpected number of properties: %d", len(p.Properties))
	}
	prop1, ok1 := p.Properties[xml.Name{Space: "http://purl.org/dc/elements/1.1/", Local: "source"}]
	prop2, ok2 := p.Properties[xml.Name{Space: "http://ns.adobe.com/xap/1.0/", Local: "BaseURL"}]
	prop3, ok3 := p.Properties[xml.Name{Space: "http://purl.org/dc/elements/1.1/", Local: "subject"}]
	if !(ok1 && ok2 && ok3) {
		t.Fatalf("missing property")
	}

	ex1 := textValue{
		Value: "Adobe XMP Specification, April 2010",
		Q: []Qualifier{
			{xml.Name{Space: "http://www.w3.org/XML/1998/namespace", Local: "lang"}, textValue{Value: "en-us"}},
		},
	}
	if d := cmp.Diff(prop1, ex1); d != "" {
		t.Fatalf("unexpected text value: %s", d)
	}

	ex2 := uriValue{
		Value: mustParseURL("http://www.adobe.com/"),
		Q: []Qualifier{
			{xml.Name{Space: "http://www.w3.org/XML/1998/namespace", Local: "lang"}, textValue{Value: "en"}},
		},
	}
	if d := cmp.Diff(prop2, ex2); d != "" {
		t.Fatalf("unexpected uri value: %s", d)
	}

	ex3 := arrayValue{
		Type: tpUnordered,
		Value: []Value{
			textValue{Value: "XMP"},
			textValue{Value: "metadata"},
			textValue{Value: "ISO standard"},
			textValue{Value: "Norme internationale de l’ISO",
				Q: []Qualifier{
					{xml.Name{Space: "http://www.w3.org/XML/1998/namespace", Local: "lang"}, textValue{Value: "fr"}},
				},
			},
		},
		Q: []Qualifier{
			{xml.Name{Space: "http://www.w3.org/XML/1998/namespace", Local: "lang"}, textValue{Value: "en"}},
		},
	}
	if d := cmp.Diff(prop3, ex3); d != "" {
		t.Fatalf("unexpected array value: %s", d)
	}
}

func TestDecodeQualifiers(t *testing.T) {
	// This is example 2 in section 7.8 of ISO 16684-1:2011.
	const in = `<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:xmp="http://ns.adobe.com/xap/1.0/" xmlns:xe="http://ns.adobe.com/xmp-example/">
		<rdf:Description rdf:about="">
		<dc:source>
		<rdf:Description>
		<rdf:value>Adobe XMP Specification, April 2010</rdf:value>
		<xe:qualifier>artificial example</xe:qualifier>
		</rdf:Description>
		</dc:source>
		<xmp:BaseURL>
		<rdf:Description>
		<rdf:value rdf:resource="http://www.adobe.com/"/>
		<xe:qualifier>artificial example</xe:qualifier>
		</rdf:Description>
		</xmp:BaseURL>
		<dc:subject>
		<rdf:Bag>
		<rdf:li>XMP</rdf:li>
		<rdf:li>
		<rdf:Description>
		<rdf:value>metadata</rdf:value> <xe:qualifier>artificial example</xe:qualifier>
		</rdf:Description>
		</rdf:li>
		<rdf:li>
		<rdf:Description>
		<rdf:value>ISO standard</rdf:value>
		</rdf:Description>
		</rdf:li>
		</rdf:Bag>
		</dc:subject>
		</rdf:Description>
		</rdf:RDF>`
	p, err := Read(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}

	if len(p.Properties) != 3 {
		t.Fatalf("unexpected number of properties: %d", len(p.Properties))
	}
	prop1, ok1 := p.Properties[xml.Name{Space: "http://purl.org/dc/elements/1.1/", Local: "source"}]
	prop2, ok2 := p.Properties[xml.Name{Space: "http://ns.adobe.com/xap/1.0/", Local: "BaseURL"}]
	prop3, ok3 := p.Properties[xml.Name{Space: "http://purl.org/dc/elements/1.1/", Local: "subject"}]
	if !(ok1 && ok2 && ok3) {
		t.Fatalf("missing property")
	}

	ex1 := textValue{
		Value: "Adobe XMP Specification, April 2010",
		Q: []Qualifier{
			{xml.Name{Space: "http://ns.adobe.com/xmp-example/", Local: "qualifier"}, textValue{Value: "artificial example"}},
		},
	}
	if d := cmp.Diff(prop1, ex1); d != "" {
		t.Fatalf("unexpected value (-got +want): %s", d)
	}

	ex2 := uriValue{
		Value: mustParseURL("http://www.adobe.com/"),
		Q: []Qualifier{
			{xml.Name{Space: "http://ns.adobe.com/xmp-example/", Local: "qualifier"}, textValue{Value: "artificial example"}},
		},
	}
	if d := cmp.Diff(prop2, ex2); d != "" {
		t.Fatalf("unexpected value (-got +want): %s", d)
	}

	ex3 := arrayValue{
		Type: tpUnordered,
		Value: []Value{
			textValue{Value: "XMP"},
			textValue{
				Value: "metadata",
				Q: []Qualifier{
					{xml.Name{Space: "http://ns.adobe.com/xmp-example/", Local: "qualifier"}, textValue{Value: "artificial example"}},
				},
			},
			textValue{Value: "ISO standard"},
		},
	}
	if d := cmp.Diff(prop3, ex3); d != "" {
		t.Fatalf("unexpected value (-got +want): %s", d)
	}
}

func TestShortenedElements(t *testing.T) {
	// This is the example in section 7.9.2.2 of ISO 16684-1:2011.
	const in = `<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:xmp="http://ns.adobe.com/xap/1.0/" xmlns:xmpTPg="http://ns.adobe.com/xap/1.0/t/pg/" xmlns:stDim="http://ns.adobe.com/xap/1.0/sType/Dimensions#" xmlns:xe="http://ns.adobe.com/xmp-example/">
		<rdf:Description rdf:about="" xmp:Rating="3">
		<xmpTPg:MaxPageSize>
		<rdf:Description stDim:h="11.0" stDim:w="8.5">
		<stDim:unit>inch</stDim:unit>
		</rdf:Description>
		</xmpTPg:MaxPageSize>
		<xmp:BaseURL>
		<rdf:Description xe:qualifier="artificial example">
		<rdf:value rdf:resource="http://www.adobe.com/"/>
		</rdf:Description>
		</xmp:BaseURL>
		</rdf:Description>
		</rdf:RDF>`
	p, err := Read(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}

	if len(p.Properties) != 3 {
		t.Fatalf("unexpected number of properties: %d", len(p.Properties))
	}
	prop1, ok1 := p.Properties[xml.Name{Space: "http://ns.adobe.com/xap/1.0/", Local: "Rating"}]
	prop2, ok2 := p.Properties[xml.Name{Space: "http://ns.adobe.com/xap/1.0/t/pg/", Local: "MaxPageSize"}]
	prop3, ok3 := p.Properties[xml.Name{Space: "http://ns.adobe.com/xap/1.0/", Local: "BaseURL"}]
	if !(ok1 && ok2 && ok3) {
		t.Fatalf("missing property")
	}

	if val := prop1.(textValue); val.Value != "3" || len(val.Q) != 0 {
		t.Fatalf("unexpected value: %v", val)
	}

	e2 := structValue{
		Value: map[xml.Name]Value{
			{Space: "http://ns.adobe.com/xap/1.0/sType/Dimensions#", Local: "h"}:    textValue{Value: "11.0"},
			{Space: "http://ns.adobe.com/xap/1.0/sType/Dimensions#", Local: "w"}:    textValue{Value: "8.5"},
			{Space: "http://ns.adobe.com/xap/1.0/sType/Dimensions#", Local: "unit"}: textValue{Value: "inch"},
		},
	}
	if d := cmp.Diff(prop2, e2); d != "" {
		t.Fatalf("unexpected struct value (-got +want): %s", d)
	}

	e3 := uriValue{
		Value: mustParseURL("http://www.adobe.com/"),
		Q: []Qualifier{
			{xml.Name{Space: "http://ns.adobe.com/xmp-example/", Local: "qualifier"}, textValue{Value: "artificial example"}},
		},
	}
	if d := cmp.Diff(prop3, e3); d != "" {
		t.Fatalf("unexpected value (-got +want): %s", d)
	}
}

func mustParseURL(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

func FuzzRoundTrip(f *testing.F) {
	for _, tc := range testCases {
		body, err := tc.in.Encode(true)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(body)
	}

	f.Fuzz(func(t *testing.T, body []byte) {
		p1, err := Read(bytes.NewReader(body))
		if err != nil {
			return
		}

		body2, err := p1.Encode(true)
		if err != nil {
			t.Fatal(err)
		}

		p2, err := Read(bytes.NewReader(body2))
		if err != nil {
			t.Fatal(err)
		}

		if d := cmp.Diff(p1, p2); d != "" {
			t.Fatalf("RoundTrip mismatch (-want +got):\n%s", d)
		}
	})
}
