// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jvxml

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
	"testing"
)

func TestMarshalFlush(t *testing.T) {
	var buf strings.Builder
	enc := NewEncoder(&buf)
	if err := enc.EncodeToken(xml.CharData("hello world")); err != nil {
		t.Fatalf("enc.EncodeToken: %v", err)
	}
	if buf.Len() > 0 {
		t.Fatalf("enc.EncodeToken caused write: %q", buf.String())
	}
	if err := enc.Flush(); err != nil {
		t.Fatalf("enc.Flush: %v", err)
	}
	if buf.String() != "hello world" {
		t.Fatalf("after enc.Flush, buf.String() = %q, want %q", buf.String(), "hello world")
	}
}

var encodeTokenTests = []struct {
	desc string
	toks []Token
	want string
	err  string
}{{
	desc: "start element with name space",
	toks: []Token{
		xml.StartElement{xml.Name{"space", "local"}, nil},
	},
	want: `<local xmlns="space">`,
}, {
	desc: "start element with no name",
	toks: []Token{
		xml.StartElement{xml.Name{"space", ""}, nil},
	},
	err: "xml: start tag with no name",
}, {
	desc: "end element with no name",
	toks: []Token{
		xml.EndElement{xml.Name{"space", ""}},
	},
	err: "xml: end tag with no name",
}, {
	desc: "char data",
	toks: []Token{
		xml.CharData("foo"),
	},
	want: `foo`,
}, {
	desc: "char data with escaped chars",
	toks: []Token{
		xml.CharData(" \t\n"),
	},
	want: " &#x9;\n",
}, {
	desc: "comment",
	toks: []Token{
		xml.Comment("foo"),
	},
	want: `<!--foo-->`,
}, {
	desc: "comment with invalid content",
	toks: []Token{
		xml.Comment("foo-->"),
	},
	err: "xml: EncodeToken of Comment containing --> marker",
}, {
	desc: "proc instruction",
	toks: []Token{
		xml.ProcInst{"Target", []byte("Instruction")},
	},
	want: `<?Target Instruction?>`,
}, {
	desc: "proc instruction with empty target",
	toks: []Token{
		xml.ProcInst{"", []byte("Instruction")},
	},
	err: "xml: EncodeToken of ProcInst with invalid Target",
}, {
	desc: "proc instruction with bad content",
	toks: []Token{
		xml.ProcInst{"", []byte("Instruction?>")},
	},
	err: "xml: EncodeToken of ProcInst with invalid Target",
}, {
	desc: "directive",
	toks: []Token{
		xml.Directive("foo"),
	},
	want: `<!foo>`,
}, {
	desc: "more complex directive",
	toks: []Token{
		xml.Directive("DOCTYPE doc [ <!ELEMENT doc '>'> <!-- com>ment --> ]"),
	},
	want: `<!DOCTYPE doc [ <!ELEMENT doc '>'> <!-- com>ment --> ]>`,
}, {
	desc: "directive instruction with bad name",
	toks: []Token{
		xml.Directive("foo>"),
	},
	err: "xml: EncodeToken of Directive containing wrong < or > markers",
}, {
	desc: "end tag without start tag",
	toks: []Token{
		xml.EndElement{xml.Name{"foo", "bar"}},
	},
	err: "xml: end tag </bar> without start tag",
}, {
	desc: "mismatching end tag local name",
	toks: []Token{
		xml.StartElement{xml.Name{"", "foo"}, nil},
		xml.EndElement{xml.Name{"", "bar"}},
	},
	err:  "xml: end tag </bar> does not match start tag <foo>",
	want: `<foo>`,
}, {
	desc: "mismatching end tag namespace",
	toks: []Token{
		xml.StartElement{xml.Name{"space", "foo"}, nil},
		xml.EndElement{xml.Name{"another", "foo"}},
	},
	err:  "xml: end tag </foo> in namespace another does not match start tag <foo> in namespace space",
	want: `<foo xmlns="space">`,
}, {
	desc: "start element with explicit namespace",
	toks: []Token{
		xml.StartElement{xml.Name{"space", "local"}, []xml.Attr{
			{xml.Name{"xmlns", "x"}, "space"},
			{xml.Name{"space", "foo"}, "value"},
		}},
	},
	want: `<local xmlns="space" xmlns:_xmlns="xmlns" _xmlns:x="space" xmlns:space="space" space:foo="value">`,
}, {
	desc: "start element with explicit namespace and colliding prefix",
	toks: []Token{
		xml.StartElement{xml.Name{"space", "local"}, []xml.Attr{
			{xml.Name{"xmlns", "x"}, "space"},
			{xml.Name{"space", "foo"}, "value"},
			{xml.Name{"x", "bar"}, "other"},
		}},
	},
	want: `<local xmlns="space" xmlns:_xmlns="xmlns" _xmlns:x="space" xmlns:space="space" space:foo="value" xmlns:x="x" x:bar="other">`,
}, {
	desc: "start element using previously defined namespace",
	toks: []Token{
		xml.StartElement{xml.Name{"", "local"}, []xml.Attr{
			{xml.Name{"xmlns", "x"}, "space"},
		}},
		xml.StartElement{xml.Name{"space", "foo"}, []xml.Attr{
			{xml.Name{"space", "x"}, "y"},
		}},
	},
	want: `<local xmlns:_xmlns="xmlns" _xmlns:x="space"><foo xmlns="space" xmlns:space="space" space:x="y">`,
}, {
	desc: "nested name space with same prefix",
	toks: []Token{
		xml.StartElement{xml.Name{"", "foo"}, []xml.Attr{
			{xml.Name{"xmlns", "x"}, "space1"},
		}},
		xml.StartElement{xml.Name{"", "foo"}, []xml.Attr{
			{xml.Name{"xmlns", "x"}, "space2"},
		}},
		xml.StartElement{xml.Name{"", "foo"}, []xml.Attr{
			{xml.Name{"space1", "a"}, "space1 value"},
			{xml.Name{"space2", "b"}, "space2 value"},
		}},
		xml.EndElement{xml.Name{"", "foo"}},
		xml.EndElement{xml.Name{"", "foo"}},
		xml.StartElement{xml.Name{"", "foo"}, []xml.Attr{
			{xml.Name{"space1", "a"}, "space1 value"},
			{xml.Name{"space2", "b"}, "space2 value"},
		}},
	},
	want: `<foo xmlns:_xmlns="xmlns" _xmlns:x="space1"><foo _xmlns:x="space2"><foo xmlns:space1="space1" space1:a="space1 value" xmlns:space2="space2" space2:b="space2 value"></foo></foo><foo xmlns:space1="space1" space1:a="space1 value" xmlns:space2="space2" space2:b="space2 value">`,
}, {
	desc: "start element defining several prefixes for the same name space",
	toks: []Token{
		xml.StartElement{xml.Name{"space", "foo"}, []xml.Attr{
			{xml.Name{"xmlns", "a"}, "space"},
			{xml.Name{"xmlns", "b"}, "space"},
			{xml.Name{"space", "x"}, "value"},
		}},
	},
	want: `<foo xmlns="space" xmlns:_xmlns="xmlns" _xmlns:a="space" _xmlns:b="space" xmlns:space="space" space:x="value">`,
}, {
	desc: "nested element redefines name space",
	toks: []Token{
		xml.StartElement{xml.Name{"", "foo"}, []xml.Attr{
			{xml.Name{"xmlns", "x"}, "space"},
		}},
		xml.StartElement{xml.Name{"space", "foo"}, []xml.Attr{
			{xml.Name{"xmlns", "y"}, "space"},
			{xml.Name{"space", "a"}, "value"},
		}},
	},
	want: `<foo xmlns:_xmlns="xmlns" _xmlns:x="space"><foo xmlns="space" _xmlns:y="space" xmlns:space="space" space:a="value">`,
}, {
	desc: "nested element creates alias for default name space",
	toks: []Token{
		xml.StartElement{xml.Name{"space", "foo"}, []xml.Attr{
			{xml.Name{"", "xmlns"}, "space"},
		}},
		xml.StartElement{xml.Name{"space", "foo"}, []xml.Attr{
			{xml.Name{"xmlns", "y"}, "space"},
			{xml.Name{"space", "a"}, "value"},
		}},
	},
	want: `<foo xmlns="space" xmlns="space"><foo xmlns="space" xmlns:_xmlns="xmlns" _xmlns:y="space" xmlns:space="space" space:a="value">`,
}, {
	desc: "nested element defines default name space with existing prefix",
	toks: []Token{
		xml.StartElement{xml.Name{"", "foo"}, []xml.Attr{
			{xml.Name{"xmlns", "x"}, "space"},
		}},
		xml.StartElement{xml.Name{"space", "foo"}, []xml.Attr{
			{xml.Name{"", "xmlns"}, "space"},
			{xml.Name{"space", "a"}, "value"},
		}},
	},
	want: `<foo xmlns:_xmlns="xmlns" _xmlns:x="space"><foo xmlns="space" xmlns="space" xmlns:space="space" space:a="value">`,
}, {
	desc: "nested element uses empty attribute name space when default ns defined",
	toks: []Token{
		xml.StartElement{xml.Name{"space", "foo"}, []xml.Attr{
			{xml.Name{"", "xmlns"}, "space"},
		}},
		xml.StartElement{xml.Name{"space", "foo"}, []xml.Attr{
			{xml.Name{"", "attr"}, "value"},
		}},
	},
	want: `<foo xmlns="space" xmlns="space"><foo xmlns="space" attr="value">`,
}, {
	desc: "redefine xmlns",
	toks: []Token{
		xml.StartElement{xml.Name{"", "foo"}, []xml.Attr{
			{xml.Name{"foo", "xmlns"}, "space"},
		}},
	},
	want: `<foo xmlns:foo="foo" foo:xmlns="space">`,
}, {
	desc: "xmlns with explicit name space #1",
	toks: []Token{
		xml.StartElement{xml.Name{"space", "foo"}, []xml.Attr{
			{xml.Name{"xml", "xmlns"}, "space"},
		}},
	},
	want: `<foo xmlns="space" xmlns:_xml="xml" _xml:xmlns="space">`,
}, {
	desc: "xmlns with explicit name space #2",
	toks: []Token{
		xml.StartElement{xml.Name{"space", "foo"}, []xml.Attr{
			{xml.Name{xmlURL, "xmlns"}, "space"},
		}},
	},
	want: `<foo xmlns="space" xml:xmlns="space">`,
}, {
	desc: "empty name space declaration is ignored",
	toks: []Token{
		xml.StartElement{xml.Name{"", "foo"}, []xml.Attr{
			{xml.Name{"xmlns", "foo"}, ""},
		}},
	},
	want: `<foo xmlns:_xmlns="xmlns" _xmlns:foo="">`,
}, {
	desc: "attribute with no name is ignored",
	toks: []Token{
		xml.StartElement{xml.Name{"", "foo"}, []xml.Attr{
			{xml.Name{"", ""}, "value"},
		}},
	},
	want: `<foo>`,
}, {
	desc: "namespace URL with non-valid name",
	toks: []Token{
		xml.StartElement{xml.Name{"/34", "foo"}, []xml.Attr{
			{xml.Name{"/34", "x"}, "value"},
		}},
	},
	want: `<foo xmlns="/34" xmlns:_="/34" _:x="value">`,
}, {
	desc: "nested element resets default namespace to empty",
	toks: []Token{
		xml.StartElement{xml.Name{"space", "foo"}, []xml.Attr{
			{xml.Name{"", "xmlns"}, "space"},
		}},
		xml.StartElement{xml.Name{"", "foo"}, []xml.Attr{
			{xml.Name{"", "xmlns"}, ""},
			{xml.Name{"", "x"}, "value"},
			{xml.Name{"space", "x"}, "value"},
		}},
	},
	want: `<foo xmlns="space" xmlns="space"><foo xmlns="" x="value" xmlns:space="space" space:x="value">`,
}, {
	desc: "nested element requires empty default name space",
	toks: []Token{
		xml.StartElement{xml.Name{"space", "foo"}, []xml.Attr{
			{xml.Name{"", "xmlns"}, "space"},
		}},
		xml.StartElement{xml.Name{"", "foo"}, nil},
	},
	want: `<foo xmlns="space" xmlns="space"><foo>`,
}, {
	desc: "attribute uses name space from xmlns",
	toks: []Token{
		xml.StartElement{xml.Name{"some/space", "foo"}, []xml.Attr{
			{xml.Name{"", "attr"}, "value"},
			{xml.Name{"some/space", "other"}, "other value"},
		}},
	},
	want: `<foo xmlns="some/space" attr="value" xmlns:space="some/space" space:other="other value">`,
}, {
	desc: "default name space should not be used by attributes",
	toks: []Token{
		xml.StartElement{xml.Name{"space", "foo"}, []xml.Attr{
			{xml.Name{"", "xmlns"}, "space"},
			{xml.Name{"xmlns", "bar"}, "space"},
			{xml.Name{"space", "baz"}, "foo"},
		}},
		xml.StartElement{xml.Name{"space", "baz"}, nil},
		xml.EndElement{xml.Name{"space", "baz"}},
		xml.EndElement{xml.Name{"space", "foo"}},
	},
	want: `<foo xmlns="space" xmlns="space" xmlns:_xmlns="xmlns" _xmlns:bar="space" xmlns:space="space" space:baz="foo"><baz xmlns="space"></baz></foo>`,
}, {
	desc: "default name space not used by attributes, not explicitly defined",
	toks: []Token{
		xml.StartElement{xml.Name{"space", "foo"}, []xml.Attr{
			{xml.Name{"", "xmlns"}, "space"},
			{xml.Name{"space", "baz"}, "foo"},
		}},
		xml.StartElement{xml.Name{"space", "baz"}, nil},
		xml.EndElement{xml.Name{"space", "baz"}},
		xml.EndElement{xml.Name{"space", "foo"}},
	},
	want: `<foo xmlns="space" xmlns="space" xmlns:space="space" space:baz="foo"><baz xmlns="space"></baz></foo>`,
}, {
	desc: "impossible xmlns declaration",
	toks: []Token{
		xml.StartElement{xml.Name{"", "foo"}, []xml.Attr{
			{xml.Name{"", "xmlns"}, "space"},
		}},
		xml.StartElement{xml.Name{"space", "bar"}, []xml.Attr{
			{xml.Name{"space", "attr"}, "value"},
		}},
	},
	want: `<foo xmlns="space"><bar xmlns="space" xmlns:space="space" space:attr="value">`,
}, {
	desc: "reserved namespace prefix -- all lower case",
	toks: []Token{
		xml.StartElement{xml.Name{"", "foo"}, []xml.Attr{
			{xml.Name{"http://www.w3.org/2001/xmlSchema-instance", "nil"}, "true"},
		}},
	},
	want: `<foo xmlns:_xmlSchema-instance="http://www.w3.org/2001/xmlSchema-instance" _xmlSchema-instance:nil="true">`,
}, {
	desc: "reserved namespace prefix -- all upper case",
	toks: []Token{
		xml.StartElement{xml.Name{"", "foo"}, []xml.Attr{
			{xml.Name{"http://www.w3.org/2001/XMLSchema-instance", "nil"}, "true"},
		}},
	},
	want: `<foo xmlns:_XMLSchema-instance="http://www.w3.org/2001/XMLSchema-instance" _XMLSchema-instance:nil="true">`,
}, {
	desc: "reserved namespace prefix -- all mixed case",
	toks: []Token{
		xml.StartElement{xml.Name{"", "foo"}, []xml.Attr{
			{xml.Name{"http://www.w3.org/2001/XmLSchema-instance", "nil"}, "true"},
		}},
	},
	want: `<foo xmlns:_XmLSchema-instance="http://www.w3.org/2001/XmLSchema-instance" _XmLSchema-instance:nil="true">`,
}, {
	desc: "empty element",
	toks: []Token{
		EmptyElement{xml.Name{"", "foo"}, nil},
	},
	want: "<foo/>",
}, {
	desc: "empty element",
	toks: []Token{
		EmptyElement{xml.Name{"", "xmp:BaseURL"}, []xml.Attr{
			{xml.Name{"", "rdf:resource"}, "http://www.example.com/"},
		}},
	},
	want: "<xmp:BaseURL rdf:resource=\"http://www.example.com/\"/>",
}}

func TestEncodeToken(t *testing.T) {
loop:
	for i, tt := range encodeTokenTests {
		var buf strings.Builder
		enc := NewEncoder(&buf)
		var err error
		for j, tok := range tt.toks {
			err = enc.EncodeToken(tok)
			if err != nil && j < len(tt.toks)-1 {
				t.Errorf("#%d %s token #%d: %v", i, tt.desc, j, err)
				continue loop
			}
		}
		errorf := func(f string, a ...any) {
			t.Errorf("#%d %s token #%d:%s", i, tt.desc, len(tt.toks)-1, fmt.Sprintf(f, a...))
		}
		switch {
		case tt.err != "" && err == nil:
			errorf(" expected error; got none")
			continue
		case tt.err == "" && err != nil:
			errorf(" got error: %v", err)
			continue
		case tt.err != "" && err != nil && tt.err != err.Error():
			errorf(" error mismatch; got %v, want %v", err, tt.err)
			continue
		}
		if err := enc.Flush(); err != nil {
			errorf(" %v", err)
			continue
		}
		if got := buf.String(); got != tt.want {
			errorf("\ngot  %v\nwant %v", got, tt.want)
			continue
		}
	}
}

func TestProcInstEncodeToken(t *testing.T) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf)

	if err := enc.EncodeToken(xml.ProcInst{"xml", []byte("Instruction")}); err != nil {
		t.Fatalf("enc.EncodeToken: expected to be able to encode xml target ProcInst as first token, %s", err)
	}

	if err := enc.EncodeToken(xml.ProcInst{"Target", []byte("Instruction")}); err != nil {
		t.Fatalf("enc.EncodeToken: expected to be able to add non-xml target ProcInst")
	}

	if err := enc.EncodeToken(xml.ProcInst{"xml", []byte("Instruction")}); err == nil {
		t.Fatalf("enc.EncodeToken: expected to not be allowed to encode xml target ProcInst when not first token")
	}
}

func TestDecodeEncode(t *testing.T) {
	var in, out bytes.Buffer
	in.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<?Target Instruction?>
<root>
</root>
`)
	dec := xml.NewDecoder(&in)
	enc := NewEncoder(&out)
	for tok, err := dec.Token(); err == nil; tok, err = dec.Token() {
		err = enc.EncodeToken(tok)
		if err != nil {
			t.Fatalf("enc.EncodeToken: Unable to encode token (%#v), %v", tok, err)
		}
	}
}

func TestIsValidDirective(t *testing.T) {
	testOK := []string{
		"<>",
		"< < > >",
		"<!DOCTYPE '<' '>' '>' <!--nothing-->>",
		"<!DOCTYPE doc [ <!ELEMENT doc ANY> <!ELEMENT doc ANY> ]>",
		"<!DOCTYPE doc [ <!ELEMENT doc \"ANY> '<' <!E\" LEMENT '>' doc ANY> ]>",
		"<!DOCTYPE doc <!-- just>>>> a < comment --> [ <!ITEM anything> ] >",
	}
	testKO := []string{
		"<",
		">",
		"<!--",
		"-->",
		"< > > < < >",
		"<!dummy <!-- > -->",
		"<!DOCTYPE doc '>",
		"<!DOCTYPE doc '>'",
		"<!DOCTYPE doc <!--comment>",
	}
	for _, s := range testOK {
		if !isValidDirective(xml.Directive(s)) {
			t.Errorf("Directive %q is expected to be valid", s)
		}
	}
	for _, s := range testKO {
		if isValidDirective(xml.Directive(s)) {
			t.Errorf("Directive %q is expected to be invalid", s)
		}
	}
}

// Issue 11719. EncodeToken used to silently eat tokens with an invalid type.
func TestSimpleUseOfEncodeToken(t *testing.T) {
	var buf strings.Builder
	enc := NewEncoder(&buf)
	if err := enc.EncodeToken(&xml.StartElement{Name: xml.Name{"", "object1"}}); err == nil {
		t.Errorf("enc.EncodeToken: pointer type should be rejected")
	}
	if err := enc.EncodeToken(&xml.EndElement{Name: xml.Name{"", "object1"}}); err == nil {
		t.Errorf("enc.EncodeToken: pointer type should be rejected")
	}
	if err := enc.EncodeToken(xml.StartElement{Name: xml.Name{"", "object2"}}); err != nil {
		t.Errorf("enc.EncodeToken: StartElement %s", err)
	}
	if err := enc.EncodeToken(xml.EndElement{Name: xml.Name{"", "object2"}}); err != nil {
		t.Errorf("enc.EncodeToken: EndElement %s", err)
	}
	if err := enc.Flush(); err != nil {
		t.Errorf("enc.Flush: %s", err)
	}
	if buf.Len() == 0 {
		t.Errorf("enc.EncodeToken: empty buffer")
	}
	want := "<object2></object2>"
	if buf.String() != want {
		t.Errorf("enc.EncodeToken: expected %q; got %q", want, buf.String())
	}
}

var closeTests = []struct {
	desc string
	toks []Token
	want string
	err  string
}{{
	desc: "unclosed start element",
	toks: []Token{
		xml.StartElement{xml.Name{"", "foo"}, nil},
	},
	want: `<foo>`,
	err:  "unclosed tag <foo>",
}, {
	desc: "closed element",
	toks: []Token{
		xml.StartElement{xml.Name{"", "foo"}, nil},
		xml.EndElement{xml.Name{"", "foo"}},
	},
	want: `<foo></foo>`,
}, {
	desc: "directive",
	toks: []Token{
		xml.Directive("foo"),
	},
	want: `<!foo>`,
}}

func TestClose(t *testing.T) {
	for _, tt := range closeTests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			var out strings.Builder
			enc := NewEncoder(&out)
			for j, tok := range tt.toks {
				if err := enc.EncodeToken(tok); err != nil {
					t.Fatalf("token #%d: %v", j, err)
				}
			}
			err := enc.Close()
			switch {
			case tt.err != "" && err == nil:
				t.Error(" expected error; got none")
			case tt.err == "" && err != nil:
				t.Errorf(" got error: %v", err)
			case tt.err != "" && err != nil && tt.err != err.Error():
				t.Errorf(" error mismatch; got %v, want %v", err, tt.err)
			}
			if got := out.String(); got != tt.want {
				t.Errorf("\ngot  %v\nwant %v", got, tt.want)
			}
			t.Log(enc.p.closed)
			if err := enc.EncodeToken(xml.Directive("foo")); err == nil {
				t.Errorf("unexpected success when encoding after Close")
			}
		})
	}
}
