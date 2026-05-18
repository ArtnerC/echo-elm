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

	if len(lib.Statements) > 0 {
		stmts := &elm.StatementDefs{}
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
// Most complex expression types return UnimplementedNode until Phase 4.
func translateExpr(expr ast.Expr) elm.Expression {
	if expr == nil {
		return &elm.NullNode{}
	}
	switch v := expr.(type) {
	case *ast.BooleanLiteral:
		val := "false"
		if v.Value {
			val = "true"
		}
		return &elm.LiteralNode{
			ValueType: typesystem.TypeBoolean,
			Value:     val,
		}
	case *ast.IntegerLiteral:
		return &elm.LiteralNode{
			ValueType: typesystem.TypeInteger,
			Value:     fmt.Sprintf("%d", v.Value),
		}
	case *ast.LongLiteral:
		return &elm.LiteralNode{
			ValueType: typesystem.TypeLong,
			Value:     fmt.Sprintf("%d", v.Value),
		}
	case *ast.DecimalLiteral:
		return &elm.LiteralNode{
			ValueType: typesystem.TypeDecimal,
			Value:     v.Value,
		}
	case *ast.StringLiteral:
		return &elm.LiteralNode{
			ValueType: typesystem.TypeString,
			Value:     v.Value,
		}
	case *ast.NullLiteral:
		return &elm.NullNode{}
	case *ast.IdentifierRef:
		return &elm.ExpressionRefNode{Name: v.Name}
	case *ast.QualifiedRef:
		return &elm.ExpressionRefNode{Name: v.Name, LibraryName: v.LibraryName}
	default:
		return &elm.UnimplementedNode{TypeName: fmt.Sprintf("%T", expr)}
	}
}
