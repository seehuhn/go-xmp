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
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var (
	elemTest  = xml.Name{Space: "http://ns.seehuhn.de/test/#", Local: "prop"}
	elemTestA = xml.Name{Space: "http://ns.seehuhn.de/test/#", Local: "a"}
	elemTestB = xml.Name{Space: "http://ns.seehuhn.de/test/#", Local: "b"}
	elemTestC = xml.Name{Space: "http://ns.seehuhn.de/test/#", Local: "c"}
	elemTestQ = xml.Name{Space: "http://ns.seehuhn.de/test/#", Local: "q"}

	testURL = &url.URL{Scheme: "http", Host: "example.com"}
)

type encodeTestCase struct {
	desc    string
	in      *Packet
	pattern []string
}

var encodeTestCases = []encodeTestCase{
	{
		desc: "without about URL",
		in: &Packet{
			Properties: map[xml.Name]Value{},
		},
		pattern: []string{"<rdf:Description rdf:about=\"\">"},
	},
	{
		desc: "with about URL",
		in: &Packet{
			Properties: map[xml.Name]Value{},
			About:      testURL,
		},
		pattern: []string{"<rdf:Description rdf:about=\"http://example.com\">"},
	},
	{
		desc: "simple non-URI value",
		in: &Packet{
			Properties: map[xml.Name]Value{
				elemTest: TextValue{Value: "testvalue"},
			},
		},
		pattern: []string{"<test:prop>testvalue</test:prop>"},
	},
	{
		desc: "simple URI value",
		in: &Packet{
			Properties: map[xml.Name]Value{
				elemTest: URIValue{Value: testURL},
			},
		},
		pattern: []string{"<test:prop rdf:resource=\"http://example.com\"/>"},
	},
	{
		desc: "XML markup in text value",
		in: &Packet{
			Properties: map[xml.Name]Value{
				elemTest: TextValue{Value: "<b>test</b>"},
			},
		},
		pattern: []string{"<test:prop>&lt;b&gt;test&lt;/b&gt;</test:prop>"},
	},
	{
		desc: "structure value",
		in: &Packet{
			Properties: map[xml.Name]Value{
				{Space: "http://ns.seehuhn.de/test/#", Local: "s"}: StructValue{
					Value: map[xml.Name]Value{
						elemTestA: TextValue{Value: "1", Q: Q{{elemTestQ, TextValue{Value: "q"}}}},
						elemTestB: TextValue{Value: "2", Q: Q{{elemTestQ, TextValue{Value: "q"}}}},
						elemTestC: TextValue{Value: "3", Q: Q{{elemTestQ, TextValue{Value: "q"}}}},
					},
				},
			},
		},
		pattern: []string{
			"<test:s rdf:parseType=\"Resource\">",
			"<test:a test:q=\"q\" rdf:value=\"1\"/>",
			"<test:b test:q=\"q\" rdf:value=\"2\"/>",
			"<test:c test:q=\"q\" rdf:value=\"3\"/>",
			"</test:s>",
		},
	},
	{
		desc: "compact struct",
		in: &Packet{
			Properties: map[xml.Name]Value{
				elemTest: StructValue{
					Value: map[xml.Name]Value{
						elemTestA: TextValue{Value: "1"},
						elemTestB: TextValue{Value: "2"},
						elemTestC: TextValue{Value: "3"},
					},
				},
			},
		},
		pattern: []string{
			"<test:prop test:a=\"1\" test:b=\"2\" test:c=\"3\"/>",
		},
	},
	{
		desc: "empty structure ",
		in: &Packet{
			Properties: map[xml.Name]Value{
				{Space: "http://ns.seehuhn.de/test/#", Local: "s"}: StructValue{
					Value: map[xml.Name]Value{},
				},
			},
		},
		pattern: []string{
			"<test:s rdf:parseType=\"Resource\">",
			"</test:s>",
		},
	},
	{
		desc: "xml:lang on property",
		in: &Packet{
			Properties: map[xml.Name]Value{
				elemTest: TextValue{
					Value: "testvalue",
					Q:     Q{{Name: nameXMLLang, Value: TextValue{Value: "de-DE"}}},
				},
			},
		},
		pattern: []string{"<test:prop xml:lang=\"de-DE\">testvalue</test:prop>"},
	},
	{
		desc: "xml:lang on URI value",
		in: &Packet{
			Properties: map[xml.Name]Value{
				elemTest: URIValue{
					Value: testURL,
					Q:     Q{{Name: nameXMLLang, Value: TextValue{Value: "de-DE"}}},
				},
			},
		},
		pattern: []string{"<test:prop xml:lang=\"de-DE\" rdf:resource=\"http://example.com\"/>"},
	},
	{
		desc: "xml:lang on structure field",
		in: &Packet{
			Properties: map[xml.Name]Value{
				elemTest: StructValue{
					Value: map[xml.Name]Value{
						elemTestA: TextValue{
							Value: "Hallo",
							Q:     Q{{Name: nameXMLLang, Value: TextValue{Value: "de"}}},
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
				elemTest: ArrayValue{
					Value: []Value{
						TextValue{Value: "a"},
						TextValue{
							Value: "b",
							Q:     Q{{Name: nameXMLLang, Value: TextValue{Value: "fr"}}},
						},
						TextValue{Value: "c"},
					},
					Type: Ordered,
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
				elemTest: TextValue{
					Value: "test value",
					Q: []Qualifier{
						{elemTestQ, URIValue{Value: &url.URL{Scheme: "http", Host: "example.com"}}},
					},
				},
			},
		},
		pattern: []string{
			"<test:prop rdf:parseType=\"Resource\">",
			"<rdf:value>test value</rdf:value>",
			"<test:q rdf:resource=\"http://example.com\"/>",
			"</test:prop>",
		},
	},
	{
		desc: "xml:lang on qualifier value",
		in: &Packet{
			Properties: map[xml.Name]Value{
				elemTest: TextValue{
					Value: "test value",
					Q: []Qualifier{
						{elemTestQ, TextValue{
							Value: "qualifier",
							Q: []Qualifier{
								{nameXMLLang, TextValue{Value: "te-ST"}},
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
				elemTest: URIValue{
					Value: testURL,
					Q: []Qualifier{
						{nameXMLLang, TextValue{Value: "te-ST"}},
						{elemTestQ, TextValue{Value: "qualifier"}},
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
	for i, tc := range encodeTestCases {
		opt := &WriterOptions{
			Pretty: true,
		}
		t.Run(tc.desc, func(t *testing.T) {
			buf := &bytes.Buffer{}
			err := tc.in.Write(buf, opt)
			if err != nil {
				t.Fatal(err)
			}

			bodyString := buf.String()
			// fmt.Println(bodyString)

			var parts []string
			for _, p := range tc.pattern {
				parts = append(parts, regexp.QuoteMeta(p))
			}
			pat := regexp.MustCompile(strings.Join(parts, `\s*`))

			if pat.FindString(bodyString) == "" {
				t.Fatalf("%d: wrong encoding: want\n%q", i, tc.pattern)
			}

			out, err := Read(strings.NewReader(bodyString))
			if err != nil {
				t.Fatal(err)
			}

			if d := cmp.Diff(tc.in, out); d != "" {
				t.Fatalf("RoundTrip mismatch (-want +got):\n%s", d)
			}
		})
	}
}
