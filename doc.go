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

// Package xmp reads and writes Extensible Metadata Platform (XMP) data.
//
// # XMP Packets
//
// The main type in this package is the [Packet] type, which represents an XMP
// packet.  XMP packets can be read from file using the [Read] function and
// written to file using the [Packet.Write] method.
//
// # Properties
//
// An XMP packet stores a set of properties.  Each property is identified by a
// namespace and a name.  The value of a property has type which implements
// [Value], the specific type depends on the property namespace and name.  Use
// [GetValue] to read a property from an XMP packet and [Packet.Set] to set a
// property in an XMP packet.
//
// The package provides the following types for XMP values:
//
//   - [Text] represents a generic text string.
//   - [AgentName] represents the name of some document creator software.
//   - [AlternativeArray] is an ordered array of values.
//   - [Date] represents a date and time.
//   - [GUID] represents a globally unique identifier.
//   - [Locale] represents a language code.
//   - [Localized] represents a localized text value
//   - [MimeType] represents the media type of a file.
//   - [OptionalBool] represents a value which can be true, false or unset.
//   - [OrderedArray] is an ordered array of values.
//   - [ProperName] represents a proper name.
//   - [Real] represents a floating-point number.
//   - [RenditionClass] states the form or intended usage of a resource
//     (e.g. "draft" or "low-res").
//   - [ResourceRef] represents a reference to an external resource.
//   - [URL] is a URL or URI.
//   - [UnorderedArray] is an unordered array of values.
//
// Additional types can be defined by implementing the [Value] interface.
//
// Every XMP value can be annotated with a list of qualifiers, for example to
// specify the language of a text value.  Qualifiers are identified by a
// namespace and a name.  The value of a qualifier is again a [Value].
//
// # Models
//
// Models can be used get or set several properties from a namespace at once.
// Use [Packet.Get] to read values from an XMP packet into a model, and
// [Packet.Set] to store values from a model into an XMP packet. The following
// models are defined in this library:
//
//   - [DublinCore] represents the Dublin Core namespace.
//   - [MediaManagement] represents the XMP Media Management namespace.
//   - [RightsManagement] represents the XMP RightsManagement Management namespace.
//   - [Basic] represents the XMP basic namespace.
//
// Additional models can be defined by defining a struct with fields of type
// [Value] and using the Go struct tags to specify the XMP property name where
// this is different from the field name.  See [DublinCore], [Namespace] and
// [Prefix] for examples.
package xmp
