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
			Properties: map[xml.Name]Raw{},
		},
		pattern: []string{"<rdf:Description rdf:about=\"\">"},
	},
	{
		desc: "with about URL",
		in: &Packet{
			Properties: map[xml.Name]Raw{},
			About:      testURL,
		},
		pattern: []string{"<rdf:Description rdf:about=\"http://example.com\">"},
	},
	{
		desc: "simple non-URI value",
		in: &Packet{
			Properties: map[xml.Name]Raw{
				elemTest: Text{V: "testvalue"},
			},
		},
		pattern: []string{"<test:prop>testvalue</test:prop>"},
	},
	{
		desc: "simple URI value",
		in: &Packet{
			Properties: map[xml.Name]Raw{
				elemTest: URL{V: testURL},
			},
		},
		pattern: []string{"<test:prop rdf:resource=\"http://example.com\"/>"},
	},
	{
		desc: "XML markup in text value",
		in: &Packet{
			Properties: map[xml.Name]Raw{
				elemTest: Text{V: "<b>test</b>"},
			},
		},
		pattern: []string{"<test:prop>&lt;b&gt;test&lt;/b&gt;</test:prop>"},
	},
	{
		desc: "structure value",
		in: &Packet{
			Properties: map[xml.Name]Raw{
				{Space: "http://ns.seehuhn.de/test/#", Local: "s"}: RawStruct{
					Value: map[xml.Name]Raw{
						elemTestA: Text{V: "1", Q: Q{{elemTestQ, Text{V: "q"}}}},
						elemTestB: Text{V: "2", Q: Q{{elemTestQ, Text{V: "q"}}}},
						elemTestC: Text{V: "3", Q: Q{{elemTestQ, Text{V: "q"}}}},
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
		pattern: []string{
			"<test:prop test:a=\"1\" test:b=\"2\" test:c=\"3\"/>",
		},
	},
	{
		desc: "empty structure ",
		in: &Packet{
			Properties: map[xml.Name]Raw{
				{Space: "http://ns.seehuhn.de/test/#", Local: "s"}: RawStruct{
					Value: map[xml.Name]Raw{},
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
			Properties: map[xml.Name]Raw{
				elemTest: Text{
					V: "testvalue",
					Q: Q{{Name: nameXMLLang, Value: Text{V: "de-DE"}}},
				},
			},
		},
		pattern: []string{"<test:prop xml:lang=\"de-DE\">testvalue</test:prop>"},
	},
	{
		desc: "xml:lang on URI value",
		in: &Packet{
			Properties: map[xml.Name]Raw{
				elemTest: URL{
					V: testURL,
					Q: Q{{Name: nameXMLLang, Value: Text{V: "de-DE"}}},
				},
			},
		},
		pattern: []string{"<test:prop xml:lang=\"de-DE\" rdf:resource=\"http://example.com\"/>"},
	},
	{
		desc: "xml:lang on structure field",
		in: &Packet{
			Properties: map[xml.Name]Raw{
				elemTest: RawStruct{
					Value: map[xml.Name]Raw{
						elemTestA: Text{
							V: "Hallo",
							Q: Q{{Name: nameXMLLang, Value: Text{V: "de"}}},
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
			Properties: map[xml.Name]Raw{
				elemTest: RawArray{
					Value: []Raw{
						Text{V: "a"},
						Text{
							V: "b",
							Q: Q{{Name: nameXMLLang, Value: Text{V: "fr"}}},
						},
						Text{V: "c"},
					},
					Kind: Ordered,
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
			Properties: map[xml.Name]Raw{
				elemTest: Text{
					V: "test value",
					Q: []Qualifier{
						{elemTestQ, URL{V: &url.URL{Scheme: "http", Host: "example.com"}}},
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
			Properties: map[xml.Name]Raw{
				elemTest: Text{
					V: "test value",
					Q: []Qualifier{
						{elemTestQ, Text{
							V: "qualifier",
							Q: []Qualifier{
								{nameXMLLang, Text{V: "te-ST"}},
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
			Properties: map[xml.Name]Raw{
				elemTest: URL{
					V: testURL,
					Q: []Qualifier{
						{nameXMLLang, Text{V: "te-ST"}},
						{elemTestQ, Text{V: "qualifier"}},
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
		opt := &PacketOptions{
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

			if d := cmp.Diff(tc.in, out, cmp.AllowUnexported(Packet{})); d != "" {
				t.Fatalf("RoundTrip mismatch (-want +got):\n%s", d)
			}
		})
	}
}
