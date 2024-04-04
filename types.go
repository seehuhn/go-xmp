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

import "encoding/xml"

type Text struct {
	Value string
	Q
}

func (t Text) EncodeValue(e *Encoder) error {
	token := xml.CharData(t.Value)
	return e.EncodeToken(token)
}

type ProperName struct {
	Text
}

type UnorderedArray[T Value] struct {
	Values []T
	Q
}

func (a UnorderedArray[T]) EncodeValue(e *Encoder) error {
	prefix := "rdf" // TODO(voss)
	outer := xml.Name{
		Local: prefix + ":Bag",
	}
	err := e.EncodeToken(xml.StartElement{Name: outer})
	if err != nil {
		return err
	}

	inner := xml.Name{
		Local: prefix + ":li",
	}
	for _, v := range a.Values {
		err = e.EncodeToken(xml.StartElement{Name: inner})
		if err != nil {
			return err
		}
		err = v.EncodeValue(e)
		if err != nil {
			return err
		}
		err = e.EncodeToken(xml.EndElement{Name: inner})
		if err != nil {
			return err
		}
	}

	err = e.EncodeToken(xml.EndElement{Name: outer})
	if err != nil {
		return err
	}
	return nil
}
