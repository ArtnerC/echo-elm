package translator

// Included libraries, library-qualified calls, statement emission order and the implicit context accessor.

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"sync"

	"github.com/artnerc/echo-elm/internal/ast"
	"github.com/artnerc/echo-elm/internal/elm"
	"github.com/artnerc/echo-elm/internal/parser"
)

// buildSymMap builds a name→symKind map from an ast.Library, used for cross-library
// QualifiedRef resolution.
func buildSymMap(lib *ast.Library) map[string]symKind {
	m := make(map[string]symKind)
	for _, vs := range lib.Valuesets {
		m[vs.Name] = symValueSet
	}
	for _, cs := range lib.Codesystems {
		m[cs.Name] = symCodeSystem
	}
	for _, c := range lib.Codes {
		m[c.Name] = symCode
	}
	for _, con := range lib.Concepts {
		m[con.Name] = symConcept
	}
	for _, p := range lib.Parameters {
		m[p.Name] = symParameter
	}
	for _, s := range lib.Statements {
		if _, already := m[s.Name]; !already {
			m[s.Name] = symExpression
		}
	}
	return m
}

// IncludeCache holds what a translation learns from each included library —
// its symbols, fluent functions, types and function counts — so that a run over
// many libraries loads each shared helper once. Every measure library includes
// FHIRHelpers; without the cache a 300-library run parses and types it 300
// times, which was about half the time of an in-process run.
//
// Only derived, read-only data is cached, never a parse tree: translation
// rewrites the AST it is given, so a shared tree would be both wrong and a data
// race. Entries are keyed by the include's path, version, a hash of its source
// and the translation options that shape its types, so the same name resolved
// from two directories, or translated two ways, cannot collide.
//
// It is safe for concurrent use. Two workers that miss on the same entry at
// once both compute it and the first to finish is kept — a little duplicate
// work at the start of a run, in exchange for no waiting, so an include cycle
// (invalid CQL, but possible input) can never deadlock one worker on another.
type IncludeCache struct {
	mu      sync.Mutex
	entries map[string]*includedLibrary
}

// NewIncludeCache returns an empty cache.
func NewIncludeCache() *IncludeCache {
	return &IncludeCache{entries: map[string]*includedLibrary{}}
}

func (c *IncludeCache) get(key string) *includedLibrary {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.entries[key]
}

func (c *IncludeCache) put(key string, il *includedLibrary) *includedLibrary {
	c.mu.Lock()
	defer c.mu.Unlock()
	if prev, ok := c.entries[key]; ok {
		return prev
	}
	c.entries[key] = il
	return il
}

// includedLibrary is what an includer needs from one included library.
type includedLibrary struct {
	syms       map[string]symKind
	fluent     []string
	types      *includeTypes
	funcCounts map[string]int
}

// includeTypes are the types an included library exposes to its includer.
type includeTypes struct {
	defs     map[string]typeSpec
	funcs    map[string]typeSpec
	operands map[string][]typeSpec
}

// loadInclude loads an included library through the cache, or returns nil when
// its source cannot be found or parsed.
//
// Types come from translating the include, which recurses into the include's
// own includes: it used to stop at one level, so a definition two includes deep
// came out List<Any> (issues/07). A library already being typed further up this
// chain yields no types rather than looping. Anything typed with such a gap
// below it is right only for this chain, so it is not cached.
func (t *Translator) loadInclude(path, version string) *includedLibrary {
	src, found, err := t.opts.LibrarySource.GetLibrarySource(path, version)
	if err != nil || !found {
		return nil
	}
	sum := sha256.Sum256(src)
	key := path + "|" + version + "|" + hex.EncodeToString(sum[:]) + "|" + optionsKey(&t.opts)
	if il := t.opts.IncludeCache.get(key); il != nil {
		return il
	}
	pr, err := parser.ParseBytes(src, path+".cql")
	if err != nil {
		return nil
	}
	il := &includedLibrary{
		syms:       buildSymMap(pr.Library),
		funcCounts: functionCounts(pr.Library),
	}
	for _, st := range pr.Library.Statements {
		if st.IsFunction && st.IsFluent && !containsString(il.fluent, st.Name) {
			il.fluent = append(il.fluent, st.Name)
		}
	}

	if t.includeVisiting == nil {
		t.includeVisiting = map[string]bool{}
	}
	chainKey := path + "|" + version
	if t.includeVisiting[chainKey] {
		t.includeTruncated = true
		return il // a cycle: no types, and not cached
	}
	t.includeVisiting[chainKey] = true
	sub := New(t.opts)
	sub.includeVisiting = t.includeVisiting
	sub.Translate(pr.Library, path+".cql")
	delete(t.includeVisiting, chainKey)
	il.types = &includeTypes{defs: sub.defTypeSpecs, funcs: sub.funcReturnSpecs, operands: sub.localFuncOperands}
	if sub.includeTruncated {
		t.includeTruncated = true
		return il
	}
	return t.opts.IncludeCache.put(key, il)
}

// optionsKey renders the options that shape a translation's types, for the
// include cache key. The library source and the cache itself are excluded:
// they identify where libraries come from, not how they translate.
func optionsKey(opts *Options) string {
	o := *opts
	o.LibrarySource = nil
	o.IncludeCache = nil
	o.TranslatorVersion = ""
	return fmt.Sprintf("%+v", o)
}

// fluentLibraryFor returns the include alias declaring a fluent function called
// name, when exactly one does.
//
// A local definition wins: a fluent function defined in this library needs no
// qualifier, and stamping one would point the reference somewhere else. So does
// ambiguity — if two includes declare the same fluent name, choosing between
// them needs the receiver's element type, which is overload resolution rather
// than qualification. Guessing there would produce a confidently wrong
// libraryName, which is worse than the missing one; issues/05 3.6 tracks it.
func (t *Translator) fluentLibraryFor(name string) (string, bool) {
	if t.localFuncCounts[name] > 0 {
		return "", false
	}
	libs := t.libFluentFuncs[name]
	if len(libs) != 1 {
		return "", false
	}
	return libs[0], true
}

// containsString reports whether xs contains s.
func containsString(xs []string, s string) bool {
	return slices.Contains(xs, s)
}

// positionedAccessor is an implicit context accessor together with the source
// position of the `context` declaration that produced it.
type positionedAccessor struct {
	at  ast.Position
	def *elm.StatementDef
}

// leadingAccessors returns the accessors declared before the first statement,
// or all of them when there are no statements at all.
func leadingAccessors(pending *[]positionedAccessor, ordered []*ast.ExpressionDefinition) []*elm.StatementDef {
	if len(ordered) == 0 {
		return drainAccessors(pending)
	}
	return accessorsBefore(pending, ordered[0].Loc().Start)
}

// accessorsBefore removes and returns every pending accessor declared at or
// before pos, in declaration order.
func accessorsBefore(pending *[]positionedAccessor, pos ast.Position) []*elm.StatementDef {
	var out []*elm.StatementDef
	kept := (*pending)[:0]
	for _, a := range *pending {
		if beforePosition(a.at, pos) {
			out = append(out, a.def)
			continue
		}
		kept = append(kept, a)
	}
	*pending = kept
	return out
}

// drainAccessors removes and returns everything still pending.
func drainAccessors(pending *[]positionedAccessor) []*elm.StatementDef {
	out := make([]*elm.StatementDef, 0, len(*pending))
	for _, a := range *pending {
		out = append(out, a.def)
	}
	*pending = (*pending)[:0]
	return out
}

// beforePosition reports whether a precedes b in the source.
func beforePosition(a, b ast.Position) bool {
	if a.Line != b.Line {
		return a.Line < b.Line
	}
	return a.Column < b.Column
}

// resolveLibraryQualifiedCalls rewrites `Alias.Func(args)` from the shape the
// grammar produces into a library-qualified call.
//
// CQL spells a library-qualified call and a method invocation identically —
// `<term>.<name>(...)` — so the parser cannot tell them apart and builds both as
// FunctionRef{Name: "Func", Operands: [Alias, args...]}. Only the include
// aliases distinguish them, and those are not known until resolution.
//
// This runs once, before anything reads the statement bodies, because two
// separate consumers have to agree about it: emission needs libraryName on the
// node, and statementEmissionOrder needs to NOT see a dependency on a local
// definition of the same name. Recovering the qualifier at emission time alone
// leaves the ordering pass believing `H."Doubled"(21)` depends on a local
// "Doubled", which hoists that definition ahead of its referent and reorders
// statements.def[] away from CQF.
//
// Only the alias must be known, not the function: a library's functions are not
// all reachable through libSyms, and an unknown name under a known alias is
// still library-qualified. This mirrors the PropertyExpr path, which resolves
// `Alias.Symbol` the same way.
func (t *Translator) resolveLibraryQualifiedCalls(stmts []*ast.ExpressionDefinition) {
	for _, s := range stmts {
		if s == nil || s.Expression == nil {
			continue
		}
		ast.Walk(s.Expression, func(n ast.Node) {
			fr, isCall := n.(*ast.FunctionRef)
			if !isCall || fr.LibraryName != "" || len(fr.Operands) == 0 {
				return
			}
			if ir, isIdent := fr.Operands[0].(*ast.IdentifierRef); isIdent {
				if _, isLib := t.libSyms[ir.Name]; isLib || t.includeAliases[ir.Name] {
					fr.LibraryName = ir.Name
					fr.Operands = fr.Operands[1:]
					return
				}
			}
			// Otherwise the receiver is an ordinary expression, which makes this
			// a fluent invocation: `X.Func()` is `Func(X)` where Func is declared
			// `fluent`. The receiver stays as the first operand — only the
			// library qualifier has to be recovered.
			if lib, ok := t.fluentLibraryFor(fr.Name); ok {
				fr.LibraryName = lib
			}
		})
	}
}

// statementEmissionOrder returns the statements in the order CQF emits them.
//
// CQF resolves definitions lazily: translating a body that references another
// define resolves — and therefore emits — that define first. The observable
// result is dependency order, with source order as the tiebreaker, so a define
// that forward-references a later one appears after it in the ELM.
//
// A reference cycle is invalid CQL; if one is present the statements involved
// keep source order rather than looping.
func statementEmissionOrder(stmts []*ast.ExpressionDefinition) []*ast.ExpressionDefinition {
	byName := make(map[string]*ast.ExpressionDefinition, len(stmts))
	for _, s := range stmts {
		if _, dup := byName[s.Name]; !dup {
			byName[s.Name] = s
		}
	}

	out := make([]*ast.ExpressionDefinition, 0, len(stmts))
	emitted := make(map[*ast.ExpressionDefinition]bool, len(stmts))
	visiting := make(map[*ast.ExpressionDefinition]bool, len(stmts))

	var emit func(s *ast.ExpressionDefinition)
	emit = func(s *ast.ExpressionDefinition) {
		if emitted[s] || visiting[s] {
			return
		}
		visiting[s] = true
		for _, dep := range localReferences(s, byName) {
			emit(dep)
		}
		visiting[s] = false
		emitted[s] = true
		out = append(out, s)
	}
	for _, s := range stmts {
		emit(s)
	}
	return out
}

// localReferences returns the statements in byName that s references, in the
// order they appear in s's body. A function's own operands shadow nothing here:
// operand names are not statement names, so a collision would already be a
// redefinition error upstream.
func localReferences(s *ast.ExpressionDefinition, byName map[string]*ast.ExpressionDefinition) []*ast.ExpressionDefinition {
	var deps []*ast.ExpressionDefinition
	seen := map[*ast.ExpressionDefinition]bool{}
	add := func(name string) {
		if dep, ok := byName[name]; ok && dep != s && !seen[dep] {
			seen[dep] = true
			deps = append(deps, dep)
		}
	}
	ast.Walk(s.Expression, func(n ast.Node) {
		switch v := n.(type) {
		case *ast.IdentifierRef:
			add(v.Name)
		case *ast.FunctionRef:
			if v.LibraryName == "" {
				add(v.Name)
			}
		}
	})
	return deps
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

	// The accessor's own nodes are synthesized and carry no result type, but
	// references to it resolve to the context's model type.
	if t.defTypeSpecs != nil {
		t.defTypeSpecs[contextName] = namedTS{dataType}
	}

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

// functionCounts counts a library's function definitions by name.
func functionCounts(lib *ast.Library) map[string]int {
	counts := map[string]int{}
	for _, st := range lib.Statements {
		if st.IsFunction {
			counts[st.Name]++
		}
	}
	return counts
}
