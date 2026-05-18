package parser

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/artnerc/echo-elm/internal/ast"
	"github.com/artnerc/echo-elm/internal/parser/cqlparser"
)

// -----------------------------------------------------------------------
// Top-level: expression
// -----------------------------------------------------------------------

func (b *astBuilder) buildExpr(ctx cqlparser.IExpressionContext) ast.Expr {
	if ctx == nil {
		return nil
	}
	switch c := ctx.(type) {
	case *cqlparser.TermExpressionContext:
		return b.buildExprTerm(c.ExpressionTerm())

	case *cqlparser.RetrieveExpressionContext:
		return b.buildRetrieve(c.Retrieve())

	case *cqlparser.QueryExpressionContext:
		return b.buildQuery(c.Query())

	case *cqlparser.NotExpressionContext:
		return &ast.UnaryExpr{Op: "Not", Operand: b.buildExpr(c.Expression())}

	case *cqlparser.ExistenceExpressionContext:
		return &ast.UnaryExpr{Op: "Exists", Operand: b.buildExpr(c.Expression())}

	case *cqlparser.BooleanExpressionContext:
		operand := b.buildExpr(c.Expression())
		text := strings.ToLower(c.GetText())
		negated := strings.Contains(text, "isnot") || strings.Contains(text, "not")
		switch {
		case strings.HasSuffix(text, "isnull"):
			return &ast.TypeIsExpr{Operand: operand, IsNull: true}
		case strings.HasSuffix(text, "isnotnull"):
			return &ast.TypeIsExpr{Operand: operand, IsNull: true, Negated: true}
		case strings.HasSuffix(text, "istrue"):
			return &ast.TypeIsExpr{Operand: operand, IsTrue: true, Negated: negated}
		case strings.HasSuffix(text, "isfalse"):
			return &ast.TypeIsExpr{Operand: operand, IsFalse: true, Negated: negated}
		default:
			return &ast.TypeIsExpr{Operand: operand, IsNull: true}
		}

	case *cqlparser.TypeExpressionContext:
		operand := b.buildExpr(c.Expression())
		ts := b.buildTypeSpecifier(c.TypeSpecifier())
		text := strings.ToLower(c.GetText())
		if strings.Contains(text, " is ") || strings.HasPrefix(text, "is") {
			return &ast.TypeIsExpr{Operand: operand, TypeSpec: ts}
		}
		return &ast.TypeAsExpr{Operand: operand, TypeSpec: ts, Strict: false}

	case *cqlparser.CastExpressionContext:
		operand := b.buildExpr(c.Expression())
		ts := b.buildTypeSpecifier(c.TypeSpecifier())
		return &ast.TypeAsExpr{Operand: operand, TypeSpec: ts, Strict: true}

	case *cqlparser.BetweenExpressionContext:
		exprTerms := c.AllExpressionTerm()
		var low, high ast.Expr
		if len(exprTerms) > 0 {
			low = b.buildExprTerm(exprTerms[0])
		}
		if len(exprTerms) > 1 {
			high = b.buildExprTerm(exprTerms[1])
		}
		text := strings.ToLower(c.GetText())
		properly := strings.Contains(text, "properly")
		return &ast.BetweenExpr{
			Operand:  b.buildExpr(c.Expression()),
			Low:      low,
			High:     high,
			Properly: properly,
		}

	case *cqlparser.DurationBetweenExpressionContext:
		prec := b.singularizePrecision(c.PluralDateTimePrecision().GetText())
		exprTerms := c.AllExpressionTerm()
		var low, high ast.Expr
		if len(exprTerms) > 0 {
			low = b.buildExprTerm(exprTerms[0])
		}
		if len(exprTerms) > 1 {
			high = b.buildExprTerm(exprTerms[1])
		}
		return &ast.DurationBetweenExpr{Precision: prec, Low: low, High: high}

	case *cqlparser.DifferenceBetweenExpressionContext:
		prec := b.singularizePrecision(c.PluralDateTimePrecision().GetText())
		exprTerms := c.AllExpressionTerm()
		var low, high ast.Expr
		if len(exprTerms) > 0 {
			low = b.buildExprTerm(exprTerms[0])
		}
		if len(exprTerms) > 1 {
			high = b.buildExprTerm(exprTerms[1])
		}
		return &ast.DifferenceBetweenExpr{Precision: prec, Low: low, High: high}

	case *cqlparser.InFixSetExpressionContext:
		exprs := c.AllExpression()
		var left, right ast.Expr
		if len(exprs) > 0 {
			left = b.buildExpr(exprs[0])
		}
		if len(exprs) > 1 {
			right = b.buildExpr(exprs[1])
		}
		op := b.setOp(c)
		return &ast.BinaryExpr{Op: op, Left: left, Right: right}

	case *cqlparser.InequalityExpressionContext:
		exprs := c.AllExpression()
		var left, right ast.Expr
		if len(exprs) > 0 {
			left = b.buildExpr(exprs[0])
		}
		if len(exprs) > 1 {
			right = b.buildExpr(exprs[1])
		}
		op := b.inequalityOp(c)
		return &ast.BinaryExpr{Op: op, Left: left, Right: right}

	case *cqlparser.EqualityExpressionContext:
		exprs := c.AllExpression()
		var left, right ast.Expr
		if len(exprs) > 0 {
			left = b.buildExpr(exprs[0])
		}
		if len(exprs) > 1 {
			right = b.buildExpr(exprs[1])
		}
		op := b.equalityOp(c)
		return &ast.BinaryExpr{Op: op, Left: left, Right: right}

	case *cqlparser.MembershipExpressionContext:
		exprs := c.AllExpression()
		var left, right ast.Expr
		if len(exprs) > 0 {
			left = b.buildExpr(exprs[0])
		}
		if len(exprs) > 1 {
			right = b.buildExpr(exprs[1])
		}
		text := c.GetText()
		op := "In"
		if strings.Contains(strings.ToLower(text), "contains") {
			op = "Contains"
		}
		prec := ""
		if dtps := c.DateTimePrecisionSpecifier(); dtps != nil {
			prec = b.singularizePrecision(strings.TrimSuffix(dtps.GetText(), "of"))
		}
		return &ast.BinaryExpr{Op: op, Left: left, Right: right, Precision: prec}

	case *cqlparser.AndExpressionContext:
		exprs := c.AllExpression()
		return &ast.BinaryExpr{Op: "And",
			Left:  b.buildExpr(exprs[0]),
			Right: b.buildExpr(exprs[1]),
		}

	case *cqlparser.OrExpressionContext:
		exprs := c.AllExpression()
		text := strings.ToLower(c.GetText())
		op := "Or"
		if strings.Contains(text, "xor") {
			op = "Xor"
		}
		return &ast.BinaryExpr{Op: op,
			Left:  b.buildExpr(exprs[0]),
			Right: b.buildExpr(exprs[1]),
		}

	case *cqlparser.ImpliesExpressionContext:
		exprs := c.AllExpression()
		return &ast.BinaryExpr{Op: "Implies",
			Left:  b.buildExpr(exprs[0]),
			Right: b.buildExpr(exprs[1]),
		}

	case *cqlparser.TimingExpressionContext:
		exprs := c.AllExpression()
		var left, right ast.Expr
		if len(exprs) > 0 {
			left = b.buildExpr(exprs[0])
		}
		if len(exprs) > 1 {
			right = b.buildExpr(exprs[1])
		}
		op, prec := b.timingOp(c.IntervalOperatorPhrase())
		return &ast.TimingExpr{Op: op, Left: left, Right: right, Precision: prec}

	default:
		return &ast.IdentifierRef{Name: ctx.GetText()}
	}
}

// -----------------------------------------------------------------------
// Expression terms
// -----------------------------------------------------------------------

func (b *astBuilder) buildExprTerm(ctx cqlparser.IExpressionTermContext) ast.Expr {
	if ctx == nil {
		return nil
	}
	switch c := ctx.(type) {
	case *cqlparser.TermExpressionTermContext:
		return b.buildTerm(c.Term())

	case *cqlparser.InvocationExpressionTermContext:
		source := b.buildExprTerm(c.ExpressionTerm())
		return b.buildQualifiedInvocation(c.QualifiedInvocation(), source)

	case *cqlparser.IndexedExpressionTermContext:
		source := b.buildExprTerm(c.ExpressionTerm())
		index := b.buildExpr(c.Expression())
		return &ast.IndexedAccessExpr{Source: source, Index: index}

	case *cqlparser.ConversionExpressionTermContext:
		operand := b.buildExpr(c.Expression())
		if ts := c.TypeSpecifier(); ts != nil {
			return &ast.ConvertExpr{Operand: operand, TypeSpec: b.buildTypeSpecifier(ts)}
		}
		// unit-based conversion — treat as function call
		return &ast.FunctionRef{Name: "convert", Operands: []ast.Expr{operand}}

	case *cqlparser.PolarityExpressionTermContext:
		operand := b.buildExprTerm(c.ExpressionTerm())
		if strings.HasPrefix(c.GetText(), "-") {
			return &ast.UnaryExpr{Op: "Negate", Operand: operand}
		}
		return operand

	case *cqlparser.TimeBoundaryExpressionTermContext:
		source := b.buildExprTerm(c.ExpressionTerm())
		text := strings.ToLower(c.GetText())
		boundary := "Start"
		if strings.HasPrefix(text, "end") {
			boundary = "End"
		}
		return &ast.TimeBoundaryExpr{Boundary: boundary, Source: source}

	case *cqlparser.TimeUnitExpressionTermContext:
		source := b.buildExprTerm(c.ExpressionTerm())
		comp := b.singularizePrecision(c.DateTimeComponent().GetText())
		return &ast.DateTimeComponentExpr{Precision: comp, Source: source}

	case *cqlparser.DurationExpressionTermContext:
		source := b.buildExprTerm(c.ExpressionTerm())
		prec := b.singularizePrecision(c.PluralDateTimePrecision().GetText())
		return &ast.DurationExpr{Precision: prec, Source: source}

	case *cqlparser.DifferenceExpressionTermContext:
		source := b.buildExprTerm(c.ExpressionTerm())
		prec := b.singularizePrecision(c.PluralDateTimePrecision().GetText())
		return &ast.DifferenceExpr{Precision: prec, Source: source}

	case *cqlparser.WidthExpressionTermContext:
		source := b.buildExprTerm(c.ExpressionTerm())
		return &ast.WidthExpr{Source: source}

	case *cqlparser.SuccessorExpressionTermContext:
		source := b.buildExprTerm(c.ExpressionTerm())
		return &ast.SuccessorExpr{Source: source}

	case *cqlparser.PredecessorExpressionTermContext:
		source := b.buildExprTerm(c.ExpressionTerm())
		return &ast.PredecessorExpr{Source: source}

	case *cqlparser.ElementExtractorExpressionTermContext:
		source := b.buildExprTerm(c.ExpressionTerm())
		return &ast.SingletonFromExpr{Source: source}

	case *cqlparser.PointExtractorExpressionTermContext:
		source := b.buildExprTerm(c.ExpressionTerm())
		return &ast.PointFromExpr{Source: source}

	case *cqlparser.TypeExtentExpressionTermContext:
		ts := b.buildNamedTypeSpecifier(c.NamedTypeSpecifier())
		text := strings.ToLower(c.GetText())
		extent := "minimum"
		if strings.HasPrefix(text, "maximum") {
			extent = "maximum"
		}
		return &ast.TypeExtentExpr{Extent: extent, TypeSpec: ts}

	case *cqlparser.PowerExpressionTermContext:
		terms := c.AllExpressionTerm()
		return &ast.BinaryExpr{Op: "Power",
			Left:  b.buildExprTerm(terms[0]),
			Right: b.buildExprTerm(terms[1]),
		}

	case *cqlparser.MultiplicationExpressionTermContext:
		terms := c.AllExpressionTerm()
		left := b.buildExprTerm(terms[0])
		right := b.buildExprTerm(terms[1])
		op := b.multiplicationOp(c)
		return &ast.BinaryExpr{Op: op, Left: left, Right: right}

	case *cqlparser.AdditionExpressionTermContext:
		terms := c.AllExpressionTerm()
		left := b.buildExprTerm(terms[0])
		right := b.buildExprTerm(terms[1])
		op := b.additionOp(c)
		return &ast.BinaryExpr{Op: op, Left: left, Right: right}

	case *cqlparser.IfThenElseExpressionTermContext:
		exprs := c.AllExpression()
		return &ast.TernaryExpr{
			Condition: b.buildExpr(exprs[0]),
			ThenExpr:  b.buildExpr(exprs[1]),
			ElseExpr:  b.buildExpr(exprs[2]),
		}

	case *cqlparser.CaseExpressionTermContext:
		exprs := c.AllExpression()
		items := c.AllCaseExpressionItem()
		ce := &ast.CaseExpr{}
		// If exprs count > len(items)+1, the first expr is the comparand.
		// Grammar: 'case' expression? caseItem+ 'else' expression 'end'
		// exprs[0..n-2] are within case items when-then, exprs[n-1] is else
		// If no comparand: only case items + else = (2*n + 1) exprs for n items
		// If comparand: (2*n + 2) exprs
		numItems := len(items)
		hasComparand := len(exprs) == 2*numItems+2
		idx := 0
		if hasComparand {
			ce.Comparand = b.buildExpr(exprs[idx])
			idx++
		}
		for _, item := range items {
			itemExprs := item.(*cqlparser.CaseExpressionItemContext).AllExpression()
			ci := &ast.CaseItem{}
			if len(itemExprs) > 0 {
				ci.When = b.buildExpr(itemExprs[0])
			}
			if len(itemExprs) > 1 {
				ci.Then = b.buildExpr(itemExprs[1])
			}
			ce.Items = append(ce.Items, ci)
		}
		// Last expression is the else
		ce.Else = b.buildExpr(exprs[len(exprs)-1])
		return ce

	case *cqlparser.AggregateExpressionTermContext:
		operand := b.buildExpr(c.Expression())
		text := strings.ToLower(c.GetText())
		op := "Distinct"
		if strings.HasPrefix(text, "flatten") {
			op = "Flatten"
		}
		return &ast.AggregateExpr{Op: op, Operand: operand}

	case *cqlparser.SetAggregateExpressionTermContext:
		exprs := c.AllExpression()
		var operand ast.Expr
		if len(exprs) > 0 {
			operand = b.buildExpr(exprs[0])
		}
		text := strings.ToLower(c.GetText())
		op := "Expand"
		if strings.HasPrefix(text, "collapse") {
			op = "Collapse"
		}
		var perClause ast.Expr
		if len(exprs) > 1 {
			perClause = b.buildExpr(exprs[1])
		}
		return &ast.SetAggregateExpr{Op: op, Operand: operand, PerClause: perClause}

	default:
		return &ast.IdentifierRef{Name: ctx.GetText()}
	}
}

// -----------------------------------------------------------------------
// Term
// -----------------------------------------------------------------------

func (b *astBuilder) buildTerm(ctx cqlparser.ITermContext) ast.Expr {
	if ctx == nil {
		return nil
	}
	switch c := ctx.(type) {
	case *cqlparser.LiteralTermContext:
		return b.buildLiteral(c.Literal())

	case *cqlparser.InvocationTermContext:
		return b.buildInvocation(c.Invocation(), nil)

	case *cqlparser.ParenthesizedTermContext:
		return b.buildExpr(c.Expression())

	case *cqlparser.IntervalSelectorTermContext:
		return b.buildIntervalSelector(c.IntervalSelector())

	case *cqlparser.TupleSelectorTermContext:
		return b.buildTupleSelector(c.TupleSelector())

	case *cqlparser.ListSelectorTermContext:
		return b.buildListSelector(c.ListSelector())

	case *cqlparser.InstanceSelectorTermContext:
		return b.buildInstanceSelector(c.InstanceSelector())

	case *cqlparser.CodeSelectorTermContext:
		return b.buildCodeSelector(c.CodeSelector())

	case *cqlparser.ConceptSelectorTermContext:
		return b.buildConceptSelector(c.ConceptSelector())

	case *cqlparser.ExternalConstantTermContext:
		ec := c.ExternalConstant().(*cqlparser.ExternalConstantContext)
		name := ec.GetText()
		if len(name) > 1 && name[0] == '%' {
			name = name[1:]
		}
		name = unquoteIdentifier(unquoteString(name))
		return &ast.ExternalConstantExpr{Name: name}

	default:
		return &ast.IdentifierRef{Name: ctx.GetText()}
	}
}

// -----------------------------------------------------------------------
// Invocation
// -----------------------------------------------------------------------

func (b *astBuilder) buildInvocation(ctx cqlparser.IInvocationContext, source ast.Expr) ast.Expr {
	if ctx == nil {
		return source
	}
	switch c := ctx.(type) {
	case *cqlparser.MemberInvocationContext:
		name := unquoteIdentifier(c.ReferentialIdentifier().GetText())
		if source != nil {
			return &ast.PropertyExpr{Source: source, Path: name}
		}
		return &ast.IdentifierRef{Name: name}

	case *cqlparser.FunctionInvocationContext:
		fn := c.Function().(*cqlparser.FunctionContext)
		name := unquoteIdentifier(fn.ReferentialIdentifier().GetText())
		var operands []ast.Expr
		if pl := fn.ParamList(); pl != nil {
			for _, expr := range pl.(*cqlparser.ParamListContext).AllExpression() {
				operands = append(operands, b.buildExpr(expr))
			}
		}
		if source != nil {
			// method call: source.name(args)
			return &ast.FunctionRef{LibraryName: "", Name: name,
				Operands: append([]ast.Expr{source}, operands...)}
		}
		return &ast.FunctionRef{Name: name, Operands: operands}

	case *cqlparser.ThisInvocationContext:
		return &ast.ThisExpr{}

	case *cqlparser.IndexInvocationContext:
		return &ast.IndexExpr{}

	case *cqlparser.TotalInvocationContext:
		return &ast.TotalExpr{}

	default:
		return &ast.IdentifierRef{Name: ctx.GetText()}
	}
}

func (b *astBuilder) buildQualifiedInvocation(ctx cqlparser.IQualifiedInvocationContext, source ast.Expr) ast.Expr {
	if ctx == nil {
		return source
	}
	switch c := ctx.(type) {
	case *cqlparser.QualifiedMemberInvocationContext:
		name := unquoteIdentifier(c.ReferentialIdentifier().GetText())
		return &ast.PropertyExpr{Source: source, Path: name}

	case *cqlparser.QualifiedFunctionInvocationContext:
		qf := c.QualifiedFunction().(*cqlparser.QualifiedFunctionContext)
		name := unquoteIdentifier(qf.IdentifierOrFunctionIdentifier().GetText())
		var operands []ast.Expr
		if pl := qf.ParamList(); pl != nil {
			for _, expr := range pl.(*cqlparser.ParamListContext).AllExpression() {
				operands = append(operands, b.buildExpr(expr))
			}
		}
		return &ast.FunctionRef{Name: name,
			Operands: append([]ast.Expr{source}, operands...)}

	default:
		return source
	}
}

// -----------------------------------------------------------------------
// Literals
// -----------------------------------------------------------------------

func (b *astBuilder) buildLiteral(ctx cqlparser.ILiteralContext) ast.Expr {
	if ctx == nil {
		return &ast.NullLiteral{}
	}
	switch c := ctx.(type) {
	case *cqlparser.BooleanLiteralContext:
		return &ast.BooleanLiteral{Value: strings.ToLower(c.GetText()) == "true"}

	case *cqlparser.NullLiteralContext:
		return &ast.NullLiteral{}

	case *cqlparser.StringLiteralContext:
		return &ast.StringLiteral{Value: unquoteString(c.GetText())}

	case *cqlparser.NumberLiteralContext:
		text := c.GetText()
		if strings.Contains(text, ".") {
			return &ast.DecimalLiteral{Value: text}
		}
		return &ast.IntegerLiteral{Value: parseInt(text)}

	case *cqlparser.LongNumberLiteralContext:
		text := c.GetText()
		if len(text) > 0 && (text[len(text)-1] == 'L' || text[len(text)-1] == 'l') {
			text = text[:len(text)-1]
		}
		return &ast.LongLiteral{Value: parseInt(text)}

	case *cqlparser.DateTimeLiteralContext:
		text := c.GetText()
		if len(text) > 1 && text[0] == '@' {
			text = text[1:]
		}
		return &ast.DateTimeLiteral{Value: text}

	case *cqlparser.DateLiteralContext:
		text := c.GetText()
		if len(text) > 1 && text[0] == '@' {
			text = text[1:]
		}
		return &ast.DateLiteral{Value: text}

	case *cqlparser.TimeLiteralContext:
		text := c.GetText()
		if len(text) > 2 && text[0] == '@' && (text[1] == 'T' || text[1] == 't') {
			text = text[2:]
		}
		return &ast.TimeLiteral{Value: text}

	case *cqlparser.QuantityLiteralContext:
		return b.buildQuantityLiteral(c.Quantity())

	case *cqlparser.RatioLiteralContext:
		rc := c.Ratio().(*cqlparser.RatioContext)
		quantities := rc.AllQuantity()
		var num, den *ast.QuantityLiteral
		if len(quantities) > 0 {
			n := b.buildQuantityLiteral(quantities[0])
			num = n
		}
		if len(quantities) > 1 {
			d := b.buildQuantityLiteral(quantities[1])
			den = d
		}
		return &ast.RatioLiteral{Numerator: num, Denominator: den}

	default:
		return &ast.NullLiteral{}
	}
}

func (b *astBuilder) buildQuantityLiteral(ctx cqlparser.IQuantityContext) *ast.QuantityLiteral {
	if ctx == nil {
		return &ast.QuantityLiteral{}
	}
	qc := ctx.(*cqlparser.QuantityContext)
	value := qc.NUMBER().GetText()
	unit := ""
	if u := qc.Unit(); u != nil {
		unit = unquoteString(u.GetText())
	}
	return &ast.QuantityLiteral{Value: value, Unit: unit}
}

// -----------------------------------------------------------------------
// Selectors
// -----------------------------------------------------------------------

func (b *astBuilder) buildIntervalSelector(ctx cqlparser.IIntervalSelectorContext) ast.Expr {
	isc := ctx.(*cqlparser.IntervalSelectorContext)
	exprs := isc.AllExpression()
	var low, high ast.Expr
	if len(exprs) > 0 {
		low = b.buildExpr(exprs[0])
	}
	if len(exprs) > 1 {
		high = b.buildExpr(exprs[1])
	}
	text := isc.GetText()
	// text starts with "Interval" followed by '[' or '('
	lowClosed := true
	highClosed := true
	if len(text) > 8 {
		if text[8] == '(' {
			lowClosed = false
		}
		if text[len(text)-1] == ')' {
			highClosed = false
		}
	}
	return &ast.IntervalExpr{
		Low:        low,
		High:       high,
		LowClosed:  lowClosed,
		HighClosed: highClosed,
	}
}

func (b *astBuilder) buildTupleSelector(ctx cqlparser.ITupleSelectorContext) ast.Expr {
	tsc := ctx.(*cqlparser.TupleSelectorContext)
	te := &ast.TupleExpr{}
	for _, elem := range tsc.AllTupleElementSelector() {
		esc := elem.(*cqlparser.TupleElementSelectorContext)
		key := esc.ReferentialIdentifier().GetText()
		val := b.buildExpr(esc.Expression())
		te.Elements = append(te.Elements, &ast.TupleElement{Name: key, Expression: val})
	}
	return te
}

func (b *astBuilder) buildListSelector(ctx cqlparser.IListSelectorContext) ast.Expr {
	lsc := ctx.(*cqlparser.ListSelectorContext)
	le := &ast.ListExpr{}
	if ts := lsc.TypeSpecifier(); ts != nil {
		le.TypeSpec = b.buildTypeSpecifier(ts)
	}
	for _, expr := range lsc.AllExpression() {
		le.Elements = append(le.Elements, b.buildExpr(expr))
	}
	return le
}

func (b *astBuilder) buildInstanceSelector(ctx cqlparser.IInstanceSelectorContext) ast.Expr {
	isc := ctx.(*cqlparser.InstanceSelectorContext)
	ie := &ast.InstanceExpr{
		TypeSpec: b.buildNamedTypeSpecifier(isc.NamedTypeSpecifier()),
	}
	for _, elem := range isc.AllInstanceElementSelector() {
		esc := elem.(*cqlparser.InstanceElementSelectorContext)
		key := esc.ReferentialIdentifier().GetText()
		val := b.buildExpr(esc.Expression())
		ie.Elements = append(ie.Elements, &ast.TupleElement{Name: key, Expression: val})
	}
	return ie
}

func (b *astBuilder) buildCodeSelector(ctx cqlparser.ICodeSelectorContext) ast.Expr {
	csc := ctx.(*cqlparser.CodeSelectorContext)
	code := unquoteString(csc.STRING().GetText())
	system := ""
	if csid := csc.CodesystemIdentifier(); csid != nil {
		system = csid.GetText()
	}
	display := ""
	if dc := csc.DisplayClause(); dc != nil {
		parts := strings.SplitN(dc.GetText(), "'", 3)
		if len(parts) >= 2 {
			display = parts[1]
		}
	}
	return &ast.CodeExpr{Code: code, System: system, Display: display}
}

func (b *astBuilder) buildConceptSelector(ctx cqlparser.IConceptSelectorContext) ast.Expr {
	csc := ctx.(*cqlparser.ConceptSelectorContext)
	ce := &ast.ConceptExpr{}
	for _, cs := range csc.AllCodeSelector() {
		code := b.buildCodeSelector(cs)
		if c, ok := code.(*ast.CodeExpr); ok {
			ce.Codes = append(ce.Codes, c)
		}
	}
	if dc := csc.DisplayClause(); dc != nil {
		parts := strings.SplitN(dc.GetText(), "'", 3)
		if len(parts) >= 2 {
			ce.Display = parts[1]
		}
	}
	return ce
}

// -----------------------------------------------------------------------
// Retrieve
// -----------------------------------------------------------------------

func (b *astBuilder) buildRetrieve(ctx cqlparser.IRetrieveContext) ast.Expr {
	if ctx == nil {
		return nil
	}
	rc := ctx.(*cqlparser.RetrieveContext)
	re := &ast.RetrieveExpr{}

	if nts := rc.NamedTypeSpecifier(); nts != nil {
		ntsc := nts.(*cqlparser.NamedTypeSpecifierContext)
		qualifiers := ntsc.AllQualifier()
		typeName := ntsc.ReferentialOrTypeNameIdentifier().GetText()
		if len(qualifiers) > 0 {
			re.DataType = qualifiers[0].GetText() + "." + typeName
		} else {
			re.DataType = typeName
		}
	}

	if cp := rc.CodePath(); cp != nil {
		re.CodeProperty = cp.GetText()
	}

	if cc := rc.CodeComparator(); cc != nil {
		re.CodeComparator = cc.GetText()
	}

	if term := rc.Terminology(); term != nil {
		re.Codes = b.buildExpr(term.(*cqlparser.TerminologyContext).Expression())
	}

	return re
}

// -----------------------------------------------------------------------
// Query
// -----------------------------------------------------------------------

func (b *astBuilder) buildQuery(ctx cqlparser.IQueryContext) ast.Expr {
	if ctx == nil {
		return nil
	}
	qc := ctx.(*cqlparser.QueryContext)
	q := &ast.QueryExpression{}

	// Sources
	if sc := qc.SourceClause(); sc != nil {
		for _, aqsc := range sc.(*cqlparser.SourceClauseContext).AllAliasedQuerySource() {
			q.Sources = append(q.Sources, b.buildAliasedQuerySource(aqsc))
		}
	}

	// Let
	if lc := qc.LetClause(); lc != nil {
		for _, item := range lc.(*cqlparser.LetClauseContext).AllLetClauseItem() {
			lci := item.(*cqlparser.LetClauseItemContext)
			q.Let = append(q.Let, &ast.LetClause{
				Identifier: unquoteIdentifier(lci.Identifier().GetText()),
				Expression: b.buildExpr(lci.Expression()),
			})
		}
	}

	// Relationships (with/without)
	for _, incl := range qc.AllQueryInclusionClause() {
		ic := incl.(*cqlparser.QueryInclusionClauseContext)
		if wc := ic.WithClause(); wc != nil {
			w := wc.(*cqlparser.WithClauseContext)
			rel := ast.QueryRelationship{Kind: "with"}
			rel.Source = b.buildAliasedQuerySource(w.AliasedQuerySource())
			rel.SuchThat = b.buildExpr(w.Expression())
			q.Relationship = append(q.Relationship, rel)
		}
		if woc := ic.WithoutClause(); woc != nil {
			wo := woc.(*cqlparser.WithoutClauseContext)
			rel := ast.QueryRelationship{Kind: "without"}
			rel.Source = b.buildAliasedQuerySource(wo.AliasedQuerySource())
			rel.SuchThat = b.buildExpr(wo.Expression())
			q.Relationship = append(q.Relationship, rel)
		}
	}

	// Where
	if wc := qc.WhereClause(); wc != nil {
		q.Where = b.buildExpr(wc.(*cqlparser.WhereClauseContext).Expression())
	}

	// Return
	if rc := qc.ReturnClause(); rc != nil {
		rcc := rc.(*cqlparser.ReturnClauseContext)
		text := strings.ToLower(rcc.GetText())
		q.Return = &ast.ReturnClause{
			Distinct:   strings.Contains(text, "distinct"),
			Expression: b.buildExpr(rcc.Expression()),
		}
	}

	// Aggregate
	if ac := qc.AggregateClause(); ac != nil {
		acc := ac.(*cqlparser.AggregateClauseContext)
		text := strings.ToLower(acc.GetText())
		agg := &ast.AggregateClause{
			Distinct:   strings.Contains(text, "distinct"),
			Identifier: unquoteIdentifier(acc.Identifier().GetText()),
			Expression: b.buildExpr(acc.Expression()),
		}
		if sc := acc.StartingClause(); sc != nil {
			if scExpr := sc.(*cqlparser.StartingClauseContext); scExpr != nil {
				if e := scExpr.Expression(); e != nil {
					agg.Starting = b.buildExpr(e)
				}
			}
		}
		q.Aggregate = agg
	}

	// Sort
	if sc := qc.SortClause(); sc != nil {
		q.Sort = b.buildSortClause(sc)
	}

	return q
}

func (b *astBuilder) buildAliasedQuerySource(ctx cqlparser.IAliasedQuerySourceContext) *ast.AliasedQuerySource {
	if ctx == nil {
		return nil
	}
	aqsc := ctx.(*cqlparser.AliasedQuerySourceContext)
	aqs := &ast.AliasedQuerySource{}
	if alias := aqsc.Alias(); alias != nil {
		aqs.Alias = unquoteIdentifier(alias.(*cqlparser.AliasContext).Identifier().GetText())
	}
	if qs := aqsc.QuerySource(); qs != nil {
		qsc := qs.(*cqlparser.QuerySourceContext)
		if ret := qsc.Retrieve(); ret != nil {
			aqs.Expression = b.buildRetrieve(ret)
		} else if qie := qsc.QualifiedIdentifierExpression(); qie != nil {
			aqs.Expression = &ast.IdentifierRef{Name: qie.GetText()}
		} else if expr := qsc.Expression(); expr != nil {
			aqs.Expression = b.buildExpr(expr)
		}
	}
	return aqs
}

func (b *astBuilder) buildSortClause(ctx cqlparser.ISortClauseContext) *ast.SortClause {
	if ctx == nil {
		return nil
	}
	sc := ctx.(*cqlparser.SortClauseContext)
	sortClause := &ast.SortClause{}
	for _, item := range sc.AllSortByItem() {
		sbi := item.(*cqlparser.SortByItemContext)
		dir := ast.SortAsc
		if sd := sbi.SortDirection(); sd != nil && strings.ToLower(sd.GetText()) == "desc" {
			dir = ast.SortDesc
		}
		expr := b.buildExprTerm(sbi.ExpressionTerm())
		sortClause.Items = append(sortClause.Items, &ast.SortByItem{
			Expression: expr,
			Direction:  dir,
		})
	}
	return sortClause
}

// -----------------------------------------------------------------------
// Operator name helpers
// -----------------------------------------------------------------------

func (b *astBuilder) setOp(ctx *cqlparser.InFixSetExpressionContext) string {
	text := ctx.GetText()
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "intersect"):
		return "Intersect"
	case strings.Contains(lower, "except"):
		return "Except"
	case strings.Contains(lower, "union") || strings.Contains(lower, "|"):
		return "Union"
	}
	return "Union"
}

func (b *astBuilder) inequalityOp(ctx *cqlparser.InequalityExpressionContext) string {
	text := ctx.GetText()
	// walk children to find the operator token
	for i := 0; i < ctx.GetChildCount(); i++ {
		child := ctx.GetChild(i)
		if t, ok := child.(antlr.TerminalNode); ok {
			switch t.GetText() {
			case "<=":
				return "LessOrEqual"
			case "<":
				return "Less"
			case ">":
				return "Greater"
			case ">=":
				return "GreaterOrEqual"
			}
		}
	}
	// fallback: detect from text
	if strings.Contains(text, "<=") {
		return "LessOrEqual"
	}
	if strings.Contains(text, ">=") {
		return "GreaterOrEqual"
	}
	if strings.Contains(text, "<") {
		return "Less"
	}
	return "Greater"
}

func (b *astBuilder) equalityOp(ctx *cqlparser.EqualityExpressionContext) string {
	for i := 0; i < ctx.GetChildCount(); i++ {
		if t, ok := ctx.GetChild(i).(antlr.TerminalNode); ok {
			switch t.GetText() {
			case "=":
				return "Equal"
			case "!=":
				return "NotEqual"
			case "~":
				return "Equivalent"
			case "!~":
				return "NotEquivalent"
			}
		}
	}
	return "Equal"
}

func (b *astBuilder) multiplicationOp(ctx *cqlparser.MultiplicationExpressionTermContext) string {
	for i := 0; i < ctx.GetChildCount(); i++ {
		if t, ok := ctx.GetChild(i).(antlr.TerminalNode); ok {
			switch t.GetText() {
			case "*":
				return "Multiply"
			case "/":
				return "Divide"
			case "div":
				return "TruncatedDivide"
			case "mod":
				return "Modulo"
			}
		}
	}
	return "Multiply"
}

func (b *astBuilder) additionOp(ctx *cqlparser.AdditionExpressionTermContext) string {
	for i := 0; i < ctx.GetChildCount(); i++ {
		if t, ok := ctx.GetChild(i).(antlr.TerminalNode); ok {
			switch t.GetText() {
			case "+":
				return "Add"
			case "-":
				return "Subtract"
			case "&":
				return "Concatenate"
			}
		}
	}
	return "Add"
}

// timingOp maps an intervalOperatorPhrase to an ELM operator name + precision.
func (b *astBuilder) timingOp(ctx cqlparser.IIntervalOperatorPhraseContext) (op, precision string) {
	if ctx == nil {
		return "SameAs", ""
	}
	text := strings.ToLower(ctx.GetText())

	switch c := ctx.(type) {
	case *cqlparser.BeforeOrAfterIntervalOperatorPhraseContext:
		prec := ""
		if dp := c.DateTimePrecisionSpecifier(); dp != nil {
			prec = b.singularizePrecision(strings.TrimSuffix(dp.GetText(), "of"))
		}
		if strings.Contains(text, "before") {
			return "Before", prec
		}
		return "After", prec

	case *cqlparser.ConcurrentWithIntervalOperatorPhraseContext:
		prec := ""
		if dp := c.DateTimePrecision(); dp != nil {
			prec = b.singularizePrecision(dp.GetText())
		}
		return "SameAs", prec

	case *cqlparser.IncludesIntervalOperatorPhraseContext:
		properly := strings.Contains(text, "properly")
		prec := ""
		if dp := c.DateTimePrecisionSpecifier(); dp != nil {
			prec = b.singularizePrecision(strings.TrimSuffix(dp.GetText(), "of"))
		}
		if properly {
			return "ProperIncludes", prec
		}
		return "Includes", prec

	case *cqlparser.IncludedInIntervalOperatorPhraseContext:
		properly := strings.Contains(text, "properly")
		prec := ""
		if dp := c.DateTimePrecisionSpecifier(); dp != nil {
			prec = b.singularizePrecision(strings.TrimSuffix(dp.GetText(), "of"))
		}
		if properly {
			return "ProperIn", prec
		}
		return "In", prec

	case *cqlparser.MeetsIntervalOperatorPhraseContext:
		prec := ""
		if dp := c.DateTimePrecisionSpecifier(); dp != nil {
			prec = b.singularizePrecision(strings.TrimSuffix(dp.GetText(), "of"))
		}
		if strings.Contains(text, "before") {
			return "MeetsBefore", prec
		}
		if strings.Contains(text, "after") {
			return "MeetsAfter", prec
		}
		return "Meets", prec

	case *cqlparser.OverlapsIntervalOperatorPhraseContext:
		prec := ""
		if dp := c.DateTimePrecisionSpecifier(); dp != nil {
			prec = b.singularizePrecision(strings.TrimSuffix(dp.GetText(), "of"))
		}
		if strings.Contains(text, "before") {
			return "OverlapsBefore", prec
		}
		if strings.Contains(text, "after") {
			return "OverlapsAfter", prec
		}
		return "Overlaps", prec

	case *cqlparser.StartsIntervalOperatorPhraseContext:
		prec := ""
		if dp := c.DateTimePrecisionSpecifier(); dp != nil {
			prec = b.singularizePrecision(strings.TrimSuffix(dp.GetText(), "of"))
		}
		return "Starts", prec

	case *cqlparser.EndsIntervalOperatorPhraseContext:
		prec := ""
		if dp := c.DateTimePrecisionSpecifier(); dp != nil {
			prec = b.singularizePrecision(strings.TrimSuffix(dp.GetText(), "of"))
		}
		return "Ends", prec

	case *cqlparser.WithinIntervalOperatorPhraseContext:
		return "In", ""
	}

	return "SameAs", ""
}

// singularizePrecision converts plural to singular precision name (years→Year).
func (b *astBuilder) singularizePrecision(s string) string {
	lower := strings.ToLower(strings.TrimSpace(s))
	switch lower {
	case "years", "year":
		return "Year"
	case "months", "month":
		return "Month"
	case "weeks", "week":
		return "Week"
	case "days", "day":
		return "Day"
	case "hours", "hour":
		return "Hour"
	case "minutes", "minute":
		return "Minute"
	case "seconds", "second":
		return "Second"
	case "milliseconds", "millisecond":
		return "Millisecond"
	case "date":
		return "Date"
	case "time":
		return "Time"
	case "timezoneoffset":
		return "TimezoneOffset"
	}
	if len(s) > 0 {
		return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
	}
	return s
}

// parseInt converts a decimal-string integer, returning 0 on parse failure.
func parseInt(s string) int64 {
	if len(s) == 0 {
		return 0
	}
	neg := false
	if s[0] == '-' {
		neg = true
		s = s[1:]
	}
	var v int64
	for _, c := range s {
		if c < '0' || c > '9' {
			break
		}
		v = v*10 + int64(c-'0')
	}
	if neg {
		return -v
	}
	return v
}
