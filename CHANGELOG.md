# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.7.3] (2026-05-19)

### Fixed
- `Scan` worst-case work on adversarial host bytes is now bounded
  regardless of input length (at most 16 candidate wrappers, each
  bounded by the 16 MiB packet cap).  Earlier versions could degrade
  to retry-heavy behaviour on blobs full of corrupt nested wrappers.
- `Scan` no longer misaligns when UTF-16 payload bytes happen to form
  a valid UTF-8 multibyte sequence (e.g. U+CC80 in UTF-16BE =
  `0xCC 0x80`, also valid UTF-8 for U+0300).  The `(nil, nil)` vs
  `(nil, err)` contract is tightened to match the documented intent.

## [0.7.2] (2026-05-11)

### Added
- `Scan` function that locates an XMP packet wrapper inside arbitrary
  host bytes, preferring the document-level packet (`rdf:about=""`).
- `Read` accepts UTF-16BE and UTF-16LE input (with or without BOM) in
  addition to UTF-8, and enforces a 16 MiB cap on packet size.
- `Packet.PadToLength` field requests trailing whitespace padding so
  the encoded packet reaches a fixed byte length.  When set, the
  trailer switches from `end="r"` to `end="w"` to mark the packet as
  in-place editable, and `Read` records the source byte count so an
  unmodified round-trip refits the original host-file segment.
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
- Sentinel errors `ErrMalformed`, `ErrPacketTooLarge`,
  `ErrPacketTooLong`, and `ErrInvalidName`.  Malformed-input errors
  from `Read` are wrapped so `errors.Is(err, ErrMalformed)` reports
  them, while I/O errors from the underlying reader pass through
  unchanged.
- `Localized.Best` method, returning the best language match in the
  packet (or `Default` when no reasonable match exists).
- `Date.String` method returning its canonical XMP serialization
  (truncated according to `Date.Precision`).  It returns `""` for a
  zero `V` and clamps an out-of-range `Precision` so that fmt-style
  formatting stays panic-free.

### Fixed
- `ResourceRef` now actually implements `Value`; previously
  `Packet.Set(&MediaManagement{...})` panicked.
- `Scan` no longer exhibits Θ(n²) behaviour on inputs containing many
  bare `id="W5M0..."` attributes without an enclosing `<?xpacket`
  header.
- `Read` no longer poisons `PadToLength` when the source is too short
  to re-emit; previously a malformed end-PI on too-short input made
  every subsequent `Write` return `ErrPacketTooLong`.
- `xmlns` declarations on a property element no longer affect how the
  element is classified by the RDF property-element rules.  This
  matches the spec and lets XMP produced by writers such as pikepdf
  decode correctly.
- Property parsing has a recursion-depth cap to bound work on
  adversarial input.

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
- `Packet.SetValue` now returns `error` instead of panicking on
  invalid property identifiers.  The error wraps `ErrInvalidName`.
- `MediaManagement.RenditionClass` has type `RenditionClass` instead
  of `Text`, matching the spec.
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
- Minimum Go version raised to 1.25.

## [0.7.1] (2026-03-31)

### Changed
- Replaced `golang.org/x/exp` with standard library.

## [0.7.0] (2025-01-25)

Initial tagged release.
