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
	"encoding/xml"
	"io"
	"sort"

	"golang.org/x/exp/maps"
	"seehuhn.de/go/xmp/jvxml"
)

// PacketOptions can be used to control the output format of the [Packet.Write]
// method.
type PacketOptions struct {
	Pretty bool
}

// Write writes the XMP packet to the given writer.
func (p *Packet) Write(w io.Writer, opt *PacketOptions) error {
	e, err := p.newEncoder(w, opt)
	if err != nil {
		return err
	}

	names := maps.Keys(p.Properties)
	sort.Slice(names, func(i, j int) bool {
		if names[i].Space != names[j].Space {
			return names[i].Space < names[j].Space
		}
		return names[i].Local < names[j].Local
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
	w io.Writer
	*jvxml.Encoder
	nsToPrefix map[string]string
	prefixToNS map[string]string
}

// newEncoder returns a new encoder that writes to w.
func (p *Packet) newEncoder(w io.Writer, opt *PacketOptions) (*encoder, error) {
	nsUsed := make(map[string]struct{})
	nsUsed[xmlNamespace] = struct{}{}
	nsUsed[rdfNamespace] = struct{}{}
	for key, value := range p.Properties {
		nsUsed[key.Space] = struct{}{}
		value.getNamespaces(nsUsed)
	}

	nsToPrefix := make(map[string]string)
	prefixToNS := make(map[string]string)
	// register default namespaces first, ...
	for ns := range nsUsed {
		if pfx, isDefault := defaultPrefix[ns]; isDefault {
			nsToPrefix[ns] = pfx
			prefixToNS[pfx] = ns
		}
	}
	// ... then the ones registered in the packet, ...
	for ns := range nsUsed {
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
	for ns := range nsUsed {
		if _, alreadyDone := nsToPrefix[ns]; alreadyDone {
			continue
		}
		pfx := getPrefix(prefixToNS, ns)
		nsToPrefix[ns] = pfx
		prefixToNS[pfx] = ns
	}

	enc := jvxml.NewEncoder(w)
	if opt != nil && opt.Pretty {
		enc.Indent("", "\t")
	}
	e := &encoder{
		w:          w,
		Encoder:    enc,
		nsToPrefix: nsToPrefix,
		prefixToNS: prefixToNS,
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
	namespaces := maps.Keys(e.nsToPrefix)
	sort.Strings(namespaces)
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

	err = e.EncodeToken(xml.ProcInst{
		Target: "xpacket",
		Inst:   []byte("end=\"w\""),
	})
	if err != nil {
		return err
	}

	err = e.Encoder.Close()
	if err != nil {
		return err
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
