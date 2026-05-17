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

func (b baseNode) nodeMarker() {}
func (b baseNode) Loc() Interval { return b.loc }

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
	Context     *ContextDefinition
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
	Name        string
	Code        string
	SystemName  string
	Display     string
	AccessLevel AccessLevel
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
	Name        string
	ParameterType *TypeSpecifier
	Default     Expr
	AccessLevel AccessLevel
}

// ContextDefinition: context Patient
type ContextDefinition struct {
	baseNode
	Name string
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

// -----------------------------------------------------------------------
// Unary / binary / ternary
// -----------------------------------------------------------------------

// UnaryExpr wraps a single-operand expression.
type UnaryExpr struct {
	baseExpr
	Op      string
	Operand Expr
}

// BinaryExpr wraps a two-operand expression.
type BinaryExpr struct {
	baseExpr
	Op    string
	Left  Expr
	Right Expr
}

// TernaryExpr: If then else
type TernaryExpr struct {
	baseExpr
	Condition  Expr
	ThenExpr   Expr
	ElseExpr   Expr
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
// Aggregate / query
// -----------------------------------------------------------------------

// QueryExpression represents a CQL query (from ... return ...)
type QueryExpression struct {
	baseExpr
	Sources    []*AliasedQuerySource
	Let        []*LetClause
	Relationship []Expr
	Where      Expr
	Return     *ReturnClause
	Aggregate  *AggregateClause
	Sort       *SortClause
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

// ReturnClause: return distinct? expr
type ReturnClause struct {
	baseNode
	Distinct   bool
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
