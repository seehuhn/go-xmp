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
	Properties map[xml.Name]Value

	// About (optional) is the URL of the resource described by the XMP packet.
	About *url.URL
}

// NewPacket allocates a new XMP packet.
func NewPacket() *Packet {
	return &Packet{
		Properties: make(map[xml.Name]Value),
	}
}

// Set stores the given value in the packet.
func (p *Packet) Set(nameSpace, propertyName string, value Type) {
	name := xml.Name{Space: nameSpace, Local: propertyName}
	if !isValidPropertyName(name) {
		panic("invalid property name")
	}
	p.Properties[name] = value.GetXMP()
}

// Get retrieves the value of the given property from the packet.
//
// In case the value is not found, [ErrNotFound] is returned.
// If the value exists but has the wrong format, [ErrInvalid] is returned.
func Get[E Type](val *E, p *Packet, nameSpace, propertyName string) error {
	name := xml.Name{Space: nameSpace, Local: propertyName}
	v, ok := p.Properties[name]
	if !ok {
		return ErrNotFound
	}
	u, err := (*val).DecodeAnother(v)
	if err != nil {
		return err
	}
	*val = u.(E)
	return nil
}

// Value is one of [TextValue], [URIValue], [StructValue], or [ArrayValue].
type Value interface {
	getNamespaces(m map[string]struct{})
	appendXML(tokens []xml.Token, name xml.Name) []xml.Token
}

// A Qualifier can be used to attach additional information to a [Value].
type Qualifier struct {
	Name  xml.Name
	Value Value
}

// Language returns a qualifier which specifies the language of a value.
func Language(l language.Tag) Qualifier {
	return Qualifier{
		Name:  nameXMLLang,
		Value: TextValue{Value: l.String()},
	}
}

// Q stores the qualifiers of a [Value].
type Q []Qualifier

// getLang appends the xml:lang attribute if needed.
func (q Q) getLang(attr []xml.Attr) []xml.Attr {
	for _, q := range q {
		if q.Name == nameXMLLang {
			if v, ok := q.Value.(TextValue); ok {
				// We don't need to use .makeName() here, since the prefix
				// is always "xml".
				attr = append(attr, xml.Attr{Name: nameXMLLang, Value: v.Value})
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
		if val, ok := q.Value.(TextValue); !ok || len(val.Q) > 0 {
			return false
		}
	}
	return true
}

// TextValue is a simple text (i.e. non-URI) value.
type TextValue struct {
	Value string
	Q
}

// getNamespaces implements the [Value] interface.
func (t TextValue) getNamespaces(m map[string]struct{}) {
	for _, q := range t.Q {
		m[q.Name.Space] = struct{}{}
		q.Value.getNamespaces(m)
	}
}

// appendXML implements the [Value] interface.
func (t TextValue) appendXML(tokens []xml.Token, name xml.Name) []xml.Token {
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
		attr := t.Q.getLang(nil)
		tokens = append(tokens,
			xml.StartElement{Name: name, Attr: attr},
			xml.CharData(t.Value),
			xml.EndElement{Name: name},
		)
	} else if t.Q.allSimple() { // use option 5
		attr := make([]xml.Attr, 0, len(t.Q)+1)
		for _, q := range t.Q {
			attr = append(attr,
				xml.Attr{Name: q.Name, Value: q.Value.(TextValue).Value})
		}
		attr = append(attr, xml.Attr{Name: nameRDFValue, Value: t.Value})
		tokens = append(tokens, jvxml.EmptyElement{Name: name, Attr: attr})
	} else { // use option 4
		attr := t.Q.getLang(nil)
		attr = append(attr, attrParseTypeResource)
		tokens = append(tokens,
			xml.StartElement{Name: name, Attr: attr},
			xml.StartElement{Name: nameRDFValue},
			xml.CharData(t.Value),
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

// GetXMP implements the [Type] interface.
func (t TextValue) GetXMP() Value {
	return t
}

// DecodeAnother implements the [Type] interface.
func (TextValue) DecodeAnother(v Value) (Type, error) {
	if v, ok := v.(TextValue); ok {
		return v, nil
	}
	return nil, ErrInvalid
}

// URIValue is a simple URI value.
type URIValue struct {
	Value *url.URL
	Q
}

// getNamespaces implements the [Value] interface.
func (u URIValue) getNamespaces(m map[string]struct{}) {
	for _, q := range u.Q {
		m[q.Name.Space] = struct{}{}
		q.Value.getNamespaces(m)
	}
}

// appendXML implements the [Value] interface.
func (u URIValue) appendXML(tokens []xml.Token, name xml.Name) []xml.Token {
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

	attr := u.Q.getLang(nil)

	if u.Q.hasQualifiers() { // use option 4
		attr = append(attr, attrParseTypeResource)
		tokens = append(tokens,
			xml.StartElement{Name: name, Attr: attr},
			jvxml.EmptyElement{Name: nameRDFValue,
				Attr: []xml.Attr{{Name: nameRDFResource, Value: u.Value.String()}},
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
			Value: u.Value.String(),
		})
		tokens = append(tokens,
			jvxml.EmptyElement{Name: name, Attr: attr},
		)
	}

	return tokens
}

// GetXMP implements the [Type] interface.
func (u URIValue) GetXMP() Value {
	return u
}

// DecodeAnother implements the [Type] interface.
func (URIValue) DecodeAnother(v Value) (Type, error) {
	if v, ok := v.(URIValue); ok {
		return v, nil
	}
	return nil, ErrInvalid
}

// StructValue is an XMP structure.
type StructValue struct {
	Value map[xml.Name]Value
	Q
}

// GetXMP implements the [Type] interface.
func (s StructValue) GetXMP() Value {
	return s
}

// DecodeAnother implements the [Type] interface.
func (StructValue) DecodeAnother(v Value) (Type, error) {
	if v, ok := v.(StructValue); ok {
		return v, nil
	}
	return nil, ErrInvalid
}

// getNamespaces implements the [Value] interface.
func (s StructValue) getNamespaces(m map[string]struct{}) {
	for key, val := range s.Value {
		m[key.Space] = struct{}{}
		val.getNamespaces(m)
	}
	for _, q := range s.Q {
		m[q.Name.Space] = struct{}{}
		q.Value.getNamespaces(m)
	}
}

// appendXML implements the [Value] interface.
func (s StructValue) appendXML(tokens []xml.Token, name xml.Name) []xml.Token {
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

	attr := s.Q.getLang(nil)

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
				Value: s.Value[fieldName].(TextValue).Value,
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
func (s *StructValue) fieldNames() []xml.Name {
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
func (s *StructValue) allSimple() bool {
	for _, v := range s.Value {
		if v, ok := v.(TextValue); !ok || len(v.Q) > 0 {
			return false
		}
	}
	return true
}

// ArrayValue is an XMP array.
// This can be an unordered array, an ordered array, or an alternative array,
// depending on the value of the Type field.
type ArrayValue struct {
	Value []Value
	Type  ArrayType
	Q
}

// GetXMP implements the [Type] interface.
func (a ArrayValue) GetXMP() Value {
	return a
}

// DecodeAnother implements the [Type] interface.
func (ArrayValue) DecodeAnother(v Value) (Type, error) {
	if v, ok := v.(ArrayValue); ok {
		return v, nil
	}
	return nil, ErrInvalid
}

// getNamespaces implements the [Value] interface.
func (a ArrayValue) getNamespaces(m map[string]struct{}) {
	for _, v := range a.Value {
		v.getNamespaces(m)
	}
	for _, q := range a.Q {
		m[q.Name.Space] = struct{}{}
		q.Value.getNamespaces(m)
	}
}

// appendXML implements the [Value] interface.
func (a ArrayValue) appendXML(tokens []xml.Token, name xml.Name) []xml.Token {
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

	attr := a.Q.getLang(nil)

	var envName xml.Name
	switch a.Type {
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

// ArrayType represents the type of an XMP array (unordered, ordered, or
// alternative).
type ArrayType int

// These are the possible array types in XMP.
const (
	Unordered ArrayType = iota + 1
	Ordered
	Alternative
)

// A Type is a data type which can be represented as XMP data.
type Type interface {
	// GetXMP returns the XMP representation of a value.
	GetXMP() Value

	DecodeAnother(Value) (Type, error)
}

var (
	// ErrInvalid is returned when XMP data is present but does not have
	// the expected structure.
	ErrInvalid = errors.New("invalid XMP data")

	// ErrNotFound is returned when a property is not present in the packet.
	ErrNotFound = errors.New("property not found")
)
