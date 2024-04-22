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

// WriterOptions can be used to control the output format of the [Packet.Write]
// method.
type WriterOptions struct {
	Pretty bool
}

// Write writes the XMP packet to the given writer.
func (p *Packet) Write(w io.Writer, opt *WriterOptions) error {
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

		propName := e.makeName(name.Space, name.Local)
		tokens := e.appendProperty(nil, propName, value)
		for _, t := range tokens {
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

// An encoder writes XMP data to an output stream.
type encoder struct {
	w io.Writer
	*jvxml.Encoder
	nsToPrefix map[string]string
	prefixToNS map[string]string
}

// newEncoder returns a new encoder that writes to w.
func (p *Packet) newEncoder(w io.Writer, opt *WriterOptions) (*encoder, error) {
	nsUsed := p.getNamespaces()

	nsUsed[xmlNamespace] = struct{}{}
	nsUsed[rdfNamespace] = struct{}{}

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
		Name: e.makeName(rdfNamespace, "RDF"),
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
	attrs = append(attrs, xml.Attr{Name: e.makeName(rdfNamespace, "about"), Value: about})
	err = e.EncodeToken(xml.StartElement{
		Name: e.makeName(rdfNamespace, "Description"),
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
		Name: e.makeName(rdfNamespace, "Description"),
	})
	if err != nil {
		return err
	}

	err = e.EncodeToken(xml.EndElement{
		Name: e.makeName(rdfNamespace, "RDF"),
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
	rdfValue := e.makeName(rdfNamespace, "value")
	rdfResource := e.makeName(rdfNamespace, "resource")
	attrParseTypeResource := xml.Attr{
		Name:  e.makeName(rdfNamespace, "parseType"),
		Value: "Resource",
	}

	switch val := value.(type) {
	case TextValue:
		// Possible ways to encode the value:
		//
		// option 1 (no non-lang qualifiers):
		// <test:prop xml:lang="te-ST">value</test:prop>
		//
		// option 2 (with non-lang qualifiers):
		// <test:prop xml:lang="te-ST">
		//   <rdf:Description>
		//     <rdf:value>value</rdf:value>
		//     <test:q>v</test:q>
		//   </rdf:Description>
		// </test:prop>
		//
		// option 3a (with simple non-lang qualifiers, shorter form):
		// <test:prop xml:lang="te-ST">
		//   <rdf:Description test:q="q" rdf:value="value" />
		// </test:prop>
		//
		// option 3b (with non-lang qualifiers, shorter form):
		// <test:prop xml:lang="te-ST">
		//   <rdf:Description test:q1="v1" rdf:value="value">
		//     <test:q2 rdf:resource="test:v2"/>
		//   </rdf:Description>
		// </test:prop>
		//
		// option 4 (with non-lang qualifiers, shorter form):
		// <test:prop xml:lang="te-ST" rdf:parseType="Resource">
		//   <rdf:value>value</rdf:value>
		//   <test:q>v</test:q>
		// </test:prop>
		//
		// option 5 (with simple qualifiers, compact form):
		// <test:prop xml:lang="te-ST" test:q="q" rdf:value="value"/>

		if !val.Q.hasQualifiers() { // use option 1
			attr := val.Q.getLang(nil)
			tokens = append(tokens,
				xml.StartElement{Name: name, Attr: attr},
				xml.CharData(val.Value),
				xml.EndElement{Name: name},
			)
		} else if val.Q.allSimple() { // use option 5
			attr := make([]xml.Attr, 0, len(val.Q)+1)
			for _, q := range val.Q {
				attr = append(attr, xml.Attr{Name: e.makeName(q.Name.Space, q.Name.Local), Value: q.Value.(TextValue).Value})
			}
			attr = append(attr, xml.Attr{Name: rdfValue, Value: val.Value})
			tokens = append(tokens, jvxml.EmptyElement{Name: name, Attr: attr})
		} else { // use option 4
			attr := val.Q.getLang(nil)
			attr = append(attr, xml.Attr{
				Name:  e.makeName(rdfNamespace, "parseType"),
				Value: "Resource",
			})
			tokens = append(tokens,
				xml.StartElement{Name: name, Attr: attr},
				xml.StartElement{Name: rdfValue},
				xml.CharData(val.Value),
				xml.EndElement{Name: rdfValue},
			)
			for _, q := range val.Q {
				if q.Name == attrXMLLang {
					continue
				}
				tokens = e.appendProperty(tokens, e.makeName(q.Name.Space, q.Name.Local), q.Value)
			}
			tokens = append(tokens, xml.EndElement{Name: name})
		}

	case URIValue:
		// Possible ways to encode the value:
		//
		// option 1 (no non-lang qualifiers):
		// <test:prop xml:lang="te-ST" rdf:resource="http://example.com"/>
		//
		// option 2 (with non-lang qualifiers):
		// <test:prop xml:lang="te-ST">
		//   <rdf:Description>
		//     <rdf:value rdf:resource="http://example.com"/>
		//     <test:q rdf:resource="http://example.com"/>
		//   </rdf:Description>
		// </test:prop>
		//
		// option 3 (with non-lang qualifiers, shorter form):
		// <test:prop xml:lang="te-ST">
		//   <rdf:Description test:q="q">
		//     <rdf:value rdf:resource="http://example.com"/>
		//   </rdf:Description>
		// </test:prop>
		//
		// option 4 (with non-lang qualifiers, shorter form):
		// <test:prop xml:lang="te-ST" rdf:parseType="Resource">
		//   <rdf:value rdf:resource="http://example.com"/>
		//   <test:q rdf:resource="http://example.com"/>
		// </test:prop>

		attr := val.Q.getLang(nil)

		if val.Q.hasQualifiers() { // use option 4
			attr = append(attr, xml.Attr{
				Name:  e.makeName(rdfNamespace, "parseType"),
				Value: "Resource",
			})
			tokens = append(tokens,
				xml.StartElement{Name: name, Attr: attr},
				jvxml.EmptyElement{Name: rdfValue,
					Attr: []xml.Attr{{Name: rdfResource, Value: val.Value.String()}},
				},
			)
			for _, q := range val.Q {
				if q.Name == attrXMLLang {
					continue
				}
				tokens = e.appendProperty(tokens, e.makeName(q.Name.Space, q.Name.Local), q.Value)
			}
			tokens = append(tokens, xml.EndElement{Name: name})
		} else { // use option 1
			attr = append(attr, xml.Attr{
				Name:  rdfResource,
				Value: val.Value.String(),
			})
			tokens = append(tokens,
				jvxml.EmptyElement{Name: name, Attr: attr},
			)
		}

	case StructValue:
		// Possible ways to encode the value:
		//
		// option 1a (no non-lang qualifiers):
		// <test:prop xml:lang="te-ST">
		//   <rdf:Description>
		//     <test:a>1</test:a>
		//     <test:b>2</test:b>
		//   </rdf:Description>
		// </test:prop>
		//
		// option 1b (no non-lang qualifiers):
		// <test:prop xml:lang="te-ST" rdf:parseType="Resource">
		//   <test:a>1</test:a>
		//   <test:b>2</test:b>
		// </test:prop>
		//
		// option 1c (no non-lang qualifiers, simple values, shortened):
		// <test:prop xml:lang="te-ST" test:a="1", test:b="2"/>
		//
		// option 2 (with non-lang qualifiers):
		// <test:prop xml:lang="te-ST">
		//   <rdf:Description>
		//	   <rdf:value>
		//       <rdf:Description>
		//         <test:a>1</test:a>
		//         <test:b>2</test:b>
		//       </rdf:Description>
		//     </rdf:value>
		//     <test:q>v</test:q>
		//   </rdf:Description>
		// </test:prop>
		//
		// option 3 (with non-lang qualifiers, shorter form):
		// <test:prop xml:lang="te-ST">
		//   <rdf:Description test:q1="v">
		//	   <rdf:value>
		//       <rdf:Description test:a="1">
		//         <test:b rdf:resource="test:b"/>
		//       </rdf:Description>
		//     </rdf:value>
		//     <test:q2 rdf:resource="test:q2"/>
		//   </rdf:Description>
		// </test:prop>
		//
		// option 4 (with non-lang qualifiers, shorter form):
		// <test:prop xml:lang="te-ST" rdf:parseType="Resource">
		//	 <rdf:value rdf:parseType="Resource">
		//     <test:a>1</test:a>
		//     <test:b>2</test:b>
		//   </rdf:value>
		//   <test:q>v</test:q>
		// </test:prop>

		attr := val.Q.getLang(nil)

		fieldNames := val.fieldNames()
		if val.Q.hasQualifiers() { // use option 4
			attr = append(attr, attrParseTypeResource)
			tokens = append(tokens,
				xml.StartElement{Name: name, Attr: attr},
				xml.StartElement{Name: rdfValue, Attr: []xml.Attr{attrParseTypeResource}},
			)
			for _, fieldName := range fieldNames {
				fName := e.makeName(fieldName.Space, fieldName.Local)
				tokens = e.appendProperty(tokens, fName, val.Value[fieldName])
			}
			tokens = append(tokens, xml.EndElement{Name: rdfValue})
			for _, q := range val.Q {
				if q.Name == attrXMLLang {
					continue
				}
				qName := e.makeName(q.Name.Space, q.Name.Local)
				tokens = e.appendProperty(tokens, qName, q.Value)
			}
			tokens = append(tokens, xml.EndElement{Name: name})
		} else if val.allSimple() && len(val.Value) > 0 { // use option 1c
			for _, fieldName := range fieldNames {
				fName := e.makeName(fieldName.Space, fieldName.Local)
				attr = append(attr, xml.Attr{Name: fName, Value: val.Value[fieldName].(TextValue).Value})
			}
			tokens = append(tokens, jvxml.EmptyElement{Name: name, Attr: attr})
		} else { // use option 1b
			attr = append(attr, attrParseTypeResource)
			tokens = append(tokens, xml.StartElement{Name: name, Attr: attr})
			for _, fieldName := range fieldNames {
				fName := e.makeName(fieldName.Space, fieldName.Local)
				tokens = e.appendProperty(tokens, fName, val.Value[fieldName])
			}
			tokens = append(tokens, xml.EndElement{Name: name})
		}

	case ArrayValue:
		// Possible ways to encode the value:
		//
		// option 1 (no non-lang qualifiers):
		// <test:prop xml:lang="te-ST">
		//   <rdf:Seq>
		//     <rdf:li>1</rdf:li>
		//     <rdf:li>2</rdf:li>
		//   </rdf:Seq>
		// </test:prop>
		//
		// option 2 (with non-lang qualifiers):
		// <test:prop xml:lang="te-ST">
		//   <rdf:Description>
		//     <rdf:value>
		//       <rdf:Seq>
		//         <rdf:li>1</rdf:li>
		//         <rdf:li>2</rdf:li>
		//       </rdf:Seq>
		//     </rdf:value>
		//     <test:q>v</test:q>
		//   </rdf:Description>
		// </test:prop>
		//
		// option 3 (with non-lang qualifiers, shorter form):
		// <test:prop xml:lang="te-ST">
		//   <rdf:Description test:q="v">
		//     <rdf:value>
		//       <rdf:Seq>
		//         <rdf:li>1</rdf:li>
		//         <rdf:li>2</rdf:li>
		//       </rdf:Seq>
		//     </rdf:value>
		//   </rdf:Description>
		// </test:prop>
		//
		// option 4 (with non-lang qualifiers, shorter form):
		// <test:prop xml:lang="te-ST" rdf:parseType="Resource">
		//   <rdf:value>
		//     <rdf:Seq>
		//       <rdf:li>1</rdf:li>
		//       <rdf:li>2</rdf:li>
		//     </rdf:Seq>
		//   </rdf:value>
		//   <test:q>v</test:q>
		// </test:prop>

		attr := val.Q.getLang(nil)

		var env string
		switch val.Type {
		case Unordered:
			env = "Bag"
		case Ordered:
			env = "Seq"
		case Alternative:
			env = "Alt"
		default:
			panic("unexpected array type")
		}
		envName := e.makeName(rdfNamespace, env)
		liName := e.makeName(rdfNamespace, "li")

		if val.Q.hasQualifiers() { // use option 4
			attr = append(attr, attrParseTypeResource)
			tokens = append(tokens,
				xml.StartElement{Name: name, Attr: attr},
				xml.StartElement{Name: rdfValue},
				xml.StartElement{Name: envName})
			for _, v := range val.Value {
				tokens = e.appendProperty(tokens, liName, v)
			}
			tokens = append(tokens, xml.EndElement{Name: envName})
			tokens = append(tokens, xml.EndElement{Name: rdfValue})
			for _, q := range val.Q {
				if q.Name == attrXMLLang {
					continue
				}
				qName := e.makeName(q.Name.Space, q.Name.Local)
				tokens = e.appendProperty(tokens, qName, q.Value)
			}
			tokens = append(tokens, xml.EndElement{Name: name})
		} else { // use option 1
			tokens = append(tokens,
				xml.StartElement{Name: name, Attr: attr},
				xml.StartElement{Name: envName})
			for _, v := range val.Value {
				tokens = e.appendProperty(tokens, liName, v)
			}
			tokens = append(tokens,
				xml.EndElement{Name: envName},
				xml.EndElement{Name: name})
		}

	default:
		panic("unreachable")
	}

	return tokens
}
