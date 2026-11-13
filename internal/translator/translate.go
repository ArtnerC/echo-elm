package translator

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/artnerc/echo-elm/internal/ast"
	"github.com/artnerc/echo-elm/internal/elm"
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

	// IncludeCache, when set, shares what is learned from each included library
	// across every translation that uses it — a run translating many libraries
	// loads each shared helper once. Nil gives each translation its own.
	IncludeCache *IncludeCache
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
	syms                 map[string]symKind                          // symbol table built before statement pass
	paramTypes           map[string]string                           // maps parameter name → ELM qualified type name (named types only)
	paramTypeSpecs       map[string]typeSpec                         // maps parameter name → full typeSpec (named, list, interval)
	defTypeSpecs         map[string]typeSpec                         // maps statement-def name → full typeSpec inferred from its body
	queryAliases         []map[string]bool                           // stack of alias sets for current query scopes
	queryLetScopes       []map[string]bool                           // stack of let-identifier sets for current query scopes
	queryAliasTypes      []map[string]string                         // stack: alias name → FHIR resource type (e.g. "Encounter")
	queryAliasTypeSpecs  []map[string]typeSpec                       // stack: alias name → element typeSpec (for sig inference)
	queryLetTypeSpecs    []map[string]typeSpec                       // stack: let-id → inferred typeSpec for QueryLetRef inference
	modelsByAlias        map[string]string                           // model local-identifier → model URI (populated per Translate call)
	primaryModelURI      string                                      // URI of the first non-System declared model
	primaryModelName     string                                      // original model name of the first non-System declared model (e.g. "QUICK", "FHIR")
	currentContextName   string                                      // context being translated (for age function expansion)
	functionParamScope   map[string]bool                             // set of operand (parameter) names in the current function body
	operandTypeSpecs     map[string]typeSpec                         // function operand name → declared typeSpec, set during function body translation
	listNodeTypes        map[*elm.ListNode]typeSpec                  // typed-list literals: ListNode pointer → element typeSpec (does not serialize)
	fhirHelpersLocalName string                                      // local identifier of included FHIRHelpers library, or "" if not included
	queryTypes           map[*elm.QueryNode]typeSpec                 // each query's type, recorded while its aliases were in scope
	includeAliases       map[string]bool                             // every include's local name, whether or not its source resolved
	libSyms              map[string]map[string]symKind               // included library alias → name → symKind (for QualifiedRef resolution)
	libFluentFuncs       map[string][]string                         // fluent function name → include aliases declaring it (for `receiver.Func()` resolution)
	localOverloads       map[string][]*funcOverload                  // local function name → every overload (see overloads.go)
	overloadOf           map[*ast.ExpressionDefinition]*funcOverload // function definition → its overload entry, to record the inferred return type
	libOverloads         map[string]map[string][]*funcOverload       // include alias → function name → every overload
	funcReturnSpecs      map[string]typeSpec                         // local function name → inferred return type (unambiguous names only)
	localFuncOperands    map[string][]typeSpec                       // local function name → declared operand types (unambiguous names only)
	libFuncReturns       map[string]map[string]typeSpec              // included library alias → function name → inferred return type
	skipLocatorStamp     bool                                        // when true, translateExpr skips stamping locator on its outer result (set by callees that placed locator on an inner node)
	inSortScope          bool                                        // when true, identifiers resolve to IdentifierRef (sort-by expressions are scoped to the query result element)
	noFHIRCoerce         bool                                        // when true, the next expression is in an unconsumed position and skips implicit FHIRHelpers coercion
	localFuncCounts      map[string]int                              // local function name → number of definitions (>1 means overloaded)
	localFuncReturns     map[string]typeSpec                         // local function name → declared return type
	libDefTypes          map[string]map[string]typeSpec              // included library alias → definition name → inferred type
	defContexts          map[string]string                           // definition name → the context it was declared in
	sortElementTS        typeSpec                                    // result element type of the query whose sort clause is being translated
	anyAccumulators      map[string]bool                             // aggregate accumulators with no starting value, whose type is Any
	libFuncOperands      map[string]map[string][]typeSpec            // included library alias → function name → declared operand types
	includeVisiting      map[string]bool                             // include path|version currently being typed in this chain, to stop cycles
	includeTruncated     bool                                        // an include below this translation was cut short by a cycle
	converted            map[elm.Expression]bool                     // conversions and casts the translator synthesized
	elmDiagnostics       []*elm.CqlToElmError                        // CqlToElmError annotations, in the order they arise
	libraryID            string                                      // this library's identifier and version, for diagnostics
	libraryVersion       string
	includeLibraryNames  map[string]string         // include alias → the included library's identifier
	libFuncCounts        map[string]map[string]int // include alias → function name → number of definitions
	skipResultTypeStamp  bool                      // when true, translateExpr skips stamping a result type on its outer result (set by callees that stamped an inner node instead)
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
	if v.Kind() != reflect.Pointer || v.IsNil() {
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

// clearLocator removes the Locator field value from an ELM expression node (if present).
// Used to match CQF behavior in cases where a locator is not emitted (e.g. parameter defaults).
func clearLocator(e elm.Expression) {
	if e == nil {
		return
	}
	v := reflect.ValueOf(e)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return
	}
	f := v.FieldByName("Locator")
	if f.IsValid() && f.CanSet() && f.Kind() == reflect.String {
		f.SetString("")
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
//
// The names and their order mirror the CqlTranslatorOptions.Options enum that
// CQF serializes, so that a consumer reading the ELM sees the same description
// of how it was produced. Only non-default options appear. ValidateUnits has no
// counterpart in that enum and so is never reported.
func (t *Translator) optionsString() string {
	var parts []string
	for _, o := range []struct {
		on   bool
		name string
	}{
		{t.opts.EnableAnnotations, "EnableAnnotations"},
		{t.opts.EnableLocators, "EnableLocators"},
		{t.opts.EnableResultTypes, "EnableResultTypes"},
		{t.opts.DisableListTraversal, "DisableListTraversal"},
		{t.opts.DisableListDemotion, "DisableListDemotion"},
		{t.opts.DisableListPromotion, "DisableListPromotion"},
		{t.opts.EnableIntervalDemotion, "EnableIntervalDemotion"},
		{t.opts.EnableIntervalPromotion, "EnableIntervalPromotion"},
		{t.opts.DisableMethodInvocation, "DisableMethodInvocation"},
		{t.opts.RequireFromKeyword, "RequireFromKeyword"},
	} {
		if o.on {
			parts = append(parts, o.name)
		}
	}
	return strings.Join(parts, ",")
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
			if ts := t.declaredTypeSpec(*p.ParameterType); ts != nil {
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

	// Count local function definitions per name so that SignatureLevel=Overloads
	// can tell an overloaded call from an unambiguous one, and record each
	// function's declared return type for calls to resolve against.
	t.localFuncCounts = make(map[string]int)
	t.localFuncReturns = make(map[string]typeSpec)
	t.funcReturnSpecs = make(map[string]typeSpec)
	t.queryTypes = make(map[*elm.QueryNode]typeSpec)
	t.localFuncOperands = make(map[string][]typeSpec)
	t.localOverloads = make(map[string][]*funcOverload)
	t.overloadOf = make(map[*ast.ExpressionDefinition]*funcOverload)
	t.defContexts = make(map[string]string)
	for _, s := range lib.Statements {
		t.defContexts[s.Name] = s.Context
	}
	for _, s := range lib.Statements {
		if !s.IsFunction {
			continue
		}
		t.localFuncCounts[s.Name]++
		ov := &funcOverload{operands: make([]typeSpec, len(s.Operands)), fluent: s.IsFluent}
		for i, op := range s.Operands {
			if op.Type != nil {
				ov.operands[i] = t.declaredTypeSpec(*op.Type)
			}
		}
		if s.ReturnType != nil {
			if ts := t.declaredTypeSpec(*s.ReturnType); ts != nil {
				t.localFuncReturns[s.Name] = ts
				ov.returns = ts
			}
		}
		t.recordLocalOverload(s.Name, ov)
		t.overloadOf[s] = ov
		// Declared operand types, for casting untyped arguments at call sites.
		// Only unambiguous names: with overloads the parameter list depends on
		// which one the call resolves to, which is not decided here.
		if t.localFuncCounts[s.Name] > 1 {
			delete(t.localFuncOperands, s.Name)
		} else {
			specs := make([]typeSpec, len(s.Operands))
			for i, op := range s.Operands {
				if op.Type != nil {
					specs[i] = t.declaredTypeSpec(*op.Type)
				}
			}
			t.localFuncOperands[s.Name] = specs
		}
	}

	// Build cross-library symbol tables for QualifiedRef resolution.
	t.libSyms = make(map[string]map[string]symKind)
	// Every include's alias is a library qualifier whether or not its source can
	// be found: `FHIRHelpers.ToString(x)` is a library call either way. Keying
	// this off libSyms, which only holds includes that resolved, turned such a
	// call into a method invocation on an identifier named FHIRHelpers — an
	// ExpressionRef operand and no libraryName. (issues/06)
	t.includeAliases = make(map[string]bool, len(lib.Includes))
	t.includeLibraryNames = make(map[string]string, len(lib.Includes))
	t.libFuncCounts = make(map[string]map[string]int)
	t.elmDiagnostics = nil
	for _, inc := range lib.Includes {
		alias := inc.LocalName
		if alias == "" {
			alias = inc.Path
		}
		t.includeAliases[alias] = true
		name := inc.Path
		if i := strings.LastIndex(name, "."); i >= 0 {
			name = name[i+1:]
		}
		t.includeLibraryNames[alias] = name
	}
	if lib.Name != nil {
		t.libraryID, t.libraryVersion = lib.Name.Name, lib.Name.Version
	}
	t.libDefTypes = make(map[string]map[string]typeSpec)
	t.libFluentFuncs = make(map[string][]string)
	t.libFuncReturns = make(map[string]map[string]typeSpec)
	t.libFuncOperands = make(map[string]map[string][]typeSpec)
	t.libOverloads = make(map[string]map[string][]*funcOverload)
	if t.opts.LibrarySource != nil {
		if t.opts.IncludeCache == nil {
			t.opts.IncludeCache = NewIncludeCache()
		}
		for _, inc := range lib.Includes {
			localName := inc.LocalName
			if localName == "" {
				localName = inc.Path
			}
			il := t.loadInclude(inc.Path, inc.Version)
			if il == nil {
				continue
			}
			t.libSyms[localName] = il.syms
			// `receiver.Func()` can only resolve to a fluent function, so
			// record which includes declare one under each name.
			for _, name := range il.fluent {
				if !containsString(t.libFluentFuncs[name], localName) {
					t.libFluentFuncs[name] = append(t.libFluentFuncs[name], localName)
				}
			}
			// A qualified reference resolves to the type the included library
			// gives that definition. Those types are needed for implicit
			// conversions as well as result types, so they are not gated on
			// --result-types.
			if il.types != nil {
				t.libDefTypes[localName] = il.types.defs
				t.libFuncReturns[localName] = il.types.funcs
				t.libFuncOperands[localName] = il.types.operands
				t.libOverloads[localName] = overloadsUnder(il.types.overloads, localName)
			}
			t.libFuncCounts[localName] = il.funcCounts
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

	// CqlToElmInfo annotation. CQF carries no compatibilityLevel here — the
	// level shapes the translation but is not reported in the ELM.
	info := &elm.CqlToElmInfo{
		TranslatorVersion: t.opts.TranslatorVersion,
		TranslatorOptions: t.optionsString(),
		SignatureLevel:    t.opts.SignatureLevel,
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
			T    json.RawMessage `json:"t,omitempty"`
			S    *sBlock         `json:"s,omitempty"`
			Type string          `json:"type"`
		}
		ann := annJSON{Type: "Annotation", T: t.cqfEmptyArrayField(), S: &sBlock{S: []sNode{{Value: []string{text}}}}}
		if b, mErr := json.Marshal(ann); mErr == nil {
			out.Annotation = append(out.Annotation, json.RawMessage(b))
		}
	}
	// Tags in the block comment before `library` (FHIRHelpers' @author and
	// @description) annotate the library itself. (issues/08 #15)
	if tagged := t.withTags(nil, lib.Annotations); tagged != nil {
		if len(out.Annotation) > 1 {
			if merged := t.withTags(json.RawMessage("["+string(out.Annotation[len(out.Annotation)-1])+"]"), lib.Annotations); merged != nil {
				var one []json.RawMessage
				if json.Unmarshal(merged, &one) == nil && len(one) == 1 {
					out.Annotation[len(out.Annotation)-1] = one[0]
				}
			}
		} else {
			var entries []json.RawMessage
			if json.Unmarshal(tagged, &entries) == nil {
				out.Annotation = append(out.Annotation, entries...)
			}
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
				LocalID:        t.defID(),
				ResultTypeName: t.declResultType(typesystem.TypeCodeSystem),
				Annotation:     t.mergeAnnotations(t.cqfAnnotation(), t.srcAnnotationJSON(cs.Loc())),
				Name:           cs.Name,
				ID:             cs.ID,
				Version:        cs.Version,
				AccessLevel:    accessLevelStr(cs.AccessLevel),
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
				LocalID:        t.defID(),
				ResultTypeName: t.declResultType(typesystem.TypeValueSet),
				Annotation:     t.mergeAnnotations(t.cqfAnnotation(), t.srcAnnotationJSON(vs.Loc())),
				Name:           vs.Name,
				ID:             vs.ID,
				Version:        vs.Version,
				AccessLevel:    accessLevelStr(vs.AccessLevel),
				CodeSystems:    json.RawMessage("[]"),
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
				Annotation:     t.cqfAnnotation(),
				Name:           c.SystemName,
				ResultTypeName: t.declResultType(typesystem.TypeCodeSystem),
			}
			if t.opts.EnableLocators {
				csRef.Locator = locatorStr(c.SystemLocator)
			}
			cd := &elm.CodeDef{
				LocalID:        t.defID(),
				ResultTypeName: t.declResultType(typesystem.TypeCode),
				Annotation:     t.mergeAnnotations(t.cqfAnnotation(), t.srcAnnotationJSON(c.Loc())),
				Name:           c.Name,
				ID:             c.Code,
				Display:        c.Display,
				AccessLevel:    accessLevelStr(c.AccessLevel),
				CodeSystem:     csRef,
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
				LocalID:        t.defID(),
				ResultTypeName: t.declResultType(typesystem.TypeConcept),
				Annotation:     t.mergeAnnotations(t.cqfAnnotation(), t.srcAnnotationJSON(con.Loc())),
				Name:           con.Name,
				Display:        con.Display,
				AccessLevel:    accessLevelStr(con.AccessLevel),
			}
			if t.opts.EnableLocators {
				cond.Locator = locatorStr(con.Loc())
			}
			for _, cref := range con.Codes {
				cr := &elm.CodeRef{
					Name:           cref.Name,
					LibraryName:    cref.LibraryName,
					ResultTypeName: t.declResultType(typesystem.TypeCode),
				}
				if t.opts.EnableLocators {
					cr.Locator = locatorStr(cref.Locator)
				}
				cond.Code = append(cond.Code, cr)
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
				Annotation:  t.withTags(t.mergeAnnotations(t.cqfAnnotation(), t.srcAnnotationJSON(p.Loc())), p.Annotations),
			}
			if t.opts.EnableLocators {
				pd.Locator = locatorStr(p.Loc())
			}
			if p.ParameterType != nil {
				pd.ParameterTypeSpecifier = t.translateTypeSpecifier(*p.ParameterType)
				if t.opts.EnableResultTypes {
					stampTypeSpecifier(pd.ParameterTypeSpecifier)
				}
				if t.opts.EnableResultTypes {
					pd.ResultTypeName, pd.ResultTypeSpecifier = resultTypeOf(t.declaredTypeSpec(*p.ParameterType))
				}
			}
			if p.Default != nil {
				pd.Default = t.translateExpr(p.Default)
				// When the declared type is Interval<DateTime> but the default bounds are
				// Date literals (e.g. Interval[@2024-01-01, @2024-12-31]), CQF inserts an
				// implicit ToDateTime promotion around each Date bound, and the interval
				// it emits is then the rebuilt one: no locator and no result type.
				//
				// Every other default keeps its locator. This used to clear it
				// unconditionally, because the one default shape the corpus had was
				// this promoted one. (issues/07)
				if p.ParameterType != nil && t.defaultNeedsDateTimePromotion(*p.ParameterType, pd.Default) {
					pd.Default = t.promoteIntervalDateBoundsToDateTime(*p.ParameterType, pd.Default)
					clearLocator(pd.Default)
					elm.SetResultType(pd.Default, "", nil)
				}
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

	// Implicit context accessor for each unique context that (a) is not
	// "Unfiltered", and (b) has no explicit definition with the same name.
	//
	// Each accessor is keyed by the source position of its `context` declaration,
	// not prepended: CQF walks declarations in source order and creates the
	// accessor when it reaches the declaration, so a define written *above*
	// `context Patient` is emitted above the Patient accessor.
	accessors := make([]positionedAccessor, 0, len(lib.Contexts))
	seenAccessor := map[string]bool{}
	for _, ctx := range lib.Contexts {
		if ctx.Name == "Unfiltered" || explicitNames[ctx.Name] || seenAccessor[ctx.Name] {
			continue
		}
		seenAccessor[ctx.Name] = true
		accessors = append(accessors, positionedAccessor{
			at:  ctx.Loc().Start,
			def: t.buildContextAccessor(ctx.Name, lib),
		})
	}

	if len(lib.Statements) > 0 || len(accessors) > 0 {
		// Recover library qualifiers before ordering, so the dependency analysis
		// and the emitted nodes agree about what each call refers to.
		t.resolveLibraryQualifiedCalls(lib.Statements)
		ordered := statementEmissionOrder(lib.Statements)
		stmts := &elm.StatementDefs{Def: leadingAccessors(&accessors, ordered)}
		for _, s := range ordered {
			// Emit any accessor whose context declaration precedes this statement
			// in the source before the statement itself.
			stmts.Def = append(stmts.Def, accessorsBefore(&accessors, s.Loc().Start)...)
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
			// resultTypeSpecifier is result-type information, which CQF only emits
			// under --result-types. A declared `returns` clause does not by itself
			// put the type in the ELM.
			if s.ReturnType != nil && t.opts.EnableResultTypes {
				declared := t.declaredTypeSpec(*s.ReturnType)
				sd.ResultTypeName, sd.ResultTypeSpecifier = resultTypeOf(declared)
				// resultTypeOf treats Any as "unresolved", but a declared
				// `returns Any` is resolved — to Any — and CQF records it.
				if sd.ResultTypeName == "" && sd.ResultTypeSpecifier == nil && isAnyTS(declared) {
					sd.ResultTypeName = typesystem.TypeAny
				}
			}
			// Build function parameter scope before translating the body,
			// so IdentifierRef nodes matching operand names emit OperandRef.
			if s.IsFunction && len(s.Operands) > 0 {
				t.functionParamScope = make(map[string]bool, len(s.Operands))
				t.operandTypeSpecs = make(map[string]typeSpec, len(s.Operands))
				for _, op := range s.Operands {
					t.functionParamScope[op.Name] = true
					if op.Type != nil {
						if ts := t.declaredTypeSpec(*op.Type); ts != nil {
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
					if t.opts.EnableResultTypes {
						stampTypeSpecifier(opd.OperandTypeSpecifier)
					}
				}
				sd.Operand = append(sd.Operand, opd)
			}
			switch {
			case s.IsExternal:
				// An external function has no body to translate. Emitting a Null
				// expression in its place gave an engine something to evaluate —
				// null — instead of the call it must make. (issues/08 W12)
				sd.External = true
			case s.Expression != nil:
				sd.Expression = t.translateExprUnconsumed(s.Expression)
			default:
				sd.Expression = &elm.NullNode{Annotation: t.cqfAnnotation()}
			}
			// A declared `returns` type wins; otherwise the definition's type is
			// whatever its body resolved to. Prefer the type already recorded on
			// the body over re-inferring it — a FHIR property reports its FHIR
			// type, not the converted one.
			if t.opts.EnableResultTypes && sd.ResultTypeName == "" && sd.ResultTypeSpecifier == nil {
				if name, spec := elm.GetResultType(sd.Expression); name != "" || spec != nil {
					sd.ResultTypeName, sd.ResultTypeSpecifier = name, spec
				} else {
					sd.ResultTypeName, sd.ResultTypeSpecifier = resultTypeOf(t.inferTypeSpec(sd.Expression))
				}
			}
			if !s.IsFunction {
				if ts := t.inferTypeSpec(sd.Expression); ts != nil {
					t.defTypeSpecs[s.Name] = ts
				}
			} else if ov := t.overloadOf[s]; ov != nil && ov.returns == nil {
				// Every overload gets its body's type, so a call can resolve to the
				// right one by argument type — overloaded or not.
				ov.returns = t.inferTypeSpec(sd.Expression)
			}
			if s.IsFunction && t.localFuncCounts[s.Name] == 1 {
				// Record the inferred return type so callers in *other* libraries
				// can resolve it. defTypeSpecs deliberately excludes functions, so
				// this is a separate map. Overloaded names are skipped: picking
				// between them needs the argument types, and recording whichever
				// came last would be a guess.
				if ts := t.inferTypeSpec(sd.Expression); ts != nil {
					t.funcReturnSpecs[s.Name] = ts
				}
			}
			t.functionParamScope = nil
			t.operandTypeSpecs = nil
			stmts.Def = append(stmts.Def, sd)
		}
		// Any accessor whose declaration sits after every statement.
		stmts.Def = append(stmts.Def, drainAccessors(&accessors)...)
		out.Statements = stmts
	}

	if t.opts.CQFMode {
		// Diagnostics follow CqlToElmInfo directly, ahead of the library's
		// source-text annotation, as CQF orders them.
		if len(t.elmDiagnostics) > 0 && len(out.Annotation) > 0 {
			diags := make([]json.RawMessage, 0, len(t.elmDiagnostics))
			for _, d := range t.elmDiagnostics {
				if b, err := json.Marshal(d); err == nil {
					diags = append(diags, json.RawMessage(b))
				}
			}
			rest := append([]json.RawMessage{}, out.Annotation[1:]...)
			out.Annotation = append(append(out.Annotation[:1], diags...), rest...)
		}
		// Every Element carries an annotation container in CQF's output, empty
		// or not. Filling it here reaches the synthesized nodes too.
		elm.FillEmptyAnnotations(out)
	}
	result.Library = out
	if len(t.diags) > 0 {
		result.Diagnostics = append(result.Diagnostics, t.diags...)
	}
	return result
}

// castNullTo wraps an untyped Null in the As node CQF emits for it, naming the
// declared parameter type. A named type goes in asType; a structured one
// (Interval<Date>, List<Date>) needs asTypeSpecifier.
func (t *Translator) castNullTo(arg elm.Expression, declared typeSpec) elm.Expression {
	as := &elm.AsNode{
		Annotation: t.cqfAnnotation(),
		Signature:  t.cqfEmptyArrayField(),
		Operand:    arg,
	}
	if named, ok := declared.(namedTS); ok {
		as.AsType = named.name
		return t.markConverted(as)
	}
	spec := toELMTypeSpecifier(declared)
	if spec == nil {
		return arg
	}
	// Deliberately not stamped: stampTypeSpecifier is for specifiers written in
	// the source. This As is synthesized, so its target is too, and CQF leaves
	// synthesized specifiers bare.
	as.AsTypeSpecifier = spec
	return t.markConverted(as)
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
	result := t.applyImplicitConversions(t.translateExprCore(expr))
	if ir, ok := expr.(*ast.IdentifierRef); ok && ir.PathRoot {
		// See ast.IdentifierRef.PathRoot: CQF stamps only the outer path node.
		t.skipLocatorStamp, t.skipResultTypeStamp = false, false
		return result
	}
	if th, ok := expr.(*ast.ThisExpr); ok && th.Implicit {
		// A $this the translator supplied has no source of its own.
		t.skipLocatorStamp, t.skipResultTypeStamp = false, false
		return result
	}
	if expr != nil && t.opts.EnableLocators && !t.skipLocatorStamp {
		setLocator(result, locatorStr(expr.Loc()))
	}
	if expr != nil && t.opts.EnableResultTypes && !t.skipResultTypeStamp {
		t.stampResultType(result)
	}
	t.skipLocatorStamp = false
	t.skipResultTypeStamp = false
	return result
}

// typeNullFromSibling records on a null operand the type of the operand it is
// being combined with, which is how CQF types an otherwise untyped null.
func (t *Translator) typeNullFromSibling(target, sibling elm.Expression) {
	if _, isNull := target.(*elm.NullNode); !isNull {
		return
	}
	if elm.HasResultType(target) {
		return
	}
	name, spec := resultTypeOf(t.nodeResultTS(sibling))
	if name == "" && spec == nil {
		return
	}
	elm.SetResultType(target, name, spec)
}

// isDefinitelyAny reports whether a node's Any type is a resolved answer rather
// than a failure to infer: a null literal, or an operator all of whose operands
// are nulls.
func isDefinitelyAny(e elm.Expression) bool {
	switch v := e.(type) {
	case *elm.NullNode:
		return true
	case *elm.NamedOperatorExpressionNode:
		// Message returns the type of its source, so a null source makes it Any.
		return isDefinitelyAnyBranch(v)
	case *elm.OperatorExpressionNode:
		if len(v.Operand) == 0 {
			return false
		}
		// Coalesce yields the common type of its alternatives, so a single
		// untyped null makes the whole thing Any.
		if v.Operator == "Coalesce" {
			// Only a *bare* null: once a null is cast to the type an earlier
			// operand established, it no longer makes the whole thing Any.
			// Counting As(null) here typed Coalesce(Now(), null) as Any where
			// CQF says DateTime. (issues/06)
			for _, op := range v.Operand {
				if _, bare := op.(*elm.NullNode); bare {
					return true
				}
			}
			return false
		}
		for _, op := range v.Operand {
			if !isNullLike(op) {
				return false
			}
		}
		return true
	}
	return false
}

// stampResultType records the node's resolved type, as CQF does under
// --result-types. Only expressions that came from source reach here:
// synthesized nodes (the implicit context accessor, inserted conversions) are
// built directly rather than through translateExpr, and CQF leaves those
// unstamped too.
func (t *Translator) stampResultType(e elm.Expression) {
	if e == nil || elm.HasResultType(e) {
		return
	}
	ts := t.inferTypeSpec(e)
	name, spec := resultTypeOf(ts)
	if name == "" && spec == nil {
		// Any is otherwise treated as "not resolved" and left unstamped, but some
		// expressions really are Any-typed and CQF records them as such: a bare
		// null literal, and an operator over nothing but nulls.
		if isDefinitelyAny(e) || t.definitelyAny(e) {
			elm.SetResultType(e, typesystem.TypeAny, nil)
		}
		return
	}
	elm.SetResultType(e, name, spec)
}

// translateExprUnconsumed translates an expression appearing in a position that
// does not impose a target type — a define body, a query return expression, a
// list element, or the source of a property navigation. CQF only inserts the
// implicit FHIRHelpers conversion where a value is consumed as an operand of a
// typed operator, so these positions keep the raw FHIR property value.
//
// The flag applies to the immediate expression only: translateExprCore clears it
// before descending, so `define X: Patient.gender = 'f'` still coerces the
// property inside the Equal.
func (t *Translator) translateExprUnconsumed(expr ast.Expr) elm.Expression {
	prev := t.noFHIRCoerce
	t.noFHIRCoerce = true
	result := t.translateExpr(expr)
	t.noFHIRCoerce = prev
	return result
}

// translateExprCore converts an AST expression to an ELM expression.
// All expression nodes receive annotation:[] in CQF mode (ELM Element base).
// OperatorExpression subclasses also receive signature:[].
func (t *Translator) translateExprCore(expr ast.Expr) elm.Expression {
	if expr == nil {
		return &elm.NullNode{Annotation: t.cqfAnnotation()}
	}
	// Read and clear the no-coercion flag: it marks this expression's own
	// position, not the positions of its operands.
	noCoerce := t.noFHIRCoerce
	t.noFHIRCoerce = false
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
		// The literal's value is recorded as written: CQF keeps `01` as "01".
		value := v.Text
		if value == "" {
			value = fmt.Sprintf("%d", v.Value)
		}
		return &elm.LiteralNode{Annotation: ann, ValueType: typesystem.TypeInteger, Value: value}
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
		// Both halves of a ratio are quantities; CQF records that on each.
		// Only a quantity written with a unit is treated as a quantity literal in
		// its own right; the bare `5:10` form carries no type of its own, which
		// is the same distinction the locator above uses.
		if t.opts.EnableResultTypes {
			if v.Numerator.Unit != "" {
				num.ResultTypeName = typesystem.TypeQuantity
			}
			if v.Denominator.Unit != "" {
				den.ResultTypeName = typesystem.TypeQuantity
			}
		}
		return &elm.RatioNode{
			Annotation:  ann,
			Numerator:   num,
			Denominator: den,
		}

	// ---- References ----
	case *ast.IdentifierRef:
		// Inside a sort-by expression the scope is the query's result element,
		// which the translator cannot resolve — CQF defers to the engine by
		// emitting IdentifierRef.
		if t.inSortScope {
			ref := &elm.IdentifierRefNode{Annotation: ann, Name: v.Name}
			// A field of a FHIR element is converted like any other property
			// access: `sort by start of period` takes the Start of
			// ToInterval(period), not of the FHIR Period. The conversion is
			// synthesized; the identifier keeps the locator. (issues/07)
			// noCoerce, not t.noFHIRCoerce: translateExprCore has already moved
			// the flag into noCoerce and cleared it. Reading the field meant an
			// unconsumed identifier — the source of `created.value`, or the
			// argument of an explicit FHIRHelpers.ToDateTime(created) — was
			// converted anyway, the latter twice. (issues/08 W13)
			if fhirFunc, fhirType := t.sortFieldCoercion(v.Name, noCoerce); fhirFunc != "" {
				if t.opts.EnableLocators {
					setLocator(ref, locatorStr(v.Loc()))
					t.skipLocatorStamp = true
				}
				t.skipResultTypeStamp = true
				if t.opts.EnableResultTypes {
					elm.SetResultType(ref, "{http://hl7.org/fhir}"+fhirType, nil)
				}
				t.noteOverloadedCall(t.fhirHelpersLocalName, fhirFunc)
				return t.markConverted(&elm.FunctionRefNode{
					Annotation:  ann,
					Signature:   t.buildSigFromTypes("FunctionRef", []typeSpec{namedTS{"{http://hl7.org/fhir}" + fhirType}}),
					Name:        fhirFunc,
					LibraryName: t.fhirHelpersLocalName,
					Operand:     []elm.Expression{ref},
				})
			}
			return ref
		}
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
				return t.aliasRefNode(ann, v.Name, locatorStr(v.Loc()), noCoerce)
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
			return t.newValueSetRef(ann, v.Name, "")
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
		// Resolve the kind of the referenced symbol in the included library.
		if libMap, ok := t.libSyms[v.LibraryName]; ok {
			switch libMap[v.Name] {
			case symValueSet:
				return t.newValueSetRef(ann, v.Name, v.LibraryName)
			case symCodeSystem:
				return &elm.CodeSystemRefNode{Annotation: ann, LibraryName: v.LibraryName, Name: v.Name}
			case symCode:
				return &elm.CodeRefNode{Annotation: ann, LibraryName: v.LibraryName, Name: v.Name}
			case symConcept:
				return &elm.ConceptRefNode{Annotation: ann, LibraryName: v.LibraryName, Name: v.Name}
			case symParameter:
				return &elm.ParameterRefNode{Annotation: ann, LibraryName: v.LibraryName, Name: v.Name}
			}
		}
		return &elm.ExpressionRefNode{Annotation: ann, Name: v.Name, LibraryName: v.LibraryName}
	case *ast.AliasRef:
		return t.aliasRefNode(ann, v.Name, locatorStr(v.Loc()), noCoerce)
	case *ast.LetRef:
		return &elm.LetRefNode{Annotation: ann, Name: v.Name}
	case *ast.ThisExpr:
		if t.inSortScope {
			return &elm.IdentifierRefNode{Annotation: ann, Name: "$this"}
		}
		return &elm.QueryThisRefNode{Annotation: ann}
	case *ast.IndexExpr:
		return &elm.OperatorExpressionNode{Annotation: ann, Signature: sig, Operator: "QueryIndexRef"}
	case *ast.TotalExpr:
		return &elm.OperatorExpressionNode{Annotation: ann, Signature: sig, Operator: "Total"}
	case *ast.ExternalConstantExpr:
		return &elm.ExternalConstantNode{Annotation: ann, Name: v.Name}
	case *ast.FunctionRef:
		// FHIRPath-style method invocations `.descendents()` / `.children()`
		// name the system operators Descendents and Children.
		if v.LibraryName == "" && len(v.Operands) == 1 {
			switch v.Name {
			case "descendents":
				v.Name = "Descendents"
			case "children":
				v.Name = "Children"
			}
		}
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
			// Now, Today and TimeOfDay are ELM operators in their own right,
			// taking no operands. Emitted as a FunctionRef they name a library
			// function that does not exist, which no engine can evaluate. A local
			// function of the same name still wins. (issues/06)
			if nullarySystemOps[v.Name] && len(v.Operands) == 0 && v.LibraryName == "" && t.localFuncCounts[v.Name] == 0 {
				return &elm.NullaryOperatorNode{
					Annotation: ann,
					Signature:  t.cqfEmptyArrayField(),
					Operator:   v.Name,
				}
			}
			// Map unary system operators to UnaryExpressionNode — unless this
			// library defines a function of the same name that fits the argument.
			// FHIRHelpers calls its own ToQuantity(FHIR.Quantity) on Range.low;
			// the System ToQuantity takes no FHIR type, so CQF resolves to the
			// local function and emitting the operator was a different call.
			// (issues/08 #15)
			if unarySystemOps[v.Name] && len(v.Operands) == 1 && !t.localFunctionFits(v.Name, v.Operands) {
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
						{Name: "endIndex", Value: t.typedNull(intTS)},
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
						{Name: "endIndex", Value: t.typedNull(intTS)},
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
				// A list of FHIR values is converted element-wise first — that
				// is the conversion the typed overload needs. Only a list that is
				// already System-typed falls through to the decimal widening,
				// which otherwise turned Avg over FHIR Quantity into Avg over
				// Decimal. (issues/06 follow-up)
				if lifted, ok := t.liftFHIRListOperand(v.Name, src); ok {
					src = lifted
				} else if decimalAggregateOps[v.Name] && !t.isDecimalListExpr(v.Operands[0]) &&
					!isQuantityList(t.inferTypeSpec(src)) {
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
							val = t.markConverted(&elm.UnaryExpressionNode{
								Annotation: t.cqfAnnotation(),
								Signature:  t.computeSig("ToDecimal", []elm.Expression{val}),
								Operator:   "ToDecimal",
								Operand:    val,
							})
						}
					}
					// IndexOf has only List<T> overload in ELM; for String args
					// CQF wraps the source in ToList (String → List<String>).
					if v.Name == "IndexOf" && i == 0 {
						if _, isStr := o.(*ast.StringLiteral); isStr {
							val = t.markConverted(&elm.UnaryExpressionNode{
								Annotation: t.cqfAnnotation(),
								Signature:  t.cqfEmptyArrayField(),
								Operator:   "ToList",
								Operand:    val,
							})
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
							ops[i] = t.markConverted(&elm.UnaryExpressionNode{
								Annotation: t.cqfAnnotation(),
								Signature:  t.computeSig("ToDecimal", []elm.Expression{ops[i]}),
								Operator:   "ToDecimal",
								Operand:    ops[i],
							})
						}
					}
				}
				// Contains has only List<T> overload in ELM; for String args CQF
				// wraps the source string in ToList.
				if v.Name == "Contains" {
					if _, isStr := v.Operands[0].(*ast.StringLiteral); isStr {
						ops[0] = t.markConverted(&elm.UnaryExpressionNode{
							Annotation: t.cqfAnnotation(),
							Signature:  t.cqfEmptyArrayField(),
							Operator:   "ToList",
							Operand:    ops[0],
						})
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
				// The type an earlier operand established is read from the
				// translated node, so calls and structured types count. The
				// syntactic probe used before only recognised literals, which
				// left Coalesce(Now(), null) with a bare Null. (issues/06 4.1)
				var established typeSpec
				for _, o := range v.Operands {
					elem := t.translateExpr(o)
					if _, isNull := o.(*ast.NullLiteral); isNull {
						if established != nil {
							elem = t.castNullTo(elem, established)
						}
					} else if established == nil {
						established = knownTS(t.inferTypeSpec(elem))
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
			if t.localFuncCounts[v.Name] == 0 {
				if calc := t.calculateAgeCall(ann, v.Name, v.Operands); calc != nil {
					return calc
				}
			}
		}
		declared := t.localFuncOperands[v.Name]
		if v.LibraryName != "" {
			declared = t.libFuncOperands[v.LibraryName][v.Name]
		}
		operands := make([]elm.Expression, 0, len(v.Operands))
		for i, o := range v.Operands {
			var arg elm.Expression
			if v.LibraryName != "" && v.LibraryName == t.fhirHelpersLocalName {
				// FHIRHelpers functions take FHIR types: their argument is what
				// gets converted, not a site that needs converting. Treating it as
				// consumed converted it twice — ToInterval(ToInterval(E.period)).
				// (issues/06)
				arg = t.translateExprUnconsumed(o)
			} else {
				arg = t.translateExpr(o)
			}
			// A bare `null` carries no type, so CQF wraps it in an As naming the
			// parameter's declared type. Without it the argument is an untyped
			// Null and the call's overload cannot be recovered from the ELM.
			if _, isNull := o.(*ast.NullLiteral); isNull && i < len(declared) && declared[i] != nil {
				arg = t.castNullTo(arg, declared[i])
			} else if i < len(declared) {
				arg = t.castChoiceArgument(arg, declared[i])
			}
			operands = append(operands, arg)
		}
		lib := v.LibraryName
		if lib == "" && t.localFuncCounts[v.Name] == 0 && len(t.libFluentFuncs[v.Name]) > 1 {
			// A fluent name declared by several includes: the receiver's type
			// picks the library, which needs the translated operands, so it
			// cannot be settled in the pre-pass. Left unresolved this emitted a
			// call naming no library at all — 99.7% of issues/08 W4's ~9,800
			// sites. (#13)
			args := make([]typeSpec, len(operands))
			for i, o := range operands {
				args[i] = t.inferTypeSpec(o)
			}
			lib = t.fluentLibraryByArgs(v.Name, args)
		}
		t.noteOverloadedCall(lib, v.Name)
		return &elm.FunctionRefNode{
			Annotation:  ann,
			Signature:   t.computeFunctionRefSig(lib, v.Name, operands),
			Name:        v.Name,
			LibraryName: lib,
			Operand:     operands,
		}
	case *ast.PropertyExpr:
		// When the source is a library alias and the path matches a known symbol in
		// that library, emit the correct typed Ref node (ValueSetRef, CodeRef, etc.)
		// rather than a PropertyNode. This handles e.g. Terminology."Office Visit".
		if ir, ok := v.Source.(*ast.IdentifierRef); ok {
			if libMap, hasLib := t.libSyms[ir.Name]; hasLib {
				switch libMap[v.Path] {
				case symValueSet:
					return t.newValueSetRef(ann, v.Path, ir.Name)
				case symCodeSystem:
					return &elm.CodeSystemRefNode{Annotation: ann, LibraryName: ir.Name, Name: v.Path}
				case symCode:
					return &elm.CodeRefNode{Annotation: ann, LibraryName: ir.Name, Name: v.Path}
				case symConcept:
					return &elm.ConceptRefNode{Annotation: ann, LibraryName: ir.Name, Name: v.Path}
				case symParameter:
					return &elm.ParameterRefNode{Annotation: ann, LibraryName: ir.Name, Name: v.Path}
				case symExpression:
					return &elm.ExpressionRefNode{Annotation: ann, LibraryName: ir.Name, Name: v.Path}
				}
			}
		}
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
			// The source of a navigation is the FHIR structure being traversed,
			// never a converted CQL value, so it is translated unconsumed.
			src := t.translateExprUnconsumed(v.Source)
			if traversal := t.listTraversal(src, v.Path); traversal != nil {
				return traversal
			}
			propertyNode = &elm.PropertyNode{
				Annotation: ann,
				Path:       v.Path,
				Source:     src,
			}
		}
		// Apply implicit FHIRHelpers coercion when FHIRHelpers is included and the
		// property resolves to a FHIR primitive type. CQF places the source locator
		// on the inner PropertyNode, not the synthetic FunctionRef wrapper.
		// A FHIR property carries its declared FHIR type, not the CQL type it
		// converts to — the conversion node above it is what produces the CQL
		// value, and CQF leaves that synthesized node unstamped.
		if t.opts.EnableResultTypes {
			if fhirType := t.resolveFHIRPropertyType(v.Source, v.Path); fhirType != "" {
				fhirTS := typeSpec(namedTS{"{http://hl7.org/fhir}" + fhirType})
				if sourceType := t.resolveFHIRSourceType(v.Source); sourceType != "" &&
					typesystem.IsFHIRListProperty(sourceType, v.Path) {
					fhirTS = listTS{fhirTS}
				}
				name, spec := resultTypeOf(fhirTS)
				elm.SetResultType(propertyNode, name, spec)
			}
		}
		if t.fhirHelpersLocalName != "" && !noCoerce {
			if fhirFunc := t.resolveFHIRPropertyCoercion(v.Source, v.Path); fhirFunc != "" {
				if t.opts.EnableLocators {
					setLocator(propertyNode, locatorStr(v.Loc()))
					t.skipLocatorStamp = true
				}
				// The conversion is synthesized, so it carries no result type.
				t.skipResultTypeStamp = true
				// The ModelInfo records the declared type, so a bound property
				// already reports its binding name here (Patient.gender is
				// AdministrativeGender) — which is what CQF puts in the signature.
				wrapperSig := sig
				if fhirType := t.resolveFHIRPropertyType(v.Source, v.Path); fhirType != "" {
					argSpec := namedTS{"{http://hl7.org/fhir}" + fhirType}
					wrapperSig = t.buildSigFromTypes("FunctionRef", []typeSpec{argSpec})
				}
				t.noteOverloadedCall(t.fhirHelpersLocalName, fhirFunc)
				return t.markConverted(&elm.FunctionRefNode{
					Annotation:  ann,
					Signature:   wrapperSig,
					Name:        fhirFunc,
					LibraryName: t.fhirHelpersLocalName,
					Operand:     []elm.Expression{propertyNode},
				})
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
		return t.castNullOperand(v, t.translateBinaryExpr(v))
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
		// CQF types a bare-null branch with the type of the other branch. That
		// type is read from the translated node, so calls — `else Now()` — and
		// structured types count; the syntactic probe used before only
		// recognised literals. (issues/06 4.1)
		if _, elseIsNull := v.ElseExpr.(*ast.NullLiteral); elseIsNull {
			if ts := knownTS(t.inferTypeSpec(thenExpr)); ts != nil {
				elseExpr = t.castNullTo(elseExpr, ts)
			}
		}
		if _, thenIsNull := v.ThenExpr.(*ast.NullLiteral); thenIsNull {
			if ts := knownTS(t.inferTypeSpec(elseExpr)); ts != nil {
				thenExpr = t.castNullTo(thenExpr, ts)
			}
		}
		return &elm.IfNode{
			Annotation: ann,
			Condition:  condition,
			Then:       thenExpr,
			Else:       elseExpr,
		}
	case *ast.CaseExpr:
		// A null branch takes the type the other branches establish, wrapped in
		// As. That type is read from the translated branches rather than their
		// syntax: the syntactic probe used before only recognised literals, so
		// an Interval<DateTime> branch — or a call — left the null bare.
		// (issues/06 4.1)
		thens := make([]elm.Expression, len(v.Items))
		for i, item := range v.Items {
			thens[i] = t.translateExpr(item.Then)
		}
		elseExpr := t.translateExpr(v.Else)
		branchType := t.caseBranchType(v, thens, elseExpr)
		if unified := t.unifyCaseBranches(thens, elseExpr); unified != nil {
			// Branches of different types: CQF types the case as a Choice of them
			// and casts each branch to it. (issues/08 #15, FHIRHelpers.ToValue)
			branchType = unified
		}
		if _, isNull := v.Else.(*ast.NullLiteral); isNull && branchType != nil {
			elseExpr = t.castNullTo(elseExpr, branchType)
		} else {
			elseExpr = t.castCaseBranch(elseExpr, branchType)
		}
		cn := &elm.CaseNode{Annotation: ann, Else: elseExpr}
		if v.Comparand != nil {
			cn.Comparand = t.translateExpr(v.Comparand)
		}
		for i, item := range v.Items {
			thenExpr := thens[i]
			if _, isNull := item.Then.(*ast.NullLiteral); isNull && branchType != nil {
				thenExpr = t.castNullTo(thenExpr, branchType)
			} else {
				thenExpr = t.castCaseBranch(thenExpr, branchType)
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
		if t.opts.EnableResultTypes {
			stampTypeSpecifier(ts)
		}
		asNode := &elm.AsNode{
			Annotation:      ann,
			Signature:       sig,
			Operand:         t.translateExpr(v.Operand),
			AsTypeSpecifier: ts,
			Strict:          &v.Strict,
		}
		// `x as FHIR.dateTime` yields a FHIR value, and a consumed FHIR value is
		// converted through FHIRHelpers exactly as a property is:
		// `(O.effective as FHIR.dateTime) during P` compares ToDateTime(...).
		// Unconsumed positions — a define body, an explicit FHIRHelpers argument
		// — keep the bare cast. (issues/06)
		if !noCoerce {
			if wrapped := t.convertFHIRCast(asNode, ts, locatorStr(v.Loc())); wrapped != nil {
				return wrapped
			}
		}
		return asNode
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
		// ELM R1 has no Between operator: `X between A and B` is
		// X >= A and X <= B (strict comparisons for `properly between`). The
		// operand is translated once per comparison, as CQF emits two nodes.
		lowOp, highOp := "GreaterOrEqual", "LessOrEqual"
		if v.Properly {
			lowOp, highOp = "Greater", "Less"
		}
		cmp := func(op string, bound ast.Expr) elm.Expression {
			operands := []elm.Expression{t.translateExpr(v.Operand), t.translateExpr(bound)}
			n := &elm.OperatorExpressionNode{
				Annotation: t.cqfAnnotation(),
				Signature:  t.computeSig(op, operands),
				Operator:   op,
				Operand:    operands,
			}
			return t.applyImplicitConversions(n)
		}
		operands := []elm.Expression{cmp(lowOp, v.Low), cmp(highOp, v.High)}
		return &elm.OperatorExpressionNode{
			Annotation: ann,
			Signature:  t.computeSig("And", operands),
			Operator:   "And",
			Operand:    operands,
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
		// `Interval[E.period.start, E.period.end]` builds an interval of the
		// FHIR type itself: CQF does not convert selector bounds, and records
		// Interval<FHIR.dateTime>. Only when both bounds are properties — a
		// mixed interval needs its bounds unified, which is a conversion.
		// (issues/06)
		translateBound := t.translateExpr
		if _, lowProp := v.Low.(*ast.PropertyExpr); lowProp {
			if _, highProp := v.High.(*ast.PropertyExpr); highProp {
				translateBound = t.translateExprUnconsumed
			}
		}
		low := translateBound(v.Low)
		high := translateBound(v.High)
		// Null interval bounds: CQF wraps null in an As cast typed from the non-null bound.
		// The As carries the narrowing; the null inside it keeps its own type,
		// Any. Stamping the bound type on the Null as well is the mismatch
		// issues/06 4.1 reported.
		if _, isNull := low.(*elm.NullNode); isNull {
			if bndType := t.boundTypeName(v.High, high); bndType != "" {
				if t.opts.EnableResultTypes {
					elm.SetResultType(low, typesystem.TypeAny, nil)
				}
				low = &elm.AsNode{
					Annotation: t.cqfAnnotation(),
					Signature:  t.cqfEmptyArrayField(),
					AsType:     bndType,
					Operand:    low,
				}
			}
		}
		if _, isNull := high.(*elm.NullNode); isNull {
			if bndType := t.boundTypeName(v.Low, low); bndType != "" {
				if t.opts.EnableResultTypes {
					elm.SetResultType(high, typesystem.TypeAny, nil)
				}
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
		// `date from`, `time from` and `timezoneoffset from` are distinct ELM
		// unary operators, not DateTimeComponentFrom precisions. Only the
		// calendar/clock components (Year..Millisecond) use DateTimeComponentFrom.
		src := t.translateExpr(v.Source)
		if op, ok := dateTimeComponentOperators[v.Precision]; ok {
			// These three operators are DateTime-only, so a Date operand is
			// implicitly converted. DateTimeComponentFrom has a Date overload
			// and takes its operand as-is.
			if n, isNamed := t.inferTypeSpec(src).(namedTS); isNamed && n.name == typesystem.TypeDate {
				src = &elm.UnaryExpressionNode{
					Annotation: t.cqfAnnotation(),
					Signature:  t.computeSig("ToDateTime", []elm.Expression{src}),
					Operator:   "ToDateTime",
					Operand:    src,
				}
			}
			return &elm.UnaryExpressionNode{
				Annotation: ann,
				Signature:  t.computeSig(op, []elm.Expression{src}),
				Operator:   op,
				Operand:    src,
			}
		}
		return &elm.UnaryPrecisionOperatorNode{
			Annotation: ann,
			Signature:  t.computeSig("DateTimeComponentFrom", []elm.Expression{src}),
			Operator:   "DateTimeComponentFrom",
			Precision:  v.Precision,
			Operand:    src,
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
			elem := t.translateExprUnconsumed(e)
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
				switch {
				case nt.Qualifier == "System" || systemClassTypes[nt.Name]:
					in.ClassType = typesystem.SystemELMName + nt.Name
				case nt.Qualifier != "":
					in.ClassType = t.qualifyTypeName(nt.Qualifier + "." + nt.Name)
				default:
					in.ClassType = t.qualifyTypeName(nt.Name)
				}
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
			// A bare number is a unitless quantity: `per 2` is 2 '1'.
			per := v.PerClause
			switch lit := per.(type) {
			case *ast.IntegerLiteral:
				per = &ast.QuantityLiteral{Value: lit.Text, Unit: "1"}
			case *ast.DecimalLiteral:
				per = &ast.QuantityLiteral{Value: lit.Value, Unit: "1"}
			}
			if _, synthesized := v.PerClause.(*ast.QuantityLiteral); per != v.PerClause || (synthesized && v.PerClause.Loc().Start.Line == 0) {
				// The quantity was made from a bare number or a precision
				// keyword: CQF builds it without a locator or result type.
				operands = append(operands, t.translateExprCore(per))
			} else {
				operands = append(operands, t.translateExpr(per))
			}
		} else if v.Op == "Collapse" || v.Op == "Expand" {
			// CQF always emits null as the second (per) argument when absent, and
			// types it as the Quantity that slot declares.
			perNull := &elm.NullNode{Annotation: t.cqfAnnotation()}
			if t.opts.EnableResultTypes {
				perNull.ResultTypeName = typesystem.TypeQuantity
			}
			operands = append(operands, perNull)
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

// newValueSetRef builds a ValueSetRef expression node. CQF stamps preserve=true
// on every ValueSetRef under CQL 1.5 (runtime terminology evaluation); the flag
// did not exist before 1.5, so it is suppressed at compatibility level 1.4.
func (t *Translator) newValueSetRef(ann json.RawMessage, name, libraryName string) *elm.ValueSetRefNode {
	var preserve *bool
	if compat := t.opts.CompatibilityLevel; compat == "" || compat >= "1.5" {
		b := true
		preserve = &b
	}
	return &elm.ValueSetRefNode{
		Annotation:  ann,
		Name:        name,
		LibraryName: libraryName,
		Preserve:    preserve,
	}
}

// valueSetRefName resolves expr to a declared value set, returning its name and
// the local identifier of the library declaring it ("" when declared locally).
// It accepts the unqualified form ("Diabetes VS") as well as both AST shapes a
// library-qualified reference can take (Terminology."Diabetes VS").
func (t *Translator) valueSetRefName(expr ast.Expr) (name, libraryName string, ok bool) {
	switch e := expr.(type) {
	case *ast.IdentifierRef:
		if t.syms[e.Name] == symValueSet {
			return e.Name, "", true
		}
	case *ast.QualifiedRef:
		if t.libSyms[e.LibraryName][e.Name] == symValueSet {
			return e.Name, e.LibraryName, true
		}
	case *ast.PropertyExpr:
		if src, isID := e.Source.(*ast.IdentifierRef); isID {
			if t.libSyms[src.Name][e.Path] == symValueSet {
				return e.Path, src.Name, true
			}
		}
	}
	return "", "", false
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
		if vsName, libName, ok := t.valueSetRefName(v.Right); ok {
			compat := t.opts.CompatibilityLevel
			emitPreserve := compat == "" || compat >= "1.5"
			var preservePtr *bool
			if emitPreserve {
				b := true
				preservePtr = &b
			}
			vs := &elm.ValueSetRefNode{
				Annotation:  t.cqfAnnotation(),
				Name:        vsName,
				LibraryName: libName,
				Preserve:    preservePtr,
			}
			if t.opts.EnableResultTypes {
				vs.ResultTypeName = typesystem.TypeValueSet
			}
			if t.opts.EnableLocators {
				vs.Locator = locatorStr(v.Right.Loc())
			}
			// A list of codes — List<CodeableConcept> such as Condition.category,
			// or List<Coding> — is tested with AnyInValueSet, true when any
			// element is in the value set. InValueSet over the list tests the
			// list itself. A FHIR list is converted element-wise first, the
			// same lift CQF uses everywhere a list of FHIR values is consumed.
			// (Found rechecking an early gap: this applies with list traversal
			// disabled too, i.e. under the measure profile.)
			if raw := t.translateExprUnconsumed(v.Left); isListTS(t.inferTypeSpec(raw)) {
				codes := raw
				if lifted, ok := t.convertFHIRPrimitiveList(v.Left, raw); ok {
					codes = lifted
				}
				anySig := t.cqfEmptyArrayField()
				if lvl := t.opts.SignatureLevel; lvl != "" && lvl != "None" {
					anySig = t.buildInferredSig([]elm.Expression{codes})
				}
				return &elm.AnyInValueSetNode{
					Annotation: t.cqfAnnotation(),
					Signature:  anySig,
					Codes:      codes,
					ValueSet:   vs,
				}
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

	var lhs, rhs elm.Expression
	if t.sharesFHIRType(v.Op, v.Left, v.Right) {
		// Equality between two compatible FHIR values resolves to the generic
		// Equal<T>(T, T) with no conversion: `T.sequence = I.sequence` compares
		// positiveInt with positiveInt, and `Id.value = E.id` compares string with
		// id, which derives from string. Converting both sides to System values
		// is a different (though usually equal-valued) expression. (issues/08 W9)
		lhs = t.translateExprUnconsumed(v.Left)
		rhs = t.translateExprUnconsumed(v.Right)
	} else {
		lhs = t.translateExpr(v.Left)
		rhs = t.translateExpr(v.Right)
	}

	// Membership in a value set given as an expression — an operand or
	// parameter typed System.ValueSet — is InValueSet with a valuesetExpression,
	// not list In. The value set is not a list; In over it asked a different
	// question. (issues/08 W5)
	if v.Op == "In" && namedName(t.inferTypeSpec(rhs)) == typesystem.TypeValueSet {
		sig := t.cqfEmptyArrayField()
		if lvl := t.opts.SignatureLevel; lvl != "" && lvl != "None" {
			sig = t.buildInferredSig([]elm.Expression{lhs, rhs})
		}
		return &elm.InValueSetNode{
			Annotation:         t.cqfAnnotation(),
			Signature:          sig,
			Code:               t.convertFHIRCode(lhs),
			ValuesetExpression: rhs,
		}
	}

	op := t.demotePointInclusionOp(v.Op, v.Left, lhs)
	t.noteInclusionDemotion(op, v.Left, v.Right, lhs, rhs)

	// For the "In" operator, apply either interval promotion or list promotion when
	// the RHS cannot already satisfy an In overload for the LHS type. The check uses
	// the inferred types rather than the syntactic form, so a reference to a list- or
	// interval-valued definition is not wrongly promoted.
	// A list-valued FHIR primitive on the right of `in` is converted element by
	// element, not promoted: the list already satisfies In(T, List<T>) once its
	// elements are converted.
	if v.Op == "In" {
		if converted, ok := t.convertFHIRPrimitiveList(v.Right, rhs); ok {
			rhs = converted
			operands := []elm.Expression{lhs, rhs}
			return &elm.OperatorExpressionNode{
				Annotation: t.cqfAnnotation(),
				Signature:  t.computeInSig(lhs, rhs, false),
				Operator:   "In",
				Operand:    operands,
			}
		}
	}

	var listPromoted bool
	if v.Op == "In" && !t.opts.DisableListPromotion && !rhsSatisfiesIn(t.inferTypeSpec(lhs), t.inferTypeSpec(rhs)) {
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
					rhs = t.markConverted(&elm.UnaryExpressionNode{
						Annotation: t.cqfAnnotation(),
						Signature:  t.cqfEmptyArrayField(),
						Operator:   "ToList",
						Operand:    rhs,
					})
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
	// Decided from operand types, not syntax: `"Label" + 'x'` concatenates a
	// String-valued define. (issues/07)
	if op == "Add" && ((isStringLikeExpr(v.Left) && isStringLikeExpr(v.Right)) ||
		(namedName(t.typeOf(lhs)) == typesystem.TypeString && namedName(t.typeOf(rhs)) == typesystem.TypeString)) {
		op = "Concatenate"
	}
	// `&` is null-tolerant concatenation: a null operand counts as the empty
	// string, where `+` propagates it. CQF expresses that by wrapping each
	// operand in Coalesce(x, ''); without it `'a' & null` is null instead of
	// 'a'. (issues/07)
	if v.Op == "Concatenate" {
		lhs, rhs = t.coalesceEmpty(lhs), t.coalesceEmpty(rhs)
	}

	// Numeric promotion (Integer → Long → Decimal → Quantity) is applied by
	// applyImplicitConversions once the node is built, from operand *types*.

	lhs, rhs = t.promoteDateToDateTime(op, lhs, rhs)
	if t.opts.EnableResultTypes {
		t.typeNullFromSibling(lhs, rhs)
		t.typeNullFromSibling(rhs, lhs)
	}

	operands := []elm.Expression{lhs, rhs}
	// For Equivalent (~): when one side is a CodeableConcept (coerces to Concept via ToConcept)
	// and the other is a CodeRef or CodeSystemRef, promote the code to Concept via ToConcept.
	if op == "Equivalent" && t.fhirHelpersLocalName != "" {
		lhsConcept := t.isFHIRConceptExpr(v.Left)
		rhsConcept := t.isFHIRConceptExpr(v.Right)
		if lhsConcept && !rhsConcept {
			rhs = t.wrapToConcept(rhs)
			operands = []elm.Expression{lhs, rhs}
		} else if rhsConcept && !lhsConcept {
			lhs = t.wrapToConcept(lhs)
			operands = []elm.Expression{lhs, rhs}
		}
	}
	if v.Precision != "" {
		return &elm.PrecisionOperatorNode{
			Annotation: t.cqfAnnotation(),
			Signature:  t.computeSig(op, operands),
			Operator:   op,
			Precision:  v.Precision,
			Operand:    operands,
		}
	}
	// CQL `!=` and `!~` have no dedicated ELM operator: CQF lowers them to
	// Not(Equal(...)) and Not(Equivalent(...)). Both nodes carry the source span
	// of the whole comparison.
	if inner, negated := negatedEqualityOps[op]; negated {
		node := &elm.OperatorExpressionNode{
			Annotation: t.cqfAnnotation(),
			Signature:  t.computeSig(inner, operands),
			Operator:   inner,
			Operand:    operands,
		}
		t.applyImplicitConversions(node)
		if t.opts.EnableLocators {
			node.Locator = locatorStr(v.Loc())
		}
		// CQF records a result type on the inner comparison as well as the Not.
		if t.opts.EnableResultTypes {
			t.stampResultType(node)
		}
		return &elm.UnaryExpressionNode{
			Annotation: t.cqfAnnotation(),
			Signature:  t.computeSig("Not", []elm.Expression{node}),
			Operator:   "Not",
			Operand:    node,
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

func (t *Translator) translateTypeIs(v *ast.TypeIsExpr) elm.Expression {
	ann := t.cqfAnnotation()
	// `X is null` takes Any, so there is no System type to convert X to: a FHIR
	// value is tested as it is. Converting it would test whether the
	// *conversion* is null. (issues/07)
	var operand elm.Expression
	if v.IsNull {
		operand = t.translateExprUnconsumed(v.Operand)
	} else {
		operand = t.translateExpr(v.Operand)
	}
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
			if t.opts.EnableResultTypes {
				t.stampResultType(isNull)
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
			if t.opts.EnableResultTypes {
				t.stampResultType(node)
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
			if t.opts.EnableResultTypes {
				t.stampResultType(node)
			}
			return &elm.UnaryExpressionNode{Annotation: ann, Signature: t.computeSig("Not", []elm.Expression{node}), Operator: "Not", Operand: node}
		}
		return node
	}
	ts := t.translateTypeSpecifier(v.TypeSpec)
	if t.opts.EnableResultTypes {
		stampTypeSpecifier(ts)
	}
	return &elm.IsNode{
		Annotation:      ann,
		Signature:       t.cqfEmptyArrayField(),
		Operand:         operand,
		IsTypeSpecifier: ts,
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
		// Infer default codeProperty from FHIR ModelInfo primaryCodePath when not explicit.
		if r.CodeProperty == "" {
			// dt format: "{http://hl7.org/fhir}Encounter" — extract the local name.
			localType := dt
			if i := strings.LastIndex(localType, "}"); i >= 0 {
				localType = localType[i+1:]
			}
			if cp, ok := typesystem.FHIRPrimaryCodePath[localType]; ok {
				r.CodeProperty = cp
			}
		}
		// For ValueSetRef codes: set preserve=true and default codeComparator="in".
		if vsRef, ok := r.Codes.(*elm.ValueSetRefNode); ok {
			tr := true
			vsRef.Preserve = &tr
			if v.CodeComparator != "" {
				r.CodeComparator = v.CodeComparator
			} else {
				r.CodeComparator = "in"
			}
		} else {
			// Non-valueset codes: default codeComparator="~".
			if v.CodeComparator != "" {
				r.CodeComparator = v.CodeComparator
			} else {
				r.CodeComparator = "~"
			}
		}

		// When FHIRHelpers is included and the code property is a reference-choice
		// type (e.g. MedicationRequest.medication which can be CodeableConcept or
		// Reference(Medication)), expand to a Union of the direct retrieve and a
		// reference-resolution sub-query — matching CQF's profile-informed behavior.
		if t.fhirHelpersLocalName != "" {
			localType := dt
			if i := strings.LastIndex(localType, "}"); i >= 0 {
				localType = localType[i+1:]
			}
			if refTarget, ok := typesystem.FHIRReferenceChoiceCodeProperty[localType]; ok {
				// Only expand when the codes expression is a ValueSetRef (code-in-valueset retrieves).
				if vsRef2, ok2 := r.Codes.(*elm.ValueSetRefNode); ok2 {
					// Pass the original retrieve's locator for CQF locator parity.
					loc := ""
					if t.opts.EnableLocators {
						loc = locatorStr(v.Loc())
					}
					refQuery := t.buildMedicationReferenceSubQuery(localType, refTarget, vsRef2, loc)
					ann := t.cqfAnnotation()
					return &elm.OperatorExpressionNode{
						Annotation: ann,
						Signature:  t.cqfEmptyArrayField(),
						Operator:   "Union",
						Operand:    []elm.Expression{r, refQuery},
					}
				}
			}
		}
	}
	return r
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

// caseBranchType returns the type a case expression's non-null branches
// establish — the first one that is known — or nil when none is.
func (t *Translator) caseBranchType(v *ast.CaseExpr, thens []elm.Expression, elseExpr elm.Expression) typeSpec {
	for i, item := range v.Items {
		if _, isNull := item.Then.(*ast.NullLiteral); isNull {
			continue
		}
		if ts := knownTS(t.inferTypeSpec(thens[i])); ts != nil {
			return ts
		}
	}
	if _, isNull := v.Else.(*ast.NullLiteral); !isNull {
		return knownTS(t.inferTypeSpec(elseExpr))
	}
	return nil
}

// boundTypeName is the named type a null interval bound is cast to: the type of
// the other bound. The translated bound is asked first, since the syntactic
// probe only recognises literals and so missed a function call such as
// FHIRHelpers' Interval[null, ToQuantityIgnoringComparator(quantity)).
// (issues/08 #15)
func (t *Translator) boundTypeName(other ast.Expr, translated elm.Expression) string {
	if n, ok := knownTS(t.inferTypeSpec(translated)).(namedTS); ok {
		return n.name
	}
	return t.inferBoundType(other)
}

// unifyCaseBranches returns the Choice CQF types a case as when its non-null
// branches have incompatible types, or nil when they unify to a single type (or
// too little is known to say).
//
// It follows CQF's accumulation: the result starts as the first branch's type,
// and each later branch either fits it (equal, a FHIR subtype, or implicitly
// convertible to it), widens it (the result converts to the branch), or — when
// neither — turns it into a Choice. Once the result is a Choice, a new type that
// is not already an alternative simply joins it, which is why FHIRHelpers'
// ToValue carries both Integer and Decimal. Alternatives sort as CQF orders them.
func (t *Translator) unifyCaseBranches(thens []elm.Expression, elseExpr elm.Expression) typeSpec {
	var result typeSpec
	for _, e := range append(append([]elm.Expression{}, thens...), elseExpr) {
		if e == nil || isNullLike(e) || isDefinitelyAnyBranch(e) {
			continue
		}
		ts := knownTS(t.inferTypeSpec(e))
		if ts == nil {
			continue
		}
		result = accumulateCaseType(result, ts)
	}
	c, ok := result.(choiceTS)
	if !ok {
		return nil
	}
	sortChoiceTS(c.alts)
	return c
}

// accumulateCaseType folds one more branch type into a case's result type.
func accumulateCaseType(result, ts typeSpec) typeSpec {
	if result == nil {
		return ts
	}
	if c, ok := result.(choiceTS); ok {
		for _, a := range c.alts {
			if typeSpecEqual(a, ts) || fhirSubtypeTS(ts, a) {
				return c
			}
		}
		if tc, ok := ts.(choiceTS); ok {
			for _, a := range tc.alts {
				c = accumulateCaseType(c, a).(choiceTS)
			}
			return c
		}
		return choiceTS{alts: append(append([]typeSpec{}, c.alts...), ts)}
	}
	switch {
	case typeSpecEqual(result, ts), fhirSubtypeTS(ts, result), implicitlyConvertible(ts, result):
		return result
	case fhirSubtypeTS(result, ts), implicitlyConvertible(result, ts):
		return ts
	}
	if tc, ok := ts.(choiceTS); ok {
		return accumulateCaseType(choiceTS{alts: []typeSpec{result}}, tc)
	}
	return choiceTS{alts: []typeSpec{result, ts}}
}

// fhirSubtypeTS reports a FHIR named type deriving from another.
func fhirSubtypeTS(sub, super typeSpec) bool {
	sl, pl := fhirLocalTypeName(sub), fhirLocalTypeName(super)
	return sl != "" && pl != "" && sl != pl && fhirDerivesFrom(sl, pl)
}

// castCaseBranch casts a non-null case branch to the case's result type when it
// is not already of that type: a branch of one alternative of a Choice result,
// or a branch that is definitely Any, such as Message over a null source. A
// branch whose type is merely unknown to echo-elm is left alone — casting it
// would add an As that CQF, which does know the type, does not emit.
func (t *Translator) castCaseBranch(e elm.Expression, result typeSpec) elm.Expression {
	if e == nil || result == nil {
		return e
	}
	if rc, isChoice := result.(choiceTS); isChoice {
		ts := t.inferTypeSpec(e)
		if ts == nil || typeSpecEqual(ts, result) || choiceWithin(ts, rc) {
			return e
		}
		return t.castNullTo(e, result)
	}
	if isDefinitelyAnyBranch(e) {
		return t.castNullTo(e, result)
	}
	return e
}

// choiceWithin reports a choice type all of whose alternatives are alternatives
// of within: a subtype of it, which CQF does not cast. A single type is not a
// choice and is cast, even when it is one of the alternatives.
func choiceWithin(ts typeSpec, within choiceTS) bool {
	c, ok := ts.(choiceTS)
	if !ok {
		return false
	}
	for _, a := range c.alts {
		found := false
		for _, w := range within.alts {
			if typeSpecEqual(a, w) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// isDefinitelyAnyBranch is a non-null expression whose type is Any by
// construction: Message returns the type of its source, and a null source makes
// that Any.
func isDefinitelyAnyBranch(e elm.Expression) bool {
	n, ok := e.(*elm.NamedOperatorExpressionNode)
	return ok && n.Operator == "Message" && isNullLike(namedOperand(n, "source"))
}

// castNullOperand wraps a bare null operand of a same-type binary operator in
// the As CQF emits for it, typed from the other operand. It looks through the
// Not that `!=` and `!~` lower to. Anything else is returned unchanged.
// (issues/06 4.1)
func (t *Translator) castNullOperand(v *ast.BinaryExpr, e elm.Expression) elm.Expression {
	target := e
	if not, ok := e.(*elm.UnaryExpressionNode); ok && not.Operator == "Not" {
		target = not.Operand
	}
	node, ok := target.(*elm.OperatorExpressionNode)
	if !ok || !sameTypeBinaryOps[node.Operator] || len(node.Operand) != 2 {
		return e
	}
	_, leftNull := v.Left.(*ast.NullLiteral)
	_, rightNull := v.Right.(*ast.NullLiteral)
	switch {
	case leftNull && !rightNull:
		if ts := knownTS(t.inferTypeSpec(node.Operand[1])); ts != nil {
			node.Operand[0] = t.castNullTo(node.Operand[0], ts)
		}
	case rightNull && !leftNull:
		if ts := knownTS(t.inferTypeSpec(node.Operand[0])); ts != nil {
			node.Operand[1] = t.castNullTo(node.Operand[1], ts)
		}
	default:
		return e
	}
	node.Signature = t.computeSig(node.Operator, node.Operand)
	return e
}

// coalesceEmpty is the Coalesce(x, ”) CQF wraps each operand of `&` in. Both
// the Coalesce and its ” are synthesized: no locator, no result type.
func (t *Translator) coalesceEmpty(e elm.Expression) elm.Expression {
	empty := &elm.LiteralNode{Annotation: t.cqfAnnotation(), ValueType: typesystem.TypeString, Value: ""}
	operands := []elm.Expression{e, empty}
	return &elm.OperatorExpressionNode{
		Annotation: t.cqfAnnotation(),
		Signature:  t.computeSig("Coalesce", operands),
		Operator:   "Coalesce",
		Operand:    operands,
	}
}
