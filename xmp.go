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
	case textValue:
		q = v.Q
	case uriValue:
		q = v.Q
	case structValue:
		for key, val := range v.Value {
			m[key.Space] = struct{}{}
			getValueNameSpaces(m, val)
		}
		q = v.Q
	case arrayValue:
		for _, val := range v.Value {
			getValueNameSpaces(m, val)
		}
		q = v.Q
	default:
		panic("unexpected value type")
	}

	for _, q := range q {
		getValueNameSpaces(m, q.Value)
	}
}

// Value is the value of an XMP property.
type Value interface {
	Qualifiers() []Qualifier
}

// A Qualifier can be used to attach additional information to a [Value].
type Qualifier struct {
	Name  xml.Name
	Value Value
}

// Q is used to simplify the implementation of [Value] objects.
// It provides a default implementation of the Qualifiers method.
type Q []Qualifier

// Qualifiers implements part of the [Value] interface.
func (q Q) Qualifiers() []Qualifier {
	return q
}

// getLang appends the xml:lang attribute if needed.
func (q Q) getLang(attr []xml.Attr) []xml.Attr {
	for _, q := range q {
		if q.Name == attrXMLLang {
			if v, ok := q.Value.(textValue); ok {
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

// textValue represents a simple non-URI value.
type textValue struct {
	Value string
	Q
}

// uriValue represents a simple URI value.
type uriValue struct {
	Value *url.URL
	Q
}

type structValue struct {
	Value map[xml.Name]Value
	Q
}

// fieldNames returns the field names sorted by namespace and local name.
func (s *structValue) fieldNames() []xml.Name {
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
func (s *structValue) allSimple() bool {
	for _, v := range s.Value {
		if v, ok := v.(textValue); !ok || len(v.Q) > 0 {
			return false
		}
	}
	return true
}

type arrayValue struct {
	Value []Value
	Type  arrayType
	Q
}

type arrayType int

const (
	tpUnordered arrayType = iota + 1
	tpOrdered
	tpAlternative
)
