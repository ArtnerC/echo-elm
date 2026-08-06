// Package parity implements the CQFramework parity test harness.
package parity

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// OptionProfile defines a named set of translator options and their CQF CLI flag equivalents.
type OptionProfile struct {
	Description       string                 `yaml:"description"`
	CLIFlags          []string               `yaml:"cliFlags"`
	TranslatorOptions map[string]interface{} `yaml:"translatorOptions"`
	// PendingImplementation marks profiles whose corresponding echo-elm feature is
	// not yet implemented. TestGoldenCorpus skips these profiles rather than failing.
	PendingImplementation bool `yaml:"pendingImplementation"`
}

// Fixture is a single test entry from corpus.yaml.
type Fixture struct {
	Path           string `yaml:"path"`
	Description    string `yaml:"description"`
	ExpectedStatus string `yaml:"expectedStatus"`
	// Profiles is either the string "all" or a list of profile names from option_profiles.
	// If absent, defaults to ["default"].
	RawProfiles interface{} `yaml:"profiles"`
	// ExcludeProfiles lists profile names to exclude even when Profiles is "all".
	// Use this when the upstream CQF CLI itself produces no output for a given
	// fixture+profile combination (upstream-error), so no golden can exist.
	ExcludeProfiles []string `yaml:"excludeProfiles"`
	// VersionDivergent marks a fixture whose CQF output legitimately differs
	// between the pinned upstream versions. Golden generation skips it during
	// the cross-version equality check and takes the newest version's output,
	// instead of failing the whole run. Document the divergence in the fixture.
	VersionDivergent bool     `yaml:"versionDivergent"`
	Tags             []string `yaml:"tags"`

	// Options is retained for backward compatibility but superseded by Profiles.
	Options map[string]string `yaml:"options"`
}

// ProfileNames returns the resolved list of profile names for this fixture,
// expanding "all" to every profile in the provided profiles map and then
// removing any ExcludeProfiles entries.
func (fix *Fixture) ProfileNames(allProfiles map[string]OptionProfile) []string {
	excluded := make(map[string]bool, len(fix.ExcludeProfiles))
	for _, e := range fix.ExcludeProfiles {
		excluded[e] = true
	}

	filter := func(names []string) []string {
		if len(excluded) == 0 {
			return names
		}
		out := names[:0:len(names)]
		for _, n := range names {
			if !excluded[n] {
				out = append(out, n)
			}
		}
		return out
	}

	if fix.RawProfiles == nil {
		return filter([]string{"default"})
	}
	switch v := fix.RawProfiles.(type) {
	case string:
		if v == "all" {
			names := make([]string, 0, len(allProfiles))
			// Return in deterministic order matching the YAML key order.
			// gopkg.in/yaml.v3 preserves insertion order for maps decoded into
			// map[string]T, but map iteration order in Go is random — so we
			// collect from the profiles map and sort.
			for k := range allProfiles {
				names = append(names, k)
			}
			return filter(names)
		}
		return filter([]string{v})
	case []interface{}:
		names := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				names = append(names, s)
			}
		}
		return filter(names)
	}
	return []string{"default"}
}

// Corpus holds all fixtures from a corpus.yaml file.
type Corpus struct {
	Root           string
	OptionProfiles map[string]OptionProfile `yaml:"option_profiles"`
	Fixtures       []Fixture                `yaml:"fixtures"`
}

// LoadCorpus reads and parses corpus.yaml from the given directory.
func LoadCorpus(dir string) (*Corpus, error) {
	f, err := os.Open(fmt.Sprintf("%s/corpus.yaml", dir))
	if err != nil {
		return nil, fmt.Errorf("open corpus.yaml: %w", err)
	}
	defer func() { _ = f.Close() }()

	var c Corpus
	if err := yaml.NewDecoder(f).Decode(&c); err != nil {
		return nil, fmt.Errorf("parse corpus.yaml: %w", err)
	}
	c.Root = dir

	// Ensure "default" profile exists even if corpus.yaml omits option_profiles.
	if c.OptionProfiles == nil {
		c.OptionProfiles = map[string]OptionProfile{
			"default": {Description: "CQF defaults"},
		}
	}
	return &c, nil
}

// HasTag reports whether the fixture has the given tag.
func (fix *Fixture) HasTag(tag string) bool {
	for _, t := range fix.Tags {
		if t == tag {
			return true
		}
	}
	return false
}
