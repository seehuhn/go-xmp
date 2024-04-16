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

package dc

import (
	"encoding/xml"

	"seehuhn.de/go/xmp"
)

const (
	// NameSpace is the namespace URI of the Dublin Core namespace.
	NameSpace = "http://purl.org/dc/elements/1.1/"
)

// DublinCore represents the properties in the Dublin Core namespace.
type DublinCore struct {
	// Contributor is a list of contributors to the resource.
	// This should not include names listed in the Creator field.
	Contributor xmp.UnorderedArray[xmp.ProperName]

	// Coverage is the extent or scope of the resource.
	Coverage xmp.Text

	// Creator is a list of the creators of the resource.  Entities should be
	// listed in order of decreasing precedence, if such order is significant.
	Creator xmp.OrderedArray[xmp.ProperName]

	// Date is a point or period of time associated with an event in the life
	// cycle of the resource.
	Date xmp.OrderedArray[xmp.Date]

	// Description string
	// Format      string
	// Identifier  string
	// Language    string
	// Publisher   string
	// Relation    string
	// Rights      string

	// Source is a reference to a resource from which the present resource is
	// derived, either in whole or in part.
	Source xmp.Text

	// Subject is a list of descriptive phrases or keywords that specify the
	// content of the resource.
	Subject xmp.UnorderedArray[xmp.Text]

	// Title       string
	// Type        string
}

// EncodeXMP implements the [xmp.Model] interface.
func (dc *DublinCore) EncodeXMP(e *xmp.Encoder, pfx string) error {
	err := e.EncodeProperty(NameSpace, "contributor", dc.Contributor)
	if err != nil {
		return err
	}

	err = e.EncodeProperty(NameSpace, "coverage", dc.Coverage)
	if err != nil {
		return err
	}

	err = e.EncodeProperty(NameSpace, "creator", dc.Creator)
	if err != nil {
		return err
	}

	err = e.EncodeProperty(NameSpace, "date", dc.Date)
	if err != nil {
		return err
	}

	// TODO(voss):
	// - Description
	// - Format
	// - Identifier
	// - Language
	// - Publisher
	// - Relation
	// - Rights

	err = e.EncodeProperty(NameSpace, "source", dc.Source)
	if err != nil {
		return err
	}

	err = e.EncodeProperty(NameSpace, "subject", dc.Subject)
	if err != nil {
		return err
	}

	// TODO(voss):
	// - Title
	// - Type

	return nil
}

// NameSpaces implements the [xmp.Model] interface.
func (dc *DublinCore) NameSpaces(m map[string]struct{}) {
	m[NameSpace] = struct{}{}

	// TODO(voss): add namespaces of all properties
	m[xmp.RDFNamespace] = struct{}{}
}

func updateDublinCore(m xmp.Model, name string, tokens []xml.Token) (xmp.Model, error) {
	var dc *DublinCore
	if m, ok := m.(*DublinCore); ok {
		dc = m
	} else {
		dc = &DublinCore{}
	}

	switch name {
	case "contributor":
		v, err := xmp.DecodeUnorderedArray(tokens, xmp.DecodeProperName)
		if err != nil {
			return nil, err
		}
		dc.Contributor = v
	case "coverage":
		v, err := xmp.DecodeText(tokens)
		if err != nil {
			return nil, err
		}
		dc.Coverage = v
	case "creator":
		v, err := xmp.DecodeOrderedArray(tokens, xmp.DecodeProperName)
		if err != nil {
			return nil, err
		}
		dc.Creator = v
	case "date":
		v, err := xmp.DecodeOrderedArray(tokens, xmp.DecodeDate)
		if err != nil {
			return nil, err
		}
		dc.Date = v

	case "source":
		v, err := xmp.DecodeText(tokens)
		if err != nil {
			return nil, err
		}
		dc.Source = v
	case "subject":
		v, err := xmp.DecodeUnorderedArray(tokens, xmp.DecodeText)
		if err != nil {
			return nil, err
		}
		dc.Subject = v

	}

	return dc, nil
}

func init() {
	xmp.RegisterModel(NameSpace, "dc", updateDublinCore)
}
