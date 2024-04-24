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
	"fmt"
	"reflect"
)

// DublinCore represents the properties in the Dublin Core namespace.
//
// See section 8.4 of ISO 16684-1:2011.
type DublinCore struct {
	_ Namespace `xmp:"http://purl.org/dc/elements/1.1/"`
	_ Prefix    `xmp:"dc"`

	// Contributor is a list of contributors to the resource.
	// This should not include names listed in the Creator field.
	Contributor UnorderedArray[ProperName] `xmp:"contributor"`

	// Coverage is the extent or scope of the resource.
	Coverage Text `xmp:"coverage"`

	// Creator is a list of the creators of the resource.  Entities should be
	// listed in order of decreasing precedence, if such order is significant.
	Creator OrderedArray[ProperName] `xmp:"creator"`

	// Date is a point or period of time associated with an event in the life
	// cycle of the resource.
	Date OrderedArray[Date] `xmp:"date"`

	// Description is a textual description of the content of the resource.
	Description Localized `xmp:"description"`

	// Format is the media type of the resource.
	Format MimeType `xmp:"format"`

	// Identifier is an unambiguous reference for the resource.
	Identifier Text `xmp:"identifier"`

	// Language is a list of languages used in the content of the resource.
	Language UnorderedArray[Locale] `xmp:"language"`

	// Publisher is a list of publishers of the resource.
	Publisher UnorderedArray[ProperName] `xmp:"publisher"`

	// Relation is a list of related resources.
	Relation UnorderedArray[Text] `xmp:"relation"`

	// Rights is an informal rights statement for the resource.
	Rights Localized `xmp:"rights"`

	// Source is a reference to a resource from which the present resource is
	// derived, either in whole or in part.
	Source Text `xmp:"source"`

	// Subject is a list of descriptive phrases or keywords that specify the
	// content of the resource.
	Subject UnorderedArray[Text] `xmp:"subject"`

	// Title is the title or name of the resource.
	Title Localized `xmp:"title"`

	// Type is the nature or genre of the resource.
	Type UnorderedArray[Text] `xmp:"type"`
}

// XMP represents the XMP basic namespace.
//
// See section 8.4 of ISO 16684-1:2011.
type XMP struct {
	_ Namespace `xmp:"http://ns.adobe.com/xap/1.0/"`
	_ Prefix    `xmp:"xmp"`

	// CreateDate is the date and time the resource was originally created.
	CreateDate Date

	// CreatorTool is the name of the first known tool used to create the
	// resource.
	CreatorTool AgentName

	// Identifier is an unambiguous reference to the resource within a given
	// context.  An array item may be qualified with xmpidq:Scheme to specify
	// the identification system for that item.
	Identifier UnorderedArray[Text]

	// Label is a short informal name for the resource.
	Label Text

	// MetadataDate is the date and time that any metadata for this resource was
	// last modified.
	MetadataDate Date

	// ModifyDate is the date and time the resource was last modified.
	ModifyDate Date

	// Rating is a user-assigned rating for this resource.
	//
	// The value must be -1 (rejected), 0 (unrated) or a rating in the range
	// (0, 5].
	Rating Real
}

// Set sets all the (non-zero) fields from a namespace struct.
func (p *Packet) Set(v any) error {
	s := reflect.Indirect(reflect.ValueOf(v))
	if s.Kind() != reflect.Struct {
		return errors.New("no struct found")
	}
	st := s.Type()

	var namespace, prefix string
	for i := 0; i < st.NumField(); i++ {
		fVal := s.Field(i)
		fInfo := st.Field(i)

		if fVal.Type() == nsTagType {
			namespace = fInfo.Tag.Get("xmp")
		} else if fVal.Type() == prefixTagType {
			prefix = fInfo.Tag.Get("xmp")
		}
	}
	if namespace == "" {
		return errors.New("XMP namespace not specified")
	}

	p.RegisterPrefix(namespace, prefix)

	for i := 0; i < st.NumField(); i++ {
		fVal := s.Field(i)
		fInfo := st.Field(i)

		if fVal.Type() == nsTagType || fVal.Type() == prefixTagType {
			continue
		}

		var val Value
		if fVal.CanInterface() && fVal.Type().Implements(typeType) {
			val = fVal.Interface().(Value)
		}

		propertyName := fInfo.Tag.Get("xmp")
		if propertyName == "" {
			propertyName = fInfo.Name
		} else if val == nil {
			return fmt.Errorf("field %s does not implement Type", fInfo.Name)
		}
		if !val.IsZero() {
			p.SetValue(namespace, propertyName, val)
		}
	}

	return nil
}

// Fill fills the fields in a namespace struct using data from the packet.
//
// The argument dst must be a pointer to an XMP namespace struct or the
// function will panic.
func (p *Packet) Fill(dst any) {
	s := reflect.Indirect(reflect.ValueOf(dst))
	st := s.Type()

	var namespace string
	for i := 0; i < st.NumField(); i++ {
		fVal := s.Field(i)
		fInfo := st.Field(i)

		if fVal.Type() == nsTagType {
			namespace = fInfo.Tag.Get("xmp")
		}
	}
	if namespace == "" {
		panic("not an XMP namespace struct")
	}

	for i := 0; i < st.NumField(); i++ {
		fVal := s.Field(i)
		fInfo := st.Field(i)

		if !fVal.CanInterface() || !fVal.Type().Implements(typeType) {
			continue
		}

		propertyName := fInfo.Tag.Get("xmp")
		if propertyName == "" {
			propertyName = fInfo.Name
		}

		name := xml.Name{Space: namespace, Local: propertyName}
		xmpData, ok := p.Properties[name]
		if !ok {
			continue
		}

		val := fVal.Interface().(Value)
		u, err := val.DecodeAnother(xmpData)
		if err != nil {
			continue
		}
		fVal.Set(reflect.ValueOf(u))
	}
}

var (
	nsTagType     = reflect.TypeFor[Namespace]()
	prefixTagType = reflect.TypeFor[Prefix]()
	typeType      = reflect.TypeFor[Value]()
)

// Namespace must be used in XMP namespace structs to specify the namespace
// URI.  The namespace URI is specified using a struct tag on a field of type
// Namespace.  For example:
//
//	type MyNamespace struct {
//	    _ Namespace `xmp:"http://example.com/ns/my/namespace/"`
//	    ...
//	}
type Namespace struct{}

// Prefix can be used in XMP namespace structs to optionally specify the
// preferred XML prefix for the namespace.  The prefix is specified using a
// struct tag on a field of type Prefix.  For example:
//
//	type MyNamespace struct {
//	    _ Namespace `xmp:"http://example.com/ns/my/namespace/"`
//	    _ Prefix    `xmp:"myns"`
//	    ...
//	}
//
// If no prefix is specified (or if there is a prefix name clash), a prefix is
// automatically chosen.
type Prefix struct{}
