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
	"mime"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/language"
)

// Text represents a simple text value.
type Text struct {
	V string
	Q
}

// NewText creates a new XMP text value.
func NewText(s string, qualifiers ...Qualifier) Text {
	return Text{V: s, Q: Q(qualifiers)}
}

func (t Text) String() string {
	return t.V
}

// IsZero implements the [Value] interface.
func (t Text) IsZero() bool {
	return t.V == "" && len(t.Q) == 0
}

// GetXMP implements the [Value] interface.
func (t Text) GetXMP() Raw {
	return RawText{
		Value: t.V,
		Q:     t.Q,
	}
}

// DecodeAnother implements the [Value] interface.
func (Text) DecodeAnother(val Raw) (Value, error) {
	v, ok := val.(RawText)
	if !ok {
		return nil, ErrInvalid
	}
	return Text{v.Value, v.Q}, nil
}

// ProperName represents a proper name.
type ProperName struct {
	V string
	Q
}

func (p ProperName) String() string {
	return p.V
}

// IsZero implements the [Value] interface.
func (p ProperName) IsZero() bool {
	return p.V == "" && len(p.Q) == 0
}

// GetXMP implements the [Value] interface.
func (p ProperName) GetXMP() Raw {
	return RawText{
		Value: p.V,
		Q:     p.Q,
	}
}

// DecodeAnother implements the [Value] interface.
func (ProperName) DecodeAnother(val Raw) (Value, error) {
	v, ok := val.(RawText)
	if !ok {
		return nil, ErrInvalid
	}
	return ProperName{v.Value, v.Q}, nil
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

// IsZero implements the [Value] interface.
func (t AgentName) IsZero() bool {
	return t.V == "" && len(t.Q) == 0
}

// GetXMP implements the [Value] interface.
func (t AgentName) GetXMP() Raw {
	return RawText{
		Value: t.V,
		Q:     t.Q,
	}
}

// DecodeAnother implements the [Value] interface.
func (AgentName) DecodeAnother(val Raw) (Value, error) {
	v, ok := val.(RawText)
	if !ok {
		return nil, ErrInvalid
	}
	return AgentName{v.Value, v.Q}, nil
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

// GetXMP implements the [Value] interface.
func (t GUID) GetXMP() Raw {
	return RawText{
		Value: t.V,
		Q:     t.Q,
	}
}

// DecodeAnother implements the [Value] interface.
func (GUID) DecodeAnother(val Raw) (Value, error) {
	v, ok := val.(RawText)
	if !ok {
		return nil, ErrInvalid
	}
	return GUID{v.Value, v.Q}, nil
}

// URL represents an XMP URL value.
type URL struct {
	V *url.URL
	Q
}

// NewURL creates a new XMP URL value.
func NewURL(u *url.URL, qualifiers ...Qualifier) URL {
	return URL{V: u, Q: Q(qualifiers)}
}

func (u URL) String() string {
	return u.V.String()
}

// IsZero implements the [Value] interface.
func (u URL) IsZero() bool {
	return u.V == nil && len(u.Q) == 0
}

// GetXMP implements the [Value] interface.
func (u URL) GetXMP() Raw {
	return RawURI{
		Value: u.V,
		Q:     u.Q,
	}
}

// DecodeAnother implements the [Value] interface.
func (URL) DecodeAnother(val Raw) (Value, error) {
	v, ok := val.(RawURI)
	if !ok {
		return nil, ErrInvalid
	}
	return URL{v.Value, v.Q}, nil
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

// GetXMP implements the [Value] interface.
func (r Real) GetXMP() Raw {
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
	return RawText{
		Value: out,
		Q:     r.Q,
	}
}

var (
	tailRegexp = regexp.MustCompile(`(?:\..*[1-9](0+)|(\.0+))$`)
)

// DecodeAnother implements the [Value] interface.
func (Real) DecodeAnother(val Raw) (Value, error) {
	v, ok := val.(RawText)
	if !ok {
		return nil, ErrInvalid
	}
	f, err := strconv.ParseFloat(v.Value, 64)
	if err != nil {
		return nil, ErrInvalid
	}
	return Real{f, v.Q}, nil
}

// Date represents a date and time.
type Date struct {
	V          time.Time
	NumOmitted int // 1=omit nano, 2=omit sec, 3=omit time, 4=omit day, 5=month
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

// GetXMP implements the [Value] interface.
func (d Date) GetXMP() Raw {
	numOmitted := d.NumOmitted
	numOmitted = min(numOmitted, len(dateFormats)-1)
	numOmitted = max(numOmitted, 0)
	format := dateFormats[numOmitted]
	return RawText{
		Value: d.V.Format(format),
		Q:     d.Q,
	}
}

// DecodeAnother implements the [Value] interface.
func (Date) DecodeAnother(val Raw) (Value, error) {
	v, ok := val.(RawText)
	if !ok {
		return nil, ErrInvalid
	}
	dateString := v.Value

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

// Locale represents an XMP language code.
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

// GetXMP implements the [Value] interface.
func (l Locale) GetXMP() Raw {
	return RawText{
		Value: l.V.String(),
		Q:     l.Q,
	}
}

// DecodeAnother implements the [Value] interface.
func (Locale) DecodeAnother(val Raw) (Value, error) {
	v, ok := val.(RawText)
	if !ok {
		return nil, ErrInvalid
	}
	tag, err := language.Parse(v.Value)
	if err != nil {
		return nil, ErrInvalid
	}
	return Locale{
		V: tag,
		Q: v.Q,
	}, nil
}

// MimeType represents a MIME type.
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

// GetXMP implements the [Value] interface.
func (m MimeType) GetXMP() Raw {
	return RawText{
		Value: mime.FormatMediaType(m.V, m.Param),
		Q:     m.Q,
	}
}

// DecodeAnother implements the [Value] interface.
func (m MimeType) DecodeAnother(val Raw) (Value, error) {
	v, ok := val.(RawText)
	if !ok {
		return nil, ErrInvalid
	}
	mt, param, err := mime.ParseMediaType(v.Value)
	if err != nil {
		return nil, ErrInvalid
	}
	return MimeType{
		V:     mt,
		Param: param,
		Q:     v.Q,
	}, nil
}

// OptionalBool represents an optional boolean value,
// with the values "True", "False", and unset.
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

// GetXMP implements the [Value] interface.
func (o OptionalBool) GetXMP() Raw {
	switch o.V {
	case 1:
		return RawText{
			Value: "False",
			Q:     o.Q,
		}
	case 2:
		return RawText{
			Value: "True",
			Q:     o.Q,
		}
	}
	return RawText{
		Value: "",
		Q:     o.Q,
	}
}

// DecodeAnother implements the [Value] interface.
func (OptionalBool) DecodeAnother(val Raw) (Value, error) {
	v, ok := val.(RawText)
	if !ok {
		return nil, ErrInvalid
	}

	switch strings.ToLower(v.Value) {
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

// UnorderedArray represents an unordered array of values.
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

// GetXMP implements the [Value] interface.
func (u UnorderedArray[E]) GetXMP() Raw {
	var vals []Raw
	for _, v := range u.V {
		vals = append(vals, v.GetXMP())
	}
	return RawArray{
		Value: vals,
		Kind:  Unordered,
		Q:     u.Q,
	}
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

// OrderedArray represents an ordered array of values.
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

// GetXMP implements the [Value] interface.
func (o OrderedArray[E]) GetXMP() Raw {
	var vals []Raw
	for _, v := range o.V {
		vals = append(vals, v.GetXMP())
	}
	return RawArray{
		Value: vals,
		Kind:  Ordered,
		Q:     o.Q,
	}
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

// AlternativeArray represents an ordered array of values.
type AlternativeArray[E Value] struct {
	V []E
	Q
}

// IsZero implements the [Value] interface.
func (a AlternativeArray[E]) IsZero() bool {
	return len(a.V) == 0 && len(a.Q) == 0
}

// GetXMP implements the [Value] interface.
func (a AlternativeArray[E]) GetXMP() Raw {
	var vals []Raw
	for _, v := range a.V {
		vals = append(vals, v.GetXMP())
	}
	return RawArray{
		Value: vals,
		Kind:  Alternative,
		Q:     a.Q,
	}
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

// Localized represents an XMP "Language Alternative" value.
type Localized struct {
	V map[language.Tag]Text

	// Default (optional) is the default value for the property.
	// If Value is non-empty, the text contents of Default must coincide with
	// the text contents of one of the values in the map.
	Default Text

	Q
}

// Set sets the text value for a specific language.
func (l *Localized) Set(lang language.Tag, txt string, qualifiers ...Qualifier) {
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

// GetXMP implements the [Value] interface.
func (l Localized) GetXMP() Raw {
	var vals []Raw

	if l.Default.V != "" {
		t := RawText{
			Value: l.Default.V,
			Q:     l.Default.Q.WithLanguage(defaultLanguage),
		}
		vals = append(vals, t)
	}
	for lang, txt := range l.V {
		t := RawText{
			Value: txt.V,
			Q:     txt.Q.WithLanguage(lang),
		}
		vals = append(vals, t)
	}

	return RawArray{
		Value: vals,
		Kind:  Alternative,
		Q:     l.Q,
	}
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
		v, ok := val.(RawText)
		if !ok {
			return nil, ErrInvalid
		}
		lang, Q := v.Q.StripLanguage()
		if lang == defaultLanguage {
			res.Default = Text{V: v.Value, Q: Q}
		} else {
			res.V[lang] = Text{V: v.Value, Q: Q}
		}
	}
	return res, nil
}

type ResourceRef struct {
	// DocumentID is the document ID of the referenced resource,
	// as found in the xmpMM:DocumentID field.
	DocumentID GUID

	// FilePath is the file path or URL of the referenced resource.
	FilePath URL

	// InstanceID is the instance ID of the referenced resource,
	// as found in the xmpMM:InstanceID field.
	InstanceID GUID

	ReditionClass Text // TODO(voss): change the type to "RenditionClass"

	RenditionParams Text

	Q
}

// IsZero implements the [Value] interface.
func (r *ResourceRef) IsZero() bool {
	return r == nil
}

// GetXMP implements the [Value] interface.
func (r *ResourceRef) GetXMP() Raw {
	ns := "http://ns.adobe.com/xap/1.0/sType/ResourceRef#"
	// TODO(voss): how to set the "stRef" prefix for this namespace?
	res := &RawStruct{}
	if !r.DocumentID.IsZero() {
		res.Value[xml.Name{Space: ns, Local: "documentID"}] = r.DocumentID.GetXMP()
	}
	if !r.FilePath.IsZero() {
		res.Value[xml.Name{Space: ns, Local: "filePath"}] = r.FilePath.GetXMP()
	}
	if !r.InstanceID.IsZero() {
		res.Value[xml.Name{Space: ns, Local: "instanceID"}] = r.InstanceID.GetXMP()
	}
	if !r.ReditionClass.IsZero() {
		res.Value[xml.Name{Space: ns, Local: "reditionClass"}] = r.ReditionClass.GetXMP()
	}
	if !r.RenditionParams.IsZero() {
		res.Value[xml.Name{Space: ns, Local: "renditionParams"}] = r.RenditionParams.GetXMP()
	}

	return res
}
