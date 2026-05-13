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
	"regexp"
	"sort"
	"strings"
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

// maxScanMatches caps the number of wrapper-shaped candidates Scan
// considers per pass.  Each candidate triggers at most one [Read]
// call, itself bounded by maxPacketSize bytes, so total work stays
// O(maxScanMatches × maxPacketSize) regardless of input length.
const maxScanMatches = 16

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
// packet — identified by parsing to About == nil — is preferred.
// Otherwise the first wrapper that parses cleanly with at least one
// property is returned.  Wrappers are considered in input order;
// empty wrappers are silently skipped.
//
// Scan returns:
//   - (packet, nil) when at least one wrapper parsed successfully
//     and carried at least one property;
//   - (nil, nil) when no usable wrapper was found — the input
//     contained no wrapper at all, or every located wrapper was
//     empty (parsed cleanly but had no properties);
//   - (nil, err) when one or more wrappers were located but every
//     attempt to parse them failed.  In that case err is the
//     [errors.Join] of all per-wrapper parse errors, each wrapping
//     [ErrMalformed].  Parse failures encountered while another
//     wrapper later succeeds are dropped from the returned error.
//
// "No XMP" is a normal outcome for host files where XMP is
// optional, so absence is reported as (nil, nil) rather than as an
// error; callers that don't care about parse failures can discard
// the error and treat a nil packet as "no metadata".
func Scan(data []byte) (*Packet, error) {
	var fallback *Packet
	var errs []error
	for _, m := range collectMatches(data, wrapperRegexes) {
		if m.stop-m.start > maxPacketSize {
			continue
		}
		packet, err := Read(bytes.NewReader(data[m.start:m.stop]))
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if packet == nil || len(packet.Properties) == 0 {
			continue
		}
		if packet.About == nil {
			return packet, nil
		}
		if fallback == nil {
			fallback = packet
		}
	}
	if fallback != nil {
		return fallback, nil
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return nil, nil
}

// match is a [start, stop) byte range located by one of the wrapper
// regexes.
type match struct{ start, stop int }

// collectMatches gathers wrapper candidates returned by all per-
// encoding regexes, sorts them by start position, removes duplicates,
// and truncates to maxScanMatches.  Inner candidates engulfed by a
// failing outer wrapper are reachable because each per-encoding scan
// advances by a single byte after every hit, allowing the next hit
// to start anywhere strictly inside the previous one.
func collectMatches(data []byte, regexes []*regexp.Regexp) []match {
	var matches []match
	for _, re := range regexes {
		offset := 0
		for range maxScanMatches {
			if offset >= len(data) {
				break
			}
			loc := re.FindIndex(data[offset:])
			if loc == nil {
				break
			}
			m := match{loc[0] + offset, loc[1] + offset}
			matches = append(matches, m)
			offset = m.start + 1
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].start != matches[j].start {
			return matches[i].start < matches[j].start
		}
		return matches[i].stop < matches[j].stop
	})
	// dedupe adjacent (sorted) duplicates produced when more than one
	// encoding's regex matches the same byte range
	n := 0
	for i := range matches {
		if i == 0 || matches[i] != matches[n-1] {
			matches[n] = matches[i]
			n++
		}
	}
	matches = matches[:n]
	if len(matches) > maxScanMatches {
		matches = matches[:maxScanMatches]
	}
	return matches
}

// scanEncoding holds the regex fragments that differ between the
// byte-level encodings (UTF-8, UTF-16BE, UTF-16LE) Scan supports.
// The wrapper's surrounding-attribute pattern ([^?]) and payload
// pattern ([\s\S]) are encoding-independent: the begin PI is pure
// ASCII attribute syntax in every encoding, and the payload's
// suffix anchor carries explicit \x00 bytes in its UTF-16 forms,
// which is what re-locks the search onto a codeunit boundary after
// rune-by-rune stepping.
type scanEncoding struct {
	lit func(string) string // ASCII text → regex matching its encoded bytes
	ws  string              // regex matching one whitespace codeunit
}

var scanEncodings = []scanEncoding{
	{
		lit: regexp.QuoteMeta,
		ws:  `\s`,
	},
	{
		lit: func(s string) string { return utf16Lit(s, true) },
		ws:  `\x00[\t\n\r ]`,
	},
	{
		lit: func(s string) string { return utf16Lit(s, false) },
		ws:  `[\t\n\r ]\x00`,
	},
}

// utf16Lit returns a regex that matches the bytes of the ASCII string
// s encoded in UTF-16 (big-endian if bigEndian, otherwise little-
// endian).  Each ASCII byte is regex-escaped where required.
func utf16Lit(s string, bigEndian bool) string {
	var b strings.Builder
	b.Grow(len(s) * 4)
	for i := 0; i < len(s); i++ {
		if bigEndian {
			b.WriteString(`\x00`)
		}
		b.WriteString(regexp.QuoteMeta(string(s[i])))
		if !bigEndian {
			b.WriteString(`\x00`)
		}
	}
	return b.String()
}

// wrapperPattern returns a regex source matching an XMP packet
// wrapper in the given encoding.
//
// The bounded {0,200} repeats inside the begin PI cap how far the
// regex engine will search for an attribute boundary, so a
// pathological PI cannot extend the match arbitrarily.  The payload
// quantifier is non-greedy and unbounded; RE2 keeps total time linear
// in the input regardless.
func wrapperPattern(e scanEncoding) string {
	ws1 := `(?:` + e.ws + `)`
	wsStar := ws1 + `*`
	wsPlus := ws1 + `+`
	const (
		notQ = `[^?]{0,200}` // bounded non-'?' run inside the begin PI
		any  = `[\s\S]*?`    // lazy, unbounded payload between begin and end PI
	)
	id := `(?:` + e.lit(`"`+xmpPacketID+`"`) + `|` + e.lit(`'`+xmpPacketID+`'`) + `)`
	rwAttr := `(?:` + e.lit(`"r"`) + `|` + e.lit(`"w"`) + `|` + e.lit(`'r'`) + `|` + e.lit(`'w'`) + `)`

	var b strings.Builder
	b.Grow(800)

	// begin PI
	b.WriteString(e.lit("<?xpacket"))
	b.WriteString(ws1)
	b.WriteString(notQ)
	b.WriteString(e.lit("id"))
	b.WriteString(wsStar)
	b.WriteString(e.lit("="))
	b.WriteString(wsStar)
	b.WriteString(id)
	b.WriteString(notQ)
	b.WriteString(e.lit("?>"))

	// payload
	b.WriteString(any)

	// end PI
	b.WriteString(e.lit("<?xpacket"))
	b.WriteString(wsPlus)
	b.WriteString(e.lit("end"))
	b.WriteString(wsStar)
	b.WriteString(e.lit("="))
	b.WriteString(wsStar)
	b.WriteString(rwAttr)
	b.WriteString(wsStar)
	b.WriteString(e.lit("?>"))

	return b.String()
}

var wrapperRegexes = func() []*regexp.Regexp {
	out := make([]*regexp.Regexp, len(scanEncodings))
	for i, e := range scanEncodings {
		out[i] = regexp.MustCompile(wrapperPattern(e))
	}
	return out
}()
