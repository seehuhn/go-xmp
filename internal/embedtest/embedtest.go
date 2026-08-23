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

// Package embedtest provides namespace structs whose embedded fields are
// unexported *in a different package than [seehuhn.de/go/xmp]*, so that
// tests of xmp.Packet.Set/Get can exercise cross-package field-exportedness
// rules that cannot be reproduced from inside the xmp package's own tests
// (there, an "unexported" type is still in the same package as the
// reflecting code, which is not the scenario being tested).
package embedtest

import "seehuhn.de/go/xmp"

// sharedValue is an unexported struct, embedded anonymously (by value) by
// ValueOuter below, to pin that [xmp.Packet.Set] and [xmp.Packet.Get] can
// read and write its exported leaf field even though the embedding struct
// type itself is unexported and declared outside the xmp package.
type sharedValue struct {
	Foo xmp.Text `xmp:"foo"`
}

// ValueOuter embeds sharedValue anonymously by value.
type ValueOuter struct {
	_ xmp.Namespace `xmp:"http://example.com/test/embedtest-unexported-value/"`
	_ xmp.Prefix    `xmp:"uv"`
	sharedValue
}

// sharedPointer is an unexported struct, embedded anonymously *by pointer*
// by PointerOuter below, to test that [xmp.Packet.Get] reports a clear
// error rather than silently losing data when it cannot allocate a nil
// embedded pointer field because the field itself -- not just its
// subfields -- is unexported relative to the xmp package.
type sharedPointer struct {
	Foo xmp.Text `xmp:"foo"`
}

// PointerOuter embeds sharedPointer anonymously by pointer.
type PointerOuter struct {
	_ xmp.Namespace `xmp:"http://example.com/test/embedtest-unexported-pointer/"`
	_ xmp.Prefix    `xmp:"up"`
	*sharedPointer
}

// Mid embeds sharedPointer anonymously by pointer and is itself embedded
// by pointer in NestedOuter below, so that reaching Foo from a zero
// NestedOuter requires two levels of pointer allocation, of which the
// inner one must fail (sharedPointer is unexported relative to xmp).
type Mid struct {
	*sharedPointer
}

// NestedOuter embeds Mid anonymously by pointer.
type NestedOuter struct {
	_ xmp.Namespace `xmp:"http://example.com/test/embedtest-nested-pointer/"`
	_ xmp.Prefix    `xmp:"np"`
	*Mid
}
