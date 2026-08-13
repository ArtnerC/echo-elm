package elm

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// marshalWithType injects a "type" key as the first field in a JSON object.
func marshalWithType(typeName string, v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	if len(b) < 2 || b[0] != '{' {
		return b, nil
	}
	typeJSON, _ := json.Marshal(typeName)
	var buf bytes.Buffer
	buf.WriteString(`{"type":`)
	buf.Write(typeJSON)
	if len(b) > 2 {
		buf.WriteByte(',')
		buf.Write(b[1:]) // skip opening '{'
	} else {
		buf.WriteByte('}')
	}
	return buf.Bytes(), nil
}

// -----------------------------------------------------------------------
// Top-level envelope
// -----------------------------------------------------------------------

// LibraryEnvelope is the top-level JSON wrapper: {"library": {...}}.
type LibraryEnvelope struct {
	Library *Library `json:"library"`
}

// -----------------------------------------------------------------------
// Library
// -----------------------------------------------------------------------

// Library is the root ELM Library node.
type Library struct {
	LocalID          string               `json:"localId,omitempty"`
	Locator          string               `json:"locator,omitempty"`
	SchemaIdentifier *VersionedIdentifier `json:"schemaIdentifier"`
	Identifier       VersionedIdentifier  `json:"identifier"`
	Annotation       []json.RawMessage    `json:"annotation,omitempty"`
	Usings           *UsingDefs           `json:"usings,omitempty"`
	Includes         *IncludeDefs         `json:"includes,omitempty"`
	Parameters       *ParameterDefs       `json:"parameters,omitempty"`
	CodeSystems      *CodeSystemDefs      `json:"codeSystems,omitempty"`
	ValueSets        *ValueSetDefs        `json:"valueSets,omitempty"`
	Codes            *CodeDefs            `json:"codes,omitempty"`
	Concepts         *ConceptDefs         `json:"concepts,omitempty"`
	Contexts         *ContextDefs         `json:"contexts,omitempty"`
	Statements       *StatementDefs       `json:"statements,omitempty"`
}

// VersionedIdentifier holds an id and optional version string.
type VersionedIdentifier struct {
	ID      string `json:"id,omitempty"`
	System  string `json:"system,omitempty"`
	Version string `json:"version,omitempty"`
}

// -----------------------------------------------------------------------
// Section wrappers
// -----------------------------------------------------------------------

// UsingDefs wraps the list of using definitions.
type UsingDefs struct {
	Def []*UsingDef `json:"def"`
}

// IncludeDefs wraps the list of include definitions.
type IncludeDefs struct {
	Def []*IncludeDef `json:"def"`
}

// ParameterDefs wraps the list of parameter definitions.
type ParameterDefs struct {
	Def []*ParameterDef `json:"def"`
}

// CodeSystemDefs wraps the list of code system definitions.
type CodeSystemDefs struct {
	Def []*CodeSystemDef `json:"def"`
}

// ValueSetDefs wraps the list of value set definitions.
type ValueSetDefs struct {
	Def []*ValueSetDef `json:"def"`
}

// CodeDefs wraps the list of code definitions.
type CodeDefs struct {
	Def []*CodeDef `json:"def"`
}

// ConceptDefs wraps the list of concept definitions.
type ConceptDefs struct {
	Def []*ConceptDef `json:"def"`
}

// ContextDefs wraps the list of context definitions.
type ContextDefs struct {
	Def []*ContextDef `json:"def"`
}

// StatementDefs wraps the list of statement (expression/function) definitions.
type StatementDefs struct {
	Def []*StatementDef `json:"def"`
}

// ContextDef is a named evaluation context definition.
type ContextDef struct {
	LocalID    string          `json:"localId,omitempty"`
	Locator    string          `json:"locator,omitempty"`
	Name       string          `json:"name"`
	Annotation json.RawMessage `json:"annotation,omitempty"`
}

// -----------------------------------------------------------------------
// Def types
// -----------------------------------------------------------------------

// UsingDef corresponds to a `using` declaration.
type UsingDef struct {
	LocalID         string          `json:"localId,omitempty"`
	Locator         string          `json:"locator,omitempty"`
	LocalIdentifier string          `json:"localIdentifier"`
	URI             string          `json:"uri"`
	Version         string          `json:"version,omitempty"`
	AccessLevel     string          `json:"accessLevel,omitempty"`
	Annotation      json.RawMessage `json:"annotation,omitempty"`
}

// IncludeDef corresponds to an `include` declaration.
type IncludeDef struct {
	LocalID         string          `json:"localId,omitempty"`
	Locator         string          `json:"locator,omitempty"`
	LocalIdentifier string          `json:"localIdentifier"`
	Path            string          `json:"path"`
	Version         string          `json:"version,omitempty"`
	AccessLevel     string          `json:"accessLevel,omitempty"`
	Annotation      json.RawMessage `json:"annotation,omitempty"`
}

// CodeSystemDef corresponds to a `codesystem` declaration.
type CodeSystemDef struct {
	LocalID        string          `json:"localId,omitempty"`
	Locator        string          `json:"locator,omitempty"`
	Annotation     json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName string          `json:"resultTypeName,omitempty"`
	Name           string          `json:"name"`
	ID             string          `json:"id"`
	Version        string          `json:"version,omitempty"`
	AccessLevel    string          `json:"accessLevel,omitempty"`
}

// ValueSetDef corresponds to a `valueset` declaration.
type ValueSetDef struct {
	LocalID        string          `json:"localId,omitempty"`
	Locator        string          `json:"locator,omitempty"`
	Annotation     json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName string          `json:"resultTypeName,omitempty"`
	Name           string          `json:"name"`
	ID             string          `json:"id"`
	Version        string          `json:"version,omitempty"`
	AccessLevel    string          `json:"accessLevel,omitempty"`
	CodeSystems    json.RawMessage `json:"codeSystem,omitempty"`
}

// CodeDef corresponds to a `code` declaration.
type CodeDef struct {
	LocalID        string                   `json:"localId,omitempty"`
	Locator        string                   `json:"locator,omitempty"`
	Annotation     json.RawMessage          `json:"annotation,omitempty"`
	ResultTypeName string                   `json:"resultTypeName,omitempty"`
	Name           string                   `json:"name"`
	ID             string                   `json:"id"`
	Display        string                   `json:"display,omitempty"`
	AccessLevel    string                   `json:"accessLevel,omitempty"`
	CodeSystem     *CodeSystemDefinitionRef `json:"codeSystem,omitempty"`
}

// CodeSystemDefinitionRef is the structural code-system reference used inside
// CodeDef (and ConceptDef). Unlike CodeSystemRef (an expression node), it does
// NOT carry a "type" discriminator — upstream CQF emits {"annotation":[],"name":"LOINC"}.
type CodeSystemDefinitionRef struct {
	LocalID        string          `json:"localId,omitempty"`
	Locator        string          `json:"locator,omitempty"`
	Annotation     json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName string          `json:"resultTypeName,omitempty"`
	Name           string          `json:"name"`
	LibraryName    string          `json:"libraryName,omitempty"`
}

// CodeSystemRef is a reference to a code system by name.
type CodeSystemRef struct {
	LocalID     string `json:"localId,omitempty"`
	Locator     string `json:"locator,omitempty"`
	Name        string `json:"name"`
	LibraryName string `json:"libraryName,omitempty"`
}

func (r *CodeSystemRef) MarshalJSON() ([]byte, error) {
	type alias CodeSystemRef
	return marshalWithType("CodeSystemRef", (*alias)(r))
}

// ConceptDef corresponds to a `concept` declaration.
type ConceptDef struct {
	LocalID        string          `json:"localId,omitempty"`
	Locator        string          `json:"locator,omitempty"`
	Annotation     json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName string          `json:"resultTypeName,omitempty"`
	Name           string          `json:"name"`
	Display        string          `json:"display,omitempty"`
	AccessLevel    string          `json:"accessLevel,omitempty"`
	Code           []*CodeRef      `json:"code,omitempty"`
}

// CodeRef is a reference to a code by name from within a ConceptDef. The
// declaration position already implies the node type, so — matching CQF —
// no "type" discriminator is emitted here (unlike the CodeRefNode expression).
type CodeRef struct {
	LocalID        string `json:"localId,omitempty"`
	Locator        string `json:"locator,omitempty"`
	ResultTypeName string `json:"resultTypeName,omitempty"`
	Name           string `json:"name"`
	LibraryName    string `json:"libraryName,omitempty"`
}

// ParameterDef corresponds to a `parameter` declaration.
type ParameterDef struct {
	LocalID                string          `json:"localId,omitempty"`
	Locator                string          `json:"locator,omitempty"`
	Annotation             json.RawMessage `json:"annotation,omitempty"`
	Name                   string          `json:"name"`
	AccessLevel            string          `json:"accessLevel,omitempty"`
	ResultTypeName         string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier    TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	ParameterTypeSpecifier TypeSpecifier   `json:"parameterTypeSpecifier,omitempty"`
	Default                Expression      `json:"default,omitempty"`
}

// StatementDef covers both ExpressionDef and FunctionDef.
// IsFunction controls which type name is emitted in JSON/XML.
type StatementDef struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Name                string          `json:"name"`
	Context             string          `json:"context,omitempty"`
	AccessLevel         string          `json:"accessLevel,omitempty"`
	IsFunction          bool            `json:"-"`
	IsFluent            bool            `json:"fluent,omitempty"`
	Operand             []*OperandDef   `json:"operand,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	Expression          Expression      `json:"expression,omitempty"`
}

func (s *StatementDef) MarshalJSON() ([]byte, error) {
	type alias StatementDef
	if s.IsFunction {
		// A FunctionDef always carries an operand list, empty included: CQF emits
		// "operand": [] for a zero-argument function, and the field is what
		// separates a FunctionDef from an ExpressionDef, which never has one.
		//
		// Operand stays omitempty on the struct so ExpressionDef does not grow an
		// "operand": null, and omitempty drops an empty slice whether or not it is
		// nil — so the tag has to be shadowed rather than the value replaced. The
		// outer field wins over the embedded one by depth.
		type funcDef struct {
			*alias
			Operand []*OperandDef `json:"operand"`
		}
		operands := s.Operand
		if operands == nil {
			operands = []*OperandDef{}
		}
		return marshalWithType("FunctionDef", funcDef{alias: (*alias)(s), Operand: operands})
	}
	// ExpressionDef: upstream cqframework does NOT emit the "type" discriminator.
	return json.Marshal((*alias)(s))
}

// OperandDef is a function parameter definition.
type OperandDef struct {
	LocalID              string          `json:"localId,omitempty"`
	Locator              string          `json:"locator,omitempty"`
	Annotation           json.RawMessage `json:"annotation,omitempty"`
	Name                 string          `json:"name"`
	OperandTypeSpecifier TypeSpecifier   `json:"operandTypeSpecifier,omitempty"`
}

// OperandRefNode: a reference to a function parameter (operand).
type OperandRefNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Name                string          `json:"name"`
}

func (*OperandRefNode) isExpression() {}
func (n *OperandRefNode) MarshalJSON() ([]byte, error) {
	type alias OperandRefNode
	return marshalWithType("OperandRef", (*alias)(n))
}

// -----------------------------------------------------------------------
// TypeSpecifier interface and concrete types
// -----------------------------------------------------------------------

// TypeSpecifier is the interface for all ELM type specifiers.
type TypeSpecifier interface {
	isTypeSpecifier()
	typeSpecifierJSON() ([]byte, error)
}

// NamedTypeSpecifier: a named type like Integer or FHIR.Patient.
type NamedTypeSpecifier struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Name                string          `json:"name,omitempty"`
}

func (*NamedTypeSpecifier) isTypeSpecifier()                     {}
func (n *NamedTypeSpecifier) typeSpecifierJSON() ([]byte, error) { return n.MarshalJSON() }
func (n *NamedTypeSpecifier) MarshalJSON() ([]byte, error) {
	type alias NamedTypeSpecifier
	return marshalWithType("NamedTypeSpecifier", (*alias)(n))
}

// IntervalTypeSpecifier: Interval<T>.
type IntervalTypeSpecifier struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	PointType           TypeSpecifier   `json:"pointType,omitempty"`
}

func (*IntervalTypeSpecifier) isTypeSpecifier()                     {}
func (n *IntervalTypeSpecifier) typeSpecifierJSON() ([]byte, error) { return n.MarshalJSON() }
func (n *IntervalTypeSpecifier) MarshalJSON() ([]byte, error) {
	type alias IntervalTypeSpecifier
	return marshalWithType("IntervalTypeSpecifier", (*alias)(n))
}

// ListTypeSpecifier: List<T>.
type ListTypeSpecifier struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	ElementType         TypeSpecifier   `json:"elementType,omitempty"`
}

func (*ListTypeSpecifier) isTypeSpecifier()                     {}
func (n *ListTypeSpecifier) typeSpecifierJSON() ([]byte, error) { return n.MarshalJSON() }
func (n *ListTypeSpecifier) MarshalJSON() ([]byte, error) {
	type alias ListTypeSpecifier
	return marshalWithType("ListTypeSpecifier", (*alias)(n))
}

// TupleTypeSpecifier: Tuple { name Type, ... }.
type TupleTypeSpecifier struct {
	LocalID             string                    `json:"localId,omitempty"`
	Locator             string                    `json:"locator,omitempty"`
	Annotation          json.RawMessage           `json:"annotation,omitempty"`
	ResultTypeName      string                    `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier             `json:"resultTypeSpecifier,omitempty"`
	Element             []*TupleElementDefinition `json:"element,omitempty"`
}

func (*TupleTypeSpecifier) isTypeSpecifier()                     {}
func (n *TupleTypeSpecifier) typeSpecifierJSON() ([]byte, error) { return n.MarshalJSON() }
func (n *TupleTypeSpecifier) MarshalJSON() ([]byte, error) {
	type alias TupleTypeSpecifier
	return marshalWithType("TupleTypeSpecifier", (*alias)(n))
}

// TupleElementDefinition is one element in a tuple type.
type TupleElementDefinition struct {
	LocalID     string          `json:"localId,omitempty"`
	Locator     string          `json:"locator,omitempty"`
	Annotation  json.RawMessage `json:"annotation,omitempty"`
	Name        string          `json:"name"`
	ElementType TypeSpecifier   `json:"elementType,omitempty"`
}

// ChoiceTypeSpecifier: Choice<T1, T2, ...>.
type ChoiceTypeSpecifier struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Choice              []TypeSpecifier `json:"choice,omitempty"`
}

func (*ChoiceTypeSpecifier) isTypeSpecifier()                     {}
func (n *ChoiceTypeSpecifier) typeSpecifierJSON() ([]byte, error) { return n.MarshalJSON() }
func (n *ChoiceTypeSpecifier) MarshalJSON() ([]byte, error) {
	type alias ChoiceTypeSpecifier
	return marshalWithType("ChoiceTypeSpecifier", (*alias)(n))
}

// -----------------------------------------------------------------------
// Expression interface and concrete types
// -----------------------------------------------------------------------

// Expression is the interface for all ELM expression nodes.
type Expression interface {
	isExpression()
}

// LiteralNode: a typed literal value.
type LiteralNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ValueType           string          `json:"valueType"`
	Value               string          `json:"value"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
}

func (*LiteralNode) isExpression() {}
func (n *LiteralNode) MarshalJSON() ([]byte, error) {
	type alias LiteralNode
	return marshalWithType("Literal", (*alias)(n))
}

// NullNode: the null literal.
type NullNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
}

func (*NullNode) isExpression() {}
func (n *NullNode) MarshalJSON() ([]byte, error) {
	type alias NullNode
	return marshalWithType("Null", (*alias)(n))
}

// ExpressionRefNode: a reference to a named expression.
type ExpressionRefNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Name                string          `json:"name"`
	LibraryName         string          `json:"libraryName,omitempty"`
}

func (*ExpressionRefNode) isExpression() {}
func (n *ExpressionRefNode) MarshalJSON() ([]byte, error) {
	type alias ExpressionRefNode
	return marshalWithType("ExpressionRef", (*alias)(n))
}

// ParameterRefNode: a reference to a parameter.
type ParameterRefNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Name                string          `json:"name"`
	LibraryName         string          `json:"libraryName,omitempty"`
}

func (*ParameterRefNode) isExpression() {}
func (n *ParameterRefNode) MarshalJSON() ([]byte, error) {
	type alias ParameterRefNode
	return marshalWithType("ParameterRef", (*alias)(n))
}

// ValueSetRefNode: a reference to a value set.
type ValueSetRefNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Name                string          `json:"name"`
	LibraryName         string          `json:"libraryName,omitempty"`
	Preserve            *bool           `json:"preserve,omitempty"`
}

func (*ValueSetRefNode) isExpression() {}
func (n *ValueSetRefNode) MarshalJSON() ([]byte, error) {
	type alias ValueSetRefNode
	return marshalWithType("ValueSetRef", (*alias)(n))
}

// InValueSetNode: tests whether a code belongs to a value set.
// Emitted instead of In(ToList(ValueSetRef)) when the RHS is a value set.
type InValueSetNode struct {
	LocalID             string           `json:"localId,omitempty"`
	Locator             string           `json:"locator,omitempty"`
	Annotation          json.RawMessage  `json:"annotation,omitempty"`
	ResultTypeName      string           `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier    `json:"resultTypeSpecifier,omitempty"`
	Signature           json.RawMessage  `json:"signature,omitempty"`
	Code                Expression       `json:"code"`
	ValueSet            *ValueSetRefNode `json:"valueset"`
}

func (*InValueSetNode) isExpression() {}
func (n *InValueSetNode) MarshalJSON() ([]byte, error) {
	// The valueset field inside InValueSet carries no "type" discriminator, so it
	// is marshalled through an alias (which suppresses ValueSetRefNode's own
	// MarshalJSON) rather than by copying its fields out one by one.
	type vsAlias ValueSetRefNode
	type alias InValueSetNode
	return marshalWithType("InValueSet", struct {
		*alias
		ValueSet *vsAlias `json:"valueset"`
	}{
		alias:    (*alias)(n),
		ValueSet: (*vsAlias)(n.ValueSet),
	})
}

// CalculateAgeNode: expands AgeInYears/AgeInMonths/etc. in a Patient context.
// Extends UnaryExpression — operand is a single object (not array).
type CalculateAgeNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Signature           json.RawMessage `json:"signature,omitempty"`
	Precision           string          `json:"precision,omitempty"`
	Operand             Expression      `json:"operand"`
}

func (*CalculateAgeNode) isExpression() {}
func (n *CalculateAgeNode) MarshalJSON() ([]byte, error) {
	type alias CalculateAgeNode
	return marshalWithType("CalculateAge", (*alias)(n))
}

// ToConceptNode: converts a Code or List<Code> to a Concept.
// This is the ELM built-in ToConcept operator — not the FHIRHelpers FunctionRef version.
// CQF emits this node when promoting a CodeRef on the RHS of a Concept ~ comparison.
type ToConceptNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Signature           json.RawMessage `json:"signature,omitempty"`
	Operand             Expression      `json:"operand"`
}

func (*ToConceptNode) isExpression() {}
func (n *ToConceptNode) MarshalJSON() ([]byte, error) {
	type alias ToConceptNode
	return marshalWithType("ToConcept", (*alias)(n))
}

// SplitNode: splits a string by a separator into a List<String>.
// ELM operator type "Split".
type SplitNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Signature           json.RawMessage `json:"signature,omitempty"`
	StringToSplit       Expression      `json:"stringToSplit"`
	Separator           Expression      `json:"separator"`
}

func (*SplitNode) isExpression() {}
func (n *SplitNode) MarshalJSON() ([]byte, error) {
	type alias SplitNode
	return marshalWithType("Split", (*alias)(n))
}

// LastNode: returns the last element of a list or string.
// ELM operator type "Last".
type LastNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Signature           json.RawMessage `json:"signature,omitempty"`
	Source              Expression      `json:"source"`
}

func (*LastNode) isExpression() {}
func (n *LastNode) MarshalJSON() ([]byte, error) {
	type alias LastNode
	return marshalWithType("Last", (*alias)(n))
}

// CodeSystemRefNode: a reference to a code system (as expression).
type CodeSystemRefNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Name                string          `json:"name"`
	LibraryName         string          `json:"libraryName,omitempty"`
}

func (*CodeSystemRefNode) isExpression() {}
func (n *CodeSystemRefNode) MarshalJSON() ([]byte, error) {
	type alias CodeSystemRefNode
	return marshalWithType("CodeSystemRef", (*alias)(n))
}

// CodeRefNode: a reference to a code (as expression).
type CodeRefNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Name                string          `json:"name"`
	LibraryName         string          `json:"libraryName,omitempty"`
}

func (*CodeRefNode) isExpression() {}
func (n *CodeRefNode) MarshalJSON() ([]byte, error) {
	type alias CodeRefNode
	return marshalWithType("CodeRef", (*alias)(n))
}

// ConceptRefNode: a reference to a concept (as expression).
type ConceptRefNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Name                string          `json:"name"`
	LibraryName         string          `json:"libraryName,omitempty"`
}

func (*ConceptRefNode) isExpression() {}
func (n *ConceptRefNode) MarshalJSON() ([]byte, error) {
	type alias ConceptRefNode
	return marshalWithType("ConceptRef", (*alias)(n))
}

// FunctionRefNode: a function invocation (extends OperatorExpression — has signature[]).
type FunctionRefNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Signature           json.RawMessage `json:"signature,omitempty"`
	Name                string          `json:"name"`
	LibraryName         string          `json:"libraryName,omitempty"`
	Operand             []Expression    `json:"operand,omitempty"`
}

func (*FunctionRefNode) isExpression() {}

// marshalFunctionRef emits the operand list even when it is empty, matching CQF:
// a call to a zero-argument function is "operand": [], not an absent key.
//
// As with FunctionDef, omitempty drops an empty slice regardless of nil-ness, so
// the tag is shadowed by an outer field rather than the value replaced.
func marshalFunctionRef(n *FunctionRefNode) ([]byte, error) {
	type alias FunctionRefNode
	type funcRef struct {
		*alias
		Operand []Expression `json:"operand"`
	}
	operands := n.Operand
	if operands == nil {
		operands = []Expression{}
	}
	return marshalWithType("FunctionRef", funcRef{alias: (*alias)(n), Operand: operands})
}

func (n *FunctionRefNode) MarshalJSON() ([]byte, error) {
	return marshalFunctionRef(n)
}

// OperatorExpressionNode: any operator expression (Add, Equal, Not, etc.).
type OperatorExpressionNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Signature           json.RawMessage `json:"signature,omitempty"`
	Operator            string          `json:"-"`
	Operand             []Expression    `json:"operand,omitempty"`
}

func (*OperatorExpressionNode) isExpression() {}
func (n *OperatorExpressionNode) MarshalJSON() ([]byte, error) {
	type alias OperatorExpressionNode
	return marshalWithType(n.Operator, (*alias)(n))
}

// PropertyNode: a property access (source.path).
type PropertyNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Path                string          `json:"path"`
	Source              Expression      `json:"source,omitempty"`
	Scope               string          `json:"scope,omitempty"`
}

func (*PropertyNode) isExpression() {}
func (n *PropertyNode) MarshalJSON() ([]byte, error) {
	type alias PropertyNode
	return marshalWithType("Property", (*alias)(n))
}

// RetrieveNode: a CQL retrieve expression.
type RetrieveNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	DataType            string          `json:"dataType"`
	TemplateID          string          `json:"templateId,omitempty"`
	CodeProperty        string          `json:"codeProperty,omitempty"`
	CodeComparator      string          `json:"codeComparator,omitempty"`
	Codes               Expression      `json:"codes,omitempty"`
	DateProperty        string          `json:"dateProperty,omitempty"`
	DateRange           Expression      `json:"dateRange,omitempty"`
	Context             string          `json:"context,omitempty"`
	// CQF-mode filter arrays — emitted as [] when CQFMode=true.
	Include     json.RawMessage `json:"include,omitempty"`
	CodeFilter  json.RawMessage `json:"codeFilter,omitempty"`
	DateFilter  json.RawMessage `json:"dateFilter,omitempty"`
	OtherFilter json.RawMessage `json:"otherFilter,omitempty"`
}

func (*RetrieveNode) isExpression() {}
func (n *RetrieveNode) MarshalJSON() ([]byte, error) {
	type alias RetrieveNode
	return marshalWithType("Retrieve", (*alias)(n))
}

// SingletonFromNode: wraps an expression to extract the single element of a list.
type SingletonFromNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Signature           json.RawMessage `json:"signature,omitempty"`
	Operand             Expression      `json:"operand"`
}

func (*SingletonFromNode) isExpression() {}
func (n *SingletonFromNode) MarshalJSON() ([]byte, error) {
	type alias SingletonFromNode
	return marshalWithType("SingletonFrom", (*alias)(n))
}

// QuantityNode: a UCUM quantity literal expression.
type QuantityNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Value               json.Number     `json:"value"`
	Unit                string          `json:"unit,omitempty"`
}

func (*QuantityNode) isExpression() {}
func (n *QuantityNode) MarshalJSON() ([]byte, error) {
	type alias QuantityNode
	return marshalWithType("Quantity", (*alias)(n))
}

// QuantityLiteral is a non-expression Quantity value used in Ratio sub-quantities.
// Unlike QuantityNode (an expression), it has no type discriminator and unit is always emitted.
type QuantityLiteral struct {
	Annotation     json.RawMessage `json:"annotation,omitempty"`
	Locator        string          `json:"locator,omitempty"`
	ResultTypeName string          `json:"resultTypeName,omitempty"`
	Unit           string          `json:"unit"`
	Value          json.Number     `json:"value"`
}

// RatioNode: a CQL ratio literal.
type RatioNode struct {
	LocalID             string           `json:"localId,omitempty"`
	Locator             string           `json:"locator,omitempty"`
	Annotation          json.RawMessage  `json:"annotation,omitempty"`
	ResultTypeName      string           `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier    `json:"resultTypeSpecifier,omitempty"`
	Numerator           *QuantityLiteral `json:"numerator"`
	Denominator         *QuantityLiteral `json:"denominator"`
}

func (*RatioNode) isExpression() {}
func (n *RatioNode) MarshalJSON() ([]byte, error) {
	type alias RatioNode
	return marshalWithType("Ratio", (*alias)(n))
}

type UnimplementedNode struct {
	LocalID             string        `json:"localId,omitempty"`
	Locator             string        `json:"locator,omitempty"`
	ResultTypeName      string        `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier `json:"resultTypeSpecifier,omitempty"`
	TypeName            string        `json:"-"`
}

func (*UnimplementedNode) isExpression() {}
func (n *UnimplementedNode) MarshalJSON() ([]byte, error) {
	typeName := n.TypeName
	if typeName == "" {
		typeName = "Unimplemented"
	}
	type alias UnimplementedNode
	return marshalWithType(typeName, (*alias)(n))
}

// UnaryExpressionNode is an ELM operator with a single operand (e.g. Abs, Not, ToList).
// In ELM JSON, single-operand operators emit operand as an object, not an array.
type UnaryExpressionNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Signature           json.RawMessage `json:"signature,omitempty"`
	Operator            string          `json:"-"`
	Operand             Expression      `json:"operand"`
}

func (*UnaryExpressionNode) isExpression() {}
func (n *UnaryExpressionNode) MarshalJSON() ([]byte, error) {
	type alias UnaryExpressionNode
	return marshalWithType(n.Operator, (*alias)(n))
}

// AggregateExpressionNode is an ELM aggregate operator (e.g. Count, Sum, Avg).
// Unlike UnaryExpression, the ELM schema uses "source" (single object) not "operand".
type AggregateExpressionNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Signature           json.RawMessage `json:"signature,omitempty"`
	Operator            string          `json:"-"`
	Source              Expression      `json:"source"`
}

func (*AggregateExpressionNode) isExpression() {}
func (n *AggregateExpressionNode) MarshalJSON() ([]byte, error) {
	type alias AggregateExpressionNode
	return marshalWithType(n.Operator, (*alias)(n))
}

// NamedOperatorExpressionNode is an ELM operator whose operands are emitted as
// named JSON fields (e.g. Substring → stringToSub/startIndex/length). Each
// operand is stored with the field name CQF emits for that operator slot.
type NamedOperatorExpressionNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Signature           json.RawMessage `json:"signature,omitempty"`
	Operator            string          `json:"-"`
	// Operands is an ordered list of (field-name, expression) pairs, emitted as
	// named JSON fields rather than an operand array.
	Operands []NamedOperand `json:"-"`
}

// NamedOperand is one named slot in a NamedOperatorExpressionNode.
type NamedOperand struct {
	Name  string
	Value Expression
}

func (*NamedOperatorExpressionNode) isExpression() {}
func (n *NamedOperatorExpressionNode) MarshalJSON() ([]byte, error) {
	// The scalar fields come from the struct so that new ones need no change
	// here; only the per-operator named operand slots are added by hand.
	type alias NamedOperatorExpressionNode
	base, err := marshalWithType(n.Operator, (*alias)(n))
	if err != nil {
		return nil, err
	}
	if len(n.Operands) == 0 {
		return base, nil
	}

	var operands bytes.Buffer
	operands.WriteByte('{')
	first := true
	for _, op := range n.Operands {
		if op.Value == nil {
			continue
		}
		b, mErr := json.Marshal(op.Value)
		if mErr != nil {
			return nil, mErr
		}
		if !first {
			operands.WriteByte(',')
		}
		first = false
		nameJSON, _ := json.Marshal(op.Name)
		operands.Write(nameJSON)
		operands.WriteByte(':')
		operands.Write(b)
	}
	operands.WriteByte('}')
	if first {
		return base, nil
	}
	return mergeJSONObjects(base, operands.Bytes())
}

// mergeJSONObjects concatenates the members of two JSON objects into one.
func mergeJSONObjects(a, b []byte) ([]byte, error) {
	if len(a) < 2 || a[0] != '{' || len(b) < 2 || b[0] != '{' {
		return nil, fmt.Errorf("elm: cannot merge non-object JSON")
	}
	if len(b) == 2 { // "{}"
		return a, nil
	}
	var buf bytes.Buffer
	buf.Write(a[:len(a)-1]) // drop closing brace
	if len(a) > 2 {
		buf.WriteByte(',')
	}
	buf.Write(b[1:]) // skip opening brace, keeps closing one
	return buf.Bytes(), nil
}

type DateNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Signature           json.RawMessage `json:"signature,omitempty"`
	Year                Expression      `json:"year,omitempty"`
	Month               Expression      `json:"month,omitempty"`
	Day                 Expression      `json:"day,omitempty"`
}

func (*DateNode) isExpression() {}
func (n *DateNode) MarshalJSON() ([]byte, error) {
	type alias DateNode
	return marshalWithType("Date", (*alias)(n))
}

// DateTimeNode: a CQL datetime literal or DateTime() function call.
type DateTimeNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Signature           json.RawMessage `json:"signature,omitempty"`
	Year                Expression      `json:"year,omitempty"`
	Month               Expression      `json:"month,omitempty"`
	Day                 Expression      `json:"day,omitempty"`
	Hour                Expression      `json:"hour,omitempty"`
	Minute              Expression      `json:"minute,omitempty"`
	Second              Expression      `json:"second,omitempty"`
	Millisecond         Expression      `json:"millisecond,omitempty"`
	TimezoneOffset      Expression      `json:"timezoneOffset,omitempty"`
}

func (*DateTimeNode) isExpression() {}
func (n *DateTimeNode) MarshalJSON() ([]byte, error) {
	type alias DateTimeNode
	return marshalWithType("DateTime", (*alias)(n))
}

// TimeNode: a CQL time literal or Time() function call.
type TimeNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Signature           json.RawMessage `json:"signature,omitempty"`
	Hour                Expression      `json:"hour,omitempty"`
	Minute              Expression      `json:"minute,omitempty"`
	Second              Expression      `json:"second,omitempty"`
	Millisecond         Expression      `json:"millisecond,omitempty"`
}

func (*TimeNode) isExpression() {}
func (n *TimeNode) MarshalJSON() ([]byte, error) {
	type alias TimeNode
	return marshalWithType("Time", (*alias)(n))
}

// -----------------------------------------------------------------------
// Additional expression types
// -----------------------------------------------------------------------

// IfNode: if condition then thenClause else elseClause.
type IfNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Condition           Expression      `json:"condition"`
	Then                Expression      `json:"then"`
	Else                Expression      `json:"else"`
}

func (*IfNode) isExpression() {}
func (n *IfNode) MarshalJSON() ([]byte, error) {
	type alias IfNode
	return marshalWithType("If", (*alias)(n))
}

// CaseItem is one when-then pair in a CaseNode.
type CaseItem struct {
	LocalID    string          `json:"localId,omitempty"`
	Locator    string          `json:"locator,omitempty"`
	Annotation json.RawMessage `json:"annotation,omitempty"`
	When       Expression      `json:"when"`
	Then       Expression      `json:"then"`
}

// CaseNode: case [comparand] when ... then ... else ... end.
type CaseNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Comparand           Expression      `json:"comparand,omitempty"`
	CaseItem            []*CaseItem     `json:"caseItem"`
	Else                Expression      `json:"else"`
}

func (*CaseNode) isExpression() {}
func (n *CaseNode) MarshalJSON() ([]byte, error) {
	type alias CaseNode
	return marshalWithType("Case", (*alias)(n))
}

// IsNode: x is TypeSpecifier (extends OperatorExpression — has signature[]).
type IsNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Signature           json.RawMessage `json:"signature,omitempty"`
	Operand             Expression      `json:"operand"`
	IsTypeSpecifier     TypeSpecifier   `json:"isTypeSpecifier,omitempty"`
}

func (*IsNode) isExpression() {}
func (n *IsNode) MarshalJSON() ([]byte, error) {
	type alias IsNode
	return marshalWithType("Is", (*alias)(n))
}

// AsNode: x as TypeSpecifier (strict=false) or cast x as TypeSpecifier (strict=true).
// Strict is a *bool: nil = omit (synthesized As), non-nil = emit (explicit CQL as/cast).
type AsNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Signature           json.RawMessage `json:"signature,omitempty"`
	Operand             Expression      `json:"operand"`
	AsType              string          `json:"asType,omitempty"`
	AsTypeSpecifier     TypeSpecifier   `json:"asTypeSpecifier,omitempty"`
	Strict              *bool           `json:"strict,omitempty"`
}

func (*AsNode) isExpression() {}
func (n *AsNode) MarshalJSON() ([]byte, error) {
	type alias AsNode
	return marshalWithType("As", (*alias)(n))
}

// ConvertNode: convert x to TypeSpecifier (extends OperatorExpression — has signature[]).
type ConvertNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Signature           json.RawMessage `json:"signature,omitempty"`
	Operand             Expression      `json:"operand"`
	ToTypeSpecifier     TypeSpecifier   `json:"toTypeSpecifier,omitempty"`
}

func (*ConvertNode) isExpression() {}
func (n *ConvertNode) MarshalJSON() ([]byte, error) {
	type alias ConvertNode
	return marshalWithType("Convert", (*alias)(n))
}

// IntervalNode: Interval selector.
type IntervalNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Low                 Expression      `json:"low,omitempty"`
	High                Expression      `json:"high,omitempty"`
	LowClosed           bool            `json:"lowClosed"`
	HighClosed          bool            `json:"highClosed"`
}

func (*IntervalNode) isExpression() {}
func (n *IntervalNode) MarshalJSON() ([]byte, error) {
	type alias IntervalNode
	return marshalWithType("Interval", (*alias)(n))
}

// ListNode: List selector.
type ListNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	TypeSpec            TypeSpecifier   `json:"typeSpecifier,omitempty"`
	Element             []Expression    `json:"element"`
}

func (*ListNode) isExpression() {}
func (n *ListNode) MarshalJSON() ([]byte, error) {
	// "element" is always present, even when empty; everything else comes from
	// the struct so new fields need no change here.
	type alias ListNode
	a := (*alias)(n)
	if a.Element == nil {
		cp := *a
		cp.Element = []Expression{}
		a = &cp
	}
	return marshalWithType("List", a)
}

// TupleElementNode is a named element in a Tuple or Instance selector.
type TupleElementNode struct {
	Name  string     `json:"name"`
	Value Expression `json:"value"`
}

// TupleNode: Tuple selector.
type TupleNode struct {
	LocalID             string              `json:"localId,omitempty"`
	Locator             string              `json:"locator,omitempty"`
	Annotation          json.RawMessage     `json:"annotation,omitempty"`
	ResultTypeName      string              `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier       `json:"resultTypeSpecifier,omitempty"`
	Element             []*TupleElementNode `json:"element,omitempty"`
}

func (*TupleNode) isExpression() {}
func (n *TupleNode) MarshalJSON() ([]byte, error) {
	type alias TupleNode
	return marshalWithType("Tuple", (*alias)(n))
}

// InstanceNode: Instance/class selector.
type InstanceNode struct {
	LocalID             string              `json:"localId,omitempty"`
	Locator             string              `json:"locator,omitempty"`
	Annotation          json.RawMessage     `json:"annotation,omitempty"`
	ResultTypeName      string              `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier       `json:"resultTypeSpecifier,omitempty"`
	ClassType           string              `json:"classType,omitempty"`
	Element             []*TupleElementNode `json:"element,omitempty"`
}

func (*InstanceNode) isExpression() {}
func (n *InstanceNode) MarshalJSON() ([]byte, error) {
	type alias InstanceNode
	return marshalWithType("Instance", (*alias)(n))
}

// CodeNode: Code selector literal.
type CodeNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Code                string          `json:"code"`
	System              *CodeSystemRef  `json:"system,omitempty"`
	Display             string          `json:"display,omitempty"`
}

func (*CodeNode) isExpression() {}
func (n *CodeNode) MarshalJSON() ([]byte, error) {
	type alias CodeNode
	return marshalWithType("Code", (*alias)(n))
}

// ConceptNode: Concept selector literal.
type ConceptNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Code                []*CodeNode     `json:"code,omitempty"`
	Display             string          `json:"display,omitempty"`
}

func (*ConceptNode) isExpression() {}
func (n *ConceptNode) MarshalJSON() ([]byte, error) {
	type alias ConceptNode
	return marshalWithType("Concept", (*alias)(n))
}

// QueryThisRefNode: $this — current iteration element.
type QueryThisRefNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
}

func (*QueryThisRefNode) isExpression() {}
func (n *QueryThisRefNode) MarshalJSON() ([]byte, error) {
	type alias QueryThisRefNode
	return marshalWithType("QueryThisRef", (*alias)(n))
}

// IdentifierRefNode: an identifier that is resolved by the engine at evaluation
// time rather than by the translator. CQF emits these for identifiers appearing
// in a sort-by expression, where the scope is the query's result element.
type IdentifierRefNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Name                string          `json:"name"`
	LibraryName         string          `json:"libraryName,omitempty"`
}

func (*IdentifierRefNode) isExpression() {}
func (n *IdentifierRefNode) MarshalJSON() ([]byte, error) {
	type alias IdentifierRefNode
	return marshalWithType("IdentifierRef", (*alias)(n))
}

// AliasRefNode: reference to a query source alias.
type AliasRefNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Name                string          `json:"name"`
}

func (*AliasRefNode) isExpression() {}
func (n *AliasRefNode) MarshalJSON() ([]byte, error) {
	type alias AliasRefNode
	return marshalWithType("AliasRef", (*alias)(n))
}

// LetRefNode: reference to a let clause binding inside a query.
// Serializes as ELM "QueryLetRef" to match ELM spec.
type LetRefNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Name                string          `json:"name"`
}

func (*LetRefNode) isExpression() {}
func (n *LetRefNode) MarshalJSON() ([]byte, error) {
	type alias LetRefNode
	return marshalWithType("QueryLetRef", (*alias)(n))
}

// ExternalConstantNode: %name external constant.
type ExternalConstantNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Name                string          `json:"name"`
}

func (*ExternalConstantNode) isExpression() {}
func (n *ExternalConstantNode) MarshalJSON() ([]byte, error) {
	type alias ExternalConstantNode
	return marshalWithType("ExternalConstant", (*alias)(n))
}

// PrecisionOperatorNode: an operator with a precision attribute (DurationBetween, DifferenceBetween, etc.).
// Extends OperatorExpression — has annotation[] and signature[].
type PrecisionOperatorNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Signature           json.RawMessage `json:"signature,omitempty"`
	Operator            string          `json:"-"`
	Precision           string          `json:"precision,omitempty"`
	Operand             []Expression    `json:"operand,omitempty"`
}

func (*PrecisionOperatorNode) isExpression() {}
func (n *PrecisionOperatorNode) MarshalJSON() ([]byte, error) {
	type alias PrecisionOperatorNode
	return marshalWithType(n.Operator, (*alias)(n))
}

// UnaryPrecisionOperatorNode: a single-operand operator with a precision attribute
// (DateTimeComponentFrom). ELM models these as UnaryExpression, so "operand" is a
// single object rather than the array used by PrecisionOperatorNode.
type UnaryPrecisionOperatorNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Signature           json.RawMessage `json:"signature,omitempty"`
	Operator            string          `json:"-"`
	Precision           string          `json:"precision,omitempty"`
	Operand             Expression      `json:"operand,omitempty"`
}

func (*UnaryPrecisionOperatorNode) isExpression() {}
func (n *UnaryPrecisionOperatorNode) MarshalJSON() ([]byte, error) {
	type alias UnaryPrecisionOperatorNode
	return marshalWithType(n.Operator, (*alias)(n))
}

// AliasedQuerySourceELM is the ELM representation of an aliased source.
type AliasedQuerySourceELM struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Alias               string          `json:"alias"`
	Expression          Expression      `json:"expression"`
}

// LetClauseELM is the ELM representation of a let clause.
type LetClauseELM struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Identifier          string          `json:"identifier"`
	Expression          Expression      `json:"expression"`
}

// RelationshipClauseELM is the ELM representation of a with/without clause.
type RelationshipClauseELM struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Kind                string          `json:"-"` // "With" or "Without"
	Alias               string          `json:"alias"`
	Expression          Expression      `json:"expression"`
	SuchThat            Expression      `json:"suchThat,omitempty"`
}

func (r *RelationshipClauseELM) MarshalJSON() ([]byte, error) {
	type alias RelationshipClauseELM
	return marshalWithType(r.Kind, (*alias)(r))
}

// ReturnClauseELM is the ELM return clause inside a query.
type ReturnClauseELM struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Distinct            *bool           `json:"distinct,omitempty"`
	Expression          Expression      `json:"expression"`
}

// AggregateClauseELM is the ELM aggregate clause inside a query.
// Distinct is a *bool so that an explicit `aggregate all` emits distinct:false
// rather than being swallowed by omitempty.
type AggregateClauseELM struct {
	LocalID             string        `json:"localId,omitempty"`
	Locator             string        `json:"locator,omitempty"`
	ResultTypeName      string        `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier `json:"resultTypeSpecifier,omitempty"`
	Distinct            *bool         `json:"distinct,omitempty"`
	Identifier          string        `json:"identifier"`
	Expression          Expression    `json:"expression"`
	Starting            Expression    `json:"starting,omitempty"`
}

// SortByItemELM is one item in a sort clause.
type SortByItemELM struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Direction           string          `json:"direction"`
	Path                string          `json:"path,omitempty"`
	Expression          Expression      `json:"expression,omitempty"`
}

func (s *SortByItemELM) MarshalJSON() ([]byte, error) {
	type alias SortByItemELM
	typeName := "ByExpression"
	if s.Path != "" && s.Expression == nil {
		typeName = "ByColumn"
	} else if s.Expression == nil {
		typeName = "ByDirection"
	}
	return marshalWithType(typeName, (*alias)(s))
}

// SortClauseELM holds the sort directives for a query.
type SortClauseELM struct {
	Annotation json.RawMessage  `json:"annotation,omitempty"`
	Locator    string           `json:"locator,omitempty"`
	By         []*SortByItemELM `json:"by,omitempty"`
}

// QueryNode is the ELM Query expression.
type QueryNode struct {
	LocalID             string                   `json:"localId,omitempty"`
	Locator             string                   `json:"locator,omitempty"`
	Annotation          json.RawMessage          `json:"annotation,omitempty"`
	ResultTypeName      string                   `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier            `json:"resultTypeSpecifier,omitempty"`
	Source              []*AliasedQuerySourceELM `json:"source"`
	Let                 []*LetClauseELM          `json:"let"`
	Relationship        []*RelationshipClauseELM `json:"relationship"`
	Where               Expression               `json:"where,omitempty"`
	Return              *ReturnClauseELM         `json:"return,omitempty"`
	Aggregate           *AggregateClauseELM      `json:"aggregate,omitempty"`
	Sort                *SortClauseELM           `json:"sort,omitempty"`
}

func (*QueryNode) isExpression() {}
func (n *QueryNode) MarshalJSON() ([]byte, error) {
	type alias QueryNode
	return marshalWithType("Query", (*alias)(n))
}

// MinMaxValueNode: MinValue<T> or MaxValue<T> — the extent of a type.
// These are OperatorExpression subclasses in ELM — they carry annotation and signature.
type MinMaxValueNode struct {
	LocalID             string          `json:"localId,omitempty"`
	Locator             string          `json:"locator,omitempty"`
	Annotation          json.RawMessage `json:"annotation,omitempty"`
	ResultTypeName      string          `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	Signature           json.RawMessage `json:"signature,omitempty"`
	Operator            string          `json:"-"` // "MinValue" or "MaxValue"
	ValueType           string          `json:"valueType,omitempty"`
}

func (*MinMaxValueNode) isExpression() {}
func (n *MinMaxValueNode) MarshalJSON() ([]byte, error) {
	type alias MinMaxValueNode
	return marshalWithType(n.Operator, (*alias)(n))
}

// -----------------------------------------------------------------------
// Annotation types
// -----------------------------------------------------------------------

// CqlToElmInfo is the standard translator annotation injected into the Library.
type CqlToElmInfo struct {
	TranslatorOptions  string `json:"translatorOptions"` // always emit even if empty
	TranslatorVersion  string `json:"translatorVersion,omitempty"`
	SignatureLevel     string `json:"signatureLevel,omitempty"`
	CompatibilityLevel string `json:"compatibilityLevel,omitempty"`
}

func (c *CqlToElmInfo) MarshalJSON() ([]byte, error) {
	type alias CqlToElmInfo
	return marshalWithType("CqlToElmInfo", (*alias)(c))
}
