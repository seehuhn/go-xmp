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
)

type Encoder struct {
	buf *bytes.Buffer
	*xml.Encoder
	nsPrefix map[string]string
	prefixNS map[string]string
}

func NewEncoder() (*Encoder, error) {
	buf := &bytes.Buffer{}
	enc := xml.NewEncoder(buf)
	enc.Indent("", "  ") // TODO(voss): remove
	e := &Encoder{
		buf:      buf,
		Encoder:  enc,
		nsPrefix: make(map[string]string),
		prefixNS: make(map[string]string),
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

	e.addNamespace(rdfNS, "rdf")
	err = e.EncodeToken(xml.StartElement{
		Name: e.makeName(rdfNS, "RDF"),
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "xmlns:rdf"}, Value: rdfNS},
		},
	})
	if err != nil {
		return nil, err
	}

	return e, nil
}

func (e *Encoder) Close() error {
	err := e.EncodeToken(xml.EndElement{
		Name: e.makeName(rdfNS, "RDF"),
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

func (e *Encoder) EncodeValue(ns, name string, value Value) error {
	if value.IsZero() {
		return nil
	}

	err := e.EncodeToken(xml.StartElement{Name: e.makeName(ns, name)})
	if err != nil {
		return err
	}
	err = value.EncodeXMP(e)
	if err != nil {
		return err
	}
	err = e.EncodeToken(xml.EndElement{Name: e.makeName(ns, name)})
	if err != nil {
		return err
	}

	return nil
}

func (e *Encoder) makeName(space, local string) xml.Name {
	prefix, ok := e.nsPrefix[space]
	if !ok {
		panic("namespace not found")
	}
	return xml.Name{Local: prefix + ":" + local}
}

func (e *Encoder) addNamespace(ns, defaultPrefix string) string {
	if prefix, ok := e.nsPrefix[ns]; ok {
		return prefix
	}

	prefix := defaultPrefix
	if _, ok := e.prefixNS[prefix]; ok || prefix == "" {
		// choose a new prefix
		switch ns {
		default:
			panic("not implemented") // TODO(voss): implement
		}
	}

	e.nsPrefix[ns] = prefix
	e.prefixNS[prefix] = ns
	return prefix
}

const (
	rdfNS = "http://www.w3.org/1999/02/22-rdf-syntax-ns#"
)
