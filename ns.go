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
	"strconv"
	"strings"

	"seehuhn.de/go/xmp/jvxml"
)

// getPrefix chooses a new prefix for the given namespace.
// The new prefix is chosen to be different from the ones already in the
// nsToPrefix map.
func getPrefix(nsToPrefix map[string]string, ns string) string {
	// The following code is a modified version of code from
	// encoding/xml/marshal.go in the Go standard library.

	// Pick a name. We try to use the final element of the path
	// but fall back to _.
	prefix := strings.TrimRight(ns, "/#")
	if i := strings.LastIndex(prefix, "/"); i >= 0 {
		prefix = prefix[i+1:]
	}
	if prefix == "" || !jvxml.IsName([]byte(prefix)) || strings.Contains(prefix, ":") {
		prefix = "_"
	}
	// xmlanything is reserved and any variant of it regardless of
	// case should be matched, so:
	//    (('X'|'x') ('M'|'m') ('L'|'l'))
	// See Section 2.3 of https://www.w3.org/TR/REC-xml/
	if len(prefix) >= 3 && strings.EqualFold(prefix[:3], "xml") {
		prefix = "_" + prefix
	}

	if nsToPrefix[prefix] != "" {
		// Name is taken.  Find a better one.
		idx := len(nsToPrefix) + 1
		for {
			if id := prefix + "_" + strconv.Itoa(idx); nsToPrefix[id] == "" {
				prefix = id
				break
			}
			idx--
		}
	}
	// End of code from encoding/xml/marshal.go

	return prefix
}

var defaultPrefix = map[string]string{
	xmlNamespace:                                     "xml",
	RDFNamespace:                                     "rdf",
	"http://ns.adobe.com/xap/1.0/":                   "xmp",
	"http://ns.adobe.com/xap/1.0/mm/":                "xmpMM",     // XMP Media Management
	"http://ns.adobe.com/xap/1.0/rights/":            "xmpRights", // XMP Rights Management
	"http://ns.adobe.com/xap/1.0/sType/ResourceRef#": "stRef",     // ResourceRef
	"http://ns.adobe.com/xmp/Identifier/qual/1.0/":   "xmpidq",
	"http://purl.org/dc/elements/1.1/":               "dc", // Dublin Core
}

const (
	// xmlNamespace is the namespace for XML.
	xmlNamespace = "http://www.w3.org/XML/1998/namespace"

	// RDFNamespace is the namespace for RDF.
	RDFNamespace = "http://www.w3.org/1999/02/22-rdf-syntax-ns#"
)
