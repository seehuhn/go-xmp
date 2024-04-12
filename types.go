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
)

type Text struct {
	Value string
	Q
}

func (t Text) IsZero() bool {
	return t.Value == ""
}

func (t Text) EncodeXMP(e *Encoder) error {
	token := xml.CharData(t.Value)
	return e.EncodeToken(token)
}

func (t *Text) DecodeXMP(tokens []xml.Token) error {
	t.Value = ""
	for _, token := range tokens {
		switch token := token.(type) {
		case xml.CharData:
			t.Value += string(token)
		case xml.StartElement, xml.EndElement:
			return errMalformedXMP
		}
	}
	return nil
}

type ProperName struct {
	Text
}

type UnorderedArray[T Value] struct {
	Values []T
	Q
}

func (a UnorderedArray[T]) IsZero() bool {
	return len(a.Values) == 0
}

func (a UnorderedArray[T]) EncodeXMP(e *Encoder) error {
	outer := e.makeName(rdfNS, "Bag")
	err := e.EncodeToken(xml.StartElement{Name: outer})
	if err != nil {
		return err
	}

	inner := e.makeName(rdfNS, "li")
	for _, v := range a.Values {
		err = e.EncodeToken(xml.StartElement{Name: inner})
		if err != nil {
			return err
		}
		err = v.EncodeXMP(e)
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

func (a *UnorderedArray[T]) DecodeXMP(tokens []xml.Token) error {
	// An unordered array is encoded as a sequence of <rdf:li> elements, inside
	// an <rdf:Bag> element.  We ignore all other tokens (comments, etc).
	insideBag := false
	childLevel := 0
	var childStart int
	for i, t := range tokens {
		switch t := t.(type) {
		case xml.StartElement:
			isLi := t.Name.Space == rdfNS && t.Name.Local == "li"
			if childLevel > 0 {
				// pass
			} else if insideBag && childLevel == 0 && isLi {
				childStart = i + 1
			} else if t.Name.Space == rdfNS && t.Name.Local == "Bag" {
				insideBag = true
				a.Values = a.Values[:0]
			} else {
				return errMalformedXMP
			}
			if isLi {
				childLevel++
			}
		case xml.EndElement:
			if t.Name.Space == rdfNS && t.Name.Local == "li" {
				childLevel--
				if childLevel == 0 {
					var v T // TODO(voss): how to create a new instance of T?
					err := v.DecodeXMP(tokens[childStart:i])
					if err != nil {
						return err
					}
					a.Values = append(a.Values, v)
				}
			} else if insideBag && childLevel == 0 && t.Name.Space == rdfNS && t.Name.Local == "Bag" {
				insideBag = false
			} else {
				return errMalformedXMP
			}
		}
	}
	return nil
}

var (
	errMalformedXMP = errors.New("malformed XMP data")
)
