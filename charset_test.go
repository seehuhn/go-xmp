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
	"io"
	"testing"
	"unicode/utf8"

	"github.com/google/go-cmp/cmp"
)

// transcodeUTF8ToUTF16 converts UTF-8 bytes into UTF-16 (BE or LE) for
// use as test input.  Invalid UTF-8 is passed through as U+FFFD per
// utf8.DecodeRune; tests in this file only feed it well-formed UTF-8
// produced by the encoder.
func transcodeUTF8ToUTF16(data []byte, bigEndian bool) []byte {
	out := make([]byte, 0, 2*len(data))
	for i := 0; i < len(data); {
		r, size := utf8.DecodeRune(data[i:])
		i += size
		out = appendUTF16(out, r, bigEndian)
	}
	return out
}

func appendUTF16(out []byte, r rune, bigEndian bool) []byte {
	if r < 0x10000 {
		if bigEndian {
			return append(out, byte(r>>8), byte(r))
		}
		return append(out, byte(r), byte(r>>8))
	}
	r -= 0x10000
	hi := uint16(0xD800 | (r >> 10))
	lo := uint16(0xDC00 | (r & 0x3FF))
	if bigEndian {
		return append(out, byte(hi>>8), byte(hi), byte(lo>>8), byte(lo))
	}
	return append(out, byte(hi), byte(hi>>8), byte(lo), byte(lo>>8))
}

func TestRead_UTF16BE(t *testing.T) {
	p := NewPacket()
	p.SetValue(dcNS, "title", Text{V: "héllo 𝄞"})
	var buf bytes.Buffer
	if err := p.Write(&buf, nil); err != nil {
		t.Fatalf("encode: %v", err)
	}
	utf16be := transcodeUTF8ToUTF16(buf.Bytes(), true)

	got, err := Read(bytes.NewReader(utf16be))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if d := cmp.Diff(p, got, cmp.AllowUnexported(Packet{})); d != "" {
		t.Errorf("UTF-16BE round trip mismatch (-want +got):\n%s", d)
	}
}

func TestRead_UTF16LE(t *testing.T) {
	p := NewPacket()
	p.SetValue(dcNS, "title", Text{V: "héllo 𝄞"})
	var buf bytes.Buffer
	if err := p.Write(&buf, nil); err != nil {
		t.Fatalf("encode: %v", err)
	}
	utf16le := transcodeUTF8ToUTF16(buf.Bytes(), false)

	got, err := Read(bytes.NewReader(utf16le))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if d := cmp.Diff(p, got, cmp.AllowUnexported(Packet{})); d != "" {
		t.Errorf("UTF-16LE round trip mismatch (-want +got):\n%s", d)
	}
}

// TestRead_UTF16WithEncodingDeclaration covers the realistic case in
// which a UTF-16 XMP packet carries an explicit
// `<?xml version="1.0" encoding="UTF-16"?>` declaration.  packetReader
// transcodes the byte stream to UTF-8, but the XML declaration that
// xml.Decoder then parses still reads "UTF-16"; without an identity
// CharsetReader to bridge the gap, decoding fails.
func TestRead_UTF16WithEncodingDeclaration(t *testing.T) {
	const utf8Src = `<?xml version="1.0" encoding="UTF-16"?>
<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
  <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
    <rdf:Description rdf:about="" xmlns:dc="http://purl.org/dc/elements/1.1/">
      <dc:title>Hello</dc:title>
    </rdf:Description>
  </rdf:RDF>
</x:xmpmeta>
<?xpacket end="r"?>`

	for _, tc := range []struct {
		name      string
		bigEndian bool
	}{
		{"UTF-16BE", true},
		{"UTF-16LE", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := transcodeUTF8ToUTF16([]byte(utf8Src), tc.bigEndian)
			p, err := Read(bytes.NewReader(body))
			if err != nil {
				t.Fatalf("Read: %v", err)
			}
			v, err := PacketGetValue[Text](p, dcNS, "title")
			if err != nil {
				t.Fatalf("title lookup: %v", err)
			}
			if v.V != "Hello" {
				t.Errorf("got %q, want Hello", v.V)
			}
		})
	}
}

// TestRead_LeadingBOM exercises packets that begin with a Unicode
// byte-order mark.  The stdlib XML decoder strips a leading UTF-8 BOM
// on its own; UTF-16 BOMs are only recognised by packetReader's sniff.
func TestRead_LeadingBOM(t *testing.T) {
	p := NewPacket()
	p.SetValue(dcNS, "title", Text{V: "héllo 𝄞"})
	var buf bytes.Buffer
	if err := p.Write(&buf, nil); err != nil {
		t.Fatalf("encode: %v", err)
	}
	utf8Body := buf.Bytes()

	cases := []struct {
		name string
		body []byte
	}{
		{
			name: "UTF-8 BOM",
			body: append([]byte{0xEF, 0xBB, 0xBF}, utf8Body...),
		},
		{
			name: "UTF-16BE BOM",
			body: append([]byte{0xFE, 0xFF}, transcodeUTF8ToUTF16(utf8Body, true)...),
		},
		{
			name: "UTF-16LE BOM",
			body: append([]byte{0xFF, 0xFE}, transcodeUTF8ToUTF16(utf8Body, false)...),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Read(bytes.NewReader(tc.body))
			if err != nil {
				t.Fatalf("Read: %v", err)
			}
			if d := cmp.Diff(p, got, cmp.AllowUnexported(Packet{})); d != "" {
				t.Errorf("round trip mismatch (-want +got):\n%s", d)
			}
		})
	}
}

func TestRead_TooLarge(t *testing.T) {
	src := &repeatReader{b: 'a', remaining: maxPacketSize + 1}
	_, err := Read(src)
	if !errors.Is(err, ErrPacketTooLarge) {
		t.Errorf("got err %v, want ErrPacketTooLarge", err)
	}
}

func TestPacketReader_AtCap(t *testing.T) {
	src := &repeatReader{b: 'a', remaining: maxPacketSize}
	pr, err := newPacketReader(src)
	if err != nil {
		t.Fatalf("newPacketReader: %v", err)
	}
	total, drainErr := drainCount(pr)
	if drainErr != io.EOF {
		t.Fatalf("drain at cap: got %v, want io.EOF", drainErr)
	}
	if total != maxPacketSize {
		t.Errorf("drained %d bytes, want %d", total, maxPacketSize)
	}
}

func TestPacketReader_OneByteOver(t *testing.T) {
	src := &repeatReader{b: 'a', remaining: maxPacketSize + 1}
	pr, err := newPacketReader(src)
	if err != nil {
		t.Fatalf("newPacketReader: %v", err)
	}
	_, drainErr := drainCount(pr)
	if !errors.Is(drainErr, ErrPacketTooLarge) {
		t.Errorf("drain over cap: got %v, want ErrPacketTooLarge", drainErr)
	}
}

func TestPacketReader_MalformedUTF16(t *testing.T) {
	// Each input begins with `0x00 0x3C` ('<' in UTF-16BE) so that
	// sniffEncoding picks UTF-16BE; the malformed bytes follow.
	cases := []struct {
		name string
		in   []byte
		want string
	}{
		{
			name: "odd_length",
			in:   []byte{0x00, 0x3C, 0x00, 0x41, 0x00},
			want: "<A�",
		},
		{
			name: "lone_high_surrogate_at_end",
			in:   []byte{0x00, 0x3C, 0xD8, 0x00},
			want: "<�",
		},
		{
			name: "lone_low_surrogate",
			in:   []byte{0x00, 0x3C, 0xDC, 0x00, 0x00, 0x41},
			want: "<�A",
		},
		{
			name: "high_followed_by_non_low",
			in:   []byte{0x00, 0x3C, 0xD8, 0x00, 0x00, 0x41},
			want: "<�A",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pr, err := newPacketReader(bytes.NewReader(tc.in))
			if err != nil {
				t.Fatalf("newPacketReader: %v", err)
			}
			out, drainErr := drainBytes(pr)
			if drainErr != io.EOF {
				t.Fatalf("drain: got %v, want io.EOF", drainErr)
			}
			if got := string(out); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPacketReader_StuckSource(t *testing.T) {
	// A source that always returns (0, nil) is technically allowed by
	// io.Reader but indicates misbehaviour.  packetReader must surface
	// io.ErrNoProgress instead of looping or recursing forever.
	pr, err := newPacketReader(stuckReader{})
	if err != nil {
		t.Fatalf("newPacketReader: %v", err)
	}
	_, err = pr.ReadByte()
	if !errors.Is(err, io.ErrNoProgress) {
		t.Errorf("got %v, want io.ErrNoProgress", err)
	}
}

func TestPacketReader_OneByteAtATime(t *testing.T) {
	// Build a UTF-16BE input containing a non-BMP rune (U+1D11E,
	// MUSICAL SYMBOL G CLEF) so the surrogate pair is exercised, and
	// feed it through a one-byte-per-Read reader to force refills mid
	// code unit.
	utf8Src := []byte("<\U0001D11E>")
	utf16 := transcodeUTF8ToUTF16(utf8Src, true)

	pr, err := newPacketReader(&oneByteReader{src: bytes.NewReader(utf16)})
	if err != nil {
		t.Fatalf("newPacketReader: %v", err)
	}
	out, drainErr := drainBytes(pr)
	if drainErr != io.EOF {
		t.Fatalf("drain: got %v, want io.EOF", drainErr)
	}
	if got := string(out); got != string(utf8Src) {
		t.Errorf("got %q, want %q", got, string(utf8Src))
	}
}

func FuzzRoundTrip_UTF16(f *testing.F) {
	for _, tc := range encodeTestCases {
		buf := &bytes.Buffer{}
		if err := tc.in.Write(buf, nil); err != nil {
			f.Fatal(err)
		}
		f.Add(transcodeUTF8ToUTF16(buf.Bytes(), true))
		f.Add(transcodeUTF8ToUTF16(buf.Bytes(), false))
	}

	f.Fuzz(func(t *testing.T, body []byte) {
		// Read transparently detects UTF-16 from the leading bytes;
		// Write always emits UTF-8.  The packet should round-trip
		// semantically regardless of input encoding.
		p1, err := Read(bytes.NewReader(body))
		if err != nil {
			return
		}
		buf := &bytes.Buffer{}
		if err := p1.Write(buf, nil); err != nil {
			t.Fatal(err)
		}
		p2, err := Read(bytes.NewReader(buf.Bytes()))
		if err != nil {
			t.Fatal(err)
		}
		if d := cmp.Diff(p1, p2, cmp.AllowUnexported(Packet{})); d != "" {
			t.Fatalf("UTF-16 round-trip mismatch (-want +got):\n%s", d)
		}
	})
}

// repeatReader yields a fixed byte up to a configured count, then
// returns io.EOF.  Used by the cap tests to avoid a multi-MB allocation.
type repeatReader struct {
	b         byte
	remaining int64
}

func (r *repeatReader) Read(p []byte) (int, error) {
	if r.remaining <= 0 {
		return 0, io.EOF
	}
	n := min(int64(len(p)), r.remaining)
	for i := range n {
		p[i] = r.b
	}
	r.remaining -= n
	return int(n), nil
}

// stuckReader always reports (0, nil) and is used to verify that
// packetReader does not hang or recurse on a misbehaving source.
type stuckReader struct{}

func (stuckReader) Read(p []byte) (int, error) { return 0, nil }

// oneByteReader serves at most one byte per Read call, which forces
// refills to land mid code unit and exercises the streaming reader's
// boundary handling.
type oneByteReader struct {
	src io.Reader
}

func (r *oneByteReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	return r.src.Read(p[:1])
}

// drainCount calls ReadByte until an error is returned and reports the
// total number of successfully delivered bytes.
func drainCount(pr *packetReader) (int64, error) {
	var n int64
	buf := make([]byte, 64<<10)
	for {
		m, err := pr.Read(buf)
		n += int64(m)
		if err != nil {
			return n, err
		}
	}
}

// drainBytes collects all bytes delivered by pr until the first error.
func drainBytes(pr *packetReader) ([]byte, error) {
	var out []byte
	buf := make([]byte, 256)
	for {
		m, err := pr.Read(buf)
		out = append(out, buf[:m]...)
		if err != nil {
			return out, err
		}
	}
}
