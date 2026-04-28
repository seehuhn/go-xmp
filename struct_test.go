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
	p.Get(&dc2)

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

// roundTrip stores in into a fresh packet via [Packet.Set], reads it back
// into out via [Packet.Get], and reports any differences.
func roundTrip(t *testing.T, in, out any) {
	t.Helper()
	p := NewPacket()
	if err := p.Set(in); err != nil {
		t.Fatalf("Set: %v", err)
	}
	p.Get(out)
	if d := cmp.Diff(in, out, cmpopts.EquateComparable(language.Tag{})); d != "" {
		t.Errorf("round trip failed (-want +got):\n%s", d)
	}
}
