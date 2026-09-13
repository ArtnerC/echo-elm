package parity

// Public helpers for golden tests.

// NormalizeForGolden re-marshals JSON with sorted keys and strips volatile
// fields (translator version/options/diagnostic annotations) so a golden
// comparison stays stable across translator versions.
func NormalizeForGolden(s string) string {
	return normalizeJSON(s)
}

// SimpleDiff returns a naive line-by-line diff summary for use in test
// failure messages.
func SimpleDiff(want, got string) string {
	return simpleDiff(want, got)
}

// NormalizeBundleShape is NormalizeForGolden for reference ELM that came out of
// a FHIR Library.content[] attachment rather than the cql-to-elm CLI. The two
// writers disagree about empty collections, implied type discriminators and
// signatureLevel — none of which carry meaning — so both sides are reduced to
// the lean CLI shape before comparison.
//
// Use it only for bundle-sourced references. On the CLI path those fields are
// compared, and must be.
func NormalizeBundleShape(s string) string {
	return normalizeShape(s, true)
}
