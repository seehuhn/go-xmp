// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jvxml

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// An Encoder writes XML data to an output stream.
type Encoder struct {
	p printer
}

// NewEncoder returns a new encoder that writes to w.
func NewEncoder(w io.Writer) *Encoder {
	e := &Encoder{printer{w: bufio.NewWriter(w)}}
	e.p.encoder = e
	return e
}

// Indent sets the encoder to generate XML in which each element
// begins on a new indented line that starts with prefix and is followed by
// one or more copies of indent according to the nesting depth.
func (enc *Encoder) Indent(prefix, indent string) {
	enc.p.prefix = prefix
	enc.p.indent = indent
}

var (
	begComment  = []byte("<!--")
	endComment  = []byte("-->")
	endProcInst = []byte("?>")
)

// EncodeToken writes the given XML token to the stream.
// It returns an error if [StartElement] and [EndElement] tokens are not properly matched.
//
// EncodeToken does not call [Encoder.Flush], because usually it is part of a larger operation
// such as [Encoder.Encode] or [Encoder.EncodeElement] (or a custom [Marshaler]'s MarshalXML invoked
// during those), and those will call Flush when finished.
// Callers that create an Encoder and then invoke EncodeToken directly, without
// using Encode or EncodeElement, need to call Flush when finished to ensure
// that the XML is written to the underlying writer.
//
// EncodeToken allows writing a [ProcInst] with Target set to "xml" only as the first token
// in the stream.
func (enc *Encoder) EncodeToken(t Token) error {
	p := &enc.p
	switch t := t.(type) {
	case xml.StartElement:
		if err := p.writeStart(t.Name, t.Attr, false); err != nil {
			return err
		}
	case EmptyElement:
		if err := p.writeStart(t.Name, t.Attr, true); err != nil {
			return err
		}
	case xml.EndElement:
		if err := p.writeEnd(t.Name); err != nil {
			return err
		}
	case xml.CharData:
		escapeText(p, t, false)
	case xml.Comment:
		if bytes.Contains(t, endComment) {
			return fmt.Errorf("xml: EncodeToken of Comment containing --> marker")
		}
		p.WriteString("<!--")
		p.Write(t)
		p.WriteString("-->")
		return p.cachedWriteError()
	case xml.ProcInst:
		// First token to be encoded which is also a ProcInst with target of xml
		// is the xml declaration. The only ProcInst where target of xml is allowed.
		if t.Target == "xml" && p.w.Buffered() != 0 {
			return fmt.Errorf("xml: EncodeToken of ProcInst xml target only valid for xml declaration, first token encoded")
		}
		if !isNameString(t.Target) {
			return fmt.Errorf("xml: EncodeToken of ProcInst with invalid Target")
		}
		if bytes.Contains(t.Inst, endProcInst) {
			return fmt.Errorf("xml: EncodeToken of ProcInst containing ?> marker")
		}
		p.WriteString("<?")
		p.WriteString(t.Target)
		if len(t.Inst) > 0 {
			p.WriteByte(' ')
			p.Write(t.Inst)
		}
		p.WriteString("?>")
	case xml.Directive:
		if !isValidDirective(t) {
			return fmt.Errorf("xml: EncodeToken of Directive containing wrong < or > markers")
		}
		p.WriteString("<!")
		p.Write(t)
		p.WriteString(">")
	default:
		return fmt.Errorf("xml: EncodeToken of invalid token type")

	}
	return p.cachedWriteError()
}

// isValidDirective reports whether dir is a valid directive text,
// meaning angle brackets are matched, ignoring comments and strings.
func isValidDirective(dir xml.Directive) bool {
	var (
		depth     int
		inquote   uint8
		incomment bool
	)
	for i, c := range dir {
		switch {
		case incomment:
			if c == '>' {
				if n := 1 + i - len(endComment); n >= 0 && bytes.Equal(dir[n:i+1], endComment) {
					incomment = false
				}
			}
			// Just ignore anything in comment
		case inquote != 0:
			if c == inquote {
				inquote = 0
			}
			// Just ignore anything within quotes
		case c == '\'' || c == '"':
			inquote = c
		case c == '<':
			if i+len(begComment) < len(dir) && bytes.Equal(dir[i:i+len(begComment)], begComment) {
				incomment = true
			} else {
				depth++
			}
		case c == '>':
			if depth == 0 {
				return false
			}
			depth--
		}
	}
	return depth == 0 && inquote == 0 && !incomment
}

// Flush flushes any buffered XML to the underlying writer.
// See the [Encoder.EncodeToken] documentation for details about when it is necessary.
func (enc *Encoder) Flush() error {
	return enc.p.w.Flush()
}

// Close the Encoder, indicating that no more data will be written. It flushes
// any buffered XML to the underlying writer and returns an error if the
// written XML is invalid (e.g. by containing unclosed elements).
func (enc *Encoder) Close() error {
	return enc.p.Close()
}

type printer struct {
	w          *bufio.Writer
	encoder    *Encoder
	seq        int
	indent     string
	prefix     string
	depth      int
	indentedIn bool
	putNewline bool
	attrNS     map[string]string // map prefix -> name space
	attrPrefix map[string]string // map name space -> prefix
	prefixes   []string
	tags       []xml.Name
	closed     bool
	err        error
}

// createAttrPrefix finds the name space prefix attribute to use for the given name space,
// defining a new prefix if necessary. It returns the prefix.
func (p *printer) createAttrPrefix(url string) string {
	if prefix := p.attrPrefix[url]; prefix != "" {
		return prefix
	}

	// The "http://www.w3.org/XML/1998/namespace" name space is predefined as "xml"
	// and must be referred to that way.
	// (The "http://www.w3.org/2000/xmlns/" name space is also predefined as "xmlns",
	// but users should not be trying to use that one directly - that's our job.)
	if url == xmlURL {
		return xmlPrefix
	}

	// Need to define a new name space.
	if p.attrPrefix == nil {
		p.attrPrefix = make(map[string]string)
		p.attrNS = make(map[string]string)
	}

	// Pick a name. We try to use the final element of the path
	// but fall back to _.
	prefix := strings.TrimRight(url, "/")
	if i := strings.LastIndex(prefix, "/"); i >= 0 {
		prefix = prefix[i+1:]
	}
	if prefix == "" || !IsName([]byte(prefix)) || strings.Contains(prefix, ":") {
		prefix = "_"
	}
	// xmlanything is reserved and any variant of it regardless of
	// case should be matched, so:
	//    (('X'|'x') ('M'|'m') ('L'|'l'))
	// See Section 2.3 of https://www.w3.org/TR/REC-xml/
	if len(prefix) >= 3 && strings.EqualFold(prefix[:3], "xml") {
		prefix = "_" + prefix
	}
	if p.attrNS[prefix] != "" {
		// Name is taken. Find a better one.
		for p.seq++; ; p.seq++ {
			if id := prefix + "_" + strconv.Itoa(p.seq); p.attrNS[id] == "" {
				prefix = id
				break
			}
		}
	}

	p.attrPrefix[url] = prefix
	p.attrNS[prefix] = url

	p.WriteString(`xmlns:`)
	p.WriteString(prefix)
	p.WriteString(`="`)
	xml.EscapeText(p, []byte(url))
	p.WriteString(`" `)

	p.prefixes = append(p.prefixes, prefix)

	return prefix
}

// deleteAttrPrefix removes an attribute name space prefix.
func (p *printer) deleteAttrPrefix(prefix string) {
	delete(p.attrPrefix, p.attrNS[prefix])
	delete(p.attrNS, prefix)
}

func (p *printer) markPrefix() {
	p.prefixes = append(p.prefixes, "")
}

func (p *printer) popPrefix() {
	for len(p.prefixes) > 0 {
		prefix := p.prefixes[len(p.prefixes)-1]
		p.prefixes = p.prefixes[:len(p.prefixes)-1]
		if prefix == "" {
			break
		}
		p.deleteAttrPrefix(prefix)
	}
}

// writeStart writes the given start element.
func (p *printer) writeStart(name xml.Name, attr []xml.Attr, empty bool) error {
	if name.Local == "" {
		return fmt.Errorf("xml: start tag with no name")
	}

	if !empty {
		p.tags = append(p.tags, name)
		p.markPrefix()
	}

	p.writeIndent(1)
	p.WriteByte('<')
	p.WriteString(name.Local)

	if name.Space != "" {
		p.WriteString(` xmlns="`)
		p.EscapeString(name.Space)
		p.WriteByte('"')
	}

	// Attributes
	for _, attr := range attr {
		name := attr.Name
		if name.Local == "" {
			continue
		}
		p.WriteByte(' ')
		if name.Space != "" {
			p.WriteString(p.createAttrPrefix(name.Space))
			p.WriteByte(':')
		}
		p.WriteString(name.Local)
		p.WriteString(`="`)
		p.EscapeString(attr.Value)
		p.WriteByte('"')
	}
	if empty {
		p.WriteString("/>")
		p.writeIndent(-1)
	} else {
		p.WriteByte('>')
	}
	return nil
}

func (p *printer) writeEnd(name xml.Name) error {
	if name.Local == "" {
		return fmt.Errorf("xml: end tag with no name")
	}
	if len(p.tags) == 0 || p.tags[len(p.tags)-1].Local == "" {
		return fmt.Errorf("xml: end tag </%s> without start tag", name.Local)
	}
	if top := p.tags[len(p.tags)-1]; top != name {
		if top.Local != name.Local {
			return fmt.Errorf("xml: end tag </%s> does not match start tag <%s>", name.Local, top.Local)
		}
		return fmt.Errorf("xml: end tag </%s> in namespace %s does not match start tag <%s> in namespace %s", name.Local, name.Space, top.Local, top.Space)
	}
	p.tags = p.tags[:len(p.tags)-1]

	p.writeIndent(-1)
	p.WriteByte('<')
	p.WriteByte('/')
	p.WriteString(name.Local)
	p.WriteByte('>')
	p.popPrefix()
	return nil
}

// Write implements io.Writer
func (p *printer) Write(b []byte) (n int, err error) {
	if p.closed && p.err == nil {
		p.err = errors.New("use of closed Encoder")
	}
	if p.err == nil {
		n, p.err = p.w.Write(b)
	}
	return n, p.err
}

// WriteString implements io.StringWriter
func (p *printer) WriteString(s string) (n int, err error) {
	if p.closed && p.err == nil {
		p.err = errors.New("use of closed Encoder")
	}
	if p.err == nil {
		n, p.err = p.w.WriteString(s)
	}
	return n, p.err
}

// WriteByte implements io.ByteWriter
func (p *printer) WriteByte(c byte) error {
	if p.closed && p.err == nil {
		p.err = errors.New("use of closed Encoder")
	}
	if p.err == nil {
		p.err = p.w.WriteByte(c)
	}
	return p.err
}

// Close the Encoder, indicating that no more data will be written. It flushes
// any buffered XML to the underlying writer and returns an error if the
// written XML is invalid (e.g. by containing unclosed elements).
func (p *printer) Close() error {
	if p.closed {
		return nil
	}
	p.closed = true
	if err := p.w.Flush(); err != nil {
		return err
	}
	if len(p.tags) > 0 {
		return fmt.Errorf("unclosed tag <%s>", p.tags[len(p.tags)-1].Local)
	}
	return nil
}

// return the bufio Writer's cached write error
func (p *printer) cachedWriteError() error {
	_, err := p.Write(nil)
	return err
}

func (p *printer) writeIndent(depthDelta int) {
	if len(p.prefix) == 0 && len(p.indent) == 0 {
		return
	}
	if depthDelta < 0 {
		p.depth--
		if p.indentedIn {
			p.indentedIn = false
			return
		}
		p.indentedIn = false
	}
	if p.putNewline {
		p.WriteByte('\n')
	} else {
		p.putNewline = true
	}
	if len(p.prefix) > 0 {
		p.WriteString(p.prefix)
	}
	if len(p.indent) > 0 {
		for i := 0; i < p.depth; i++ {
			p.WriteString(p.indent)
		}
	}
	if depthDelta > 0 {
		p.depth++
		p.indentedIn = true
	}
}
