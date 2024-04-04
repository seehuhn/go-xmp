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
	"bytes"
	"encoding/xml"
)

type Value interface {
	EncodeValue(e *Encoder) error
	Qualifiers() []Qualifier
}

type Qualifier struct {
	Name  xml.Name
	Value Value
}

type Q []Qualifier

func (q Q) Qualifiers() []Qualifier {
	return q
}

type Model interface {
	EncodeProperties(e *Encoder, prefix string) error
}

type Encoder struct {
	buf *bytes.Buffer
	*xml.Encoder
}

func NewEncoder() *Encoder {
	buf := &bytes.Buffer{}
	enc := xml.NewEncoder(buf)
	return &Encoder{
		buf:     buf,
		Encoder: enc,
	}
}
