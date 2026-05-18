package translator

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/artnerc/echo-elm/internal/ast"
	"github.com/artnerc/echo-elm/internal/elm"
	"github.com/artnerc/echo-elm/internal/typesystem"
)

// Version is the translator version string embedded in output.
const Version = "0.0.0-dev"

// Options controls translation behavior.
type Options struct {
	EnableAnnotations    bool
	EnableLocators       bool
	DisableListDemotion  bool
	DisableListPromotion bool
	ValidateUnits        bool
	CompatibilityLevel   string
	SignatureLevel       string
	ErrorLevel           string
	TranslatorVersion    string
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

// Translator is the core CQL→ELM translation engine.
type Translator struct {
	opts    Options
	counter int
	source  string
}

// New creates a new Translator with the given options.
func New(opts Options) *Translator {
	return &Translator{opts: opts}
}

func (t *Translator) nextID() string {
	t.counter++
	return fmt.Sprintf("%d", t.counter)
}

func locatorStr(loc ast.Interval) string {
	if loc.Start.Line == 0 {
		return ""
	}
	return fmt.Sprintf("%d:%d-%d:%d",
		loc.Start.Line, loc.Start.Column,
		loc.Stop.Line, loc.Stop.Column)
}

func accessLevelStr(level ast.AccessLevel) string {
	if level == ast.Private {
		return "Private"
	}
	return "Public"
}

// optionsString builds the translator options string for CqlToElmInfo.
func (t *Translator) optionsString() string {
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
	if t.opts.ValidateUnits {
		parts = append(parts, "ValidateUnits")
	}
	return strings.Join(parts, ",")
}

// Translate converts an AST library to an ELM library.
func (t *Translator) Translate(lib *ast.Library, sourceName string) *Result {
	t.source = sourceName
	t.counter = 0

	result := &Result{}
	out := &elm.Library{
		SchemaIdentifier: &elm.VersionedIdentifier{
			ID:      "urn:hl7-org:elm",
			Version: "r1",
		},
	}

	// Library identifier
	if lib.Name != nil {
		out.Identifier = &elm.VersionedIdentifier{
			ID:      lib.Name.Name,
			Version: lib.Name.Version,
		}
	}

	// CqlToElmInfo annotation
	info := &elm.CqlToElmInfo{
		TranslatorVersion: t.opts.TranslatorVersion,
		TranslatorOptions: t.optionsString(),
		SignatureLevel:    t.opts.SignatureLevel,
	}
	infoJSON, err := json.Marshal(info)
	if err == nil {
		out.Annotation = append(out.Annotation, json.RawMessage(infoJSON))
	}

	// Usings: always add implicit System first
	usings := &elm.UsingDefs{}
	usings.Def = append(usings.Def, &elm.UsingDef{
		LocalIdentifier: "System",
		URI:             typesystem.SystemURI,
	})
	for _, u := range lib.Usings {
		ud := &elm.UsingDef{
			LocalID:         t.nextID(),
			LocalIdentifier: u.LocalName,
			URI:             t.modelURI(u.ModelName),
			Version:         u.Version,
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
			id := &elm.IncludeDef{
				LocalID:         t.nextID(),
				LocalIdentifier: inc.LocalName,
				Path:            inc.Path,
				Version:         inc.Version,
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
				LocalID:     t.nextID(),
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
				LocalID:     t.nextID(),
				Name:        vs.Name,
				ID:          vs.ID,
				Version:     vs.Version,
				AccessLevel: accessLevelStr(vs.AccessLevel),
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
			cd := &elm.CodeDef{
				LocalID:     t.nextID(),
				Name:        c.Name,
				ID:          c.Code,
				Display:     c.Display,
				AccessLevel: accessLevelStr(c.AccessLevel),
				CodeSystem: &elm.CodeSystemRef{
					Name: c.SystemName,
				},
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
				LocalID:     t.nextID(),
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
				LocalID:     t.nextID(),
				Name:        p.Name,
				AccessLevel: accessLevelStr(p.AccessLevel),
			}
			if t.opts.EnableLocators {
				pd.Locator = locatorStr(p.Loc())
			}
			if p.ParameterType != nil {
				pd.ParameterTypeSpecifier = translateTypeSpecifier(*p.ParameterType)
			}
			params.Def = append(params.Def, pd)
		}
		out.Parameters = params
	}

	// Statements
	contextName := ""
	if lib.Context != nil {
		contextName = lib.Context.Name
	}

	// Contexts section
	if contextName != "" {
		out.Contexts = &elm.ContextDefs{
			Def: []*elm.ContextDef{{Name: contextName}},
		}
	}

	// Statements — prepend implicit context accessor if a context is declared.
	var stmtDefs []*elm.StatementDef
	if contextName != "" {
		stmtDefs = append(stmtDefs, t.buildContextAccessor(contextName, lib))
	}

	if len(lib.Statements) > 0 || len(stmtDefs) > 0 {
		stmts := &elm.StatementDefs{Def: stmtDefs}
		for _, s := range lib.Statements {
			sd := &elm.StatementDef{
				LocalID:     t.nextID(),
				Name:        s.Name,
				Context:     contextName,
				AccessLevel: accessLevelStr(s.AccessLevel),
				IsFunction:  s.IsFunction,
				IsFluent:    s.IsFluent,
			}
			if t.opts.EnableLocators {
				sd.Locator = locatorStr(s.Loc())
			}
			if s.ReturnType != nil {
				sd.ResultTypeSpecifier = translateTypeSpecifier(*s.ReturnType)
			}
			for _, op := range s.Operands {
				opd := &elm.OperandDef{Name: op.Name}
				if op.Type != nil {
					opd.OperandTypeSpecifier = translateTypeSpecifier(*op.Type)
				}
				sd.Operand = append(sd.Operand, opd)
			}
			if s.Expression != nil {
				sd.Expression = translateExpr(s.Expression)
			} else {
				sd.Expression = &elm.NullNode{}
			}
			stmts.Def = append(stmts.Def, sd)
		}
		out.Statements = stmts
	}

	result.Library = out
	return result
}

func (t *Translator) modelURI(name string) string {
	if uri, ok := typesystem.ModelURIByName[name]; ok {
		return uri
	}
	return name
}

// buildContextAccessor creates the implicit singleton-from-retrieve statement for the declared context.
// For example, `context Patient` generates a Patient statement that retrieves the singleton Patient.
func (t *Translator) buildContextAccessor(contextName string, lib *ast.Library) *elm.StatementDef {
	// Determine the data type from the declared model.
	modelURI := "http://hl7.org/fhir" // default to FHIR
	for _, u := range lib.Usings {
		if uri, ok := typesystem.ModelURIByName[u.ModelName]; ok {
			modelURI = uri
			break
		}
	}
	dataType := "{" + modelURI + "}" + contextName

	// FHIR template ID follows the StructureDefinition pattern.
	templateID := ""
	if modelURI == "http://hl7.org/fhir" {
		templateID = "http://hl7.org/fhir/StructureDefinition/" + contextName
	}

	retrieve := &elm.RetrieveNode{
		DataType:   dataType,
		TemplateID: templateID,
	}

	return &elm.StatementDef{
		Name:       contextName,
		Context:    contextName,
		Expression: &elm.SingletonFromNode{Operand: retrieve},
	}
}

// translateTypeSpecifier converts an AST TypeSpecifier to an ELM TypeSpecifier.
func translateTypeSpecifier(ts ast.TypeSpecifier) elm.TypeSpecifier {
	if ts == nil {
		return nil
	}
	switch v := ts.(type) {
	case *ast.NamedTypeSpecifier:
		name := v.Name
		if v.Qualifier != "" {
			name = v.Qualifier + "." + v.Name
		}
		name = resolveTypeName(name)
		return &elm.NamedTypeSpecifier{Name: name}
	case *ast.IntervalTypeSpecifier:
		return &elm.IntervalTypeSpecifier{
			PointType: translateTypeSpecifier(v.PointType),
		}
	case *ast.ListTypeSpecifier:
		return &elm.ListTypeSpecifier{
			ElementType: translateTypeSpecifier(v.ElementType),
		}
	case *ast.TupleTypeSpecifier:
		tts := &elm.TupleTypeSpecifier{}
		for _, elem := range v.Elements {
			tts.Element = append(tts.Element, &elm.TupleElementDefinition{
				Name: elem.Name,
				Type: translateTypeSpecifier(elem.Type),
			})
		}
		return tts
	case *ast.ChoiceTypeSpecifier:
		cts := &elm.ChoiceTypeSpecifier{}
		for _, ct := range v.Types {
			cts.Choice = append(cts.Choice, translateTypeSpecifier(ct))
		}
		return cts
	default:
		return &elm.NamedTypeSpecifier{Name: typesystem.TypeAny}
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

// translateExpr converts an AST expression to an ELM expression.
func translateExpr(expr ast.Expr) elm.Expression {
	if expr == nil {
		return &elm.NullNode{}
	}
	switch v := expr.(type) {
	// ---- Literals ----
	case *ast.BooleanLiteral:
		val := "false"
		if v.Value {
			val = "true"
		}
		return &elm.LiteralNode{ValueType: typesystem.TypeBoolean, Value: val}
	case *ast.IntegerLiteral:
		return &elm.LiteralNode{ValueType: typesystem.TypeInteger, Value: fmt.Sprintf("%d", v.Value)}
	case *ast.LongLiteral:
		return &elm.LiteralNode{ValueType: typesystem.TypeLong, Value: fmt.Sprintf("%d", v.Value)}
	case *ast.DecimalLiteral:
		return &elm.LiteralNode{ValueType: typesystem.TypeDecimal, Value: v.Value}
	case *ast.StringLiteral:
		return &elm.LiteralNode{ValueType: typesystem.TypeString, Value: v.Value}
	case *ast.NullLiteral:
		return &elm.NullNode{}
	case *ast.DateLiteral:
		return &elm.LiteralNode{ValueType: typesystem.TypeDate, Value: v.Value}
	case *ast.DateTimeLiteral:
		return &elm.LiteralNode{ValueType: typesystem.TypeDateTime, Value: v.Value}
	case *ast.TimeLiteral:
		return &elm.LiteralNode{ValueType: typesystem.TypeTime, Value: v.Value}
	case *ast.QuantityLiteral:
		return &elm.QuantityNode{Value: v.Value, Unit: v.Unit}
	case *ast.RatioLiteral:
		return &elm.RatioNode{
			Numerator:   &elm.QuantityNode{Value: v.Numerator.Value, Unit: v.Numerator.Unit},
			Denominator: &elm.QuantityNode{Value: v.Denominator.Value, Unit: v.Denominator.Unit},
		}

	// ---- References ----
	case *ast.IdentifierRef:
		return &elm.ExpressionRefNode{Name: v.Name}
	case *ast.QualifiedRef:
		return &elm.ExpressionRefNode{Name: v.Name, LibraryName: v.LibraryName}
	case *ast.AliasRef:
		return &elm.AliasRefNode{Name: v.Name}
	case *ast.LetRef:
		return &elm.LetRefNode{Name: v.Name}
	case *ast.ThisExpr:
		return &elm.QueryThisRefNode{}
	case *ast.IndexExpr:
		return &elm.OperatorExpressionNode{Operator: "QueryIndexRef"}
	case *ast.TotalExpr:
		return &elm.OperatorExpressionNode{Operator: "Total"}
	case *ast.ExternalConstantExpr:
		return &elm.ExternalConstantNode{Name: v.Name}
	case *ast.FunctionRef:
		var operands []elm.Expression
		for _, o := range v.Operands {
			operands = append(operands, translateExpr(o))
		}
		return &elm.FunctionRefNode{
			Name:        v.Name,
			LibraryName: v.LibraryName,
			Operand:     operands,
		}
	case *ast.PropertyExpr:
		return &elm.PropertyNode{
			Path:   v.Path,
			Source: translateExpr(v.Source),
		}
	case *ast.IndexedAccessExpr:
		return &elm.OperatorExpressionNode{
			Operator: "Indexer",
			Operand:  []elm.Expression{translateExpr(v.Source), translateExpr(v.Index)},
		}

	// ---- Unary / binary ----
	case *ast.UnaryExpr:
		return &elm.OperatorExpressionNode{
			Operator: v.Op,
			Operand:  []elm.Expression{translateExpr(v.Operand)},
		}
	case *ast.BinaryExpr:
		return translateBinaryExpr(v)
	case *ast.TernaryExpr:
		return &elm.IfNode{
			Condition: translateExpr(v.Condition),
			Then:      translateExpr(v.ThenExpr),
			Else:      translateExpr(v.ElseExpr),
		}
	case *ast.CaseExpr:
		cn := &elm.CaseNode{Else: translateExpr(v.Else)}
		if v.Comparand != nil {
			cn.Comparand = translateExpr(v.Comparand)
		}
		for _, item := range v.Items {
			cn.CaseItem = append(cn.CaseItem, &elm.CaseItem{
				When: translateExpr(item.When),
				Then: translateExpr(item.Then),
			})
		}
		return cn

	// ---- Type operators ----
	case *ast.TypeIsExpr:
		return translateTypeIs(v)
	case *ast.TypeAsExpr:
		ts := translateTypeSpecifier(v.TypeSpec)
		return &elm.AsNode{
			Operand:         []elm.Expression{translateExpr(v.Operand)},
			AsTypeSpecifier: ts,
			Strict:          v.Strict,
		}
	case *ast.ConvertExpr:
		ts := translateTypeSpecifier(v.TypeSpec)
		return &elm.ConvertNode{
			Operand:         []elm.Expression{translateExpr(v.Operand)},
			ToTypeSpecifier: ts,
		}

	// ---- Timing / interval ----
	case *ast.TimingExpr:
		return translateTimingExpr(v)
	case *ast.BetweenExpr:
		if v.Properly {
			return &elm.OperatorExpressionNode{
				Operator: "ProperBetween",
				Operand: []elm.Expression{
					translateExpr(v.Operand),
					translateExpr(v.Low),
					translateExpr(v.High),
				},
			}
		}
		return &elm.OperatorExpressionNode{
			Operator: "Between",
			Operand: []elm.Expression{
				translateExpr(v.Operand),
				translateExpr(v.Low),
				translateExpr(v.High),
			},
		}
	case *ast.DurationBetweenExpr:
		return &elm.PrecisionOperatorNode{
			Operator:  "DurationBetween",
			Precision: v.Precision,
			Operand:   []elm.Expression{translateExpr(v.Low), translateExpr(v.High)},
		}
	case *ast.DifferenceBetweenExpr:
		return &elm.PrecisionOperatorNode{
			Operator:  "DifferenceBetween",
			Precision: v.Precision,
			Operand:   []elm.Expression{translateExpr(v.Low), translateExpr(v.High)},
		}
	case *ast.IntervalExpr:
		return &elm.IntervalNode{
			Low:        translateExpr(v.Low),
			High:       translateExpr(v.High),
			LowClosed:  v.LowClosed,
			HighClosed: v.HighClosed,
		}
	case *ast.TimeBoundaryExpr:
		return &elm.OperatorExpressionNode{
			Operator: titleCase(v.Boundary),
			Operand:  []elm.Expression{translateExpr(v.Source)},
		}
	case *ast.DateTimeComponentExpr:
		return &elm.PrecisionOperatorNode{
			Operator:  "DateTimeComponentFrom",
			Precision: v.Precision,
			Operand:   []elm.Expression{translateExpr(v.Source)},
		}
	case *ast.DurationExpr:
		return &elm.PrecisionOperatorNode{
			Operator:  "DurationBetween",
			Precision: v.Precision,
			Operand: []elm.Expression{
				&elm.OperatorExpressionNode{
					Operator: "Start",
					Operand:  []elm.Expression{translateExpr(v.Source)},
				},
				&elm.OperatorExpressionNode{
					Operator: "End",
					Operand:  []elm.Expression{translateExpr(v.Source)},
				},
			},
		}
	case *ast.DifferenceExpr:
		return &elm.PrecisionOperatorNode{
			Operator:  "DifferenceBetween",
			Precision: v.Precision,
			Operand: []elm.Expression{
				&elm.OperatorExpressionNode{
					Operator: "Start",
					Operand:  []elm.Expression{translateExpr(v.Source)},
				},
				&elm.OperatorExpressionNode{
					Operator: "End",
					Operand:  []elm.Expression{translateExpr(v.Source)},
				},
			},
		}
	case *ast.WidthExpr:
		return &elm.OperatorExpressionNode{
			Operator: "Width",
			Operand:  []elm.Expression{translateExpr(v.Source)},
		}
	case *ast.SuccessorExpr:
		return &elm.OperatorExpressionNode{
			Operator: "Successor",
			Operand:  []elm.Expression{translateExpr(v.Source)},
		}
	case *ast.PredecessorExpr:
		return &elm.OperatorExpressionNode{
			Operator: "Predecessor",
			Operand:  []elm.Expression{translateExpr(v.Source)},
		}
	case *ast.SingletonFromExpr:
		return &elm.SingletonFromNode{Operand: translateExpr(v.Source)}
	case *ast.PointFromExpr:
		return &elm.OperatorExpressionNode{
			Operator: "PointFrom",
			Operand:  []elm.Expression{translateExpr(v.Source)},
		}
	case *ast.TypeExtentExpr:
		op := "MinValue"
		if v.Extent == "maximum" {
			op = "MaxValue"
		}
		ts := translateTypeSpecifier(v.TypeSpec)
		return &elm.IsNode{
			IsTypeSpecifier: ts,
			Operand:         []elm.Expression{&elm.OperatorExpressionNode{Operator: op}},
		}

	// ---- Selectors ----
	case *ast.ListExpr:
		ln := &elm.ListNode{}
		for _, e := range v.Elements {
			ln.Element = append(ln.Element, translateExpr(e))
		}
		return ln
	case *ast.TupleExpr:
		tn := &elm.TupleNode{}
		for _, e := range v.Elements {
			tn.Element = append(tn.Element, &elm.TupleElementNode{
				Name:  e.Name,
				Value: translateExpr(e.Expression),
			})
		}
		return tn
	case *ast.InstanceExpr:
		in := &elm.InstanceNode{}
		if v.TypeSpec != nil {
			if nt, ok := v.TypeSpec.(*ast.NamedTypeSpecifier); ok {
				in.ClassType = nt.Name
			}
		}
		for _, e := range v.Elements {
			in.Element = append(in.Element, &elm.TupleElementNode{
				Name:  e.Name,
				Value: translateExpr(e.Expression),
			})
		}
		return in
	case *ast.CodeExpr:
		cn := &elm.CodeNode{Code: v.Code, Display: v.Display}
		if v.System != "" {
			cn.System = &elm.CodeSystemRef{Name: v.System}
		}
		return cn
	case *ast.ConceptExpr:
		cn := &elm.ConceptNode{Display: v.Display}
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
		return &elm.OperatorExpressionNode{
			Operator: v.Op,
			Operand:  []elm.Expression{translateExpr(v.Operand)},
		}
	case *ast.SetAggregateExpr:
		operands := []elm.Expression{translateExpr(v.Operand)}
		if v.PerClause != nil {
			operands = append(operands, translateExpr(v.PerClause))
		}
		return &elm.OperatorExpressionNode{Operator: v.Op, Operand: operands}

	// ---- Retrieve ----
	case *ast.RetrieveExpr:
		return translateRetrieve(v)

	// ---- Query ----
	case *ast.QueryExpression:
		return translateQuery(v)

	default:
		return &elm.UnimplementedNode{TypeName: fmt.Sprintf("%T", expr)}
	}
}

func translateBinaryExpr(v *ast.BinaryExpr) elm.Expression {
	operands := []elm.Expression{translateExpr(v.Left), translateExpr(v.Right)}
	if v.Precision != "" {
		return &elm.PrecisionOperatorNode{
			Operator:  v.Op,
			Precision: v.Precision,
			Operand:   operands,
		}
	}
	return &elm.OperatorExpressionNode{Operator: v.Op, Operand: operands}
}

func translateTypeIs(v *ast.TypeIsExpr) elm.Expression {
	operand := translateExpr(v.Operand)
	if v.IsNull {
		isNull := &elm.OperatorExpressionNode{
			Operator: "IsNull",
			Operand:  []elm.Expression{operand},
		}
		if v.Negated {
			return &elm.OperatorExpressionNode{
				Operator: "Not",
				Operand:  []elm.Expression{isNull},
			}
		}
		return isNull
	}
	if v.IsTrue {
		op := "IsTrue"
		node := &elm.OperatorExpressionNode{Operator: op, Operand: []elm.Expression{operand}}
		if v.Negated {
			return &elm.OperatorExpressionNode{Operator: "Not", Operand: []elm.Expression{node}}
		}
		return node
	}
	if v.IsFalse {
		op := "IsFalse"
		node := &elm.OperatorExpressionNode{Operator: op, Operand: []elm.Expression{operand}}
		if v.Negated {
			return &elm.OperatorExpressionNode{Operator: "Not", Operand: []elm.Expression{node}}
		}
		return node
	}
	ts := translateTypeSpecifier(v.TypeSpec)
	return &elm.IsNode{
		Operand:         []elm.Expression{operand},
		IsTypeSpecifier: ts,
	}
}

func translateTimingExpr(v *ast.TimingExpr) elm.Expression {
	operands := []elm.Expression{translateExpr(v.Left), translateExpr(v.Right)}
	if v.Precision != "" {
		return &elm.PrecisionOperatorNode{
			Operator:  v.Op,
			Precision: v.Precision,
			Operand:   operands,
		}
	}
	return &elm.OperatorExpressionNode{Operator: v.Op, Operand: operands}
}

func translateRetrieve(v *ast.RetrieveExpr) elm.Expression {
	r := &elm.RetrieveNode{DataType: v.DataType}
	if v.Codes != nil {
		r.Codes = translateExpr(v.Codes)
		r.CodeProperty = v.CodeProperty
	}
	return r
}

func translateQuery(q *ast.QueryExpression) elm.Expression {
	qn := &elm.QueryNode{}
	for _, src := range q.Sources {
		qn.Source = append(qn.Source, &elm.AliasedQuerySourceELM{
			Alias:      src.Alias,
			Expression: translateExpr(src.Expression),
		})
	}
	for _, let := range q.Let {
		qn.Let = append(qn.Let, &elm.LetClauseELM{
			Identifier: let.Identifier,
			Expression: translateExpr(let.Expression),
		})
	}
	for _, rel := range q.Relationship {
		kind := "With"
		if rel.Kind == "without" {
			kind = "Without"
		}
		r := &elm.RelationshipClauseELM{
			Kind:      kind,
			Alias:     rel.Source.Alias,
			Expression: translateExpr(rel.Source.Expression),
			SuchThat:  translateExpr(rel.SuchThat),
		}
		qn.Relationship = append(qn.Relationship, r)
	}
	if q.Where != nil {
		qn.Where = translateExpr(q.Where)
	}
	if q.Return != nil {
		qn.Return = &elm.ReturnClauseELM{
			Distinct:   q.Return.Distinct,
			Expression: translateExpr(q.Return.Expression),
		}
	}
	if q.Aggregate != nil {
		qn.Aggregate = &elm.AggregateClauseELM{
			Distinct:   q.Aggregate.Distinct,
			Identifier: q.Aggregate.Identifier,
			Expression: translateExpr(q.Aggregate.Expression),
			Starting:   translateExpr(q.Aggregate.Starting),
		}
	}
	if q.Sort != nil {
		sortClause := &elm.SortClauseELM{}
		for _, item := range q.Sort.Items {
			dir := "asc"
			if item.Direction == ast.SortDesc {
				dir = "desc"
			}
			sortClause.By = append(sortClause.By, &elm.SortByItemELM{
				Direction:  dir,
				Expression: translateExpr(item.Expression),
			})
		}
		qn.Sort = sortClause
	}
	return qn
}

func titleCase(s string) string {
	if len(s) == 0 {
		return s
	}
	lower := strings.ToLower(s)
	return strings.ToUpper(lower[:1]) + lower[1:]
}
