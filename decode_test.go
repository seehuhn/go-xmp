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
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type decodeTestCase struct {
	desc string
	in   string
	out  *Packet
	err  error
}

const (
	head = `<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:test="http://ns.seehuhn.de/test/#">\n`
	foot = `\n</rdf:RDF>\n`
)

// decodeTestCases is a set of test cases for the Read function.
// The input is wrapped in an rdf:RDF element:
//
//	<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:test="http://ns.seehuhn.de/test/#">
//	  ...
//	</rdf:RDF>
var decodeTestCases = []decodeTestCase{
	{
		desc: "simple",
		in:   `<rdf:Description rdf:about=""><test:prop>testvalue</test:prop></rdf:Description>`,
		out: &Packet{
			Properties: map[xml.Name]Raw{
				elemTest: Text{V: "testvalue"},
			},
		},
	},
	{
		desc: "simple URI",
		in:   `<rdf:Description rdf:about=""><test:prop rdf:resource="http://example.com"/></rdf:Description>`,
		out: &Packet{
			Properties: map[xml.Name]Raw{
				elemTest: URL{V: &url.URL{Scheme: "http", Host: "example.com"}},
			},
		},
	},
	{
		desc: "CDATA",
		in:   `<rdf:Description rdf:about=""><test:prop><![CDATA[</test:prop>]]></test:prop></rdf:Description>`,
		out: &Packet{
			Properties: map[xml.Name]Raw{
				elemTest: Text{V: "</test:prop>"},
			},
		},
	},
	{
		desc: "structure value",
		in: `<rdf:Description rdf:about=""><test:prop>
					<rdf:Description>
						<test:a>1</test:a>
						<test:b>2</test:b>
						<test:c>3</test:c>
					</rdf:Description>
				</test:prop></rdf:Description>`,
		out: &Packet{
			Properties: map[xml.Name]Raw{
				elemTest: RawStruct{
					Value: map[xml.Name]Raw{
						elemTestA: Text{V: "1"},
						elemTestB: Text{V: "2"},
						elemTestC: Text{V: "3"},
					},
				},
			},
		},
	},
	{
		desc: "unordered array",
		in: `<rdf:Description rdf:about=""><test:prop>
					<rdf:Bag>
						<rdf:li>1</rdf:li>
						<rdf:li>2</rdf:li>
						<rdf:li>3</rdf:li>
					</rdf:Bag>
				</test:prop></rdf:Description>`,
		out: &Packet{
			Properties: map[xml.Name]Raw{
				elemTest: RawArray{
					Value: []Raw{
						Text{V: "1"},
						Text{V: "2"},
						Text{V: "3"},
					},
					Kind: Unordered,
				},
			},
		},
	},
	{
		desc: "ordered array",
		in: `<rdf:Description rdf:about=""><test:prop>
					<rdf:Seq>
						<rdf:li>4</rdf:li>
						<rdf:li>5</rdf:li>
						<rdf:li>6</rdf:li>
					</rdf:Seq>
				</test:prop></rdf:Description>`,
		out: &Packet{
			Properties: map[xml.Name]Raw{
				elemTest: RawArray{
					Value: []Raw{
						Text{V: "4"},
						Text{V: "5"},
						Text{V: "6"},
					},
					Kind: Ordered,
				},
			},
		},
	},
	{
		desc: "alternative array",
		in: `<rdf:Description rdf:about=""><test:prop>
					<rdf:Alt>
						<rdf:li>7</rdf:li>
						<rdf:li>8</rdf:li>
						<rdf:li>9</rdf:li>
					</rdf:Alt>
				</test:prop></rdf:Description>`,
		out: &Packet{
			Properties: map[xml.Name]Raw{
				elemTest: RawArray{
					Value: []Raw{
						Text{V: "7"},
						Text{V: "8"},
						Text{V: "9"},
					},
					Kind: Alternative,
				},
			},
		},
	},
	{
		desc: "xml:lang on property",
		in:   `<rdf:Description rdf:about=""><test:prop xml:lang="de">Hallo</test:prop></rdf:Description>`,
		out: &Packet{
			Properties: map[xml.Name]Raw{
				elemTest: Text{
					V: "Hallo",
					Q: Q{{Name: nameXMLLang, Value: Text{V: "de"}}},
				},
			},
		},
	},
	{
		desc: "xml:lang on URI value",
		in:   `<rdf:Description rdf:about=""><test:prop rdf:resource="http://example.com" xml:lang="de"/></rdf:Description>`,
		out: &Packet{
			Properties: map[xml.Name]Raw{
				elemTest: URL{
					V: &url.URL{Scheme: "http", Host: "example.com"},
					Q: Q{{Name: nameXMLLang, Value: Text{V: "de"}}},
				},
			},
		},
	},
	{
		desc: "xml:lang on structure field",
		in: `<rdf:Description rdf:about=""><test:prop>
					<rdf:Description>
						<test:a>1</test:a>
						<test:b>2</test:b>
						<test:c xml:lang="de">drei</test:c>
					</rdf:Description>
				</test:prop></rdf:Description>`,
		out: &Packet{
			Properties: map[xml.Name]Raw{
				elemTest: RawStruct{
					Value: map[xml.Name]Raw{
						elemTestA: Text{V: "1"},
						elemTestB: Text{V: "2"},
						elemTestC: Text{
							V: "drei",
							Q: Q{{Name: nameXMLLang, Value: Text{V: "de"}}},
						},
					},
				},
			},
		},
	},
	{
		desc: "xml:lang on array item",
		in: `<rdf:Description rdf:about=""><test:prop>
					<rdf:Alt>
						<rdf:li xml:lang="x-default">zwei</rdf:li>
						<rdf:li xml:lang="en">two</rdf:li>
						<rdf:li xml:lang="de-de">zwei</rdf:li>
					</rdf:Alt>
				</test:prop></rdf:Description>`,
		out: &Packet{
			Properties: map[xml.Name]Raw{
				elemTest: RawArray{
					Value: []Raw{
						Text{V: "zwei", Q: Q{{Name: nameXMLLang, Value: Text{V: "x-default"}}}},
						Text{V: "two", Q: Q{{Name: nameXMLLang, Value: Text{V: "en"}}}},
						Text{V: "zwei", Q: Q{{Name: nameXMLLang, Value: Text{V: "de-de"}}}},
					},
					Kind: Alternative,
				},
			},
		},
	},
	{
		desc: "xml:lang on qualifier value",
		in: `<rdf:Description rdf:about="">
				<test:prop><rdf:Description>
					<rdf:value>Hallo</rdf:value>
					<test:q xml:lang="de">Eigenschaft</test:q>
				</rdf:Description></test:prop>
			</rdf:Description>`,
		out: &Packet{
			Properties: map[xml.Name]Raw{
				elemTest: Text{
					V: "Hallo",
					Q: Q{
						{Name: elemTestQ, Value: Text{
							V: "Eigenschaft",
							Q: []Qualifier{{Name: nameXMLLang, Value: Text{V: "de"}}},
						}},
					},
				},
			},
		},
	},
	{
		desc: "general qualfiers",
		in: `<rdf:Description rdf:about="">
			<test:prop>
				<rdf:Description>
					<rdf:value>test value</rdf:value>
					<test:q>qualifier</test:q>
				</rdf:Description>
			</test:prop>
			</rdf:Description>`,
		out: &Packet{
			Properties: map[xml.Name]Raw{
				elemTest: Text{
					V: "test value",
					Q: []Qualifier{
						{elemTestQ, Text{V: "qualifier"}},
					},
				},
			},
		},
	},
	{
		desc: "general qualfiers on URI value",
		in: `<rdf:Description rdf:about="">
			<test:prop>
				<rdf:Description>
					<rdf:value rdf:resource="http://example.com"/>
					<test:q>qualifier</test:q>
				</rdf:Description>
			</test:prop>
			</rdf:Description>`,
		out: &Packet{
			Properties: map[xml.Name]Raw{
				elemTest: URL{
					V: &url.URL{Scheme: "http", Host: "example.com"},
					Q: []Qualifier{
						{elemTestQ, Text{V: "qualifier"}},
					},
				},
			},
		},
	},
	{
		desc: "general qualifier on structure field",
		in: `<rdf:Description rdf:about=""><test:prop>
					<rdf:Description>
						<test:a>1</test:a>
						<test:b>2</test:b>
						<test:c>
							<rdf:Description>
								<rdf:value>3</rdf:value>
								<test:q>qualifier</test:q>
							</rdf:Description>
						</test:c>
					</rdf:Description>
				</test:prop></rdf:Description>`,
		out: &Packet{
			Properties: map[xml.Name]Raw{
				elemTest: RawStruct{
					Value: map[xml.Name]Raw{
						elemTestA: Text{V: "1"},
						elemTestB: Text{V: "2"},
						elemTestC: Text{
							V: "3",
							Q: Q{{elemTestQ, Text{V: "qualifier"}}},
						},
					},
				},
			},
		},
	},
	{
		desc: "general qualifier on array item",
		in: `<rdf:Description rdf:about=""><test:prop>
					<rdf:Seq>
						<rdf:li>eins</rdf:li>
						<rdf:li>zwei</rdf:li>
						<rdf:li>
							<rdf:Description>
								<rdf:value>drei</rdf:value>
								<test:q>qualifier</test:q>
							</rdf:Description>
						</rdf:li>
					</rdf:Seq>
				</test:prop></rdf:Description>`,
		out: &Packet{
			Properties: map[xml.Name]Raw{
				elemTest: RawArray{
					Value: []Raw{
						Text{V: "eins"},
						Text{V: "zwei"},
						Text{V: "drei",
							Q: Q{{elemTestQ, Text{V: "qualifier"}}}},
					},
					Kind: Ordered,
				},
			},
		},
	},
	{
		desc: "list of zero qualifiers",
		in: `<rdf:Description rdf:about="">
			<test:prop>
				<rdf:Description>
					<rdf:value>test value</rdf:value>
				</rdf:Description>
			</test:prop>
			</rdf:Description>`,
		out: &Packet{
			Properties: map[xml.Name]Raw{
				elemTest: Text{
					V: "test value",
				},
			},
		},
	},

	{
		desc: "simple text as property",
		in:   `<rdf:Description rdf:about="" test:prop="value"/>`,
		out: &Packet{
			Properties: map[xml.Name]Raw{
				elemTest: Text{V: "value"},
			},
		},
	},
	{
		desc: "some structure values as properties",
		in: `<rdf:Description rdf:about=""><test:prop>
					<rdf:Description test:a="1" test:b="2">
						<test:c>3</test:c>
					</rdf:Description>
				</test:prop></rdf:Description>`,
		out: &Packet{
			Properties: map[xml.Name]Raw{
				elemTest: RawStruct{
					Value: map[xml.Name]Raw{
						elemTestA: Text{V: "1"},
						elemTestB: Text{V: "2"},
						elemTestC: Text{V: "3"},
					},
				},
			},
		},
	},
	{
		desc: "all structure values as properties",
		in: `<rdf:Description rdf:about=""><test:prop>
					<rdf:Description test:a="1" test:b="2" test:c="3"/>
				</test:prop></rdf:Description>`,
		out: &Packet{
			Properties: map[xml.Name]Raw{
				elemTest: RawStruct{
					Value: map[xml.Name]Raw{
						elemTestA: Text{V: "1"},
						elemTestB: Text{V: "2"},
						elemTestC: Text{V: "3"},
					},
				},
			},
		},
	},
	{
		desc: "some general qualfiers as properties",
		in: `<rdf:Description rdf:about="">
			<test:prop>
				<rdf:Description test:q="qualifier">
					<rdf:value>test value</rdf:value>
				</rdf:Description>
			</test:prop>
			</rdf:Description>`,
		out: &Packet{
			Properties: map[xml.Name]Raw{
				elemTest: Text{
					V: "test value",
					Q: []Qualifier{
						{elemTestQ, Text{V: "qualifier"}},
					},
				},
			},
		},
	},
	{
		desc: "all general qualfiers as properties",
		in: `<rdf:Description rdf:about="">
			<test:prop>
				<rdf:Description test:q="qualifier" rdf:value="test value"/>
			</test:prop>
			</rdf:Description>`,
		out: &Packet{
			Properties: map[xml.Name]Raw{
				elemTest: Text{
					V: "test value",
					Q: []Qualifier{
						{elemTestQ, Text{V: "qualifier"}},
					},
				},
			},
		},
	},
	{
		desc: "short form structure",
		in: `<rdf:Description rdf:about=""><test:prop rdf:parseType="Resource">
					<test:a>1</test:a>
					<test:b>2</test:b>
					<test:c>3</test:c>
				</test:prop></rdf:Description>`,
		out: &Packet{
			Properties: map[xml.Name]Raw{
				elemTest: RawStruct{
					Value: map[xml.Name]Raw{
						elemTestA: Text{V: "1"},
						elemTestB: Text{V: "2"},
						elemTestC: Text{V: "3"},
					},
				},
			},
		},
	},
	{
		desc: "short form general qualfiers",
		in: `<rdf:Description rdf:about="">
			<test:prop rdf:parseType="Resource">
				<rdf:value>test value</rdf:value>
				<test:q>qualifier</test:q>
			</test:prop>
			</rdf:Description>`,
		out: &Packet{
			Properties: map[xml.Name]Raw{
				elemTest: Text{
					V: "test value",
					Q: []Qualifier{
						{elemTestQ, Text{V: "qualifier"}},
					},
				},
			},
		},
	},
	{
		desc: "very short form structure",
		in: `<rdf:Description rdf:about="">
				<test:prop test:a="1" test:b="2" test:c="3"/>
			</rdf:Description>`,
		out: &Packet{
			Properties: map[xml.Name]Raw{
				elemTest: RawStruct{
					Value: map[xml.Name]Raw{
						elemTestA: Text{V: "1"},
						elemTestB: Text{V: "2"},
						elemTestC: Text{V: "3"},
					},
				},
			},
		},
	},

	{
		desc: "typed node for structure",
		in: `<rdf:Description rdf:about="">
			<test:prop>
			<test:Type>
				<test:a>1</test:a>
			</test:Type>
			</test:prop>
		</rdf:Description>`,
		out: &Packet{
			Properties: map[xml.Name]Raw{
				elemTest: RawStruct{
					Value: map[xml.Name]Raw{
						elemTestA: Text{V: "1"},
					},
					Q: Q{{
						Name:  nameRDFType,
						Value: URL{V: &url.URL{Scheme: "http", Host: "ns.seehuhn.de", Path: "/test/", Fragment: "Type"}},
					}},
				},
			},
		},
	},
	{ // this is the same as the previous test, but without the typed node
		desc: "avoiding typed node",
		in: `<rdf:Description rdf:about="">
			<test:prop>
			<rdf:Description>
			<rdf:value rdf:parseType="Resource">
			<test:a>1</test:a>
			</rdf:value>
			<rdf:type rdf:resource="http://ns.seehuhn.de/test/#Type"/>
			</rdf:Description>
			</test:prop>
		</rdf:Description>`,
		out: &Packet{
			Properties: map[xml.Name]Raw{
				elemTest: RawStruct{
					Value: map[xml.Name]Raw{
						elemTestA: Text{V: "1"},
					},
					Q: Q{{
						Name:  nameRDFType,
						Value: URL{V: &url.URL{Scheme: "http", Host: "ns.seehuhn.de", Path: "/test/", Fragment: "Type"}},
					}},
				},
			},
		},
	},
	{
		desc: "typed node for a property with general qualifiers",
		in: `<rdf:Description rdf:about="">
			<test:prop>
			<test:Type>
				<rdf:value>1</rdf:value>
				<test:q>q</test:q>
			</test:Type>
			</test:prop>
		</rdf:Description>`,
		out: &Packet{
			Properties: map[xml.Name]Raw{
				elemTest: Text{
					V: "1",
					Q: Q{
						{
							Name:  nameRDFType,
							Value: URL{V: &url.URL{Scheme: "http", Host: "ns.seehuhn.de", Path: "/test/", Fragment: "Type"}},
						},
						{elemTestQ, Text{V: "q"}},
					},
				},
			},
		},
	},

	{
		desc: "strange namespace prefix",
		in: `<rdf:Description rdf:about="" xmlns:_="http://example.com">
				<_:prop _:q=""/>
			</rdf:Description>`,
		out: &Packet{
			Properties: map[xml.Name]Raw{
				{Space: "http://example.com", Local: "prop"}: RawStruct{
					Value: map[xml.Name]Raw{
						{Space: "http://example.com", Local: "q"}: Text{V: ""},
					},
				},
			},
		},
	},

	// xmlns declarations on the property element itself (rather than on
	// the rdf:Description or rdf:RDF parent) must not change how the
	// element is classified.  Some writers, including pikepdf, emit XMP
	// in this style.
	{
		desc: "xmlns on literal property element",
		in:   `<rdf:Description rdf:about=""><foo:p xmlns:foo="http://example.com/">hello</foo:p></rdf:Description>`,
		out: &Packet{
			Properties: map[xml.Name]Raw{
				{Space: "http://example.com/", Local: "p"}: Text{V: "hello"},
			},
		},
	},
	{
		desc: "xmlns on resource property element with rdf:Alt",
		in:   `<rdf:Description rdf:about=""><foo:p xmlns:foo="http://example.com/"><rdf:Alt><rdf:li xml:lang="x-default">a title</rdf:li></rdf:Alt></foo:p></rdf:Description>`,
		out: &Packet{
			Properties: map[xml.Name]Raw{
				{Space: "http://example.com/", Local: "p"}: RawArray{
					Value: []Raw{
						Text{V: "a title", Q: Q{{Name: nameXMLLang, Value: Text{V: "x-default"}}}},
					},
					Kind: Alternative,
				},
			},
		},
	},
	{
		desc: "xmlns on resource property element with rdf:Seq",
		in:   `<rdf:Description rdf:about=""><foo:p xmlns:foo="http://example.com/"><rdf:Seq><rdf:li>x</rdf:li><rdf:li>y</rdf:li></rdf:Seq></foo:p></rdf:Description>`,
		out: &Packet{
			Properties: map[xml.Name]Raw{
				{Space: "http://example.com/", Local: "p"}: RawArray{
					Value: []Raw{Text{V: "x"}, Text{V: "y"}},
					Kind:  Ordered,
				},
			},
		},
	},
}

func TestDecode(t *testing.T) {
	for i, tc := range decodeTestCases {
		t.Run(tc.desc, func(t *testing.T) {
			in := head + tc.in + foot
			p, err := Read(strings.NewReader(in))
			if err != tc.err {
				t.Fatalf("%d: unexpected error: %v != %v", i, err, tc.err)
			}
			if d := cmp.Diff(p, tc.out, cmp.AllowUnexported(Packet{})); d != "" {
				t.Fatalf("%d: unexpected packet (-got +want):\n%s", i, d)
			}
		})
	}
}

func TestIsValidPropertyName(t *testing.T) {
	type testCases struct {
		in    xml.Name
		valid bool
	}
	tests := []testCases{
		{xml.Name{Space: "http://example.com", Local: "p"}, true},
		{xml.Name{Space: "", Local: "p"}, false},
		{xml.Name{Space: "http://example.com", Local: ""}, false},

		{nameRDFType, true}, // the only valid name in RDF namespace
		{xml.Name{Space: NSRDF, Local: "resource"}, false},
		{xml.Name{Space: NSRDF, Local: "p"}, false},
		{nameRDFValue, false},

		// all of the xml: namespace is forbidden
		{nameXMLLang, false},
		{xml.Name{Space: NSXML, Local: "p"}, false},

		{xml.Name{Space: "0", Local: ":"}, false},
	}
	for i, tc := range tests {
		t.Run(fmt.Sprintf("%02d", i), func(t *testing.T) {
			if got := isValidPropertyName(tc.in); got != tc.valid {
				t.Fatalf("%d: unexpected result: %v != %v", i, got, tc.valid)
			}
		})
	}

}

func TestIsValidQualifierName(t *testing.T) {
	type testCases struct {
		in    xml.Name
		valid bool
	}
	tests := []testCases{
		{xml.Name{Space: "http://example.com", Local: "q"}, true},
		{xml.Name{Space: "", Local: "q"}, false},
		{xml.Name{Space: "http://example.com", Local: ""}, false},

		{nameRDFType, true}, // the only valid name in RDF namespace
		{xml.Name{Space: NSRDF, Local: "resource"}, false},
		{xml.Name{Space: NSRDF, Local: "q"}, false},
		{nameRDFValue, false},

		{nameXMLLang, true}, // the only valid name in XML namespace
		{xml.Name{Space: NSXML, Local: "q"}, false},
	}
	for i, tc := range tests {
		t.Run(fmt.Sprintf("%02d", i), func(t *testing.T) {
			if got := isValidQualifierName(tc.in); got != tc.valid {
				t.Fatalf("%d: unexpected result: %v != %v", i, got, tc.valid)
			}
		})
	}
}

func FuzzRoundTrip(f *testing.F) {
	for _, tc := range decodeTestCases {
		in := head + tc.in + foot
		f.Add([]byte(in))
	}
	for _, tc := range encodeTestCases {
		buf := &bytes.Buffer{}
		err := tc.in.Write(buf, nil)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(buf.Bytes())
	}

	urlCmp := cmp.Comparer(func(u1, u2 *url.URL) bool {
		if u1 == nil && u2 == nil {
			return true
		}
		if u1 == nil || u2 == nil {
			return false
		}
		return u1.String() == u2.String()
	})

	f.Fuzz(func(t *testing.T, body []byte) {
		p1, err := Read(bytes.NewReader(body))
		if err != nil {
			return
		}

		buf := &bytes.Buffer{}
		err = p1.Write(buf, nil)
		if err != nil {
			t.Fatal(err)
		}
		body2 := buf.Bytes()

		p2, err := Read(bytes.NewReader(body2))
		if err != nil {
			fmt.Println()
			fmt.Println(string(body))
			fmt.Println()
			fmt.Println(string(body2))
			t.Fatal(err)
		}

		if d := cmp.Diff(p1, p2, urlCmp, cmp.AllowUnexported(Packet{})); d != "" {
			fmt.Println()
			fmt.Println(string(body))
			fmt.Println()
			fmt.Println(string(body2))
			fmt.Println()
			t.Fatalf("RoundTrip mismatch (-want +got):\n%s", d)
		}
	})
}

func TestRead_PadToLengthReadOnly(t *testing.T) {
	const body = `<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>` +
		`<x:xmpmeta xmlns:x="adobe:ns:meta/">` +
		`<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">` +
		`<rdf:Description rdf:about="" xmlns:dc="http://purl.org/dc/elements/1.1/">` +
		`<dc:title>x</dc:title>` +
		`</rdf:Description></rdf:RDF></x:xmpmeta>` +
		`<?xpacket end="r"?>`
	p, err := Read(strings.NewReader(body))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if p.PadToLength != 0 {
		t.Errorf("read-only packet: PadToLength = %d, want 0", p.PadToLength)
	}
}

func TestRead_PadToLengthMissingTrailer(t *testing.T) {
	// Same body as above, but without the closing xpacket PI.
	const body = `<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>` +
		`<x:xmpmeta xmlns:x="adobe:ns:meta/">` +
		`<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">` +
		`<rdf:Description rdf:about="" xmlns:dc="http://purl.org/dc/elements/1.1/">` +
		`<dc:title>x</dc:title>` +
		`</rdf:Description></rdf:RDF></x:xmpmeta>`
	p, err := Read(strings.NewReader(body))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if p.PadToLength != 0 {
		t.Errorf("missing trailer: PadToLength = %d, want 0", p.PadToLength)
	}
}

func TestRead_PadToLengthWritable(t *testing.T) {
	const head = `<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>` +
		`<x:xmpmeta xmlns:x="adobe:ns:meta/">` +
		`<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">` +
		`<rdf:Description rdf:about="" xmlns:dc="http://purl.org/dc/elements/1.1/">` +
		`<dc:title>x</dc:title>` +
		`</rdf:Description></rdf:RDF></x:xmpmeta>`
	const trailer = `<?xpacket end="w"?>`
	body := head + strings.Repeat(" ", 200) + trailer
	p, err := Read(strings.NewReader(body))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if p.PadToLength != len(body) {
		t.Errorf("writable trailer: PadToLength = %d, want %d", p.PadToLength, len(body))
	}
}

func TestRead_PadToLengthTooSmall(t *testing.T) {
	// Inputs short enough that pr.nRead is below the encoder's
	// irreducible scaffolding: the writable trailer must be ignored,
	// otherwise Write would later fail with ErrPacketTooLong.
	cases := []struct {
		name, body string
	}{
		{"trailer-only-double-quote", `<?xpacket end="w"?>`},
		{"trailer-only-single-quote", `<?xpacket end='w'?>`},
		{"leading-space", ` <?xpacket end="w"?>`},
		{"begin-and-trailer-no-content",
			`<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>` +
				`<?xpacket end="w"?>`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, err := Read(strings.NewReader(tc.body))
			if err != nil {
				t.Fatalf("Read: %v", err)
			}
			if p.PadToLength != 0 {
				t.Errorf("PadToLength = %d, want 0", p.PadToLength)
			}
			// the returned Packet must be writable
			if err := p.Write(&bytes.Buffer{}, nil); err != nil {
				t.Errorf("Write of Read result: %v", err)
			}
		})
	}
}

func TestRead_PadToLengthBeginPISmuggle(t *testing.T) {
	// A malformed begin PI whose id value contains the substring
	// `end="w"`.  No closing xpacket PI follows.  PadToLength must
	// stay 0; only PIs whose body starts with end= are honoured.
	const body = `<?xpacket begin="" id="x end=` + `"w" "?>` +
		`<x:xmpmeta xmlns:x="adobe:ns:meta/">` +
		`<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">` +
		`<rdf:Description rdf:about="" xmlns:dc="http://purl.org/dc/elements/1.1/">` +
		`<dc:title>x</dc:title>` +
		`</rdf:Description></rdf:RDF></x:xmpmeta>`
	p, err := Read(strings.NewReader(body))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if p.PadToLength != 0 {
		t.Errorf("smuggled end= in begin PI: PadToLength = %d, want 0", p.PadToLength)
	}
}

func TestNewPacket_PadToLengthZero(t *testing.T) {
	p := NewPacket()
	if p.PadToLength != 0 {
		t.Errorf("NewPacket: PadToLength = %d, want 0", p.PadToLength)
	}
}

func TestRead_MalformedClassification(t *testing.T) {
	// A truncated xpacket — the XML decoder reaches EOF mid-element.
	const body = `<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>` +
		`<x:xmpmeta xmlns:x="adobe:ns:meta/">` +
		`<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">` +
		`<rdf:Description rdf:about=""`
	_, err := Read(strings.NewReader(body))
	if err == nil {
		t.Fatal("Read on truncated input: expected error, got nil")
	}
	if !errors.Is(err, ErrMalformed) {
		t.Errorf("Read on truncated input: got %v, want errors.Is(err, ErrMalformed)", err)
	}
}

func TestRead_IOErrorPassThrough(t *testing.T) {
	// Custom reader that returns a sentinel error mid-stream.
	want := errors.New("custom I/O failure")
	br := &errReader{
		head: []byte(`<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>` +
			`<x:xmpmeta xmlns:x="adobe:ns:meta/">`),
		err: want,
	}
	_, err := Read(br)
	if err == nil {
		t.Fatal("Read with failing reader: expected error, got nil")
	}
	if !errors.Is(err, want) {
		t.Errorf("Read with failing reader: got %v, want errors.Is(err, want)", err)
	}
	if errors.Is(err, ErrMalformed) {
		t.Errorf("Read with failing reader: I/O error wrapped as ErrMalformed: %v", err)
	}
}

// errReader serves head bytes, then returns err.
type errReader struct {
	head []byte
	pos  int
	err  error
}

func (r *errReader) Read(p []byte) (int, error) {
	if r.pos < len(r.head) {
		n := copy(p, r.head[r.pos:])
		r.pos += n
		return n, nil
	}
	return 0, r.err
}

func TestRead_DeepNestingDoesNotOverflowStack(t *testing.T) {
	// Build a packet whose value is nested far deeper than
	// maxPropertyDepth.  Read must not crash or grow the goroutine
	// stack without bound; the depth cap silently drops nesting
	// beyond the limit.
	const nestDepth = 5000
	var sb strings.Builder
	sb.Grow(nestDepth * 80)
	sb.WriteString(`<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>`)
	sb.WriteString(`<x:xmpmeta xmlns:x="adobe:ns:meta/">`)
	sb.WriteString(`<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">`)
	sb.WriteString(`<rdf:Description rdf:about="" xmlns:dc="http://purl.org/dc/elements/1.1/">`)
	sb.WriteString(`<dc:title rdf:parseType="Resource">`)
	for range nestDepth {
		sb.WriteString(`<dc:n rdf:parseType="Resource">`)
	}
	sb.WriteString(`<dc:leaf>x</dc:leaf>`)
	for range nestDepth {
		sb.WriteString(`</dc:n>`)
	}
	sb.WriteString(`</dc:title></rdf:Description></rdf:RDF></x:xmpmeta>`)
	sb.WriteString(`<?xpacket end="r"?>`)

	p, err := Read(strings.NewReader(sb.String()))
	if err != nil {
		t.Fatalf("Read failed on deeply nested input: %v", err)
	}
	titleName := xml.Name{Space: "http://purl.org/dc/elements/1.1/", Local: "title"}
	if _, ok := p.Properties[titleName]; !ok {
		t.Errorf("title property not present after deep-nesting parse")
	}
}
