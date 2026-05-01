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
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.org/x/text/language"
)

func TestTag(t *testing.T) {
	dc1 := &DublinCore{}
	dc1.Date.Append(NewDate(time.Now()))
	dc1.Title.Set(language.English, "Hello, World!")
	dc1.Title.Set(language.German, "Grüß Gott!")
	dc1.Title.Default = NewText("Hello, World!")

	p := NewPacket()
	err := p.Set(dc1)
	if err != nil {
		t.Fatal(err)
	}

	dc2 := DublinCore{}
	if err := p.Get(&dc2); err != nil {
		t.Fatalf("Get: %v", err)
	}

	if d := cmp.Diff(dc1, &dc2); d != "" {
		t.Errorf("dc1 and dc2 differ (-want +got):\n%s", d)
	}
}

func TestRoundTripDublinCore(t *testing.T) {
	in := &DublinCore{
		Coverage:   NewText("Worldwide"),
		Format:     MimeType{V: "application/pdf"},
		Identifier: NewText("urn:isbn:0451450523"),
		Source:     NewText("urn:isbn:0451450524"),
	}
	in.Contributor.Append(NewProperName("Alice"))
	in.Contributor.Append(NewProperName("Bob"))
	in.Creator.Append(NewProperName("Carol"))
	in.Date.Append(NewDate(time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)))
	in.Description.Set(language.English, "an example")
	in.Language.Append(NewLocale(language.English))
	in.Publisher.Append(NewProperName("Acme"))
	in.Relation.V = []Text{NewText("urn:isbn:0451450525")}
	in.Rights.Set(language.English, "All rights reserved")
	in.Subject.V = []Text{NewText("example"), NewText("test")}
	in.Title.Default = NewText("Example")
	in.Title.Set(language.English, "Example")
	in.Type.V = []Text{NewText("Text")}

	out := &DublinCore{}
	roundTrip(t, in, out)
}

func TestRoundTripBasic(t *testing.T) {
	in := &Basic{
		CreateDate:   NewDate(time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)),
		CreatorTool:  NewAgentName("Acme Editor 1.0"),
		Label:        NewText("draft"),
		MetadataDate: NewDate(time.Date(2024, 6, 7, 8, 9, 10, 0, time.UTC)),
		ModifyDate:   NewDate(time.Date(2024, 6, 7, 8, 9, 11, 0, time.UTC)),
		Rating:       Real{V: 4},
	}
	in.Identifier.V = []Text{NewText("urn:example:1")}

	out := &Basic{}
	roundTrip(t, in, out)
}

func TestRoundTripRightsManagement(t *testing.T) {
	in := &RightsManagement{
		Certificate:  NewText("https://example.com/cert"),
		Marked:       OptionalBool{V: 2},
		WebStatement: NewText("https://example.com/rights"),
	}
	in.Owner.Append(NewProperName("Acme Inc."))
	in.UsageTerms.Set(language.English, "Use as you wish.")

	out := &RightsManagement{}
	roundTrip(t, in, out)
}

func TestRoundTripPDF(t *testing.T) {
	in := &PDF{
		Keywords:   NewText("xmp, pdf, metadata"),
		PDFVersion: NewText("2.0"),
		Producer:   NewAgentName("seehuhn.de/go/xmp test"),
		Trapped:    NewText("False"),
	}

	out := &PDF{}
	roundTrip(t, in, out)
}

func TestRoundTripPDFAID(t *testing.T) {
	// PDF/A-2u
	in := &PDFAID{
		Part:        NewInteger(2),
		Conformance: NewText("U"),
	}
	roundTrip(t, in, &PDFAID{})

	// PDF/A-4 with rev and amendment
	inA4 := &PDFAID{
		Part: NewInteger(4),
		Rev:  NewInteger(2020),
		Amd:  NewText("1:2025"),
	}
	roundTrip(t, inA4, &PDFAID{})
}

func TestRoundTripPDFX(t *testing.T) {
	in := &PDFX{
		Version:     NewText("PDF/X-1:2001"),
		Conformance: NewText("PDF/X-1a:2001"),
	}
	roundTrip(t, in, &PDFX{})
}

func TestRoundTripPDFXID(t *testing.T) {
	in := &PDFXID{
		Version: NewText("PDF/X-4"),
	}
	roundTrip(t, in, &PDFXID{})
}

func TestRoundTripMediaManagement(t *testing.T) {
	in := &MediaManagement{
		DerivedFrom: ResourceRef{
			DocumentID:      GUID{V: "uuid:source-doc"},
			InstanceID:      GUID{V: "uuid:source-instance"},
			RenditionClass:  RenditionClass{V: "default"},
			RenditionParams: NewText("page=1"),
		},
		DocumentID:         NewText("uuid:doc"),
		InstanceID:         NewText("uuid:inst"),
		OriginalDocumentID: NewText("uuid:orig"),
		RenditionClass:     RenditionClass{V: "draft"},
		RenditionParams:    NewText("page=2"),
	}

	out := &MediaManagement{}
	roundTrip(t, in, out)
}

// TestGetReturnsDecodeError checks that [Packet.Get] surfaces decode
// failures via [errors.Join] of [*PropertyError] values, while still
// populating fields whose data decoded successfully.
func TestGetReturnsDecodeError(t *testing.T) {
	p := NewPacket()
	// good field: pdfaid:part = 2
	p.SetValue(NSPDFAID, "part", NewInteger(2))
	// bad field: pdfaid:rev set to a non-numeric Text
	p.SetValue(NSPDFAID, "rev", Text{V: "not a number"})

	var got PDFAID
	err := p.Get(&got)
	if err == nil {
		t.Fatal("Get: expected error, got nil")
	}
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("errors.Is(err, ErrInvalid) = false, want true (err: %v)", err)
	}

	var pe *PropertyError
	if !errors.As(err, &pe) {
		t.Fatalf("errors.As(*PropertyError) = false, want true (err: %v)", err)
	}
	want := xml.Name{Space: NSPDFAID, Local: "rev"}
	if pe.Name != want {
		t.Errorf("PropertyError.Name = %v, want %v", pe.Name, want)
	}

	// the good field still populates
	if got.Part.V != 2 {
		t.Errorf("Part.V = %d, want 2", got.Part.V)
	}
	// the bad field was left at the zero value
	if got.Rev.V != 0 {
		t.Errorf("Rev.V = %d, want 0 (zero on decode failure)", got.Rev.V)
	}
}

// TestGetReturnsDecodeErrorArray checks that decode failures from
// nested array elements still propagate as [*PropertyError] wrapping
// [ErrInvalid], so the contract documented on [Packet.Get] holds for
// non-scalar types too.
func TestGetReturnsDecodeErrorArray(t *testing.T) {
	p := NewPacket()
	// dc:creator must be an OrderedArray of ProperName (Text).  Plant a
	// non-Text entry so OrderedArray.DecodeAnother propagates the inner
	// element's ErrInvalid up the stack.
	p.Properties[xml.Name{Space: NSDublinCore, Local: "creator"}] = RawArray{
		Kind:  Ordered,
		Value: []Raw{Text{V: "Alice"}, RawArray{}},
	}

	var dc DublinCore
	err := p.Get(&dc)
	if err == nil {
		t.Fatal("Get: expected error, got nil")
	}
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("errors.Is(err, ErrInvalid) = false, want true (err: %v)", err)
	}
	var pe *PropertyError
	if !errors.As(err, &pe) {
		t.Fatalf("errors.As(*PropertyError) = false, want true (err: %v)", err)
	}
	want := xml.Name{Space: NSDublinCore, Local: "creator"}
	if pe.Name != want {
		t.Errorf("PropertyError.Name = %v, want %v", pe.Name, want)
	}
}

// roundTrip stores in into a fresh packet via [Packet.Set], reads it back
// into out via [Packet.Get], and reports any differences.
func roundTrip(t *testing.T, in, out any) {
	t.Helper()
	p := NewPacket()
	if err := p.Set(in); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := p.Get(out); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if d := cmp.Diff(in, out, cmpopts.EquateComparable(language.Tag{})); d != "" {
		t.Errorf("round trip failed (-want +got):\n%s", d)
	}
}
