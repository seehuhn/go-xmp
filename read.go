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
	propertyNS := ""
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
			if level > 0 || t.Name.Space == RDFNamespace && t.Name.Local == "RDF" {
				level++
			} else {
				continue tokenLoop
			}
			if descriptionLevel < 0 && t.Name.Space == RDFNamespace && t.Name.Local == "Description" {
				var about string
				for _, a := range t.Attr {
					if a.Name.Space == RDFNamespace && a.Name.Local == "about" {
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
					return nil, fmt.Errorf("inconsistent `about` attributes: %s != %s", p.About, aboutURL)
				}
				descriptionLevel = level
			} else if descriptionLevel >= 0 && propertyLevel < 0 {
				// start recording the XML tokens which make up the property
				propertyNS = t.Name.Space
				propertyLevel = level
				propertyTokens = nil
			}
		case xml.EndElement:
			if level == propertyLevel {
				// propertyTokens contains the XML tokens which make up the property,
				// including the start element (but not the end element).

				start := propertyTokens[0].(xml.StartElement)
				for _, a := range start.Attr {
					if a.Name == nameXMLLang {
						lang := a.Value
						fmt.Println(start, lang)
					}
				}

				propertyName := start.Name.Local
				info, ok := modelReaders[propertyNS]
				update := updateGeneric
				if ok {
					update = info.update
				}
				model, err := update(p.Models[propertyNS], propertyName, propertyTokens[1:])
				if err != nil {
					return nil, err
				}
				p.Models[propertyNS] = model
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
			propertyTokens = append(propertyTokens, xml.CopyToken(t))
		}
	}
	return p, nil
}

var (
	nameXMLLang        = xml.Name{Space: xmlNamespace, Local: "lang"}
	nameRDFDescription = xml.Name{Space: RDFNamespace, Local: "Description"}
)
