// seehuhn.de/go/xmp - Extensible Metadata Platform in Go
// Copyright (C) 2026  Jochen Voss <voss@seehuhn.de>
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
	"bytes"
	"errors"
)

// xmpPacketID is the fixed identifier required by ISO 16684-1
// (Annex H, "XMP Packets") in the begin processing instruction of
// every conformant XMP packet wrapper:
//
//	<?xpacket begin="..." id="W5M0MpCehiHzreSzNTczkc9d"?>
//
// The string is chosen to be vanishingly unlikely to appear by
// accident in non-XMP bytes; XMP-aware tools (ExifTool, the Adobe XMP
// Toolkit) anchor their scans on it for that reason.
const xmpPacketID = "W5M0MpCehiHzreSzNTczkc9d"

// maxScanParseFailures bounds the worst-case work [Scan] performs on
// adversarial input.  Each parse failure forces [Scan] to retry past
// the matched id attribute, which is cheap on real-world data
// (typically zero failures) but unbounded on a hand-crafted blob full
// of corrupt wrappers that engulf one another.  After this many
// parse failures, [Scan] gives up.  Successful parses do not count.
const maxScanParseFailures = 16

// Scan locates an XMP packet wrapper inside arbitrary host bytes and
// returns the parsed Packet.  The XMP packet wrapper is designed for
// exactly this kind of byte-level extraction, allowing metadata to be
// recovered without parsing the host file format (PDF, JPEG, TIFF, …).
//
// Wrappers are identified by the magic id attribute
// id="W5M0MpCehiHzreSzNTczkc9d" (single quotes are also accepted)
// inside an enclosing <?xpacket begin="..." id="..."?> processing
// instruction.  Wrappers encoded in UTF-8, UTF-16BE, or UTF-16LE are
// found.  UTF-32 wrappers are not detected.
//
// When the data contains multiple packets (e.g. a PDF with both
// document-level metadata and per-image XMP), the document-level
// packet — identified by an empty rdf:about attribute — is preferred.
// Otherwise the first packet that parses successfully is returned.
// Packets that parse to no properties are skipped: an empty wrapper
// conveys no metadata and would otherwise spuriously match the
// document-level case.
//
// Scan returns:
//   - (packet, nil) when at least one wrapper parsed successfully;
//   - (nil, nil) when no wrapper was found in the input at all;
//   - (nil, err) when one or more wrappers were located but every
//     attempt to parse them failed.  In that case err is the
//     [errors.Join] of all per-wrapper parse errors, each wrapping
//     [ErrMalformed].
//
// Parse failures encountered while another wrapper later parses
// successfully are silently dropped: the caller has the data they
// wanted.
func Scan(data []byte) (*Packet, error) {
	var fallback *Packet
	var errs []error
	pos := 0
	failures := 0
	for pos < len(data) {
		bestStart, bestStop, bestIDAt := -1, -1, -1
		var bestPattern scanPattern
		for _, p := range scanPatterns {
			start, stop, idAt, ok := findWrapper(data, pos, p)
			if !ok {
				continue
			}
			if bestStart < 0 || start < bestStart {
				bestStart, bestStop, bestIDAt = start, stop, idAt
				bestPattern = p
			}
		}
		if bestStart < 0 {
			break
		}

		packet, err := Read(bytes.NewReader(data[bestStart:bestStop]))
		if err == nil && packet != nil && len(packet.Properties) > 0 {
			if packet.About == nil {
				return packet, nil
			}
			if fallback == nil {
				fallback = packet
			}
		}

		if err != nil {
			errs = append(errs, err)
			failures++
			if failures > maxScanParseFailures {
				break
			}
			// The candidate range may have been a corrupt outer
			// wrapper that engulfed a real inner wrapper (the same
			// id attribute can appear inside the outer's malformed
			// content).  Resume just past the matched id attribute
			// so any inner id match becomes the next candidate.
			pos = bestIDAt + len(bestPattern.id)
		} else {
			pos = bestStop
		}
	}

	if fallback != nil {
		return fallback, nil
	}
	return nil, errors.Join(errs...)
}

// findWrapper locates the next XMP wrapper at or after pos, encoded
// according to p.  It anchors on the id="W5M0..." attribute, then
// verifies that the attribute falls inside an enclosing <?xpacket ...?>
// processing instruction and that a matching <?xpacket end=...?>
// follows.  Returns the byte range that contains the full wrapper and
// the position of the id attribute.
//
// If the first id-attribute occurrence is outside any <?xpacket ?> PI
// (theoretically possible when an unrelated tool emits a literal
// id="W5M0..." in CharData), the search advances past it and tries
// the next occurrence — otherwise a stray match before a real wrapper
// would mask the wrapper.
func findWrapper(data []byte, pos int, p scanPattern) (start, stop, idAt int, ok bool) {
	for pos < len(data) {
		idIdx := bytes.Index(data[pos:], p.id)
		if idIdx < 0 {
			return 0, 0, 0, false
		}
		idAt = pos + idIdx

		// The id attribute must lie inside <?xpacket ... ?>.  Walk
		// back to the nearest "<?xpacket" before idAt; if a closing
		// "?>" separates them, the attribute is outside the PI and
		// this candidate is rejected.  Skip past it and try the next
		// match.
		wrapperStart := bytes.LastIndex(data[:idAt], p.xpacket)
		if wrapperStart < 0 || bytes.Contains(data[wrapperStart:idAt], p.close) {
			pos = idAt + len(p.id)
			continue
		}

		beginCloseRel := bytes.Index(data[idAt:], p.close)
		if beginCloseRel < 0 {
			return 0, 0, 0, false
		}
		beginPIEnd := idAt + beginCloseRel + len(p.close)

		endRel := bytes.Index(data[beginPIEnd:], p.endPI)
		if endRel < 0 {
			return 0, 0, 0, false
		}
		endStart := beginPIEnd + endRel

		endCloseRel := bytes.Index(data[endStart:], p.close)
		if endCloseRel < 0 {
			return 0, 0, 0, false
		}
		return wrapperStart, endStart + endCloseRel + len(p.close), idAt, true
	}
	return 0, 0, 0, false
}

// scanPattern holds the byte-level form of the wrapper sentinels in
// one Unicode encoding for one quote style.
type scanPattern struct {
	id      []byte // id="W5M0..." or id='W5M0...'
	xpacket []byte // "<?xpacket"
	endPI   []byte // "<?xpacket end="
	close   []byte // "?>"
}

// scanPatterns lists wrapper sentinel bytes for each supported
// encoding and quote style.  The XMP spec writes the id attribute
// with double quotes; single quotes are also valid XML and accepted
// here as well.  UTF-16 variants are produced by zero-padding each
// ASCII byte on either side.
var scanPatterns = func() []scanPattern {
	idDouble := `id="` + xmpPacketID + `"`
	idSingle := `id='` + xmpPacketID + `'`
	const (
		xpacket = "<?xpacket"
		endPI   = "<?xpacket end="
		close   = "?>"
	)
	return []scanPattern{
		{id: []byte(idDouble), xpacket: []byte(xpacket), endPI: []byte(endPI), close: []byte(close)},
		{id: []byte(idSingle), xpacket: []byte(xpacket), endPI: []byte(endPI), close: []byte(close)},
		{id: asciiToUTF16BE(idDouble), xpacket: asciiToUTF16BE(xpacket), endPI: asciiToUTF16BE(endPI), close: asciiToUTF16BE(close)},
		{id: asciiToUTF16BE(idSingle), xpacket: asciiToUTF16BE(xpacket), endPI: asciiToUTF16BE(endPI), close: asciiToUTF16BE(close)},
		{id: asciiToUTF16LE(idDouble), xpacket: asciiToUTF16LE(xpacket), endPI: asciiToUTF16LE(endPI), close: asciiToUTF16LE(close)},
		{id: asciiToUTF16LE(idSingle), xpacket: asciiToUTF16LE(xpacket), endPI: asciiToUTF16LE(endPI), close: asciiToUTF16LE(close)},
	}
}()

func asciiToUTF16BE(s string) []byte {
	out := make([]byte, 2*len(s))
	for i := range s {
		out[2*i+1] = s[i]
	}
	return out
}

func asciiToUTF16LE(s string) []byte {
	out := make([]byte, 2*len(s))
	for i := range s {
		out[2*i] = s[i]
	}
	return out
}
