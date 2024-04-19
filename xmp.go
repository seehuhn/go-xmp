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

var (
	elemRDFRoot        = xml.Name{Space: RDFNamespace, Local: "RDF"}
	elemRDFDescription = xml.Name{Space: RDFNamespace, Local: "Description"}
	elemRDFBag         = xml.Name{Space: RDFNamespace, Local: "Bag"}
	elemRDFSeq         = xml.Name{Space: RDFNamespace, Local: "Seq"}
	elemRDFAlt         = xml.Name{Space: RDFNamespace, Local: "Alt"}

	attrRDFAbout     = xml.Name{Space: RDFNamespace, Local: "about"}
	attrRDFDataType  = xml.Name{Space: RDFNamespace, Local: "datatype"}
	attrRDFID        = xml.Name{Space: RDFNamespace, Local: "ID"}
	attrRDFNodeID    = xml.Name{Space: RDFNamespace, Local: "nodeID"}
	attrRDFParseType = xml.Name{Space: RDFNamespace, Local: "parseType"}
	attrRDFResource  = xml.Name{Space: RDFNamespace, Local: "resource"}
	attrRDFValue     = xml.Name{Space: RDFNamespace, Local: "value"}
	attrXMLLang      = xml.Name{Space: xmlNamespace, Local: "lang"}
)
