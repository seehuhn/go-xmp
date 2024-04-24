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
)

// Read reads an XMP packet from a reader.
func Read(r io.Reader) (*Packet, error) {
	dec := xml.NewDecoder(r)
	p := &Packet{
		Properties: make(map[xml.Name]Raw),
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
		} else if err != nil {
			return nil, err
		}

		switch t := t.(type) {
		case xml.StartElement:
			if level > 0 || t.Name == nameRDFRoot {
				// TODO(voss): currently, if a sequence of rdf:RDF elements is
				// encountered, the contents are merged into a single packet.
				// Should we return an error instead?
				level++
			} else {
				continue tokenLoop
			}
			if descriptionLevel < 0 && t.Name == nameRDFDescription {
				for _, a := range t.Attr {
					if a.Name.Space == "xmlns" {
						continue
					}
					switch a.Name {
					case nameRDFAbout:
						var aboutURL *url.URL
						if a.Value != "" {
							aboutURL, _ = url.Parse(a.Value)
							if aboutURL != nil && aboutURL.String() == "" {
								// This is triggered when a.Value is "//#".
								aboutURL = nil
							}
						}
						if p.About == nil {
							p.About = aboutURL
						} else if aboutURL != nil && *aboutURL != *p.About {
							return nil, fmt.Errorf("inconsistent `about` attributes: %s != %s", p.About, aboutURL)
						}
					default:
						// Property [...] elements that have non-URI simple,
						// unqualified values may be replaced with attributes
						// in the rdf:Description element.
						if isValidPropertyName(a.Name) {
							p.Properties[a.Name] = RawText{Value: a.Value}
						}
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
				if isValidPropertyName(start.Name) {
					val := parsePropertyElement(start, propertyElement[1:], nil)
					if val != nil {
						p.Properties[start.Name] = val
					}
				}
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
//
// Invalid XML is ignored, and the function decodes as much of the property
// element as possible.  If no valid data is found, the function returns nil.
func parsePropertyElement(start xml.StartElement, tokens []xml.Token, qq Q) Raw {
	tp := getProperyElementType(start, tokens)
	switch tp {
	case literalPropertyElt:
		// A literalPropertyElt is the typical element form of a simple
		// property.  The text content is the property value.  Attributes of
		// the element become qualifiers in the XMP data model.
		//
		// See appendix C.2.7 (The literalPropertyElt) of ISO 16684-1:2011.
		for _, a := range start.Attr {
			if isValidQualifierName(a.Name) {
				qq = append(qq, Qualifier{Name: a.Name, Value: RawText{Value: a.Value}})
			}
		}

		var text string
		for _, t := range tokens {
			if c, ok := t.(xml.CharData); ok {
				text += string(c)
			}
		}
		return RawText{Value: text, Q: qq}

	case resourcePropertyElt:
		// A resourcePropertyElt most commonly represents an XMP struct or
		// array property. It can also represent a property with general
		// qualifiers (other than xml:lang as an attribute).
		//
		// A resourcePropertyElt can have an xml:lang attribute; it becomes an
		// xml:lang qualifier on the XMP value represented by the
		// resourcePropertyElt.
		//
		// See appendix C.2.6 (The resourcePropertyElt) of ISO 16684-1:2011.
		for _, a := range start.Attr {
			if a.Name == nameXMLLang {
				qq = append(qq, Qualifier{Name: a.Name, Value: RawText{Value: a.Value}})
			}
		}

		children := getChildElements(tokens)
		if len(children) == 0 {
			return nil
		}
		child := children[0] // valid XMP has exactly one child element
		switch {
		case child.name == nameRDFDescription: // a structure or general qualifiers
			descStart := tokens[child.start].(xml.StartElement)
			inner := tokens[child.start+1 : child.end]
			fields := getChildElements(inner)

			// General qualifiers are distinguished from structs by the presence
			// of an rdf:value field or attribute.
			attrIdx := -1
			valueIdx := -1
			for i, a := range descStart.Attr {
				if a.Name == nameRDFValue {
					attrIdx = i
					break
				}
			}
			for i, f := range fields {
				if f.name == nameRDFValue {
					valueIdx = i
					break
				}
			}
			if attrIdx >= 0 || valueIdx >= 0 {
				for _, a := range descStart.Attr {
					if isValidQualifierName(a.Name) { // this excludes elemRDFValue
						qq = append(qq, Qualifier{Name: a.Name, Value: RawText{Value: a.Value}})
					}
				}

				for _, f := range fields {
					if isValidQualifierName(f.name) { // this excludes elemRDFValue
						val := parsePropertyElement(inner[f.start].(xml.StartElement), inner[f.start+1:f.end], nil)
						if val != nil {
							qq = append(qq, Qualifier{Name: f.name, Value: val})
						}
					}
				}

				if attrIdx >= 0 {
					return RawText{Value: descStart.Attr[attrIdx].Value, Q: qq}
				}
				f := fields[valueIdx]
				val := parsePropertyElement(inner[f.start].(xml.StartElement), inner[f.start+1:f.end], qq)
				if val == nil {
					val = RawText{Value: ""}
				}
				return val
			}

			res := RawStruct{
				Value: make(map[xml.Name]Raw, len(fields)),
				Q:     qq,
			}
			for _, a := range descStart.Attr {
				if isValidPropertyName(a.Name) { // this excludes xml:lang
					res.Value[a.Name] = RawText{Value: a.Value}
				}
			}
			for _, f := range fields {
				if !isValidPropertyName(f.name) {
					continue
				}
				val := parsePropertyElement(inner[f.start].(xml.StartElement), inner[f.start+1:f.end], nil)
				if val != nil {
					res.Value[f.name] = val
				}
			}

			return res
		case child.name == nameRDFBag || child.name == nameRDFSeq || child.name == nameRDFAlt: // an array
			var tp RawArrayType
			switch child.name {
			case nameRDFBag:
				tp = Unordered
			case nameRDFSeq:
				tp = Ordered
			case nameRDFAlt:
				tp = Alternative
			}
			inner := tokens[child.start+1 : child.end]
			items := getChildElements(inner)
			res := RawArray{
				Value: make([]Raw, 0, len(items)),
				Kind:  tp,
				Q:     qq,
			}
			for _, i := range items {
				val := parsePropertyElement(inner[i.start].(xml.StartElement), inner[i.start+1:i.end], nil)
				if val != nil {
					res.Value = append(res.Value, val)
				}
			}
			return res
		default: // a typed node
			inner := tokens[child.start+1 : child.end]
			fields := getChildElements(inner)

			typeURLString := child.name.Space + child.name.Local
			typeURL, _ := url.Parse(typeURLString)
			if typeURL != nil {
				qq = append(qq, Qualifier{Name: nameRDFType, Value: RawURI{Value: typeURL}})
			}

			// General qualifiers are distinguished from structs by the presence
			// of an rdf:value field.
			valueIdx := -1
			for i, f := range fields {
				if f.name == nameRDFValue {
					valueIdx = i
					break
				}
			}
			if valueIdx >= 0 {
				for _, f := range fields {
					if isValidQualifierName(f.name) {
						val := parsePropertyElement(inner[f.start].(xml.StartElement), inner[f.start+1:f.end], nil)
						if val != nil {
							qq = append(qq, Qualifier{Name: f.name, Value: val})
						}
					}
				}

				f := fields[valueIdx]
				val := parsePropertyElement(inner[f.start].(xml.StartElement), inner[f.start+1:f.end], qq)
				if val == nil {
					val = RawText{Value: ""}
				}
				return val
			}

			res := RawStruct{
				Value: make(map[xml.Name]Raw, len(fields)),
				Q:     qq,
			}
			for _, f := range fields {
				if !isValidPropertyName(f.name) {
					continue
				}
				val := parsePropertyElement(inner[f.start].(xml.StartElement), inner[f.start+1:f.end], nil)
				if val != nil {
					res.Value[f.name] = val
				}
			}

			return res
		}

	case parseTypeResourcePropertyElt:
		// A parseTypeResourcePropertyElt is a form of shorthand that replaces
		// the inner nodeElement of a resourcePropertyElt with an
		// rdf:parseType="Resource" attribute on the outer element. This form
		// is commonly used in XMP as a cleaner way to represent a struct.
		//
		// See appendix C.2.9 (The parseTypeResourcePropertyElt) of ISO 16684-1:2011.

		for _, a := range start.Attr {
			if a.Name == nameXMLLang {
				qq = append(qq, Qualifier{Name: a.Name, Value: RawText{Value: a.Value}})
			}
		}

		fields := getChildElements(tokens)

		// General qualifiers are distinguished from structure elements by the
		// presence of an rdf:value field
		isQualifierStruct := false
		for _, f := range fields {
			if f.name == nameRDFValue {
				isQualifierStruct = true
				break
			}
		}
		if isQualifierStruct {
			var valueIndex int
			for i, f := range fields {
				if f.name == nameRDFValue {
					valueIndex = i
				} else if isValidQualifierName(f.name) {
					val := parsePropertyElement(tokens[f.start].(xml.StartElement), tokens[f.start+1:f.end], nil)
					if val != nil {
						qq = append(qq, Qualifier{Name: f.name, Value: val})
					}
				}
			}
			f := fields[valueIndex]
			return parsePropertyElement(tokens[f.start].(xml.StartElement), tokens[f.start+1:f.end], qq)
		}

		// this is a structure element
		res := RawStruct{
			Value: make(map[xml.Name]Raw),
			Q:     qq,
		}
		for _, f := range fields {
			if !isValidPropertyName(f.name) {
				continue
			}
			val := parsePropertyElement(tokens[f.start].(xml.StartElement), tokens[f.start+1:f.end], nil)
			if val != nil {
				res.Value[f.name] = val
			}
		}
		return res

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

		isSimpleProperty := false
		isURIProperty := false
		isEmptyValue := true
		for _, a := range start.Attr {
			switch a.Name {
			case nameRDFValue:
				isSimpleProperty = true
			case nameRDFResource:
				isURIProperty = true
			}
			if a.Name != nameXMLLang && a.Name != nameRDFID && a.Name != nameRDFNodeID {
				isEmptyValue = false
			}
		}
		switch { // the order is important here
		case isSimpleProperty:
			// If there is an rdf:value attribute, then this is a simple
			// property.  All other attributes are qualifiers.
			var value string
			var qq Q
			for _, a := range start.Attr {
				if a.Name == nameRDFValue {
					value = a.Value
				} else if isValidQualifierName(a.Name) {
					qq = append(qq, Qualifier{Name: a.Name, Value: RawText{Value: a.Value}})
				}
			}
			return RawText{Value: value, Q: qq}
		case isURIProperty:
			// If there is an rdf:resource attribute, then this is a simple
			// property with a URI value.  All other attributes are qualifiers.
			var uriString string
			for _, a := range start.Attr {
				if a.Name == nameRDFResource {
					uriString = a.Value
				} else if isValidQualifierName(a.Name) {
					qq = append(qq, Qualifier{Name: a.Name, Value: RawText{Value: a.Value}})
				}
			}
			uri, err := url.Parse(uriString)
			if err != nil {
				return nil
			}
			return RawURI{Value: uri, Q: qq}
		case isEmptyValue:
			// If there are no attributes other than xml:lang, rdf:ID, or
			// rdf:nodeID, then this is a simple property with an empty value.
			for _, a := range start.Attr {
				if a.Name == nameXMLLang {
					res := RawText{
						Q: Q{{Name: nameXMLLang, Value: RawText{Value: a.Value}}},
					}
					return res
				}
			}
			return RawText{}
		default:
			// Otherwise, this is a struct, and the attributes other than
			// xml:lang, rdf:ID, or rdf:nodeID are the fields.
			res := RawStruct{
				Value: make(map[xml.Name]Raw),
				Q:     qq,
			}
			for _, a := range start.Attr {
				if a.Name == nameXMLLang {
					res.Q = append(res.Q, Qualifier{Name: a.Name, Value: RawText{Value: a.Value}})
				} else if isValidPropertyName(a.Name) {
					res.Value[a.Name] = RawText{Value: a.Value}
				}
			}
			return res
		}

	case parseTypeLiteralPropertyElt, parseTypeCollectionPropertyElt, parseTypeOtherPropertyElt:
		return nil // not allowed in XMP

	default:
		panic("unreachable")
	}
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
		case nameXMLLang:
			continue
		case nameRDFID: // not allowed in XMP
			continue
		case nameRDFDataType: // not allowed in XMP
			return literalPropertyElt
		case nameRDFParseType:
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
		default:
			return emptyPropertyElt
		}
	}

	hasCharData := false
	for _, t := range tokens {
		switch t.(type) {
		case xml.StartElement:
			return resourcePropertyElt
		case xml.CharData:
			hasCharData = true
		}
	}
	if hasCharData {
		return literalPropertyElt
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

type childElement struct {
	name       xml.Name
	start, end int
}

func getChildElements(tokens []xml.Token) []childElement {
	var children []childElement
	level := 0
	for i, t := range tokens {
		switch t := t.(type) {
		case xml.StartElement:
			if level == 0 {
				children = append(children, childElement{name: t.Name, start: i})
			}
			level++
		case xml.EndElement:
			level--
			if level == 0 {
				children[len(children)-1].end = i
			}
		}
	}
	return children
}

func isValidPropertyName(n xml.Name) bool {
	if n.Space == "" || n.Local == "" || n.Space == xmlNamespace {
		return false
	}
	if n.Space == rdfNamespace && n != nameRDFType {
		return false
	}
	if _, err := url.Parse(n.Space); err != nil {
		return false
	}
	return true
}

func isValidQualifierName(n xml.Name) bool {
	if n.Space == "" || n.Local == "" {
		return false
	}
	if n.Space == rdfNamespace && n != nameRDFType {
		return false
	}
	if n.Space == xmlNamespace && n != nameXMLLang {
		return false
	}
	if _, err := url.Parse(n.Space); err != nil {
		return false
	}
	return true
}

var (
	nameRDFAbout       = xml.Name{Space: rdfNamespace, Local: "about"}
	nameRDFAlt         = xml.Name{Space: rdfNamespace, Local: "Alt"}
	nameRDFBag         = xml.Name{Space: rdfNamespace, Local: "Bag"}
	nameRDFDataType    = xml.Name{Space: rdfNamespace, Local: "datatype"}
	nameRDFDescription = xml.Name{Space: rdfNamespace, Local: "Description"}
	nameRDFID          = xml.Name{Space: rdfNamespace, Local: "ID"}
	nameRDFLi          = xml.Name{Space: rdfNamespace, Local: "li"}
	nameRDFNodeID      = xml.Name{Space: rdfNamespace, Local: "nodeID"}
	nameRDFParseType   = xml.Name{Space: rdfNamespace, Local: "parseType"}
	nameRDFResource    = xml.Name{Space: rdfNamespace, Local: "resource"}
	nameRDFRoot        = xml.Name{Space: rdfNamespace, Local: "RDF"}
	nameRDFSeq         = xml.Name{Space: rdfNamespace, Local: "Seq"}
	nameRDFType        = xml.Name{Space: rdfNamespace, Local: "type"}
	nameRDFValue       = xml.Name{Space: rdfNamespace, Local: "value"}
	nameXMLLang        = xml.Name{Space: xmlNamespace, Local: "lang"}

	attrParseTypeResource = xml.Attr{Name: nameRDFParseType, Value: "Resource"}
)
