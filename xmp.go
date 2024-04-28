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
	"errors"
	"net/url"
	"sort"

	"golang.org/x/exp/maps"
	"golang.org/x/text/language"
	"seehuhn.de/go/xmp/jvxml"
)

// Packet represents an XMP packet.
type Packet struct {
	// Properties maps namespaces to models.
	Properties map[xml.Name]Raw

	// About (optional) is the URL of the resource described by the XMP packet.
	About *url.URL

	nsToPrefix map[string]string
}

// NewPacket allocates a new XMP packet.
func NewPacket() *Packet {
	return &Packet{
		Properties: make(map[xml.Name]Raw),
	}
}

// RegisterPrefix registers a namespace prefix.
func (p *Packet) RegisterPrefix(ns, prefix string) {
	if p.nsToPrefix == nil {
		p.nsToPrefix = make(map[string]string)
	}
	p.nsToPrefix[ns] = prefix
}

// SetValue stores the given value in the packet.
func (p *Packet) SetValue(namespace, propertyName string, value Value) {
	if !isValidPropertyName(xml.Name{Space: namespace, Local: propertyName}) {
		panic("invalid property name")
	}
	name := xml.Name{Space: namespace, Local: propertyName}
	if !isValidPropertyName(name) {
		panic("invalid property name")
	}
	p.Properties[name] = value.GetXMP(p)
}

// ClearValue removes the given property from the packet.
func (p *Packet) ClearValue(namespace, propertyName string) {
	name := xml.Name{Space: namespace, Local: propertyName}
	delete(p.Properties, name)
}

// GetValue retrieves the value of the given property from the packet.
//
// In case the value is not found, [ErrNotFound] is returned.
// If the value exists but has the wrong format, [ErrInvalid] is returned.
func GetValue[E Value](p *Packet, namespace, propertyName string) (E, error) {
	var zero E
	name := xml.Name{Space: namespace, Local: propertyName}
	xmpData, ok := p.Properties[name]
	if !ok {
		return zero, ErrNotFound
	}
	u, err := zero.DecodeAnother(xmpData)
	if err != nil {
		return zero, err
	}
	return u.(E), nil
}

// Raw is one of [Text], [URL], [RawStruct], or [RawArray].
type Raw interface {
	getNamespaces(m map[string]struct{})
	appendXML(tokens []xml.Token, name xml.Name) []xml.Token
}

// A Qualifier can be used to attach additional information to the value
// of an XMP property.
type Qualifier struct {
	Name  xml.Name
	Value Raw
}

// Language returns a qualifier which specifies the language of a value.
func Language(l language.Tag) Qualifier {
	return Qualifier{
		Name:  nameXMLLang,
		Value: Text{V: l.String()},
	}
}

// Q represents a list of qualifiers.
type Q []Qualifier

// StripLanguage returns the language qualifier of a [Q] and
// a new [Q] with the language qualifier removed.
// If no language qualifier is present, [language.Und] is returned.
func (q Q) StripLanguage() (language.Tag, Q) {
	var lang language.Tag
	var stripped Q
	for _, q := range q {
		if q.Name == nameXMLLang {
			if v, ok := q.Value.(Text); ok {
				if l, err := language.Parse(v.V); err == nil && lang == language.Und {
					lang = l
				}
			}
		} else {
			stripped = append(stripped, q)
		}
	}
	return lang, stripped
}

// WithLanguage returns a new [Q] with the given language qualifier.
// Any pre-existing language qualifier is removed.
func (q Q) WithLanguage(l language.Tag) Q {
	res := make(Q, 0, len(q)+1)
	res = append(res, Language(l))
	for _, q := range q {
		if q.Name != nameXMLLang {
			res = append(res, q)
		}
	}
	return res
}

// getLangAttr appends the xml:lang attribute if needed.
func (q Q) getLangAttr(attr []xml.Attr) []xml.Attr {
	for _, q := range q {
		if q.Name == nameXMLLang {
			if v, ok := q.Value.(Text); ok {
				attr = append(attr, xml.Attr{Name: nameXMLLang, Value: v.V})
			}
		}
	}
	return attr
}

// hasQualifiers returns true if there are any qualifiers other than xml:lang.
func (q Q) hasQualifiers() bool {
	for _, q := range q {
		if q.Name != nameXMLLang {
			return true
		}
	}
	return false
}

// hasQualifiers returns true if there are any qualifiers other than xml:lang.
func (q Q) allSimple() bool {
	for _, q := range q {
		if val, ok := q.Value.(Text); !ok || len(val.Q) > 0 {
			return false
		}
	}
	return true
}

// Text is a simple text (i.e. non-URI) value.
//
// Text implements both the [Value] and [Raw] interfaces.
type Text struct {
	V string
	Q
}

// NewText creates a new XMP text value.
func NewText(s string, qualifiers ...Qualifier) Text {
	return Text{V: s, Q: Q(qualifiers)}
}

func (t Text) String() string {
	return t.V
}

// IsZero implements the [Value] interface.
func (t Text) IsZero() bool {
	return t.V == "" && len(t.Q) == 0
}

// GetXMP implements the [Value] interface.
func (t Text) GetXMP(*Packet) Raw {
	return t
}

// DecodeAnother implements the [Value] interface.
func (Text) DecodeAnother(val Raw) (Value, error) {
	v, ok := val.(Text)
	if !ok {
		return nil, ErrInvalid
	}
	return Text{v.V, v.Q}, nil
}

// getNamespaces implements the [Raw] interface.
func (t Text) getNamespaces(m map[string]struct{}) {
	for _, q := range t.Q {
		m[q.Name.Space] = struct{}{}
		q.Value.getNamespaces(m)
	}
}

// appendXML implements the [Raw] interface.
func (t Text) appendXML(tokens []xml.Token, name xml.Name) []xml.Token {
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

	if !t.Q.hasQualifiers() { // use option 1
		attr := t.Q.getLangAttr(nil)
		tokens = append(tokens,
			xml.StartElement{Name: name, Attr: attr},
			xml.CharData(t.V),
			xml.EndElement{Name: name},
		)
	} else if t.Q.allSimple() { // use option 5
		attr := make([]xml.Attr, 0, len(t.Q)+1)
		for _, q := range t.Q {
			attr = append(attr,
				xml.Attr{Name: q.Name, Value: q.Value.(Text).V})
		}
		attr = append(attr, xml.Attr{Name: nameRDFValue, Value: t.V})
		tokens = append(tokens, jvxml.EmptyElement{Name: name, Attr: attr})
	} else { // use option 4
		attr := t.Q.getLangAttr(nil)
		attr = append(attr, attrParseTypeResource)
		tokens = append(tokens,
			xml.StartElement{Name: name, Attr: attr},
			xml.StartElement{Name: nameRDFValue},
			xml.CharData(t.V),
			xml.EndElement{Name: nameRDFValue},
		)
		for _, q := range t.Q {
			if q.Name == nameXMLLang {
				continue
			}
			tokens = q.Value.appendXML(tokens, q.Name)
		}
		tokens = append(tokens, xml.EndElement{Name: name})
	}
	return tokens
}

// URL is a simple URL (or URI) value.
//
// URL implements both the [Value] and [Raw] interfaces.
type URL struct {
	V *url.URL
	Q
}

// NewURL creates a new XMP URL value.
func NewURL(u *url.URL, qualifiers ...Qualifier) URL {
	return URL{V: u, Q: Q(qualifiers)}
}

func (u URL) String() string {
	return u.V.String()
}

// IsZero implements the [Value] interface.
func (u URL) IsZero() bool {
	return u.V == nil && len(u.Q) == 0
}

// GetXMP implements the [Value] interface.
func (u URL) GetXMP(*Packet) Raw {
	return u
}

// DecodeAnother implements the [Value] interface.
func (URL) DecodeAnother(val Raw) (Value, error) {
	v, ok := val.(URL)
	if !ok {
		return nil, ErrInvalid
	}
	return URL{v.V, v.Q}, nil
}

// getNamespaces implements the [Raw] interface.
func (u URL) getNamespaces(m map[string]struct{}) {
	for _, q := range u.Q {
		m[q.Name.Space] = struct{}{}
		q.Value.getNamespaces(m)
	}
}

// appendXML implements the [Raw] interface.
func (u URL) appendXML(tokens []xml.Token, name xml.Name) []xml.Token {
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

	attr := u.Q.getLangAttr(nil)

	if u.Q.hasQualifiers() { // use option 4
		attr = append(attr, attrParseTypeResource)
		tokens = append(tokens,
			xml.StartElement{Name: name, Attr: attr},
			jvxml.EmptyElement{Name: nameRDFValue,
				Attr: []xml.Attr{{Name: nameRDFResource, Value: u.V.String()}},
			},
		)
		for _, q := range u.Q {
			if q.Name == nameXMLLang {
				continue
			}
			tokens = q.Value.appendXML(tokens, q.Name)
		}
		tokens = append(tokens, xml.EndElement{Name: name})
	} else { // use option 1
		attr = append(attr, xml.Attr{
			Name:  nameRDFResource,
			Value: u.V.String(),
		})
		tokens = append(tokens,
			jvxml.EmptyElement{Name: name, Attr: attr},
		)
	}

	return tokens
}

// RawStruct is an XMP structure.
type RawStruct struct {
	Value map[xml.Name]Raw
	Q
}

// getNamespaces implements the [Raw] interface.
func (s RawStruct) getNamespaces(m map[string]struct{}) {
	for key, val := range s.Value {
		m[key.Space] = struct{}{}
		val.getNamespaces(m)
	}
	for _, q := range s.Q {
		m[q.Name.Space] = struct{}{}
		q.Value.getNamespaces(m)
	}
}

// appendXML implements the [Raw] interface.
func (s RawStruct) appendXML(tokens []xml.Token, name xml.Name) []xml.Token {
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

	attr := s.Q.getLangAttr(nil)

	fieldNames := s.fieldNames()
	if s.Q.hasQualifiers() { // use option 4
		attr = append(attr, attrParseTypeResource)
		tokens = append(tokens,
			xml.StartElement{Name: name, Attr: attr},
			xml.StartElement{Name: nameRDFValue, Attr: []xml.Attr{attrParseTypeResource}},
		)
		for _, fieldName := range fieldNames {
			tokens = s.Value[fieldName].appendXML(tokens, fieldName)
		}
		tokens = append(tokens, xml.EndElement{Name: nameRDFValue})
		for _, q := range s.Q {
			if q.Name == nameXMLLang {
				continue
			}
			tokens = q.Value.appendXML(tokens, q.Name)
		}
		tokens = append(tokens, xml.EndElement{Name: name})
	} else if s.allSimple() && len(s.Value) > 0 { // use option 1c
		for _, fieldName := range fieldNames {
			attr = append(attr, xml.Attr{
				Name:  fieldName,
				Value: s.Value[fieldName].(Text).V,
			})
		}
		tokens = append(tokens, jvxml.EmptyElement{Name: name, Attr: attr})
	} else { // use option 1b
		attr = append(attr, attrParseTypeResource)
		tokens = append(tokens, xml.StartElement{Name: name, Attr: attr})
		for _, fieldName := range fieldNames {
			tokens = s.Value[fieldName].appendXML(tokens, fieldName)
		}
		tokens = append(tokens, xml.EndElement{Name: name})
	}
	return tokens
}

// fieldNames returns the field names sorted by namespace and local name.
func (s *RawStruct) fieldNames() []xml.Name {
	fieldNames := maps.Keys(s.Value)
	sort.Slice(fieldNames, func(i, j int) bool {
		if fieldNames[i].Space != fieldNames[j].Space {
			return fieldNames[i].Space < fieldNames[j].Space
		}
		return fieldNames[i].Local < fieldNames[j].Local
	})
	return fieldNames
}

// allSimple returns true if all values are simple non-URI values, with no
// qualifiers.
func (s *RawStruct) allSimple() bool {
	for _, v := range s.Value {
		if v, ok := v.(Text); !ok || len(v.Q) > 0 {
			return false
		}
	}
	return true
}

// RawArray is an XMP array.
// This can be an unordered array, an ordered array, or an alternative array,
// depending on the value of the Type field.
type RawArray struct {
	Value []Raw
	Kind  RawArrayType
	Q
}

// getNamespaces implements the [Raw] interface.
func (a RawArray) getNamespaces(m map[string]struct{}) {
	for _, v := range a.Value {
		v.getNamespaces(m)
	}
	for _, q := range a.Q {
		m[q.Name.Space] = struct{}{}
		q.Value.getNamespaces(m)
	}
}

// appendXML implements the [Raw] interface.
func (a RawArray) appendXML(tokens []xml.Token, name xml.Name) []xml.Token {
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

	attr := a.Q.getLangAttr(nil)

	var envName xml.Name
	switch a.Kind {
	case Unordered:
		envName = nameRDFBag
	case Ordered:
		envName = nameRDFSeq
	case Alternative:
		envName = nameRDFAlt
	default:
		panic("unexpected array type")
	}

	if a.Q.hasQualifiers() { // use option 4
		attr = append(attr, attrParseTypeResource)
		tokens = append(tokens,
			xml.StartElement{Name: name, Attr: attr},
			xml.StartElement{Name: nameRDFValue},
			xml.StartElement{Name: envName})
		for _, v := range a.Value {
			tokens = v.appendXML(tokens, nameRDFLi)
		}
		tokens = append(tokens, xml.EndElement{Name: envName})
		tokens = append(tokens, xml.EndElement{Name: nameRDFValue})
		for _, q := range a.Q {
			if q.Name == nameXMLLang {
				continue
			}
			tokens = q.Value.appendXML(tokens, q.Name)
		}
		tokens = append(tokens, xml.EndElement{Name: name})
	} else { // use option 1
		tokens = append(tokens,
			xml.StartElement{Name: name, Attr: attr},
			xml.StartElement{Name: envName})
		for _, v := range a.Value {
			tokens = v.appendXML(tokens, nameRDFLi)
		}
		tokens = append(tokens,
			xml.EndElement{Name: envName},
			xml.EndElement{Name: name})
	}

	return tokens
}

// RawArrayType represents the type of an XMP array (unordered, ordered, or
// alternative).
type RawArrayType int

// These are the possible array types in XMP.
const (
	Unordered RawArrayType = iota + 1
	Ordered
	Alternative
)

// ErrInvalid is returned by [GetValue] when XMP data is present in the XML
// file, but the data does not have the expected structure.
var ErrInvalid = errors.New("invalid XMP data")

// ErrNotFound is returned by [GetValue] when a requested property is not
// present in the packet.
var ErrNotFound = errors.New("property not found")
