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
				{Space: "http://ns.seehuhn.de/test/", Local: "testname"}: textValue{Value: "testvalue"},
			},
		},
		pattern: []string{"<test:testname>testvalue</test:testname>"},
	},
	{
		desc: "simple URI value",
		in: &Packet{
			Properties: map[xml.Name]Value{
				{Space: "http://ns.seehuhn.de/test/", Local: "testname"}: uriValue{Value: &url.URL{Scheme: "http", Host: "example.com"}},
			},
		},
		pattern: []string{"<test:testname rdf:resource=\"http://example.com\"/>"},
	},
	{
		desc: "XML markup in text value",
		in: &Packet{
			Properties: map[xml.Name]Value{
				{Space: "http://ns.seehuhn.de/test/", Local: "testname"}: textValue{Value: "<b>test</b>"},
			},
		},
		pattern: []string{"<test:testname>&lt;b&gt;test&lt;/b&gt;</test:testname>"},
	},
	{
		desc: "struct value",
		in: &Packet{
			Properties: map[xml.Name]Value{
				{Space: "http://ns.seehuhn.de/test/", Local: "s"}: structValue{
					Value: map[xml.Name]Value{
						{Space: "http://ns.seehuhn.de/test/", Local: "a"}: textValue{Value: "1"},
						{Space: "http://ns.seehuhn.de/test/", Local: "b"}: textValue{Value: "2"},
						{Space: "http://ns.seehuhn.de/test/", Local: "c"}: textValue{Value: "3"},
					},
				},
			},
		},
		pattern: []string{
			"<test:s>",
			"<rdf:Description>",
			"<test:a>1</test:a>",
			"<test:b>2</test:b>",
			"<test:c>3</test:c>",
			"</rdf:Description>",
			"</test:s>",
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
				t.Fatalf("missing property %q in test case %d", tc.pattern, i)
			}

			p2, err := Read(bytes.NewReader(body))
			if err != nil {
				t.Fatal(err)
			}

			if d := cmp.Diff(tc.in, p2); d != "" {
				t.Fatalf("RoundTrip mismatch (-want +got):\n%s", d)
			}
		})
	}
}
