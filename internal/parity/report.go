package parity

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RunReport is the JSON/Markdown report written per parity run.
type RunReport struct {
	ID         string          `json:"id"`
	CQFVersion string          `json:"cqfVersion"`
	RunAt      time.Time       `json:"runAt"`
	Total      int             `json:"total"`
	Match      int             `json:"match"`
	DifferJSON int             `json:"differJson"`
	Errors     int             `json:"errors"`
	Fixtures   []FixtureResult `json:"fixtures"`
}

// WriteReport serialises results to report.json and report.md under outDir.
// outDir is created if it does not exist.
func WriteReport(outDir, id, cqfVersion string, results []FixtureResult) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", outDir, err)
	}

	summary := Summary(results)
	report := RunReport{
		ID:         id,
		CQFVersion: cqfVersion,
		RunAt:      time.Now().UTC(),
		Total:      len(results),
		Match:      summary[StatusMatch],
		DifferJSON: summary[StatusDifferJSON],
		Errors:     summary[StatusEchoError] + summary[StatusUpstreamError],
		Fixtures:   results,
	}

	// Write report.json
	jsonPath := filepath.Join(outDir, "report.json")
	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}
	if err := os.WriteFile(jsonPath, b, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", jsonPath, err)
	}

	// Write report.md
	mdPath := filepath.Join(outDir, "report.md")
	md := renderMarkdown(report)
	if err := os.WriteFile(mdPath, []byte(md), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", mdPath, err)
	}

	return nil
}

//nolint:gocritic // hugeParam: internal helper; large struct copy acceptable here
func renderMarkdown(r RunReport) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "# Parity Report — cqframework %s\n\n", r.CQFVersion)
	fmt.Fprintf(&sb, "**Run ID:** `%s`  \n", r.ID)
	fmt.Fprintf(&sb, "**Run At:** %s  \n\n", r.RunAt.Format(time.RFC3339))
	fmt.Fprintf(&sb, "## Summary\n\n")
	fmt.Fprintf(&sb, "| Status | Count |\n|---|---|\n")
	fmt.Fprintf(&sb, "| ✓ Match | %d |\n", r.Match)
	fmt.Fprintf(&sb, "| ≠ Differ (JSON) | %d |\n", r.DifferJSON)
	fmt.Fprintf(&sb, "| ✗ Errors | %d |\n", r.Errors)
	fmt.Fprintf(&sb, "| **Total** | **%d** |\n\n", r.Total)

	fmt.Fprintf(&sb, "## Fixtures\n\n")
	fmt.Fprintf(&sb, "| Fixture | Status | Duration |\n|---|---|---|\n")
	for i := range r.Fixtures {
		f := &r.Fixtures[i]
		icon := statusMDIcon(f.Status)
		fmt.Fprintf(&sb, "| `%s` | %s %s | %s |\n",
			f.Fixture, icon, string(f.Status), f.Duration.Round(time.Millisecond))
	}
	fmt.Fprintf(&sb, "\n")

	// Append diffs for failing fixtures.
	var hasDiffs bool
	for i := range r.Fixtures {
		f := &r.Fixtures[i]
		if f.Status == StatusDifferJSON && f.Diff != "" {
			if !hasDiffs {
				fmt.Fprintf(&sb, "## Diffs\n\n")
				hasDiffs = true
			}
			fmt.Fprintf(&sb, "### `%s`\n\n```diff\n%s\n```\n\n", f.Fixture, f.Diff)
		}
		if f.Error != "" {
			if !hasDiffs {
				fmt.Fprintf(&sb, "## Errors\n\n")
				hasDiffs = true
			}
			fmt.Fprintf(&sb, "### `%s`\n\n```\n%s\n```\n\n", f.Fixture, f.Error)
		}
	}

	return sb.String()
}

func statusMDIcon(s Status) string {
	switch s {
	case StatusMatch:
		return "✓"
	case StatusDifferJSON:
		return "≠"
	default:
		return "✗"
	}
}
