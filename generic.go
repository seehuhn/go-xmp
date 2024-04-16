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
	"sort"

	"golang.org/x/exp/maps"
)

type genericModel struct {
	Properties map[string]genericValue
}

func (g *genericModel) NameSpaces(m map[string]struct{}) {
	for _, value := range g.Properties {
		value.NameSpaces(m)
	}
}

func (g *genericModel) EncodeXMP(e *Encoder, prefix string) error {
	names := maps.Keys(g.Properties)
	sort.Strings(names)

	for _, name := range names {
		value := g.Properties[name]
		err := e.EncodeProperty(prefix, name, value)
		if err != nil {
			return err
		}
	}
	return nil
}

type genericValue struct {
	Tokens []xml.Token
	Q
}

func decodeGenericValue(tokens []xml.Token, qq []Qualifier) (Value, error) {
	val := genericValue{
		Tokens: tokens,
		Q:      qq,
	}
	return val, nil
}

func (v genericValue) IsZero() bool {
	return false
}

func (v genericValue) NameSpaces(m map[string]struct{}) {
	v.Q.NameSpaces(m)
	for _, token := range v.Tokens {
		switch token := token.(type) {
		case xml.StartElement:
			m[token.Name.Space] = struct{}{}
		}
	}
}

func (v genericValue) EncodeXMP(e *Encoder) error {
	for _, token := range v.Tokens {
		err := e.EncodeToken(token)
		if err != nil {
			return err
		}
	}
	return nil
}

func updateGeneric(m Model, name string, tokens []xml.Token, qq []Qualifier) (Model, error) {
	var g *genericModel
	if m, ok := m.(*genericModel); ok {
		g = m
	} else {
		g = &genericModel{
			Properties: make(map[string]genericValue),
		}
	}

	value := genericValue{
		Tokens: tokens,
		Q:      qq,
	}
	g.Properties[name] = value

	return g, nil
}
