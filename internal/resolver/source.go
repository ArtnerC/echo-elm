// Package resolver provides implementations of the LibrarySource interface
// for loading CQL library source files from various storage backends.
package resolver

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// LibrarySource resolves CQL library source by name and optional version.
// This mirrors cqframework's LibrarySourceProvider interface semantics.
type LibrarySource interface {
	// GetLibrarySource returns the CQL source bytes for the named library.
	// version may be empty to match any version.
	// Returns (source, true, nil) on success; (nil, false, nil) when not found;
	// (nil, false, err) on I/O error.
	GetLibrarySource(name, version string) ([]byte, bool, error)
}

// -----------------------------------------------------------------------
// DirSource — filesystem directory
// -----------------------------------------------------------------------

// DirSource resolves CQL libraries from a directory on the local filesystem.
// It looks for files in this order:
//  1. <dir>/<name>-<version>.cql  (when version is non-empty)
//  2. <dir>/<name>.cql
type DirSource struct {
	Dir string
}

// NewDirSource creates a DirSource that reads from dir.
func NewDirSource(dir string) *DirSource {
	return &DirSource{Dir: dir}
}

func (d *DirSource) GetLibrarySource(name, version string) ([]byte, bool, error) {
	candidates := libraryFilenames(name, version)
	for _, fname := range candidates {
		path := filepath.Join(d.Dir, fname)
		b, err := os.ReadFile(path)
		if err == nil {
			return b, true, nil
		}
		if !os.IsNotExist(err) {
			return nil, false, fmt.Errorf("resolver: reading %s: %w", path, err)
		}
	}
	return nil, false, nil
}

// -----------------------------------------------------------------------
// FSSource — io/fs.FS
// -----------------------------------------------------------------------

// FSSource resolves CQL libraries from an fs.FS (useful for embedded files
// or test fixtures without a real filesystem).
// The optional Dir field is the base path within the FS.
type FSSource struct {
	FS  fs.FS
	Dir string
}

// NewFSSource creates an FSSource rooted at the given fs.FS.
// Pass dir="" to search from the FS root, or a subdirectory path.
func NewFSSource(fsys fs.FS, dir string) *FSSource {
	return &FSSource{FS: fsys, Dir: dir}
}

func (f *FSSource) GetLibrarySource(name, version string) ([]byte, bool, error) {
	base := f.Dir
	candidates := libraryFilenames(name, version)
	for _, fname := range candidates {
		var path string
		if base != "" {
			path = base + "/" + fname
		} else {
			path = fname
		}
		b, err := fs.ReadFile(f.FS, path)
		if err == nil {
			return b, true, nil
		}
		// fs.ErrNotExist and path-error on not-exist are both acceptable.
	}
	return nil, false, nil
}

// -----------------------------------------------------------------------
// MultiSource — chain of sources
// -----------------------------------------------------------------------

// MultiSource chains multiple LibrarySources, returning the first match.
// Useful when libraries live in multiple directories (e.g., project lib + FHIR helpers).
type MultiSource struct {
	Sources []LibrarySource
}

// NewMultiSource creates a MultiSource from the given sources, checked in order.
func NewMultiSource(sources ...LibrarySource) *MultiSource {
	return &MultiSource{Sources: sources}
}

func (m *MultiSource) GetLibrarySource(name, version string) ([]byte, bool, error) {
	for _, s := range m.Sources {
		b, ok, err := s.GetLibrarySource(name, version)
		if err != nil {
			return nil, false, err
		}
		if ok {
			return b, true, nil
		}
	}
	return nil, false, nil
}

// -----------------------------------------------------------------------
// MapSource — in-memory map (for tests)
// -----------------------------------------------------------------------

// MapSource resolves libraries from an in-memory map keyed by "Name|version"
// or just "Name" (for version-independent lookup).
// Useful in tests to inject CQL source without touching the filesystem.
type MapSource struct {
	Libraries map[string][]byte
}

// NewMapSource creates a MapSource from a name→source map.
// Keys may be plain names ("FHIRHelpers") or versioned ("FHIRHelpers|4.0.1").
func NewMapSource(libs map[string][]byte) *MapSource {
	return &MapSource{Libraries: libs}
}

func (m *MapSource) GetLibrarySource(name, version string) ([]byte, bool, error) {
	// Versioned key first.
	if version != "" {
		if b, ok := m.Libraries[name+"|"+version]; ok {
			return b, true, nil
		}
	}
	// Unversioned key.
	if b, ok := m.Libraries[name]; ok {
		return b, true, nil
	}
	return nil, false, nil
}

// -----------------------------------------------------------------------
// helpers
// -----------------------------------------------------------------------

// libraryFilenames returns the candidate filenames to search for a library.
func libraryFilenames(name, version string) []string {
	if version != "" {
		return []string{
			name + "-" + version + ".cql",
			name + ".cql",
		}
	}
	return []string{name + ".cql"}
}
