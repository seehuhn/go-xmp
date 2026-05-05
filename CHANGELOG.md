# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `PDF` namespace model for the Adobe PDF namespace
  (`http://ns.adobe.com/pdf/1.3/`), with fields `Keywords`, `PDFVersion`,
  `Producer`, and `Trapped`.
- `PDFAID` namespace model for the PDF/A identification schema
  (`http://www.aiim.org/pdfa/ns/id/`), covering PDF/A-1 through
  PDF/A-4.
- `PDFX` namespace model for the legacy Adobe PDF/X identification
  schema (`http://ns.adobe.com/pdfx/1.3/`), used by PDF/X-1a, PDF/X-2,
  and PDF/X-3.
- `PDFXID` namespace model for the PDF/X identification schema
  (`http://www.npes.org/pdfx/ns/id/`) introduced in PDF/X-4.
- `Integer` value type for signed integer XMP properties.
- Exported namespace URI constants (`NSDublinCore`, `NSBasic`,
  `NSRightsManagement`, `NSMediaManagement`, `NSPDF`, `NSPDFAID`,
  `NSPDFX`, `NSPDFXID`, `NSResourceRef`, `NSXML`, `NSRDF`).
- `PropertyError` type identifying which property failed to decode
  during a `Packet.Get` call.
- `ErrInvalidName` sentinel error returned (wrapped) by
  `Packet.SetValue` when the namespace or property name is rejected
  by the XMP property-name rules.
- `Localized.Best` method, returning the best language match in the
  packet (or `Default` when no reasonable match exists).
- `Date` has a new `String` method returning its canonical XMP
  serialization (truncated according to `Date.Precision`).  It
  returns `""` for a zero `V` and clamps an out-of-range
  `Precision` so that fmt-style formatting stays panic-free.

### Fixed
- `xmlns` declarations on a property element no longer affect how the
  element is classified by the RDF property-element rules.  This
  matches the spec and lets XMP produced by writers such as pikepdf
  decode correctly.

### Changed
- The `Value` interface's `EncodeXMP` method now returns
  `(Raw, error)`.  Implementations validate their inputs and return
  an error wrapping `ErrInvalid` when the value cannot be represented
  in a valid XMP packet (`Real` rejects NaN and infinities, `MimeType`
  rejects malformed media types, `Localized` rejects an x-default key
  in `V`).  `Packet.SetValue` propagates these errors and does not
  store the property when encoding fails.
- `Packet.Get` now returns `error`.  Per-property decode failures are
  reported as `*PropertyError` values aggregated via `errors.Join`;
  fields whose data decoded successfully are still populated.
- `Scan` now returns `(*Packet, error)`.  The error is non-nil when
  XMP-shaped wrappers were present in the input but every parse
  attempt failed; in that case the returned packet is nil and the
  error wraps `ErrMalformed`.  `(nil, nil)` continues to mean "no
  XMP packet found in the input."
- `Packet.SetValue` now returns `error` instead of panicking on
  invalid property identifiers.  The error wraps `ErrInvalidName`.
- `Localized.Set` now stores `x-default` values in the `Default`
  field rather than as a key in `V`, matching how the decoder
  reconstructs Lang Alt arrays.
- `Localized` now treats `language.Und` as a synonym for `x-default`
  in `Set`, `EncodeXMP`, `DecodeAnother`, and `Best`.  Calling code
  should prefer `language.Und` over the parsed `x-default` tag.
- `Date.NumOmitted int` is replaced by `Date.Precision DatePrecision`,
  with named constants `PrecisionFull`, `PrecisionSecond`,
  `PrecisionMinute`, `PrecisionDay`, `PrecisionMonth`, and
  `PrecisionYear`.
- `NewDate` now takes the precision as a required second argument:
  `NewDate(t time.Time, p DatePrecision, qualifiers ...Qualifier)`.
- `Date.EncodeXMP` rejects an out-of-range `Precision` with an error
  wrapping `ErrInvalid`.
- The canonical encoding of a `PrecisionFull` value now always emits
  nine fractional-second digits, so a Decode/Encode/Decode cycle is
  idempotent for inputs containing explicit zero fractional seconds.

## [0.7.1] (2026-03-31)

### Changed
- Replaced `golang.org/x/exp` with standard library.

## [0.7.0] (2025-01-25)

Initial tagged release.
