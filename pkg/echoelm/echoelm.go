// Package echoelm is the public API for the echo-elm CQL→ELM translator.
//
// See contracts/go-api.md for the full API surface specification.
package echoelm

import (
	"encoding/json"

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

// WithLibrarySource sets a custom LibrarySource for resolving include declarations.
func WithLibrarySource(src LibrarySource) Option {
	return func(o *translator.Options) { o.LibrarySource = src }
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

// MarshalXML returns the ELM XML bytes.
func (r *TranslateResult) MarshalXML() ([]byte, error) {
	return elm.MarshalXML(r.Library, elm.XMLOptions{Indent: true})
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

