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

// DublinCore represents the properties in the Dublin Core namespace.
type DublinCore struct {
	Contributor UnorderedArray[ProperName]
	Coverage    Text
	// Creator     string
	// Date        string
	// Description string
	// Format      string
	// Identifier  string
	// Language    string
	// Publisher   string
	// Relation    string
	// Rights      string
	// Source      string
	// Subject     string
	// Title       string
	// Type        string
}

func (dc *DublinCore) EncodeProperties(e *Encoder, pfx string) error {
	if !dc.Contributor.IsZero() {
		err := e.EncodeToken(xml.StartElement{Name: e.MakeName(dublinCoreNS, "contributor")})
		if err != nil {
			return err
		}
		err = dc.Contributor.EncodeValue(e)
		if err != nil {
			return err
		}
		err = e.EncodeToken(xml.EndElement{Name: e.MakeName(dublinCoreNS, "contributor")})
		if err != nil {
			return err
		}
	}

	if !dc.Coverage.IsZero() {
		err := e.EncodeToken(xml.StartElement{Name: e.MakeName(dublinCoreNS, "coverage")})
		if err != nil {
			return err
		}
		err = dc.Coverage.EncodeValue(e)
		if err != nil {
			return err
		}
		err = e.EncodeToken(xml.EndElement{Name: e.MakeName(dublinCoreNS, "coverage")})
		if err != nil {
			return err
		}
	}

	return nil
}

func (dc *DublinCore) NameSpaces() []string {
	return []string{dublinCoreNS, rdfNS}
}

func (dc *DublinCore) DefaultPrefix() string {
	return "dc"
}

const (
	dublinCoreNS = "http://purl.org/dc/elements/1.1/"
)
