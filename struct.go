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
	"fmt"
	"reflect"
	"slices"
)

// DublinCore represents the properties in the Dublin Core namespace.
//
// See section 8.4 of ISO 16684-1:2011.
type DublinCore struct {
	_ Namespace `xmp:"http://purl.org/dc/elements/1.1/"`
	_ Prefix    `xmp:"dc"`

	// Contributor is a list of contributors to the resource.
	// This should not include names listed in the Creator field.
	Contributor UnorderedArray[ProperName] `xmp:"contributor"`

	// Coverage is the extent or scope of the resource.
	Coverage Text `xmp:"coverage"`

	// Creator is a list of the creators of the resource.  Entities should be
	// listed in order of decreasing precedence, if such order is significant.
	Creator OrderedArray[ProperName] `xmp:"creator"`

	// Date is a point or period of time associated with an event in the life
	// cycle of the resource.
	Date OrderedArray[Date] `xmp:"date"`

	// Description is a textual description of the content of the resource.
	Description Localized `xmp:"description"`

	// Format is the media type of the resource.
	Format MimeType `xmp:"format"`

	// Identifier is an unambiguous reference for the resource.
	Identifier Text `xmp:"identifier"`

	// Language is a list of languages used in the content of the resource.
	Language UnorderedArray[Locale] `xmp:"language"`

	// Publisher is a list of publishers of the resource.
	Publisher UnorderedArray[ProperName] `xmp:"publisher"`

	// Relation is a list of related resources.
	Relation UnorderedArray[Text] `xmp:"relation"`

	// Rights is an informal rights statement for the resource.
	Rights Localized `xmp:"rights"`

	// Source is a reference to a resource from which the present resource is
	// derived, either in whole or in part.
	Source Text `xmp:"source"`

	// Subject is a list of descriptive phrases or keywords that specify the
	// content of the resource.
	Subject UnorderedArray[Text] `xmp:"subject"`

	// Title is the title or name of the resource.
	Title Localized `xmp:"title"`

	// Type is the nature or genre of the resource.
	Type UnorderedArray[Text] `xmp:"type"`
}

// Basic represents the XMP basic namespace.
//
// See section 8.4 of ISO 16684-1:2011 for details.
type Basic struct {
	_ Namespace `xmp:"http://ns.adobe.com/xap/1.0/"`
	_ Prefix    `xmp:"xmp"`

	// CreateDate is the date and time the resource was originally created.
	CreateDate Date

	// CreatorTool is the name of the first known tool used to create the
	// resource.
	CreatorTool AgentName

	// Identifier is an unambiguous reference to the resource within a given
	// context.  An array item may be qualified with xmpidq:Scheme to specify
	// the identification system for that item.
	Identifier UnorderedArray[Text]

	// Label is a word or short phrase that identifies a resource within a
	// local context.
	Label Text

	// MetadataDate is the date and time that any metadata for this resource was
	// last modified.
	MetadataDate Date

	// ModifyDate is the date and time the resource was last modified.
	ModifyDate Date

	// Rating is a user-assigned rating for this resource.
	//
	// The value must be -1 (rejected), 0 (unrated) or a rating in the range
	// (0, 5].
	Rating Real
}

// RightsManagement represents the XMP Rights Management namespace.
//
// See section 8.5 of ISO 16684-1:2011 for details.
type RightsManagement struct {
	_ Namespace `xmp:"http://ns.adobe.com/xap/1.0/rights/"`
	_ Prefix    `xmp:"xmpRights"`

	// Certificate is a reference to a digital certificate that can be used to
	// verify the rights management information.
	//
	// For historical reasons, this field has type Text instead of URL.
	Certificate Text

	// Marked is true if the document has been marked as copyrighted.
	Marked OptionalBool

	// Owner is a list of legal owners of the resource.
	Owner UnorderedArray[ProperName]

	// UsageTerms is a statement that specifies the terms and conditions under
	// which the document can be used.
	UsageTerms Localized

	// WebStatement is a URL that can be used to access a rights management
	// information statement.
	//
	// For historical reasons, this field has type Text instead of URL.
	WebStatement Text
}

// PDF represents the Adobe PDF namespace, which carries PDF-specific
// metadata that does not have a natural home in the core XMP schemas.
//
// See https://developer.adobe.com/xmp/docs/XMPNamespaces/pdf/ for details.
type PDF struct {
	_ Namespace `xmp:"http://ns.adobe.com/pdf/1.3/"`
	_ Prefix    `xmp:"pdf"`

	// Keywords is a list of keywords describing the document, as a single
	// free-form string.  XMP does not specify how the individual keywords
	// are separated.
	Keywords Text

	// PDFVersion is the PDF file version (e.g. "1.7", "2.0").
	PDFVersion Text

	// Producer is the name of the tool that produced the PDF document.
	Producer AgentName

	// Trapped indicates whether the document has been trapped.
	// Allowed values are "True", "False", and "Unknown".
	Trapped Text
}

// PDFAID represents the PDF/A identification namespace, which carries
// the markers a PDF/A reader uses to recognise the conformance level
// of a file.
//
// See ISO 19005 (parts 1, 2, 3, 4) and PDF Association TechNote 0008.
type PDFAID struct {
	_ Namespace `xmp:"http://www.aiim.org/pdfa/ns/id/"`
	_ Prefix    `xmp:"pdfaid"`

	// Part is the part of the PDF/A standard (1, 2, 3, or 4).  Required
	// in every PDF/A file.
	Part Integer `xmp:"part"`

	// Conformance is the conformance level of the PDF/A standard.  In
	// PDF/A-1, 2, and 3 it is one of "A", "B", or "U" and required.  In
	// PDF/A-4 it is empty, "F", or "E" and optional.
	Conformance Text `xmp:"conformance"`

	// Amd is the optional amendment identifier, formatted as
	// "<amendment-number>:<year>" (e.g. "1:2025").
	Amd Text `xmp:"amd"`

	// Rev is the dated revision of the PDF/A-4 standard, given as the
	// 4-digit year of publication (e.g. 2020).  Required for PDF/A-4;
	// absent in earlier parts.
	Rev Integer `xmp:"rev"`
}

// PDFX represents the legacy Adobe PDF/X identification namespace,
// used by PDF/X-1a, PDF/X-2, and PDF/X-3.  PDF/X-4 and later use
// [PDFXID] instead.
//
// See ISO 15930, parts 1, 4, 5, and 6.
type PDFX struct {
	_ Namespace `xmp:"http://ns.adobe.com/pdfx/1.3/"`
	_ Prefix    `xmp:"pdfx"`

	// Version is the ID of the PDF/X standard, e.g. "PDF/X-1:2001"
	// or "PDF/X-3:2002".
	Version Text `xmp:"GTS_PDFXVersion"`

	// Conformance is the conformance level of the PDF/X standard,
	// e.g. "PDF/X-1a:2001".
	Conformance Text `xmp:"GTS_PDFXConformance"`
}

// PDFXID represents the PDF/X identification namespace introduced in
// PDF/X-4 and used by PDF/X-4 and PDF/X-5.
//
// See ISO 15930-7 (PDF/X-4) and ISO 15930-8 (PDF/X-5).
type PDFXID struct {
	_ Namespace `xmp:"http://www.npes.org/pdfx/ns/id/"`
	_ Prefix    `xmp:"pdfxid"`

	// Version is the ID of the PDF/X standard, e.g. "PDF/X-4".
	Version Text `xmp:"GTS_PDFXVersion"`
}

// MediaManagement represents the XMP Media Management namespace.
//
// See section 8.6 of ISO 16684-1:2011 for details.
type MediaManagement struct {
	_ Namespace `xmp:"http://ns.adobe.com/xap/1.0/mm/"`
	_ Prefix    `xmp:"xmpMM"`

	// DerivedFrom is a reference to a resource from which the present resource
	// is derived, either in whole or in part.  Missing fields are assumed to
	// be unchanged from the source.
	DerivedFrom ResourceRef

	// DocumentID is a unique identifier for the document.
	DocumentID Text

	// InstanceID is a unique identifier for the document instance.
	InstanceID Text

	// OriginalDocumentID is a unique identifier for the original document.
	OriginalDocumentID Text

	// RenditionClass is a rendition class name for this resource.
	RenditionClass RenditionClass

	// RenditionParams can be used to provide additional rendition parameters
	RenditionParams Text
}

// Set stores the property values from one or more XMP namespace structs in
// the packet.  Predefined namespace structs include [DublinCore], [Basic],
// [RightsManagement], [PDF], [PDFAID], [PDFX], [PDFXID], and
// [MediaManagement]; see [Namespace] for how to define your own.
//
// Each argument must be a (pointer to a) namespace struct.  Set returns
// an error if an argument has the wrong shape, e.g. is not a struct,
// lacks a [Namespace] field, has two fields ambiguously tied under the
// same xmp name (see [Namespace]), or embeds a pointer to its own type;
// it also propagates the errors from [Packet.SetValue] when a field's
// value cannot be encoded as a valid XMP property (wrapping [ErrInvalid]
// or [ErrInvalidName]).
func (p *Packet) Set(models ...any) error {
	for _, v := range models {
		if err := p.setOne(v); err != nil {
			return err
		}
	}
	return nil
}

func (p *Packet) setOne(v any) error {
	s := reflect.Indirect(reflect.ValueOf(v))
	if s.Kind() != reflect.Struct {
		return errors.New("no struct found")
	}

	fields, groupParent, err := collectFields(s.Type())
	if err != nil {
		return err
	}

	namespace, prefix, err := resolveMarkers(fields, groupParent)
	if err != nil {
		return err
	}
	if namespace == "" {
		return errors.New("XMP namespace not specified")
	}

	props, err := resolveProperties(fields)
	if err != nil {
		return err
	}

	p.RegisterPrefix(namespace, prefix)

	for _, f := range props {
		fVal, ok := fieldByIndex(s, f.index)
		if !ok {
			// The field lives behind a nil embedded pointer.  Set never
			// allocates (it only reads values), so there is no value to
			// read IsZero from; a nil embedded pointer is itself the zero
			// value for its whole subtree, so every property it would
			// have promoted is cleared, the same as an ordinary zero
			// field would be.
			p.ClearValue(namespace, f.propName)
			continue
		}

		val := fVal.Interface().(Value)
		if !val.IsZero() {
			if err := p.SetValue(namespace, f.propName, val); err != nil {
				return err
			}
		} else {
			p.ClearValue(namespace, f.propName)
		}
	}

	return nil
}

// Get fills the fields of an XMP namespace struct with property values from
// the packet.  Predefined namespace structs include [DublinCore], [Basic],
// [RightsManagement], [PDF], [PDFAID], [PDFX], [PDFXID], and
// [MediaManagement]; see [Namespace] for how to define your own.
//
// The argument dst must be a non-nil pointer to such a struct, otherwise
// Get panics; it also panics if fields are ambiguously tied under the
// same xmp name (see [Namespace]) or if dst's type embeds a pointer to
// itself, directly or indirectly. Properties not present in the packet
// are set to the zero value of their field type; an embedded pointer
// field is left nil rather than allocated if the packet has no property
// for anything in its subtree.
//
// Get returns nil when every present property decoded successfully.
// When one or more present properties cannot be decoded into their
// fields, those fields (and any embedded pointer on the path to them)
// are left unchanged, the other fields are still populated, and Get
// returns the per-property errors joined via [errors.Join].  Each
// individual error is a [*PropertyError]; decode failures wrap
// [ErrInvalid].
func (p *Packet) Get(dst any) error {
	s := reflect.Indirect(reflect.ValueOf(dst))

	fields, groupParent, err := collectFields(s.Type())
	if err != nil {
		panic(err)
	}

	namespace, _, err := resolveMarkers(fields, groupParent)
	if err != nil {
		panic(err)
	}
	if namespace == "" {
		panic("not an XMP namespace struct")
	}

	props, err := resolveProperties(fields)
	if err != nil {
		panic(err)
	}

	var errs []error
	for _, f := range props {
		name := xml.Name{Space: namespace, Local: f.propName}
		xmpData, ok := p.Properties[name]
		if !ok {
			if fVal, ok := fieldByIndex(s, f.index); ok {
				// No allocation is needed to reach this field, so
				// zeroing it costs nothing and matches Get's usual
				// "properties not present are set to the zero value"
				// behaviour, e.g. when dst is a reused struct.
				fVal.Set(reflect.Zero(f.info.Type))
			}
			// If the field lies behind a nil embedded pointer, there is
			// nothing to zero: the nil pointer already represents the
			// zero value for its whole subtree, and allocating it just
			// to zero fields that are already conceptually zero would
			// defeat the point.
			continue
		}

		// Decode before touching dst, so that a property which fails to
		// decode does not allocate embedded pointers or otherwise mutate
		// the destination struct.
		zero := reflect.Zero(f.info.Type).Interface().(Value)
		u, err := zero.DecodeAnother(xmpData)
		if err != nil {
			errs = append(errs, &PropertyError{Name: name, Err: err})
			continue
		}

		fVal, err := allocFieldByIndex(s, f.index)
		if err != nil {
			errs = append(errs, &PropertyError{Name: name, Err: err})
			continue
		}
		fVal.Set(reflect.ValueOf(u))
	}
	return errors.Join(errs...)
}

// fieldEntry is one candidate field found while walking a namespace
// struct's fields with [collectFields].
type fieldEntry struct {
	// depth is the embedding depth at which the field was found: 0 for a
	// field declared directly on the struct passed to collectFields, 1
	// for one promoted from a struct it anonymously embeds, and so on.
	depth int

	// group identifies the particular struct field list (the struct
	// passed to collectFields itself, or one of the structs it
	// anonymously embeds, however deeply) that this field was declared
	// in. Two entries with the same group were declared side by side in
	// the same struct type.
	group int

	info reflect.StructField

	// propName is the field's effective xmp property name: its "xmp" tag,
	// or its Go field name if untagged. Meaningless for the [Namespace]/
	// [Prefix] marker fields, which use info.Tag directly instead.
	propName string

	// index is the field index path from the root struct to this field,
	// for use with [fieldByIndex] and [allocFieldByIndex].
	index []int
}

// groupParent maps a field group (see [fieldEntry.group]) to the group of
// the struct that anonymously embeds it, or -1 for the group of the
// struct passed to [collectFields] itself.
type groupParent = map[int]int

// collectFields walks t's fields, descending into anonymous (embedded)
// struct fields -- including anonymous pointer-to-struct fields -- the
// same way encoding/json does, and returns every field found together
// with enough bookkeeping ([fieldEntry.depth] and [fieldEntry.group]) for
// [resolveMarkers] and [resolveProperties] to resolve fields that are
// promoted from more than one place under the same effective name.  Each
// entry records its field index path ([fieldEntry.index]); the field
// values themselves are looked up later, via [fieldByIndex] or
// [allocFieldByIndex], only for a field that is actually read or written.
//
// An anonymous field is only descended into when it does not itself
// implement [Value] and is not a [Namespace] or [Prefix] marker; those
// are always leaves, even when embedded anonymously without the usual `_`
// field name, since decomposing them would defeat their purpose.
//
// A struct type that (directly or indirectly) embeds a pointer to itself
// is structurally cyclic: the walk cannot terminate, and neither could
// [Packet.Get], since each freshly allocated instance's own embedded
// pointer is nil again. collectFields reports such a type as an error.
//
// Note: unlike encoding/json, this is not cached per type; it re-walks
// the struct type on every [Packet.Set] and [Packet.Get] call. Namespace
// structs are small and this is not called in a hot loop, so the
// simplicity is worth the (unmeasured) cost.
func collectFields(t reflect.Type) ([]fieldEntry, groupParent, error) {
	var entries []fieldEntry
	parents := groupParent{}
	nextGroup := 0
	newGroup := func(parent int) int {
		g := nextGroup
		nextGroup++
		parents[g] = parent
		return g
	}

	// descend reports whether fInfo is an anonymous field whose fields
	// should be promoted into its enclosing struct, rather than being
	// treated as a leaf field itself.
	descend := func(fInfo reflect.StructField) bool {
		if fInfo.Type == nsTagType || fInfo.Type == prefixTagType {
			return false // markers are always leaves, even if anonymous
		}
		if !fInfo.Anonymous || fInfo.Type.Implements(typeType) {
			return false
		}
		switch fInfo.Type.Kind() {
		case reflect.Struct:
			return true
		case reflect.Pointer:
			return fInfo.Type.Elem().Kind() == reflect.Struct
		default:
			return false
		}
	}

	// seenPtrs holds the anonymous pointer types on the path currently
	// being walked; entries are removed again on the way back up, so a
	// diamond embedding (the same pointer type in two sibling branches)
	// is not mistaken for a cycle.
	seenPtrs := map[reflect.Type]bool{}

	var walk func(t reflect.Type, index []int, depth, group int) error
	walk = func(t reflect.Type, index []int, depth, group int) error {
		for i := 0; i < t.NumField(); i++ {
			fInfo := t.Field(i)
			idx := append(slices.Clone(index), i)

			if descend(fInfo) {
				childGroup := newGroup(group)
				if fInfo.Type.Kind() == reflect.Pointer {
					if seenPtrs[fInfo.Type] {
						return fmt.Errorf("xmp: cyclic embedded pointer type %s", fInfo.Type)
					}
					seenPtrs[fInfo.Type] = true
					err := walk(fInfo.Type.Elem(), idx, depth+1, childGroup)
					delete(seenPtrs, fInfo.Type)
					if err != nil {
						return err
					}
				} else {
					if err := walk(fInfo.Type, idx, depth+1, childGroup); err != nil {
						return err
					}
				}
				continue
			}

			name := fInfo.Tag.Get("xmp")
			if name == "" {
				name = fInfo.Name
			}
			entries = append(entries, fieldEntry{depth: depth, group: group, info: fInfo, propName: name, index: idx})
		}
		return nil
	}

	rootGroup := newGroup(-1)
	if err := walk(t, nil, 0, rootGroup); err != nil {
		return nil, nil, err
	}
	return entries, parents, nil
}

// fieldByIndex returns the field of s at the given index path, following
// anonymous pointer fields on the way.  ok is false if a nil pointer
// lies on the path, i.e. the field does not currently exist.
func fieldByIndex(s reflect.Value, index []int) (v reflect.Value, ok bool) {
	v = s
	for _, i := range index {
		if v.Kind() == reflect.Pointer {
			if v.IsNil() {
				return reflect.Value{}, false
			}
			v = v.Elem()
		}
		v = v.Field(i)
	}
	return v, true
}

// allocFieldByIndex is like [fieldByIndex], but allocates nil anonymous
// pointer fields on the path.  When it reaches a nil pointer it cannot
// set (an anonymous field that is itself unexported, see [Namespace]),
// it undoes the allocations it already made, leaving s unchanged, and
// returns an error.
//
// An anonymously embedded struct type may itself be unexported (e.g. a
// lower-case helper type shared between a family of exported namespace
// structs in another package): reflect still allows reading and setting
// its exported leaf fields, because it clears the read-only flag it
// would otherwise attach (flagEmbedRO) once the field reached through
// the embedding is itself exported. This relief does not extend to
// allocating an unexported anonymous pointer field: that means setting
// the unexported field itself, not a promoted subfield.
func allocFieldByIndex(s reflect.Value, index []int) (reflect.Value, error) {
	var allocated []reflect.Value
	v := s
	for _, i := range index {
		if v.Kind() == reflect.Pointer {
			if v.IsNil() {
				if !v.CanSet() {
					for _, pv := range allocated {
						pv.Set(reflect.Zero(pv.Type()))
					}
					return reflect.Value{}, fmt.Errorf("xmp: cannot allocate embedded field of unexported type %s", v.Type())
				}
				v.Set(reflect.New(v.Type().Elem()))
				allocated = append(allocated, v)
			}
			v = v.Elem()
		}
		v = v.Field(i)
	}
	return v, nil
}

// resolveMarkers finds the effective [Namespace]/[Prefix] pair among
// entries, or an error if it is ambiguous (see [Namespace]).
//
// The namespace is taken from the shallowest embedding depth at which
// exactly one [Namespace] field is found; more than one at that depth is
// reported as an ambiguity error rather than silently resolving to no
// namespace at all.
//
// The prefix search is then restricted to the group that declared the
// winning Namespace field and the chain of groups that (however deeply)
// anonymously embed it -- never a sibling group that merely happens to
// share the same depth -- so a [Prefix] can only pair with a [Namespace]
// it actually, lexically encloses. Among the groups in that chain that
// declare a Prefix, the shallowest wins, so a struct's own Prefix always
// overrides one inherited from something it embeds.
func resolveMarkers(entries []fieldEntry, parents groupParent) (namespace, prefix string, err error) {
	nsDepth := -1
	var nsCandidates []fieldEntry
	for _, e := range entries {
		if e.info.Type != nsTagType {
			continue
		}
		switch {
		case nsDepth == -1 || e.depth < nsDepth:
			nsDepth = e.depth
			nsCandidates = []fieldEntry{e}
		case e.depth == nsDepth:
			nsCandidates = append(nsCandidates, e)
		}
	}
	if len(nsCandidates) == 0 {
		return "", "", nil
	}
	if len(nsCandidates) > 1 {
		return "", "", fmt.Errorf("xmp: ambiguous Namespace field at embedding depth %d", nsDepth)
	}
	nsGroup := nsCandidates[0].group
	namespace = nsCandidates[0].info.Tag.Get("xmp")
	if namespace == "" {
		// Failing here, rather than reporting "no namespace found",
		// points at the actual defect: an untagged Namespace field
		// would otherwise silently shadow a tagged one found deeper.
		return "", "", errors.New("xmp: Namespace field has no xmp tag")
	}

	enclosing := map[int]bool{nsGroup: true}
	for g := nsGroup; parents[g] != -1; {
		g = parents[g]
		enclosing[g] = true
	}

	prefixDepth := -1
	var prefixCandidates []fieldEntry
	for _, e := range entries {
		if e.info.Type != prefixTagType || !enclosing[e.group] {
			continue
		}
		switch {
		case prefixDepth == -1 || e.depth < prefixDepth:
			prefixDepth = e.depth
			prefixCandidates = []fieldEntry{e}
		case e.depth == prefixDepth:
			prefixCandidates = append(prefixCandidates, e)
		}
	}
	if len(prefixCandidates) > 1 {
		return "", "", fmt.Errorf("xmp: ambiguous Prefix field at embedding depth %d", prefixDepth)
	}
	if len(prefixCandidates) == 1 {
		prefix = prefixCandidates[0].info.Tag.Get("xmp")
	}
	return namespace, prefix, nil
}

// resolveProperties returns the leaf property fields to visit -- those
// whose type implements [Value] and is exported -- resolved to one
// winner per effective xmp name ([fieldEntry.propName]) using the same
// shallowest-depth-wins rule as [resolveMarkers]: a name that appears at
// more than one depth resolves to the shallowest one. Names that tie at
// the same depth are reported as an ambiguity error rather than silently
// dropped (see [Namespace]). The result preserves the order in which
// each winning name was first encountered.
func resolveProperties(entries []fieldEntry) ([]fieldEntry, error) {
	type slot struct {
		depth int
		es    []fieldEntry
	}
	byName := make(map[string]*slot)
	var order []string
	for _, e := range entries {
		if e.info.PkgPath != "" || !e.info.Type.Implements(typeType) {
			continue
		}

		sl, ok := byName[e.propName]
		if !ok {
			byName[e.propName] = &slot{depth: e.depth, es: []fieldEntry{e}}
			order = append(order, e.propName)
			continue
		}
		switch {
		case e.depth < sl.depth:
			sl.depth = e.depth
			sl.es = []fieldEntry{e}
		case e.depth == sl.depth:
			sl.es = append(sl.es, e)
		}
	}

	var result []fieldEntry
	var errs []error
	for _, name := range order {
		sl := byName[name]
		if len(sl.es) == 1 {
			result = append(result, sl.es[0])
			continue
		}
		errs = append(errs, fmt.Errorf("xmp: ambiguous field for property %q at embedding depth %d", name, sl.depth))
	}
	return result, errors.Join(errs...)
}

var (
	nsTagType     = reflect.TypeFor[Namespace]()
	prefixTagType = reflect.TypeFor[Prefix]()
	typeType      = reflect.TypeFor[Value]()
)

// Namespace must be used in XMP namespace structs to specify the namespace
// URI.  The namespace URI is specified using a struct tag on a field of type
// Namespace.  For example:
//
//	type MyNamespace struct {
//	    _ Namespace `xmp:"http://example.com/ns/my/namespace/"`
//	    ...
//	}
//
// # Sharing fields between namespace structs via embedding
//
// A family of near-identical namespace versions (e.g. PRISM Basic
// 2.0/2.1/3.0) can share a common field set, or a common Namespace/[Prefix]
// pair, by factoring it out into a separate struct and embedding it
// anonymously, instead of repeating it in every variant:
//
//	type sharedFields struct {
//	    Foo Text `xmp:"foo"`
//	}
//
//	type MyNamespaceV1 struct {
//	    _ Namespace `xmp:"http://example.com/ns/my/namespace/1.0/"`
//	    sharedFields
//	}
//
// [Packet.Set] and [Packet.Get] promote the fields of an anonymously
// embedded struct -- including through an anonymous pointer-to-struct
// field -- into the enclosing struct, the way encoding/json promotes
// fields of embedded structs. A field of a type that implements [Value]
// (e.g. [Text], [Real]) is always treated as a single leaf property, even
// when embedded anonymously, since it is not meant to be decomposed into
// its own fields; Namespace and Prefix are leaves for the same reason,
// even without their usual `_` field name.
//
// A field declared directly on a struct always takes priority over one
// promoted from something it embeds, and a Namespace/Prefix pair is only
// ever taken from a single struct in the embedding chain -- the struct
// that declares the Namespace, or, failing that, the shallowest struct
// enclosing it that declares a Prefix -- never from an unrelated sibling
// embed. Two fields that tie under the same effective xmp name at the
// same embedding depth -- whether promoted from embedded structs or
// declared side by side in a single struct -- and likewise two Namespace
// (or two Prefix) fields tied at the same depth, are an ambiguity: Set
// reports it as an error and Get panics, rather than silently dropping
// or mispairing fields.
//
// An anonymously embedded pointer-to-struct field is promoted lazily: Set
// never allocates a nil one (it only reads values, so a nil embedded
// pointer's fields are simply treated as zero, clearing any property they
// would otherwise have written), and Get allocates one only once it
// actually has a property to decode into it. A struct type that embeds a
// pointer to itself, directly or indirectly, is rejected -- as an error
// by Set, with a panic by Get -- rather than walked forever. An
// anonymously embedded pointer field that
// is itself unexported (relative to the package that calls Set or Get)
// cannot be allocated at all; Get reports this as a per-property error
// instead of silently losing the data, matching encoding/json's
// Unmarshal.
type Namespace struct{}

// Prefix can be used in XMP namespace structs to optionally specify the
// preferred XML prefix for the namespace.  The prefix is specified using a
// struct tag on a field of type Prefix.  For example:
//
//	type MyNamespace struct {
//	    _ Namespace `xmp:"http://example.com/ns/my/namespace/"`
//	    _ Prefix    `xmp:"myns"`
//	    ...
//	}
//
// If no prefix is specified (or if there is a prefix name clash), a prefix is
// automatically chosen.  See [Namespace] for how a Prefix declared on an
// embedded struct is paired with the enclosing struct's Namespace.
type Prefix struct{}
