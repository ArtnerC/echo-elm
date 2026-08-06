package parity

import (
	"fmt"
	"os"
	"path/filepath"
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
// not meaningful. Pass the single profile the bundle was compiled with.
func MaterializeBundle(b *bundle.Bundle, dir, profileName string) (Config, error) {
	if profileName == "" {
		profileName = "default"
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
		OptionProfiles: map[string]OptionProfile{
			profileName: {Description: "Options the bundle's ELM was compiled with"},
		},
		Fixtures: fixtures,
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
