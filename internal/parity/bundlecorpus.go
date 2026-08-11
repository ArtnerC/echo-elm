package parity

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/artnerc/echo-elm/pkg/bundle"
)

// MaterializeBundle lays a FHIR Bundle out on disk as a parity corpus and
// returns the Config that runs against it.
//
// It writes each library's CQL into <dir>/corpus, a corpus.yaml describing them,
// and each library's bundled ELM into <dir>/ref/<profile>. The returned Config
// therefore takes its reference ELM from the bundle and resolves includes across
// the extracted libraries — no CQF JAR, and no manual extraction step.
//
// A bundle carries exactly one compiled ELM per library, produced with whatever
// options its publisher used, so comparing it against several option profiles is
// not meaningful.
//
// The profile is therefore not invented: each library's ELM records the options
// it was compiled with in its CqlToElmInfo annotation, and that declaration is
// authoritative. Inventing a profile instead is how a reference compiled with
// EnableLocators and EnableResultTypes ends up compared against a translation
// with neither, making every locator and resultType node mismatch by
// construction — thousands of differences that say nothing about the translator.
//
// Libraries that disagree with each other about their options are rejected
// rather than reconciled, since no single profile can be right for all of them.
func MaterializeBundle(b *bundle.Bundle, dir, profileName string) (Config, error) {
	if profileName == "" {
		profileName = "bundle"
	}
	declared, err := declaredBundleOptions(b)
	if err != nil {
		return Config{}, err
	}
	profile, unmapped := declared.Profile("Options the bundle's ELM declares it was compiled with")
	if len(unmapped) > 0 {
		return Config{}, fmt.Errorf(
			"parity: bundle declares translator option(s) %s that no corpus profile field maps to; "+
				"comparing against a profile that silently ignores them would report differences "+
				"caused by the harness, not the translator",
			strings.Join(unmapped, ", "))
	}
	corpusDir := filepath.Join(dir, "corpus")
	refDir := filepath.Join(dir, "ref", profileName)
	for _, d := range []string{corpusDir, refDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return Config{}, fmt.Errorf("parity: create %s: %w", d, err)
		}
	}

	libs := b.Libraries()
	fixtures := make([]Fixture, 0, len(libs))
	for i := range libs {
		lib := &libs[i]
		base := strings.TrimSuffix(lib.FileName(), ".cql")

		if err := os.WriteFile(filepath.Join(corpusDir, base+".cql"), lib.CQLSource, 0o644); err != nil {
			return Config{}, fmt.Errorf("parity: write %s: %w", base+".cql", err)
		}
		// A library with no bundled ELM has nothing to compare against; it is
		// still extracted so that other libraries can resolve includes to it,
		// but it does not become a fixture.
		if len(lib.ReferenceELM) == 0 {
			continue
		}
		if err := os.WriteFile(filepath.Join(refDir, base+".json"), lib.ReferenceELM, 0o644); err != nil {
			return Config{}, fmt.Errorf("parity: write %s: %w", base+".json", err)
		}
		fixtures = append(fixtures, Fixture{
			Path:           base + ".cql",
			Description:    libraryDescription(lib),
			ExpectedStatus: "success",
			RawProfiles:    []any{profileName},
			Tags:           []string{"bundle"},
		})
	}
	if len(fixtures) == 0 {
		return Config{}, fmt.Errorf("parity: bundle has no library carrying both CQL and ELM")
	}

	corpus := Corpus{
		OptionProfiles: map[string]OptionProfile{profileName: profile},
		Fixtures:       fixtures,
	}
	encoded, err := yaml.Marshal(&corpus)
	if err != nil {
		return Config{}, fmt.Errorf("parity: encode corpus.yaml: %w", err)
	}
	if err := os.WriteFile(filepath.Join(corpusDir, "corpus.yaml"), encoded, 0o644); err != nil {
		return Config{}, fmt.Errorf("parity: write corpus.yaml: %w", err)
	}

	return Config{
		CorpusDir:     corpusDir,
		RefDir:        filepath.Join(dir, "ref"),
		LibDir:        corpusDir,
		ProfileFilter: profileName,
	}, nil
}

// libraryDescription names a library for the parity report.
func libraryDescription(lib *bundle.Library) string {
	if lib.Version == "" {
		return "bundled library " + lib.Name
	}
	return "bundled library " + lib.Name + " " + lib.Version
}

// declaredBundleOptions returns the translator options the bundle's libraries
// say they were compiled with, requiring them to agree.
//
// A library carrying ELM but declaring no CqlToElmInfo annotation is skipped: it
// says nothing about its own compilation, so it constrains nothing. If no
// library declares anything, the zero value stands for CQF's own defaults, which
// is what an ELM document with no annotation is claiming.
func declaredBundleOptions(b *bundle.Bundle) (DeclaredOptions, error) {
	var found bool
	var agreed DeclaredOptions
	var agreedName string

	libs := b.Libraries()
	for i := range libs {
		lib := &libs[i]
		if len(lib.ReferenceELM) == 0 {
			continue
		}
		declared, ok := ReadDeclaredOptions(lib.ReferenceELM)
		if !ok {
			continue
		}
		if !found {
			agreed, agreedName, found = declared, lib.Name, true
			continue
		}
		if !agreed.Equal(declared) {
			return DeclaredOptions{}, fmt.Errorf(
				"parity: bundle libraries disagree about their compile options — "+
					"%s declares [%s] but %s declares [%s]; no single profile can be "+
					"correct for both, so split the bundle or compare them separately",
				agreedName, agreed, lib.Name, declared)
		}
	}
	return agreed, nil
}

// ValidateProfileAgainstBundle reports whether an explicitly requested profile
// matches what the bundle declares, so that `parity --bundle --profile X` fails
// loudly instead of producing a diff full of harness artifacts.
func ValidateProfileAgainstBundle(b *bundle.Bundle, profileName string, profile OptionProfile) error {
	declared, err := declaredBundleOptions(b)
	if err != nil {
		return err
	}
	want, unmapped := declared.Profile("")
	if len(unmapped) > 0 {
		return fmt.Errorf("parity: bundle declares unmappable translator option(s) %s",
			strings.Join(unmapped, ", "))
	}
	// Collect every mismatch and sort them: reporting one arbitrary key out of
	// several would be both unhelpful and non-reproducible, since ranging a map
	// picks a different one each run.
	var mismatches []string
	for key, wantVal := range want.TranslatorOptions {
		if got, ok := profile.TranslatorOptions[key]; !ok || got != wantVal {
			mismatches = append(mismatches, fmt.Sprintf("%s=%v (bundle needs %v)", key, got, wantVal))
		}
	}
	if len(mismatches) == 0 {
		return nil
	}
	sort.Strings(mismatches)
	return fmt.Errorf(
		"parity: bundle declares %s but profile %q translates with %s. "+
			"Refusing to compare mismatched profiles; omit --profile to derive it from the bundle",
		declared, profileName, strings.Join(mismatches, ", "))
}
