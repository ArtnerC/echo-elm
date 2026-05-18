package elm

import (
	"bytes"
	"encoding/json"
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
	Identifier       *VersionedIdentifier `json:"identifier,omitempty"`
	Annotation       []json.RawMessage    `json:"annotation,omitempty"`
	Usings           *UsingDefs           `json:"usings,omitempty"`
	Includes         *IncludeDefs         `json:"includes,omitempty"`
	Parameters       *ParameterDefs       `json:"parameters,omitempty"`
	CodeSystems      *CodeSystemDefs      `json:"codeSystems,omitempty"`
	ValueSets        *ValueSetDefs        `json:"valueSets,omitempty"`
	Codes            *CodeDefs            `json:"codes,omitempty"`
	Concepts         *ConceptDefs         `json:"concepts,omitempty"`
	Statements       *StatementDefs       `json:"statements,omitempty"`
}

// VersionedIdentifier holds an id and optional version string.
type VersionedIdentifier struct {
	ID      string `json:"id"`
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

// StatementDefs wraps the list of statement (expression/function) definitions.
type StatementDefs struct {
	Def []*StatementDef `json:"def"`
}

// -----------------------------------------------------------------------
// Def types
// -----------------------------------------------------------------------

// UsingDef corresponds to a `using` declaration.
type UsingDef struct {
	LocalID         string `json:"localId,omitempty"`
	Locator         string `json:"locator,omitempty"`
	LocalIdentifier string `json:"localIdentifier"`
	URI             string `json:"uri"`
	Version         string `json:"version,omitempty"`
	AccessLevel     string `json:"accessLevel,omitempty"`
}

// IncludeDef corresponds to an `include` declaration.
type IncludeDef struct {
	LocalID         string `json:"localId,omitempty"`
	Locator         string `json:"locator,omitempty"`
	LocalIdentifier string `json:"localIdentifier"`
	Path            string `json:"path"`
	Version         string `json:"version,omitempty"`
	AccessLevel     string `json:"accessLevel,omitempty"`
}

// CodeSystemDef corresponds to a `codesystem` declaration.
type CodeSystemDef struct {
	LocalID     string `json:"localId,omitempty"`
	Locator     string `json:"locator,omitempty"`
	Name        string `json:"name"`
	ID          string `json:"id"`
	Version     string `json:"version,omitempty"`
	AccessLevel string `json:"accessLevel,omitempty"`
}

// ValueSetDef corresponds to a `valueset` declaration.
type ValueSetDef struct {
	LocalID     string           `json:"localId,omitempty"`
	Locator     string           `json:"locator,omitempty"`
	Name        string           `json:"name"`
	ID          string           `json:"id"`
	Version     string           `json:"version,omitempty"`
	AccessLevel string           `json:"accessLevel,omitempty"`
	CodeSystems []*CodeSystemRef `json:"codeSystem,omitempty"`
}

// CodeDef corresponds to a `code` declaration.
type CodeDef struct {
	LocalID     string         `json:"localId,omitempty"`
	Locator     string         `json:"locator,omitempty"`
	Name        string         `json:"name"`
	ID          string         `json:"id"`
	Display     string         `json:"display,omitempty"`
	AccessLevel string         `json:"accessLevel,omitempty"`
	CodeSystem  *CodeSystemRef `json:"codeSystem,omitempty"`
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
	LocalID     string     `json:"localId,omitempty"`
	Locator     string     `json:"locator,omitempty"`
	Name        string     `json:"name"`
	Display     string     `json:"display,omitempty"`
	AccessLevel string     `json:"accessLevel,omitempty"`
	Code        []*CodeRef `json:"code,omitempty"`
}

// CodeRef is a reference to a code by name.
type CodeRef struct {
	LocalID     string `json:"localId,omitempty"`
	Locator     string `json:"locator,omitempty"`
	Name        string `json:"name"`
	LibraryName string `json:"libraryName,omitempty"`
}

func (r *CodeRef) MarshalJSON() ([]byte, error) {
	type alias CodeRef
	return marshalWithType("CodeRef", (*alias)(r))
}

// ParameterDef corresponds to a `parameter` declaration.
type ParameterDef struct {
	LocalID                string        `json:"localId,omitempty"`
	Locator                string        `json:"locator,omitempty"`
	Name                   string        `json:"name"`
	AccessLevel            string        `json:"accessLevel,omitempty"`
	ParameterTypeSpecifier TypeSpecifier `json:"parameterTypeSpecifier,omitempty"`
	Default                Expression    `json:"default,omitempty"`
}

// StatementDef covers both ExpressionDef and FunctionDef.
// IsFunction controls which type name is emitted in JSON/XML.
type StatementDef struct {
	LocalID             string        `json:"localId,omitempty"`
	Locator             string        `json:"locator,omitempty"`
	Name                string        `json:"name"`
	Context             string        `json:"context,omitempty"`
	AccessLevel         string        `json:"accessLevel,omitempty"`
	IsFunction          bool          `json:"-"`
	IsFluent            bool          `json:"fluent,omitempty"`
	Operand             []*OperandDef `json:"operand,omitempty"`
	ResultTypeSpecifier TypeSpecifier `json:"resultTypeSpecifier,omitempty"`
	Expression          Expression    `json:"expression,omitempty"`
}

func (s *StatementDef) MarshalJSON() ([]byte, error) {
	typeName := "ExpressionDef"
	if s.IsFunction {
		typeName = "FunctionDef"
	}
	type alias StatementDef
	return marshalWithType(typeName, (*alias)(s))
}

// OperandDef is a function parameter definition.
type OperandDef struct {
	LocalID              string        `json:"localId,omitempty"`
	Locator              string        `json:"locator,omitempty"`
	Name                 string        `json:"name"`
	OperandTypeSpecifier TypeSpecifier `json:"operandTypeSpecifier,omitempty"`
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
	LocalID string `json:"localId,omitempty"`
	Locator string `json:"locator,omitempty"`
	Name    string `json:"name,omitempty"`
}

func (*NamedTypeSpecifier) isTypeSpecifier() {}
func (n *NamedTypeSpecifier) typeSpecifierJSON() ([]byte, error) { return n.MarshalJSON() }
func (n *NamedTypeSpecifier) MarshalJSON() ([]byte, error) {
	type alias NamedTypeSpecifier
	return marshalWithType("NamedTypeSpecifier", (*alias)(n))
}

// IntervalTypeSpecifier: Interval<T>.
type IntervalTypeSpecifier struct {
	LocalID   string        `json:"localId,omitempty"`
	Locator   string        `json:"locator,omitempty"`
	PointType TypeSpecifier `json:"pointType,omitempty"`
}

func (*IntervalTypeSpecifier) isTypeSpecifier() {}
func (n *IntervalTypeSpecifier) typeSpecifierJSON() ([]byte, error) { return n.MarshalJSON() }
func (n *IntervalTypeSpecifier) MarshalJSON() ([]byte, error) {
	type alias IntervalTypeSpecifier
	return marshalWithType("IntervalTypeSpecifier", (*alias)(n))
}

// ListTypeSpecifier: List<T>.
type ListTypeSpecifier struct {
	LocalID     string        `json:"localId,omitempty"`
	Locator     string        `json:"locator,omitempty"`
	ElementType TypeSpecifier `json:"elementType,omitempty"`
}

func (*ListTypeSpecifier) isTypeSpecifier() {}
func (n *ListTypeSpecifier) typeSpecifierJSON() ([]byte, error) { return n.MarshalJSON() }
func (n *ListTypeSpecifier) MarshalJSON() ([]byte, error) {
	type alias ListTypeSpecifier
	return marshalWithType("ListTypeSpecifier", (*alias)(n))
}

// TupleTypeSpecifier: Tuple { name Type, ... }.
type TupleTypeSpecifier struct {
	LocalID string                    `json:"localId,omitempty"`
	Locator string                    `json:"locator,omitempty"`
	Element []*TupleElementDefinition `json:"element,omitempty"`
}

func (*TupleTypeSpecifier) isTypeSpecifier() {}
func (n *TupleTypeSpecifier) typeSpecifierJSON() ([]byte, error) { return n.MarshalJSON() }
func (n *TupleTypeSpecifier) MarshalJSON() ([]byte, error) {
	type alias TupleTypeSpecifier
	return marshalWithType("TupleTypeSpecifier", (*alias)(n))
}

// TupleElementDefinition is one element in a tuple type.
type TupleElementDefinition struct {
	LocalID string        `json:"localId,omitempty"`
	Locator string        `json:"locator,omitempty"`
	Name    string        `json:"name"`
	Type    TypeSpecifier `json:"type,omitempty"`
}

// ChoiceTypeSpecifier: Choice<T1, T2, ...>.
type ChoiceTypeSpecifier struct {
	LocalID string          `json:"localId,omitempty"`
	Locator string          `json:"locator,omitempty"`
	Choice  []TypeSpecifier `json:"choice,omitempty"`
}

func (*ChoiceTypeSpecifier) isTypeSpecifier() {}
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
	LocalID        string `json:"localId,omitempty"`
	Locator        string `json:"locator,omitempty"`
	ValueType      string `json:"valueType"`
	Value          string `json:"value"`
	ResultTypeName string `json:"resultTypeName,omitempty"`
}

func (*LiteralNode) isExpression() {}
func (n *LiteralNode) MarshalJSON() ([]byte, error) {
	type alias LiteralNode
	return marshalWithType("Literal", (*alias)(n))
}

// NullNode: the null literal.
type NullNode struct {
	LocalID        string `json:"localId,omitempty"`
	Locator        string `json:"locator,omitempty"`
	ResultTypeName string `json:"resultTypeName,omitempty"`
}

func (*NullNode) isExpression() {}
func (n *NullNode) MarshalJSON() ([]byte, error) {
	type alias NullNode
	return marshalWithType("Null", (*alias)(n))
}

// ExpressionRefNode: a reference to a named expression.
type ExpressionRefNode struct {
	LocalID     string `json:"localId,omitempty"`
	Locator     string `json:"locator,omitempty"`
	Name        string `json:"name"`
	LibraryName string `json:"libraryName,omitempty"`
}

func (*ExpressionRefNode) isExpression() {}
func (n *ExpressionRefNode) MarshalJSON() ([]byte, error) {
	type alias ExpressionRefNode
	return marshalWithType("ExpressionRef", (*alias)(n))
}

// ParameterRefNode: a reference to a parameter.
type ParameterRefNode struct {
	LocalID     string `json:"localId,omitempty"`
	Locator     string `json:"locator,omitempty"`
	Name        string `json:"name"`
	LibraryName string `json:"libraryName,omitempty"`
}

func (*ParameterRefNode) isExpression() {}
func (n *ParameterRefNode) MarshalJSON() ([]byte, error) {
	type alias ParameterRefNode
	return marshalWithType("ParameterRef", (*alias)(n))
}

// ValueSetRefNode: a reference to a value set.
type ValueSetRefNode struct {
	LocalID     string `json:"localId,omitempty"`
	Locator     string `json:"locator,omitempty"`
	Name        string `json:"name"`
	LibraryName string `json:"libraryName,omitempty"`
}

func (*ValueSetRefNode) isExpression() {}
func (n *ValueSetRefNode) MarshalJSON() ([]byte, error) {
	type alias ValueSetRefNode
	return marshalWithType("ValueSetRef", (*alias)(n))
}

// CodeSystemRefNode: a reference to a code system (as expression).
type CodeSystemRefNode struct {
	LocalID     string `json:"localId,omitempty"`
	Locator     string `json:"locator,omitempty"`
	Name        string `json:"name"`
	LibraryName string `json:"libraryName,omitempty"`
}

func (*CodeSystemRefNode) isExpression() {}
func (n *CodeSystemRefNode) MarshalJSON() ([]byte, error) {
	type alias CodeSystemRefNode
	return marshalWithType("CodeSystemRef", (*alias)(n))
}

// CodeRefNode: a reference to a code (as expression).
type CodeRefNode struct {
	LocalID     string `json:"localId,omitempty"`
	Locator     string `json:"locator,omitempty"`
	Name        string `json:"name"`
	LibraryName string `json:"libraryName,omitempty"`
}

func (*CodeRefNode) isExpression() {}
func (n *CodeRefNode) MarshalJSON() ([]byte, error) {
	type alias CodeRefNode
	return marshalWithType("CodeRef", (*alias)(n))
}

// ConceptRefNode: a reference to a concept (as expression).
type ConceptRefNode struct {
	LocalID string `json:"localId,omitempty"`
	Locator string `json:"locator,omitempty"`
	Name    string `json:"name"`
}

func (*ConceptRefNode) isExpression() {}
func (n *ConceptRefNode) MarshalJSON() ([]byte, error) {
	type alias ConceptRefNode
	return marshalWithType("ConceptRef", (*alias)(n))
}

// FunctionRefNode: a function invocation.
type FunctionRefNode struct {
	LocalID     string       `json:"localId,omitempty"`
	Locator     string       `json:"locator,omitempty"`
	Name        string       `json:"name"`
	LibraryName string       `json:"libraryName,omitempty"`
	Operand     []Expression `json:"operand,omitempty"`
}

func (*FunctionRefNode) isExpression() {}
func (n *FunctionRefNode) MarshalJSON() ([]byte, error) {
	type alias FunctionRefNode
	return marshalWithType("FunctionRef", (*alias)(n))
}

// OperatorExpressionNode: any operator expression (Add, Equal, Not, etc.).
type OperatorExpressionNode struct {
	LocalID  string       `json:"localId,omitempty"`
	Locator  string       `json:"locator,omitempty"`
	Operator string       `json:"-"`
	Operand  []Expression `json:"operand,omitempty"`
}

func (*OperatorExpressionNode) isExpression() {}
func (n *OperatorExpressionNode) MarshalJSON() ([]byte, error) {
	type alias OperatorExpressionNode
	return marshalWithType(n.Operator, (*alias)(n))
}

// PropertyNode: a property access (source.path).
type PropertyNode struct {
	LocalID string     `json:"localId,omitempty"`
	Locator string     `json:"locator,omitempty"`
	Path    string     `json:"path"`
	Source  Expression `json:"source,omitempty"`
	Scope   string     `json:"scope,omitempty"`
}

func (*PropertyNode) isExpression() {}
func (n *PropertyNode) MarshalJSON() ([]byte, error) {
	type alias PropertyNode
	return marshalWithType("Property", (*alias)(n))
}

// RetrieveNode: a CQL retrieve expression.
type RetrieveNode struct {
	LocalID      string     `json:"localId,omitempty"`
	Locator      string     `json:"locator,omitempty"`
	DataType     string     `json:"dataType"`
	TemplateID   string     `json:"templateId,omitempty"`
	CodeProperty string     `json:"codeProperty,omitempty"`
	Codes        Expression `json:"codes,omitempty"`
	DateProperty string     `json:"dateProperty,omitempty"`
	DateRange    Expression `json:"dateRange,omitempty"`
	Context      string     `json:"context,omitempty"`
}

func (*RetrieveNode) isExpression() {}
func (n *RetrieveNode) MarshalJSON() ([]byte, error) {
	type alias RetrieveNode
	return marshalWithType("Retrieve", (*alias)(n))
}

// UnimplementedNode is a placeholder for expressions not yet translated.
type UnimplementedNode struct {
	LocalID  string `json:"localId,omitempty"`
	Locator  string `json:"locator,omitempty"`
	TypeName string `json:"-"`
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

// -----------------------------------------------------------------------
// Annotation types
// -----------------------------------------------------------------------

// CqlToElmInfo is the standard translator annotation injected into the Library.
type CqlToElmInfo struct {
	TranslatorOptions string `json:"translatorOptions,omitempty"`
	TranslatorVersion string `json:"translatorVersion,omitempty"`
	SignatureLevel    string `json:"signatureLevel,omitempty"`
}

func (c *CqlToElmInfo) MarshalJSON() ([]byte, error) {
	type alias CqlToElmInfo
	return marshalWithType("CqlToElmInfo", (*alias)(c))
}
