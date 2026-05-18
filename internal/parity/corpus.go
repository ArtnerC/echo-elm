// Package parity implements the CQFramework parity test harness.
package parity

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Fixture is a single test entry from corpus.yaml.
type Fixture struct {
	Path           string            `yaml:"path"`
	Description    string            `yaml:"description"`
	Options        map[string]string `yaml:"options"`
	ExpectedStatus string            `yaml:"expectedStatus"`
	Tags           []string          `yaml:"tags"`
}

// Corpus holds all fixtures from a corpus.yaml file.
type Corpus struct {
	Root     string
	Fixtures []Fixture `yaml:"fixtures"`
}

// LoadCorpus reads and parses corpus.yaml from the given directory.
func LoadCorpus(dir string) (*Corpus, error) {
	f, err := os.Open(fmt.Sprintf("%s/corpus.yaml", dir))
	if err != nil {
		return nil, fmt.Errorf("open corpus.yaml: %w", err)
	}
	defer f.Close()

	var c Corpus
	if err := yaml.NewDecoder(f).Decode(&c); err != nil {
		return nil, fmt.Errorf("parse corpus.yaml: %w", err)
	}
	c.Root = dir
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
