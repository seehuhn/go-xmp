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
	"errors"
	"io"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
)

// maxPacketSize bounds the input accepted by Read and Scan.  Real-world
// XMP packets are well under 1 MiB; 16 MiB is a comfortable ceiling that
// guards against malicious or accidental unbounded readers.
const maxPacketSize = 16 << 20

// inBufSize is the source-side buffer used by packetReader.  Small on
// purpose: the reader streams byte-by-byte to xml.NewDecoder and
// transient memory should stay well under 2 KiB regardless of input
// size.
const inBufSize = 1024

// ErrPacketTooLarge is returned by [Read] when the input exceeds the
// internal size limit.
var ErrPacketTooLarge = errors.New("xmp: packet exceeds size limit")

// packetReader streams an XMP packet from a source io.Reader,
// transcoding UTF-16BE / UTF-16LE input to UTF-8 on the fly and
// enforcing maxPacketSize on the source-byte count.
//
// It implements io.ByteReader, which lets xml.NewDecoder consume bytes
// directly without adding its own bufio wrap (encoding/xml/xml.go
// switchToReader checks for io.ByteReader).
//
// Memory: one fixed inBuf (inBufSize bytes) for source bytes plus a
// 4-byte output queue for the at-most-one pending UTF-8 rune; the
// total per-reader footprint is independent of input size.
type packetReader struct {
	src io.Reader

	inBuf  [inBufSize]byte
	inPos  int
	inLen  int
	nRead  int // bytes consumed from src (cap accounting)
	eof    bool
	srcErr error // sticky non-EOF error from src

	outBuf [utf8.UTFMax]byte
	outPos int
	outLen int

	pendingHigh rune  // saved high surrogate, or 0
	pendingU    int32 // queued code unit after orphan-high replacement, or -1

	bigEndian bool
	isUTF16   bool

	tooLarge bool
}

// newPacketReader prepares a streaming reader and sniffs the input
// encoding from the first two bytes.  A UTF-16 BOM (FE FF or FF FE)
// is recognised and consumed; other sniffed bytes remain in the
// source buffer and are delivered normally by ReadByte.  A leading
// UTF-8 BOM is left in place because the XML decoder strips it
// itself.  The initial fill loops across short Read calls so that a
// one-byte-at-a-time source still yields a valid 2-byte sniff window.
func newPacketReader(src io.Reader) (*packetReader, error) {
	r := &packetReader{
		src:      src,
		pendingU: -1,
	}
	for r.inLen < 2 {
		quota := maxPacketSize + 1 - r.nRead
		if quota <= 0 {
			r.tooLarge = true
			break
		}
		avail := min(len(r.inBuf)-r.inLen, quota)
		m, err := src.Read(r.inBuf[r.inLen : r.inLen+avail])
		r.nRead += m
		r.inLen += m
		if r.nRead > maxPacketSize {
			r.tooLarge = true
			r.inLen = 0
			break
		}
		if err == io.EOF {
			r.eof = true
			break
		}
		if err != nil {
			r.srcErr = err
			break
		}
		if m == 0 {
			// no progress and no error; src misbehaving
			break
		}
	}
	if r.inLen >= 2 {
		switch {
		case r.inBuf[0] == 0xFE && r.inBuf[1] == 0xFF:
			// UTF-16BE BOM; consume so it does not reach the parser
			r.isUTF16, r.bigEndian = true, true
			r.inPos = 2
		case r.inBuf[0] == 0xFF && r.inBuf[1] == 0xFE:
			// UTF-16LE BOM; consume so it does not reach the parser
			r.isUTF16, r.bigEndian = true, false
			r.inPos = 2
		case r.inBuf[0] == 0x00 && r.inBuf[1] != 0x00:
			r.isUTF16, r.bigEndian = true, true
		case r.inBuf[0] != 0x00 && r.inBuf[1] == 0x00:
			r.isUTF16, r.bigEndian = true, false
		}
	}
	return r, nil
}

// Read satisfies io.Reader so packetReader can be passed to
// xml.NewDecoder.  In practice the XML decoder discovers the
// io.ByteReader implementation and uses ReadByte instead; this path
// is exercised by tests and by the unlikely consumer that calls Read
// directly.
//
// The UTF-8 case copies straight out of inBuf in chunks; UTF-16 falls
// back to per-byte transcoding via ReadByte.
func (r *packetReader) Read(p []byte) (int, error) {
	n := 0
	for n < len(p) {
		if r.outPos < r.outLen {
			c := copy(p[n:], r.outBuf[r.outPos:r.outLen])
			r.outPos += c
			n += c
			continue
		}
		if r.isUTF16 {
			b, err := r.ReadByte()
			if err != nil {
				if n > 0 {
					return n, nil
				}
				return 0, err
			}
			p[n] = b
			n++
			continue
		}
		if r.inPos >= r.inLen {
			if err := r.refill(); err != nil {
				if n > 0 {
					return n, nil
				}
				return 0, err
			}
		}
		c := copy(p[n:], r.inBuf[r.inPos:r.inLen])
		r.inPos += c
		n += c
	}
	return n, nil
}

// ReadByte serves the next UTF-8 byte of the (possibly transcoded)
// packet.  The returned error is io.EOF at end of stream,
// ErrPacketTooLarge if the source exceeded maxPacketSize, or any
// non-EOF error reported by src.
func (r *packetReader) ReadByte() (byte, error) {
	if r.outPos < r.outLen {
		b := r.outBuf[r.outPos]
		r.outPos++
		return b, nil
	}
	if !r.isUTF16 {
		if r.inPos >= r.inLen {
			if err := r.refill(); err != nil {
				return 0, err
			}
		}
		b := r.inBuf[r.inPos]
		r.inPos++
		return b, nil
	}
	ru, err := r.nextRune()
	if err != nil {
		return 0, err
	}
	r.outLen = utf8.EncodeRune(r.outBuf[:], ru)
	r.outPos = 1
	return r.outBuf[0], nil
}

// refill reads more bytes from src into inBuf, bounded so that
// nRead never exceeds maxPacketSize+1.  Crossing the cap sets the
// sticky tooLarge flag and discards the just-read bytes — once we
// know the input is too long, we don't serve any more of it.
func (r *packetReader) refill() error {
	if r.tooLarge {
		return ErrPacketTooLarge
	}
	if r.srcErr != nil {
		return r.srcErr
	}
	if r.eof {
		return io.EOF
	}

	quota := maxPacketSize + 1 - r.nRead
	if quota <= 0 {
		r.tooLarge = true
		return ErrPacketTooLarge
	}
	n := min(len(r.inBuf), quota)

	m, err := r.src.Read(r.inBuf[:n])
	r.nRead += m

	if r.nRead > maxPacketSize {
		// We deliberately read one byte past the limit so that
		// "fits exactly" and "exceeds" are distinguishable.  Drop
		// the buffer and surface the sentinel.
		r.tooLarge = true
		r.inLen = 0
		r.inPos = 0
		return ErrPacketTooLarge
	}

	r.inPos = 0
	r.inLen = m

	if err == io.EOF {
		r.eof = true
		if m > 0 {
			return nil
		}
		return io.EOF
	}
	if err != nil {
		r.srcErr = err
		if m > 0 {
			return nil
		}
		return err
	}
	if m == 0 {
		// io.Reader allows (0, nil) but discourages it.  We refuse to
		// retry: a source that yields zero bytes without progress is
		// either misbehaving or hostile, and looping or recursing here
		// would let it hang or stack-overflow Read.  Surface the
		// standard sentinel for this condition.
		return io.ErrNoProgress
	}
	return nil
}

// nextRune reads and returns the next decoded Unicode code point from
// a UTF-16 source.  Lone surrogates, broken pairs, and odd trailing
// bytes yield U+FFFD per the permissive-reader principle; an orphan
// high surrogate followed by a non-low-surrogate emits U+FFFD and
// the offending unit is reprocessed (Unicode best practice).
func (r *packetReader) nextRune() (rune, error) {
	for {
		var u uint16
		if r.pendingU >= 0 {
			u = uint16(r.pendingU)
			r.pendingU = -1
		} else {
			v, err := r.nextU16()
			if err != nil {
				if err == errOddByte {
					// odd trailing byte after at least one valid
					// code unit; emit one replacement covering both
					// the corruption and any pending orphan high
					// surrogate, then EOF
					r.pendingHigh = 0
					return unicode.ReplacementChar, nil
				}
				if r.pendingHigh != 0 && err == io.EOF {
					// emit replacement now; the next call will see
					// the same EOF via refill and surface it
					r.pendingHigh = 0
					return unicode.ReplacementChar, nil
				}
				return 0, err
			}
			u = v
		}

		if r.pendingHigh != 0 {
			if u >= 0xDC00 && u <= 0xDFFF {
				ru := utf16.DecodeRune(r.pendingHigh, rune(u))
				r.pendingHigh = 0
				return ru, nil
			}
			// orphan high; queue u for reprocessing on the next call
			r.pendingHigh = 0
			r.pendingU = int32(u)
			return unicode.ReplacementChar, nil
		}

		switch {
		case u >= 0xD800 && u <= 0xDBFF:
			r.pendingHigh = rune(u)
			continue // need the second code unit
		case u >= 0xDC00 && u <= 0xDFFF:
			return unicode.ReplacementChar, nil
		default:
			return rune(u), nil
		}
	}
}

// errOddByte is returned by nextU16 when EOF arrives between the two
// bytes of a code unit.  Callers translate it into a U+FFFD emission.
var errOddByte = errors.New("xmp: odd-length UTF-16 input")

// nextU16 reads two source bytes and assembles them into a uint16
// according to the configured endianness.
func (r *packetReader) nextU16() (uint16, error) {
	var b [2]byte
	for i := range 2 {
		if r.inPos >= r.inLen {
			if err := r.refill(); err != nil {
				if err == io.EOF && i > 0 {
					return 0, errOddByte
				}
				return 0, err
			}
			// refill returning nil guarantees r.inLen > 0:
			// (m=0, err=nil) yields ErrNoProgress, (m=0, err=EOF)
			// yields io.EOF, and the (m>0, *) cases set inLen = m.
		}
		b[i] = r.inBuf[r.inPos]
		r.inPos++
	}
	if r.bigEndian {
		return uint16(b[0])<<8 | uint16(b[1]), nil
	}
	return uint16(b[0]) | uint16(b[1])<<8, nil
}
