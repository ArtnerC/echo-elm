package parser

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/artnerc/echo-elm/internal/ast"
	"github.com/artnerc/echo-elm/internal/parser/cqlparser"
)

// astBuilder converts the ANTLR parse tree into an echo-elm AST.
type astBuilder struct {
	sourceName string
	stream     *antlr.CommonTokenStream
}

func newASTBuilder(sourceName string, stream *antlr.CommonTokenStream) *astBuilder {
	return &astBuilder{sourceName: sourceName, stream: stream}
}

// unquoteString removes surrounding single-quotes and unescapes \' sequences.
func unquoteString(s string) string {
	if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		s = s[1 : len(s)-1]
	}
	return strings.ReplaceAll(s, "\\'", "'")
}

// unquoteIdentifier handles IDENTIFIER, QUOTEDIDENTIFIER (double-quoted), and DELIMITEDIDENTIFIER (backtick-quoted).
func unquoteIdentifier(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '`' && s[len(s)-1] == '`') {
			inner := s[1 : len(s)-1]
			// Unescape \" → " inside quoted identifiers (CQL spec allows escaped quotes)
			return strings.ReplaceAll(inner, `\"`, `"`)
		}
	}
	return s
}

func accessLevel(ctx antlr.ParserRuleContext) ast.AccessLevel {
	if ctx == nil {
		return ast.Public
	}
	if strings.ToLower(ctx.GetText()) == "private" {
		return ast.Private
	}
	return ast.Public
}

// intervalFromCtx extracts a 1-based source interval from an ANTLR parser rule context.
// Start column is antlr 0-indexed + 1; stop column is the last char position (1-indexed).
func intervalFromCtx(ctx antlr.ParserRuleContext) ast.Interval {
	if ctx == nil {
		return ast.Interval{}
	}
	start := ctx.GetStart()
	if start == nil || start.GetLine() == 0 {
		return ast.Interval{}
	}
	startLine := start.GetLine()
	startCol := start.GetColumn() + 1

	stop := ctx.GetStop()
	endLine, endCol := startLine, startCol
	if stop != nil && stop.GetTokenIndex() >= 0 {
		endLine = stop.GetLine()
		endCol = stop.GetColumn() + len(stop.GetText())
	} else {
		endCol = startCol + len(start.GetText()) - 1
	}
	return ast.Interval{
		Start: ast.Position{Line: startLine, Column: startCol},
		Stop:  ast.Position{Line: endLine, Column: endCol},
	}
}

// setLoc sets the loc on an AST expression node, overriding any previously set location.
// Outer wrapper builders use this to widen inner spans to the full context span.
// The new location is only applied when the context provides a valid (non-zero) interval.
func setLoc(n ast.Expr, ctx antlr.ParserRuleContext) ast.Expr {
	if n == nil || ctx == nil {
		return n
	}
	loc := intervalFromCtx(ctx)
	if loc.Start.Line == 0 {
		return n
	}
	type setter interface{ SetLoc(ast.Interval) }
	if s, ok := n.(setter); ok {
		s.SetLoc(loc)
	}
	return n
}

// -----------------------------------------------------------------------
// Library (root)
// -----------------------------------------------------------------------

func (b *astBuilder) buildLibrary(ctx cqlparser.ILibraryContext) *ast.Library {
	lc := ctx.(*cqlparser.LibraryContext)
	lib := &ast.Library{}

	if ld := lc.LibraryDefinition(); ld != nil {
		lib.Name = b.buildVersionedIdentifier(ld)
	}

	// Header-level definitions (using, include, codesystem, valueset, code, concept, parameter)
	for _, def := range lc.AllDefinition() {
		dc := def.(*cqlparser.DefinitionContext)
		switch {
		case dc.UsingDefinition() != nil:
			lib.Usings = append(lib.Usings, b.buildUsing(dc.UsingDefinition()))
		case dc.IncludeDefinition() != nil:
			lib.Includes = append(lib.Includes, b.buildInclude(dc.IncludeDefinition()))
		case dc.CodesystemDefinition() != nil:
			lib.Codesystems = append(lib.Codesystems, b.buildCodesystem(dc.CodesystemDefinition()))
		case dc.ValuesetDefinition() != nil:
			lib.Valuesets = append(lib.Valuesets, b.buildValueset(dc.ValuesetDefinition()))
		case dc.CodeDefinition() != nil:
			lib.Codes = append(lib.Codes, b.buildCode(dc.CodeDefinition()))
		case dc.ConceptDefinition() != nil:
			lib.Concepts = append(lib.Concepts, b.buildConcept(dc.ConceptDefinition()))
		case dc.ParameterDefinition() != nil:
			lib.Parameters = append(lib.Parameters, b.buildParameter(dc.ParameterDefinition()))
		}
	}

	// Statement-level definitions (context, define, define function)
	var currentContext string // tracks the most recently declared context name
	for _, stmt := range lc.AllStatement() {
		sc := stmt.(*cqlparser.StatementContext)
		switch {
		case sc.ContextDefinition() != nil:
			ctx := b.buildContext(sc.ContextDefinition())
			lib.Context = ctx                                     // last (backward compat)
			lib.Contexts = append(lib.Contexts, ctx)             // all contexts
			currentContext = ctx.Name
		case sc.ExpressionDefinition() != nil:
			def := b.buildExpressionDef(sc.ExpressionDefinition())
			def.Context = currentContext
			lib.Statements = append(lib.Statements, def)
		case sc.FunctionDefinition() != nil:
			def := b.buildFunctionDef(sc.FunctionDefinition())
			def.Context = currentContext
			lib.Statements = append(lib.Statements, def)
		}
	}

	return lib
}

func (b *astBuilder) buildVersionedIdentifier(ctx cqlparser.ILibraryDefinitionContext) *ast.VersionedIdentifier {
	ldc := ctx.(*cqlparser.LibraryDefinitionContext)
	vi := &ast.VersionedIdentifier{}
	vi.SetLoc(intervalFromCtx(ldc))
	if qi := ldc.QualifiedIdentifier(); qi != nil {
		vi.Name = qi.GetText()
	}
	if vs := ldc.VersionSpecifier(); vs != nil {
		vi.Version = unquoteString(vs.GetText())
	}
	return vi
}

// -----------------------------------------------------------------------
// Usings
// -----------------------------------------------------------------------

func (b *astBuilder) buildUsing(ctx cqlparser.IUsingDefinitionContext) *ast.UsingDefinition {
	udc := ctx.(*cqlparser.UsingDefinitionContext)
	ud := &ast.UsingDefinition{}
	// Model name comes from the QualifiedIdentifier (e.g. "FHIR")
	if qi := udc.QualifiedIdentifier(); qi != nil {
		ud.ModelName = qi.GetText()
	}
	if li := udc.LocalIdentifier(); li != nil {
		ud.LocalName = li.GetText()
	}
	if ud.LocalName == "" {
		ud.LocalName = ud.ModelName
	}
	if vs := udc.VersionSpecifier(); vs != nil {
		ud.Version = unquoteString(vs.GetText())
	}
	ud.SetLoc(intervalFromCtx(udc))
	return ud
}

// -----------------------------------------------------------------------
// Includes
// -----------------------------------------------------------------------

func (b *astBuilder) buildInclude(ctx cqlparser.IIncludeDefinitionContext) *ast.IncludeDefinition {
	idc := ctx.(*cqlparser.IncludeDefinitionContext)
	id := &ast.IncludeDefinition{}
	if qi := idc.QualifiedIdentifier(); qi != nil {
		id.Path = qi.GetText()
	}
	if vs := idc.VersionSpecifier(); vs != nil {
		id.Version = unquoteString(vs.GetText())
	}
	if li := idc.LocalIdentifier(); li != nil {
		id.LocalName = li.GetText()
	}
	id.SetLoc(intervalFromCtx(idc))
	return id
}

// -----------------------------------------------------------------------
// Codesystems
// -----------------------------------------------------------------------

func (b *astBuilder) buildCodesystem(ctx cqlparser.ICodesystemDefinitionContext) *ast.CodesystemDefinition {
	cdc := ctx.(*cqlparser.CodesystemDefinitionContext)
	cs := &ast.CodesystemDefinition{}
	if am := cdc.AccessModifier(); am != nil {
		cs.AccessLevel = accessLevel(am)
	}
	if idc := cdc.Identifier(); idc != nil {
		cs.Name = unquoteIdentifier(idc.GetText())
	}
	if csid := cdc.CodesystemId(); csid != nil {
		cs.ID = unquoteString(csid.GetText())
	}
	if vs := cdc.VersionSpecifier(); vs != nil {
		cs.Version = unquoteString(vs.GetText())
	}
	cs.SetLoc(intervalFromCtx(cdc))
	return cs
}

// -----------------------------------------------------------------------
// Valuesets
// -----------------------------------------------------------------------

func (b *astBuilder) buildValueset(ctx cqlparser.IValuesetDefinitionContext) *ast.ValuesetDefinition {
	vdc := ctx.(*cqlparser.ValuesetDefinitionContext)
	vs := &ast.ValuesetDefinition{}
	if am := vdc.AccessModifier(); am != nil {
		vs.AccessLevel = accessLevel(am)
	}
	if idc := vdc.Identifier(); idc != nil {
		vs.Name = unquoteIdentifier(idc.GetText())
	}
	if vid := vdc.ValuesetId(); vid != nil {
		vs.ID = unquoteString(vid.GetText())
	}
	if vspec := vdc.VersionSpecifier(); vspec != nil {
		vs.Version = unquoteString(vspec.GetText())
	}
	vs.SetLoc(intervalFromCtx(vdc))
	return vs
}

// -----------------------------------------------------------------------
// Codes
// -----------------------------------------------------------------------

func (b *astBuilder) buildCode(ctx cqlparser.ICodeDefinitionContext) *ast.CodeDefinition {
	cdc := ctx.(*cqlparser.CodeDefinitionContext)
	cd := &ast.CodeDefinition{}
	if am := cdc.AccessModifier(); am != nil {
		cd.AccessLevel = accessLevel(am)
	}
	if idc := cdc.Identifier(); idc != nil {
		cd.Name = unquoteIdentifier(idc.GetText())
	}
	if cid := cdc.CodeId(); cid != nil {
		cd.Code = unquoteString(cid.GetText())
	}
	if csid := cdc.CodesystemIdentifier(); csid != nil {
		csic := csid.(*cqlparser.CodesystemIdentifierContext)
		if libid := csic.LibraryIdentifier(); libid != nil {
			cd.SystemName = unquoteIdentifier(libid.GetText()) + "." + unquoteIdentifier(csic.Identifier().GetText())
		} else {
			cd.SystemName = unquoteIdentifier(csic.Identifier().GetText())
		}
		cd.SystemLocator = intervalFromCtx(csic)
	}
	if disp := cdc.DisplayClause(); disp != nil {
		dc := disp.(*cqlparser.DisplayClauseContext)
		if s := dc.STRING(); s != nil {
			cd.Display = unquoteString(s.GetText())
		}
	}
	cd.SetLoc(intervalFromCtx(cdc))
	return cd
}

// -----------------------------------------------------------------------
// Concepts
// -----------------------------------------------------------------------

func (b *astBuilder) buildConcept(ctx cqlparser.IConceptDefinitionContext) *ast.ConceptDefinition {
	cdc := ctx.(*cqlparser.ConceptDefinitionContext)
	cd := &ast.ConceptDefinition{}
	if am := cdc.AccessModifier(); am != nil {
		cd.AccessLevel = accessLevel(am)
	}
	if idc := cdc.Identifier(); idc != nil {
		cd.Name = unquoteIdentifier(idc.GetText())
	}
	for _, ci := range cdc.AllCodeIdentifier() {
		cd.Codes = append(cd.Codes, ci.GetText())
	}
	cd.SetLoc(intervalFromCtx(cdc))
	return cd
}

// -----------------------------------------------------------------------
// Parameters
// -----------------------------------------------------------------------

func (b *astBuilder) buildParameter(ctx cqlparser.IParameterDefinitionContext) *ast.ParameterDefinition {
	pdc := ctx.(*cqlparser.ParameterDefinitionContext)
	pd := &ast.ParameterDefinition{}
	if am := pdc.AccessModifier(); am != nil {
		pd.AccessLevel = accessLevel(am)
	}
	if idc := pdc.Identifier(); idc != nil {
		pd.Name = unquoteIdentifier(idc.GetText())
	}
	if ts := pdc.TypeSpecifier(); ts != nil {
		spec := b.buildTypeSpecifier(ts)
		pd.ParameterType = &spec
	}
	if defaultExpr := pdc.Expression(); defaultExpr != nil {
		pd.Default = b.buildExpr(defaultExpr)
	}
	pd.SetLoc(intervalFromCtx(pdc))
	return pd
}

// -----------------------------------------------------------------------
// Context
// -----------------------------------------------------------------------

func (b *astBuilder) buildContext(ctx cqlparser.IContextDefinitionContext) *ast.ContextDefinition {
	cdc := ctx.(*cqlparser.ContextDefinitionContext)
	cd := &ast.ContextDefinition{}
	if mi := cdc.ModelIdentifier(); mi != nil {
		cd.Name = mi.GetText() + "." + cdc.Identifier().GetText()
	} else if idc := cdc.Identifier(); idc != nil {
		cd.Name = idc.GetText()
	}
	cd.SetLoc(intervalFromCtx(cdc))
	return cd
}

// -----------------------------------------------------------------------
// Statements
// -----------------------------------------------------------------------

// extractAnnotations extracts structured @tag annotations from block comments
// immediately preceding the token at tokenIndex (using the hidden channel).
func (b *astBuilder) extractAnnotations(tokenIndex int) []ast.CQLAnnotation {
	if b.stream == nil {
		return nil
	}
	hidden := b.stream.GetHiddenTokensToLeft(tokenIndex, antlr.TokenHiddenChannel)
	if len(hidden) == 0 {
		return nil
	}
	var result []ast.CQLAnnotation
	for _, tok := range hidden {
		text := tok.GetText()
		if !strings.HasPrefix(text, "/*") {
			continue
		}
		// Strip /* prefix and */ suffix, preserving internal content (including # markers).
		inner := text[2:] // strip /*
		if idx := strings.LastIndex(inner, "*/"); idx >= 0 {
			inner = inner[:idx]
		}
		if !strings.Contains(inner, "@") {
			continue
		}
		tags := parseBlockCommentTags(inner)
		if len(tags) > 0 {
			result = append(result, ast.CQLAnnotation{Tags: tags})
		}
	}
	return result
}

// parseBlockCommentTags parses CQL @tag: value annotations from the raw interior
// (already stripped of /* and */) of a CQL block comment.
// Tags may have single-line values (`@tag: value`) or multi-line values enclosed
// in `#...#` markers (`@tag: #multi\nline#`). The # markers are preserved verbatim
// in the emitted value to match CQF 4.8.0 behavior.
func parseBlockCommentTags(inner string) []ast.CQLAnnotationTag {
	var tags []ast.CQLAnnotationTag
	var currentName string
	var currentValue strings.Builder
	inHash := false
	inTag := false

	lines := strings.Split(inner, "\n")
	for _, line := range lines {
		// Detect start of a new @tag on this line.
		trimmed := strings.TrimSpace(line)
		atIdx := strings.Index(trimmed, "@")
		if !inHash && atIdx >= 0 && (atIdx == 0 || isOnlyWhitespace(trimmed[:atIdx])) {
			// Flush previous tag.
			if inTag {
				tags = append(tags, ast.CQLAnnotationTag{
					Name:  currentName,
					Value: strings.TrimRight(currentValue.String(), " \t"),
				})
				currentValue.Reset()
			}
			// Parse "@name: value" or "@name value".
			rest := trimmed[atIdx+1:]
			colonIdx := strings.Index(rest, ":")
			spaceIdx := strings.IndexByte(rest, ' ')
			var nameEnd int
			var afterTag string
			if colonIdx >= 0 && (spaceIdx < 0 || colonIdx <= spaceIdx) {
				nameEnd = colonIdx
				afterTag = strings.TrimLeft(rest[colonIdx+1:], " \t")
			} else if spaceIdx >= 0 {
				nameEnd = spaceIdx
				afterTag = strings.TrimLeft(rest[spaceIdx+1:], " \t")
			} else {
				nameEnd = len(rest)
				afterTag = ""
			}
			currentName = strings.TrimSpace(rest[:nameEnd])
			inTag = true
			// Check whether the value starts with # (multi-line hash-delimited).
			if strings.HasPrefix(afterTag, "#") && !strings.HasSuffix(afterTag, "#") {
				// Multi-line hash value: value starts with # and end # is on a later line.
				inHash = true
				currentValue.WriteString(afterTag)
			} else {
				inHash = false
				currentValue.WriteString(afterTag)
			}
			continue
		}

		if !inTag {
			continue
		}

		// Continuation lines.
		if inHash {
			// Append raw line (with leading/trailing space preserved, matching CQF output).
			currentValue.WriteByte('\n')
			// Check if this line ends the hash block.
			if strings.HasSuffix(trimmed, "#") {
				currentValue.WriteString(trimmed)
				inHash = false
			} else {
				currentValue.WriteString(line)
			}
		}
		// Single-line tags don't have continuation lines.
	}
	// Flush last tag.
	if inTag {
		tags = append(tags, ast.CQLAnnotationTag{
			Name:  currentName,
			Value: strings.TrimRight(currentValue.String(), " \t"),
		})
	}
	return tags
}

// isOnlyWhitespace returns true if all runes in s are whitespace.
func isOnlyWhitespace(s string) bool {
	for _, r := range s {
		if r != ' ' && r != '\t' && r != '\r' && r != '\n' {
			return false
		}
	}
	return true
}

func (b *astBuilder) buildExpressionDef(ctx cqlparser.IExpressionDefinitionContext) *ast.ExpressionDefinition {
	edc := ctx.(*cqlparser.ExpressionDefinitionContext)
	ed := &ast.ExpressionDefinition{}
	if am := edc.AccessModifier(); am != nil {
		ed.AccessLevel = accessLevel(am)
	}
	if idc := edc.Identifier(); idc != nil {
		ed.Name = unquoteIdentifier(idc.GetText())
	}
	if expr := edc.Expression(); expr != nil {
		ed.Expression = b.buildExpr(expr)
	}
	startTok := edc.GetStart()
	if startTok != nil {
		ed.Annotations = b.extractAnnotations(startTok.GetTokenIndex())
	}
	ed.SetLoc(intervalFromCtx(edc))
	return ed
}

func (b *astBuilder) buildFunctionDef(ctx cqlparser.IFunctionDefinitionContext) *ast.ExpressionDefinition {
	fdc := ctx.(*cqlparser.FunctionDefinitionContext)
	ed := &ast.ExpressionDefinition{IsFunction: true}
	if am := fdc.AccessModifier(); am != nil {
		ed.AccessLevel = accessLevel(am)
	}
	if fdc.FluentModifier() != nil {
		ed.IsFluent = true
	}
	if idc := fdc.IdentifierOrFunctionIdentifier(); idc != nil {
		ed.Name = unquoteIdentifier(idc.GetText())
	}
	for _, op := range fdc.AllOperandDefinition() {
		odc := op.(*cqlparser.OperandDefinitionContext)
		od := &ast.OperandDef{}
		if idc := odc.ReferentialIdentifier(); idc != nil {
			od.Name = idc.GetText()
		}
		if ts := odc.TypeSpecifier(); ts != nil {
			spec := b.buildTypeSpecifier(ts)
			od.Type = &spec
		}
		ed.Operands = append(ed.Operands, od)
	}
	if ts := fdc.TypeSpecifier(); ts != nil {
		spec := b.buildTypeSpecifier(ts)
		ed.ReturnType = &spec
	}
	if fb := fdc.FunctionBody(); fb != nil {
		if expr := fb.(*cqlparser.FunctionBodyContext).Expression(); expr != nil {
			ed.Expression = b.buildExpr(expr)
		}
	}
	startTok := fdc.GetStart()
	if startTok != nil {
		ed.Annotations = b.extractAnnotations(startTok.GetTokenIndex())
	}
	ed.SetLoc(intervalFromCtx(fdc))
	return ed
}

// buildTypeSpecifier builds an ast.TypeSpecifier from a TypeSpecifierContext.
func (b *astBuilder) buildTypeSpecifier(ctx cqlparser.ITypeSpecifierContext) ast.TypeSpecifier {
	if ctx == nil {
		return nil
	}
	if named := ctx.NamedTypeSpecifier(); named != nil {
		return b.buildNamedTypeSpecifier(named)
	}
	if list := ctx.ListTypeSpecifier(); list != nil {
		ldc := list.(*cqlparser.ListTypeSpecifierContext)
		lt := &ast.ListTypeSpecifier{}
		if inner := ldc.TypeSpecifier(); inner != nil {
			lt.ElementType = b.buildTypeSpecifier(inner)
		}
		lt.SetLoc(intervalFromCtx(ldc))
		return lt
	}
	if interval := ctx.IntervalTypeSpecifier(); interval != nil {
		idc := interval.(*cqlparser.IntervalTypeSpecifierContext)
		it := &ast.IntervalTypeSpecifier{}
		if inner := idc.TypeSpecifier(); inner != nil {
			it.PointType = b.buildTypeSpecifier(inner)
		}
		it.SetLoc(intervalFromCtx(idc))
		return it
	}
	if tuple := ctx.TupleTypeSpecifier(); tuple != nil {
		tdc := tuple.(*cqlparser.TupleTypeSpecifierContext)
		tt := &ast.TupleTypeSpecifier{}
		for _, elem := range tdc.AllTupleElementDefinition() {
			edc := elem.(*cqlparser.TupleElementDefinitionContext)
			te := &ast.TupleTypeElement{}
			if ri := edc.ReferentialIdentifier(); ri != nil {
				te.Name = ri.GetText()
			}
			if ts := edc.TypeSpecifier(); ts != nil {
				te.Type = b.buildTypeSpecifier(ts)
			}
			te.SetLoc(intervalFromCtx(edc))
			tt.Elements = append(tt.Elements, te)
		}
		tt.SetLoc(intervalFromCtx(tdc))
		return tt
	}
	if choice := ctx.ChoiceTypeSpecifier(); choice != nil {
		cdc := choice.(*cqlparser.ChoiceTypeSpecifierContext)
		ct := &ast.ChoiceTypeSpecifier{}
		for _, ts := range cdc.AllTypeSpecifier() {
			ct.Types = append(ct.Types, b.buildTypeSpecifier(ts))
		}
		ct.SetLoc(intervalFromCtx(cdc))
		return ct
	}
	return nil
}

func (b *astBuilder) buildNamedTypeSpecifier(ctx cqlparser.INamedTypeSpecifierContext) *ast.NamedTypeSpecifier {
	ndc := ctx.(*cqlparser.NamedTypeSpecifierContext)
	nts := &ast.NamedTypeSpecifier{}
	qualifiers := ndc.AllQualifier()
	if len(qualifiers) > 0 {
		nts.Qualifier = qualifiers[0].GetText()
	}
	if ri := ndc.ReferentialOrTypeNameIdentifier(); ri != nil {
		nts.Name = ri.GetText()
	}
	nts.SetLoc(intervalFromCtx(ndc))
	return nts
}
