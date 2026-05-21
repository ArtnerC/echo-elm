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
