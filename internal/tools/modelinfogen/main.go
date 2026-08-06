// modelinfogen generates internal/typesystem/fhir_modelinfo_gen.go from the
// authoritative FHIR ModelInfo that the CQF toolchain itself compiles against.
//
// The ModelInfo XML ships inside the cqframework "quick" JAR, which
// `task parity:jar` installs under tools/coursier/. Run from the repo root:
//
//	task generate:modelinfo
//
// Regenerate after bumping the pinned CQF version; the output is committed so
// that building echo-elm never requires the JARs.
package main

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"go/format"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	modelInfoEntry = "org/hl7/fhir/fhir-modelinfo-4.0.1.xml"
	outputPath     = "internal/typesystem/fhir_modelinfo_gen.go"
)

// quickJarGlobs locates the quick JAR in the Coursier cache, newest CQF first.
var quickJarGlobs = []string{
	"tools/coursier/cache/**/org/cqframework/quick/*/quick-*.jar",
	"tools/coursier/cache/**/info/cqframework/quick/*/quick-*.jar",
}

// ---------------------------------------------------------------------------
// ModelInfo XML schema (the subset we consume)
// ---------------------------------------------------------------------------

type modelInfo struct {
	TypeInfos []typeInfo `xml:"typeInfo"`
}

type typeInfo struct {
	Name            string    `xml:"name,attr"`
	Namespace       string    `xml:"namespace,attr"`
	BaseType        string    `xml:"baseType,attr"`
	Retrievable     string    `xml:"retrievable,attr"`
	PrimaryCodePath string    `xml:"primaryCodePath,attr"`
	Elements        []element `xml:"element"`
}

type element struct {
	Name string `xml:"name,attr"`
	// ElementType is set when the type is given as an attribute on <element>.
	ElementType string `xml:"elementType,attr"`
	// Specifier is set when the type is given as a child <elementTypeSpecifier>.
	Specifier *typeSpecifier `xml:"elementTypeSpecifier"`
}

type typeSpecifier struct {
	XSIType     string `xml:"http://www.w3.org/2001/XMLSchema-instance type,attr"`
	ElementType string `xml:"elementType,attr"`
	Name        string `xml:"name,attr"`
}

// resolve returns the element's declared type and whether it is list-valued.
// A choice element ([x] in FHIR) has no single type, so ok is false and the
// property is omitted — matching CQF, which leaves a choice property uncoerced.
func (e element) resolve() (typeName string, isList, ok bool) {
	if e.ElementType != "" {
		return e.ElementType, false, true
	}
	if e.Specifier == nil {
		return "", false, false
	}
	switch e.Specifier.XSIType {
	case "ListTypeSpecifier":
		if e.Specifier.ElementType == "" {
			return "", false, false
		}
		return e.Specifier.ElementType, true, true
	case "NamedTypeSpecifier":
		if e.Specifier.ElementType != "" {
			return e.Specifier.ElementType, false, true
		}
		if e.Specifier.Name != "" {
			return "FHIR." + e.Specifier.Name, false, true
		}
		return "", false, false
	default: // ChoiceTypeSpecifier and anything else we do not model
		return "", false, false
	}
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "modelinfogen: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	jarPath, err := findQuickJar()
	if err != nil {
		return err
	}
	fmt.Printf("Reading %s from %s\n", modelInfoEntry, jarPath)

	raw, err := readJarEntry(jarPath, modelInfoEntry)
	if err != nil {
		return err
	}

	var mi modelInfo
	if err := xml.Unmarshal(raw, &mi); err != nil {
		return fmt.Errorf("parse model info: %w", err)
	}

	tables := build(mi)
	src, err := render(tables, filepath.Base(jarPath))
	if err != nil {
		return err
	}
	if err := os.WriteFile(outputPath, src, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", outputPath, err)
	}

	fmt.Printf("Wrote %s\n", outputPath)
	fmt.Printf("  %d property types\n", len(tables.propertyType))
	fmt.Printf("  %d list-valued properties\n", len(tables.listProperty))
	fmt.Printf("  %d primitive/binding value types\n", len(tables.valueType))
	fmt.Printf("  %d primary code paths\n", len(tables.primaryCodePath))
	return nil
}

type tables struct {
	propertyType    map[string]string
	listProperty    map[string]bool
	valueType       map[string]string
	primaryCodePath map[string]string
}

func build(mi modelInfo) tables {
	t := tables{
		propertyType:    map[string]string{},
		listProperty:    map[string]bool{},
		valueType:       map[string]string{},
		primaryCodePath: map[string]string{},
	}

	for _, ti := range mi.TypeInfos {
		if ti.Namespace != "FHIR" || ti.Name == "" {
			continue
		}
		if ti.PrimaryCodePath != "" && ti.Retrievable == "true" {
			t.primaryCodePath[ti.Name] = ti.PrimaryCodePath
		}

		// A type whose sole element is `value` of a System type is a FHIR
		// primitive (FHIR.code) or a binding/enum wrapper (FHIR.AdministrativeGender).
		// Both convert through FHIRHelpers based on that System type.
		if len(ti.Elements) == 1 && ti.Elements[0].Name == "value" {
			if st := ti.Elements[0].ElementType; strings.HasPrefix(st, "System.") {
				t.valueType[ti.Name] = strings.TrimPrefix(st, "System.")
			}
		}

		for _, el := range ti.Elements {
			typeName, isList, ok := el.resolve()
			if !ok || el.Name == "" {
				continue
			}
			// Only FHIR-namespaced types participate in coercion; a System type
			// is already a CQL value and needs none.
			if !strings.HasPrefix(typeName, "FHIR.") {
				continue
			}
			key := ti.Name + "." + el.Name
			t.propertyType[key] = strings.TrimPrefix(typeName, "FHIR.")
			if isList {
				t.listProperty[key] = true
			}
		}
	}
	return t
}

func render(t tables, jarName string) ([]byte, error) {
	var b strings.Builder

	fmt.Fprintf(&b, `// Code generated by internal/tools/modelinfogen. DO NOT EDIT.
//
// Source: %s from %s
// Regenerate with: task generate:modelinfo

package typesystem

// FHIRPropertyType maps "TypeName.propertyName" to the declared FHIR type of that
// property, verbatim from the ModelInfo and without the "FHIR." namespace prefix.
//
// Binding types are kept as declared rather than reduced to their underlying
// primitive: CQF records the bound name in the conversion signature, so
// Patient.gender is "AdministrativeGender", not "code". Use FHIRPrimitiveValueType
// to reach the System type a value actually converts through.
//
// Choice elements ([x] in FHIR) have no single type and are omitted, which leaves
// them uncoerced — the behavior CQF exhibits for e.g. Patient.deceased.
var FHIRPropertyType = map[string]string{
`, modelInfoEntry, jarName)
	writeStringMap(&b, t.propertyType)

	b.WriteString(`}

// FHIRListProperty holds every "TypeName.propertyName" whose element is
// list-valued. A list of FHIR primitives is not converted in place: CQF lifts the
// conversion into a per-element query, so echo-elm leaves these uncoerced rather
// than wrapping the list itself.
var FHIRListProperty = map[string]bool{
`)
	keys := make([]string, 0, len(t.listProperty))
	for k := range t.listProperty {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&b, "\t%q: true,\n", k)
	}

	b.WriteString(`}

// FHIRPrimitiveValueType maps a FHIR primitive or binding type to the System type
// of its "value" element — the type a FHIRHelpers conversion produces. It covers
// both real primitives (code, dateTime) and enum bindings (AdministrativeGender).
var FHIRPrimitiveValueType = map[string]string{
`)
	writeStringMap(&b, t.valueType)

	b.WriteString(`}

// FHIRPrimaryCodePath maps a retrievable FHIR type to its primaryCodePath — the
// default codeProperty for a Retrieve carrying a terminology filter but no
// explicit code path.
var FHIRPrimaryCodePath = map[string]string{
`)
	writeStringMap(&b, t.primaryCodePath)
	b.WriteString("}\n")

	src, err := format.Source([]byte(b.String()))
	if err != nil {
		return nil, fmt.Errorf("gofmt generated source: %w", err)
	}
	return src, nil
}

func writeStringMap(b *strings.Builder, m map[string]string) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(b, "\t%q: %q,\n", k, m[k])
	}
}

// ---------------------------------------------------------------------------
// JAR access
// ---------------------------------------------------------------------------

func findQuickJar() (string, error) {
	var candidates []string
	for _, pattern := range quickJarGlobs {
		matches, err := globStar(pattern)
		if err != nil {
			return "", err
		}
		candidates = append(candidates, matches...)
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("no cqframework quick JAR found under tools/coursier — run: task parity:jar")
	}
	// Newest coordinate last alphabetically is good enough for x.y.z under a
	// single major line; prefer org.cqframework (4.x) over info.cqframework (3.x).
	sort.Strings(candidates)
	for i := len(candidates) - 1; i >= 0; i-- {
		if strings.Contains(filepath.ToSlash(candidates[i]), "/org/cqframework/") {
			return candidates[i], nil
		}
	}
	return candidates[len(candidates)-1], nil
}

// globStar expands a pattern containing a single "**" path segment.
func globStar(pattern string) ([]string, error) {
	idx := strings.Index(pattern, "**/")
	if idx < 0 {
		return filepath.Glob(filepath.FromSlash(pattern))
	}
	root := filepath.FromSlash(pattern[:idx])
	rest := pattern[idx+len("**/"):]

	var out []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			//nolint:nilerr // an unreadable subtree just contributes no matches
			return nil
		}
		matches, mErr := filepath.Glob(filepath.Join(path, filepath.FromSlash(rest)))
		if mErr != nil {
			return mErr
		}
		out = append(out, matches...)
		return nil
	})
	if os.IsNotExist(err) {
		return nil, nil
	}
	return out, err
}

func readJarEntry(jarPath, entry string) ([]byte, error) {
	zr, err := zip.OpenReader(jarPath)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", jarPath, err)
	}
	defer func() { _ = zr.Close() }()

	for _, f := range zr.File {
		if f.Name != entry {
			continue
		}
		return readZipEntry(f, entry)
	}
	return nil, fmt.Errorf("%s not found in %s", entry, jarPath)
}

// readZipEntry reads one entry, closing it before returning so the reader is
// not held open by a deferred close inside the caller's loop.
func readZipEntry(f *zip.File, entry string) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("open entry %s: %w", entry, err)
	}
	defer func() { _ = rc.Close() }()
	return io.ReadAll(rc)
}
