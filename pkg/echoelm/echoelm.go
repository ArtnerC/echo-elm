// Package echoelm is the public API for the echo-elm CQL→ELM translator.
//
// See contracts/go-api.md for the full API surface specification.
package echoelm

import (
	"encoding/json"

	"github.com/artnerc/echo-elm/internal/cqloptions"
	"github.com/artnerc/echo-elm/internal/elm"
	"github.com/artnerc/echo-elm/internal/parser"
	"github.com/artnerc/echo-elm/internal/resolver"
	"github.com/artnerc/echo-elm/internal/translator"
)

// LibrarySource provides CQL source bytes for a named/versioned library.
// Use it to resolve `include` declarations from non-filesystem sources.
type LibrarySource = resolver.LibrarySource

// Option is a functional option for Translate.
type Option func(*translator.Options)

// WithOptions replaces all translator options at once.
//
//nolint:gocritic // hugeParam: public API, changing to pointer would break callers
func WithOptions(opts translator.Options) Option {
	return func(o *translator.Options) {
		*o = opts
	}
}

// WithAnnotations enables or disables ELM annotations.
func WithAnnotations(v bool) Option {
	return func(o *translator.Options) { o.EnableAnnotations = v }
}

// WithLocators enables or disables source locators.
func WithLocators(v bool) Option {
	return func(o *translator.Options) { o.EnableLocators = v }
}

// WithSignatureLevel sets the signature level.
func WithSignatureLevel(level string) Option {
	return func(o *translator.Options) { o.SignatureLevel = level }
}

// WithCQFMode enables CQF parity mode: empty annotation/filter arrays on all
// defs, empty translatorOptions string — matching cqframework CLI output exactly.
func WithCQFMode(v bool) Option {
	return func(o *translator.Options) { o.CQFMode = v }
}

// WithCQFOptions applies the full cqframework-compatible default option set:
// CQFMode=true, EnableAnnotations=false, EnableLocators=false, SignatureLevel="None".
// Use this instead of WithCQFMode for parity testing and `echo-elm cqf translate`.
func WithCQFOptions() Option {
	return func(o *translator.Options) {
		*o = translator.CQFDefaultOptions()
	}
}

// WithIntervalDemotion enables implicit demotion of Interval<T> to T.
// CQF flag: --enable-interval-demotion. Default off.
func WithIntervalDemotion(v bool) Option {
	return func(o *translator.Options) { o.EnableIntervalDemotion = v }
}

// WithIntervalPromotion enables implicit promotion of T to Interval<T>.
// CQF flag: --enable-interval-promotion. Default off.
func WithIntervalPromotion(v bool) Option {
	return func(o *translator.Options) { o.EnableIntervalPromotion = v }
}

// WithLibrarySource sets a custom LibrarySource for resolving include declarations.
func WithLibrarySource(src LibrarySource) Option {
	return func(o *translator.Options) { o.LibrarySource = src }
}

// WithCQLOptionsFile loads a cql-options.json file and applies its settings.
// Path may be absolute or relative. Errors are deferred to Translate time
// via a panic-free option: if loading fails, the option is a no-op (parse
// error surfaces as a translation error when the caller exercises the file).
// Use cqloptions.LoadFile directly if you need explicit error handling.
func WithCQLOptionsFile(path string) Option {
	return func(o *translator.Options) {
		f, err := cqloptions.LoadFile(path)
		if err != nil {
			return
		}
		*o = f.Apply(o)
	}
}

// WithCQLOptions applies a parsed cqloptions.File onto the translator options.
func WithCQLOptions(f *cqloptions.File) Option {
	return func(o *translator.Options) {
		if f == nil {
			return
		}
		*o = f.Apply(o)
	}
}

// TranslateResult holds the ELM library and diagnostics from a translation.
type TranslateResult struct {
	Library     *elm.Library
	Diagnostics []translator.Diagnostic
}

// MarshalJSON returns the ELM JSON envelope: {"library": {...}}.
func (r *TranslateResult) MarshalJSON() ([]byte, error) {
	env := elm.LibraryEnvelope{Library: r.Library}
	return json.Marshal(&env)
}

// XMLBytes returns the ELM XML bytes.
func (r *TranslateResult) XMLBytes() ([]byte, error) {
	return elm.MarshalXML(r.Library, elm.XMLOptions{Indent: true})
}

// Validate performs a pragmatic structural check on serialized ELM bytes.
// format must be "xml" or "json". This is not a full XSD validation —
// use a libxml2-based pipeline for that — but it catches well-formedness
// errors, wrong root elements, and missing namespaces.
func Validate(data []byte, format string) error {
	return elm.Validate(data, format)
}

// Translate parses CQL source and translates it to an ELM library.
func Translate(src []byte, sourceName string, opts ...Option) (*TranslateResult, error) {
	parseResult, err := parser.ParseBytes(src, sourceName)
	if err != nil {
		return nil, err
	}

	tOpts := translator.DefaultOptions()
	for _, o := range opts {
		o(&tOpts)
	}

	t := translator.New(tOpts)
	t.SetSourceText(string(src))
	transResult := t.Translate(parseResult.Library, sourceName)

	result := &TranslateResult{
		Library:     transResult.Library,
		Diagnostics: transResult.Diagnostics,
	}

	// Surface parse diagnostics as translator diagnostics
	for _, d := range parseResult.Diagnostics {
		sev := "error"
		if d.Severity == parser.Warning {
			sev = "warning"
		}
		result.Diagnostics = append(result.Diagnostics, translator.Diagnostic{
			Severity: sev,
			Message:  d.Message,
			Locator:  d.Loc.String(),
		})
	}

	return result, nil
}

// FirelyJSON returns the ELM as fully type-discriminated JSON — the shape the
// JAXB/MOXy `elm-json` writer produces, which is what appears inside FHIR
// Library resources and what the Firely CQL SDK deserializes.
//
// The cql-to-elm CLI omits `"type"` on declaration, container and clause nodes
// because their position determines what they are. That is echo-elm's default
// output, since CLI parity is the compatibility target; a deserializer that
// dispatches on `"type"` needs this form instead.
func (r *TranslateResult) FirelyJSON() ([]byte, error) {
	plain, err := r.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return elm.AddTypeDiscriminators(plain)
}
