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
			Properties: map[xml.Name]Value{
				elemTest: textValue{Value: "testvalue"},
			},
		},
	},
	{
		desc: "simple URI",
		in:   `<rdf:Description rdf:about=""><test:prop rdf:resource="http://example.com"/></rdf:Description>`,
		out: &Packet{
			Properties: map[xml.Name]Value{
				elemTest: uriValue{Value: &url.URL{Scheme: "http", Host: "example.com"}},
			},
		},
	},
	{
		desc: "CDATA",
		in:   `<rdf:Description rdf:about=""><test:prop><![CDATA[</test:prop>]]></test:prop></rdf:Description>`,
		out: &Packet{
			Properties: map[xml.Name]Value{
				elemTest: textValue{Value: "</test:prop>"},
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
			Properties: map[xml.Name]Value{
				elemTest: structValue{
					Value: map[xml.Name]Value{
						elemTestA: textValue{Value: "1"},
						elemTestB: textValue{Value: "2"},
						elemTestC: textValue{Value: "3"},
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
			Properties: map[xml.Name]Value{
				elemTest: arrayValue{
					Value: []Value{
						textValue{Value: "1"},
						textValue{Value: "2"},
						textValue{Value: "3"},
					},
					Type: tpUnordered,
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
			Properties: map[xml.Name]Value{
				elemTest: arrayValue{
					Value: []Value{
						textValue{Value: "4"},
						textValue{Value: "5"},
						textValue{Value: "6"},
					},
					Type: tpOrdered,
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
			Properties: map[xml.Name]Value{
				elemTest: arrayValue{
					Value: []Value{
						textValue{Value: "7"},
						textValue{Value: "8"},
						textValue{Value: "9"},
					},
					Type: tpAlternative,
				},
			},
		},
	},
	{
		desc: "xml:lang on property",
		in:   `<rdf:Description rdf:about=""><test:prop xml:lang="de">Hallo</test:prop></rdf:Description>`,
		out: &Packet{
			Properties: map[xml.Name]Value{
				elemTest: textValue{
					Value: "Hallo",
					Q:     Q{{Name: attrXMLLang, Value: textValue{Value: "de"}}},
				},
			},
		},
	},
	{
		desc: "xml:lang on URI value",
		in:   `<rdf:Description rdf:about=""><test:prop rdf:resource="http://example.com" xml:lang="de"/></rdf:Description>`,
		out: &Packet{
			Properties: map[xml.Name]Value{
				elemTest: uriValue{
					Value: &url.URL{Scheme: "http", Host: "example.com"},
					Q:     Q{{Name: attrXMLLang, Value: textValue{Value: "de"}}},
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
			Properties: map[xml.Name]Value{
				elemTest: structValue{
					Value: map[xml.Name]Value{
						elemTestA: textValue{Value: "1"},
						elemTestB: textValue{Value: "2"},
						elemTestC: textValue{
							Value: "drei",
							Q:     Q{{Name: attrXMLLang, Value: textValue{Value: "de"}}},
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
			Properties: map[xml.Name]Value{
				elemTest: arrayValue{
					Value: []Value{
						textValue{Value: "zwei", Q: Q{{Name: attrXMLLang, Value: textValue{Value: "x-default"}}}},
						textValue{Value: "two", Q: Q{{Name: attrXMLLang, Value: textValue{Value: "en"}}}},
						textValue{Value: "zwei", Q: Q{{Name: attrXMLLang, Value: textValue{Value: "de-de"}}}},
					},
					Type: tpAlternative,
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
			Properties: map[xml.Name]Value{
				elemTest: textValue{
					Value: "Hallo",
					Q: Q{
						{Name: elemTestQ, Value: textValue{
							Value: "Eigenschaft",
							Q:     []Qualifier{{Name: attrXMLLang, Value: textValue{Value: "de"}}},
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
			Properties: map[xml.Name]Value{
				elemTest: textValue{
					Value: "test value",
					Q: []Qualifier{
						{elemTestQ, textValue{Value: "qualifier"}},
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
			Properties: map[xml.Name]Value{
				elemTest: uriValue{
					Value: &url.URL{Scheme: "http", Host: "example.com"},
					Q: []Qualifier{
						{elemTestQ, textValue{Value: "qualifier"}},
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
			Properties: map[xml.Name]Value{
				elemTest: structValue{
					Value: map[xml.Name]Value{
						elemTestA: textValue{Value: "1"},
						elemTestB: textValue{Value: "2"},
						elemTestC: textValue{
							Value: "3",
							Q:     Q{{elemTestQ, textValue{Value: "qualifier"}}},
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
			Properties: map[xml.Name]Value{
				elemTest: arrayValue{
					Value: []Value{
						textValue{Value: "eins"},
						textValue{Value: "zwei"},
						textValue{Value: "drei",
							Q: Q{{elemTestQ, textValue{Value: "qualifier"}}}},
					},
					Type: tpOrdered,
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
			Properties: map[xml.Name]Value{
				elemTest: textValue{
					Value: "test value",
				},
			},
		},
	},

	{
		desc: "simple text as property",
		in:   `<rdf:Description rdf:about="" test:prop="value"/>`,
		out: &Packet{
			Properties: map[xml.Name]Value{
				elemTest: textValue{Value: "value"},
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
			Properties: map[xml.Name]Value{
				elemTest: structValue{
					Value: map[xml.Name]Value{
						elemTestA: textValue{Value: "1"},
						elemTestB: textValue{Value: "2"},
						elemTestC: textValue{Value: "3"},
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
			Properties: map[xml.Name]Value{
				elemTest: structValue{
					Value: map[xml.Name]Value{
						elemTestA: textValue{Value: "1"},
						elemTestB: textValue{Value: "2"},
						elemTestC: textValue{Value: "3"},
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
			Properties: map[xml.Name]Value{
				elemTest: textValue{
					Value: "test value",
					Q: []Qualifier{
						{elemTestQ, textValue{Value: "qualifier"}},
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
			Properties: map[xml.Name]Value{
				elemTest: textValue{
					Value: "test value",
					Q: []Qualifier{
						{elemTestQ, textValue{Value: "qualifier"}},
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
			Properties: map[xml.Name]Value{
				elemTest: structValue{
					Value: map[xml.Name]Value{
						elemTestA: textValue{Value: "1"},
						elemTestB: textValue{Value: "2"},
						elemTestC: textValue{Value: "3"},
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
			Properties: map[xml.Name]Value{
				elemTest: textValue{
					Value: "test value",
					Q: []Qualifier{
						{elemTestQ, textValue{Value: "qualifier"}},
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
			Properties: map[xml.Name]Value{
				elemTest: structValue{
					Value: map[xml.Name]Value{
						elemTestA: textValue{Value: "1"},
						elemTestB: textValue{Value: "2"},
						elemTestC: textValue{Value: "3"},
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
			Properties: map[xml.Name]Value{
				elemTest: structValue{
					Value: map[xml.Name]Value{
						elemTestA: textValue{Value: "1"},
					},
					Q: Q{{
						Name:  attrRDFType,
						Value: uriValue{Value: &url.URL{Scheme: "http", Host: "ns.seehuhn.de", Path: "/test/", Fragment: "Type"}},
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
			Properties: map[xml.Name]Value{
				elemTest: structValue{
					Value: map[xml.Name]Value{
						elemTestA: textValue{Value: "1"},
					},
					Q: Q{{
						Name:  attrRDFType,
						Value: uriValue{Value: &url.URL{Scheme: "http", Host: "ns.seehuhn.de", Path: "/test/", Fragment: "Type"}},
					}},
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
			Properties: map[xml.Name]Value{
				{Space: "http://example.com", Local: "prop"}: structValue{
					Value: map[xml.Name]Value{
						{Space: "http://example.com", Local: "q"}: textValue{Value: ""},
					},
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
			if d := cmp.Diff(p, tc.out); d != "" {
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

		{attrRDFType, true}, // the only valid name in RDF namespace
		{xml.Name{Space: RDFNamespace, Local: "resource"}, false},
		{xml.Name{Space: RDFNamespace, Local: "p"}, false},
		{elemRDFValue, false},

		// all of the xml: namespace is forbidden
		{attrXMLLang, false},
		{xml.Name{Space: xmlNamespace, Local: "p"}, false},
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

		{attrRDFType, true}, // the only valid name in RDF namespace
		{xml.Name{Space: RDFNamespace, Local: "resource"}, false},
		{xml.Name{Space: RDFNamespace, Local: "q"}, false},
		{elemRDFValue, false},

		{attrXMLLang, true}, // the only valid name in XML namespace
		{xml.Name{Space: xmlNamespace, Local: "q"}, false},
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
			fmt.Println()
			fmt.Println(string(body))
			fmt.Println()
			fmt.Println(string(body2))
			fmt.Println()
			t.Fatalf("RoundTrip mismatch (-want +got):\n%s", d)
		}
	})
}
