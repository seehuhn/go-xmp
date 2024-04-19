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
	"sort"

	"golang.org/x/exp/maps"
	"seehuhn.de/go/xmp/jvxml"
)

// Encode encodes the packet to an XML byte slice.
func (p *Packet) Encode() ([]byte, error) {
	ns := p.getNamespaces()

	e, err := newEncoder(p.About, ns)
	if err != nil {
		return nil, err
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

		start := xml.StartElement{Name: e.makeName(name.Space, name.Local)}
		var attr []xml.Attr
		var tokens []xml.Token
		end := xml.EndElement{Name: e.makeName(name.Space, name.Local)}

		switch value := value.(type) {
		case textValue:
			tokens = append(tokens, xml.CharData(value.val))
		case uriValue:
			attr = append(attr, xml.Attr{
				Name:  e.makeName(RDFNamespace, "resource"),
				Value: value.val.String(),
			})
		}

		for _, q := range value.Qualifiers() {
			if q.Name == attrXMLLang {
				panic("not implemented")
				continue
			}
			panic("not implemented")
		}

		start.Attr = attr
		if len(tokens) == 0 {
			empty := jvxml.EmptyElement{
				Name: start.Name,
				Attr: start.Attr,
			}
			err = e.EncodeToken(empty)
			if err != nil {
				return nil, err
			}
		} else {
			err = e.EncodeToken(start)
			if err != nil {
				return nil, err
			}
			for _, token := range tokens {
				err = e.EncodeToken(token)
				if err != nil {
					return nil, err
				}
			}
			err = e.EncodeToken(end)
			if err != nil {
				return nil, err
			}
		}
	}

	err = e.Close()
	if err != nil {
		return nil, err
	}

	return e.buf.Bytes(), nil
}

// An encoder writes XMP data to an output stream.
type encoder struct {
	buf *bytes.Buffer
	*jvxml.Encoder
	nsToPrefix map[string]string
	prefixToNS map[string]string
}

// newEncoder returns a new encoder that writes to w.
func newEncoder(aboutURL *url.URL, nsUsed map[string]struct{}) (*encoder, error) {
	nsUsed[xmlNamespace] = struct{}{}
	nsUsed[RDFNamespace] = struct{}{}

	nsToPrefix := make(map[string]string)
	prefixToNS := make(map[string]string)
	// register default namespaces first, ...
	for ns := range nsUsed {
		if pfx, isDefault := defaultPrefix[ns]; isDefault {
			nsToPrefix[ns] = pfx
			prefixToNS[pfx] = ns
		}
	}
	// ... and then the others
	for ns := range nsUsed {
		if _, alreadyDone := nsToPrefix[ns]; alreadyDone {
			continue
		}
		pfx := getPrefix(nsToPrefix, ns)
		nsToPrefix[ns] = pfx
		prefixToNS[pfx] = ns
	}

	buf := &bytes.Buffer{}
	enc := jvxml.NewEncoder(buf)
	enc.Indent("", "  ") // TODO(voss): remove?
	e := &encoder{
		buf:        buf,
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
	nameSpaces := maps.Keys(e.nsToPrefix)
	sort.Strings(nameSpaces)
	for _, ns := range nameSpaces {
		if ns == xmlNamespace {
			continue
		}
		pfx := e.nsToPrefix[ns]
		attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "xmlns:" + pfx}, Value: ns})
	}
	err = e.EncodeToken(xml.StartElement{
		Name: e.makeName(RDFNamespace, "RDF"),
		Attr: attrs,
	})
	if err != nil {
		return nil, err
	}

	attrs = attrs[:0]
	about := ""
	if aboutURL != nil {
		about = aboutURL.String()
	}
	attrs = append(attrs, xml.Attr{Name: e.makeName(RDFNamespace, "about"), Value: about})
	err = e.EncodeToken(xml.StartElement{
		Name: e.makeName(RDFNamespace, "Description"),
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
		Name: e.makeName(RDFNamespace, "Description"),
	})
	if err != nil {
		return err
	}

	err = e.EncodeToken(xml.EndElement{
		Name: e.makeName(RDFNamespace, "RDF"),
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

func (e *encoder) makeName(ns, local string) xml.Name {
	pfx, ok := e.nsToPrefix[ns]
	if !ok {
		panic("namespace not registered: " + ns)
	}
	return xml.Name{Local: pfx + ":" + local}
}
