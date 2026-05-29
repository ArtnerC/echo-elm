package translator

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/artnerc/echo-elm/internal/ast"
	"github.com/artnerc/echo-elm/internal/elm"
	"github.com/artnerc/echo-elm/internal/elmops"
	"github.com/artnerc/echo-elm/internal/resolver"
	"github.com/artnerc/echo-elm/internal/typesystem"
	"github.com/artnerc/echo-elm/internal/ucum"
)

// Version is the translator version string embedded in output.
const Version = "0.0.0-dev"

// Options controls translation behavior.
type Options struct {
	EnableAnnotations       bool
	EnableLocators          bool
	DisableListDemotion     bool
	DisableListPromotion    bool
	DisableListTraversal    bool
	DisableMethodInvocation bool
	RequireFromKeyword      bool
	// EnableIntervalDemotion allows implicit demotion of Interval<T> to T.
	// CQF flag: --enable-interval-demotion. Default off.
	EnableIntervalDemotion bool
	// EnableIntervalPromotion allows implicit promotion of T to Interval<T>.
	// CQF flag: --enable-interval-promotion. Default off.
	EnableIntervalPromotion bool
	ValidateUnits           bool
	// EnableResultTypes records type information on each ELM expression node.
	// CQF flag: --result-types. Implied by --debug.
	EnableResultTypes  bool
	CompatibilityLevel string
	SignatureLevel     string
	ErrorLevel         string
	TranslatorVersion  string

	// CQFMode emits cqframework-compatible output: annotation:[], signature:[],
	// and empty filter arrays on Retrieve nodes. Also forces translatorOptions:"".
	// Auto-set by `echo-elm cqf translate`.
	CQFMode bool

	// LibrarySource resolves CQL library includes by name and version.
	// When nil, includes are recorded in ELM output but not resolved.
	LibrarySource resolver.LibrarySource
}

// DefaultOptions returns the modern echo-elm defaults per spec.
func DefaultOptions() Options {
	return Options{
		EnableAnnotations:    false,
		EnableLocators:       true,
		DisableListDemotion:  false,
		DisableListPromotion: false,
		ValidateUnits:        true,
		CompatibilityLevel:   "1.5",
		SignatureLevel:       "Overloads",
		ErrorLevel:           "Info",
		TranslatorVersion:    Version,
		CQFMode:              false,
	}
}

// CQFDefaultOptions returns the options matching the cqframework CLI defaults.
// EnableAnnotations=false, EnableLocators=false, SignatureLevel="None", CQFMode=true.
func CQFDefaultOptions() Options {
	return Options{
		EnableAnnotations:    false,
		EnableLocators:       false,
		DisableListDemotion:  false,
		DisableListPromotion: false,
		ValidateUnits:        true,
		CompatibilityLevel:   "1.5",
		SignatureLevel:       "None",
		ErrorLevel:           "Info",
		TranslatorVersion:    Version,
		CQFMode:              true,
	}
}

// Diagnostic represents a translation-time message.
type Diagnostic struct {
	Severity string
	Message  string
	Locator  string
}

func (d Diagnostic) String() string {
	return fmt.Sprintf("%s:[%s] %s", d.Severity, d.Locator, d.Message)
}

// Result is the output of a translation.
type Result struct {
	Library     *elm.Library
	Diagnostics []Diagnostic
}

// symKind classifies what kind of definition an identifier resolves to,
// enabling correct ELM Ref node selection (ExpressionRef, ParameterRef, etc.).
type symKind uint8

const (
	symExpression symKind = iota
	symParameter
	symValueSet
	symCodeSystem
	symCode
	symConcept
)

// Translator is the core CQL→ELM translation engine.
type Translator struct {
	opts                 Options
	counter              int
	sourceName           string
	sourceText           string // original CQL source for annotation s-tree extraction
	lineOffsets          []int  // byte offsets of each line start in sourceText (1-indexed via [line-1])
	diags                []Diagnostic
	syms                 map[string]symKind         // symbol table built before statement pass
	paramTypes           map[string]string          // maps parameter name → ELM qualified type name (named types only)
	paramTypeSpecs       map[string]typeSpec        // maps parameter name → full typeSpec (named, list, interval)
	defTypeSpecs         map[string]typeSpec        // maps statement-def name → full typeSpec inferred from its body
	queryAliases         []map[string]bool          // stack of alias sets for current query scopes
	queryLetScopes       []map[string]bool          // stack of let-identifier sets for current query scopes
	queryAliasTypes      []map[string]string        // stack: alias name → FHIR resource type (e.g. "Encounter")
	queryAliasTypeSpecs  []map[string]typeSpec      // stack: alias name → element typeSpec (for sig inference)
	queryLetTypeSpecs    []map[string]typeSpec      // stack: let-id → inferred typeSpec for QueryLetRef inference
	modelsByAlias        map[string]string          // model local-identifier → model URI (populated per Translate call)
	primaryModelURI      string                     // URI of the first non-System declared model
	primaryModelName     string                     // original model name of the first non-System declared model (e.g. "QUICK", "FHIR")
	currentContextName   string                     // context being translated (for age function expansion)
	functionParamScope   map[string]bool            // set of operand (parameter) names in the current function body
	operandTypeSpecs     map[string]typeSpec        // function operand name → declared typeSpec, set during function body translation
	listNodeTypes        map[*elm.ListNode]typeSpec // typed-list literals: ListNode pointer → element typeSpec (does not serialize)
	fhirHelpersLocalName string                     // local identifier of included FHIRHelpers library, or "" if not included
	skipLocatorStamp     bool                       // when true, translateExpr skips stamping locator on its outer result (set by callees that placed locator on an inner node)
}

// New creates a new Translator with the given options.
//
//nolint:gocritic // hugeParam: Options is part of the public API; pointer would break callers
func New(opts Options) *Translator {
	return &Translator{opts: opts}
}

func (t *Translator) nextID() string {
	t.counter++
	return fmt.Sprintf("%d", t.counter)
}

// defID returns the next localId only when EnableAnnotations is active.
// Def-level nodes (UsingDef, IncludeDef, ParameterDef, StatementDef, etc.)
// must not emit localId when running without --annotations, matching CQF CLI behavior.
func (t *Translator) defID() string {
	if t.opts.EnableAnnotations {
		return t.nextID()
	}
	return ""
}

func locatorStr(loc ast.Interval) string {
	if loc.Start.Line == 0 {
		return ""
	}
	if loc.Start.Line == loc.Stop.Line && loc.Start.Column == loc.Stop.Column {
		return fmt.Sprintf("%d:%d", loc.Start.Line, loc.Start.Column)
	}
	return fmt.Sprintf("%d:%d-%d:%d",
		loc.Start.Line, loc.Start.Column,
		loc.Stop.Line, loc.Stop.Column)
}

// setLocator sets the Locator field on any ELM expression node via reflection.
// Used by the translateExpr wrapper to stamp AST-derived locator strings onto nodes.
func setLocator(e elm.Expression, loc string) {
	if loc == "" || e == nil {
		return
	}
	v := reflect.ValueOf(e)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return
	}
	f := v.FieldByName("Locator")
	if f.IsValid() && f.CanSet() && f.Kind() == reflect.String {
		f.SetString(loc)
	}
}

// checkUnit validates a CQL Quantity unit against UCUM when the
// ValidateUnits option is enabled. Following CQF behavior, an unparseable
// unit produces a Warning diagnostic but does not fail the translation.
// Empty units and CQL calendar-duration keywords are always accepted.
func (t *Translator) checkUnit(unit string, loc ast.Interval) {
	if !t.opts.ValidateUnits {
		return
	}
	if unit == "" || ucum.IsCQLTemporalKeyword(unit) {
		return
	}
	if err := ucum.Validate(unit); err != nil {
		t.diags = append(t.diags, Diagnostic{
			Severity: "Warning",
			Locator:  locatorStr(loc),
			Message:  fmt.Sprintf("Could not validate UCUM unit %q: %s", unit, err.Error()),
		})
	}
}

func accessLevelStr(level ast.AccessLevel) string {
	if level == ast.Private {
		return "Private"
	}
	return "Public"
}

// optionsString builds the translator options string for CqlToElmInfo.
// In CQF mode this is always empty, matching cqframework CLI behavior.
func (t *Translator) optionsString() string {
	if t.opts.CQFMode {
		return ""
	}
	var parts []string
	if t.opts.EnableAnnotations {
		parts = append(parts, "EnableAnnotations")
	}
	if t.opts.EnableLocators {
		parts = append(parts, "EnableLocators")
	}
	if t.opts.DisableListDemotion {
		parts = append(parts, "DisableListDemotion")
	}
	if t.opts.DisableListPromotion {
		parts = append(parts, "DisableListPromotion")
	}
	if t.opts.EnableIntervalDemotion {
		parts = append(parts, "EnableIntervalDemotion")
	}
	if t.opts.EnableIntervalPromotion {
		parts = append(parts, "EnableIntervalPromotion")
	}
	if t.opts.ValidateUnits {
		parts = append(parts, "ValidateUnits")
	}
	return strings.Join(parts, ",")
}

// cqfAnnotation returns a json.RawMessage for an empty annotation array
// when in CQF mode, nil otherwise.
func (t *Translator) cqfAnnotation() json.RawMessage {
	if t.opts.CQFMode {
		return json.RawMessage("[]")
	}
	return nil
}

// cqfEmptyArrayField returns an empty JSON array when in CQF mode, nil otherwise.
func (t *Translator) cqfEmptyArrayField() json.RawMessage {
	if t.opts.CQFMode {
		return cqfEmptyArray
	}
	return nil
}

// cqfEmptyArray is the JSON raw empty array used for CQF filter fields.
var cqfEmptyArray = json.RawMessage("[]")

// Translate converts an AST library to an ELM library.
// SetSourceText records the original CQL source so that annotation s-trees
// (EnableAnnotations) can extract the literal text spans for each definition.
// Safe to call before Translate.
func (t *Translator) SetSourceText(src string) {
	t.sourceText = src
	t.lineOffsets = computeLineOffsets(src)
}

// computeLineOffsets returns a slice such that lineOffsets[i] is the byte
// offset of the start of line i+1 in src (lines are 1-based externally).
func computeLineOffsets(src string) []int {
	offsets := []int{0}
	for i := 0; i < len(src); i++ {
		if src[i] == '\n' {
			offsets = append(offsets, i+1)
		}
	}
	return offsets
}

// posToOffset converts a 1-based line/column position to a byte offset in
// sourceText. Returns -1 when the position is invalid. Column is 1-based.
func (t *Translator) posToOffset(line, col int) int {
	if line <= 0 || line > len(t.lineOffsets) {
		return -1
	}
	off := t.lineOffsets[line-1] + (col - 1)
	if off < 0 || off > len(t.sourceText) {
		return -1
	}
	return off
}

// sourceSliceWithLeading returns the source text covering loc, extended
// backward to include any preceding contiguous comment-only lines (no blank
// line separator). Mirrors CQF's annotation source-text capture behaviour.
// Carriage returns are stripped to normalize CRLF input to LF (matching CQF).
func (t *Translator) sourceSliceWithLeading(loc ast.Interval) string {
	s := t.posToOffset(loc.Start.Line, loc.Start.Column)
	e := t.posToOffset(loc.Stop.Line, loc.Stop.Column+1)
	if s < 0 || e < 0 || e < s {
		return ""
	}
	s = t.extendStartForLeadingComments(s)
	return strings.ReplaceAll(t.sourceText[s:e], "\r", "")
}

// extendStartForLeadingComments returns the offset of the earliest comment
// character in the trivia run (whitespace + comments) immediately preceding
// the def's first non-trivia character at start. If the trivia contains no
// comment (just whitespace), returns start unchanged.
//
// This mirrors ANTLR HIDDEN-channel token attachment used by CQF: all hidden
// tokens between the previous non-hidden token and the def's first non-hidden
// token are considered the def's leading trivia. CQF strips the LEADING
// whitespace (before any comment) but preserves intra-trivia whitespace.
func (t *Translator) extendStartForLeadingComments(start int) int {
	src := t.sourceText
	if start <= 0 || start > len(src) {
		return start
	}
	// Forward-scan src[0:start] classifying chars; track the position just
	// after the last code character.
	afterLastCode := 0
	n := start
	i := 0
	for i < n {
		c := src[i]
		// Line comment: // ... \n
		if c == '/' && i+1 < len(src) && src[i+1] == '/' {
			for i < n && src[i] != '\n' {
				i++
			}
			continue
		}
		// Block comment: /* ... */
		if c == '/' && i+1 < len(src) && src[i+1] == '*' {
			i += 2
			for i+1 < len(src) && !(src[i] == '*' && src[i+1] == '/') {
				i++
			}
			if i+1 < len(src) {
				i += 2
			}
			continue
		}
		// String literal '...'
		if c == '\'' {
			afterLastCode = i + 1
			i++
			for i < n && src[i] != '\'' {
				if src[i] == '\\' && i+1 < len(src) {
					i++
				}
				afterLastCode = i + 1
				i++
			}
			if i < n {
				afterLastCode = i + 1
				i++
			}
			continue
		}
		// Quoted identifier "..."
		if c == '"' {
			afterLastCode = i + 1
			i++
			for i < n && src[i] != '"' {
				if src[i] == '\\' && i+1 < len(src) {
					i++
				}
				afterLastCode = i + 1
				i++
			}
			if i < n {
				afterLastCode = i + 1
				i++
			}
			continue
		}
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			i++
			continue
		}
		afterLastCode = i + 1
		i++
	}
	// Trivia is src[afterLastCode:start]. Find the first non-whitespace char.
	trim := afterLastCode
	for trim < start {
		c := src[trim]
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			trim++
			continue
		}
		break
	}
	if trim == start {
		return start
	}
	return trim
}

// srcAnnotationJSON returns the JSON for a single source-text Annotation entry
// covering loc (with leading comments extended). Returns nil when the slice
// is empty (e.g. missing source mapping).
func (t *Translator) srcAnnotationJSON(loc ast.Interval) json.RawMessage {
	if !t.opts.EnableAnnotations {
		return nil
	}
	text := t.sourceSliceWithLeading(loc)
	if text == "" {
		return nil
	}
	type sNode struct {
		Value []string `json:"value,omitempty"`
	}
	type sBlock struct {
		S []sNode `json:"s,omitempty"`
	}
	type annJSON struct {
		S    *sBlock `json:"s,omitempty"`
		Type string  `json:"type"`
	}
	ann := []annJSON{{Type: "Annotation", S: &sBlock{S: []sNode{{Value: []string{text}}}}}}
	b, err := json.Marshal(ann)
	if err != nil {
		return nil
	}
	return json.RawMessage(b)
}

// mergeAnnotations merges a base annotation (e.g. cqfAnnotation [] or @tag
// annotations) with the source-text annotation. When base is empty/nil, returns
// the source annotation. When source is nil, returns base unchanged.
func (t *Translator) mergeAnnotations(base, src json.RawMessage) json.RawMessage {
	if src == nil {
		return base
	}
	if base == nil || string(base) == "[]" || string(base) == "" {
		return src
	}
	var baseArr, srcArr []json.RawMessage
	if err := json.Unmarshal(base, &baseArr); err != nil {
		return src
	}
	if err := json.Unmarshal(src, &srcArr); err != nil {
		return base
	}
	baseArr = append(baseArr, srcArr...)
	b, err := json.Marshal(baseArr)
	if err != nil {
		return base
	}
	return json.RawMessage(b)
}

func (t *Translator) Translate(lib *ast.Library, sourceName string) *Result {
	t.sourceName = sourceName
	t.counter = 0
	t.diags = nil
	t.fhirHelpersLocalName = ""

	// Build symbol table so expression translation can emit the correct Ref types.
	t.syms = make(map[string]symKind)
	t.paramTypes = make(map[string]string)
	t.paramTypeSpecs = make(map[string]typeSpec)
	t.defTypeSpecs = make(map[string]typeSpec)

	// Build model alias → URI map so translateRetrieve can qualify data types.
	t.modelsByAlias = map[string]string{"System": typesystem.SystemURI}
	for _, u := range lib.Usings {
		localID := u.LocalName
		if localID == "" {
			localID = u.ModelName
		}
		t.modelsByAlias[localID] = t.modelURIVersioned(u.ModelName, u.Version)
	}

	// Determine the primary (non-System) model URI for context accessor and age functions.
	t.primaryModelURI = ""
	t.primaryModelName = ""
	for _, u := range lib.Usings {
		uri := t.modelURIVersioned(u.ModelName, u.Version)
		if uri != typesystem.SystemURI {
			t.primaryModelURI = uri
			t.primaryModelName = u.ModelName
			break
		}
	}

	for _, p := range lib.Parameters {
		t.syms[p.Name] = symParameter
		if p.ParameterType != nil {
			if nts, ok := (*p.ParameterType).(*ast.NamedTypeSpecifier); ok {
				t.paramTypes[p.Name] = resolveTypeName(nts.Name)
			}
			if ts := astTypeSpecToTypeSpec(*p.ParameterType); ts != nil {
				t.paramTypeSpecs[p.Name] = ts
			}
		}
	}
	for _, vs := range lib.Valuesets {
		t.syms[vs.Name] = symValueSet
	}
	for _, cs := range lib.Codesystems {
		t.syms[cs.Name] = symCodeSystem
	}
	for _, c := range lib.Codes {
		t.syms[c.Name] = symCode
	}
	for _, con := range lib.Concepts {
		t.syms[con.Name] = symConcept
	}
	// Statements come last so they don't shadow the above.
	for _, s := range lib.Statements {
		if _, already := t.syms[s.Name]; !already {
			t.syms[s.Name] = symExpression
		}
	}

	result := &Result{}
	out := &elm.Library{
		SchemaIdentifier: &elm.VersionedIdentifier{
			ID:      "urn:hl7-org:elm",
			Version: "r1",
		},
	}

	// Library identifier — always emit (upstream always emits "identifier": {})
	if lib.Name != nil {
		out.Identifier = elm.VersionedIdentifier{
			ID:      lib.Name.Name,
			Version: lib.Name.Version,
		}
	}

	// CqlToElmInfo annotation
	info := &elm.CqlToElmInfo{
		TranslatorVersion:  t.opts.TranslatorVersion,
		TranslatorOptions:  t.optionsString(),
		SignatureLevel:     t.opts.SignatureLevel,
		CompatibilityLevel: t.opts.CompatibilityLevel,
	}
	infoJSON, err := json.Marshal(info)
	if err == nil {
		out.Annotation = append(out.Annotation, json.RawMessage(infoJSON))
	}

	// Library-level source-text annotation (EnableAnnotations).
	// CQF emits an Annotation entry whose s-tree text covers the library
	// declaration header (e.g. "library Foo version '1.0.0'") when there is
	// at least one additional definition after the library line.
	hasMore := len(lib.Usings) > 0 || len(lib.Includes) > 0 ||
		len(lib.Codesystems) > 0 || len(lib.Valuesets) > 0 ||
		len(lib.Codes) > 0 || len(lib.Concepts) > 0 ||
		len(lib.Parameters) > 0 || len(lib.Contexts) > 0 ||
		len(lib.Statements) > 0
	if t.opts.EnableAnnotations && lib.Name != nil && lib.Name.Name != "" && hasMore {
		// CQF extends the library annotation to include any leading comments
		// preceding the "library" keyword. We construct the canonical text
		// (used by the normalizer's flattened comparison) by combining the
		// preceding-comment leading text with the canonical "library X" form.
		// When lib.Name has no Loc (no source position) we synthesize the text.
		var text string
		if libLoc := lib.Name.Loc(); libLoc.Start.Line > 0 {
			text = t.sourceSliceWithLeading(libLoc)
		}
		if text == "" {
			text = "library " + lib.Name.Name
			if lib.Name.Version != "" {
				text += " version '" + lib.Name.Version + "'"
			}
		}
		type sNode struct {
			Value []string `json:"value,omitempty"`
		}
		type sBlock struct {
			S []sNode `json:"s,omitempty"`
		}
		type annJSON struct {
			S    *sBlock `json:"s,omitempty"`
			Type string  `json:"type"`
		}
		ann := annJSON{Type: "Annotation", S: &sBlock{S: []sNode{{Value: []string{text}}}}}
		if b, mErr := json.Marshal(ann); mErr == nil {
			out.Annotation = append(out.Annotation, json.RawMessage(b))
		}
	}

	// Usings: always add implicit System first
	usings := &elm.UsingDefs{}
	usings.Def = append(usings.Def, &elm.UsingDef{
		LocalIdentifier: "System",
		URI:             typesystem.SystemURI,
		Annotation:      t.cqfAnnotation(),
	})
	for _, u := range lib.Usings {
		ud := &elm.UsingDef{
			LocalID:         t.defID(),
			LocalIdentifier: u.LocalName,
			URI:             t.modelURIVersioned(u.ModelName, u.Version),
			Version:         u.Version,
			Annotation:      t.mergeAnnotations(t.cqfAnnotation(), t.srcAnnotationJSON(u.Loc())),
		}
		if t.opts.EnableLocators {
			ud.Locator = locatorStr(u.Loc())
		}
		usings.Def = append(usings.Def, ud)
	}
	out.Usings = usings

	// Includes
	if len(lib.Includes) > 0 {
		includes := &elm.IncludeDefs{}
		for _, inc := range lib.Includes {
			localID := inc.LocalName
			if localID == "" {
				// CQL spec: default local identifier = last component of the path.
				if i := strings.LastIndex(inc.Path, "."); i >= 0 {
					localID = inc.Path[i+1:]
				} else {
					localID = inc.Path
				}
			}
			// Detect FHIRHelpers inclusion so we can apply implicit property coercion.
			if localID == "FHIRHelpers" || strings.HasSuffix(inc.Path, "FHIRHelpers") {
				t.fhirHelpersLocalName = localID
			}
			id := &elm.IncludeDef{
				LocalID:         t.defID(),
				LocalIdentifier: localID,
				Path:            inc.Path,
				Version:         inc.Version,
				Annotation:      t.mergeAnnotations(t.cqfAnnotation(), t.srcAnnotationJSON(inc.Loc())),
			}
			if t.opts.EnableLocators {
				id.Locator = locatorStr(inc.Loc())
			}
			includes.Def = append(includes.Def, id)
		}
		out.Includes = includes
	}

	// CodeSystems
	if len(lib.Codesystems) > 0 {
		css := &elm.CodeSystemDefs{}
		for _, cs := range lib.Codesystems {
			csd := &elm.CodeSystemDef{
				LocalID:     t.defID(),
				Annotation:  t.mergeAnnotations(t.cqfAnnotation(), t.srcAnnotationJSON(cs.Loc())),
				Name:        cs.Name,
				ID:          cs.ID,
				Version:     cs.Version,
				AccessLevel: accessLevelStr(cs.AccessLevel),
			}
			if t.opts.EnableLocators {
				csd.Locator = locatorStr(cs.Loc())
			}
			css.Def = append(css.Def, csd)
		}
		out.CodeSystems = css
	}

	// ValueSets
	if len(lib.Valuesets) > 0 {
		vss := &elm.ValueSetDefs{}
		for _, vs := range lib.Valuesets {
			vsd := &elm.ValueSetDef{
				LocalID:     t.defID(),
				Annotation:  t.mergeAnnotations(t.cqfAnnotation(), t.srcAnnotationJSON(vs.Loc())),
				Name:        vs.Name,
				ID:          vs.ID,
				Version:     vs.Version,
				AccessLevel: accessLevelStr(vs.AccessLevel),
				CodeSystems: json.RawMessage("[]"),
			}
			if t.opts.EnableLocators {
				vsd.Locator = locatorStr(vs.Loc())
			}
			vss.Def = append(vss.Def, vsd)
		}
		out.ValueSets = vss
	}

	// Codes
	if len(lib.Codes) > 0 {
		codes := &elm.CodeDefs{}
		for _, c := range lib.Codes {
			csRef := &elm.CodeSystemDefinitionRef{
				Annotation: t.cqfAnnotation(),
				Name:       c.SystemName,
			}
			if t.opts.EnableLocators {
				csRef.Locator = locatorStr(c.SystemLocator)
			}
			cd := &elm.CodeDef{
				LocalID:     t.defID(),
				Annotation:  t.mergeAnnotations(t.cqfAnnotation(), t.srcAnnotationJSON(c.Loc())),
				Name:        c.Name,
				ID:          c.Code,
				Display:     c.Display,
				AccessLevel: accessLevelStr(c.AccessLevel),
				CodeSystem:  csRef,
			}
			if t.opts.EnableLocators {
				cd.Locator = locatorStr(c.Loc())
			}
			codes.Def = append(codes.Def, cd)
		}
		out.Codes = codes
	}

	// Concepts
	if len(lib.Concepts) > 0 {
		concepts := &elm.ConceptDefs{}
		for _, con := range lib.Concepts {
			cond := &elm.ConceptDef{
				LocalID:     t.defID(),
				Name:        con.Name,
				Display:     con.Display,
				AccessLevel: accessLevelStr(con.AccessLevel),
			}
			if t.opts.EnableLocators {
				cond.Locator = locatorStr(con.Loc())
			}
			for _, codeName := range con.Codes {
				cond.Code = append(cond.Code, &elm.CodeRef{Name: codeName})
			}
			concepts.Def = append(concepts.Def, cond)
		}
		out.Concepts = concepts
	}

	// Parameters
	if len(lib.Parameters) > 0 {
		params := &elm.ParameterDefs{}
		for _, p := range lib.Parameters {
			pd := &elm.ParameterDef{
				LocalID:     t.defID(),
				Name:        p.Name,
				AccessLevel: accessLevelStr(p.AccessLevel),
				Annotation:  t.mergeAnnotations(t.cqfAnnotation(), t.srcAnnotationJSON(p.Loc())),
			}
			if t.opts.EnableLocators {
				pd.Locator = locatorStr(p.Loc())
			}
			if p.ParameterType != nil {
				pd.ParameterTypeSpecifier = t.translateTypeSpecifier(*p.ParameterType)
			}
			if p.Default != nil {
				pd.Default = t.translateExpr(p.Default)
			}
			params.Def = append(params.Def, pd)
		}
		out.Parameters = params
	}

	// Contexts section — emit all declared contexts in declaration order (deduplicated).
	// The contexts block is an ELM R1 / CQL 1.5 addition; suppress it for 1.4 compatibility.
	compLevel := t.opts.CompatibilityLevel
	emitContexts := compLevel == "" || compLevel >= "1.5"
	if emitContexts && len(lib.Contexts) > 0 {
		seen := map[string]bool{}
		var contextDefs []*elm.ContextDef
		for _, ctx := range lib.Contexts {
			if !seen[ctx.Name] {
				seen[ctx.Name] = true
				cd := &elm.ContextDef{
					Name:       ctx.Name,
					Annotation: t.cqfAnnotation(),
				}
				if t.opts.EnableLocators {
					cd.Locator = locatorStr(ctx.Loc())
				}
				contextDefs = append(contextDefs, cd)
			}
		}
		out.Contexts = &elm.ContextDefs{Def: contextDefs}
	}

	// Build set of explicitly defined names (to skip implicit accessor if shadowed).
	explicitNames := map[string]bool{}
	for _, s := range lib.Statements {
		explicitNames[s.Name] = true
	}

	// Statements — prepend implicit context accessor for each unique context that
	// (a) is not "Unfiltered", and (b) has no explicit definition with the same name.
	stmtDefs := make([]*elm.StatementDef, 0, len(lib.Contexts))
	seenAccessor := map[string]bool{}
	for _, ctx := range lib.Contexts {
		if ctx.Name == "Unfiltered" || explicitNames[ctx.Name] || seenAccessor[ctx.Name] {
			continue
		}
		seenAccessor[ctx.Name] = true
		stmtDefs = append(stmtDefs, t.buildContextAccessor(ctx.Name, lib))
	}

	if len(lib.Statements) > 0 || len(stmtDefs) > 0 {
		stmts := &elm.StatementDefs{Def: stmtDefs}
		for _, s := range lib.Statements {
			stmtCtx := s.Context
			if stmtCtx == "" {
				stmtCtx = "Unfiltered"
			}
			t.currentContextName = stmtCtx
			sd := &elm.StatementDef{
				LocalID:     t.defID(),
				Name:        s.Name,
				Context:     stmtCtx,
				AccessLevel: accessLevelStr(s.AccessLevel),
				IsFunction:  s.IsFunction,
				IsFluent:    s.IsFluent,
				Annotation:  t.buildStatementAnnotation(s),
			}
			if t.opts.EnableLocators {
				sd.Locator = locatorStr(s.Loc())
			}
			if s.ReturnType != nil {
				sd.ResultTypeSpecifier = t.translateTypeSpecifier(*s.ReturnType)
			}
			// Build function parameter scope before translating the body,
			// so IdentifierRef nodes matching operand names emit OperandRef.
			if s.IsFunction && len(s.Operands) > 0 {
				t.functionParamScope = make(map[string]bool, len(s.Operands))
				t.operandTypeSpecs = make(map[string]typeSpec, len(s.Operands))
				for _, op := range s.Operands {
					t.functionParamScope[op.Name] = true
					if op.Type != nil {
						if ts := astTypeSpecToTypeSpec(*op.Type); ts != nil {
							t.operandTypeSpecs[op.Name] = ts
						}
					}
				}
			} else {
				t.functionParamScope = nil
				t.operandTypeSpecs = nil
			}
			for _, op := range s.Operands {
				opd := &elm.OperandDef{Name: op.Name, Annotation: t.cqfAnnotation()}
				if op.Type != nil {
					opd.OperandTypeSpecifier = t.translateTypeSpecifier(*op.Type)
				}
				sd.Operand = append(sd.Operand, opd)
			}
			if s.Expression != nil {
				sd.Expression = t.translateExpr(s.Expression)
			} else {
				sd.Expression = &elm.NullNode{Annotation: t.cqfAnnotation()}
			}
			if !s.IsFunction {
				if ts := t.inferTypeSpec(sd.Expression); ts != nil {
					t.defTypeSpecs[s.Name] = ts
				}
			}
			t.functionParamScope = nil
			t.operandTypeSpecs = nil
			stmts.Def = append(stmts.Def, sd)
		}
		out.Statements = stmts
	}

	result.Library = out
	if len(t.diags) > 0 {
		result.Diagnostics = append(result.Diagnostics, t.diags...)
	}
	return result
}

// modelURIVersioned returns the ELM namespace URI for a model name + optional version.
// For versioned models like QDM, the URI includes the version (e.g. urn:healthit-gov:qdm:v5_3).
func (t *Translator) modelURIVersioned(name, version string) string {
	if name == "QDM" && version != "" {
		v := strings.ReplaceAll(version, ".", "_")
		return "urn:healthit-gov:qdm:v" + v
	}
	if uri, ok := typesystem.ModelURIByName[name]; ok {
		return uri
	}
	return name
}

// dataNamespaceForModelURI returns the ELM data-type namespace for a model URI.
// FHIR-profiled models (QICore/QUICK/USCore) use the FHIR base namespace for ELM data types.
func dataNamespaceForModelURI(modelName, modelURI string) string {
	if modelName == "QUICK" {
		return "http://hl7.org/fhir"
	}
	return modelURI
}

// templateIDForModel generates the ELM templateId for a given model name, URI and type.
// FHIR uses the StructureDefinition URL pattern. QUICK uses the legacy QI-Core pattern.
func templateIDForModel(modelName, modelURI, typeName string) string {
	switch modelName {
	case "QUICK":
		lc := strings.ToLower(typeName)
		return lc + "-qicore-qicore-" + lc
	default:
		if modelURI == "http://hl7.org/fhir" {
			return "http://hl7.org/fhir/StructureDefinition/" + typeName
		}
	}
	return ""
}

// qualifyDataType returns the model-namespace-qualified data type and optional templateId
// for a retrieve expression. The raw type name may be unqualified ("Encounter") or
// model-prefixed ("FHIR.Encounter"). When the primary declared model is FHIR, the
// templateId follows the StructureDefinition URL pattern.
func (t *Translator) qualifyDataType(rawType string) (dataType, templateID string) {
	if strings.HasPrefix(rawType, "{") {
		return rawType, "" // already namespace-qualified
	}

	var modelAlias, typeName string
	if before, after, ok := strings.Cut(rawType, "."); ok {
		modelAlias, typeName = before, after
	} else {
		typeName = before
	}

	var (
		modelName string
		uri       string
	)
	if modelAlias != "" {
		uri = t.modelsByAlias[modelAlias]
		modelName = modelAlias
	} else {
		modelName = t.primaryModelName
		uri = t.primaryModelURI
	}

	if uri == "" {
		return rawType, ""
	}

	// For models that profile FHIR (QUICK), use the FHIR data type namespace.
	dataNS := dataNamespaceForModelURI(modelName, uri)
	dt := "{" + dataNS + "}" + typeName
	templateID = templateIDForModel(modelName, dataNS, typeName)
	return dt, templateID
}

// buildContextAccessor creates the implicit singleton-from-retrieve statement for the declared context.
// For example, `context Patient` generates a Patient statement that retrieves the singleton Patient.
func (t *Translator) buildContextAccessor(contextName string, lib *ast.Library) *elm.StatementDef {
	// Determine the data type from the declared model.
	modelURI := t.primaryModelURI
	if modelURI == "" {
		modelURI = "http://hl7.org/fhir" // default to FHIR when no model declared
	}
	dataNS := dataNamespaceForModelURI(t.primaryModelName, modelURI)
	dataType := "{" + dataNS + "}" + contextName
	templateID := templateIDForModel(t.primaryModelName, dataNS, contextName)

	// Find the source locator from the context declaration (if available).
	ctxLocator := ""
	if t.opts.EnableLocators {
		for _, c := range lib.Contexts {
			if c.Name == contextName {
				ctxLocator = locatorStr(c.Loc())
				break
			}
		}
	}

	retrieve := &elm.RetrieveNode{
		DataType:    dataType,
		TemplateID:  templateID,
		Locator:     ctxLocator,
		Annotation:  t.cqfAnnotation(),
		Include:     t.cqfEmptyArrayField(),
		CodeFilter:  t.cqfEmptyArrayField(),
		DateFilter:  t.cqfEmptyArrayField(),
		OtherFilter: t.cqfEmptyArrayField(),
	}

	return &elm.StatementDef{
		Name:       contextName,
		Context:    contextName,
		Locator:    ctxLocator,
		Annotation: t.cqfAnnotation(),
		Expression: &elm.SingletonFromNode{
			Annotation: t.cqfAnnotation(),
			Signature:  t.cqfEmptyArrayField(),
			Operand:    retrieve,
		},
	}
}

// ageFunctionPrecision maps age function names to their ELM precision string.
var ageFunctionPrecision = map[string]string{
	"AgeInYears": "Year", "AgeInYearsAt": "Year",
	"AgeInMonths": "Month", "AgeInMonthsAt": "Month",
	"AgeInWeeks": "Week", "AgeInWeeksAt": "Week",
	"AgeInDays": "Day", "AgeInDaysAt": "Day",
	"AgeInHours": "Hour", "AgeInHoursAt": "Hour",
	"AgeInMinutes": "Minute", "AgeInMinutesAt": "Minute",
	"AgeInSeconds": "Second", "AgeInSecondsAt": "Second",
}

// expandAgeFunction expands age-related FHIRHelpers function calls to CalculateAge/CalculateAgeAt.
// Returns nil if the name is not a recognized age function.
// CQF expands these during library resolution; we replicate the pattern here.
func (t *Translator) expandAgeFunction(ann, sig json.RawMessage, name string, operands []ast.Expr) elm.Expression {
	precision, ok := ageFunctionPrecision[name]
	if !ok {
		return nil
	}
	isAt := strings.HasSuffix(name, "At")
	contextName := t.currentContextName
	if contextName == "" || contextName == "Unfiltered" {
		contextName = "Patient"
	}

	// Build the birthDate access expression. The access pattern depends on the model:
	// - FHIR: Property(path="birthDate.value", source=ExpressionRef("Patient")) wrapped in ToDateTime
	// - QUICK/other: Property(path="birthDate", source=ExpressionRef("Patient")) wrapped in ToDate
	contextRef := &elm.ExpressionRefNode{
		Annotation: t.cqfAnnotation(),
		Name:       contextName,
	}

	var birthDateExpr elm.Expression
	if t.primaryModelName == "FHIR" {
		propOperand := &elm.PropertyNode{
			Annotation: t.cqfAnnotation(),
			Path:       "birthDate.value",
			Source:     contextRef,
		}
		birthDateExpr = &elm.UnaryExpressionNode{
			Annotation: t.cqfAnnotation(),
			Signature:  t.buildSigFromTypes("ToDateTime", []typeSpec{dateTS}),
			Operator:   "ToDateTime",
			Operand:    propOperand,
		}
	} else {
		propOperand := &elm.PropertyNode{
			Annotation: t.cqfAnnotation(),
			Path:       "birthDate",
			Source:     contextRef,
		}
		birthDateExpr = &elm.UnaryExpressionNode{
			Annotation: t.cqfAnnotation(),
			Signature:  t.buildSigFromTypes("ToDate", []typeSpec{dateTimeTS}),
			Operator:   "ToDate",
			Operand:    propOperand,
		}
	}

	if isAt && len(operands) > 0 {
		dateArg := t.translateExpr(operands[0])
		ops := []elm.Expression{birthDateExpr, dateArg}
		return &elm.PrecisionOperatorNode{
			Annotation: ann,
			Signature:  t.computeSig("CalculateAgeAt", ops),
			Operator:   "CalculateAgeAt",
			Precision:  precision,
			Operand:    ops,
		}
	}

	return &elm.CalculateAgeNode{
		Annotation: ann,
		Signature:  t.computeSig("CalculateAge", []elm.Expression{birthDateExpr}),
		Precision:  precision,
		Operand:    birthDateExpr,
	}
}

// translateTypeSpecifier converts an AST TypeSpecifier to an ELM TypeSpecifier.
func (t *Translator) translateTypeSpecifier(ts ast.TypeSpecifier) elm.TypeSpecifier {
	if ts == nil {
		return nil
	}
	ann := t.cqfAnnotation()
	switch v := ts.(type) {
	case *ast.NamedTypeSpecifier:
		name := v.Name
		if v.Qualifier != "" {
			name = v.Qualifier + "." + v.Name
		}
		name = resolveTypeName(name)
		nts := &elm.NamedTypeSpecifier{Annotation: ann, Name: name}
		if t.opts.EnableLocators {
			nts.Locator = locatorStr(v.Loc())
		}
		return nts
	case *ast.IntervalTypeSpecifier:
		its := &elm.IntervalTypeSpecifier{
			Annotation: ann,
			PointType:  t.translateTypeSpecifier(v.PointType),
		}
		if t.opts.EnableLocators {
			its.Locator = locatorStr(v.Loc())
		}
		return its
	case *ast.ListTypeSpecifier:
		lts := &elm.ListTypeSpecifier{
			Annotation:  ann,
			ElementType: t.translateTypeSpecifier(v.ElementType),
		}
		if t.opts.EnableLocators {
			lts.Locator = locatorStr(v.Loc())
		}
		return lts
	case *ast.TupleTypeSpecifier:
		tts := &elm.TupleTypeSpecifier{Annotation: ann}
		if t.opts.EnableLocators {
			tts.Locator = locatorStr(v.Loc())
		}
		for _, elem := range v.Elements {
			ted := &elm.TupleElementDefinition{
				Annotation:  t.cqfAnnotation(),
				Name:        elem.Name,
				ElementType: t.translateTypeSpecifier(elem.Type),
			}
			if t.opts.EnableLocators {
				ted.Locator = locatorStr(elem.Loc())
			}
			tts.Element = append(tts.Element, ted)
		}
		return tts
	case *ast.ChoiceTypeSpecifier:
		cts := &elm.ChoiceTypeSpecifier{Annotation: ann}
		if t.opts.EnableLocators {
			cts.Locator = locatorStr(v.Loc())
		}
		for _, ct := range v.Types {
			cts.Choice = append(cts.Choice, t.translateTypeSpecifier(ct))
		}
		return cts
	default:
		return &elm.NamedTypeSpecifier{Annotation: ann, Name: typesystem.TypeAny}
	}
}

// resolveTypeName maps a bare CQL type name to its ELM-qualified form.
func resolveTypeName(name string) string {
	switch name {
	case "Boolean":
		return typesystem.TypeBoolean
	case "Integer":
		return typesystem.TypeInteger
	case "Long":
		return typesystem.TypeLong
	case "Decimal":
		return typesystem.TypeDecimal
	case "String":
		return typesystem.TypeString
	case "Date":
		return typesystem.TypeDate
	case "DateTime":
		return typesystem.TypeDateTime
	case "Time":
		return typesystem.TypeTime
	case "Quantity":
		return typesystem.TypeQuantity
	case "Ratio":
		return typesystem.TypeRatio
	case "Any":
		return typesystem.TypeAny
	case "Code":
		return typesystem.TypeCode
	case "Concept":
		return typesystem.TypeConcept
	case "Vocabulary":
		return typesystem.TypeVocabulary
	}
	// Already qualified (e.g. "FHIR.Patient") or unknown — keep as-is.
	return name
}

// unarySystemOps is the set of CQL built-in function names that map to single-operand
// ELM operator expressions (i.e., "operand" is a single object, not an array).
var unarySystemOps = map[string]bool{
	// Arithmetic
	"Abs": true, "Ceiling": true, "Floor": true, "Truncate": true,
	"Exp": true, "Ln": true, "Negate": true,
	"Successor": true, "Predecessor": true,
	"Sqrt": true,
	// Boolean
	"Not": true, "IsNull": true, "IsTrue": true, "IsFalse": true,
	// Collection
	"Exists": true, "SingletonFrom": true,
	"Distinct": true, "Flatten": true,
	"Length": true,
	// String
	"Upper": true, "Lower": true,
	// Type conversions
	"ToBoolean": true, "ToDate": true, "ToDateTime": true, "ToDecimal": true,
	"ToInteger": true, "ToLong": true, "ToList": true, "ToQuantity": true,
	"ToString": true, "ToTime": true, "ToRatio": true, "ToConcept": true,
	// Type conversion checks
	"ConvertsToBoolean": true, "ConvertsToDate": true, "ConvertsToDateTime": true,
	"ConvertsToDecimal": true, "ConvertsToInteger": true, "ConvertsToLong": true,
	"ConvertsToQuantity": true, "ConvertsToRatio": true, "ConvertsToString": true,
	"ConvertsToTime": true,
	// Interval operators
	"Width": true, "Start": true, "End": true,
}

// aggregateSystemOps are CQL built-in aggregate functions that map to ELM AggregateExpression
// nodes. ELM AggregateExpression uses "source" (single object) not "operand".
var aggregateSystemOps = map[string]bool{
	"Count": true, "Sum": true, "Product": true,
	"Min": true, "Max": true,
	"Avg": true, "Median": true, "Mode": true,
	"StdDev": true, "PopulationStdDev": true,
	"Variance": true, "PopulationVariance": true,
	"AllTrue": true, "AnyTrue": true,
	"First": true, "Last": true,
}

// decimalAggregateOps are aggregate functions that CQF always wraps their argument
// in a ToDecimal query to coerce List<T> → List<Decimal>.
var decimalAggregateOps = map[string]bool{
	"Avg": true, "Median": true, "StdDev": true, "PopulationStdDev": true,
	"Variance": true, "PopulationVariance": true,
}

// namedOperatorOps maps CQL built-in operator names with named operand fields
// to the JSON field names CQF emits for each operand slot, in order.
var namedOperatorOps = map[string][]string{
	"Substring":      {"stringToSub", "startIndex", "length"},
	"Combine":        {"source", "separator"},
	"Split":          {"stringToSplit", "separator"},
	"PositionOf":     {"pattern", "string"},
	"LastPositionOf": {"pattern", "string"},
	"Round":          {"operand", "precision"},
	"IndexOf":        {"source", "element"},
}

// binaryOperandOps are CQL built-in operators that emit "operand": [a, b]
// as a generic operator expression (no named fields).
var binaryOperandOps = map[string]bool{
	"StartsWith": true, "EndsWith": true, "Matches": true,
	"Power": true, "Log": true,
	"Contains": true,
}

// naryOperandOps are CQL built-in operators emitted as "operand": [a, b, ...]
// for any arity.
var naryOperandOps = map[string]bool{
	"ReplaceMatches": true,
}

// convertToOps maps a System named type to the ELM unary operator emitted by
// CQF for `convert x to <Type>` syntax. Built-in types are normalized to their
// operator-specific form instead of a generic Convert node.
var convertToOps = map[string]string{
	"Boolean":  "ToBoolean",
	"Integer":  "ToInteger",
	"Long":     "ToLong",
	"Decimal":  "ToDecimal",
	"String":   "ToString",
	"Date":     "ToDate",
	"DateTime": "ToDateTime",
	"Time":     "ToTime",
	"Quantity": "ToQuantity",
	"Ratio":    "ToRatio",
	"Concept":  "ToConcept",
}

// Init-time consistency check: every entry in the translator's local maps
// must be present (or be an explicit translator-internal alias) in the
// shared elmops registry. This keeps internal/elmops as the single source
// of truth and surfaces drift as a build failure.
func init() {
	for name := range unarySystemOps {
		if !elmops.IsUnary(name) {
			panic("translator: unarySystemOps drift; missing/wrong kind in elmops: " + name)
		}
	}
	for name := range aggregateSystemOps {
		if !elmops.IsAggregate(name) {
			panic("translator: aggregateSystemOps drift; missing/wrong kind in elmops: " + name)
		}
	}
	for name := range namedOperatorOps {
		if elmops.NamedFields(name) == nil {
			panic("translator: namedOperatorOps drift; missing NamedFields in elmops: " + name)
		}
	}
	for from, to := range convertToOps {
		if elmops.SystemConvertToOps[from] != to {
			panic("translator: convertToOps drift for " + from + " → " + to)
		}
	}
}

// intLiteral builds an Integer literal ELM node from an int value.
func (t *Translator) intLiteral(v int) elm.Expression {
	return &elm.LiteralNode{
		Annotation: t.cqfAnnotation(),
		ValueType:  typesystem.TypeInteger,
		Value:      fmt.Sprintf("%d", v),
	}
}

// parseDateLiteral converts an ast.DateLiteral value string (YYYY[-MM[-DD]]) to a DateNode.
func (t *Translator) parseDateLiteral(value string) *elm.DateNode {
	node := &elm.DateNode{
		Annotation: t.cqfAnnotation(),
		Signature:  t.cqfEmptyArrayField(),
	}
	parts := strings.Split(value, "-")
	if len(parts) >= 1 && parts[0] != "" {
		if y, err := strconv.Atoi(parts[0]); err == nil {
			node.Year = t.intLiteral(y)
		}
	}
	if len(parts) >= 2 {
		if m, err := strconv.Atoi(parts[1]); err == nil {
			node.Month = t.intLiteral(m)
		}
	}
	if len(parts) >= 3 {
		if d, err := strconv.Atoi(parts[2]); err == nil {
			node.Day = t.intLiteral(d)
		}
	}
	return node
}

// parseTimeParts parses a time string (HH[:MM[:SS[.mmm]]]) into component expressions.
// Returns nil for absent components.
func (t *Translator) parseTimeParts(s string) (hour, minute, second, millisecond elm.Expression) {
	parts := strings.SplitN(s, ":", 3)
	if len(parts) >= 1 && parts[0] != "" {
		if h, err := strconv.Atoi(parts[0]); err == nil {
			hour = t.intLiteral(h)
		}
	}
	if len(parts) >= 2 {
		if m, err := strconv.Atoi(parts[1]); err == nil {
			minute = t.intLiteral(m)
		}
	}
	if len(parts) >= 3 {
		secStr := parts[2]
		if dotIdx := strings.Index(secStr, "."); dotIdx >= 0 {
			milStr := secStr[dotIdx+1:]
			secStr = secStr[:dotIdx]
			if ms, err := strconv.Atoi(milStr); err == nil {
				millisecond = t.intLiteral(ms)
			}
		}
		if s2, err := strconv.Atoi(secStr); err == nil {
			second = t.intLiteral(s2)
		}
	}
	return
}

// parseTimeLiteral converts an ast.TimeLiteral value string (HH[:MM[:SS[.mmm]]]) to a TimeNode.
func (t *Translator) parseTimeLiteral(value string) *elm.TimeNode {
	node := &elm.TimeNode{
		Annotation: t.cqfAnnotation(),
		Signature:  t.cqfEmptyArrayField(),
	}
	node.Hour, node.Minute, node.Second, node.Millisecond = t.parseTimeParts(value)
	return node
}

// parseDateTimeLiteral converts an ast.DateTimeLiteral value string (YYYY[-MM[-DD]]T[time][tz]) to a DateTimeNode.
func (t *Translator) parseDateTimeLiteral(value string) *elm.DateTimeNode {
	node := &elm.DateTimeNode{
		Annotation: t.cqfAnnotation(),
		Signature:  t.cqfEmptyArrayField(),
	}

	tIdx := strings.Index(value, "T")
	datePart := value
	timePart := ""
	if tIdx >= 0 {
		datePart = value[:tIdx]
		timePart = value[tIdx+1:]
	}

	// Parse date components.
	if datePart != "" {
		dateParts := strings.Split(datePart, "-")
		if len(dateParts) >= 1 && dateParts[0] != "" {
			if y, err := strconv.Atoi(dateParts[0]); err == nil {
				node.Year = t.intLiteral(y)
			}
		}
		if len(dateParts) >= 2 {
			if m, err := strconv.Atoi(dateParts[1]); err == nil {
				node.Month = t.intLiteral(m)
			}
		}
		if len(dateParts) >= 3 {
			if d, err := strconv.Atoi(dateParts[2]); err == nil {
				node.Day = t.intLiteral(d)
			}
		}
	}

	// Parse time and timezone components.
	if timePart != "" {
		var tzStr string

		// Detect timezone suffix: trailing Z or ±HH:MM after the time digits.
		if strings.HasSuffix(timePart, "Z") {
			tzStr = "0.0"
			timePart = timePart[:len(timePart)-1]
		} else if plusIdx := strings.LastIndexAny(timePart, "+-"); plusIdx > 0 {
			sign := 1.0
			if timePart[plusIdx] == '-' {
				sign = -1.0
			}
			tzParts := strings.SplitN(timePart[plusIdx+1:], ":", 2)
			timePart = timePart[:plusIdx]
			var hrs, mins float64
			if len(tzParts) >= 1 {
				if h, err := strconv.ParseFloat(tzParts[0], 64); err == nil {
					hrs = h
				}
			}
			if len(tzParts) >= 2 {
				if m, err := strconv.ParseFloat(tzParts[1], 64); err == nil {
					mins = m / 60.0
				}
			}
			offset := sign * (hrs + mins)
			tzStr = strconv.FormatFloat(offset, 'f', -1, 64)
			if !strings.Contains(tzStr, ".") {
				tzStr += ".0"
			}
		}

		node.Hour, node.Minute, node.Second, node.Millisecond = t.parseTimeParts(timePart)

		if tzStr != "" {
			node.TimezoneOffset = &elm.LiteralNode{
				Annotation: t.cqfAnnotation(),
				ValueType:  typesystem.TypeDecimal,
				Value:      tzStr,
			}
		}
	}

	return node
}

// elmTypeOfElmExpr returns the ELM qualified type name for a translated expression
// when it can be statically determined. Returns "" when unknown.
func (t *Translator) elmTypeOfElmExpr(e elm.Expression) string {
	switch v := e.(type) {
	case *elm.LiteralNode:
		return v.ValueType
	case *elm.ParameterRefNode:
		if typ, ok := t.paramTypes[v.Name]; ok {
			return typ
		}
	}
	return ""
}

// implicitAnyCoercion applies CQF-compatible implicit coercion: when one operand
// of a comparison or arithmetic operator has type Any and the other has a known
// scalar type, the Any operand is wrapped in an As cast to the known type.
func (t *Translator) implicitAnyCoercion(lhs, rhs elm.Expression) (elm.Expression, elm.Expression) {
	lType := t.elmTypeOfElmExpr(lhs)
	rType := t.elmTypeOfElmExpr(rhs)

	if lType == typesystem.TypeAny && rType != "" && rType != typesystem.TypeAny {
		lhs = &elm.AsNode{
			Annotation: t.cqfAnnotation(),
			Signature:  t.cqfEmptyArrayField(),
			AsType:     rType,
			Operand:    lhs,
		}
	} else if rType == typesystem.TypeAny && lType != "" && lType != typesystem.TypeAny {
		rhs = &elm.AsNode{
			Annotation: t.cqfAnnotation(),
			Signature:  t.cqfEmptyArrayField(),
			AsType:     lType,
			Operand:    rhs,
		}
	}
	return lhs, rhs
}

// inferBoundType returns the ELM qualified type name of an interval bound expression
// as seen at AST level. Used to type null bounds with an As cast.
func (t *Translator) inferBoundType(expr ast.Expr) string {
	if expr == nil {
		return ""
	}
	switch fr := expr.(type) {
	case *ast.IntegerLiteral:
		return typesystem.TypeInteger
	case *ast.LongLiteral:
		return typesystem.TypeLong
	case *ast.DecimalLiteral:
		return typesystem.TypeDecimal
	case *ast.DateLiteral:
		return typesystem.TypeDate
	case *ast.DateTimeLiteral:
		return typesystem.TypeDateTime
	case *ast.TimeLiteral:
		return typesystem.TypeTime
	case *ast.QuantityLiteral:
		return typesystem.TypeQuantity
	case *ast.FunctionRef:
		// DateTime/Date/Time system function calls infer the corresponding type.
		switch fr.Name {
		case "DateTime":
			return typesystem.TypeDateTime
		case "Date":
			return typesystem.TypeDate
		case "Time":
			return typesystem.TypeTime
		}
	}
	return ""
}

// translateExpr converts an AST expression to an ELM expression and, when
// EnableLocators is active, stamps the source location onto the result node.
// If a callee set skipLocatorStamp (because it manually placed the locator on
// an inner node), the outer stamp is skipped.
func (t *Translator) translateExpr(expr ast.Expr) elm.Expression {
	result := t.translateExprCore(expr)
	if expr != nil && t.opts.EnableLocators && !t.skipLocatorStamp {
		setLocator(result, locatorStr(expr.Loc()))
	}
	t.skipLocatorStamp = false
	return result
}

// translateExprCore converts an AST expression to an ELM expression.
// All expression nodes receive annotation:[] in CQF mode (ELM Element base).
// OperatorExpression subclasses also receive signature:[].
func (t *Translator) translateExprCore(expr ast.Expr) elm.Expression {
	if expr == nil {
		return &elm.NullNode{Annotation: t.cqfAnnotation()}
	}
	ann := t.cqfAnnotation()
	sig := t.cqfEmptyArrayField() // for OperatorExpression subclasses

	switch v := expr.(type) {
	// ---- Literals ----
	case *ast.BooleanLiteral:
		val := "false"
		if v.Value {
			val = "true"
		}
		return &elm.LiteralNode{Annotation: ann, ValueType: typesystem.TypeBoolean, Value: val}
	case *ast.IntegerLiteral:
		return &elm.LiteralNode{Annotation: ann, ValueType: typesystem.TypeInteger, Value: fmt.Sprintf("%d", v.Value)}
	case *ast.LongLiteral:
		return &elm.LiteralNode{Annotation: ann, ValueType: typesystem.TypeLong, Value: fmt.Sprintf("%d", v.Value)}
	case *ast.DecimalLiteral:
		return &elm.LiteralNode{Annotation: ann, ValueType: typesystem.TypeDecimal, Value: v.Value}
	case *ast.StringLiteral:
		return &elm.LiteralNode{Annotation: ann, ValueType: typesystem.TypeString, Value: v.Value}
	case *ast.NullLiteral:
		return &elm.NullNode{Annotation: ann}
	case *ast.DateLiteral:
		return t.parseDateLiteral(v.Value)
	case *ast.DateTimeLiteral:
		return t.parseDateTimeLiteral(v.Value)
	case *ast.TimeLiteral:
		return t.parseTimeLiteral(v.Value)
	case *ast.QuantityLiteral:
		t.checkUnit(v.Unit, v.Loc())
		return &elm.QuantityNode{Annotation: ann, Value: json.Number(v.Value), Unit: v.Unit}
	case *ast.RatioLiteral:
		numUnit := v.Numerator.Unit
		if numUnit == "" {
			numUnit = "1"
		}
		denomUnit := v.Denominator.Unit
		if denomUnit == "" {
			denomUnit = "1"
		}
		t.checkUnit(v.Numerator.Unit, v.Loc())
		t.checkUnit(v.Denominator.Unit, v.Loc())
		num := &elm.QuantityLiteral{Annotation: t.cqfAnnotation(), Unit: numUnit, Value: json.Number(v.Numerator.Value)}
		den := &elm.QuantityLiteral{Annotation: t.cqfAnnotation(), Unit: denomUnit, Value: json.Number(v.Denominator.Value)}
		if t.opts.EnableLocators && v.Numerator.Unit != "" {
			num.Locator = locatorStr(v.Numerator.Loc())
		}
		if t.opts.EnableLocators && v.Denominator.Unit != "" {
			den.Locator = locatorStr(v.Denominator.Loc())
		}
		return &elm.RatioNode{
			Annotation:  ann,
			Numerator:   num,
			Denominator: den,
		}

	// ---- References ----
	case *ast.IdentifierRef:
		// Check innermost query let scope first — let variables in queries become QueryLetRef.
		for i := len(t.queryLetScopes) - 1; i >= 0; i-- {
			if t.queryLetScopes[i][v.Name] {
				return &elm.LetRefNode{Annotation: ann, Name: v.Name}
			}
		}
		// Check innermost query alias scope — identifiers that match a
		// query source alias must become AliasRef, not ExpressionRef.
		for i := len(t.queryAliases) - 1; i >= 0; i-- {
			if t.queryAliases[i][v.Name] {
				return &elm.AliasRefNode{Annotation: ann, Name: v.Name}
			}
		}
		// Function parameter references inside a function body must emit OperandRef.
		if t.functionParamScope[v.Name] {
			return &elm.OperandRefNode{Annotation: ann, Name: v.Name}
		}
		// Fall through to symbol table for defined expressions/parameters/etc.
		switch t.syms[v.Name] {
		case symParameter:
			return &elm.ParameterRefNode{Annotation: ann, Name: v.Name}
		case symValueSet:
			return &elm.ValueSetRefNode{Annotation: ann, Name: v.Name}
		case symCodeSystem:
			return &elm.CodeSystemRefNode{Annotation: ann, Name: v.Name}
		case symCode:
			return &elm.CodeRefNode{Annotation: ann, Name: v.Name}
		case symConcept:
			return &elm.ConceptRefNode{Annotation: ann, Name: v.Name}
		default:
			return &elm.ExpressionRefNode{Annotation: ann, Name: v.Name}
		}
	case *ast.QualifiedRef:
		return &elm.ExpressionRefNode{Annotation: ann, Name: v.Name, LibraryName: v.LibraryName}
	case *ast.AliasRef:
		return &elm.AliasRefNode{Annotation: ann, Name: v.Name}
	case *ast.LetRef:
		return &elm.LetRefNode{Annotation: ann, Name: v.Name}
	case *ast.ThisExpr:
		return &elm.QueryThisRefNode{Annotation: ann}
	case *ast.IndexExpr:
		return &elm.OperatorExpressionNode{Annotation: ann, Signature: sig, Operator: "QueryIndexRef"}
	case *ast.TotalExpr:
		return &elm.OperatorExpressionNode{Annotation: ann, Signature: sig, Operator: "Total"}
	case *ast.ExternalConstantExpr:
		return &elm.ExternalConstantNode{Annotation: ann, Name: v.Name}
	case *ast.FunctionRef:
		// Map system Date/DateTime/Time function calls to structured ELM nodes.
		if v.LibraryName == "" {
			switch v.Name {
			case "Date":
				node := &elm.DateNode{Annotation: ann}
				operands := make([]elm.Expression, 0, len(v.Operands))
				if len(v.Operands) > 0 {
					node.Year = t.translateExpr(v.Operands[0])
					operands = append(operands, node.Year)
				}
				if len(v.Operands) > 1 {
					node.Month = t.translateExpr(v.Operands[1])
					operands = append(operands, node.Month)
				}
				if len(v.Operands) > 2 {
					node.Day = t.translateExpr(v.Operands[2])
					operands = append(operands, node.Day)
				}
				node.Signature = t.computeSig("Date", operands)
				return node
			case "DateTime":
				node := &elm.DateTimeNode{Annotation: ann}
				operands := make([]elm.Expression, 0, len(v.Operands))
				if len(v.Operands) > 0 {
					node.Year = t.translateExpr(v.Operands[0])
					operands = append(operands, node.Year)
				}
				if len(v.Operands) > 1 {
					node.Month = t.translateExpr(v.Operands[1])
					operands = append(operands, node.Month)
				}
				if len(v.Operands) > 2 {
					node.Day = t.translateExpr(v.Operands[2])
					operands = append(operands, node.Day)
				}
				if len(v.Operands) > 3 {
					node.Hour = t.translateExpr(v.Operands[3])
					operands = append(operands, node.Hour)
				}
				if len(v.Operands) > 4 {
					node.Minute = t.translateExpr(v.Operands[4])
					operands = append(operands, node.Minute)
				}
				if len(v.Operands) > 5 {
					node.Second = t.translateExpr(v.Operands[5])
					operands = append(operands, node.Second)
				}
				if len(v.Operands) > 6 {
					node.Millisecond = t.translateExpr(v.Operands[6])
					operands = append(operands, node.Millisecond)
				}
				if len(v.Operands) > 7 {
					node.TimezoneOffset = t.translateExpr(v.Operands[7])
					operands = append(operands, node.TimezoneOffset)
				}
				node.Signature = t.computeSig("DateTime", operands)
				return node
			case "Time":
				node := &elm.TimeNode{Annotation: ann}
				operands := make([]elm.Expression, 0, len(v.Operands))
				if len(v.Operands) > 0 {
					node.Hour = t.translateExpr(v.Operands[0])
					operands = append(operands, node.Hour)
				}
				if len(v.Operands) > 1 {
					node.Minute = t.translateExpr(v.Operands[1])
					operands = append(operands, node.Minute)
				}
				if len(v.Operands) > 2 {
					node.Second = t.translateExpr(v.Operands[2])
					operands = append(operands, node.Second)
				}
				if len(v.Operands) > 3 {
					node.Millisecond = t.translateExpr(v.Operands[3])
					operands = append(operands, node.Millisecond)
				}
				node.Signature = t.computeSig("Time", operands)
				return node
			}
			// Map unary system operators to UnaryExpressionNode.
			if unarySystemOps[v.Name] && len(v.Operands) == 1 {
				operand := t.translateExpr(v.Operands[0])
				return &elm.UnaryExpressionNode{
					Annotation: ann,
					Signature:  t.computeSig(v.Name, []elm.Expression{operand}),
					Operator:   v.Name,
					Operand:    operand,
				}
			}
			// Tail(list) → Slice(source: list, startIndex: 1, endIndex: Null)
			if v.Name == "Tail" && len(v.Operands) == 1 {
				src := t.translateExpr(v.Operands[0])
				return &elm.NamedOperatorExpressionNode{
					Annotation: ann,
					Signature:  t.computeSig("Slice", []elm.Expression{src}),
					Operator:   "Slice",
					Operands: []elm.NamedOperand{
						{Name: "source", Value: src},
						{Name: "startIndex", Value: t.intLiteral(1)},
						{Name: "endIndex", Value: &elm.NullNode{Annotation: t.cqfAnnotation()}},
					},
				}
			}
			// Skip(list, n) → Slice(source: list, startIndex: n, endIndex: Null)
			if v.Name == "Skip" && len(v.Operands) == 2 {
				src := t.translateExpr(v.Operands[0])
				start := t.translateExpr(v.Operands[1])
				return &elm.NamedOperatorExpressionNode{
					Annotation: ann,
					Signature:  t.computeSig("Slice", []elm.Expression{src, start}),
					Operator:   "Slice",
					Operands: []elm.NamedOperand{
						{Name: "source", Value: src},
						{Name: "startIndex", Value: start},
						{Name: "endIndex", Value: &elm.NullNode{Annotation: t.cqfAnnotation()}},
					},
				}
			}
			// Take(list, n) → Slice(source: list, startIndex: 0, endIndex: Coalesce(n, 0))
			if v.Name == "Take" && len(v.Operands) == 2 {
				src := t.translateExpr(v.Operands[0])
				n := t.translateExpr(v.Operands[1])
				coalesceOperands := []elm.Expression{n, t.intLiteral(0)}
				coalesce := &elm.OperatorExpressionNode{
					Annotation: t.cqfAnnotation(),
					Signature:  t.computeSig("Coalesce", coalesceOperands),
					Operator:   "Coalesce",
					Operand:    coalesceOperands,
				}
				return &elm.NamedOperatorExpressionNode{
					Annotation: ann,
					Signature:  t.computeSig("Slice", []elm.Expression{src, n}),
					Operator:   "Slice",
					Operands: []elm.NamedOperand{
						{Name: "source", Value: src},
						{Name: "startIndex", Value: t.intLiteral(0)},
						{Name: "endIndex", Value: coalesce},
					},
				}
			}
			// Map aggregate system operators to AggregateExpressionNode (uses "source" field).
			if aggregateSystemOps[v.Name] && len(v.Operands) == 1 {
				src := t.translateExpr(v.Operands[0])
				if decimalAggregateOps[v.Name] && !t.isDecimalListExpr(v.Operands[0]) {
					src = t.wrapInDecimalQuery(src)
				}
				return &elm.AggregateExpressionNode{
					Annotation: ann,
					Signature:  t.computeSig(v.Name, []elm.Expression{src}),
					Operator:   v.Name,
					Source:     src,
				}
			}
			// Map operators with named operand fields (Substring, Combine, etc.).
			if slots, ok := namedOperatorOps[v.Name]; ok && len(v.Operands) > 0 && len(v.Operands) <= len(slots) {
				vals := make([]elm.Expression, 0, len(v.Operands))
				namedOps := make([]elm.NamedOperand, 0, len(v.Operands))
				for i, o := range v.Operands {
					val := t.translateExpr(o)
					// Log requires Decimal operands; CQF promotes integer literals.
					if v.Name == "Log" {
						if _, isInt := o.(*ast.IntegerLiteral); isInt {
							val = &elm.UnaryExpressionNode{
								Annotation: t.cqfAnnotation(),
								Signature:  t.computeSig("ToDecimal", []elm.Expression{val}),
								Operator:   "ToDecimal",
								Operand:    val,
							}
						}
					}
					// IndexOf has only List<T> overload in ELM; for String args
					// CQF wraps the source in ToList (String → List<String>).
					if v.Name == "IndexOf" && i == 0 {
						if _, isStr := o.(*ast.StringLiteral); isStr {
							val = &elm.UnaryExpressionNode{
								Annotation: t.cqfAnnotation(),
								Signature:  t.cqfEmptyArrayField(),
								Operator:   "ToList",
								Operand:    val,
							}
						}
					}
					vals = append(vals, val)
					namedOps = append(namedOps, elm.NamedOperand{
						Name:  slots[i],
						Value: val,
					})
				}
				return &elm.NamedOperatorExpressionNode{
					Annotation: ann,
					Signature:  t.computeSig(v.Name, vals),
					Operator:   v.Name,
					Operands:   namedOps,
				}
			}
			// Map binary system operators (operand: [a, b]) to generic OperatorExpressionNode.
			if binaryOperandOps[v.Name] && len(v.Operands) == 2 {
				ops := []elm.Expression{
					t.translateExpr(v.Operands[0]),
					t.translateExpr(v.Operands[1]),
				}
				// Log requires Decimal operands; CQF promotes integer literals.
				if v.Name == "Log" {
					for i, o := range v.Operands {
						if _, isInt := o.(*ast.IntegerLiteral); isInt {
							ops[i] = &elm.UnaryExpressionNode{
								Annotation: t.cqfAnnotation(),
								Signature:  t.computeSig("ToDecimal", []elm.Expression{ops[i]}),
								Operator:   "ToDecimal",
								Operand:    ops[i],
							}
						}
					}
				}
				// Contains has only List<T> overload in ELM; for String args CQF
				// wraps the source string in ToList.
				if v.Name == "Contains" {
					if _, isStr := v.Operands[0].(*ast.StringLiteral); isStr {
						ops[0] = &elm.UnaryExpressionNode{
							Annotation: t.cqfAnnotation(),
							Signature:  t.cqfEmptyArrayField(),
							Operator:   "ToList",
							Operand:    ops[0],
						}
					}
				}
				return &elm.OperatorExpressionNode{
					Annotation: ann,
					Signature:  t.computeSig(v.Name, ops),
					Operator:   v.Name,
					Operand:    ops,
				}
			}
			// N-ary positional operators (e.g. ReplaceMatches) emit operand: [...]
			if naryOperandOps[v.Name] && len(v.Operands) > 0 {
				ops := make([]elm.Expression, len(v.Operands))
				for i, o := range v.Operands {
					ops[i] = t.translateExpr(o)
				}
				return &elm.OperatorExpressionNode{
					Annotation: ann,
					Signature:  t.computeSig(v.Name, ops),
					Operator:   v.Name,
					Operand:    ops,
				}
			}
			// Coalesce is a native ELM n-ary operator. CQF leaves leading nulls bare
			// and wraps later nulls in As(T, null) once an earlier operand has
			// established a concrete type.
			if v.Name == "Coalesce" {
				var operands []elm.Expression
				var inferredType string
				for _, o := range v.Operands {
					elem := t.translateExpr(o)
					if _, isNull := o.(*ast.NullLiteral); isNull {
						if inferredType != "" {
							elem = &elm.AsNode{
								Annotation: t.cqfAnnotation(),
								Signature:  t.cqfEmptyArrayField(),
								Operand:    elem,
								AsType:     inferredType,
							}
						}
					} else if inferredType == "" {
						inferredType = t.inferListElementType([]ast.Expr{o})
					}
					operands = append(operands, elem)
				}
				return &elm.OperatorExpressionNode{
					Annotation: ann,
					Signature:  t.computeSig("Coalesce", operands),
					Operator:   "Coalesce",
					Operand:    operands,
				}
			}
			// Expand age functions (AgeInYears, AgeInYearsAt, etc.) to CalculateAge/CalculateAgeAt.
			if expanded := t.expandAgeFunction(ann, sig, v.Name, v.Operands); expanded != nil {
				return expanded
			}
		}
		var operands []elm.Expression
		for _, o := range v.Operands {
			operands = append(operands, t.translateExpr(o))
		}
		return &elm.FunctionRefNode{
			Annotation:  ann,
			Signature:   t.computeSig("FunctionRef", operands),
			Name:        v.Name,
			LibraryName: v.LibraryName,
			Operand:     operands,
		}
	case *ast.PropertyExpr:
		// When the source is an identifier that resolves to a query alias or let variable,
		// ELM uses the "scope" string attribute rather than a "source" expression object.
		var propertyNode elm.Expression
		if ir, ok := v.Source.(*ast.IdentifierRef); ok {
			isAlias := false
			for i := len(t.queryAliases) - 1; i >= 0; i-- {
				if t.queryAliases[i][ir.Name] {
					isAlias = true
					break
				}
			}
			isLet := false
			if !isAlias {
				for i := len(t.queryLetScopes) - 1; i >= 0; i-- {
					if t.queryLetScopes[i][ir.Name] {
						isLet = true
						break
					}
				}
			}
			if isAlias || isLet {
				propertyNode = &elm.PropertyNode{Annotation: ann, Path: v.Path, Scope: ir.Name}
			}
		}
		if propertyNode == nil {
			if ar, ok := v.Source.(*ast.AliasRef); ok {
				propertyNode = &elm.PropertyNode{Annotation: ann, Path: v.Path, Scope: ar.Name}
			}
		}
		if propertyNode == nil {
			propertyNode = &elm.PropertyNode{
				Annotation: ann,
				Path:       v.Path,
				Source:     t.translateExpr(v.Source),
			}
		}
		// Apply implicit FHIRHelpers coercion when FHIRHelpers is included and the
		// property resolves to a FHIR primitive type. CQF places the source locator
		// on the inner PropertyNode, not the synthetic FunctionRef wrapper.
		if t.fhirHelpersLocalName != "" {
			if fhirFunc := t.resolveFHIRPropertyCoercion(v.Source, v.Path); fhirFunc != "" {
				if t.opts.EnableLocators {
					setLocator(propertyNode, locatorStr(v.Loc()))
					t.skipLocatorStamp = true
				}
				wrapperSig := sig
				if fhirType := t.resolveFHIRPropertyType(v.Source, v.Path); fhirType != "" {
					typeName := fhirType
					if sourceType := t.resolveFHIRSourceType(v.Source); sourceType != "" {
						if bound := typesystem.FHIRPropertyBinding[sourceType+"."+v.Path]; bound != "" {
							typeName = bound
						}
					}
					argSpec := namedTS{"{http://hl7.org/fhir}" + typeName}
					wrapperSig = t.buildSigFromTypes("FunctionRef", []typeSpec{argSpec})
				}
				return &elm.FunctionRefNode{
					Annotation:  ann,
					Signature:   wrapperSig,
					Name:        fhirFunc,
					LibraryName: t.fhirHelpersLocalName,
					Operand:     []elm.Expression{propertyNode},
				}
			}
		}
		return propertyNode
	case *ast.IndexedAccessExpr:
		operands := []elm.Expression{t.translateExpr(v.Source), t.translateExpr(v.Index)}
		return &elm.OperatorExpressionNode{
			Annotation: ann,
			Signature:  t.computeSig("Indexer", operands),
			Operator:   "Indexer",
			Operand:    operands,
		}

	// ---- Unary / binary ----
	case *ast.UnaryExpr:
		operand := t.translateExpr(v.Operand)
		return &elm.UnaryExpressionNode{
			Annotation: ann,
			Signature:  t.computeSig(v.Op, []elm.Expression{operand}),
			Operator:   v.Op,
			Operand:    operand,
		}
	case *ast.BinaryExpr:
		return t.translateBinaryExpr(v)
	case *ast.TernaryExpr:
		condition := t.translateExpr(v.Condition)
		// CQF wraps a bare-null condition in As(Boolean, null) since if requires Boolean.
		if _, isNull := v.Condition.(*ast.NullLiteral); isNull {
			condition = &elm.AsNode{
				Annotation: t.cqfAnnotation(),
				Signature:  json.RawMessage("[]"),
				Operand:    condition,
				AsType:     "{urn:hl7-org:elm-types:r1}Boolean",
			}
		}
		thenExpr := t.translateExpr(v.ThenExpr)
		elseExpr := t.translateExpr(v.ElseExpr)
		// CQF types a bare-null branch with the inferred type of the other branch.
		if _, elseIsNull := v.ElseExpr.(*ast.NullLiteral); elseIsNull {
			if typ := t.inferListElementType([]ast.Expr{v.ThenExpr}); typ != "" {
				elseExpr = &elm.AsNode{
					Annotation: t.cqfAnnotation(),
					Signature:  json.RawMessage("[]"),
					Operand:    elseExpr,
					AsType:     typ,
				}
			}
		}
		if _, thenIsNull := v.ThenExpr.(*ast.NullLiteral); thenIsNull {
			if typ := t.inferListElementType([]ast.Expr{v.ElseExpr}); typ != "" {
				thenExpr = &elm.AsNode{
					Annotation: t.cqfAnnotation(),
					Signature:  json.RawMessage("[]"),
					Operand:    thenExpr,
					AsType:     typ,
				}
			}
		}
		return &elm.IfNode{
			Annotation: ann,
			Condition:  condition,
			Then:       thenExpr,
			Else:       elseExpr,
		}
	case *ast.CaseExpr:
		// Infer the result type by scanning all branches for the first typed literal.
		var inferred string
		probe := make([]ast.Expr, 0, len(v.Items)+1)
		for _, item := range v.Items {
			probe = append(probe, item.Then)
		}
		probe = append(probe, v.Else)
		inferred = t.inferListElementType(probe)
		elseExpr := t.translateExpr(v.Else)
		if _, isNull := v.Else.(*ast.NullLiteral); isNull && inferred != "" {
			elseExpr = &elm.AsNode{
				Annotation: t.cqfAnnotation(),
				Signature:  json.RawMessage("[]"),
				Operand:    elseExpr,
				AsType:     inferred,
			}
		}
		cn := &elm.CaseNode{Annotation: ann, Else: elseExpr}
		if v.Comparand != nil {
			cn.Comparand = t.translateExpr(v.Comparand)
		}
		for _, item := range v.Items {
			thenExpr := t.translateExpr(item.Then)
			if _, isNull := item.Then.(*ast.NullLiteral); isNull && inferred != "" {
				thenExpr = &elm.AsNode{
					Annotation: t.cqfAnnotation(),
					Signature:  json.RawMessage("[]"),
					Operand:    thenExpr,
					AsType:     inferred,
				}
			}
			ci := &elm.CaseItem{
				Annotation: t.cqfAnnotation(),
				When:       t.translateExpr(item.When),
				Then:       thenExpr,
			}
			if t.opts.EnableLocators {
				ci.Locator = locatorStr(item.Loc())
			}
			cn.CaseItem = append(cn.CaseItem, ci)
		}
		return cn

	// ---- Type operators ----
	case *ast.TypeIsExpr:
		return t.translateTypeIs(v)
	case *ast.TypeAsExpr:
		ts := t.translateTypeSpecifier(v.TypeSpec)
		return &elm.AsNode{
			Annotation:      ann,
			Signature:       sig,
			Operand:         t.translateExpr(v.Operand),
			AsTypeSpecifier: ts,
			Strict:          &v.Strict,
		}
	case *ast.ConvertExpr:
		// CQF normalizes `convert x to Integer` to ToInteger(x) for built-in system
		// types (Integer, Decimal, String, Boolean, Date, DateTime, Time, Long,
		// Quantity, Ratio, Concept, Code). Use the operator-specific form when the
		// target type is a bare System named type.
		if nts, ok := v.TypeSpec.(*ast.NamedTypeSpecifier); ok && nts != nil {
			qualifier := nts.Qualifier
			if qualifier == "" || qualifier == "System" {
				if elmOp, ok := convertToOps[nts.Name]; ok {
					operand := t.translateExpr(v.Operand)
					return &elm.UnaryExpressionNode{
						Annotation: ann,
						Signature:  t.computeSig(elmOp, []elm.Expression{operand}),
						Operator:   elmOp,
						Operand:    operand,
					}
				}
			}
		}
		ts := t.translateTypeSpecifier(v.TypeSpec)
		return &elm.ConvertNode{
			Annotation:      ann,
			Signature:       sig,
			Operand:         t.translateExpr(v.Operand),
			ToTypeSpecifier: ts,
		}

	// ---- Timing / interval ----
	case *ast.TimingExpr:
		return t.translateTimingExpr(v)
	case *ast.BetweenExpr:
		if v.Properly {
			return &elm.OperatorExpressionNode{
				Annotation: ann,
				Signature:  sig,
				Operator:   "ProperBetween",
				Operand: []elm.Expression{
					t.translateExpr(v.Operand),
					t.translateExpr(v.Low),
					t.translateExpr(v.High),
				},
			}
		}
		return &elm.OperatorExpressionNode{
			Annotation: ann,
			Signature:  sig,
			Operator:   "Between",
			Operand: []elm.Expression{
				t.translateExpr(v.Operand),
				t.translateExpr(v.Low),
				t.translateExpr(v.High),
			},
		}
	case *ast.DurationBetweenExpr:
		operands := []elm.Expression{t.translateExpr(v.Low), t.translateExpr(v.High)}
		return &elm.PrecisionOperatorNode{
			Annotation: ann,
			Signature:  t.computeSig("DurationBetween", operands),
			Operator:   "DurationBetween",
			Precision:  v.Precision,
			Operand:    operands,
		}
	case *ast.DifferenceBetweenExpr:
		operands := []elm.Expression{t.translateExpr(v.Low), t.translateExpr(v.High)}
		return &elm.PrecisionOperatorNode{
			Annotation: ann,
			Signature:  t.computeSig("DifferenceBetween", operands),
			Operator:   "DifferenceBetween",
			Precision:  v.Precision,
			Operand:    operands,
		}
	case *ast.IntervalExpr:
		low := t.translateExpr(v.Low)
		high := t.translateExpr(v.High)
		// Null interval bounds: CQF wraps null in an As cast typed from the non-null bound.
		if _, isNull := low.(*elm.NullNode); isNull {
			if bndType := t.inferBoundType(v.High); bndType != "" {
				low = &elm.AsNode{
					Annotation: t.cqfAnnotation(),
					Signature:  t.cqfEmptyArrayField(),
					AsType:     bndType,
					Operand:    low,
				}
			}
		}
		if _, isNull := high.(*elm.NullNode); isNull {
			if bndType := t.inferBoundType(v.Low); bndType != "" {
				high = &elm.AsNode{
					Annotation: t.cqfAnnotation(),
					Signature:  t.cqfEmptyArrayField(),
					AsType:     bndType,
					Operand:    high,
				}
			}
		}
		return &elm.IntervalNode{
			Annotation: ann,
			Low:        low,
			High:       high,
			LowClosed:  v.LowClosed,
			HighClosed: v.HighClosed,
		}
	case *ast.TimeBoundaryExpr:
		op := titleCase(v.Boundary)
		operand := t.translateExpr(v.Source)
		return &elm.UnaryExpressionNode{
			Annotation: ann,
			Signature:  t.computeSig(op, []elm.Expression{operand}),
			Operator:   op,
			Operand:    operand,
		}
	case *ast.DateTimeComponentExpr:
		return &elm.PrecisionOperatorNode{
			Annotation: ann,
			Signature:  sig,
			Operator:   "DateTimeComponentFrom",
			Precision:  v.Precision,
			Operand:    []elm.Expression{t.translateExpr(v.Source)},
		}
	case *ast.DurationExpr:
		src := t.translateExpr(v.Source)
		startNode := &elm.UnaryExpressionNode{Annotation: ann, Signature: t.computeSig("Start", []elm.Expression{src}), Operator: "Start", Operand: src}
		endNode := &elm.UnaryExpressionNode{Annotation: ann, Signature: t.computeSig("End", []elm.Expression{src}), Operator: "End", Operand: src}
		operands := []elm.Expression{startNode, endNode}
		return &elm.PrecisionOperatorNode{
			Annotation: ann,
			Signature:  t.computeSig("DurationBetween", operands),
			Operator:   "DurationBetween",
			Precision:  v.Precision,
			Operand:    operands,
		}
	case *ast.DifferenceExpr:
		src := t.translateExpr(v.Source)
		startNode := &elm.UnaryExpressionNode{Annotation: ann, Signature: t.computeSig("Start", []elm.Expression{src}), Operator: "Start", Operand: src}
		endNode := &elm.UnaryExpressionNode{Annotation: ann, Signature: t.computeSig("End", []elm.Expression{src}), Operator: "End", Operand: src}
		operands := []elm.Expression{startNode, endNode}
		return &elm.PrecisionOperatorNode{
			Annotation: ann,
			Signature:  t.computeSig("DifferenceBetween", operands),
			Operator:   "DifferenceBetween",
			Precision:  v.Precision,
			Operand:    operands,
		}
	case *ast.WidthExpr:
		operand := t.translateExpr(v.Source)
		return &elm.UnaryExpressionNode{
			Annotation: ann,
			Signature:  t.computeSig("Width", []elm.Expression{operand}),
			Operator:   "Width",
			Operand:    operand,
		}
	case *ast.SuccessorExpr:
		operand := t.translateExpr(v.Source)
		return &elm.UnaryExpressionNode{
			Annotation: ann,
			Signature:  t.computeSig("Successor", []elm.Expression{operand}),
			Operator:   "Successor",
			Operand:    operand,
		}
	case *ast.PredecessorExpr:
		operand := t.translateExpr(v.Source)
		return &elm.UnaryExpressionNode{
			Annotation: ann,
			Signature:  t.computeSig("Predecessor", []elm.Expression{operand}),
			Operator:   "Predecessor",
			Operand:    operand,
		}
	case *ast.SingletonFromExpr:
		operand := t.translateExpr(v.Source)
		return &elm.SingletonFromNode{
			Annotation: ann,
			Signature:  t.computeSig("SingletonFrom", []elm.Expression{operand}),
			Operand:    operand,
		}
	case *ast.PointFromExpr:
		return &elm.UnaryExpressionNode{
			Annotation: ann,
			Signature:  sig,
			Operator:   "PointFrom",
			Operand:    t.translateExpr(v.Source),
		}
	case *ast.TypeExtentExpr:
		// MinValue<T> / MaxValue<T> are OperatorExpression nodes with a valueType attribute.
		op := "MinValue"
		if v.Extent == "maximum" {
			op = "MaxValue"
		}
		ts := t.translateTypeSpecifier(v.TypeSpec)
		vt := ""
		if nts, ok := ts.(*elm.NamedTypeSpecifier); ok {
			vt = nts.Name
		}
		return &elm.MinMaxValueNode{
			Annotation: ann,
			Signature:  sig,
			Operator:   op,
			ValueType:  vt,
		}

	// ---- Selectors ----
	case *ast.ListExpr:
		ln := &elm.ListNode{Annotation: ann}
		// Determine element type for null coercion.
		// Typed list: use declared element type. Untyped: infer from non-null elements.
		var nullType string
		var typedElemTS typeSpec
		if v.TypeSpec != nil {
			ts := t.translateTypeSpecifier(v.TypeSpec)
			if nts, ok := ts.(*elm.NamedTypeSpecifier); ok {
				nullType = nts.Name
				typedElemTS = namedTS{nts.Name}
			}
		} else {
			nullType = t.inferListElementType(v.Elements)
		}
		for _, e := range v.Elements {
			elem := t.translateExpr(e)
			if _, isNull := e.(*ast.NullLiteral); isNull && nullType != "" {
				elem = &elm.AsNode{
					Annotation: t.cqfAnnotation(),
					Signature:  t.cqfEmptyArrayField(),
					Operand:    elem,
					AsType:     nullType,
				}
			}
			ln.Element = append(ln.Element, elem)
		}
		if typedElemTS != nil {
			if t.listNodeTypes == nil {
				t.listNodeTypes = map[*elm.ListNode]typeSpec{}
			}
			t.listNodeTypes[ln] = typedElemTS
		}
		return ln
	case *ast.TupleExpr:
		tn := &elm.TupleNode{Annotation: ann}
		for _, e := range v.Elements {
			tn.Element = append(tn.Element, &elm.TupleElementNode{
				Name:  e.Name,
				Value: t.translateExpr(e.Expression),
			})
		}
		return tn
	case *ast.InstanceExpr:
		in := &elm.InstanceNode{Annotation: ann}
		if v.TypeSpec != nil {
			if nt, ok := v.TypeSpec.(*ast.NamedTypeSpecifier); ok {
				in.ClassType = nt.Name
			}
		}
		for _, e := range v.Elements {
			in.Element = append(in.Element, &elm.TupleElementNode{
				Name:  e.Name,
				Value: t.translateExpr(e.Expression),
			})
		}
		return in
	case *ast.CodeExpr:
		cn := &elm.CodeNode{Annotation: ann, Code: v.Code, Display: v.Display}
		if v.System != "" {
			cn.System = &elm.CodeSystemRef{Name: v.System}
		}
		return cn
	case *ast.ConceptExpr:
		cn := &elm.ConceptNode{Annotation: ann, Display: v.Display}
		for _, c := range v.Codes {
			codeNode := &elm.CodeNode{Code: c.Code, Display: c.Display}
			if c.System != "" {
				codeNode.System = &elm.CodeSystemRef{Name: c.System}
			}
			cn.Code = append(cn.Code, codeNode)
		}
		return cn

	// ---- Aggregate ----
	case *ast.AggregateExpr:
		// Distinct and Flatten are UnaryExpression in ELM (single operand object, not array).
		operand := t.translateExpr(v.Operand)
		return &elm.UnaryExpressionNode{
			Annotation: ann,
			Signature:  t.computeSig(v.Op, []elm.Expression{operand}),
			Operator:   v.Op,
			Operand:    operand,
		}
	case *ast.SetAggregateExpr:
		operands := []elm.Expression{t.translateExpr(v.Operand)}
		if v.PerClause != nil {
			operands = append(operands, t.translateExpr(v.PerClause))
		} else if v.Op == "Collapse" || v.Op == "Expand" {
			// CQF always emits null as the second (precision) argument when absent.
			operands = append(operands, &elm.NullNode{Annotation: t.cqfAnnotation()})
		}
		return &elm.OperatorExpressionNode{Annotation: ann, Signature: t.computeSig(v.Op, operands), Operator: v.Op, Operand: operands}

	// ---- Retrieve ----
	case *ast.RetrieveExpr:
		return t.translateRetrieve(v)

	// ---- Query ----
	case *ast.QueryExpression:
		return t.translateQuery(v)

	default:
		return &elm.UnimplementedNode{TypeName: fmt.Sprintf("%T", expr)}
	}
}

// isStringLikeExpr is true when expr is a string literal or a chain of
// string-concatenation Add operators producing a string.
func isStringLikeExpr(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.StringLiteral:
		return true
	case *ast.BinaryExpr:
		return e.Op == "Add" && isStringLikeExpr(e.Left) && isStringLikeExpr(e.Right)
	}
	return false
}

func (t *Translator) translateBinaryExpr(v *ast.BinaryExpr) elm.Expression {
	// InValueSet: when "In" operator's RHS is a declared value set, emit InValueSet.
	// preserve:true is an ELM R1 / CQL 1.5 feature (runtime terminology evaluation);
	// suppress it for compatibility level 1.4.
	if v.Op == "In" {
		if ref, ok := v.Right.(*ast.IdentifierRef); ok && t.syms[ref.Name] == symValueSet {
			compat := t.opts.CompatibilityLevel
			emitPreserve := compat == "" || compat >= "1.5"
			var preservePtr *bool
			if emitPreserve {
				b := true
				preservePtr = &b
			}
			vs := &elm.ValueSetRefNode{
				Annotation: t.cqfAnnotation(),
				Name:       ref.Name,
				Preserve:   preservePtr,
			}
			if t.opts.EnableLocators {
				vs.Locator = locatorStr(ref.Loc())
			}
			codeExpr := t.translateExpr(v.Left)
			invsSig := t.cqfEmptyArrayField()
			if lvl := t.opts.SignatureLevel; lvl != "" && lvl != "None" {
				invsSig = t.buildInferredSig([]elm.Expression{codeExpr})
			}
			return &elm.InValueSetNode{
				Annotation: t.cqfAnnotation(),
				Signature:  invsSig,
				Code:       codeExpr,
				ValueSet:   vs,
			}
		}
	}

	lhs := t.translateExpr(v.Left)
	rhs := t.translateExpr(v.Right)

	// IncludedIn → In: when FHIRHelpers is included and the LHS is a FHIR dateTime/instant
	// property (which coerces to a DateTime point), use "In" instead of "IncludedIn" because
	// the coerced LHS is a point, not an interval.
	op := v.Op
	if op == "IncludedIn" && t.fhirHelpersLocalName != "" {
		if pe, ok := v.Left.(*ast.PropertyExpr); ok {
			if fhirType := t.resolveFHIRPropertyType(pe.Source, pe.Path); typesystem.IsFHIRDateTimeType(fhirType) {
				op = "In"
			}
		}
	}

	// For the "In" operator, apply either interval promotion or list promotion when
	// the RHS is not already an interval or list expression.
	var listPromoted bool
	if v.Op == "In" && !t.opts.DisableListPromotion {
		if _, isInterval := v.Right.(*ast.IntervalExpr); !isInterval {
			if _, isList := v.Right.(*ast.ListExpr); !isList {
				if t.opts.EnableIntervalPromotion {
					// Interval promotion: wrap scalar RHS in If(IsNull(rhs), Null, Interval[rhs,rhs]).
					// Three independent translations of v.Right are required.
					rhs = &elm.IfNode{
						Annotation: t.cqfAnnotation(),
						Condition: &elm.UnaryExpressionNode{
							Annotation: t.cqfAnnotation(),
							Signature:  t.cqfEmptyArrayField(),
							Operator:   "IsNull",
							Operand:    rhs,
						},
						Then: &elm.NullNode{Annotation: t.cqfAnnotation()},
						Else: &elm.IntervalNode{
							Annotation: t.cqfAnnotation(),
							Low:        t.translateExpr(v.Right),
							High:       t.translateExpr(v.Right),
							LowClosed:  true,
							HighClosed: true,
						},
					}
				} else {
					// List promotion: wrap scalar RHS in ToList.
					rhs = &elm.UnaryExpressionNode{
						Annotation: t.cqfAnnotation(),
						Signature:  t.cqfEmptyArrayField(),
						Operator:   "ToList",
						Operand:    rhs,
					}
					listPromoted = true
				}
			}
		}
	}

	// Implicit Any coercion: when one operand is typed Any and the other has a
	// known scalar type, wrap the Any operand in an As cast (CQF behavior).
	comparisonOps := map[string]bool{
		"Equal": true, "NotEqual": true,
		"Less": true, "LessOrEqual": true,
		"Greater": true, "GreaterOrEqual": true,
		"Add": true, "Subtract": true, "Multiply": true, "Divide": true,
	}
	if comparisonOps[v.Op] {
		lhs, rhs = t.implicitAnyCoercion(lhs, rhs)
	}

	// String concatenation: CQL `'a' + 'b'` maps to ELM Concatenate (not Add).
	if op == "Add" && isStringLikeExpr(v.Left) && isStringLikeExpr(v.Right) {
		op = "Concatenate"
	}

	// Implicit Integer→Decimal promotion for `/` (Divide), which is decimal-only
	// in CQL. CQF wraps Integer literal operands in ToDecimal. (Power has an
	// integer overload, so we don't promote there.)
	if op == "Divide" {
		if _, ok := v.Left.(*ast.IntegerLiteral); ok {
			lhs = &elm.UnaryExpressionNode{
				Annotation: t.cqfAnnotation(),
				Signature:  t.computeSig("ToDecimal", []elm.Expression{lhs}),
				Operator:   "ToDecimal",
				Operand:    lhs,
			}
		}
		if _, ok := v.Right.(*ast.IntegerLiteral); ok {
			rhs = &elm.UnaryExpressionNode{
				Annotation: t.cqfAnnotation(),
				Signature:  t.computeSig("ToDecimal", []elm.Expression{rhs}),
				Operator:   "ToDecimal",
				Operand:    rhs,
			}
		}
	}

	// Implicit Integer→Decimal promotion for mixed-type arithmetic:
	// Add/Subtract/Multiply/Modulo with one Integer literal and one Decimal
	// literal. CQF wraps the Integer side in ToDecimal so both operands match.
	arithMixedOps := map[string]bool{"Add": true, "Subtract": true, "Multiply": true, "Modulo": true}
	if arithMixedOps[op] {
		_, leftInt := v.Left.(*ast.IntegerLiteral)
		_, leftDec := v.Left.(*ast.DecimalLiteral)
		_, rightInt := v.Right.(*ast.IntegerLiteral)
		_, rightDec := v.Right.(*ast.DecimalLiteral)
		if leftInt && rightDec {
			lhs = &elm.UnaryExpressionNode{
				Annotation: t.cqfAnnotation(),
				Signature:  t.computeSig("ToDecimal", []elm.Expression{lhs}),
				Operator:   "ToDecimal",
				Operand:    lhs,
			}
		} else if rightInt && leftDec {
			rhs = &elm.UnaryExpressionNode{
				Annotation: t.cqfAnnotation(),
				Signature:  t.computeSig("ToDecimal", []elm.Expression{rhs}),
				Operator:   "ToDecimal",
				Operand:    rhs,
			}
		}
	}

	operands := []elm.Expression{lhs, rhs}
	if v.Precision != "" {
		return &elm.PrecisionOperatorNode{
			Annotation: t.cqfAnnotation(),
			Signature:  t.computeSig(op, operands),
			Operator:   op,
			Precision:  v.Precision,
			Operand:    operands,
		}
	}
	// "In" uses special signature logic based on whether promotion was applied.
	var sig json.RawMessage
	if op == "In" {
		sig = t.computeInSig(lhs, rhs, listPromoted)
	} else {
		sig = t.computeSig(op, operands)
	}
	return &elm.OperatorExpressionNode{
		Annotation: t.cqfAnnotation(),
		Signature:  sig,
		Operator:   op,
		Operand:    operands,
	}
}

// resolveFHIRSourceType returns the FHIR resource or backbone type name for the
// expression, by inspecting query alias type bindings and the current context.
// Returns "" if the type cannot be determined.
func (t *Translator) resolveFHIRSourceType(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.IdentifierRef:
		// Check query alias type bindings (innermost scope first).
		for i := len(t.queryAliasTypes) - 1; i >= 0; i-- {
			if typ, ok := t.queryAliasTypes[i][e.Name]; ok {
				return typ
			}
		}
		// Fall back to the current context singleton (e.g., "Patient").
		if e.Name == t.currentContextName {
			return t.currentContextName
		}
	case *ast.AliasRef:
		for i := len(t.queryAliasTypes) - 1; i >= 0; i-- {
			if typ, ok := t.queryAliasTypes[i][e.Name]; ok {
				return typ
			}
		}
	case *ast.PropertyExpr:
		// For chained access (e.g., E.period.start), resolve the intermediate type.
		sourceType := t.resolveFHIRSourceType(e.Source)
		if sourceType == "" {
			return ""
		}
		key := sourceType + "." + e.Path
		return typesystem.FHIRPropertyType[key] // empty string if not found
	}
	return ""
}

// resolveFHIRPropertyType returns the FHIR type of a property access on expr.
// Returns "" if the type cannot be determined or is not in the FHIR property map.
func (t *Translator) resolveFHIRPropertyType(source ast.Expr, path string) string {
	sourceType := t.resolveFHIRSourceType(source)
	if sourceType == "" {
		return ""
	}
	return typesystem.FHIRPropertyType[sourceType+"."+path]
}

// resolveFHIRPropertyCoercion returns the FHIRHelpers function name (e.g. "ToString",
// "ToDateTime") to coerce the result of accessing `path` on `source`, or "" if no
// coercion is needed. Coercion is only applied for FHIR primitive types.
func (t *Translator) resolveFHIRPropertyCoercion(source ast.Expr, path string) string {
	fhirType := t.resolveFHIRPropertyType(source, path)
	if fhirType == "" {
		return ""
	}
	return typesystem.FHIRPrimitiveCoercion[fhirType]
}

func (t *Translator) translateTypeIs(v *ast.TypeIsExpr) elm.Expression {
	ann := t.cqfAnnotation()
	operand := t.translateExpr(v.Operand)
	loc := ""
	if t.opts.EnableLocators {
		loc = locatorStr(v.Loc())
	}
	if v.IsNull {
		isNull := &elm.UnaryExpressionNode{
			Annotation: ann,
			Signature:  t.computeSig("IsNull", []elm.Expression{operand}),
			Operator:   "IsNull",
			Operand:    operand,
		}
		if v.Negated {
			if loc != "" {
				isNull.Locator = loc
			}
			return &elm.UnaryExpressionNode{
				Annotation: ann,
				Signature:  t.computeSig("Not", []elm.Expression{isNull}),
				Operator:   "Not",
				Operand:    isNull,
			}
		}
		return isNull
	}
	if v.IsTrue {
		op := "IsTrue"
		node := &elm.UnaryExpressionNode{Annotation: ann, Signature: t.computeSig(op, []elm.Expression{operand}), Operator: op, Operand: operand}
		if v.Negated {
			if loc != "" {
				node.Locator = loc
			}
			return &elm.UnaryExpressionNode{Annotation: ann, Signature: t.computeSig("Not", []elm.Expression{node}), Operator: "Not", Operand: node}
		}
		return node
	}
	if v.IsFalse {
		op := "IsFalse"
		node := &elm.UnaryExpressionNode{Annotation: ann, Signature: t.computeSig(op, []elm.Expression{operand}), Operator: op, Operand: operand}
		if v.Negated {
			if loc != "" {
				node.Locator = loc
			}
			return &elm.UnaryExpressionNode{Annotation: ann, Signature: t.computeSig("Not", []elm.Expression{node}), Operator: "Not", Operand: node}
		}
		return node
	}
	ts := t.translateTypeSpecifier(v.TypeSpec)
	return &elm.IsNode{
		Annotation:      ann,
		Signature:       t.cqfEmptyArrayField(),
		Operand:         operand,
		IsTypeSpecifier: ts,
	}
}

func (t *Translator) translateTimingExpr(v *ast.TimingExpr) elm.Expression {
	op := v.Op

	// IncludedIn → In: when FHIRHelpers is included and the LHS is a FHIR dateTime/instant
	// property (which coerces to a DateTime point via FHIRHelpers.ToDateTime), the during
	// operator should produce point-in-interval "In" rather than interval-in-interval "IncludedIn".
	if op == "IncludedIn" && t.fhirHelpersLocalName != "" {
		if pe, ok := v.Left.(*ast.PropertyExpr); ok {
			if fhirType := t.resolveFHIRPropertyType(pe.Source, pe.Path); typesystem.IsFHIRDateTimeType(fhirType) {
				op = "In"
			}
		}
	}

	operands := []elm.Expression{t.translateExpr(v.Left), t.translateExpr(v.Right)}
	if v.Precision != "" {
		return &elm.PrecisionOperatorNode{
			Annotation: t.cqfAnnotation(),
			Signature:  t.computeSig(op, operands),
			Operator:   op,
			Precision:  v.Precision,
			Operand:    operands,
		}
	}
	return &elm.OperatorExpressionNode{
		Annotation: t.cqfAnnotation(),
		Signature:  t.computeSig(op, operands),
		Operator:   op,
		Operand:    operands,
	}
}

func (t *Translator) translateRetrieve(v *ast.RetrieveExpr) elm.Expression {
	dt, tid := t.qualifyDataType(v.DataType)
	r := &elm.RetrieveNode{
		Annotation:  t.cqfAnnotation(),
		Include:     t.cqfEmptyArrayField(),
		CodeFilter:  t.cqfEmptyArrayField(),
		DateFilter:  t.cqfEmptyArrayField(),
		OtherFilter: t.cqfEmptyArrayField(),
		DataType:    dt,
		TemplateID:  tid,
	}
	if v.Codes != nil {
		r.Codes = t.translateExpr(v.Codes)
		r.CodeProperty = v.CodeProperty
	}
	return r
}

func (t *Translator) translateQuery(q *ast.QueryExpression) elm.Expression {
	// Build the alias set for this query scope.
	aliases := make(map[string]bool)
	for _, src := range q.Sources {
		if src.Alias != "" {
			aliases[src.Alias] = true
		}
	}
	for _, rel := range q.Relationship {
		if rel.Source != nil && rel.Source.Alias != "" {
			aliases[rel.Source.Alias] = true
		}
	}
	t.queryAliases = append(t.queryAliases, aliases)
	defer func() { t.queryAliases = t.queryAliases[:len(t.queryAliases)-1] }()

	// Build alias→FHIR-type map for implicit FHIRHelpers coercion, and pre-translate
	// each source expression so we can both register the alias' element type for
	// signature inference and reuse the translated node below.
	aliasTypes := make(map[string]string)
	aliasTypeSpecs := make(map[string]typeSpec)
	translatedSources := make(map[int]elm.Expression, len(q.Sources))
	for i, src := range q.Sources {
		if src.Alias == "" {
			continue
		}
		if t.fhirHelpersLocalName != "" {
			if re, ok := src.Expression.(*ast.RetrieveExpr); ok && re.DataType != "" {
				// DataType format: "FHIR.Encounter" — extract after the last dot.
				typeName := re.DataType
				if i := strings.LastIndex(typeName, "."); i >= 0 {
					typeName = typeName[i+1:]
				}
				aliasTypes[src.Alias] = typeName
			}
		}
		srcExpr := t.translateExpr(src.Expression)
		translatedSources[i] = srcExpr
		if lt, ok := t.inferTypeSpec(srcExpr).(listTS); ok {
			aliasTypeSpecs[src.Alias] = lt.elem
		}
	}
	t.queryAliasTypes = append(t.queryAliasTypes, aliasTypes)
	defer func() { t.queryAliasTypes = t.queryAliasTypes[:len(t.queryAliasTypes)-1] }()
	t.queryAliasTypeSpecs = append(t.queryAliasTypeSpecs, aliasTypeSpecs)
	defer func() { t.queryAliasTypeSpecs = t.queryAliasTypeSpecs[:len(t.queryAliasTypeSpecs)-1] }()

	// Build the let-variable set for this query scope (for QueryLetRef resolution).
	letScope := make(map[string]bool)
	for _, let := range q.Let {
		letScope[let.Identifier] = true
	}
	t.queryLetScopes = append(t.queryLetScopes, letScope)
	defer func() { t.queryLetScopes = t.queryLetScopes[:len(t.queryLetScopes)-1] }()
	letTypeSpecs := make(map[string]typeSpec)
	t.queryLetTypeSpecs = append(t.queryLetTypeSpecs, letTypeSpecs)
	defer func() { t.queryLetTypeSpecs = t.queryLetTypeSpecs[:len(t.queryLetTypeSpecs)-1] }()

	qn := &elm.QueryNode{
		Annotation:   t.cqfAnnotation(),
		Let:          []*elm.LetClauseELM{},
		Relationship: []*elm.RelationshipClauseELM{},
	}
	for i, src := range q.Sources {
		expr, ok := translatedSources[i]
		if !ok {
			expr = t.translateExpr(src.Expression)
		}
		aqse := &elm.AliasedQuerySourceELM{
			Annotation: t.cqfAnnotation(),
			Alias:      src.Alias,
			Expression: expr,
		}
		if t.opts.EnableLocators {
			aqse.Locator = locatorStr(src.Loc())
		}
		qn.Source = append(qn.Source, aqse)
	}
	for _, let := range q.Let {
		letExpr := t.translateExpr(let.Expression)
		letTypeSpecs[let.Identifier] = t.inferTypeSpec(letExpr)
		lce := &elm.LetClauseELM{
			Annotation: t.cqfAnnotation(),
			Identifier: let.Identifier,
			Expression: letExpr,
		}
		if t.opts.EnableLocators {
			lce.Locator = locatorStr(let.Loc())
		}
		qn.Let = append(qn.Let, lce)
	}
	for _, rel := range q.Relationship {
		kind := "With"
		if rel.Kind == "without" {
			kind = "Without"
		}
		r := &elm.RelationshipClauseELM{
			Annotation: t.cqfAnnotation(),
			Kind:       kind,
			Alias:      rel.Source.Alias,
			Expression: t.translateExpr(rel.Source.Expression),
		}
		if rel.Source.Alias != "" {
			if lt, ok := t.inferTypeSpec(r.Expression).(listTS); ok {
				aliasTypeSpecs[rel.Source.Alias] = lt.elem
			}
		}
		r.SuchThat = t.translateExpr(rel.SuchThat)
		if t.opts.EnableLocators {
			r.Locator = locatorStr(rel.Loc())
		}
		qn.Relationship = append(qn.Relationship, r)
	}
	if q.Where != nil {
		qn.Where = t.translateExpr(q.Where)
	}
	if q.Return != nil {
		var distinct *bool
		// Only emit distinct:false for explicit 'return all'; plain 'return' omits the field.
		if q.Return.Distinct != nil && !*q.Return.Distinct {
			f := false
			distinct = &f
		}
		rc := &elm.ReturnClauseELM{
			Annotation: t.cqfAnnotation(),
			Distinct:   distinct,
			Expression: t.translateExpr(q.Return.Expression),
		}
		if t.opts.EnableLocators {
			rc.Locator = locatorStr(q.Return.Loc())
		}
		qn.Return = rc
	}
	if q.Aggregate != nil {
		qn.Aggregate = &elm.AggregateClauseELM{
			Distinct:   q.Aggregate.Distinct,
			Identifier: q.Aggregate.Identifier,
			Expression: t.translateExpr(q.Aggregate.Expression),
			Starting:   t.translateExpr(q.Aggregate.Starting),
		}
	}
	if q.Sort != nil {
		sortClause := &elm.SortClauseELM{Annotation: t.cqfAnnotation()}
		if t.opts.EnableLocators {
			sortClause.Locator = locatorStr(q.Sort.Loc())
		}
		for _, item := range q.Sort.Items {
			dir := item.DirectionText
			if dir == "" {
				dir = "ascending"
				if item.Direction == ast.SortDesc {
					dir = "descending"
				}
			}
			sortItem := &elm.SortByItemELM{
				Annotation: t.cqfAnnotation(),
				Direction:  dir,
			}
			if t.opts.EnableLocators {
				sortItem.Locator = locatorStr(item.Loc())
			}
			if item.Expression == nil {
				// `sort asc` / `sort desc` — direction only, no by-expression.
			} else if id, ok := item.Expression.(*ast.IdentifierRef); ok && id != nil {
				sortItem.Path = id.Name
			} else {
				sortItem.Expression = t.translateExpr(item.Expression)
			}
			sortClause.By = append(sortClause.By, sortItem)
		}
		qn.Sort = sortClause
	}
	return qn
}

// isDecimalListExpr reports whether expr is a list literal whose elements
// are all Decimal literals (or Null). CQF only coerces Avg/Median/etc. sources
// via a ToDecimal query when the element type isn't already Decimal.
func (t *Translator) isDecimalListExpr(expr ast.Expr) bool {
	le, ok := expr.(*ast.ListExpr)
	if !ok || len(le.Elements) == 0 {
		return false
	}
	sawDecimal := false
	for _, e := range le.Elements {
		switch e.(type) {
		case *ast.DecimalLiteral:
			sawDecimal = true
		case *ast.NullLiteral:
			// permitted, but doesn't establish type on its own
		default:
			return false
		}
	}
	return sawDecimal
}

// wrapInDecimalQuery wraps a list expression in a Query that maps each element
// through ToDecimal. CQF always does this for Avg/Median/StdDev/Variance
// regardless of input element type, coercing List<T> → List<Decimal>.
func (t *Translator) wrapInDecimalQuery(src elm.Expression) elm.Expression {
	const alias = "X"
	// Register the alias element type so inferTypeSpec(AliasRef) returns the
	// correct type when computing the inner ToDecimal signature.
	aliasTypeSpecs := map[string]typeSpec{}
	if lt, ok := t.inferTypeSpec(src).(listTS); ok {
		aliasTypeSpecs[alias] = lt.elem
	} else {
		aliasTypeSpecs[alias] = anyTS
	}
	t.queryAliasTypeSpecs = append(t.queryAliasTypeSpecs, aliasTypeSpecs)
	defer func() { t.queryAliasTypeSpecs = t.queryAliasTypeSpecs[:len(t.queryAliasTypeSpecs)-1] }()

	aliasRef := &elm.AliasRefNode{Annotation: t.cqfAnnotation(), Name: alias}
	return &elm.QueryNode{
		Annotation: t.cqfAnnotation(),
		Source: []*elm.AliasedQuerySourceELM{
			{
				Annotation: t.cqfAnnotation(),
				Alias:      alias,
				Expression: src,
			},
		},
		Let:          []*elm.LetClauseELM{},
		Relationship: []*elm.RelationshipClauseELM{},
		Return: &elm.ReturnClauseELM{
			Annotation: t.cqfAnnotation(),
			Distinct:   func() *bool { f := false; return &f }(),
			Expression: &elm.UnaryExpressionNode{
				Annotation: t.cqfAnnotation(),
				Signature:  t.computeSig("ToDecimal", []elm.Expression{aliasRef}),
				Operator:   "ToDecimal",
				Operand:    aliasRef,
			},
		},
	}
}

// inferListElementType infers the qualified ELM type name from a list of
// expressions by looking at the type of the first non-null literal element.
// Returns "" if the type cannot be determined.
func (t *Translator) inferListElementType(elems []ast.Expr) string {
	for _, e := range elems {
		switch e.(type) {
		case *ast.NullLiteral:
			continue
		case *ast.IntegerLiteral:
			return "{urn:hl7-org:elm-types:r1}Integer"
		case *ast.LongLiteral:
			return "{urn:hl7-org:elm-types:r1}Long"
		case *ast.DecimalLiteral:
			return "{urn:hl7-org:elm-types:r1}Decimal"
		case *ast.StringLiteral:
			return "{urn:hl7-org:elm-types:r1}String"
		case *ast.BooleanLiteral:
			return "{urn:hl7-org:elm-types:r1}Boolean"
		case *ast.DateLiteral:
			return "{urn:hl7-org:elm-types:r1}Date"
		case *ast.DateTimeLiteral:
			return "{urn:hl7-org:elm-types:r1}DateTime"
		case *ast.TimeLiteral:
			return "{urn:hl7-org:elm-types:r1}Time"
		}
	}
	return ""
}

func titleCase(s string) string {
	if len(s) == 0 {
		return s
	}
	lower := strings.ToLower(s)
	return strings.ToUpper(lower[:1]) + lower[1:]
}

// buildStatementAnnotation builds the ELM annotation for a statement definition.
// In CQFMode it emits [] (empty) when there are no CQL @tag annotations, or
// produces the structured annotation array when the AST carries @tag annotations.
// CQL compatibility level 1.4 suppresses @tag annotation data (CQF behaviour).
func (t *Translator) buildStatementAnnotation(s *ast.ExpressionDefinition) json.RawMessage {
	if !t.opts.CQFMode && !t.opts.EnableAnnotations {
		return nil
	}
	compat := t.opts.CompatibilityLevel
	emitTagData := compat == "" || compat >= "1.5"

	type sNode struct {
		R     string   `json:"r,omitempty"`
		Value []string `json:"value,omitempty"`
	}
	type sBlock struct {
		R string  `json:"r,omitempty"`
		S []sNode `json:"s,omitempty"`
	}
	type tagJSON struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	type annJSON struct {
		T    []tagJSON `json:"t,omitempty"`
		S    *sBlock   `json:"s,omitempty"`
		Type string    `json:"type"`
	}

	// Build the single Annotation entry that combines @tag annotations and
	// the source-text s-tree (CQF emits both inside one Annotation map).
	combined := annJSON{Type: "Annotation"}
	if len(s.Annotations) > 0 && emitTagData {
		// CQF merges all @tag pairs across preceding block comments into one
		// flat tag list on the single Annotation entry.
		for _, ann := range s.Annotations {
			for _, tag := range ann.Tags {
				combined.T = append(combined.T, tagJSON{Name: tag.Name, Value: tag.Value})
			}
		}
	}
	if t.opts.EnableAnnotations {
		if text := t.sourceSliceWithLeading(s.Loc()); text != "" {
			combined.S = &sBlock{S: []sNode{{Value: []string{text}}}}
		}
	}

	if combined.T == nil && combined.S == nil {
		return t.cqfAnnotation()
	}
	b, err := json.Marshal([]annJSON{combined})
	if err != nil {
		return t.cqfAnnotation()
	}
	return json.RawMessage(b)
}
