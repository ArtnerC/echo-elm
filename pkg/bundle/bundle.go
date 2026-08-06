// Package bundle reads CQL and ELM out of FHIR Bundle documents, and writes
// generated ELM back into them.
//
// HEDIS and other FHIR-based measure programs distribute a measure as a Bundle
// of Library resources, each carrying its CQL and pre-compiled ELM as
// base64-encoded content attachments. A reporting bundle typically holds the
// measure library plus every shared dependency, so the libraries resolve each
// other's `include` declarations from inside the bundle rather than from disk —
// which is what LibrarySource provides.
package bundle

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// Content types used for the CQL and ELM attachments on a Library resource.
const (
	ContentTypeCQL     = "text/cql"
	ContentTypeELMJSON = "application/elm+json"
	ContentTypeELMXML  = "application/elm+xml"
)

// Bundle is a FHIR Bundle document. The raw JSON is retained so that writing a
// modified bundle preserves every field this package does not model.
type Bundle struct {
	raw     map[string]any
	entries []*entry
}

// entry is one bundle entry that holds a Library resource.
type entry struct {
	resource map[string]any
	library  Library
}

// Library is the CQL and ELM carried by one Library resource in a bundle.
type Library struct {
	// Name and Version identify the library, matching the `library` declaration
	// in its CQL. Either may be empty if the resource omits it.
	Name    string
	Version string
	// CQLSource is the decoded text/cql attachment, nil when the resource has none.
	CQLSource []byte
	// ReferenceELM is the decoded application/elm+json attachment, nil when absent.
	ReferenceELM []byte
}

// Load reads and parses a FHIR Bundle.
func Load(r io.Reader) (*Bundle, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("bundle: read: %w", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("bundle: parse JSON: %w", err)
	}
	if rt, _ := raw["resourceType"].(string); rt != "Bundle" {
		return nil, fmt.Errorf("bundle: resourceType is %q, want Bundle", rt)
	}

	b := &Bundle{raw: raw}
	rawEntries, _ := raw["entry"].([]any)
	for _, e := range rawEntries {
		em, ok := e.(map[string]any)
		if !ok {
			continue
		}
		res, ok := em["resource"].(map[string]any)
		if !ok {
			continue
		}
		if rt, _ := res["resourceType"].(string); rt != "Library" {
			continue
		}
		lib, err := readLibrary(res)
		if err != nil {
			return nil, err
		}
		b.entries = append(b.entries, &entry{resource: res, library: lib})
	}
	return b, nil
}

// readLibrary decodes the content attachments of one Library resource.
func readLibrary(res map[string]any) (Library, error) {
	lib := Library{}
	lib.Name, _ = res["name"].(string)
	lib.Version, _ = res["version"].(string)

	contents, _ := res["content"].([]any)
	for _, c := range contents {
		cm, ok := c.(map[string]any)
		if !ok {
			continue
		}
		encoded, _ := cm["data"].(string)
		if encoded == "" {
			continue
		}
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return lib, fmt.Errorf("bundle: library %q: decode %s attachment: %w",
				lib.Name, cm["contentType"], err)
		}
		switch ct, _ := cm["contentType"].(string); ct {
		case ContentTypeCQL:
			lib.CQLSource = decoded
		case ContentTypeELMJSON:
			lib.ReferenceELM = decoded
		}
	}
	return lib, nil
}

// Libraries returns every Library resource in the bundle that carries CQL,
// ordered by name so that output is stable across runs.
func (b *Bundle) Libraries() []Library {
	out := make([]Library, 0, len(b.entries))
	for _, e := range b.entries {
		if len(e.library.CQLSource) > 0 {
			out = append(out, e.library)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Version < out[j].Version
	})
	return out
}

// LibrarySource resolves include declarations against the bundle's own
// libraries, so a measure translates without its dependencies being written to
// disk first.
func (b *Bundle) LibrarySource() *Source {
	byName := make(map[string][]Library)
	for _, e := range b.entries {
		if len(e.library.CQLSource) == 0 {
			continue
		}
		byName[e.library.Name] = append(byName[e.library.Name], e.library)
	}
	return &Source{byName: byName}
}

// Source resolves CQL libraries from a bundle's in-memory contents.
// It satisfies the resolver.LibrarySource interface.
type Source struct {
	byName map[string][]Library
}

// GetLibrarySource returns the CQL for the named library. An empty version
// matches any; a requested version matches exactly, falling back to the sole
// library of that name when the bundle carries only one.
func (s *Source) GetLibrarySource(name, version string) ([]byte, bool, error) {
	candidates := s.byName[name]
	if len(candidates) == 0 {
		return nil, false, nil
	}
	if version != "" {
		for i := range candidates {
			if candidates[i].Version == version {
				return candidates[i].CQLSource, true, nil
			}
		}
		// A bundle pins one version per library; prefer resolving over failing
		// when only one is present, matching how DirSource falls back.
		if len(candidates) == 1 {
			return candidates[0].CQLSource, true, nil
		}
		return nil, false, nil
	}
	return candidates[0].CQLSource, true, nil
}

// SetELM replaces (or adds) the application/elm+json attachment of the named
// library. An empty version matches the sole library of that name.
func (b *Bundle) SetELM(name, version string, elmJSON []byte) error {
	return b.SetContent(name, version, ContentTypeELMJSON, elmJSON)
}

// SetContent replaces (or adds) the attachment of the given content type on the
// named library. An empty version matches the sole library of that name.
func (b *Bundle) SetContent(name, version, contentType string, data []byte) error {
	target := b.findEntry(name, version)
	if target == nil {
		return fmt.Errorf("bundle: no Library resource named %q", libraryLabel(name, version))
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	contents, _ := target.resource["content"].([]any)
	for _, c := range contents {
		cm, ok := c.(map[string]any)
		if !ok {
			continue
		}
		if ct, _ := cm["contentType"].(string); ct == contentType {
			cm["data"] = encoded
			target.rememberContent(contentType, data)
			return nil
		}
	}
	target.resource["content"] = append(contents, map[string]any{
		"contentType": contentType,
		"data":        encoded,
	})
	target.rememberContent(contentType, data)
	return nil
}

// rememberContent keeps the decoded view in step with the raw resource.
func (e *entry) rememberContent(contentType string, data []byte) {
	switch contentType {
	case ContentTypeCQL:
		e.library.CQLSource = data
	case ContentTypeELMJSON:
		e.library.ReferenceELM = data
	}
}

// findEntry locates the entry for a library by name and optional version.
func (b *Bundle) findEntry(name, version string) *entry {
	var sole *entry
	matches := 0
	for _, e := range b.entries {
		if e.library.Name != name {
			continue
		}
		if version != "" && e.library.Version == version {
			return e
		}
		matches++
		sole = e
	}
	if version == "" && matches == 1 {
		return sole
	}
	if version != "" && matches == 1 {
		// One library of that name, a different version: still unambiguous.
		return sole
	}
	return nil
}

// Marshal serialises the bundle, including any ELM written back into it.
func (b *Bundle) Marshal() ([]byte, error) {
	out, err := json.MarshalIndent(b.raw, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("bundle: marshal: %w", err)
	}
	return out, nil
}

// libraryLabel renders a name/version pair for diagnostics.
func libraryLabel(name, version string) string {
	if version == "" {
		return name
	}
	return name + "|" + version
}

// FileName returns the conventional file name for a library's CQL, matching the
// layout DirSource expects so an extracted bundle resolves includes on disk.
func (l *Library) FileName() string {
	if l.Version == "" {
		return sanitize(l.Name) + ".cql"
	}
	return sanitize(l.Name) + "-" + sanitize(l.Version) + ".cql"
}

// sanitize strips path separators so a library name cannot escape its directory.
func sanitize(s string) string {
	return strings.NewReplacer("/", "_", `\`, "_", "..", "_").Replace(s)
}
