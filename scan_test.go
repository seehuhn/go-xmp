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
	"net/url"
	"testing"
	"time"
)

const dcNS = "http://purl.org/dc/elements/1.1/"

// encodePacket serialises p with the standard wrapper for use in tests.
func encodePacket(t *testing.T, p *Packet) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := p.Write(&buf, nil); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return buf.Bytes()
}

func TestScan_SinglePacket(t *testing.T) {
	p := NewPacket()
	p.SetValue(dcNS, "title", Text{V: "Hello"})
	wrapped := encodePacket(t, p)

	got, err := Scan(wrapped)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if got == nil {
		t.Fatal("Scan returned nil packet")
	}
	v, err := PacketGetValue[Text](got, dcNS, "title")
	if err != nil {
		t.Fatalf("title lookup: %v", err)
	}
	if v.V != "Hello" {
		t.Errorf("got title %q, want Hello", v.V)
	}
}

func TestScan_PrefersDocumentLevel(t *testing.T) {
	// per-image packet: rdf:about points somewhere
	imgPacket := NewPacket()
	imgPacket.About = &url.URL{Scheme: "urn", Opaque: "img:1"}
	imgPacket.SetValue(dcNS, "title", Text{V: "Image"})

	// document-level packet: rdf:about=""
	docPacket := NewPacket()
	docPacket.SetValue(dcNS, "title", Text{V: "Document"})

	var buf bytes.Buffer
	buf.Write([]byte("garbage prefix\n"))
	buf.Write(encodePacket(t, imgPacket))
	buf.Write([]byte("\nbinary stream contents\n"))
	buf.Write(encodePacket(t, docPacket))
	buf.Write([]byte("\ntrailer junk"))

	got, err := Scan(buf.Bytes())
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if got == nil {
		t.Fatal("Scan returned nil packet")
	}
	v, _ := PacketGetValue[Text](got, dcNS, "title")
	if v.V != "Document" {
		t.Errorf("got title %q, want Document (the rdf:about=\"\" packet)", v.V)
	}
	if got.About != nil {
		t.Errorf("expected document-level packet (About=nil), got About=%v", got.About)
	}
}

func TestScan_FallbackToFirstWhenNoneAreDocumentLevel(t *testing.T) {
	first := NewPacket()
	first.About = &url.URL{Scheme: "urn", Opaque: "img:1"}
	first.SetValue(dcNS, "title", Text{V: "First"})

	second := NewPacket()
	second.About = &url.URL{Scheme: "urn", Opaque: "img:2"}
	second.SetValue(dcNS, "title", Text{V: "Second"})

	var buf bytes.Buffer
	buf.Write(encodePacket(t, first))
	buf.Write(encodePacket(t, second))

	got, err := Scan(buf.Bytes())
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if got == nil {
		t.Fatal("Scan returned nil")
	}
	v, _ := PacketGetValue[Text](got, dcNS, "title")
	if v.V != "First" {
		t.Errorf("got title %q, want First (fallback)", v.V)
	}
}

func TestScan_NoPacketReturnsNil(t *testing.T) {
	got, err := Scan([]byte("this file has no XMP whatsoever, just plain bytes"))
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got packet %+v", got)
	}
}

func TestScan_TruncatedWrapperIsIgnored(t *testing.T) {
	// begin sentinel without matching end
	got, err := Scan([]byte(`prefix <?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?> nothing more`))
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for truncated wrapper, got %+v", got)
	}
}

func TestScan_UTF16BE(t *testing.T) {
	p := NewPacket()
	p.SetValue(dcNS, "title", Text{V: "héllo"})
	utf8Bytes := encodePacket(t, p)
	utf16Bytes := transcodeUTF8ToUTF16(utf8Bytes, true)

	var buf bytes.Buffer
	buf.Write([]byte("garbage prefix\n"))
	buf.Write(utf16Bytes)
	buf.Write([]byte("\ntrailer"))

	got, err := Scan(buf.Bytes())
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if got == nil {
		t.Fatal("Scan returned nil for UTF-16BE wrapper")
	}
	v, _ := PacketGetValue[Text](got, dcNS, "title")
	if v.V != "héllo" {
		t.Errorf("got %q, want héllo", v.V)
	}
}

func TestScan_UTF16LE(t *testing.T) {
	p := NewPacket()
	p.SetValue(dcNS, "title", Text{V: "héllo"})
	utf8Bytes := encodePacket(t, p)
	utf16Bytes := transcodeUTF8ToUTF16(utf8Bytes, false)

	var buf bytes.Buffer
	buf.Write([]byte("garbage prefix\n"))
	buf.Write(utf16Bytes)
	buf.Write([]byte("\ntrailer"))

	got, err := Scan(buf.Bytes())
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if got == nil {
		t.Fatal("Scan returned nil for UTF-16LE wrapper")
	}
	v, _ := PacketGetValue[Text](got, dcNS, "title")
	if v.V != "héllo" {
		t.Errorf("got %q, want héllo", v.V)
	}
}

func TestScan_PrefersDocumentLevelAcrossEncodings(t *testing.T) {
	// per-image UTF-16BE packet
	imgPacket := NewPacket()
	imgPacket.About = &url.URL{Scheme: "urn", Opaque: "img:1"}
	imgPacket.SetValue(dcNS, "title", Text{V: "Image"})
	imgUTF16 := transcodeUTF8ToUTF16(encodePacket(t, imgPacket), true)

	// document-level UTF-8 packet (rdf:about="")
	docPacket := NewPacket()
	docPacket.SetValue(dcNS, "title", Text{V: "Document"})

	var buf bytes.Buffer
	buf.Write(imgUTF16)
	buf.Write([]byte("\nbinary stream\n"))
	buf.Write(encodePacket(t, docPacket))

	got, err := Scan(buf.Bytes())
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if got == nil {
		t.Fatal("Scan returned nil")
	}
	v, _ := PacketGetValue[Text](got, dcNS, "title")
	if v.V != "Document" {
		t.Errorf("got title %q, want Document (document-level UTF-8 packet)", v.V)
	}
	if got.About != nil {
		t.Errorf("expected About=nil, got %v", got.About)
	}
}

func TestScan_SkipsEmptyPacket(t *testing.T) {
	// An empty wrapper has rdf:about="" and no properties; without the
	// "skip empty" rule, Scan would short-circuit on it as document-
	// level and miss a real packet later in the data.
	emptyPacket := NewPacket()
	realPacket := NewPacket()
	realPacket.SetValue(dcNS, "title", Text{V: "Real"})

	var buf bytes.Buffer
	buf.Write(encodePacket(t, emptyPacket))
	buf.Write([]byte("\nbinary stream contents\n"))
	buf.Write(encodePacket(t, realPacket))

	got, err := Scan(buf.Bytes())
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if got == nil {
		t.Fatal("Scan returned nil")
	}
	v, _ := PacketGetValue[Text](got, dcNS, "title")
	if v.V != "Real" {
		t.Errorf("got title %q, want Real", v.V)
	}
}

func TestScan_AllPacketsEmptyReturnsNil(t *testing.T) {
	// Two empty wrappers; nothing useful to return.
	var buf bytes.Buffer
	buf.Write(encodePacket(t, NewPacket()))
	buf.Write([]byte("\n"))
	buf.Write(encodePacket(t, NewPacket()))

	got, err := Scan(buf.Bytes())
	if got != nil {
		t.Errorf("expected nil for all-empty input, got %+v", got)
	}
	if err != nil {
		t.Errorf("expected nil error for all-empty input, got %v", err)
	}
}

func TestScan_RequiresMagicID(t *testing.T) {
	// An xpacket-shaped processing instruction that lacks the magic
	// XMP packet ID is not an XMP packet and Scan should not return
	// it, even if the surrounding XML happens to look plausible.
	data := []byte(`<?xpacket begin="" id="not-the-magic-id"?>` +
		`<x:xmpmeta xmlns:x="adobe:ns:meta/">` +
		`<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">` +
		`<rdf:Description rdf:about="" xmlns:dc="http://purl.org/dc/elements/1.1/">` +
		`<dc:title>Not XMP</dc:title>` +
		`</rdf:Description></rdf:RDF></x:xmpmeta>` +
		`<?xpacket end="r"?>`)
	got, err := Scan(data)
	if got != nil {
		t.Errorf("expected nil (no magic ID), got %+v", got)
	}
	if err != nil {
		t.Errorf("expected nil error (no magic ID), got %v", err)
	}
}

func TestScan_StrayXpacketBeforeRealWrapper(t *testing.T) {
	// A non-XMP <?xpacket ...?> processing instruction earlier in the
	// host bytes must not capture a real wrapper that follows.  With
	// the previous "<?xpacket begin=" sentinel this could happen when
	// the stray PI's matched end sentinel was the real wrapper's end
	// PI; anchoring on the magic ID rules it out.
	p := NewPacket()
	p.SetValue(dcNS, "title", Text{V: "Real"})

	var buf bytes.Buffer
	buf.WriteString(`<?xpacket begin="" id="other-tool"?>` +
		`some loose <not-properly-closed bytes`)
	buf.Write(encodePacket(t, p))
	buf.WriteString(`<?xpacket end="r"?>`) // looks-like end PI from a different tool

	got, err := Scan(buf.Bytes())
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if got == nil {
		t.Fatal("Scan returned nil; expected the real wrapper")
	}
	v, _ := PacketGetValue[Text](got, dcNS, "title")
	if v.V != "Real" {
		t.Errorf("got title %q, want Real", v.V)
	}
}

func TestScan_StopsAfterTooManyParseFailures(t *testing.T) {
	// Build many corrupt outer wrappers (each containing the magic id
	// attribute in a <?xpacket?> PI followed by malformed XML),
	// followed by a real wrapper.  Each corrupt outer's findWrapper
	// returns a range engulfing everything down to the real end PI,
	// so each parse fails.  Beyond maxScanParseFailures, Scan must
	// give up — the real wrapper that follows is unreachable.
	p := NewPacket()
	p.SetValue(dcNS, "title", Text{V: "Real"})

	var buf bytes.Buffer
	for range maxScanParseFailures + 1 {
		buf.WriteString(`<?xpacket id="` + xmpPacketID + `" begin="x"?>`)
		buf.WriteString("\n<unclosed-tag\n")
	}
	buf.Write(encodePacket(t, p))

	got, err := Scan(buf.Bytes())
	if got != nil {
		t.Errorf("Scan should have given up after the failure cap; got %+v", got)
	}
	if err == nil {
		t.Error("expected non-nil error after exceeding failure cap")
	}
	if !errors.Is(err, ErrMalformed) {
		t.Errorf("expected error to wrap ErrMalformed, got %v", err)
	}
}

func TestScan_EngulfedInnerWrapper(t *testing.T) {
	// A corrupt outer <?xpacket ... id="W5M0..."?> PI matches as a
	// begin sentinel and the only end PI in the data is the real
	// wrapper's, so findWrapper returns a range that engulfs the real
	// wrapper.  Read of that outer range fails on the malformed XML
	// between the outer PI and the inner wrapper.  Scan must retry
	// past the outer id attribute and find the inner real wrapper.
	p := NewPacket()
	p.SetValue(dcNS, "title", Text{V: "Engulfed"})

	var buf bytes.Buffer
	buf.WriteString(`<?xpacket id="` + xmpPacketID + `" begin="x"?>`)
	buf.WriteString("\n<unclosed-tag\n") // unterminated start element
	buf.Write(encodePacket(t, p))

	got, err := Scan(buf.Bytes())
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if got == nil {
		t.Fatal("Scan returned nil; expected the engulfed inner wrapper")
	}
	v, _ := PacketGetValue[Text](got, dcNS, "title")
	if v.V != "Engulfed" {
		t.Errorf("got title %q, want Engulfed", v.V)
	}
}

func TestScan_SingleQuotedID(t *testing.T) {
	// XML allows either quote style on attributes; the encoder emits
	// double quotes, but Scan must accept a hand-crafted wrapper with
	// single-quoted id as well.
	body := []byte(`<?xpacket begin="" id='` + xmpPacketID + `'?>` +
		`<x:xmpmeta xmlns:x="adobe:ns:meta/">` +
		`<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">` +
		`<rdf:Description rdf:about="" xmlns:dc="http://purl.org/dc/elements/1.1/">` +
		`<dc:title>SingleQuoted</dc:title>` +
		`</rdf:Description></rdf:RDF></x:xmpmeta>` +
		`<?xpacket end="r"?>`)
	got, err := Scan(body)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if got == nil {
		t.Fatal("Scan returned nil for single-quoted id attribute")
	}
	v, _ := PacketGetValue[Text](got, dcNS, "title")
	if v.V != "SingleQuoted" {
		t.Errorf("got title %q, want SingleQuoted", v.V)
	}
}

func TestScan_MagicIDInPlainTextBeforeRealWrapper(t *testing.T) {
	// The magic packet ID appearing in CharData earlier in the host
	// bytes must not mask a real wrapper that follows.  An earlier
	// version of findWrapper anchored on the first ID match and gave
	// up if it was outside any <?xpacket ?> PI; the current version
	// advances past the bad candidate and keeps looking.
	p := NewPacket()
	p.SetValue(dcNS, "title", Text{V: "Real"})

	var buf bytes.Buffer
	buf.WriteString("documentation: the magic XMP ID is " + xmpPacketID + "\n")
	buf.Write(encodePacket(t, p))

	got, err := Scan(buf.Bytes())
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if got == nil {
		t.Fatal("Scan returned nil; expected the real wrapper")
	}
	v, _ := PacketGetValue[Text](got, dcNS, "title")
	if v.V != "Real" {
		t.Errorf("got title %q, want Real", v.V)
	}
}

func TestScan_MagicIDOutsideProcInstIgnored(t *testing.T) {
	// The magic packet ID appearing in plain CharData (not inside an
	// <?xpacket ...?> PI) is just text and must not be treated as a
	// wrapper.  Even if it sits adjacent to an unrelated <?xpacket?>.
	data := []byte(`<?xpacket begin="" id="other"?>noise` +
		`?> not a PI ` + xmpPacketID + ` more text`)
	got, err := Scan(data)
	if got != nil {
		t.Errorf("expected nil (magic ID outside PI), got %+v", got)
	}
	if err != nil {
		t.Errorf("expected nil error (magic ID outside PI), got %v", err)
	}
}

func TestScan_GarbageAroundValidWrapper(t *testing.T) {
	p := NewPacket()
	p.SetValue(dcNS, "title", Text{V: "Anywhere"})
	wrapped := encodePacket(t, p)

	var buf bytes.Buffer
	// random bytes including some that almost look like sentinels
	buf.Write([]byte{0x00, 0x01, 0xff, 0x80})
	buf.Write([]byte("%PDF-1.7\n<<>>stream\n"))
	buf.Write(wrapped)
	buf.Write([]byte("\nendstream\n%%EOF"))

	got, err := Scan(buf.Bytes())
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if got == nil {
		t.Fatal("expected packet found inside surrounding bytes")
	}
	v, _ := PacketGetValue[Text](got, dcNS, "title")
	if v.V != "Anywhere" {
		t.Errorf("got %q, want Anywhere", v.V)
	}
}

// TestScan_LinearOnAdversarialIDs guards against quadratic blow-up in
// findWrapper.  An earlier version called bytes.LastIndex(data[:idAt],
// "<?xpacket") for every id-attribute candidate, walking the entire
// prefix on each iteration.  An attacker-controlled buffer of bare
// id="..." matches with no <?xpacket anywhere then made Scan
// Θ(n²)-time: 25 K bare ids in ~730 KiB took over ten seconds.  With
// linear behaviour the same input completes in milliseconds.
func TestScan_LinearOnAdversarialIDs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping CPU-time regression test in -short mode")
	}
	const id = `id="` + xmpPacketID + `"`
	data := bytes.Repeat([]byte(id+" "), 50000)

	const limit = time.Second
	start := time.Now()
	got, err := Scan(data)
	elapsed := time.Since(start)

	if got != nil {
		t.Errorf("expected nil packet for adversarial input, got %+v", got)
	}
	if err != nil {
		t.Errorf("expected nil error for adversarial input, got %v", err)
	}
	if elapsed > limit {
		t.Fatalf("Scan on %d-byte adversarial input took %v, want < %v (quadratic regression?)",
			len(data), elapsed, limit)
	}
}

// TestScan_MalformedWrapperReturnsError checks that Scan reports an
// error when XMP-shaped wrappers are present but every parse fails.
func TestScan_MalformedWrapperReturnsError(t *testing.T) {
	// Properly bracketed begin/end wrapper, but the XML between is
	// malformed (unclosed start tag, missing rdf:RDF).
	data := []byte(`<?xpacket begin="" id="` + xmpPacketID + `"?>` +
		`<x:xmpmeta xmlns:x="adobe:ns:meta/"><not-closed` +
		`<?xpacket end="r"?>`)
	got, err := Scan(data)
	if got != nil {
		t.Errorf("expected nil packet for malformed wrapper, got %+v", got)
	}
	if err == nil {
		t.Fatal("expected non-nil error for malformed wrapper")
	}
	if !errors.Is(err, ErrMalformed) {
		t.Errorf("expected error to wrap ErrMalformed, got %v", err)
	}
}
