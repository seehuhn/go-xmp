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

	"golang.org/x/text/language"
)

// Text represents a simple text value.
type Text struct {
	Value string
	Q
}

// DecodeText decodes a simple text value.
func DecodeText(tokens []xml.Token) (Text, error) {
	var res Text
	for _, token := range tokens {
		switch token := token.(type) {
		case xml.CharData:
			res.Value += string(token)
		case xml.StartElement, xml.EndElement:
			return Text{}, errMalformedXMP
		}
	}
	return res, nil
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

// ProperName represents a proper name.
type ProperName struct {
	Text
}

// DecodeProperName decodes a proper name.
func DecodeProperName(tokens []xml.Token) (ProperName, error) {
	var res ProperName
	for _, token := range tokens {
		switch token := token.(type) {
		case xml.CharData:
			res.Value += string(token)
		case xml.StartElement, xml.EndElement:
			return ProperName{}, errMalformedXMP
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

// DecodeDate decodes a date and time object.
func DecodeDate(tokens []xml.Token) (Date, error) {
	var dateString string
	for _, token := range tokens {
		switch token := token.(type) {
		case xml.CharData:
			dateString += string(token)
		case xml.StartElement, xml.EndElement:
			return Date{}, errMalformedXMP
		}
	}

	for i, format := range dateFormats {
		t, err := time.Parse(format, dateString)
		if err == nil {
			return Date{Value: t, NumOmitted: i}, nil
		}
	}
	return Date{}, errMalformedXMP
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

// Locale represents an RFC 3066 language code.
type Locale struct {
	Language language.Tag
	Q
}

// DecodeLocale decodes an RFC 3066 language code.
func DecodeLocale(tokens []xml.Token) (Locale, error) {
	var text string
	for _, token := range tokens {
		switch token := token.(type) {
		case xml.CharData:
			text += string(token)
		case xml.StartElement, xml.EndElement:
			return Locale{}, errMalformedXMP
		}
	}
	tag, _ := language.Parse(text)
	return Locale{Language: tag}, nil
}

// IsZero implements the [Value] interface.
func (l Locale) IsZero() bool {
	return l.Language == language.Und
}

// EncodeXMP implements the [Value] interface.
func (l Locale) EncodeXMP(e *Encoder) error {
	token := xml.CharData(l.Language.String())
	return e.EncodeToken(token)
}

// UnorderedArray represents an unordered array of values.
type UnorderedArray[T Value] struct {
	Values []T
	Q
}

// DecodeUnorderedArray decodes an unordered array of values.
func DecodeUnorderedArray[T Value](tokens []xml.Token, decodeItem func([]xml.Token) (T, error)) (UnorderedArray[T], error) {
	var res UnorderedArray[T]
	insideBag := false
	childLevel := 0
	var childStart int
	for i, t := range tokens {
		// An unordered array is encoded as a sequence of <rdf:li> elements, inside
		// an <rdf:Bag> element.  We ignore all other tokens (comments, etc).
		switch t := t.(type) {
		case xml.StartElement:
			isLi := t.Name.Space == RDFNamespace && t.Name.Local == "li"
			if childLevel > 0 {
				// pass
			} else if insideBag && childLevel == 0 && isLi {
				childStart = i + 1
			} else if t.Name.Space == RDFNamespace && t.Name.Local == "Bag" {
				insideBag = true
				res.Values = res.Values[:0]
			} else {
				return UnorderedArray[T]{}, errMalformedXMP
			}
			if isLi {
				childLevel++
			}
		case xml.EndElement:
			if t.Name.Space == RDFNamespace && t.Name.Local == "li" {
				childLevel--
				if childLevel == 0 {
					val, err := decodeItem(tokens[childStart:i])
					if err != nil {
						return UnorderedArray[T]{}, err
					}
					res.Values = append(res.Values, val)
				}
			} else if insideBag && childLevel == 0 && t.Name.Space == RDFNamespace && t.Name.Local == "Bag" {
				insideBag = false
			} else {
				return UnorderedArray[T]{}, errMalformedXMP
			}
		}
	}
	return res, nil
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
	outer := e.makeName(RDFNamespace, "Bag")
	err := e.EncodeToken(xml.StartElement{Name: outer})
	if err != nil {
		return err
	}

	inner := e.makeName(RDFNamespace, "li")
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

// OrderedArray represents an ordered array of values.
type OrderedArray[T Value] struct {
	Values []T
	Q
}

// DecodeOrderedArray decodes an ordered array of values.
func DecodeOrderedArray[T Value](tokens []xml.Token, decodeItem func([]xml.Token) (T, error)) (OrderedArray[T], error) {
	var res OrderedArray[T]
	insideBag := false
	childLevel := 0
	var childStart int
	for i, t := range tokens {
		// An unordered array is encoded as a sequence of <rdf:li> elements, inside
		// an <rdf:Bag> element.  We ignore all other tokens (comments, etc).
		switch t := t.(type) {
		case xml.StartElement:
			isLi := t.Name.Space == RDFNamespace && t.Name.Local == "li"
			if childLevel > 0 {
				// pass
			} else if insideBag && childLevel == 0 && isLi {
				childStart = i + 1
			} else if t.Name.Space == RDFNamespace && t.Name.Local == "Bag" {
				insideBag = true
				res.Values = res.Values[:0]
			} else {
				return OrderedArray[T]{}, errMalformedXMP
			}
			if isLi {
				childLevel++
			}
		case xml.EndElement:
			if t.Name.Space == RDFNamespace && t.Name.Local == "li" {
				childLevel--
				if childLevel == 0 {
					val, err := decodeItem(tokens[childStart:i])
					if err != nil {
						return OrderedArray[T]{}, err
					}
					res.Values = append(res.Values, val)
				}
			} else if insideBag && childLevel == 0 && t.Name.Space == RDFNamespace && t.Name.Local == "Bag" {
				insideBag = false
			} else {
				return OrderedArray[T]{}, errMalformedXMP
			}
		}
	}
	return res, nil
}

func (a OrderedArray[T]) IsZero() bool {
	return len(a.Values) == 0
}

func (a OrderedArray[T]) EncodeXMP(e *Encoder) error {
	outer := e.makeName(RDFNamespace, "Seq")
	err := e.EncodeToken(xml.StartElement{Name: outer})
	if err != nil {
		return err
	}

	inner := e.makeName(RDFNamespace, "li")
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

var (
	errMalformedXMP = errors.New("malformed XMP data")
)
