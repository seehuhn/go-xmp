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
	"fmt"
	"math"
	"mime"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/language"
)

// A Value represents a high-level data type for XMP values.
// The methods of this interface are used to serialize and deserialize values.
type Value interface {
	// IsZero returns true if the value is the zero value of its type, and
	// if the value has no qualifiers.
	IsZero() bool

	// EncodeXMP returns the low-level XMP representation of a value.
	// It returns an error wrapping [ErrInvalid] if the value cannot be
	// represented in a valid XMP packet.
	EncodeXMP(*Packet) (Raw, error)

	// DecodeAnother converts a low-level XMP representation into a [Value].
	// The resulting Value must have the same concrete type as the receiver.
	// The receiver is not used otherwise.  If the input is not a valid
	// representation of the concrete type, the error ErrInvalid is returned.
	DecodeAnother(Raw) (Value, error)
}

// ProperName represents a proper name.
type ProperName struct {
	V string
	Q
}

// NewProperName creates a new XMP proper name value.
func NewProperName(v string, qualifiers ...Qualifier) ProperName {
	return ProperName{V: v, Q: Q(qualifiers)}
}

func (p ProperName) String() string {
	return p.V
}

// IsZero implements the [Value] interface.
func (p ProperName) IsZero() bool {
	return p.V == "" && len(p.Q) == 0
}

// EncodeXMP implements the [Value] interface.
func (p ProperName) EncodeXMP(*Packet) (Raw, error) {
	return Text{
		V: p.V,
		Q: p.Q,
	}, nil
}

// DecodeAnother implements the [Value] interface.
func (ProperName) DecodeAnother(val Raw) (Value, error) {
	v, ok := val.(Text)
	if !ok {
		return nil, ErrInvalid
	}
	return ProperName{v.V, v.Q}, nil
}

// AgentName represents the name of some document creator software.
//
// The recommended format of this string is
//
//	Organization Software_name Version (token;token;...)
//
// where the fields have the following meanings:
//   - Organization: the company or organization providing the software, without spaces.
//   - Software_name: The full name of the software, spaces allowed.
//   - Version: The version of the software, without spaces.
//   - tokens: additional information, e.g. OS version
type AgentName struct {
	V string
	Q
}

// NewAgentName creates a new XMP AgentName object.
func NewAgentName(v string, qualifiers ...Qualifier) AgentName {
	return AgentName{V: v, Q: Q(qualifiers)}
}

func (a AgentName) String() string {
	return a.V
}

// IsZero implements the [Value] interface.
func (a AgentName) IsZero() bool {
	return a.V == "" && len(a.Q) == 0
}

// EncodeXMP implements the [Value] interface.
func (a AgentName) EncodeXMP(*Packet) (Raw, error) {
	return Text{
		V: a.V,
		Q: a.Q,
	}, nil
}

// DecodeAnother implements the [Value] interface.
func (AgentName) DecodeAnother(val Raw) (Value, error) {
	v, ok := val.(Text)
	if !ok {
		return nil, ErrInvalid
	}
	return AgentName{v.V, v.Q}, nil
}

// RenditionClass states the form or intended usage of a resource.  This is a
// series of colon-separated values, the first of which names the basic usage of
// the rendition and the rest are parameters.
//
// Defined values:
//   - "default": the default rendition of the resource (no parameters).
//   - "draft": a draft version of the resource.
//   - "low-res": a low-resolution version of the resource.
//   - "proof": a review proof.
//   - "screen": a screen-optimized version of the resource.
//   - "thumbnail": a thumbnail image.
//
// Example: "thumbnail:gif:8x8:bw"
type RenditionClass struct {
	V string
	Q
}

// IsZero implements the [Value] interface.
func (t RenditionClass) IsZero() bool {
	return t.V == "" && len(t.Q) == 0
}

// EncodeXMP implements the [Value] interface.
func (t RenditionClass) EncodeXMP(*Packet) (Raw, error) {
	return Text{
		V: t.V,
		Q: t.Q,
	}, nil
}

// DecodeAnother implements the [Value] interface.
func (RenditionClass) DecodeAnother(val Raw) (Value, error) {
	v, ok := val.(Text)
	if !ok {
		return nil, ErrInvalid
	}
	return RenditionClass{v.V, v.Q}, nil
}

// GUID represents a globally unique identifier.
type GUID struct {
	V string
	Q
}

// IsZero implements the [Value] interface.
func (t GUID) IsZero() bool {
	return t.V == "" && len(t.Q) == 0
}

// EncodeXMP implements the [Value] interface.
func (t GUID) EncodeXMP(*Packet) (Raw, error) {
	return Text{
		V: t.V,
		Q: t.Q,
	}, nil
}

// DecodeAnother implements the [Value] interface.
func (GUID) DecodeAnother(val Raw) (Value, error) {
	v, ok := val.(Text)
	if !ok {
		return nil, ErrInvalid
	}
	return GUID{v.V, v.Q}, nil
}

// Real represents a floating-point number.
type Real struct {
	V float64
	Q
}

// IsZero implements the [Value] interface.
func (r Real) IsZero() bool {
	return r.V == 0 && len(r.Q) == 0
}

// EncodeXMP implements the [Value] interface.
func (r Real) EncodeXMP(*Packet) (Raw, error) {
	if math.IsNaN(r.V) || math.IsInf(r.V, 0) {
		return nil, fmt.Errorf("xmp: Real %v: %w", r.V, ErrInvalid)
	}
	out := strconv.FormatFloat(r.V, 'f', -1, 64)
	if m := tailRegexp.FindStringSubmatchIndex(out); m != nil {
		if m[2] > 0 {
			out = out[:m[2]]
		} else if m[4] > 0 {
			out = out[:m[4]]
		}
	}
	if strings.HasPrefix(out, "0.") {
		out = out[1:]
	}
	return Text{
		V: out,
		Q: r.Q,
	}, nil
}

var (
	tailRegexp = regexp.MustCompile(`(?:\..*[1-9](0+)|(\.0+))$`)
)

// DecodeAnother implements the [Value] interface.
func (Real) DecodeAnother(val Raw) (Value, error) {
	v, ok := val.(Text)
	if !ok {
		return nil, ErrInvalid
	}
	f, err := strconv.ParseFloat(v.V, 64)
	if err != nil {
		return nil, ErrInvalid
	}
	return Real{f, v.Q}, nil
}

// Integer represents a signed integer.
type Integer struct {
	V int64
	Q
}

// NewInteger creates a new XMP integer value.
func NewInteger(v int64, qualifiers ...Qualifier) Integer {
	return Integer{V: v, Q: Q(qualifiers)}
}

// IsZero implements the [Value] interface.
func (i Integer) IsZero() bool {
	return i.V == 0 && len(i.Q) == 0
}

// EncodeXMP implements the [Value] interface.
func (i Integer) EncodeXMP(*Packet) (Raw, error) {
	return Text{
		V: strconv.FormatInt(i.V, 10),
		Q: i.Q,
	}, nil
}

// DecodeAnother implements the [Value] interface.
func (Integer) DecodeAnother(val Raw) (Value, error) {
	v, ok := val.(Text)
	if !ok {
		return nil, ErrInvalid
	}
	n, err := strconv.ParseInt(strings.TrimSpace(v.V), 10, 64)
	if err != nil {
		return nil, ErrInvalid
	}
	return Integer{n, v.Q}, nil
}

// Date represents a date and time.
type Date struct {
	V time.Time

	// NumOmitted can be used to reduce the precision of the date
	// when serializing it to XMP.  The value is a number between 0 and 5:
	// 1=omit nano, 2=omit sec, 3=omit time, 4=omit day, 5=month.
	NumOmitted int

	Q
}

// NewDate creates a new XMP date value.
func NewDate(t time.Time, qualifiers ...Qualifier) Date {
	return Date{V: t, Q: Q(qualifiers)}
}

// IsZero implements the [Value] interface.
func (d Date) IsZero() bool {
	return d.V.IsZero() && len(d.Q) == 0
}

// EncodeXMP implements the [Value] interface.
func (d Date) EncodeXMP(*Packet) (Raw, error) {
	numOmitted := d.NumOmitted
	numOmitted = min(numOmitted, len(dateFormats)-1)
	numOmitted = max(numOmitted, 0)
	format := dateFormats[numOmitted]
	return Text{
		V: d.V.Format(format),
		Q: d.Q,
	}, nil
}

// DecodeAnother implements the [Value] interface.
func (Date) DecodeAnother(val Raw) (Value, error) {
	v, ok := val.(Text)
	if !ok {
		return nil, ErrInvalid
	}
	dateString := v.V

	for i, format := range dateFormats {
		t, err := time.Parse(format, dateString)
		if err == nil {
			val := Date{
				V:          t,
				NumOmitted: i,
				Q:          v.Q,
			}
			return val, nil
		}
	}
	return nil, ErrInvalid
}

var dateFormats = []string{
	"2006-01-02T15:04:05.999999999Z07:00",
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02T15:04Z07:00",
	"2006-01-02",
	"2006-01",
	"2006",
}

// Locale represents a language code.
type Locale struct {
	V language.Tag
	Q
}

// NewLocale creates a new XMP locale value.
func NewLocale(tag language.Tag, qualifiers ...Qualifier) Locale {
	return Locale{V: tag, Q: Q(qualifiers)}
}

func (l Locale) String() string {
	return l.V.String()
}

// IsZero implements the [Value] interface.
func (l Locale) IsZero() bool {
	return l.V == language.Und && len(l.Q) == 0
}

// EncodeXMP implements the [Value] interface.
func (l Locale) EncodeXMP(*Packet) (Raw, error) {
	return Text{
		V: l.V.String(),
		Q: l.Q,
	}, nil
}

// DecodeAnother implements the [Value] interface.
func (Locale) DecodeAnother(val Raw) (Value, error) {
	v, ok := val.(Text)
	if !ok {
		return nil, ErrInvalid
	}
	tag, err := language.Parse(v.V)
	if err != nil {
		return nil, ErrInvalid
	}
	return Locale{
		V: tag,
		Q: v.Q,
	}, nil
}

// MimeType represents the media type of a file.
// The fields V and Param correspond to the values returned by
// [mime.ParseMediaType].
type MimeType struct {
	V     string
	Param map[string]string
	Q
}

func (m MimeType) String() string {
	return mime.FormatMediaType(m.V, m.Param)
}

// IsZero implements the [Value] interface.
func (m MimeType) IsZero() bool {
	return m.V == "" && len(m.Param) == 0 && len(m.Q) == 0
}

// EncodeXMP implements the [Value] interface.
func (m MimeType) EncodeXMP(*Packet) (Raw, error) {
	formatted := mime.FormatMediaType(m.V, m.Param)
	if formatted == "" && m.V != "" {
		return nil, fmt.Errorf("xmp: MimeType %q: %w", m.V, ErrInvalid)
	}
	return Text{
		V: formatted,
		Q: m.Q,
	}, nil
}

// DecodeAnother implements the [Value] interface.
func (m MimeType) DecodeAnother(val Raw) (Value, error) {
	v, ok := val.(Text)
	if !ok {
		return nil, ErrInvalid
	}
	mt, param, err := mime.ParseMediaType(v.V)
	if err != nil {
		return nil, ErrInvalid
	}
	if len(param) == 0 {
		param = nil
	}
	return MimeType{
		V:     mt,
		Param: param,
		Q:     v.Q,
	}, nil
}

// OptionalBool represents an optional boolean value.
// The possible values are "True", "False", and unset.
type OptionalBool struct {
	V int // 0 = unset, 1 = false, 2 = true
	Q
}

func (o OptionalBool) String() string {
	switch o.V {
	case 1:
		return "False"
	case 2:
		return "True"
	}
	return ""
}

// IsTrue returns true if the value is set to true.
func (o OptionalBool) IsTrue() bool {
	return o.V == 2
}

// IsFalse returns true if the value is set to false.
// Note that this is different from the value being unset.
func (o OptionalBool) IsFalse() bool {
	return o.V == 1
}

// IsZero implements the [Value] interface.
func (o OptionalBool) IsZero() bool {
	return o.V == 0 && len(o.Q) == 0
}

// EncodeXMP implements the [Value] interface.
func (o OptionalBool) EncodeXMP(*Packet) (Raw, error) {
	switch o.V {
	case 1:
		return Text{
			V: "False",
			Q: o.Q,
		}, nil
	case 2:
		return Text{
			V: "True",
			Q: o.Q,
		}, nil
	}
	return Text{
		V: "",
		Q: o.Q,
	}, nil
}

// DecodeAnother implements the [Value] interface.
func (OptionalBool) DecodeAnother(val Raw) (Value, error) {
	v, ok := val.(Text)
	if !ok {
		return nil, ErrInvalid
	}

	switch strings.ToLower(v.V) {
	case "true", "1":
		return OptionalBool{V: 2, Q: v.Q}, nil
	case "false", "0":
		return OptionalBool{V: 1, Q: v.Q}, nil
	case "":
		return OptionalBool{V: 0, Q: v.Q}, nil
	default:
		return nil, ErrInvalid
	}
}

// UnorderedArray is an unordered array of values.
// All elements of the array have the same type, E.
type UnorderedArray[E Value] struct {
	V []E
	Q
}

func (u *UnorderedArray[E]) Append(v E) {
	u.V = append(u.V, v)
}

// IsZero implements the [Value] interface.
func (u UnorderedArray[E]) IsZero() bool {
	return len(u.V) == 0 && len(u.Q) == 0
}

// EncodeXMP implements the [Value] interface.
func (u UnorderedArray[E]) EncodeXMP(p *Packet) (Raw, error) {
	var vals []Raw
	for _, v := range u.V {
		raw, err := v.EncodeXMP(p)
		if err != nil {
			return nil, err
		}
		vals = append(vals, raw)
	}
	return RawArray{
		Value: vals,
		Kind:  Unordered,
		Q:     u.Q,
	}, nil
}

// DecodeAnother implements the [Value] interface.
func (UnorderedArray[E]) DecodeAnother(val Raw) (Value, error) {
	a, ok := val.(RawArray)
	if !ok {
		// Try to fix invalid input files: if the data can be decoded as a
		// single E, return a single-element array.
		var tmp E
		if v, err := tmp.DecodeAnother(val); err == nil {
			return UnorderedArray[E]{V: []E{v.(E)}}, nil
		}

		return nil, ErrInvalid
	}
	// We ignore the array kind here.

	res := UnorderedArray[E]{Q: a.Q}
	res.V = make([]E, len(a.Value))
	for i, val := range a.Value {
		w, err := res.V[i].DecodeAnother(val)
		if err != nil {
			return nil, err
		}
		res.V[i] = w.(E)
	}
	res.Q = a.Q
	return res, nil
}

// OrderedArray is an ordered array of values.
// All elements of the array have the same type, E.
type OrderedArray[E Value] struct {
	V []E
	Q
}

// Append adds a new value to the array.
func (o *OrderedArray[E]) Append(v E) {
	o.V = append(o.V, v)
}

// IsZero implements the [Value] interface.
func (o OrderedArray[E]) IsZero() bool {
	return len(o.V) == 0 && len(o.Q) == 0
}

// EncodeXMP implements the [Value] interface.
func (o OrderedArray[E]) EncodeXMP(p *Packet) (Raw, error) {
	var vals []Raw
	for _, v := range o.V {
		raw, err := v.EncodeXMP(p)
		if err != nil {
			return nil, err
		}
		vals = append(vals, raw)
	}
	return RawArray{
		Value: vals,
		Kind:  Ordered,
		Q:     o.Q,
	}, nil
}

// DecodeAnother implements the [Value] interface.
func (OrderedArray[E]) DecodeAnother(val Raw) (Value, error) {
	a, ok := val.(RawArray)
	if !ok {
		// Try to fix invalid input files: if the data can be decoded as a
		// single E, return a single-element array.
		var tmp E
		if v, err := tmp.DecodeAnother(val); err == nil {
			return OrderedArray[E]{V: []E{v.(E)}}, nil
		}

		return nil, ErrInvalid
	}
	// We ignore the array kind here.

	res := OrderedArray[E]{Q: a.Q}
	res.V = make([]E, len(a.Value))
	for i, val := range a.Value {
		w, err := res.V[i].DecodeAnother(val)
		if err != nil {
			return nil, err
		}
		res.V[i] = w.(E)
	}
	res.Q = a.Q
	return res, nil
}

// AlternativeArray is an array of alternative values for a single property.
// All values in the array have the same type E, and represent equivalent
// renditions of the same logical value.  The order of the entries is not
// significant, except that the first entry is the default rendition.
type AlternativeArray[E Value] struct {
	V []E
	Q
}

// IsZero implements the [Value] interface.
func (a AlternativeArray[E]) IsZero() bool {
	return len(a.V) == 0 && len(a.Q) == 0
}

// EncodeXMP implements the [Value] interface.
func (a AlternativeArray[E]) EncodeXMP(p *Packet) (Raw, error) {
	var vals []Raw
	for _, v := range a.V {
		raw, err := v.EncodeXMP(p)
		if err != nil {
			return nil, err
		}
		vals = append(vals, raw)
	}
	return RawArray{
		Value: vals,
		Kind:  Alternative,
		Q:     a.Q,
	}, nil
}

// DecodeAnother implements the [Value] interface.
func (AlternativeArray[E]) DecodeAnother(val Raw) (Value, error) {
	a, ok := val.(RawArray)
	if !ok {
		// Try to fix invalid input files: if the data can be decoded as a
		// single E, return a single-element array.
		var tmp E
		if v, err := tmp.DecodeAnother(val); err == nil {
			return AlternativeArray[E]{V: []E{v.(E)}}, nil
		}

		return nil, ErrInvalid
	}
	// We ignore the array kind here.

	res := AlternativeArray[E]{Q: a.Q}
	res.V = make([]E, len(a.Value))
	for i, val := range a.Value {
		w, err := res.V[i].DecodeAnother(val)
		if err != nil {
			return nil, err
		}
		res.V[i] = w.(E)
	}
	res.Q = a.Q
	return res, nil
}

// Localized represents a localized text value.  This is a map from language
// tags to strings.  This is implemented as an XMP "Language Alternative".
//
// The XMP Language Alternative array carries one item with the special
// language tag "x-default", whose text duplicates one of the other entries
// and tells readers which language to display when no better match is
// available.  In this Go representation the x-default text lives in the
// [Localized.Default] field, never as a key in [Localized.V].
//
// In all API entry points that take a [language.Tag], the zero value
// [language.Und] is treated as a synonym for "x-default".  Prefer
// [language.Und] over the parsed "x-default" tag in calling code.
type Localized struct {
	// V holds the per-language text entries.  Neither "x-default" nor
	// [language.Und] may appear as a key here; use [Localized.Default]
	// instead.  Encoding a Localized whose V contains either key
	// returns an error wrapping [ErrInvalid].
	V map[language.Tag]Text

	// Default (optional) is the text emitted with the "x-default" tag.
	// Its contents are expected to match one of the entries in V; an
	// empty Default means the array has no default item.
	Default Text

	Q
}

// Set stores the text value for a specific language.  Passing
// [language.Und] (or the parsed "x-default" tag) updates
// [Localized.Default] instead of adding a key to [Localized.V].
func (l *Localized) Set(lang language.Tag, txt string, qualifiers ...Qualifier) {
	if isDefaultLang(lang) {
		l.Default = NewText(txt, qualifiers...)
		return
	}
	if l.V == nil {
		l.V = make(map[language.Tag]Text)
	}
	l.V[lang] = NewText(txt, qualifiers...)
}

// IsZero implements the [Value] interface.
func (l Localized) IsZero() bool {
	return len(l.V) == 0 && l.Default.IsZero() && len(l.Q) == 0
}

var defaultLanguage = language.MustParse("x-default")

// isDefaultLang reports whether tag denotes the XMP "x-default" item.
// Both the parsed "x-default" tag and [language.Und] (the zero tag) are
// accepted; library code uses this in every place that distinguishes
// "the default item" from a real per-language entry.
func isDefaultLang(tag language.Tag) bool {
	return tag == defaultLanguage || tag == language.Und
}

// EncodeXMP implements the [Value] interface.
func (l Localized) EncodeXMP(*Packet) (Raw, error) {
	for k := range l.V {
		if isDefaultLang(k) {
			return nil, fmt.Errorf("xmp: Localized.V contains x-default key: %w", ErrInvalid)
		}
	}

	var vals []Raw
	if l.Default.V != "" {
		vals = append(vals, Text{
			V: l.Default.V,
			Q: l.Default.Q.WithLanguage(defaultLanguage),
		})
	}
	for lang, txt := range l.V {
		vals = append(vals, Text{
			V: txt.V,
			Q: txt.Q.WithLanguage(lang),
		})
	}

	return RawArray{
		Value: vals,
		Kind:  Alternative,
		Q:     l.Q,
	}, nil
}

// DecodeAnother implements the [Value] interface.
func (Localized) DecodeAnother(val Raw) (Value, error) {
	a, ok := val.(RawArray)
	if !ok {
		// Try to fix invalid input files: if the data can be decoded as a
		// single E, return a single-element array.
		var tmp Text
		if v, err := tmp.DecodeAnother(val); err == nil {
			return Localized{Default: v.(Text)}, nil
		}

		return nil, ErrInvalid
	}
	// We ignore the array kind here.

	res := Localized{
		V: map[language.Tag]Text{},
		Q: a.Q,
	}
	for _, val := range a.Value {
		v, ok := val.(Text)
		if !ok {
			return nil, ErrInvalid
		}
		lang, Q := v.Q.StripLanguage()
		if isDefaultLang(lang) {
			res.Default = Text{V: v.V, Q: Q}
		} else {
			res.V[lang] = Text{V: v.V, Q: Q}
		}
	}
	return res, nil
}

// Best returns the text value most appropriate for the requested
// language.  It picks the best reasonable match for lang from
// [Localized.V] using [language.Matcher]; if no entry is a reasonable
// match (or lang denotes the default item), it returns
// [Localized.Default].  Both [language.Und] and the parsed "x-default"
// tag select the default directly.  Returns "" when neither a
// reasonable match nor a default is available.
func (l Localized) Best(lang language.Tag) string {
	if len(l.V) > 0 && !isDefaultLang(lang) {
		// sort tags so the matcher's tie-break is deterministic
		tags := make([]language.Tag, 0, len(l.V))
		for t := range l.V {
			tags = append(tags, t)
		}
		slices.SortFunc(tags, func(a, b language.Tag) int {
			return strings.Compare(a.String(), b.String())
		})
		m := language.NewMatcher(tags)
		_, idx, conf := m.Match(lang)
		if conf >= language.Low && idx >= 0 && idx < len(tags) {
			return l.V[tags[idx]].V
		}
	}
	return l.Default.V
}

// ResourceRef represents a reference to an external resource.
//
// This corresponds to the XMP "ResourceRef" structured type defined in
// section 8.6 of ISO 16684-1:2011, in the namespace
// http://ns.adobe.com/xap/1.0/sType/ResourceRef#.
type ResourceRef struct {
	// DocumentID is the document ID of the referenced resource,
	// as found in the xmpMM:DocumentID field.
	DocumentID GUID

	// FilePath is the file path or URL of the referenced resource.
	FilePath URL

	// InstanceID is the instance ID of the referenced resource,
	// as found in the xmpMM:InstanceID field.
	InstanceID GUID

	// RenditionClass is the rendition class of the referenced resource.
	RenditionClass RenditionClass

	// RenditionParams provides additional rendition parameters that are too
	// complex or volatile to encode in [RenditionClass].
	RenditionParams Text

	Q
}

// IsZero implements the [Value] interface.
func (r ResourceRef) IsZero() bool {
	return r.DocumentID.IsZero() &&
		r.FilePath.IsZero() &&
		r.InstanceID.IsZero() &&
		r.RenditionClass.IsZero() &&
		r.RenditionParams.IsZero() &&
		len(r.Q) == 0
}

// EncodeXMP implements the [Value] interface.
func (r ResourceRef) EncodeXMP(p *Packet) (Raw, error) {
	p.RegisterPrefix(NSResourceRef, "stRef")
	res := RawStruct{
		Value: map[xml.Name]Raw{},
		Q:     r.Q,
	}
	encode := func(local string, v Value) error {
		raw, err := v.EncodeXMP(p)
		if err != nil {
			return err
		}
		res.Value[xml.Name{Space: NSResourceRef, Local: local}] = raw
		return nil
	}
	if !r.DocumentID.IsZero() {
		if err := encode("documentID", r.DocumentID); err != nil {
			return nil, err
		}
	}
	if !r.FilePath.IsZero() {
		if err := encode("filePath", r.FilePath); err != nil {
			return nil, err
		}
	}
	if !r.InstanceID.IsZero() {
		if err := encode("instanceID", r.InstanceID); err != nil {
			return nil, err
		}
	}
	if !r.RenditionClass.IsZero() {
		if err := encode("renditionClass", r.RenditionClass); err != nil {
			return nil, err
		}
	}
	if !r.RenditionParams.IsZero() {
		if err := encode("renditionParams", r.RenditionParams); err != nil {
			return nil, err
		}
	}
	return res, nil
}

// DecodeAnother implements the [Value] interface.
func (ResourceRef) DecodeAnother(val Raw) (Value, error) {
	s, ok := val.(RawStruct)
	if !ok {
		return nil, ErrInvalid
	}
	res := ResourceRef{Q: s.Q}
	for name, v := range s.Value {
		if name.Space != NSResourceRef {
			continue
		}
		switch name.Local {
		case "documentID":
			x, err := GUID{}.DecodeAnother(v)
			if err != nil {
				return nil, err
			}
			res.DocumentID = x.(GUID)
		case "filePath":
			x, err := URL{}.DecodeAnother(v)
			if err != nil {
				return nil, err
			}
			res.FilePath = x.(URL)
		case "instanceID":
			x, err := GUID{}.DecodeAnother(v)
			if err != nil {
				return nil, err
			}
			res.InstanceID = x.(GUID)
		case "renditionClass":
			x, err := RenditionClass{}.DecodeAnother(v)
			if err != nil {
				return nil, err
			}
			res.RenditionClass = x.(RenditionClass)
		case "renditionParams":
			x, err := Text{}.DecodeAnother(v)
			if err != nil {
				return nil, err
			}
			res.RenditionParams = x.(Text)
		}
	}
	return res, nil
}
