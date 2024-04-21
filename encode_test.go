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
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var (
	elemTest  = xml.Name{Space: "http://ns.seehuhn.de/test/", Local: "prop"}
	elemTestA = xml.Name{Space: "http://ns.seehuhn.de/test/", Local: "a"}
	elemTestB = xml.Name{Space: "http://ns.seehuhn.de/test/", Local: "b"}
	elemTestC = xml.Name{Space: "http://ns.seehuhn.de/test/", Local: "c"}
	elemTestQ = xml.Name{Space: "http://ns.seehuhn.de/test/", Local: "q"}

	testURL = &url.URL{Scheme: "http", Host: "example.com"}
)

type testCase struct {
	desc    string
	in      *Packet
	pattern []string
}

var testCases = []testCase{
	{
		desc: "simple non-URI value",
		in: &Packet{
			Properties: map[xml.Name]Value{
				elemTest: textValue{Value: "testvalue"},
			},
		},
		pattern: []string{"<test:prop>testvalue</test:prop>"},
	},
	{
		desc: "simple URI value",
		in: &Packet{
			Properties: map[xml.Name]Value{
				elemTest: uriValue{Value: testURL},
			},
		},
		pattern: []string{"<test:prop rdf:resource=\"http://example.com\"/>"},
	},
	{
		desc: "XML markup in text value",
		in: &Packet{
			Properties: map[xml.Name]Value{
				elemTest: textValue{Value: "<b>test</b>"},
			},
		},
		pattern: []string{"<test:prop>&lt;b&gt;test&lt;/b&gt;</test:prop>"},
	},
	{
		desc: "struct value",
		in: &Packet{
			Properties: map[xml.Name]Value{
				{Space: "http://ns.seehuhn.de/test/", Local: "s"}: structValue{
					Value: map[xml.Name]Value{
						elemTestA: textValue{Value: "1", Q: Q{{elemTestQ, textValue{Value: "q"}}}},
						elemTestB: textValue{Value: "2", Q: Q{{elemTestQ, textValue{Value: "q"}}}},
						elemTestC: textValue{Value: "3", Q: Q{{elemTestQ, textValue{Value: "q"}}}},
					},
				},
			},
		},
		pattern: []string{
			"<test:s rdf:parseType=\"Resource\">",
			"<test:a rdf:parseType=\"Resource\">",
			"<rdf:value>1</rdf:value>",
			"<test:q>q</test:q>",
			"</test:a>",
			"<test:b rdf:parseType=\"Resource\">",
			"<rdf:value>2</rdf:value>",
			"<test:q>q</test:q>",
			"</test:b>",
			"<test:c rdf:parseType=\"Resource\">",
			"<rdf:value>3</rdf:value>",
			"<test:q>q</test:q>",
			"</test:c>",
			"</test:s>",
		},
	},
	{
		desc: "xml:lang on property",
		in: &Packet{
			Properties: map[xml.Name]Value{
				elemTest: textValue{
					Value: "testvalue",
					Q:     Q{{Name: attrXMLLang, Value: textValue{Value: "de-DE"}}},
				},
			},
		},
		pattern: []string{"<test:prop xml:lang=\"de-DE\">testvalue</test:prop>"},
	},
	{
		desc: "xml:lang on URI value",
		in: &Packet{
			Properties: map[xml.Name]Value{
				elemTest: uriValue{
					Value: testURL,
					Q:     Q{{Name: attrXMLLang, Value: textValue{Value: "de-DE"}}},
				},
			},
		},
		pattern: []string{"<test:prop xml:lang=\"de-DE\" rdf:resource=\"http://example.com\"/>"},
	},
	{
		desc: "xml:lang on structure field",
		in: &Packet{
			Properties: map[xml.Name]Value{
				elemTest: structValue{
					Value: map[xml.Name]Value{
						elemTestA: textValue{
							Value: "Hallo",
							Q:     Q{{Name: attrXMLLang, Value: textValue{Value: "de"}}},
						},
					},
				},
			},
		},
		pattern: []string{
			"<test:prop rdf:parseType=\"Resource\">",
			"<test:a xml:lang=\"de\">Hallo</test:a>",
			"</test:prop>",
		},
	},
	{
		desc: "xml:lang on array item",
		in: &Packet{
			Properties: map[xml.Name]Value{
				elemTest: arrayValue{
					Value: []Value{
						textValue{Value: "a"},
						textValue{
							Value: "b",
							Q:     Q{{Name: attrXMLLang, Value: textValue{Value: "fr"}}},
						},
						textValue{Value: "c"},
					},
					Type: tpOrdered,
				},
			},
		},
		pattern: []string{
			"<test:prop>",
			"<rdf:Seq>",
			"<rdf:li>a</rdf:li>",
			"<rdf:li xml:lang=\"fr\">b</rdf:li>",
			"<rdf:li>c</rdf:li>",
			"</rdf:Seq>",
			"</test:prop>",
		},
	},
	{
		desc: "general qualfiers",
		in: &Packet{
			Properties: map[xml.Name]Value{
				elemTest: textValue{
					Value: "test value",
					Q: []Qualifier{
						{elemTestQ, textValue{Value: "qualifier"}},
					},
				},
			},
		},
		pattern: []string{
			"<test:prop rdf:parseType=\"Resource\">",
			"<rdf:value>test value</rdf:value>",
			"<test:q>qualifier</test:q>",
			"</test:prop>",
		},
	},
	{
		desc: "xml:lang on qualifier value",
		in: &Packet{
			Properties: map[xml.Name]Value{
				elemTest: textValue{
					Value: "test value",
					Q: []Qualifier{
						{elemTestQ, textValue{
							Value: "qualifier",
							Q: []Qualifier{
								{attrXMLLang, textValue{Value: "te-ST"}},
							},
						}},
					},
				},
			},
		},
		pattern: []string{
			"<test:prop rdf:parseType=\"Resource\">",
			"<rdf:value>test value</rdf:value>",
			"<test:q xml:lang=\"te-ST\">qualifier</test:q>",
			"</test:prop>",
		},
	},
	{
		desc: "general qualfiers on URI value",
		in: &Packet{
			Properties: map[xml.Name]Value{
				elemTest: uriValue{
					Value: testURL,
					Q: []Qualifier{
						{attrXMLLang, textValue{Value: "te-ST"}},
						{elemTestQ, textValue{Value: "qualifier"}},
					},
				},
			},
		},
		pattern: []string{
			"<test:prop xml:lang=\"te-ST\" rdf:parseType=\"Resource\">",
			"<rdf:value rdf:resource=\"http://example.com\"/>",
			"<test:q>qualifier</test:q>",
			"</test:prop>",
		},
	},
}

func TestRoundTrip(t *testing.T) {
	for i, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			body, err := tc.in.Encode(true)
			if err != nil {
				t.Fatal(err)
			}

			bodyString := string(body)
			fmt.Println(bodyString)

			var parts []string
			for _, p := range tc.pattern {
				parts = append(parts, regexp.QuoteMeta(p))
			}
			pat := regexp.MustCompile(strings.Join(parts, `\s*`))

			if pat.FindString(bodyString) == "" {
				t.Fatalf("%d: wrong encoding: want\n%q", i, tc.pattern)
			}

			out, err := Read(bytes.NewReader(body))
			if err != nil {
				t.Fatal(err)
			}

			if d := cmp.Diff(tc.in, out); d != "" {
				t.Fatalf("RoundTrip mismatch (-want +got):\n%s", d)
			}
		})
	}
}
