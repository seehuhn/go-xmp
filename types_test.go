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
	"math"
	"testing"

	"golang.org/x/text/language"

	"github.com/google/go-cmp/cmp"
)

func TestText(t *testing.T) {
	p := NewPacket()

	A := Text{
		V: "hello world",
		Q: Q{Language(language.English)},
	}
	p.SetValue("http://ns.seehuhn.de/test/#", "prop", A)

	B, err := PacketGetValue[Text](p, "http://ns.seehuhn.de/test/#", "prop")
	if err != nil {
		t.Fatalf("p.Get: %v", err)
	}

	if d := cmp.Diff(A, B); d != "" {
		t.Errorf("A and B are different (-want +got):\n%s", d)
	}
}

// TestSetValueInvalidName checks that [Packet.SetValue] returns an error
// wrapping [ErrInvalidName] for property identifiers that XMP rejects.
func TestSetValueInvalidName(t *testing.T) {
	p := NewPacket()
	cases := []struct{ ns, name string }{
		{"", "title"},                    // empty namespace
		{"http://ns.example/", "ti tle"}, // invalid local name
		{"http://ns.example/", "1bad"},   // local name starts with digit
	}
	for _, tc := range cases {
		err := p.SetValue(tc.ns, tc.name, NewText("v"))
		if err == nil {
			t.Errorf("SetValue(%q, %q): expected error, got nil", tc.ns, tc.name)
			continue
		}
		if !errors.Is(err, ErrInvalidName) {
			t.Errorf("SetValue(%q, %q): error %v does not wrap ErrInvalidName", tc.ns, tc.name, err)
		}
	}
}

func TestInteger(t *testing.T) {
	p := NewPacket()

	A := NewInteger(2020)
	p.SetValue("http://ns.seehuhn.de/test/#", "rev", A)

	B, err := PacketGetValue[Integer](p, "http://ns.seehuhn.de/test/#", "rev")
	if err != nil {
		t.Fatalf("p.Get: %v", err)
	}
	if d := cmp.Diff(A, B); d != "" {
		t.Errorf("A and B are different (-want +got):\n%s", d)
	}

	// negative values
	C := NewInteger(-7)
	p.SetValue("http://ns.seehuhn.de/test/#", "neg", C)
	D, err := PacketGetValue[Integer](p, "http://ns.seehuhn.de/test/#", "neg")
	if err != nil {
		t.Fatalf("p.Get: %v", err)
	}
	if d := cmp.Diff(C, D); d != "" {
		t.Errorf("C and D are different (-want +got):\n%s", d)
	}
}

func TestLocalizedBest(t *testing.T) {
	loc := Localized{
		V: map[language.Tag]Text{
			language.German:  NewText("Deutscher Titel"),
			language.English: NewText("English Title"),
		},
		Default: NewText("Default Title"),
	}

	tests := []struct {
		name string
		lang language.Tag
		want string
	}{
		{"exact German", language.German, "Deutscher Titel"},
		{"exact English", language.English, "English Title"},
		{"American English matches English", language.AmericanEnglish, "English Title"},
		{"unknown language falls back to default", language.Japanese, "Default Title"},
		{"undefined falls back to default", language.Und, "Default Title"},
		{"x-default tag returns default", language.MustParse("x-default"), "Default Title"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := loc.Best(tc.lang); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}

	// no default: a reasonable match still wins, but an unrelated
	// language returns "" rather than picking an arbitrary entry.
	noDefault := Localized{
		V: map[language.Tag]Text{
			language.German:  NewText("Nur Deutsch"),
			language.Italian: NewText("Solo Italiano"),
		},
	}
	if got := noDefault.Best(language.German); got != "Nur Deutsch" {
		t.Errorf("matched language: got %q", got)
	}
	if got := noDefault.Best(language.MustParse("de-AT")); got != "Nur Deutsch" {
		t.Errorf("regional variant should match: got %q", got)
	}
	if got := noDefault.Best(language.Japanese); got != "" {
		t.Errorf("no reasonable match, no default: got %q, want empty", got)
	}

	// completely empty
	empty := Localized{}
	if got := empty.Best(language.English); got != "" {
		t.Errorf("empty: got %q, want empty string", got)
	}

	// only default
	onlyDefault := Localized{Default: NewText("Just Default")}
	if got := onlyDefault.Best(language.English); got != "Just Default" {
		t.Errorf("only default: got %q", got)
	}
}

// TestLocalizedSet checks that Set routes both the parsed "x-default"
// tag and [language.Und] to [Localized.Default], and that other
// languages land in V.
func TestLocalizedSet(t *testing.T) {
	for _, tc := range []struct {
		name string
		lang language.Tag
	}{
		{"x-default", language.MustParse("x-default")},
		{"language.Und", language.Und},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var loc Localized
			loc.Set(tc.lang, "Hello")
			loc.Set(language.German, "Hallo")

			if loc.Default.V != "Hello" {
				t.Errorf("Default: got %q, want %q", loc.Default.V, "Hello")
			}
			if _, ok := loc.V[language.MustParse("x-default")]; ok {
				t.Errorf("V should not contain an x-default key")
			}
			if _, ok := loc.V[language.Und]; ok {
				t.Errorf("V should not contain a language.Und key")
			}
			if loc.V[language.German].V != "Hallo" {
				t.Errorf("V[de]: got %q, want %q", loc.V[language.German].V, "Hallo")
			}
		})
	}
}

// TestRealEncodeRejectsNonFinite checks that NaN and infinities
// cannot be encoded as XMP, since the spec only defines a textual
// form for finite floats.
func TestRealEncodeRejectsNonFinite(t *testing.T) {
	cases := []float64{math.NaN(), math.Inf(1), math.Inf(-1)}
	for _, v := range cases {
		_, err := (Real{V: v}).EncodeXMP(nil)
		if err == nil {
			t.Errorf("EncodeXMP(%v): expected error, got nil", v)
			continue
		}
		if !errors.Is(err, ErrInvalid) {
			t.Errorf("EncodeXMP(%v): error %v does not wrap ErrInvalid", v, err)
		}
	}
}

// TestMimeTypeEncodeRejectsInvalid checks that a MimeType whose V
// cannot be serialized via [mime.FormatMediaType] is rejected at
// encode time.
func TestMimeTypeEncodeRejectsInvalid(t *testing.T) {
	// "image/" is not a well-formed media type
	_, err := MimeType{V: "image/"}.EncodeXMP(nil)
	if err == nil {
		t.Fatal("EncodeXMP: expected error, got nil")
	}
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("error %v does not wrap ErrInvalid", err)
	}
}

// TestSetValueInvalidEncodes checks that [Packet.SetValue]
// propagates encode-time validation errors instead of storing
// invalid data.
func TestSetValueInvalidEncodes(t *testing.T) {
	p := NewPacket()
	err := p.SetValue("http://ns.example/", "p", Real{V: math.NaN()})
	if err == nil {
		t.Fatal("SetValue with NaN: expected error, got nil")
	}
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("error %v does not wrap ErrInvalid", err)
	}
	if _, ok := p.Properties[xml.Name{Space: "http://ns.example/", Local: "p"}]; ok {
		t.Error("SetValue stored a property despite encode error")
	}
}

// TestLocalizedEncodeRejectsXDefaultInV checks that the encoder
// refuses to encode a Localized whose V map contains either the
// "x-default" tag or [language.Und] (both denote the default item and
// belong in [Localized.Default]), rather than silently fixing it up.
func TestLocalizedEncodeRejectsXDefaultInV(t *testing.T) {
	for _, tc := range []struct {
		name string
		key  language.Tag
	}{
		{"x-default", defaultLanguage},
		{"language.Und", language.Und},
	} {
		t.Run(tc.name, func(t *testing.T) {
			loc := Localized{
				V: map[language.Tag]Text{
					tc.key:          NewText("Stray"),
					language.German: NewText("Hallo"),
				},
			}
			_, err := loc.EncodeXMP(nil)
			if err == nil {
				t.Fatal("EncodeXMP: expected error, got nil")
			}
			if !errors.Is(err, ErrInvalid) {
				t.Errorf("error %v does not wrap ErrInvalid", err)
			}
		})
	}
}

// TestLocalizedDecodeRoutesXDefault checks that decoding a Lang Alt
// array routes the x-default entry into [Localized.Default] rather
// than leaving it as a key in [Localized.V], so the documented
// invariant on V holds for values returned by Decode.
func TestLocalizedDecodeRoutesXDefault(t *testing.T) {
	raw := RawArray{
		Kind: Alternative,
		Value: []Raw{
			Text{V: "Hello", Q: Q{{Name: nameXMLLang, Value: Text{V: "x-default"}}}},
			Text{V: "Hallo", Q: Q{{Name: nameXMLLang, Value: Text{V: "de"}}}},
		},
	}
	v, err := Localized{}.DecodeAnother(raw)
	if err != nil {
		t.Fatalf("DecodeAnother: %v", err)
	}
	loc := v.(Localized)

	if loc.Default.V != "Hello" {
		t.Errorf("Default: got %q, want %q", loc.Default.V, "Hello")
	}
	if _, ok := loc.V[defaultLanguage]; ok {
		t.Errorf("V must not contain an x-default key after decode")
	}
	if loc.V[language.German].V != "Hallo" {
		t.Errorf("V[de]: got %q, want %q", loc.V[language.German].V, "Hallo")
	}
}

// TestLocalizedDecodeRoutesUnd checks that decoding a Lang Alt array
// also routes a `xml:lang="und"` entry into [Localized.Default]: the
// library treats [language.Und] as a synonym for x-default everywhere.
func TestLocalizedDecodeRoutesUnd(t *testing.T) {
	raw := RawArray{
		Kind: Alternative,
		Value: []Raw{
			Text{V: "Hello", Q: Q{{Name: nameXMLLang, Value: Text{V: "und"}}}},
			Text{V: "Hallo", Q: Q{{Name: nameXMLLang, Value: Text{V: "de"}}}},
		},
	}
	v, err := Localized{}.DecodeAnother(raw)
	if err != nil {
		t.Fatalf("DecodeAnother: %v", err)
	}
	loc := v.(Localized)

	if loc.Default.V != "Hello" {
		t.Errorf("Default: got %q, want %q", loc.Default.V, "Hello")
	}
	if _, ok := loc.V[language.Und]; ok {
		t.Errorf("V must not contain a language.Und key after decode")
	}
	if loc.V[language.German].V != "Hallo" {
		t.Errorf("V[de]: got %q, want %q", loc.V[language.German].V, "Hallo")
	}
}

func TestUnorderedArray(t *testing.T) {
	p := NewPacket()

	A := UnorderedArray[Text]{
		V: []Text{
			{V: "Hello", Q: Q{Language(language.English)}},
			{V: "Hallo", Q: Q{Language(language.German)}},
			{V: "Bonjour", Q: Q{Language(language.French)}},
		},
	}
	p.SetValue("http://ns.seehuhn.de/test/#", "prop", A)

	B, err := PacketGetValue[UnorderedArray[Text]](p, "http://ns.seehuhn.de/test/#", "prop")
	if err != nil {
		t.Fatalf("p.Get: %v", err)
	}

	if d := cmp.Diff(A, B); d != "" {
		t.Errorf("A and B are different (-want +got):\n%s", d)
	}
}
