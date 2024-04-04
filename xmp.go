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

// Packet represents an XMP packet.
type Packet struct {
	Properties map[xml.Name]Property
	About      *url.URL
}

// Property is an XMP metadata property.
type Property struct {
	Name  string
	Value string
}

const (
	rdfNS = "http://www.w3.org/1999/02/22-rdf-syntax-ns#"
)

func Read(r io.Reader) (*Packet, error) {
	dec := xml.NewDecoder(r)
	p := &Packet{
		Properties: make(map[xml.Name]Property),
	}

	var level int
	descriptionLevel := -1
	propertyLevel := -1
	propertyNS := ""
	propertyLocal := ""
	var propertyTokens []xml.Token
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
			if level > 0 || t.Name.Space == rdfNS && t.Name.Local == "RDF" {
				level++
			} else {
				continue tokenLoop
			}
			if descriptionLevel < 0 && t.Name.Space == rdfNS && t.Name.Local == "Description" {
				var about string
				for _, a := range t.Attr {
					if a.Name.Space == rdfNS && a.Name.Local == "about" {
						about = a.Value
						break
					}
				}
				aboutURL, err := url.Parse(about)
				if err != nil {
					return nil, err
				}
				if p.About == nil {
					p.About = aboutURL
				} else if *p.About != *aboutURL {
					return nil, fmt.Errorf("inconsistent about attributes: %s != %s", p.About, aboutURL)
				}
				descriptionLevel = level
			} else if descriptionLevel >= 0 && propertyLevel < 0 {
				propertyLevel = level
				propertyNS = t.Name.Space
				propertyLocal = t.Name.Local
				propertyTokens = nil
			}
		case xml.EndElement:
			if level == propertyLevel {
				propertyTokens = append(propertyTokens, t)
				fmt.Println(propertyNS)
				fmt.Println(propertyLocal)
				for _, t := range propertyTokens {
					fmt.Println(".", t)
				}
				fmt.Println()
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
			propertyTokens = append(propertyTokens, t)
		}
	}
	return p, nil
}
