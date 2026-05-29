// Package ast defines the CQL Abstract Syntax Tree node types produced
// by the echo-elm parser. All nodes implement the Node interface and carry
// source location information (TrackingInterval) for diagnostic reporting.
package ast

import "fmt"

// -----------------------------------------------------------------------
// Source location
// -----------------------------------------------------------------------

// Position is a 1-based line/column pair.
type Position struct {
	Line   int
	Column int
}

func (p Position) String() string {
	return fmt.Sprintf("%d:%d", p.Line, p.Column)
}

// Interval is the source span of an AST node.
type Interval struct {
	Start Position
	Stop  Position
	// SourceName is the library file or identifier, empty for the root file.
	SourceName string
}

func (i Interval) String() string {
	if i.SourceName != "" {
		return fmt.Sprintf("%s %s-%s", i.SourceName, i.Start, i.Stop)
	}
	return fmt.Sprintf("%s-%s", i.Start, i.Stop)
}

// -----------------------------------------------------------------------
// Core Node interface
// -----------------------------------------------------------------------

// Node is implemented by every AST node.
type Node interface {
	nodeMarker()
	Loc() Interval
}

// baseNode embeds location and satisfies the Node interface.
type baseNode struct {
	loc Interval
}

func (b baseNode) nodeMarker()   {}
func (b baseNode) Loc() Interval { return b.loc }

// SetLoc sets the source location of this node.
func (b *baseNode) SetLoc(loc Interval) { b.loc = loc }

// -----------------------------------------------------------------------
// Library (root node)
// -----------------------------------------------------------------------

// Library is the root node of a parsed CQL file. It corresponds to a
// single CQL library definition with optional version, usings, includes,
// codesystems, valuesets, codes, concepts, parameters, and statements.
type Library struct {
	baseNode
	Name        *VersionedIdentifier // library name + version
	Usings      []*UsingDefinition
	Includes    []*IncludeDefinition
	Codesystems []*CodesystemDefinition
	Valuesets   []*ValuesetDefinition
	Codes       []*CodeDefinition
	Concepts    []*ConceptDefinition
	Parameters  []*ParameterDefinition
	Statements  []*ExpressionDefinition
	Context     *ContextDefinition   // last declared context (backward compat)
	Contexts    []*ContextDefinition // all declared contexts in declaration order
}

// VersionedIdentifier holds a qualified name and optional version string.
type VersionedIdentifier struct {
	baseNode
	Name    string
	Version string // empty if not specified
}

// -----------------------------------------------------------------------
// Definitions
// -----------------------------------------------------------------------

// UsingDefinition: using FHIR version '4.0.1'
type UsingDefinition struct {
	baseNode
	ModelName string
	Version   string
	LocalName string // alias, often same as ModelName
}

// IncludeDefinition: include "Library" version '1.0.0' called Alias
type IncludeDefinition struct {
	baseNode
	Path      string
	Version   string
	LocalName string
}

// CodesystemDefinition: codesystem "Name": 'url' version 'version'
type CodesystemDefinition struct {
	baseNode
	Name        string
	ID          string
	Version     string
	AccessLevel AccessLevel
}

// ValuesetDefinition: valueset "Name": 'url'
type ValuesetDefinition struct {
	baseNode
	Name        string
	ID          string
	Version     string
	Codesystems []string
	AccessLevel AccessLevel
}

// CodeDefinition: code "Name" from CodeSystem
type CodeDefinition struct {
	baseNode
	Name          string
	Code          string
	SystemName    string
	SystemLocator Interval // source span of the codesystem identifier
	Display       string
	AccessLevel   AccessLevel
}

// ConceptDefinition: concept "Name": { codes }
type ConceptDefinition struct {
	baseNode
	Name        string
	Codes       []string
	Display     string
	AccessLevel AccessLevel
}

// ParameterDefinition: parameter Name ParameterType default <expr>
type ParameterDefinition struct {
	baseNode
	Name          string
	ParameterType *TypeSpecifier
	Default       Expr
	AccessLevel   AccessLevel
}

// ContextDefinition: context Patient
type ContextDefinition struct {
	baseNode
	Name string
}

// CQLAnnotationTag is a @name: value tag parsed from a CQL block comment.
type CQLAnnotationTag struct {
	Name  string
	Value string
}

// CQLAnnotation is a structured annotation parsed from a CQL block comment
// containing @tag: value pairs. Maps to ELM {"t": [...], "type": "Annotation"}.
type CQLAnnotation struct {
	Tags []CQLAnnotationTag
}

// ExpressionDefinition: define "Name": <expr>  or  define function ...
type ExpressionDefinition struct {
	baseNode
	Name        string
	Expression  Expr
	AccessLevel AccessLevel
	IsFunction  bool
	Operands    []*OperandDef // non-nil when IsFunction=true
	ReturnType  *TypeSpecifier
	IsFluent    bool
	Context     string          // active context name at declaration time (empty = Unfiltered)
	Annotations []CQLAnnotation // parsed @tag: value annotations from preceding block comments
}

// OperandDef is a function parameter.
type OperandDef struct {
	baseNode
	Name string
	Type *TypeSpecifier
}

// -----------------------------------------------------------------------
// Type specifiers
// -----------------------------------------------------------------------

// TypeSpecifier is the base interface for all type references.
type TypeSpecifier interface {
	Node
	typeSpecMarker()
}

// NamedTypeSpecifier: Integer, FHIR.Patient
type NamedTypeSpecifier struct {
	baseNode
	Qualifier string
	Name      string
}

func (n *NamedTypeSpecifier) typeSpecMarker() {}

// ListTypeSpecifier: List<Integer>
type ListTypeSpecifier struct {
	baseNode
	ElementType TypeSpecifier
}

func (l *ListTypeSpecifier) typeSpecMarker() {}

// IntervalTypeSpecifier: Interval<Integer>
type IntervalTypeSpecifier struct {
	baseNode
	PointType TypeSpecifier
}

func (i *IntervalTypeSpecifier) typeSpecMarker() {}

// TupleTypeSpecifier: Tuple { name Type, ... }
type TupleTypeSpecifier struct {
	baseNode
	Elements []*TupleTypeElement
}

func (t *TupleTypeSpecifier) typeSpecMarker() {}

// TupleTypeElement is one element in a tuple type.
type TupleTypeElement struct {
	baseNode
	Name string
	Type TypeSpecifier
}

// ChoiceTypeSpecifier: Choice<Integer, String>
type ChoiceTypeSpecifier struct {
	baseNode
	Types []TypeSpecifier
}

func (c *ChoiceTypeSpecifier) typeSpecMarker() {}

// -----------------------------------------------------------------------
// Expressions (Expr)
// -----------------------------------------------------------------------

// Expr is the base interface for all expression nodes.
type Expr interface {
	Node
	exprMarker()
}

// baseExpr is embedded in all expression nodes.
type baseExpr struct{ baseNode }

func (b *baseExpr) exprMarker() {}

// -----------------------------------------------------------------------
// Literal expressions
// -----------------------------------------------------------------------

// BooleanLiteral: true | false
type BooleanLiteral struct {
	baseExpr
	Value bool
}

// IntegerLiteral: 42
type IntegerLiteral struct {
	baseExpr
	Value int64
}

// LongLiteral: 42L
type LongLiteral struct {
	baseExpr
	Value int64
}

// DecimalLiteral: 3.14
type DecimalLiteral struct {
	baseExpr
	Value string // preserve original decimal text
}

// StringLiteral: 'hello'
type StringLiteral struct {
	baseExpr
	Value string
}

// QuantityLiteral: 1 'mg'
type QuantityLiteral struct {
	baseExpr
	Value string
	Unit  string
}

// DateLiteral: @2024-01-01
type DateLiteral struct {
	baseExpr
	Value string
}

// DateTimeLiteral: @2024-01-01T12:00:00
type DateTimeLiteral struct {
	baseExpr
	Value string
}

// TimeLiteral: @T12:00:00
type TimeLiteral struct {
	baseExpr
	Value string
}

// NullLiteral: null
type NullLiteral struct {
	baseExpr
}

// RatioLiteral: 1:3
type RatioLiteral struct {
	baseExpr
	Numerator   *QuantityLiteral
	Denominator *QuantityLiteral
}

// -----------------------------------------------------------------------
// Reference / identifier expressions
// -----------------------------------------------------------------------

// IdentifierRef: Name  (local identifier)
type IdentifierRef struct {
	baseExpr
	Name string
}

// QualifiedRef: Library.Name
type QualifiedRef struct {
	baseExpr
	LibraryName string
	Name        string
}

// AliasRef: reference to a query source alias inside a query expression
type AliasRef struct {
	baseExpr
	Name string
}

// LetRef: reference to a let clause binding inside a query expression
type LetRef struct {
	baseExpr
	Name string
}

// ThisExpr: $this — the current iteration element
type ThisExpr struct{ baseExpr }

// IndexExpr: $index — the current iteration index
type IndexExpr struct{ baseExpr }

// TotalExpr: $total — the aggregate accumulator
type TotalExpr struct{ baseExpr }

// ExternalConstantExpr: %name
type ExternalConstantExpr struct {
	baseExpr
	Name string
}

// -----------------------------------------------------------------------
// Unary / binary / ternary
// -----------------------------------------------------------------------

// UnaryExpr wraps a single-operand expression.
type UnaryExpr struct {
	baseExpr
	Op        string
	Operand   Expr
	Precision string // for date-time component ops (e.g. "Year")
}

// BinaryExpr wraps a two-operand expression.
type BinaryExpr struct {
	baseExpr
	Op        string
	Left      Expr
	Right     Expr
	Precision string // for membership with precision (e.g. "day of")
}

// TernaryExpr: if cond then thenExpr else elseExpr
type TernaryExpr struct {
	baseExpr
	Condition Expr
	ThenExpr  Expr
	ElseExpr  Expr
}

// CaseExpr: case [comparand] when ... then ... else ... end
type CaseExpr struct {
	baseExpr
	Comparand Expr // nil for multi-condition (when-clause) form
	Items     []*CaseItem
	Else      Expr
}

// CaseItem is one when-then pair inside a CaseExpr.
type CaseItem struct {
	baseNode
	When Expr
	Then Expr
}

// BetweenExpr: x [properly] between low and high
type BetweenExpr struct {
	baseExpr
	Operand  Expr
	Low      Expr
	High     Expr
	Properly bool
}

// DurationBetweenExpr: [duration in] Y between low and high
type DurationBetweenExpr struct {
	baseExpr
	Precision string
	Low       Expr
	High      Expr
}

// DifferenceBetweenExpr: difference in Y between low and high
type DifferenceBetweenExpr struct {
	baseExpr
	Precision string
	Low       Expr
	High      Expr
}

// -----------------------------------------------------------------------
// Type operators
// -----------------------------------------------------------------------

// TypeIsExpr: x is null | x is not null | x is true | x is false | x is TypeSpec
type TypeIsExpr struct {
	baseExpr
	Operand  Expr
	TypeSpec TypeSpecifier // non-nil only for "is TypeSpec"
	IsNull   bool
	IsTrue   bool
	IsFalse  bool
	Negated  bool // for "is not null"
}

// TypeAsExpr: x as TypeSpec  or  cast x as TypeSpec
type TypeAsExpr struct {
	baseExpr
	Operand  Expr
	TypeSpec TypeSpecifier
	Strict   bool // true for "cast x as T"
}

// ConvertExpr: convert x to TypeSpec
type ConvertExpr struct {
	baseExpr
	Operand  Expr
	TypeSpec TypeSpecifier
}

// -----------------------------------------------------------------------
// Timing / interval
// -----------------------------------------------------------------------

// TimingExpr represents all CQL interval-operator-phrase expressions.
// Op encodes the operator name (Before, After, During, Meets, etc.).
type TimingExpr struct {
	baseExpr
	Op        string
	Left      Expr
	Right     Expr
	Precision string // optional date-time precision qualifier
}

// IntervalExpr: Interval ( '[' | '(' ) low ',' high ( ']' | ')' )
type IntervalExpr struct {
	baseExpr
	Low        Expr
	High       Expr
	LowClosed  bool
	HighClosed bool
}

// TimeBoundaryExpr: start of x | end of x
type TimeBoundaryExpr struct {
	baseExpr
	Boundary string // "start" or "end"
	Source   Expr
}

// DateTimeComponentExpr: year from x | month from x | etc.
type DateTimeComponentExpr struct {
	baseExpr
	Precision string // "Year", "Month", etc.
	Source    Expr
}

// DurationExpr: duration in Y of x
type DurationExpr struct {
	baseExpr
	Precision string
	Source    Expr
}

// DifferenceExpr: difference in Y of x
type DifferenceExpr struct {
	baseExpr
	Precision string
	Source    Expr
}

// WidthExpr: width of x
type WidthExpr struct {
	baseExpr
	Source Expr
}

// SuccessorExpr: successor of x
type SuccessorExpr struct {
	baseExpr
	Source Expr
}

// PredecessorExpr: predecessor of x
type PredecessorExpr struct {
	baseExpr
	Source Expr
}

// SingletonFromExpr: singleton from x
type SingletonFromExpr struct {
	baseExpr
	Source Expr
}

// PointFromExpr: point from x
type PointFromExpr struct {
	baseExpr
	Source Expr
}

// TypeExtentExpr: minimum TypeSpec | maximum TypeSpec
type TypeExtentExpr struct {
	baseExpr
	Extent   string // "minimum" or "maximum"
	TypeSpec TypeSpecifier
}

// -----------------------------------------------------------------------
// Collection / selector expressions
// -----------------------------------------------------------------------

// ListExpr: List<T>? { exprs }
type ListExpr struct {
	baseExpr
	TypeSpec TypeSpecifier // optional element type annotation
	Elements []Expr
}

// TupleExpr: Tuple? { key: val, ... }
type TupleExpr struct {
	baseExpr
	Elements []*TupleElement
}

// TupleElement is one key-value pair in a TupleExpr.
type TupleElement struct {
	baseNode
	Name       string
	Expression Expr
}

// InstanceExpr: TypeName { key: val, ... }
type InstanceExpr struct {
	baseExpr
	TypeSpec TypeSpecifier
	Elements []*TupleElement
}

// CodeExpr: Code 'code' from codesystemIdent [display 'text']
type CodeExpr struct {
	baseExpr
	Code    string
	System  string
	Display string
}

// ConceptExpr: Concept { code selectors } [display 'text']
type ConceptExpr struct {
	baseExpr
	Codes   []*CodeExpr
	Display string
}

// -----------------------------------------------------------------------
// Property access
// -----------------------------------------------------------------------

// PropertyExpr: source.path  (member access from an expression)
type PropertyExpr struct {
	baseExpr
	Source Expr
	Path   string
}

// IndexedAccessExpr: source[index]
type IndexedAccessExpr struct {
	baseExpr
	Source Expr
	Index  Expr
}

// -----------------------------------------------------------------------
// Retrieve
// -----------------------------------------------------------------------

// RetrieveExpr: [context? TypeName: codeFilter?]
type RetrieveExpr struct {
	baseExpr
	DataType       string // e.g. "FHIR.Condition"
	CodeProperty   string
	CodeComparator string // "in", "~", "="
	Codes          Expr
	ContextExpr    Expr
}

// -----------------------------------------------------------------------
// Aggregate / set
// -----------------------------------------------------------------------

// AggregateExpr: distinct x | flatten x
type AggregateExpr struct {
	baseExpr
	Op      string // "Distinct" or "Flatten"
	Operand Expr
}

// SetAggregateExpr: expand x [per Y] | collapse x [per Y]
type SetAggregateExpr struct {
	baseExpr
	Op        string // "Expand" or "Collapse"
	Operand   Expr
	PerClause Expr // optional
}

// -----------------------------------------------------------------------
// Invocation / function call
// -----------------------------------------------------------------------

// FunctionRef: FunctionName(args...)
type FunctionRef struct {
	baseExpr
	LibraryName string
	Name        string
	Operands    []Expr
}

// ExternalFunctionRef: ExternalLib.FunctionName(args...)
type ExternalFunctionRef struct {
	baseExpr
	LibraryName string
	Name        string
	Operands    []Expr
}

// -----------------------------------------------------------------------
// Query expression
// -----------------------------------------------------------------------

// QueryExpression represents a CQL query (from ... return ...)
type QueryExpression struct {
	baseExpr
	Sources      []*AliasedQuerySource
	Let          []*LetClause
	Relationship []QueryRelationship
	Where        Expr
	Return       *ReturnClause
	Aggregate    *AggregateClause
	Sort         *SortClause
}

// AliasedQuerySource: Source A
type AliasedQuerySource struct {
	baseNode
	Expression Expr
	Alias      string
}

// LetClause: let X: expr
type LetClause struct {
	baseNode
	Identifier string
	Expression Expr
}

// QueryRelationship is a with/without join clause.
type QueryRelationship struct {
	baseNode
	Kind     string // "with" or "without"
	Source   *AliasedQuerySource
	SuchThat Expr
}

// ReturnClause: return (distinct|all)? expr
// Distinct is nil for plain 'return' (default distinct), true for 'return distinct',
// false for 'return all'. Only 'return all' maps to ELM distinct:false.
type ReturnClause struct {
	baseNode
	Distinct   *bool
	Expression Expr
}

// AggregateClause: aggregate distinct? func starting <expr>
type AggregateClause struct {
	baseNode
	Distinct   bool
	Identifier string
	Expression Expr
	Starting   Expr
}

// SortClause: sort by [SortByItem...]
type SortClause struct {
	baseNode
	Items []*SortByItem
}

// SortByItem: expr (asc|desc)
type SortByItem struct {
	baseNode
	Expression Expr
	Direction  SortDirection
	// DirectionText preserves the original keyword form (asc/ascending/desc/descending).
	DirectionText string
}

// SortDirection is asc or desc.
type SortDirection int

const (
	SortAsc SortDirection = iota
	SortDesc
)

// -----------------------------------------------------------------------
// Enumerations
// -----------------------------------------------------------------------

// AccessLevel controls public/private visibility.
type AccessLevel int

const (
	Public  AccessLevel = iota
	Private             // private keyword
)
