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
	"sync"

	"golang.org/x/exp/maps"
)

// Model is a group of XMP properties.
type Model interface {
	// NameSpaces populates the given map with all XML namespaces used by the
	// properties of the model.  The namespace of the model itself will only be
	// added to the map, if it is also used by a property.
	NameSpaces(map[string]struct{})

	// EncodeXMP encodes all the properties of the model to the given encoder.
	// This does not include the enclosing rdf:Description element.
	EncodeXMP(e *Encoder, prefix string) error
}

// Value is the value of an XMP property.
type Value interface {
	NameSpaces(map[string]struct{})

	IsZero() bool
	Qualifiers() []Qualifier
	EncodeXMP(*Encoder) error
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

// NameSpaces implements part of the [Value] interface.
func (q Q) NameSpaces(m map[string]struct{}) {
	for _, q := range q {
		m[q.Name.Space] = struct{}{}
		q.Value.NameSpaces(m)
	}
}

// Packet represents an XMP packet.
type Packet struct {
	// Models maps namespaces to models.
	Models map[string]Model

	// About (optional) is the URL of the resource described by the XMP packet.
	About *url.URL
}

// Encode encodes the packet to an XML byte slice.
func (p *Packet) Encode() ([]byte, error) {
	e, err := NewEncoder()
	if err != nil {
		return nil, err
	}

	namespaces := maps.Keys(p.Models)
	sort.Strings(namespaces)
	about := ""
	if p.About != nil {
		about = p.About.String()
	}
	for _, ns := range namespaces {
		model := p.Models[ns]

		var attrs []xml.Attr
		attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "about"}, Value: about})

		m := make(map[string]struct{})
		m[ns] = struct{}{}
		model.NameSpaces(m)
		for ns := range m {
			_, ok := e.nsPrefix[ns]
			if !ok {
				// TODO(voss): how to rewind this once the environment is closed?
				pfx := e.addNamespace(ns)
				attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "xmlns:" + pfx}, Value: ns})
			}
		}
		err := e.EncodeToken(xml.StartElement{
			Name: e.makeName(RDFNamespace, "Description"),
			Attr: attrs,
		})
		if err != nil {
			return nil, err
		}

		err = model.EncodeXMP(e, ns)
		if err != nil {
			return nil, err
		}

		err = e.EncodeToken(xml.EndElement{
			Name: e.makeName(RDFNamespace, "Description"),
		})
	}

	err = e.Close()
	if err != nil {
		return nil, err
	}

	return e.buf.Bytes(), nil
}

// RegisterModel registers a model reader for a given namespace.
func RegisterModel(nameSpace, defaultLocal string, update func(Model, string, []xml.Token) (Model, error)) {
	modelMutex.Lock()
	defer modelMutex.Unlock()
	modelReaders[nameSpace] = &modelInfo{nameSpace, defaultLocal, update}
}

func nsPrefix(ns string) string {
	modelMutex.Lock()
	info, ok := modelReaders[ns]
	modelMutex.Unlock()

	var local string
	if ok {
		local = info.defaultLocal
	}
	if local == "" && ns == RDFNamespace {
		local = "rdf"
	}

	return local
}

type modelInfo struct {
	nameSpace, defaultLocal string
	update                  func(Model, string, []xml.Token) (Model, error)
}

var (
	modelReaders = make(map[string]*modelInfo)
	modelMutex   sync.Mutex
)
