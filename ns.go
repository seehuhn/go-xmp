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
// prefixToNS map.
func getPrefix(prefixToNS map[string]string, ns string) string {
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

	if prefixToNS[prefix] != "" {
		// Name is taken.  Find a better one.
		idx := len(prefixToNS) + 1
		for {
			if id := prefix + strconv.Itoa(idx); prefixToNS[id] == "" {
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
	NSXML: "xml",
	NSRDF: "rdf",
}

// Namespace URIs for the predefined XMP schemas, plus the auxiliary
// namespaces (XML, RDF, ResourceRef) used by the XMP serialization
// itself.  The schema URIs (NSDublinCore through NSPDFXID) are
// suitable for [Packet.SetValue], [PacketGetValue], and
// [Packet.ClearValue]; NSXML, NSRDF, and NSResourceRef are exposed
// for callers that need to recognise these namespaces in raw
// property data, not as schema arguments.
const (
	// NSXML is the namespace for XML built-in attributes such as xml:lang.
	NSXML = "http://www.w3.org/XML/1998/namespace"

	// NSRDF is the namespace for RDF.
	NSRDF = "http://www.w3.org/1999/02/22-rdf-syntax-ns#"

	// NSDublinCore is the namespace for the Dublin Core schema,
	// represented by the [DublinCore] struct.
	NSDublinCore = "http://purl.org/dc/elements/1.1/"

	// NSBasic is the namespace for the XMP basic schema,
	// represented by the [Basic] struct.
	NSBasic = "http://ns.adobe.com/xap/1.0/"

	// NSRightsManagement is the namespace for the XMP rights
	// management schema, represented by the [RightsManagement] struct.
	NSRightsManagement = "http://ns.adobe.com/xap/1.0/rights/"

	// NSMediaManagement is the namespace for the XMP media management
	// schema, represented by the [MediaManagement] struct.
	NSMediaManagement = "http://ns.adobe.com/xap/1.0/mm/"

	// NSPDF is the namespace for the Adobe PDF schema,
	// represented by the [PDF] struct.
	NSPDF = "http://ns.adobe.com/pdf/1.3/"

	// NSPDFAID is the namespace for the PDF/A identification schema,
	// represented by the [PDFAID] struct.
	NSPDFAID = "http://www.aiim.org/pdfa/ns/id/"

	// NSPDFX is the legacy Adobe PDF/X identification namespace used
	// by PDF/X-1a, PDF/X-2, and PDF/X-3.  It is represented by the
	// [PDFX] struct.
	NSPDFX = "http://ns.adobe.com/pdfx/1.3/"

	// NSPDFXID is the PDF/X identification namespace introduced in
	// PDF/X-4.  It is represented by the [PDFXID] struct.
	NSPDFXID = "http://www.npes.org/pdfx/ns/id/"

	// NSResourceRef is the namespace for the XMP ResourceRef
	// structured type, represented by the [ResourceRef] struct.
	NSResourceRef = "http://ns.adobe.com/xap/1.0/sType/ResourceRef#"
)
