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
	"net/url"
	"sort"

	"golang.org/x/exp/maps"
)

// Packet represents an XMP packet.
type Packet struct {
	// Properties maps namespaces to models.
	Properties map[xml.Name]Value

	// About (optional) is the URL of the resource described by the XMP packet.
	About *url.URL
}

func NewPacket() *Packet {
	return &Packet{
		Properties: make(map[xml.Name]Value),
	}
}

func (p *Packet) getNamespaces() map[string]struct{} {
	m := make(map[string]struct{})
	for key, value := range p.Properties {
		m[key.Space] = struct{}{}
		getValueNameSpaces(m, value)
	}
	return m
}

func getValueNameSpaces(m map[string]struct{}, v Value) {
	var q Q

	switch v := v.(type) {
	case TextValue:
		q = v.Q
	case URIValue:
		q = v.Q
	case StructValue:
		for key, val := range v.Value {
			m[key.Space] = struct{}{}
			getValueNameSpaces(m, val)
		}
		q = v.Q
	case ArrayValue:
		for _, val := range v.Value {
			getValueNameSpaces(m, val)
		}
		q = v.Q
	default:
		panic("unexpected value type") // TODO(voss): remove
	}

	for _, q := range q {
		m[q.Name.Space] = struct{}{}
		getValueNameSpaces(m, q.Value)
	}
}

// Value is one of [TextValue], [URIValue], [StructValue], or [ArrayValue].
type Value interface {
	isValue()
}

// A Qualifier can be used to attach additional information to a [Value].
type Qualifier struct {
	Name  xml.Name
	Value Value
}

// Q stores the qualifiers of a [Value].
type Q []Qualifier

// isValue implements the [Value] interface.
func (q Q) isValue() {
}

// getLang appends the xml:lang attribute if needed.
func (q Q) getLang(attr []xml.Attr) []xml.Attr {
	for _, q := range q {
		if q.Name == attrXMLLang {
			if v, ok := q.Value.(TextValue); ok {
				// We don't need to use .makeName() here, since the prefix
				// is always "xml".
				attr = append(attr, xml.Attr{Name: attrXMLLang, Value: v.Value})
			}
		}
	}
	return attr
}

// hasQualifiers returns true if there are any qualifiers other than xml:lang.
func (q Q) hasQualifiers() bool {
	for _, q := range q {
		if q.Name != attrXMLLang {
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

// URIValue is a simple URI value.
type URIValue struct {
	Value *url.URL
	Q
}

// StructValue is an XMP structure.
type StructValue struct {
	Value map[xml.Name]Value
	Q
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

// ArrayType represents the type of an XMP array (unordered, ordered, or
// alternative).
type ArrayType int

// These are the possible array types in XMP.
const (
	Unordered ArrayType = iota + 1
	Ordered
	Alternative
)
