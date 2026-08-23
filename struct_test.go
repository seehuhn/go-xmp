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
	dc1.Date.Append(NewDate(time.Now(), PrecisionFull))
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
	in.Date.Append(NewDate(time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC), PrecisionSecond))
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
		CreateDate:   NewDate(time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC), PrecisionSecond),
		CreatorTool:  NewAgentName("Acme Editor 1.0"),
		Label:        NewText("draft"),
		MetadataDate: NewDate(time.Date(2024, 6, 7, 8, 9, 10, 0, time.UTC), PrecisionSecond),
		ModifyDate:   NewDate(time.Date(2024, 6, 7, 8, 9, 11, 0, time.UTC), PrecisionSecond),
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

// SharedProperties is embedded anonymously by EmbeddingFields below, to
// test that [Packet.Set] and [Packet.Get] promote the fields of an
// anonymously embedded struct into the enclosing namespace struct, the way
// encoding/json promotes fields of embedded structs.  This is the idiom a
// family of near-identical namespace versions (e.g. PRISM Basic 2.0/2.1/3.0)
// would use to share a field set without repeating it in every variant.
type SharedProperties struct {
	Foo Text `xmp:"foo"`
	Bar Text `xmp:"bar"`
}

type EmbeddingFields struct {
	_ Namespace `xmp:"http://example.com/test/embedding-fields/"`
	_ Prefix    `xmp:"ef"`
	SharedProperties
}

// SharedMarkers is embedded anonymously by EmbeddingMarkers below, to test
// that [Packet.Set] and [Packet.Get] find a [Namespace]/[Prefix] pair even
// when it is carried on an embedded struct rather than declared directly on
// the namespace struct itself.
type SharedMarkers struct {
	_ Namespace `xmp:"http://example.com/test/embedding-markers/"`
	_ Prefix    `xmp:"em"`
}

type EmbeddingMarkers struct {
	SharedMarkers
	Foo Text `xmp:"foo"`
}

// TestEmbeddedFieldsPromoted checks that fields of an anonymously embedded
// struct are promoted, i.e. are treated as if declared directly on the
// enclosing namespace struct: Set must write them to the packet, and Get
// must fill them.
func TestEmbeddedFieldsPromoted(t *testing.T) {
	in := &EmbeddingFields{}
	in.Foo = NewText("foo value")
	in.Bar = NewText("bar value")

	out := &EmbeddingFields{}
	roundTrip(t, in, out)
}

// TestEmbeddedMarkersPromoted checks that a [Namespace]/[Prefix] pair
// carried on an anonymously embedded struct is found by Set and Get,
// like any other promoted field.
func TestEmbeddedMarkersPromoted(t *testing.T) {
	in := &EmbeddingMarkers{}
	in.Foo = NewText("foo value")

	out := &EmbeddingMarkers{}
	roundTrip(t, in, out)
}

// ShadowBase is embedded by ShadowOuter below to test that a field
// declared directly on a namespace struct takes priority over a
// same-named field promoted from an embedded struct, the way
// encoding/json resolves shadowed fields by embedding depth.
type ShadowBase struct {
	Foo Text `xmp:"foo"`
}

type ShadowOuter struct {
	_   Namespace `xmp:"http://example.com/test/shadow/"`
	_   Prefix    `xmp:"sh"`
	Foo Text      `xmp:"foo"`
	ShadowBase
}

// TestShadowedFieldOwnFieldWins checks that when a namespace struct
// declares its own field under the same xmp name as a field promoted
// from an embedded struct, the struct's own (shallower) field wins:
// Set must use its value, and a shadowed embedded field must never be
// written to nor filled by Get.
//
// If both fields were visited independently, whichever was visited last
// would decide the outcome: ShadowBase is declared after Foo, so its
// zero value would erase the property the struct's own Foo had just
// written, and Get would fill both fields from the same property.
func TestShadowedFieldOwnFieldWins(t *testing.T) {
	in := &ShadowOuter{}
	in.Foo = NewText("outer value")
	// in.ShadowBase.Foo is left at its zero value.

	p := NewPacket()
	if err := p.Set(in); err != nil {
		t.Fatal(err)
	}

	name := xml.Name{Space: "http://example.com/test/shadow/", Local: "foo"}
	raw, ok := p.Properties[name]
	if !ok {
		t.Fatal("property foo missing from packet; the struct's own field was shadowed by the embedded field")
	}
	if got, ok := raw.(Text); !ok || got.V != "outer value" {
		t.Errorf("packet property foo = %#v, want Text{V: %q}", raw, "outer value")
	}

	var out ShadowOuter
	if err := p.Get(&out); err != nil {
		t.Fatal(err)
	}
	if out.Foo.V != "outer value" {
		t.Errorf("out.Foo.V = %q, want %q", out.Foo.V, "outer value")
	}
	if out.ShadowBase.Foo.V != "" {
		t.Errorf("out.ShadowBase.Foo.V = %q, want \"\" (a shadowed embedded field must not be filled)", out.ShadowBase.Foo.V)
	}
}

// PrefixOnlyMarker is embedded by MixedDepthMarkers below to test that
// the Prefix used for a namespace always comes from the same embedding
// depth as the winning Namespace field, never from a different depth.
type PrefixOnlyMarker struct {
	_ Prefix `xmp:"inner"`
}

type MixedDepthMarkers struct {
	_ Namespace `xmp:"http://example.com/test/mixed-depth-markers/"`
	PrefixOnlyMarker
	Foo Text `xmp:"foo"`
}

// TestMarkersNotMixedAcrossDepths checks that a [Namespace] declared
// directly on a namespace struct is never paired with a [Prefix] found
// at a different embedding depth (here, one promoted from an embedded
// struct). MixedDepthMarkers declares its own Namespace but no Prefix of
// its own, so the registered prefix must be empty, not "inner".
func TestMarkersNotMixedAcrossDepths(t *testing.T) {
	in := &MixedDepthMarkers{}
	in.Foo = NewText("val")

	p := NewPacket()
	if err := p.Set(in); err != nil {
		t.Fatal(err)
	}

	const ns = "http://example.com/test/mixed-depth-markers/"
	if got := p.nsToPrefix[ns]; got != "" {
		t.Errorf("registered prefix = %q, want \"\" (Namespace and Prefix must come from the same embedding depth)", got)
	}
}

// PtrShared is embedded by pointer in EmbeddingPointer below, to test
// that Set and Get promote fields through a non-nil anonymous pointer
// field, and that Get allocates a nil one, the way encoding/json treats
// embedded pointer-to-struct fields.
type PtrShared struct {
	Foo Text `xmp:"foo"`
}

type EmbeddingPointer struct {
	_ Namespace `xmp:"http://example.com/test/embedding-pointer/"`
	_ Prefix    `xmp:"ep"`
	*PtrShared
}

// TestEmbeddedPointerPromotedOnSet checks that Set promotes the fields of
// a non-nil anonymously embedded pointer-to-struct field, the way
// encoding/json does.
func TestEmbeddedPointerPromotedOnSet(t *testing.T) {
	in := &EmbeddingPointer{PtrShared: &PtrShared{Foo: NewText("ptr value")}}

	p := NewPacket()
	if err := p.Set(in); err != nil {
		t.Fatal(err)
	}

	name := xml.Name{Space: "http://example.com/test/embedding-pointer/", Local: "foo"}
	if _, ok := p.Properties[name]; !ok {
		t.Fatal("property foo missing from packet; the embedded pointer's fields were not promoted")
	}
}

// TestNilEmbeddedPointerSkippedOnSet checks that Set neither panics nor
// writes any property when an anonymously embedded pointer field is nil;
// it has nothing to promote, so it is simply skipped (mirroring
// encoding/json's Marshal, which omits a nil embedded pointer's fields).
func TestNilEmbeddedPointerSkippedOnSet(t *testing.T) {
	in := &EmbeddingPointer{} // PtrShared is nil

	p := NewPacket()
	if err := p.Set(in); err != nil {
		t.Fatal(err)
	}

	name := xml.Name{Space: "http://example.com/test/embedding-pointer/", Local: "foo"}
	if _, ok := p.Properties[name]; ok {
		t.Error("property foo present in packet, but the embedded pointer was nil")
	}
}

// TestEmbeddedPointerAllocatedOnGet checks that Get allocates a nil
// anonymously embedded pointer field so that its fields can be filled in,
// the way encoding/json's Unmarshal does.
func TestEmbeddedPointerAllocatedOnGet(t *testing.T) {
	p := NewPacket()
	if err := p.SetValue("http://example.com/test/embedding-pointer/", "foo", NewText("ptr value")); err != nil {
		t.Fatal(err)
	}

	var out EmbeddingPointer
	if err := p.Get(&out); err != nil {
		t.Fatal(err)
	}
	if out.PtrShared == nil {
		t.Fatal("PtrShared is nil; Get did not allocate the embedded pointer")
	}
	if out.PtrShared.Foo.V != "ptr value" {
		t.Errorf("Foo.V = %q, want %q", out.PtrShared.Foo.V, "ptr value")
	}
}

// EmbeddingValueLeaf checks that an anonymously embedded field whose type
// itself implements [Value] (here [Text]) is treated as a single leaf
// property, not decomposed into its own fields.
type EmbeddingValueLeaf struct {
	_    Namespace `xmp:"http://example.com/test/embedding-value-leaf/"`
	_    Prefix    `xmp:"vl"`
	Text `xmp:"leaf"`
}

func TestEmbeddedValueStaysLeafProperty(t *testing.T) {
	in := &EmbeddingValueLeaf{Text: NewText("leaf value")}

	out := &EmbeddingValueLeaf{}
	roundTrip(t, in, out)
}

// AnonymousMarkers embeds [Namespace] and [Prefix] anonymously (i.e.
// without the usual `_` field name) directly on the namespace struct
// itself -- a legal, previously-working shape distinct from the embedded
// SharedMarkers case above, which nests the markers inside another
// struct.
type AnonymousMarkers struct {
	Namespace `xmp:"http://example.com/test/anonymous-markers/"`
	Prefix    `xmp:"am"`
	Foo       Text `xmp:"foo"`
}

// TestAnonymousMarkerFields checks that [Namespace] and [Prefix] embedded
// anonymously (without the usual `_` field name) directly on the
// namespace struct are still recognised as markers: the promotion logic
// must treat the zero-field marker structs as leaves rather than descend
// into them, since a decomposed marker has no fields left to carry the
// xmp tag.
func TestAnonymousMarkerFields(t *testing.T) {
	in := &AnonymousMarkers{}
	in.Foo = NewText("val")

	out := &AnonymousMarkers{}
	roundTrip(t, in, out)
}

// NSOnlyMarker and PrefixOnlyMarker2 are embedded as two *separate*,
// sibling anonymous fields by SiblingMarkers below, to test that a
// [Prefix] found in one embedded struct is never paired with a
// [Namespace] found in an unrelated sibling embedded struct just because
// they happen to sit at the same embedding depth.
type NSOnlyMarker struct {
	_ Namespace `xmp:"http://example.com/test/sibling-markers/"`
}

type PrefixOnlyMarker2 struct {
	_ Prefix `xmp:"unrelated"`
}

type SiblingMarkers struct {
	NSOnlyMarker
	PrefixOnlyMarker2
	Foo Text `xmp:"foo"`
}

// TestSiblingMarkersNotPaired checks that a [Prefix] embedded in one
// struct is not paired with a [Namespace] embedded in an unrelated
// sibling struct that merely sits at the same embedding depth.
func TestSiblingMarkersNotPaired(t *testing.T) {
	in := &SiblingMarkers{}
	in.Foo = NewText("val")

	p := NewPacket()
	if err := p.Set(in); err != nil {
		t.Fatal(err)
	}

	const ns = "http://example.com/test/sibling-markers/"
	if got := p.nsToPrefix[ns]; got != "" {
		t.Errorf("registered prefix = %q, want \"\" (a Prefix from an unrelated sibling embed must not be used)", got)
	}
}

// NSOnlyMarker2 is embedded by OuterPrefixOverride below, which also
// declares its own Prefix directly, to test that an outer struct's own
// Prefix overrides one that could otherwise be inherited from something
// it embeds.
type NSOnlyMarker2 struct {
	_ Namespace `xmp:"http://example.com/test/outer-prefix-override/"`
}

type OuterPrefixOverride struct {
	_ Prefix `xmp:"outer"`
	NSOnlyMarker2
	Foo Text `xmp:"foo"`
}

// TestOuterPrefixOverridesEmbeddedNamespace checks that a Prefix declared
// directly on the enclosing struct is used even though the Namespace
// field it pairs with was promoted from something the struct embeds (so
// the Prefix sits at a shallower depth than the Namespace).
func TestOuterPrefixOverridesEmbeddedNamespace(t *testing.T) {
	in := &OuterPrefixOverride{}
	in.Foo = NewText("val")

	p := NewPacket()
	if err := p.Set(in); err != nil {
		t.Fatal(err)
	}

	const ns = "http://example.com/test/outer-prefix-override/"
	if got := p.nsToPrefix[ns]; got != "outer" {
		t.Errorf("registered prefix = %q, want %q (an outer struct's own Prefix must be used for a Namespace it encloses)", got, "outer")
	}
}

// TestSetClearsStalePropertyBehindNilPointer checks that Set clears a
// packet property whose field now lives behind a nil embedded pointer,
// honoring Set's usual "zero field clears the property" contract even
// though there is no allocated field to read IsZero from.
func TestSetClearsStalePropertyBehindNilPointer(t *testing.T) {
	p := NewPacket()
	const ns = "http://example.com/test/embedding-pointer/"
	if err := p.SetValue(ns, "foo", NewText("stale")); err != nil {
		t.Fatal(err)
	}

	in := &EmbeddingPointer{} // PtrShared is nil
	if err := p.Set(in); err != nil {
		t.Fatal(err)
	}

	name := xml.Name{Space: ns, Local: "foo"}
	if _, ok := p.Properties[name]; ok {
		t.Error("stale property foo still present in packet after Set with a nil embedded pointer")
	}
}

// TestNilEmbeddedPointerStaysNilAfterRoundTrip checks that Get does not
// allocate an embedded pointer field when the packet has no property for
// anything in its subtree, so a nil embedded pointer survives a
// Set/Get round trip as nil rather than becoming a non-nil pointer to a
// zero-valued struct.
func TestNilEmbeddedPointerStaysNilAfterRoundTrip(t *testing.T) {
	in := &EmbeddingPointer{} // PtrShared is nil
	out := &EmbeddingPointer{}
	roundTrip(t, in, out)
}

// TestGetDoesNotAllocateEmbeddedPointerOnDecodeError checks that a
// malformed property does not cause Get to allocate the embedded pointer
// it would have decoded into: on an error path, dst must not be mutated.
func TestGetDoesNotAllocateEmbeddedPointerOnDecodeError(t *testing.T) {
	p := NewPacket()
	name := xml.Name{Space: "http://example.com/test/embedding-pointer/", Local: "foo"}
	p.Properties[name] = URL{} // not decodable as Text

	var out EmbeddingPointer
	if err := p.Get(&out); err == nil {
		t.Fatal("Get: expected a decode error, got nil")
	}
	if out.PtrShared != nil {
		t.Error("PtrShared was allocated even though nothing was decoded into it")
	}
}

// AmbiguousPropertyA and AmbiguousPropertyB are embedded as two sibling
// anonymous structs by AmbiguousPropertyOuter below, each declaring a
// field tagged with the same xmp name ("dup") at the same embedding
// depth, to test that Set and Get report this ambiguity explicitly
// rather than silently dropping both fields.
type AmbiguousPropertyA struct {
	Foo Text `xmp:"dup"`
}

type AmbiguousPropertyB struct {
	Bar Text `xmp:"dup"`
}

type AmbiguousPropertyOuter struct {
	_ Namespace `xmp:"http://example.com/test/ambiguous-property/"`
	AmbiguousPropertyA
	AmbiguousPropertyB
}

// TestSetErrorsOnAmbiguousProperty checks that Set returns an error, per
// the library's existing error-return convention, when two promoted
// fields tie for the same effective xmp name at the same embedding
// depth, rather than dropping the tie in silence.
func TestSetErrorsOnAmbiguousProperty(t *testing.T) {
	in := &AmbiguousPropertyOuter{}
	p := NewPacket()
	if err := p.Set(in); err == nil {
		t.Fatal("Set: expected an error for an ambiguous promoted property, got nil")
	}
}

// TestGetPanicsOnAmbiguousProperty checks that Get panics, per the
// library's existing panic convention for structurally malformed
// namespace structs (see the "not an XMP namespace struct" panic), when
// two promoted fields tie for the same effective xmp name at the same
// embedding depth.
func TestGetPanicsOnAmbiguousProperty(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("Get: expected a panic for an ambiguous promoted property")
		}
	}()
	p := NewPacket()
	var out AmbiguousPropertyOuter
	_ = p.Get(&out)
}

// DuplicateTagFields declares two ordinary (non-embedded) fields under
// the same effective xmp name in a single struct, to pin that the
// ambiguity rule applies to any two fields tying at the same embedding
// depth, not only to promoted fields.
type DuplicateTagFields struct {
	_   Namespace `xmp:"http://example.com/test/duplicate-tag/"`
	Foo Text      `xmp:"same"`
	Bar Text      `xmp:"same"`
}

func TestSetErrorsOnDuplicateTagInOneStruct(t *testing.T) {
	in := &DuplicateTagFields{}
	p := NewPacket()
	if err := p.Set(in); err == nil {
		t.Fatal("Set: expected an error for two fields with the same xmp tag, got nil")
	}
}

func TestGetPanicsOnDuplicateTagInOneStruct(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("Get: expected a panic for two fields with the same xmp tag")
		}
	}()
	p := NewPacket()
	var out DuplicateTagFields
	_ = p.Get(&out)
}

// UntaggedNSBase carries a real, tagged Namespace; UntaggedNSOuter embeds
// it but also declares a stray Namespace field without an xmp tag at a
// shallower depth, which shadows the real one under the
// shallowest-depth-wins rule.
type UntaggedNSBase struct {
	_   Namespace `xmp:"http://example.com/test/untagged-ns/"`
	Foo Text      `xmp:"foo"`
}

type UntaggedNSOuter struct {
	_ Namespace // no xmp tag
	UntaggedNSBase
}

// TestUntaggedNamespaceMarkerRejected pins that a Namespace field without
// an xmp tag is reported as a defect rather than silently resolving to
// no namespace or to a Namespace found deeper in the embedding tree.
func TestUntaggedNamespaceMarkerRejected(t *testing.T) {
	in := &UntaggedNSOuter{}
	p := NewPacket()
	if err := p.Set(in); err == nil {
		t.Fatal("Set: expected an error for an untagged Namespace field, got nil")
	}
}

// AmbiguousNSA and AmbiguousNSB are embedded as two sibling anonymous
// structs by AmbiguousMarkersOuter below, each declaring their own
// Namespace field, to test that Set and Get report the resulting
// ambiguity explicitly instead of silently failing with a misleading
// "no namespace" message.
type AmbiguousNSA struct {
	_ Namespace `xmp:"http://example.com/test/ambiguous-a/"`
}

type AmbiguousNSB struct {
	_ Namespace `xmp:"http://example.com/test/ambiguous-b/"`
}

type AmbiguousMarkersOuter struct {
	AmbiguousNSA
	AmbiguousNSB
	Foo Text `xmp:"foo"`
}

func TestSetErrorsOnAmbiguousNamespace(t *testing.T) {
	in := &AmbiguousMarkersOuter{}
	p := NewPacket()
	if err := p.Set(in); err == nil {
		t.Fatal("Set: expected an error for an ambiguous Namespace field, got nil")
	}
}

func TestGetPanicsOnAmbiguousNamespace(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("Get: expected a panic for an ambiguous Namespace field")
		}
	}()
	p := NewPacket()
	var out AmbiguousMarkersOuter
	_ = p.Get(&out)
}

// SelfRefEmbed embeds a pointer to its own type anonymously, which is
// legal Go (unlike value embedding, which the compiler rejects as an
// infinitely-sized type) but structurally cyclic.
type SelfRefEmbed struct {
	_ Namespace `xmp:"http://example.com/test/self-ref/"`
	*SelfRefEmbed
	Foo Text `xmp:"foo"`
}

// TestGetPanicsCleanlyOnCyclicEmbeddedPointer checks that Get reports a
// recoverable panic for a self-referentially embedded pointer type,
// instead of recursing forever: each freshly allocated instance's own
// embedded pointer is nil again, so a field walk that allocated and
// descended would never terminate, crashing the process with a stack
// overflow (a runtime fatal error, not a recoverable panic).
func TestGetPanicsCleanlyOnCyclicEmbeddedPointer(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("Get: expected a panic for a cyclic embedded pointer type")
		}
	}()
	p := NewPacket()
	var out SelfRefEmbed
	_ = p.Get(&out)
}

// TestSetErrorsOnCyclicEmbeddedPointer checks that Set reports a
// self-referentially embedded pointer type as an error, per Set's
// contract that arguments of the wrong shape are reported as errors
// rather than panics.
func TestSetErrorsOnCyclicEmbeddedPointer(t *testing.T) {
	p := NewPacket()
	in := &SelfRefEmbed{}
	if err := p.Set(in); err == nil {
		t.Fatal("Set: expected an error for a cyclic embedded pointer type, got nil")
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
