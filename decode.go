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
	"fmt"
	"io"
	"net/url"
	"os"

	"golang.org/x/text/language"
)

// ReadFile reads an XMP packet from a file.
func ReadFile(filename string) (*Packet, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Read(f)
}

// Read reads an XMP packet from a reader.
func Read(r io.Reader) (*Packet, error) {
	dec := xml.NewDecoder(r)
	p := &Packet{
		Models: make(map[string]Model),
	}

	var level int
	descriptionLevel := -1
	propertyLevel := -1
	var propertyElement []xml.Token
tokenLoop:
	for {
		t, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		switch t := t.(type) {
		case xml.StartElement:
			if level > 0 || t.Name == elemRDFRoot {
				// TODO(voss): currently, if a sequence of rdf:RDF elements is
				// encountered, the contents are merged into a single packet.
				// Should we return an error instead?
				level++
			} else {
				continue tokenLoop
			}
			if descriptionLevel < 0 && t.Name == elemRDFDescription {
				for _, a := range t.Attr {
					if a.Name.Space == "xmlns" {
						continue
					}
					switch a.Name {
					case attrRDFAbout:
						aboutURL, _ := url.Parse(a.Value)
						if p.About == nil {
							p.About = aboutURL
						} else if aboutURL != nil && *aboutURL != *p.About {
							return nil, fmt.Errorf("inconsistent `about` attributes: %s != %s", p.About, aboutURL)
						}
					case attrXMLLang, attrRDFID, attrRDFID, attrRDFNodeID, attrRDFDataType:
						// These are not allowed in XMP, and we simply ignore them.
					default:
						// Property [...] elements that have non-URI simple,
						// unqualified values may be replaced with attributes
						// in the rdf:Description element.
						start := xml.StartElement{Name: a.Name}
						tokens := []xml.Token{xml.CharData(a.Value)}
						p.parsePropertyElement(start, tokens)
					}
				}
				descriptionLevel = level
			} else if descriptionLevel >= 0 && propertyLevel < 0 {
				// start recording the XML tokens which make up a property element
				propertyLevel = level
				propertyElement = nil
			}
		case xml.EndElement:
			if level == propertyLevel {
				// propertyElement contains the XML tokens which make up the property,
				// including the start element, but not the end element.

				start := propertyElement[0].(xml.StartElement)
				p.parsePropertyElement(start, propertyElement[1:])
				propertyLevel = -1
			}
			if level == descriptionLevel {
				descriptionLevel = -1
			}
			if level > 0 {
				level--
			}
		}

		if propertyLevel >= 0 {
			propertyElement = append(propertyElement, xml.CopyToken(t))
		}
	}
	return p, nil
}

// ParsePropertyElement parses a property element and updates the packet. The
// argument `start` is the start element of the property element, and `tokens`
// contains the XML tokens which make up the property element (not including
// the start and end elements).
//
// This implements the rules from appendix C.2.5 (Content of a nodeElement)
// of ISO 16684-1:2011.
func (p *Packet) parsePropertyElement(start xml.StartElement, tokens []xml.Token) error {
	tp := getProperyElementType(start, tokens)
	switch tp {
	case literalPropertyElt:
		// A literalPropertyElt is the typical element form of a simple
		// property.  The text content is the property value.  Attributes of
		// the element become qualifiers in the XMP data model.
		//
		// See appendix C.2.7 (The literalPropertyElt) of ISO 16684-1:2011.
		var qq Q
		for _, a := range start.Attr {
			switch a.Name {
			case attrXMLLang:
				langAttr, _ := language.Parse(a.Value)
				if langAttr != language.Und {
					val := Locale{Language: langAttr}
					q := Qualifier{
						Name:  attrXMLLang,
						Value: val,
					}
					qq = append(qq, q)
				}
			case attrXMLLang, attrRDFID, attrRDFID, attrRDFNodeID, attrRDFDataType:
				// These are not allowed in XMP, and we simply ignore them.
			default:
				tokens := []xml.Token{xml.CharData(a.Value)}
				dec := getQualifierDecoder(a.Name)
				val, err := dec(tokens, nil)
				if err != nil {
					// we ignore malformed qualifiers
				} else {
					q := Qualifier{
						Name:  a.Name,
						Value: val,
					}
					qq = append(qq, q)
				}
			}
		}

		propertyNS := start.Name.Space
		update := getModelUpdater(propertyNS)
		propertyName := start.Name.Local
		model, err := update(p.Models[propertyNS], propertyName, tokens, qq)
		if err != nil {
			// TODO(voss): ignore malformed properties?
			return err
		}
		p.Models[propertyNS] = model

		return nil // TODO(voss): remove once all cases are implemented

	case resourcePropertyElt:
		// A resourcePropertyElt most commonly represents an XMP struct or
		// array property. It can also represent a property with general
		// qualifiers (other than xml:lang as an attribute).
		//
		// See appendix C.2.6 (The resourcePropertyElt) of ISO 16684-1:2011.

		// shortStruct := false
		// var structFieldTokens []xml.Token
		// for _, a := range start.Attr {
		// 	if a.Name == attrRDFParseType && a.Value == "Resource" {
		// 		// Short form of a struct property.
		// 		// See section 7.9.2.3.
		// 		shortStruct = true
		// 		break
		// 	}
		// }

	case parseTypeResourcePropertyElt:
		// A parseTypeResourcePropertyElt is a form of shorthand that replaces
		// the inner nodeElement of a resourcePropertyElt with an
		// rdf:parseType="Resource" attribute on the outer element. This form
		// is commonly used in XMP as a cleaner way to represent a struct.
		//
		// See appendix C.2.9 (The parseTypeResourcePropertyElt) of ISO 16684-1:2011.

	case emptyPropertyElt:
		// An emptyPropertyElt is an element with no contained content, just a
		// possibly empty set of attributes.  An emptyPropertyElt can represent
		// three special cases of simple XMP properties: a simple property with
		// an empty value; a simple property whose value is a URI; or an
		// alternative RDF form for a simple property with simple qualifiers.
		// An emptyPropertyElt can also represent an XMP struct whose fields
		// are all simple and unqualified.
		//
		// See appendix C.2.12 (The emptyPropertyElt) of ISO 16684-1:2011.

	case parseTypeLiteralPropertyElt, parseTypeCollectionPropertyElt, parseTypeOtherPropertyElt:
		// Not allowed in XMP.  We simply ignore these.
		return nil
	}

	propertyNS := start.Name.Space
	update := getModelUpdater(propertyNS)
	propertyName := start.Name.Local
	model, err := update(p.Models[propertyNS], propertyName, tokens, nil)
	if err != nil {
		return err
	}
	p.Models[propertyNS] = model

	return nil
}

// getProperyElementType determines the RDF type of a property element.
//
// This implements the rules from appendix C.2.5 (Content of a nodeElement)
// of ISO 16684-1:2011.
func getProperyElementType(start xml.StartElement, tokens []xml.Token) propertyElementType {
	if len(start.Attr) > 3 {
		return emptyPropertyElt
	}

	for _, a := range start.Attr {
		switch a.Name {
		case attrXMLLang:
			continue
		case attrRDFID: // not allowed in XMP
			continue
		case attrRDFDataType: // not allowed in XMP
			return literalPropertyElt
		case attrRDFParseType:
			return emptyPropertyElt
		default:
			switch a.Value {
			case "Literal": // not allowed in XMP
				return parseTypeLiteralPropertyElt
			case "Resource":
				return parseTypeResourcePropertyElt
			case "Collection": // not allowed in XMP
				return parseTypeCollectionPropertyElt
			default: // not allowed in XMP
				return parseTypeOtherPropertyElt
			}
		}
	}

	for _, t := range tokens {
		switch t.(type) {
		case xml.StartElement:
			return resourcePropertyElt
		case xml.CharData:
			return literalPropertyElt
		}
	}
	return emptyPropertyElt
}

type propertyElementType int

const (
	resourcePropertyElt propertyElementType = iota + 1
	literalPropertyElt
	parseTypeLiteralPropertyElt
	parseTypeResourcePropertyElt
	parseTypeCollectionPropertyElt
	parseTypeOtherPropertyElt
	emptyPropertyElt
)

var (
	elemRDFRoot        = xml.Name{Space: RDFNamespace, Local: "RDF"}
	elemRDFDescription = xml.Name{Space: RDFNamespace, Local: "Description"}

	attrRDFAbout     = xml.Name{Space: RDFNamespace, Local: "about"}
	attrRDFDataType  = xml.Name{Space: RDFNamespace, Local: "datatype"}
	attrRDFID        = xml.Name{Space: RDFNamespace, Local: "ID"}
	attrRDFNodeID    = xml.Name{Space: RDFNamespace, Local: "nodeID"}
	attrRDFParseType = xml.Name{Space: RDFNamespace, Local: "parseType"}
	attrXMLLang      = xml.Name{Space: xmlNamespace, Local: "lang"}
)
