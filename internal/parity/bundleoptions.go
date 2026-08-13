package parity

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// DeclaredOptions is the translator configuration a bundled ELM library records
// about its own compilation, read from its CqlToElmInfo annotation.
type DeclaredOptions struct {
	// Options is the raw comma-separated translatorOptions string, kept so a
	// mismatch can be reported in the same words the bundle used.
	Options string
	// SignatureLevel is the declared signature level, or "" when absent.
	SignatureLevel string
	// Flags is Options split and trimmed, in declaration order.
	Flags []string
}

// optionProfileFields maps a CQF translator option name to the corpus
// translatorOptions key that turns it on. Anything absent from this table is an
// option echo-elm's profile format has no equivalent for; see unmappedOptions.
var optionProfileFields = map[string]string{
	"EnableAnnotations":       "enableAnnotations",
	"EnableLocators":          "enableLocators",
	"EnableResultTypes":       "enableResultTypes",
	"EnableDetailedErrors":    "enableDetailedErrors",
	"DisableListDemotion":     "disableListDemotion",
	"DisableListPromotion":    "disableListPromotion",
	"DisableListTraversal":    "disableListTraversal",
	"DisableMethodInvocation": "disableMethodInvocation",
	"RequireFromKeyword":      "requireFromKeyword",
	"EnableIntervalDemotion":  "enableIntervalDemotion",
	"EnableIntervalPromotion": "enableIntervalPromotion",
}

// ReadDeclaredOptions extracts the translator options an ELM document records in
// its CqlToElmInfo annotation. ok is false when the document declares none,
// which is not an error: a hand-written or stripped ELM file simply says nothing
// about how it was produced.
func ReadDeclaredOptions(elmJSON []byte) (DeclaredOptions, bool) {
	var doc struct {
		Library struct {
			Annotation []struct {
				Type             string `json:"type"`
				TranslatorOption string `json:"translatorOptions"`
				SignatureLevel   string `json:"signatureLevel"`
			} `json:"annotation"`
		} `json:"library"`
	}
	if err := json.Unmarshal(elmJSON, &doc); err != nil {
		return DeclaredOptions{}, false
	}
	for _, a := range doc.Library.Annotation {
		if a.Type != "CqlToElmInfo" {
			continue
		}
		return DeclaredOptions{
			Options:        a.TranslatorOption,
			SignatureLevel: a.SignatureLevel,
			Flags:          splitOptions(a.TranslatorOption),
		}, true
	}
	return DeclaredOptions{}, false
}

// splitOptions splits a comma-separated translatorOptions string, dropping
// empty entries so that "" yields no flags rather than one blank one.
func splitOptions(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// Profile renders the declared options as an OptionProfile, so a bundle is
// compared against the configuration it says it was built with rather than one
// the harness invented.
//
// Options echo-elm's profile format cannot express are reported by
// unmappedOptions rather than silently ignored — quietly dropping one would put
// the comparison back in exactly the state this is meant to prevent.
func (d DeclaredOptions) Profile(description string) (OptionProfile, []string) {
	opts := map[string]interface{}{}
	var unmapped []string
	for _, flag := range d.Flags {
		key, ok := optionProfileFields[flag]
		if !ok {
			unmapped = append(unmapped, flag)
			continue
		}
		opts[key] = true
	}
	// CQF writes signatureLevel as its own annotation field rather than as a
	// translator option, and omits it only when it was never set.
	if d.SignatureLevel != "" {
		opts["signatureLevel"] = d.SignatureLevel
	}
	return OptionProfile{Description: description, TranslatorOptions: opts}, unmapped
}

// String renders the declared options for an error message.
func (d DeclaredOptions) String() string {
	if d.Options == "" {
		return fmt.Sprintf("(no translator options), signatureLevel=%s", orNone(d.SignatureLevel))
	}
	return fmt.Sprintf("%s, signatureLevel=%s", d.Options, orNone(d.SignatureLevel))
}

func orNone(s string) string {
	if s == "" {
		return "None"
	}
	return s
}

// Equal reports whether two declarations describe the same compilation, order of
// the option list aside.
func (d DeclaredOptions) Equal(o DeclaredOptions) bool {
	if orNone(d.SignatureLevel) != orNone(o.SignatureLevel) {
		return false
	}
	a, b := append([]string(nil), d.Flags...), append([]string(nil), o.Flags...)
	sort.Strings(a)
	sort.Strings(b)
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
