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

// This file uses an external (black-box) test package, rather than the
// internal "xmp" test package used by struct_test.go, because the two
// tests below need a struct whose embedded field is genuinely unexported
// *relative to the xmp package's own reflection code*. A field embedded
// inside struct_test.go (package xmp) would still be in the very package
// doing the reflecting, so it would not reproduce the cross-package
// exportedness rules being tested; embedtest.ValueOuter and
// embedtest.PointerOuter are declared in a separate package for exactly
// this reason.
package xmp_test

import (
	"errors"
	"testing"

	"seehuhn.de/go/xmp"
	"seehuhn.de/go/xmp/internal/embedtest"
)

// TestUnexportedEmbeddedValueFieldsWork pins that [xmp.Packet.Set] and
// [xmp.Packet.Get] can read and write the exported leaf field of an
// anonymously embedded struct even when the embedded struct's own type is
// unexported and declared in a different package than xmp (reflect
// clears flagEmbedRO for exported subfields reached through an
// unexported anonymous field).
func TestUnexportedEmbeddedValueFieldsWork(t *testing.T) {
	in := &embedtest.ValueOuter{}
	in.Foo = xmp.NewText("val")

	p := xmp.NewPacket()
	if err := p.Set(in); err != nil {
		t.Fatalf("Set: %v", err)
	}

	var out embedtest.ValueOuter
	if err := p.Get(&out); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if out.Foo.V != "val" {
		t.Errorf("Foo.V = %q, want %q", out.Foo.V, "val")
	}
}

// TestGetErrorsOnUnexportedNilEmbeddedPointer checks that Get reports an
// error, rather than silently discarding the property, when it needs to
// allocate a nil anonymously embedded pointer field but cannot: the field
// itself (not merely its subfields) is unexported relative to the xmp
// package, so reflect refuses to set it. encoding/json's Unmarshal errors
// in the same situation ("json: cannot set embedded pointer to unexported
// struct type ...").
func TestGetErrorsOnUnexportedNilEmbeddedPointer(t *testing.T) {
	p := xmp.NewPacket()
	err := p.SetValue("http://example.com/test/embedtest-unexported-pointer/", "foo", xmp.NewText("val"))
	if err != nil {
		t.Fatalf("SetValue: %v", err)
	}

	var out embedtest.PointerOuter
	err = p.Get(&out)
	if err == nil {
		t.Fatal("Get: expected an error (data behind an unexported nil embedded pointer was silently lost), got nil")
	}
	if errors.Is(err, xmp.ErrInvalid) {
		t.Error("Get: a struct-shape defect must not be classified as invalid XMP file data (ErrInvalid)")
	}
}

// TestGetLeavesOuterPointerNilWhenInnerAllocFails checks that when Get
// cannot decode a property because an unexported nil embedded pointer
// blocks the path to the target field, it does not leave a partially
// allocated pointer chain behind: the outer, exported pointer must stay
// nil, so that "pointer != nil" remains a reliable was-anything-decoded
// signal.
func TestGetLeavesOuterPointerNilWhenInnerAllocFails(t *testing.T) {
	p := xmp.NewPacket()
	err := p.SetValue("http://example.com/test/embedtest-nested-pointer/", "foo", xmp.NewText("val"))
	if err != nil {
		t.Fatalf("SetValue: %v", err)
	}

	var out embedtest.NestedOuter
	if err := p.Get(&out); err == nil {
		t.Fatal("Get: expected an error, got nil")
	}
	if out.Mid != nil {
		t.Error("Mid was allocated even though nothing was decoded into it")
	}
}
