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
}

func newASTBuilder(sourceName string) *astBuilder {
	return &astBuilder{sourceName: sourceName}
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
			return s[1 : len(s)-1]
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
	for _, stmt := range lc.AllStatement() {
		sc := stmt.(*cqlparser.StatementContext)
		switch {
		case sc.ContextDefinition() != nil:
			lib.Context = b.buildContext(sc.ContextDefinition())
		case sc.ExpressionDefinition() != nil:
			lib.Statements = append(lib.Statements, b.buildExpressionDef(sc.ExpressionDefinition()))
		case sc.FunctionDefinition() != nil:
			lib.Statements = append(lib.Statements, b.buildFunctionDef(sc.FunctionDefinition()))
		}
	}

	return lib
}

func (b *astBuilder) buildVersionedIdentifier(ctx cqlparser.ILibraryDefinitionContext) *ast.VersionedIdentifier {
	ldc := ctx.(*cqlparser.LibraryDefinitionContext)
	vi := &ast.VersionedIdentifier{}
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
			cd.SystemName = libid.GetText() + "." + csic.Identifier().GetText()
		} else {
			cd.SystemName = csic.Identifier().GetText()
		}
	}
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
	return cd
}

// -----------------------------------------------------------------------
// Statements
// -----------------------------------------------------------------------

func (b *astBuilder) buildExpressionDef(ctx cqlparser.IExpressionDefinitionContext) *ast.ExpressionDefinition {
	edc := ctx.(*cqlparser.ExpressionDefinitionContext)
	ed := &ast.ExpressionDefinition{}
	if am := edc.AccessModifier(); am != nil {
		ed.AccessLevel = accessLevel(am)
	}
	if idc := edc.Identifier(); idc != nil {
		ed.Name = unquoteIdentifier(idc.GetText())
	}
	// Expression body built out in Phase 2
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
		return lt
	}
	if interval := ctx.IntervalTypeSpecifier(); interval != nil {
		idc := interval.(*cqlparser.IntervalTypeSpecifierContext)
		it := &ast.IntervalTypeSpecifier{}
		if inner := idc.TypeSpecifier(); inner != nil {
			it.PointType = b.buildTypeSpecifier(inner)
		}
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
			tt.Elements = append(tt.Elements, te)
		}
		return tt
	}
	if choice := ctx.ChoiceTypeSpecifier(); choice != nil {
		cdc := choice.(*cqlparser.ChoiceTypeSpecifierContext)
		ct := &ast.ChoiceTypeSpecifier{}
		for _, ts := range cdc.AllTypeSpecifier() {
			ct.Types = append(ct.Types, b.buildTypeSpecifier(ts))
		}
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
	return nts
}
