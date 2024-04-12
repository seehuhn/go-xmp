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

type Value interface {
	IsZero() bool
	Qualifiers() []Qualifier
	EncodeXMP(*Encoder) error
	DecodeXMP([]xml.Token) error
}

// Q can be used to simplify the implementation of [Value] objects.
type Q []Qualifier

// Qualifiers implements [Value.Qualifiers].
func (q Q) Qualifiers() []Qualifier {
	return q
}

// Model is a group of XMP properties.
type Model interface {
	EncodeProperties(e *Encoder, prefix string) error
	NameSpaces() []string
	DefaultPrefix() string
}

// Packet represents an XMP packet.
type Packet struct {
	// Properties maps namespaces to models.
	Properties map[string]Model

	// About (optional) is the URL of the resource described by the XMP packet.
	About *url.URL
}

func (p *Packet) Encode() ([]byte, error) {
	e, err := NewEncoder()
	if err != nil {
		return nil, err
	}

	namespaces := maps.Keys(p.Properties)
	sort.Strings(namespaces)
	about := ""
	if p.About != nil {
		about = p.About.String()
	}
	for _, ns := range namespaces {
		model := p.Properties[ns]

		var attrs []xml.Attr
		attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "about"}, Value: about})
		for _, ns := range model.NameSpaces() {
			_, ok := e.nsPrefix[ns]
			if !ok {
				// TODO(voss): how to rewind this once the environment is closed?
				pfx := e.addNamespace(ns, model.DefaultPrefix())
				attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "xmlns:" + pfx}, Value: ns})
			}
		}
		err := e.EncodeToken(xml.StartElement{
			Name: e.makeName(rdfNS, "Description"),
			Attr: attrs,
		})
		if err != nil {
			return nil, err
		}

		err = model.EncodeProperties(e, ns)
		if err != nil {
			return nil, err
		}

		err = e.EncodeToken(xml.EndElement{
			Name: e.makeName(rdfNS, "Description"),
		})
	}

	err = e.Close()
	if err != nil {
		return nil, err
	}

	return e.buf.Bytes(), nil

}

type Qualifier struct {
	Name  xml.Name
	Value Value
}
