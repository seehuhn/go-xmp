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
	"time"
)

// Text represents a simple text value.
type Text struct {
	Value string
	Q
}

func (t Text) String() string {
	return t.Value
}

// IsZero implements the [Value] interface.
func (t Text) IsZero() bool {
	return t.Value == ""
}

// EncodeXMP implements the [Value] interface.
func (t Text) EncodeXMP(e *Encoder) error {
	token := xml.CharData(t.Value)
	return e.EncodeToken(token)
}

// DecodeAnother implements the [Value] interface.
func (Text) DecodeAnother(tokens []xml.Token) (Value, error) {
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

// ProperName represents a proper name.
type ProperName struct {
	Text
}

// DecodeAnother implements the [Value] interface.
func (ProperName) DecodeAnother(tokens []xml.Token) (Value, error) {
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

// Date represents a date and time.
type Date struct {
	Value      time.Time
	NumOmitted int // 1=omit nano, 2=omit sec, 3=omit time, 4=omit day, 5=month
	Q
}

// IsZero implements the [Value] interface.
func (d Date) IsZero() bool {
	return d.Value.IsZero()
}

var dateFormats = []string{
	"2006-01-02T15:04:05.999999999Z07:00",
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02T15:04Z07:00",
	"2006-01-02",
	"2006-01",
	"2006",
}

// EncodeXMP implements the [Value] interface.
func (d Date) EncodeXMP(e *Encoder) error {
	numOmitted := d.NumOmitted
	numOmitted = min(numOmitted, len(dateFormats)-1)
	numOmitted = max(numOmitted, 0)
	format := dateFormats[numOmitted]
	return e.EncodeToken(xml.CharData(d.Value.Format(format)))
}

// DecodeAnother implements the [Value] interface.
func (Date) DecodeAnother(tokens []xml.Token) (Value, error) {
	var dateString string
	for _, token := range tokens {
		switch token := token.(type) {
		case xml.CharData:
			dateString += string(token)
		case xml.StartElement, xml.EndElement:
			return nil, errMalformedXMP
		}
	}

	for i, format := range dateFormats {
		t, err := time.Parse(format, dateString)
		if err == nil {
			return Date{Value: t, NumOmitted: i}, nil
		}
	}
	return nil, errMalformedXMP
}

// UnorderedArray represents an unordered array of values.
type UnorderedArray[T Value] struct {
	Values []T
	Q
}

func (a UnorderedArray[T]) NameSpaces(m map[string]struct{}) {
	a.Q.NameSpaces(m)
	for _, v := range a.Values {
		v.NameSpaces(m)
	}
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

// DecodeAnother implements the [Value] interface.
func (UnorderedArray[T]) DecodeAnother(tokens []xml.Token) (Value, error) {
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

// OrderedArray represents an ordered array of values.
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

// DecodeAnother implements the [Value] interface.
func (OrderedArray[T]) DecodeAnother(tokens []xml.Token) (Value, error) {
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
