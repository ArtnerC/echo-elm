package main

import (
	"flag"

	"github.com/artnerc/echo-elm/internal/translator"
)

// translatorFlags holds the translator options shared by every subcommand that
// translates CQL, so `bundle translate` accepts the same set as `translate`
// rather than a divergent subset.
type translatorFlags struct {
	cqfMode bool

	annotations             bool
	locators                bool
	resultTypes             bool
	debugMode               bool
	strict                  bool
	disableListDemotion     bool
	disableListPromotion    bool
	disableListTraversal    bool
	disableMethodInvocation bool
	requireFromKeyword      bool
	enableIntervalDemotion  bool
	enableIntervalPromotion bool
	sigLevel                string
	compatLevel             string
}

// registerTranslatorFlags declares the shared flags on fs. In CQF mode the
// annotation and locator defaults are off and the signature level is None,
// matching the upstream CLI; otherwise echo-elm's own defaults apply.
func registerTranslatorFlags(fs *flag.FlagSet, cqfMode bool) *translatorFlags {
	t := &translatorFlags{cqfMode: cqfMode}

	// Defaults differ by mode to match each mode's natural behavior.
	fs.BoolVar(&t.annotations, "annotations", !cqfMode, "Emit ELM annotations")
	fs.BoolVar(&t.locators, "locators", !cqfMode, "Emit source locators")
	fs.BoolVar(&t.disableListDemotion, "disable-list-demotion", false, "Disable implicit list demotion")
	fs.BoolVar(&t.disableListPromotion, "disable-list-promotion", false, "Disable implicit list promotion")
	fs.BoolVar(&t.disableListTraversal, "disable-list-traversal", false, "Disable implicit list traversal")
	fs.BoolVar(&t.disableMethodInvocation, "disable-method-invocation", false, "Disable method-style invocation syntax")
	fs.BoolVar(&t.requireFromKeyword, "require-from-keyword", false, "Require explicit 'from' in queries")
	fs.BoolVar(&t.enableIntervalDemotion, "enable-interval-demotion", false, "Enable implicit interval demotion")
	fs.BoolVar(&t.enableIntervalPromotion, "enable-interval-promotion", false, "Enable implicit interval promotion")
	fs.BoolVar(&t.resultTypes, "result-types", false, "Record the resolved type on every ELM node")
	fs.BoolVar(&t.debugMode, "debug", false, "Shorthand for --annotations --locators --result-types")
	fs.BoolVar(&t.strict, "strict", false, "Strict mode (disables list traversal, demotion, promotion, and method invocation)")

	defaultSig := "Overloads"
	if cqfMode {
		defaultSig = "None"
	}
	fs.StringVar(&t.sigLevel, "signatures", defaultSig, "Signature level: None|Differing|Overloads|All")
	fs.StringVar(&t.compatLevel, "compatibility-level", "1.5", "Compatibility level: 1.3|1.4|1.5")
	return t
}

// resolveShorthands applies the flags that expand into others. Call after Parse.
func (t *translatorFlags) resolveShorthands() {
	// --debug is CQF's shorthand for annotations + locators + result types.
	if t.debugMode {
		t.annotations, t.locators, t.resultTypes = true, true, true
	}
	if t.strict {
		// CQF's --strict expands to exactly these four; it does not imply
		// --require-from-keyword.
		t.disableListTraversal = true
		t.disableListDemotion = true
		t.disableListPromotion = true
		t.disableMethodInvocation = true
	}
}

// options builds the translator options these flags describe.
func (t *translatorFlags) options() translator.Options {
	t.resolveShorthands()

	opts := translator.DefaultOptions()
	if t.cqfMode {
		opts = translator.CQFDefaultOptions()
	}
	opts.EnableAnnotations = t.annotations
	opts.EnableLocators = t.locators
	opts.EnableResultTypes = t.resultTypes
	opts.DisableListDemotion = t.disableListDemotion
	opts.DisableListPromotion = t.disableListPromotion
	opts.DisableListTraversal = t.disableListTraversal
	opts.DisableMethodInvocation = t.disableMethodInvocation
	opts.RequireFromKeyword = t.requireFromKeyword
	opts.EnableIntervalDemotion = t.enableIntervalDemotion
	opts.EnableIntervalPromotion = t.enableIntervalPromotion
	opts.SignatureLevel = t.sigLevel
	opts.CompatibilityLevel = t.compatLevel
	return opts
}
