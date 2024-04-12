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

func (t Text) String() string {
	return t.Value
}

func (t Text) IsZero() bool {
	return t.Value == ""
}

func (t Text) EncodeXMP(e *Encoder) error {
	token := xml.CharData(t.Value)
	return e.EncodeToken(token)
}

func (_ Text) DecodeAnother(tokens []xml.Token) (Value, error) {
	var res Text
	for _, token := range tokens {
		switch token := token.(type) {
		case xml.CharData:
			res.Value += string(token)
		case xml.StartElement, xml.EndElement:
			return nil, errMalformedXMP
		}
	}
	return res, nil
}

type ProperName struct {
	Text
}

func (_ ProperName) DecodeAnother(tokens []xml.Token) (Value, error) {
	var res ProperName
	for _, token := range tokens {
		switch token := token.(type) {
		case xml.CharData:
			res.Value += string(token)
		case xml.StartElement, xml.EndElement:
			return nil, errMalformedXMP
		}
	}
	return res, nil
}

type UnorderedArray[T Value] struct {
	Values []T
	Q
}

func (a UnorderedArray[T]) IsZero() bool {
	return len(a.Values) == 0
}

func (a UnorderedArray[T]) EncodeXMP(e *Encoder) error {
	outer := e.makeName(RDFNameSpace, "Bag")
	err := e.EncodeToken(xml.StartElement{Name: outer})
	if err != nil {
		return err
	}

	inner := e.makeName(RDFNameSpace, "li")
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

func (_ UnorderedArray[T]) DecodeAnother(tokens []xml.Token) (Value, error) {
	var res UnorderedArray[T]
	insideBag := false
	childLevel := 0
	var childStart int
	for i, t := range tokens {
		// An unordered array is encoded as a sequence of <rdf:li> elements, inside
		// an <rdf:Bag> element.  We ignore all other tokens (comments, etc).
		switch t := t.(type) {
		case xml.StartElement:
			isLi := t.Name.Space == RDFNameSpace && t.Name.Local == "li"
			if childLevel > 0 {
				// pass
			} else if insideBag && childLevel == 0 && isLi {
				childStart = i + 1
			} else if t.Name.Space == RDFNameSpace && t.Name.Local == "Bag" {
				insideBag = true
				res.Values = res.Values[:0]
			} else {
				return nil, errMalformedXMP
			}
			if isLi {
				childLevel++
			}
		case xml.EndElement:
			var v T
			if t.Name.Space == RDFNameSpace && t.Name.Local == "li" {
				childLevel--
				if childLevel == 0 {
					val, err := v.DecodeAnother(tokens[childStart:i])
					if err != nil {
						return nil, err
					}
					res.Values = append(res.Values, val.(T))
				}
			} else if insideBag && childLevel == 0 && t.Name.Space == RDFNameSpace && t.Name.Local == "Bag" {
				insideBag = false
			} else {
				return nil, errMalformedXMP
			}
		}
	}
	return res, nil
}

type OrderedArray[T Value] struct {
	Values []T
	Q
}

func (a OrderedArray[T]) IsZero() bool {
	return len(a.Values) == 0
}

func (a OrderedArray[T]) EncodeXMP(e *Encoder) error {
	outer := e.makeName(RDFNameSpace, "Seq")
	err := e.EncodeToken(xml.StartElement{Name: outer})
	if err != nil {
		return err
	}

	inner := e.makeName(RDFNameSpace, "li")
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

func (_ OrderedArray[T]) DecodeAnother(tokens []xml.Token) (Value, error) {
	var res OrderedArray[T]
	insideBag := false
	childLevel := 0
	var childStart int
	for i, t := range tokens {
		// An unordered array is encoded as a sequence of <rdf:li> elements, inside
		// an <rdf:Bag> element.  We ignore all other tokens (comments, etc).
		switch t := t.(type) {
		case xml.StartElement:
			isLi := t.Name.Space == RDFNameSpace && t.Name.Local == "li"
			if childLevel > 0 {
				// pass
			} else if insideBag && childLevel == 0 && isLi {
				childStart = i + 1
			} else if t.Name.Space == RDFNameSpace && t.Name.Local == "Bag" {
				insideBag = true
				res.Values = res.Values[:0]
			} else {
				return nil, errMalformedXMP
			}
			if isLi {
				childLevel++
			}
		case xml.EndElement:
			var v T
			if t.Name.Space == RDFNameSpace && t.Name.Local == "li" {
				childLevel--
				if childLevel == 0 {
					val, err := v.DecodeAnother(tokens[childStart:i])
					if err != nil {
						return nil, err
					}
					res.Values = append(res.Values, val.(T))
				}
			} else if insideBag && childLevel == 0 && t.Name.Space == RDFNameSpace && t.Name.Local == "Bag" {
				insideBag = false
			} else {
				return nil, errMalformedXMP
			}
		}
	}
	return res, nil
}

var (
	errMalformedXMP = errors.New("malformed XMP data")
)
