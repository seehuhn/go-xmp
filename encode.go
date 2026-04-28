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
	"cmp"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"

	"seehuhn.de/go/xmp/jvxml"
)

// PacketOptions can be used to control the output format of the [Packet.Write]
// method.
type PacketOptions struct {
	// Pretty controls whether the encoded XMP is indented for readability.
	Pretty bool

	// PadToLength controls trailing whitespace padding inside the XMP packet
	// wrapper.
	//
	// If zero, no padding is added and the trailer reads
	// <?xpacket end="r"?>, signalling that the packet is read-only.
	//
	// If positive, padding is added so that the encoded packet has exactly
	// the requested length in bytes, and the trailer reads
	// <?xpacket end="w"?>, signalling that the packet may be edited in
	// place inside its host file. [Packet.Write] returns an error if the
	// encoded packet does not fit in PadToLength bytes.
	PadToLength int
}

// ErrPacketTooLong is returned by [Packet.Write] when the encoded packet
// does not fit into the requested [PacketOptions.PadToLength].
var ErrPacketTooLong = errors.New("xmp: encoded packet exceeds PadToLength")

// Write writes the XMP packet to the given writer.
func (p *Packet) Write(w io.Writer, opt *PacketOptions) error {
	e, err := p.newEncoder(w, opt)
	if err != nil {
		return err
	}

	names := slices.Collect(maps.Keys(p.Properties))
	slices.SortFunc(names, func(a, b xml.Name) int {
		return cmp.Or(cmp.Compare(a.Space, b.Space), cmp.Compare(a.Local, b.Local))
	})

	for _, name := range names {
		value := p.Properties[name]
		tokens := value.appendXML(nil, name)
		for _, t := range tokens {
			t = e.fixToken(t)

			err = e.EncodeToken(t)
			if err != nil {
				return err
			}
		}
	}

	err = e.Close()
	if err != nil {
		return err
	}

	return nil
}

func (e *encoder) fixToken(t jvxml.Token) jvxml.Token {
	switch t := t.(type) {
	case xml.StartElement:
		t.Name = e.fixName(t.Name)
		for i, a := range t.Attr {
			t.Attr[i].Name = e.fixName(a.Name)
		}
		return t
	case xml.EndElement:
		t.Name = e.fixName(t.Name)
		return t
	case jvxml.EmptyElement:
		t.Name = e.fixName(t.Name)
		for i, a := range t.Attr {
			t.Attr[i].Name = e.fixName(a.Name)
		}
		return t
	case xml.CharData, xml.ProcInst, xml.Comment:
		return t
	default:
		panic("unexpected XML element type")
	}
}

// An encoder writes XMP data to an output stream.
type encoder struct {
	w *countingWriter
	*jvxml.Encoder
	nsToPrefix map[string]string
	prefixToNS map[string]string
	padTo      int
}

// countingWriter wraps an [io.Writer] and counts the bytes written.
type countingWriter struct {
	w io.Writer
	n int
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += n
	return n, err
}

// newEncoder returns a new encoder that writes to w.
func (p *Packet) newEncoder(w io.Writer, opt *PacketOptions) (*encoder, error) {
	// Gather a list of all namespaces used in the packet.
	nsUsed := make(map[string]struct{})
	nsUsed[xmlNamespace] = struct{}{}
	nsUsed[rdfNamespace] = struct{}{}
	for key, value := range p.Properties {
		nsUsed[key.Space] = struct{}{}
		value.getNamespaces(nsUsed)
	}
	nsList := slices.Sorted(maps.Keys(nsUsed))

	// Fix the namespace prefixes.
	nsToPrefix := make(map[string]string)
	prefixToNS := make(map[string]string)
	// register default namespaces first, ...
	nsToPrefix[xmlNamespace] = "xml"
	prefixToNS["xml"] = xmlNamespace
	nsToPrefix[rdfNamespace] = "rdf"
	prefixToNS["rdf"] = rdfNamespace
	// ... then the ones registered in the packet, ...
	for _, ns := range nsList {
		if _, alreadyDone := nsToPrefix[ns]; alreadyDone {
			continue
		}
		pfx, isRegistered := p.nsToPrefix[ns]
		if !isRegistered {
			continue
		}
		if _, isClash := nsToPrefix[pfx]; isClash {
			continue
		}
		nsToPrefix[ns] = pfx
		prefixToNS[pfx] = ns
	}
	// ... and then the rest:
	for _, ns := range nsList {
		if _, alreadyDone := nsToPrefix[ns]; alreadyDone {
			continue
		}
		pfx := getPrefix(prefixToNS, ns)
		nsToPrefix[ns] = pfx
		prefixToNS[pfx] = ns
	}

	cw := &countingWriter{w: w}
	enc := jvxml.NewEncoder(cw)
	if opt != nil && opt.Pretty {
		enc.Indent("", "\t")
	}
	e := &encoder{
		w:          cw,
		Encoder:    enc,
		nsToPrefix: nsToPrefix,
		prefixToNS: prefixToNS,
	}
	if opt != nil {
		e.padTo = opt.PadToLength
	}

	err := e.EncodeToken(xml.ProcInst{
		Target: "xpacket",
		Inst:   []byte("begin=\"\uFEFF\" id=\"W5M0MpCehiHzreSzNTczkc9d\""),
	})
	if err != nil {
		return nil, err
	}

	err = e.EncodeToken(xml.CharData("\n"))
	if err != nil {
		return nil, err
	}

	var attrs []xml.Attr
	namespaces := slices.Sorted(maps.Keys(e.nsToPrefix))
	for _, ns := range namespaces {
		if ns == xmlNamespace {
			continue
		}
		pfx := e.nsToPrefix[ns]
		attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "xmlns:" + pfx}, Value: ns})
	}
	err = e.EncodeToken(xml.StartElement{
		Name: e.fixName(nameRDFRoot),
		Attr: attrs,
	})
	if err != nil {
		return nil, err
	}

	attrs = attrs[:0]
	about := ""
	if p.About != nil {
		about = p.About.String()
	}
	attrs = append(attrs, xml.Attr{Name: e.fixName(nameRDFAbout), Value: about})
	err = e.EncodeToken(xml.StartElement{
		Name: e.fixName(nameRDFDescription),
		Attr: attrs,
	})
	if err != nil {
		return nil, err
	}

	return e, nil
}

// Close closes the encoder.  This must be called after all data has been
// written to the encoder.
func (e *encoder) Close() error {
	err := e.EncodeToken(xml.EndElement{
		Name: e.fixName(nameRDFDescription),
	})
	if err != nil {
		return err
	}

	err = e.EncodeToken(xml.EndElement{
		Name: e.fixName(nameRDFRoot),
	})
	if err != nil {
		return err
	}

	err = e.EncodeToken(xml.CharData("\n"))
	if err != nil {
		return err
	}

	// flush the XML encoder so e.w.n reflects the bytes written so far
	err = e.Encoder.Flush()
	if err != nil {
		return err
	}

	// write the trailer (with optional padding) directly so we can
	// match the requested packet length exactly
	var trailer []byte
	if e.padTo > 0 {
		trailer = []byte(`<?xpacket end="w"?>`)
		remaining := e.padTo - e.w.n - len(trailer)
		if remaining < 0 {
			total := e.w.n + len(trailer)
			return fmt.Errorf("%w: %d bytes needed, limit %d", ErrPacketTooLong, total, e.padTo)
		}
		if err := writePadding(e.w, remaining); err != nil {
			return err
		}
	} else {
		trailer = []byte(`<?xpacket end="r"?>`)
	}
	if _, err := e.w.Write(trailer); err != nil {
		return err
	}
	return nil
}

// writePadding writes n bytes of ASCII whitespace, formatted as lines of
// up to 80 characters separated by newlines.
func writePadding(w io.Writer, n int) error {
	if n <= 0 {
		return nil
	}
	const lineLen = 80
	var line [lineLen]byte
	for i := range line {
		line[i] = ' '
	}
	line[lineLen-1] = '\n'

	for n >= lineLen {
		if _, err := w.Write(line[:]); err != nil {
			return err
		}
		n -= lineLen
	}
	if n > 0 {
		var tail [lineLen]byte
		for i := range tail {
			tail[i] = ' '
		}
		tail[n-1] = '\n'
		if _, err := w.Write(tail[:n]); err != nil {
			return err
		}
	}
	return nil
}

func (e *encoder) fixName(name xml.Name) xml.Name {
	pfx, ok := e.nsToPrefix[name.Space]
	if !ok {
		panic("namespace not registered: " + name.Space)
	}
	return xml.Name{Local: pfx + ":" + name.Local}
}
