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
func (p *Packet) Encode(pretty bool) ([]byte, error) {
	ns := p.getNamespaces()

	e, err := newEncoder(p.About, ns, pretty)
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

		propName := e.makeName(name.Space, name.Local)
		tokens := e.appendProperty(nil, propName, value)
		for _, t := range tokens {
			err = e.EncodeToken(t)
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
func newEncoder(aboutURL *url.URL, nsUsed map[string]struct{}, pretty bool) (*encoder, error) {
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
	if pretty {
		enc.Indent("", "\t")
	}
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

func (e *encoder) appendProperty(tokens []xml.Token, name xml.Name, value Value) []xml.Token {
	// leave a space for the StartElement
	base := len(tokens)
	tokens = append(tokens, nil)

	var attr []xml.Attr

	switch value := value.(type) {
	case textValue:
		tokens = append(tokens, xml.CharData(value.Value))
	case uriValue:
		attr = append(attr, xml.Attr{
			Name:  e.makeName(RDFNamespace, "resource"),
			Value: value.Value.String(),
		})
	case structValue:
		envName := e.makeName(RDFNamespace, "Description")

		fields := maps.Keys(value.Value)
		sort.Slice(fields, func(i, j int) bool {
			if fields[i].Space != fields[j].Space {
				return fields[i].Space < fields[j].Space
			}
			return fields[i].Local < fields[j].Local
		})

		tokens = append(tokens, xml.StartElement{Name: envName})
		for _, field := range fields {
			fieldName := e.makeName(field.Space, field.Local)
			fieldValue := value.Value[field]
			tokens = e.appendProperty(tokens, fieldName, fieldValue)
		}
		tokens = append(tokens, xml.EndElement{Name: envName})
	case arrayValue:
		var tp string
		switch value.Type {
		case tpUnordered:
			tp = "Bag"
		case tpOrdered:
			tp = "Seq"
		case tpAlternative:
			tp = "Alt"
		default:
			panic("unexpected array type")
		}
		envName := e.makeName(RDFNamespace, tp)
		itemName := e.makeName(RDFNamespace, "li")

		tokens = append(tokens, xml.StartElement{Name: envName})
		for _, itemValue := range value.Value {
			tokens = e.appendProperty(tokens, itemName, itemValue)
		}
		tokens = append(tokens, xml.EndElement{Name: envName})
	default:
		panic("unexpected value type")
	}

	for _, q := range value.Qualifiers() {
		if q.Name == attrXMLLang {
			attr = append(attr, xml.Attr{
				Name:  e.makeName(xmlNamespace, "lang"),
				Value: q.Value.(textValue).Value,
			})
			continue
		}
		panic("not implemented")
	}

	if len(tokens) > base+1 {
		tokens[base] = xml.StartElement{
			Name: name,
			Attr: attr,
		}
		tokens = append(tokens, xml.EndElement{Name: name})
	} else {
		tokens[base] = jvxml.EmptyElement{
			Name: name,
			Attr: attr,
		}
	}

	return tokens
}
