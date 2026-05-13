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
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
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

// TestScan_UTF16BE_BytesFormValidMultibyteUTF8 guards a UTF-16
// regression: payload bytes that, viewed as UTF-8, form a valid
// multibyte codepoint used to throw off the per-encoding payload
// stepper.  Korean Hangul 캀 (U+CC80) encodes to UTF-16BE bytes
// 0xCC 0x80, which is also valid UTF-8 for U+0300.
func TestScan_UTF16BE_BytesFormValidMultibyteUTF8(t *testing.T) {
	p := NewPacket()
	p.SetValue(dcNS, "title", Text{V: "캀"})
	utf16Bytes := transcodeUTF8ToUTF16(encodePacket(t, p), true)

	got, err := Scan(utf16Bytes)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if got == nil {
		t.Fatal("Scan returned nil for UTF-16BE wrapper with U+CC80 payload")
	}
	v, _ := PacketGetValue[Text](got, dcNS, "title")
	if v.V != "캀" {
		t.Errorf("got %q, want 캀", v.V)
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

// TestScan_RejectsOversizedBeginPI checks that a wrapper whose begin
// processing instruction exceeds the regex's bounded look-ahead is
// rejected rather than slowing down or hanging the scanner.  The
// check runs for every supported encoding so an encoding-specific
// regression in the {0,200} bound is caught.
func TestScan_RejectsOversizedBeginPI(t *testing.T) {
	// 10 KiB of padding inside the begin PI dwarfs the regex's
	// bounded {0,200} look-ahead window in every encoding.
	padding := strings.Repeat("x", 10*1024)
	utf8Data := []byte(`<?xpacket begin="" ` + padding + ` id="` + xmpPacketID + `"?>` +
		`<x:xmpmeta xmlns:x="adobe:ns:meta/">` +
		`<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">` +
		`<rdf:Description rdf:about="" xmlns:dc="http://purl.org/dc/elements/1.1/">` +
		`<dc:title>Padded</dc:title>` +
		`</rdf:Description></rdf:RDF></x:xmpmeta>` +
		`<?xpacket end="r"?>`)

	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"UTF-8", utf8Data},
		{"UTF-16BE", transcodeUTF8ToUTF16(utf8Data, true)},
		{"UTF-16LE", transcodeUTF8ToUTF16(utf8Data, false)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Scan(tc.data)
			if got != nil {
				t.Errorf("expected nil for oversized begin-PI, got %+v", got)
			}
			if err != nil {
				t.Errorf("expected nil error for oversized begin-PI, got %v", err)
			}
		})
	}
}

// TestScan_PrefersDescriptionWithoutAboutAttribute checks that a
// document-level wrapper whose rdf:Description omits the rdf:about
// attribute (treated as rdf:about="" by the XMP spec) is preferred
// over a byte-earlier per-image wrapper.  The preference is keyed on
// the parsed Packet.About being nil rather than on a literal byte
// match of rdf:about="" in the source.
func TestScan_PrefersDescriptionWithoutAboutAttribute(t *testing.T) {
	// per-image wrapper with a non-empty rdf:about attribute
	imgWrapper := `<?xpacket begin="" id="` + xmpPacketID + `"?>` +
		`<x:xmpmeta xmlns:x="adobe:ns:meta/">` +
		`<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">` +
		`<rdf:Description rdf:about="urn:img:1" xmlns:dc="http://purl.org/dc/elements/1.1/">` +
		`<dc:title>Image</dc:title>` +
		`</rdf:Description></rdf:RDF></x:xmpmeta>` +
		`<?xpacket end="r"?>`

	// document-level wrapper that omits rdf:about entirely
	docWrapper := `<?xpacket begin="" id="` + xmpPacketID + `"?>` +
		`<x:xmpmeta xmlns:x="adobe:ns:meta/">` +
		`<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">` +
		`<rdf:Description xmlns:dc="http://purl.org/dc/elements/1.1/">` +
		`<dc:title>Document</dc:title>` +
		`</rdf:Description></rdf:RDF></x:xmpmeta>` +
		`<?xpacket end="r"?>`

	data := []byte(imgWrapper + "\n" + docWrapper)

	got, err := Scan(data)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if got == nil {
		t.Fatal("Scan returned nil")
	}
	if got.About != nil {
		t.Errorf("expected document-level packet (About=nil), got About=%v", got.About)
	}
	v, _ := PacketGetValue[Text](got, dcNS, "title")
	if v.V != "Document" {
		t.Errorf("got title %q, want Document", v.V)
	}
}

func TestScan_EngulfedInnerWrapper(t *testing.T) {
	// A corrupt outer <?xpacket ... id="W5M0..."?> PI is wrapper-
	// shaped, and the only end PI in the data is the real wrapper's,
	// so the scanner produces an outer candidate that engulfs the
	// real wrapper.  Read on that outer range fails on the malformed
	// XML between the outer PI and the inner wrapper; Scan must keep
	// considering inner candidates and recover the real wrapper.
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
	// The magic packet ID appearing in plain text earlier in the host
	// bytes must not mask a real wrapper that follows.  The scanner
	// requires the magic ID to sit inside a <?xpacket ... ?> PI, so a
	// bare occurrence in CharData is harmless and the real wrapper is
	// found.
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

// FuzzScan exercises the regex pass, match dedup, About-nil
// preference, and the (packet, error) contract on arbitrary inputs.
// Seeds cover single and back-to-back wrappers, UTF-16 variants, and
// wrappers embedded in noise resembling real host bytes.
//
// Invariants asserted on every input:
//   - Scan never panics.
//   - (packet != nil) implies (err == nil).
//   - If Scan returns a packet, re-emitting and re-scanning yields an
//     equivalent packet.
func FuzzScan(f *testing.F) {
	// canonical single wrapper
	seed := NewPacket()
	seed.SetValue(dcNS, "title", Text{V: "Seed"})
	var wrapper bytes.Buffer
	if err := seed.Write(&wrapper, nil); err != nil {
		f.Fatal(err)
	}
	wrapperBytes := wrapper.Bytes()

	// also reuse the package's encode test cases for variety
	wrappers := [][]byte{wrapperBytes}
	for _, tc := range encodeTestCases {
		var buf bytes.Buffer
		if err := tc.in.Write(&buf, nil); err == nil {
			wrappers = append(wrappers, buf.Bytes())
		}
	}

	prefixes := [][]byte{
		nil,
		[]byte("garbage prefix\n"),
		[]byte("%PDF-1.7\n<<>>stream\n"),
		{0x00, 0x01, 0xff, 0x80, 0x3c, 0x3f},
		[]byte(`<?xpacket begin="" id="other-tool"?>noise`),
		[]byte("documentation: the magic XMP ID is " + xmpPacketID + "\n"),
	}
	suffixes := [][]byte{
		nil,
		[]byte("\ntrailer junk"),
		[]byte("\nendstream\n%%EOF"),
		[]byte(`<?xpacket end="r"?>`),
	}

	for _, w := range wrappers {
		for _, p := range prefixes {
			for _, s := range suffixes {
				in := make([]byte, 0, len(p)+len(w)+len(s))
				in = append(in, p...)
				in = append(in, w...)
				in = append(in, s...)
				f.Add(in)
			}
		}
	}
	// UTF-16 variants of the canonical wrapper exercise the per-
	// encoding regex paths.
	f.Add(transcodeUTF8ToUTF16(wrapperBytes, true))
	f.Add(transcodeUTF8ToUTF16(wrapperBytes, false))
	// Two wrappers back-to-back exercise the preference / dedup paths.
	f.Add(append(append([]byte(nil), wrapperBytes...), wrapperBytes...))

	urlCmp := cmp.Comparer(func(u1, u2 *url.URL) bool {
		if u1 == nil && u2 == nil {
			return true
		}
		if u1 == nil || u2 == nil {
			return false
		}
		return u1.String() == u2.String()
	})

	f.Fuzz(func(t *testing.T, data []byte) {
		p1, err1 := Scan(data)
		if p1 != nil && err1 != nil {
			t.Fatalf("Scan returned both packet and error: packet=%+v, err=%v", p1, err1)
		}
		if p1 == nil {
			return
		}

		var buf bytes.Buffer
		if err := p1.Write(&buf, nil); err != nil {
			t.Fatalf("Write of returned packet: %v", err)
		}
		p2, err2 := Scan(buf.Bytes())
		if err2 != nil {
			t.Fatalf("Scan(Write(p)) returned error: %v\nwritten: %q", err2, buf.String())
		}
		if p2 == nil {
			t.Fatalf("Scan(Write(p)) returned nil\nwritten: %q", buf.String())
		}
		if d := cmp.Diff(p1, p2, urlCmp, cmp.AllowUnexported(Packet{})); d != "" {
			t.Fatalf("round-trip mismatch (-want +got):\n%s", d)
		}
	})
}
