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
	err  bool
}{{
	desc: "start element with name space",
	toks: []Token{
		xml.StartElement{Name: xml.Name{Space: "space", Local: "local"}},
	},
	want: `<local xmlns="space">`,
}, {
	desc: "start element with no name",
	toks: []Token{
		xml.StartElement{Name: xml.Name{Space: "space", Local: ""}},
	},
	err: true, // xml: start tag with no name
}, {
	desc: "end element with no name",
	toks: []Token{
		xml.EndElement{Name: xml.Name{Space: "space", Local: ""}},
	},
	err: true, // xml: end tag with no name
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
	err: true, // xml: EncodeToken of Comment containing --> marker
}, {
	desc: "proc instruction",
	toks: []Token{
		xml.ProcInst{Target: "Target", Inst: []byte("Instruction")},
	},
	want: `<?Target Instruction?>`,
}, {
	desc: "proc instruction with empty target",
	toks: []Token{
		xml.ProcInst{Target: "", Inst: []byte("Instruction")},
	},
	err: true, // xml: EncodeToken of ProcInst with invalid Target
}, {
	desc: "proc instruction with bad content",
	toks: []Token{
		xml.ProcInst{Target: "", Inst: []byte("Instruction?>")},
	},
	err: true, // xml: EncodeToken of ProcInst with invalid Target
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
	err: true, // xml: EncodeToken of Directive containing wrong < or > markers
}, {
	desc: "end tag without start tag",
	toks: []Token{
		xml.EndElement{Name: xml.Name{Space: "foo", Local: "bar"}},
	},
	err: true, // xml: end tag </bar> without start tag
}, {
	desc: "mismatching end tag local name",
	toks: []Token{
		xml.StartElement{Name: xml.Name{Local: "foo"}},
		xml.EndElement{Name: xml.Name{Local: "bar"}},
	},
	err:  true, // xml: end tag </bar> does not match start tag <foo>
	want: `<foo>`,
}, {
	desc: "mismatching end tag namespace",
	toks: []Token{
		xml.StartElement{Name: xml.Name{Space: "space", Local: "foo"}},
		xml.EndElement{Name: xml.Name{Space: "another", Local: "foo"}},
	},
	err:  true, // xml: end tag </foo> in namespace another does not match start tag <foo> in namespace space
	want: `<foo xmlns="space">`,
}, {
	desc: "start element with explicit namespace",
	toks: []Token{
		xml.StartElement{Name: xml.Name{Space: "space", Local: "local"}, Attr: []xml.Attr{
			{Name: xml.Name{Space: "xmlns", Local: "x"}, Value: "space"},
			{Name: xml.Name{Space: "space", Local: "foo"}, Value: "value"},
		}},
	},
	want: `<local xmlns="space" xmlns:_xmlns="xmlns" _xmlns:x="space" xmlns:space="space" space:foo="value">`,
}, {
	desc: "start element with explicit namespace and colliding prefix",
	toks: []Token{
		xml.StartElement{Name: xml.Name{Space: "space", Local: "local"}, Attr: []xml.Attr{
			{Name: xml.Name{Space: "xmlns", Local: "x"}, Value: "space"},
			{Name: xml.Name{Space: "space", Local: "foo"}, Value: "value"},
			{Name: xml.Name{Space: "x", Local: "bar"}, Value: "other"},
		}},
	},
	want: `<local xmlns="space" xmlns:_xmlns="xmlns" _xmlns:x="space" xmlns:space="space" space:foo="value" xmlns:x="x" x:bar="other">`,
}, {
	desc: "start element using previously defined namespace",
	toks: []Token{
		xml.StartElement{Name: xml.Name{Local: "local"}, Attr: []xml.Attr{
			{Name: xml.Name{Space: "xmlns", Local: "x"}, Value: "space"},
		}},
		xml.StartElement{Name: xml.Name{Space: "space", Local: "foo"}, Attr: []xml.Attr{
			{Name: xml.Name{Space: "space", Local: "x"}, Value: "y"},
		}},
	},
	want: `<local xmlns:_xmlns="xmlns" _xmlns:x="space"><foo xmlns="space" xmlns:space="space" space:x="y">`,
}, {
	desc: "nested name space with same prefix",
	toks: []Token{
		xml.StartElement{Name: xml.Name{Local: "foo"}, Attr: []xml.Attr{
			{Name: xml.Name{Space: "xmlns", Local: "x"}, Value: "space1"},
		}},
		xml.StartElement{Name: xml.Name{Local: "foo"}, Attr: []xml.Attr{
			{Name: xml.Name{Space: "xmlns", Local: "x"}, Value: "space2"},
		}},
		xml.StartElement{Name: xml.Name{Local: "foo"}, Attr: []xml.Attr{
			{Name: xml.Name{Space: "space1", Local: "a"}, Value: "space1 value"},
			{Name: xml.Name{Space: "space2", Local: "b"}, Value: "space2 value"},
		}},
		xml.EndElement{Name: xml.Name{Local: "foo"}},
		xml.EndElement{Name: xml.Name{Local: "foo"}},
		xml.StartElement{Name: xml.Name{Local: "foo"}, Attr: []xml.Attr{
			{Name: xml.Name{Space: "space1", Local: "a"}, Value: "space1 value"},
			{Name: xml.Name{Space: "space2", Local: "b"}, Value: "space2 value"},
		}},
	},
	want: `<foo xmlns:_xmlns="xmlns" _xmlns:x="space1"><foo _xmlns:x="space2"><foo xmlns:space1="space1" space1:a="space1 value" xmlns:space2="space2" space2:b="space2 value"></foo></foo><foo xmlns:space1="space1" space1:a="space1 value" xmlns:space2="space2" space2:b="space2 value">`,
}, {
	desc: "start element defining several prefixes for the same name space",
	toks: []Token{
		xml.StartElement{Name: xml.Name{Space: "space", Local: "foo"}, Attr: []xml.Attr{
			{Name: xml.Name{Space: "xmlns", Local: "a"}, Value: "space"},
			{Name: xml.Name{Space: "xmlns", Local: "b"}, Value: "space"},
			{Name: xml.Name{Space: "space", Local: "x"}, Value: "value"},
		}},
	},
	want: `<foo xmlns="space" xmlns:_xmlns="xmlns" _xmlns:a="space" _xmlns:b="space" xmlns:space="space" space:x="value">`,
}, {
	desc: "nested element redefines name space",
	toks: []Token{
		xml.StartElement{Name: xml.Name{Local: "foo"}, Attr: []xml.Attr{
			{Name: xml.Name{Space: "xmlns", Local: "x"}, Value: "space"},
		}},
		xml.StartElement{Name: xml.Name{Space: "space", Local: "foo"}, Attr: []xml.Attr{
			{Name: xml.Name{Space: "xmlns", Local: "y"}, Value: "space"},
			{Name: xml.Name{Space: "space", Local: "a"}, Value: "value"},
		}},
	},
	want: `<foo xmlns:_xmlns="xmlns" _xmlns:x="space"><foo xmlns="space" _xmlns:y="space" xmlns:space="space" space:a="value">`,
}, {
	desc: "nested element creates alias for default name space",
	toks: []Token{
		xml.StartElement{Name: xml.Name{Space: "space", Local: "foo"}, Attr: []xml.Attr{
			{Name: xml.Name{Local: "xmlns"}, Value: "space"},
		}},
		xml.StartElement{Name: xml.Name{Space: "space", Local: "foo"}, Attr: []xml.Attr{
			{Name: xml.Name{Space: "xmlns", Local: "y"}, Value: "space"},
			{Name: xml.Name{Space: "space", Local: "a"}, Value: "value"},
		}},
	},
	want: `<foo xmlns="space" xmlns="space"><foo xmlns="space" xmlns:_xmlns="xmlns" _xmlns:y="space" xmlns:space="space" space:a="value">`,
}, {
	desc: "nested element defines default name space with existing prefix",
	toks: []Token{
		xml.StartElement{Name: xml.Name{Local: "foo"}, Attr: []xml.Attr{
			{Name: xml.Name{Space: "xmlns", Local: "x"}, Value: "space"},
		}},
		xml.StartElement{Name: xml.Name{Space: "space", Local: "foo"}, Attr: []xml.Attr{
			{Name: xml.Name{Local: "xmlns"}, Value: "space"},
			{Name: xml.Name{Space: "space", Local: "a"}, Value: "value"},
		}},
	},
	want: `<foo xmlns:_xmlns="xmlns" _xmlns:x="space"><foo xmlns="space" xmlns="space" xmlns:space="space" space:a="value">`,
}, {
	desc: "nested element uses empty attribute name space when default ns defined",
	toks: []Token{
		xml.StartElement{Name: xml.Name{Space: "space", Local: "foo"}, Attr: []xml.Attr{
			{Name: xml.Name{Local: "xmlns"}, Value: "space"},
		}},
		xml.StartElement{Name: xml.Name{Space: "space", Local: "foo"}, Attr: []xml.Attr{
			{Name: xml.Name{Local: "attr"}, Value: "value"},
		}},
	},
	want: `<foo xmlns="space" xmlns="space"><foo xmlns="space" attr="value">`,
}, {
	desc: "redefine xmlns",
	toks: []Token{
		xml.StartElement{Name: xml.Name{Local: "foo"}, Attr: []xml.Attr{
			{Name: xml.Name{Space: "foo", Local: "xmlns"}, Value: "space"},
		}},
	},
	want: `<foo xmlns:foo="foo" foo:xmlns="space">`,
}, {
	desc: "xmlns with explicit name space #1",
	toks: []Token{
		xml.StartElement{Name: xml.Name{Space: "space", Local: "foo"}, Attr: []xml.Attr{
			{Name: xml.Name{Space: "xml", Local: "xmlns"}, Value: "space"},
		}},
	},
	want: `<foo xmlns="space" xmlns:_xml="xml" _xml:xmlns="space">`,
}, {
	desc: "xmlns with explicit name space #2",
	toks: []Token{
		xml.StartElement{Name: xml.Name{Space: "space", Local: "foo"}, Attr: []xml.Attr{
			{Name: xml.Name{Space: xmlURL, Local: "xmlns"}, Value: "space"},
		}},
	},
	want: `<foo xmlns="space" xml:xmlns="space">`,
}, {
	desc: "empty name space declaration is ignored",
	toks: []Token{
		xml.StartElement{Name: xml.Name{Local: "foo"}, Attr: []xml.Attr{
			{Name: xml.Name{Space: "xmlns", Local: "foo"}, Value: ""},
		}},
	},
	want: `<foo xmlns:_xmlns="xmlns" _xmlns:foo="">`,
}, {
	desc: "attribute with no name is ignored",
	toks: []Token{
		xml.StartElement{Name: xml.Name{Local: "foo"}, Attr: []xml.Attr{
			{Name: xml.Name{Local: ""}, Value: "value"},
		}},
	},
	want: `<foo>`,
}, {
	desc: "namespace URL with non-valid name",
	toks: []Token{
		xml.StartElement{Name: xml.Name{Space: "/34", Local: "foo"}, Attr: []xml.Attr{
			{Name: xml.Name{Space: "/34", Local: "x"}, Value: "value"},
		}},
	},
	want: `<foo xmlns="/34" xmlns:_="/34" _:x="value">`,
}, {
	desc: "nested element resets default namespace to empty",
	toks: []Token{
		xml.StartElement{Name: xml.Name{Space: "space", Local: "foo"}, Attr: []xml.Attr{
			{Name: xml.Name{Local: "xmlns"}, Value: "space"},
		}},
		xml.StartElement{Name: xml.Name{Local: "foo"}, Attr: []xml.Attr{
			{Name: xml.Name{Local: "xmlns"}, Value: ""},
			{Name: xml.Name{Local: "x"}, Value: "value"},
			{Name: xml.Name{Space: "space", Local: "x"}, Value: "value"},
		}},
	},
	want: `<foo xmlns="space" xmlns="space"><foo xmlns="" x="value" xmlns:space="space" space:x="value">`,
}, {
	desc: "nested element requires empty default name space",
	toks: []Token{
		xml.StartElement{Name: xml.Name{Space: "space", Local: "foo"}, Attr: []xml.Attr{
			{Name: xml.Name{Local: "xmlns"}, Value: "space"},
		}},
		xml.StartElement{Name: xml.Name{Local: "foo"}},
	},
	want: `<foo xmlns="space" xmlns="space"><foo>`,
}, {
	desc: "attribute uses name space from xmlns",
	toks: []Token{
		xml.StartElement{Name: xml.Name{Space: "some/space", Local: "foo"}, Attr: []xml.Attr{
			{Name: xml.Name{Local: "attr"}, Value: "value"},
			{Name: xml.Name{Space: "some/space", Local: "other"}, Value: "other value"},
		}},
	},
	want: `<foo xmlns="some/space" attr="value" xmlns:space="some/space" space:other="other value">`,
}, {
	desc: "default name space should not be used by attributes",
	toks: []Token{
		xml.StartElement{Name: xml.Name{Space: "space", Local: "foo"}, Attr: []xml.Attr{
			{Name: xml.Name{Local: "xmlns"}, Value: "space"},
			{Name: xml.Name{Space: "xmlns", Local: "bar"}, Value: "space"},
			{Name: xml.Name{Space: "space", Local: "baz"}, Value: "foo"},
		}},
		xml.StartElement{Name: xml.Name{Space: "space", Local: "baz"}},
		xml.EndElement{Name: xml.Name{Space: "space", Local: "baz"}},
		xml.EndElement{Name: xml.Name{Space: "space", Local: "foo"}},
	},
	want: `<foo xmlns="space" xmlns="space" xmlns:_xmlns="xmlns" _xmlns:bar="space" xmlns:space="space" space:baz="foo"><baz xmlns="space"></baz></foo>`,
}, {
	desc: "default name space not used by attributes, not explicitly defined",
	toks: []Token{
		xml.StartElement{Name: xml.Name{Space: "space", Local: "foo"}, Attr: []xml.Attr{
			{Name: xml.Name{Local: "xmlns"}, Value: "space"},
			{Name: xml.Name{Space: "space", Local: "baz"}, Value: "foo"},
		}},
		xml.StartElement{Name: xml.Name{Space: "space", Local: "baz"}},
		xml.EndElement{Name: xml.Name{Space: "space", Local: "baz"}},
		xml.EndElement{Name: xml.Name{Space: "space", Local: "foo"}},
	},
	want: `<foo xmlns="space" xmlns="space" xmlns:space="space" space:baz="foo"><baz xmlns="space"></baz></foo>`,
}, {
	desc: "impossible xmlns declaration",
	toks: []Token{
		xml.StartElement{Name: xml.Name{Local: "foo"}, Attr: []xml.Attr{
			{Name: xml.Name{Local: "xmlns"}, Value: "space"},
		}},
		xml.StartElement{Name: xml.Name{Space: "space", Local: "bar"}, Attr: []xml.Attr{
			{Name: xml.Name{Space: "space", Local: "attr"}, Value: "value"},
		}},
	},
	want: `<foo xmlns="space"><bar xmlns="space" xmlns:space="space" space:attr="value">`,
}, {
	desc: "reserved namespace prefix -- all lower case",
	toks: []Token{
		xml.StartElement{Name: xml.Name{Local: "foo"}, Attr: []xml.Attr{
			{Name: xml.Name{Space: "http://www.w3.org/2001/xmlSchema-instance", Local: "nil"}, Value: "true"},
		}},
	},
	want: `<foo xmlns:_xmlSchema-instance="http://www.w3.org/2001/xmlSchema-instance" _xmlSchema-instance:nil="true">`,
}, {
	desc: "reserved namespace prefix -- all upper case",
	toks: []Token{
		xml.StartElement{Name: xml.Name{Local: "foo"}, Attr: []xml.Attr{
			{Name: xml.Name{Space: "http://www.w3.org/2001/XMLSchema-instance", Local: "nil"}, Value: "true"},
		}},
	},
	want: `<foo xmlns:_XMLSchema-instance="http://www.w3.org/2001/XMLSchema-instance" _XMLSchema-instance:nil="true">`,
}, {
	desc: "reserved namespace prefix -- all mixed case",
	toks: []Token{
		xml.StartElement{Name: xml.Name{Local: "foo"}, Attr: []xml.Attr{
			{Name: xml.Name{Space: "http://www.w3.org/2001/XmLSchema-instance", Local: "nil"}, Value: "true"},
		}},
	},
	want: `<foo xmlns:_XmLSchema-instance="http://www.w3.org/2001/XmLSchema-instance" _XmLSchema-instance:nil="true">`,
}, {
	desc: "empty element",
	toks: []Token{
		EmptyElement{xml.Name{Local: "foo"}, nil},
	},
	want: "<foo/>",
}, {
	desc: "empty element",
	toks: []Token{
		EmptyElement{xml.Name{Local: "xmp:BaseURL"}, []xml.Attr{
			{Name: xml.Name{Local: "rdf:resource"}, Value: "http://www.example.com/"},
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
		case tt.err && err == nil:
			errorf(" expected error; got none")
			continue
		case !tt.err && err != nil:
			errorf(" got error: %v", err)
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

	if err := enc.EncodeToken(xml.ProcInst{Target: "xml", Inst: []byte("Instruction")}); err != nil {
		t.Fatalf("enc.EncodeToken: expected to be able to encode xml target ProcInst as first token, %s", err)
	}

	if err := enc.EncodeToken(xml.ProcInst{Target: "Target", Inst: []byte("Instruction")}); err != nil {
		t.Fatalf("enc.EncodeToken: expected to be able to add non-xml target ProcInst")
	}

	if err := enc.EncodeToken(xml.ProcInst{Target: "xml", Inst: []byte("Instruction")}); err == nil {
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
	if err := enc.EncodeToken(&xml.StartElement{Name: xml.Name{Local: "object1"}}); err == nil {
		t.Errorf("enc.EncodeToken: pointer type should be rejected")
	}
	if err := enc.EncodeToken(&xml.EndElement{Name: xml.Name{Local: "object1"}}); err == nil {
		t.Errorf("enc.EncodeToken: pointer type should be rejected")
	}
	if err := enc.EncodeToken(xml.StartElement{Name: xml.Name{Local: "object2"}}); err != nil {
		t.Errorf("enc.EncodeToken: StartElement %s", err)
	}
	if err := enc.EncodeToken(xml.EndElement{Name: xml.Name{Local: "object2"}}); err != nil {
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
	err  bool
}{{
	desc: "unclosed start element",
	toks: []Token{
		xml.StartElement{Name: xml.Name{Local: "foo"}},
	},
	want: `<foo>`,
	err:  true, // unclosed tag <foo>
}, {
	desc: "closed element",
	toks: []Token{
		xml.StartElement{Name: xml.Name{Local: "foo"}},
		xml.EndElement{Name: xml.Name{Local: "foo"}},
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
			case tt.err && err == nil:
				t.Error(" expected error; got none")
			case !tt.err && err != nil:
				t.Errorf(" got error: %v", err)
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
