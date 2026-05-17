// Code generated from cql.g4 by ANTLR 4.13.2. DO NOT EDIT.

package cqlparser // cql
import "github.com/antlr4-go/antlr/v4"

// BasecqlListener is a complete listener for a parse tree produced by cqlParser.
type BasecqlListener struct{}

var _ cqlListener = &BasecqlListener{}

// VisitTerminal is called when a terminal node is visited.
func (s *BasecqlListener) VisitTerminal(node antlr.TerminalNode) {}

// VisitErrorNode is called when an error node is visited.
func (s *BasecqlListener) VisitErrorNode(node antlr.ErrorNode) {}

// EnterEveryRule is called when any rule is entered.
func (s *BasecqlListener) EnterEveryRule(ctx antlr.ParserRuleContext) {}

// ExitEveryRule is called when any rule is exited.
func (s *BasecqlListener) ExitEveryRule(ctx antlr.ParserRuleContext) {}

// EnterDefinition is called when production definition is entered.
func (s *BasecqlListener) EnterDefinition(ctx *DefinitionContext) {}

// ExitDefinition is called when production definition is exited.
func (s *BasecqlListener) ExitDefinition(ctx *DefinitionContext) {}

// EnterLibrary is called when production library is entered.
func (s *BasecqlListener) EnterLibrary(ctx *LibraryContext) {}

// ExitLibrary is called when production library is exited.
func (s *BasecqlListener) ExitLibrary(ctx *LibraryContext) {}

// EnterLibraryDefinition is called when production libraryDefinition is entered.
func (s *BasecqlListener) EnterLibraryDefinition(ctx *LibraryDefinitionContext) {}

// ExitLibraryDefinition is called when production libraryDefinition is exited.
func (s *BasecqlListener) ExitLibraryDefinition(ctx *LibraryDefinitionContext) {}

// EnterUsingDefinition is called when production usingDefinition is entered.
func (s *BasecqlListener) EnterUsingDefinition(ctx *UsingDefinitionContext) {}

// ExitUsingDefinition is called when production usingDefinition is exited.
func (s *BasecqlListener) ExitUsingDefinition(ctx *UsingDefinitionContext) {}

// EnterIncludeDefinition is called when production includeDefinition is entered.
func (s *BasecqlListener) EnterIncludeDefinition(ctx *IncludeDefinitionContext) {}

// ExitIncludeDefinition is called when production includeDefinition is exited.
func (s *BasecqlListener) ExitIncludeDefinition(ctx *IncludeDefinitionContext) {}

// EnterLocalIdentifier is called when production localIdentifier is entered.
func (s *BasecqlListener) EnterLocalIdentifier(ctx *LocalIdentifierContext) {}

// ExitLocalIdentifier is called when production localIdentifier is exited.
func (s *BasecqlListener) ExitLocalIdentifier(ctx *LocalIdentifierContext) {}

// EnterAccessModifier is called when production accessModifier is entered.
func (s *BasecqlListener) EnterAccessModifier(ctx *AccessModifierContext) {}

// ExitAccessModifier is called when production accessModifier is exited.
func (s *BasecqlListener) ExitAccessModifier(ctx *AccessModifierContext) {}

// EnterParameterDefinition is called when production parameterDefinition is entered.
func (s *BasecqlListener) EnterParameterDefinition(ctx *ParameterDefinitionContext) {}

// ExitParameterDefinition is called when production parameterDefinition is exited.
func (s *BasecqlListener) ExitParameterDefinition(ctx *ParameterDefinitionContext) {}

// EnterCodesystemDefinition is called when production codesystemDefinition is entered.
func (s *BasecqlListener) EnterCodesystemDefinition(ctx *CodesystemDefinitionContext) {}

// ExitCodesystemDefinition is called when production codesystemDefinition is exited.
func (s *BasecqlListener) ExitCodesystemDefinition(ctx *CodesystemDefinitionContext) {}

// EnterValuesetDefinition is called when production valuesetDefinition is entered.
func (s *BasecqlListener) EnterValuesetDefinition(ctx *ValuesetDefinitionContext) {}

// ExitValuesetDefinition is called when production valuesetDefinition is exited.
func (s *BasecqlListener) ExitValuesetDefinition(ctx *ValuesetDefinitionContext) {}

// EnterCodesystems is called when production codesystems is entered.
func (s *BasecqlListener) EnterCodesystems(ctx *CodesystemsContext) {}

// ExitCodesystems is called when production codesystems is exited.
func (s *BasecqlListener) ExitCodesystems(ctx *CodesystemsContext) {}

// EnterCodesystemIdentifier is called when production codesystemIdentifier is entered.
func (s *BasecqlListener) EnterCodesystemIdentifier(ctx *CodesystemIdentifierContext) {}

// ExitCodesystemIdentifier is called when production codesystemIdentifier is exited.
func (s *BasecqlListener) ExitCodesystemIdentifier(ctx *CodesystemIdentifierContext) {}

// EnterLibraryIdentifier is called when production libraryIdentifier is entered.
func (s *BasecqlListener) EnterLibraryIdentifier(ctx *LibraryIdentifierContext) {}

// ExitLibraryIdentifier is called when production libraryIdentifier is exited.
func (s *BasecqlListener) ExitLibraryIdentifier(ctx *LibraryIdentifierContext) {}

// EnterCodeDefinition is called when production codeDefinition is entered.
func (s *BasecqlListener) EnterCodeDefinition(ctx *CodeDefinitionContext) {}

// ExitCodeDefinition is called when production codeDefinition is exited.
func (s *BasecqlListener) ExitCodeDefinition(ctx *CodeDefinitionContext) {}

// EnterConceptDefinition is called when production conceptDefinition is entered.
func (s *BasecqlListener) EnterConceptDefinition(ctx *ConceptDefinitionContext) {}

// ExitConceptDefinition is called when production conceptDefinition is exited.
func (s *BasecqlListener) ExitConceptDefinition(ctx *ConceptDefinitionContext) {}

// EnterCodeIdentifier is called when production codeIdentifier is entered.
func (s *BasecqlListener) EnterCodeIdentifier(ctx *CodeIdentifierContext) {}

// ExitCodeIdentifier is called when production codeIdentifier is exited.
func (s *BasecqlListener) ExitCodeIdentifier(ctx *CodeIdentifierContext) {}

// EnterCodesystemId is called when production codesystemId is entered.
func (s *BasecqlListener) EnterCodesystemId(ctx *CodesystemIdContext) {}

// ExitCodesystemId is called when production codesystemId is exited.
func (s *BasecqlListener) ExitCodesystemId(ctx *CodesystemIdContext) {}

// EnterValuesetId is called when production valuesetId is entered.
func (s *BasecqlListener) EnterValuesetId(ctx *ValuesetIdContext) {}

// ExitValuesetId is called when production valuesetId is exited.
func (s *BasecqlListener) ExitValuesetId(ctx *ValuesetIdContext) {}

// EnterVersionSpecifier is called when production versionSpecifier is entered.
func (s *BasecqlListener) EnterVersionSpecifier(ctx *VersionSpecifierContext) {}

// ExitVersionSpecifier is called when production versionSpecifier is exited.
func (s *BasecqlListener) ExitVersionSpecifier(ctx *VersionSpecifierContext) {}

// EnterCodeId is called when production codeId is entered.
func (s *BasecqlListener) EnterCodeId(ctx *CodeIdContext) {}

// ExitCodeId is called when production codeId is exited.
func (s *BasecqlListener) ExitCodeId(ctx *CodeIdContext) {}

// EnterTypeSpecifier is called when production typeSpecifier is entered.
func (s *BasecqlListener) EnterTypeSpecifier(ctx *TypeSpecifierContext) {}

// ExitTypeSpecifier is called when production typeSpecifier is exited.
func (s *BasecqlListener) ExitTypeSpecifier(ctx *TypeSpecifierContext) {}

// EnterNamedTypeSpecifier is called when production namedTypeSpecifier is entered.
func (s *BasecqlListener) EnterNamedTypeSpecifier(ctx *NamedTypeSpecifierContext) {}

// ExitNamedTypeSpecifier is called when production namedTypeSpecifier is exited.
func (s *BasecqlListener) ExitNamedTypeSpecifier(ctx *NamedTypeSpecifierContext) {}

// EnterModelIdentifier is called when production modelIdentifier is entered.
func (s *BasecqlListener) EnterModelIdentifier(ctx *ModelIdentifierContext) {}

// ExitModelIdentifier is called when production modelIdentifier is exited.
func (s *BasecqlListener) ExitModelIdentifier(ctx *ModelIdentifierContext) {}

// EnterListTypeSpecifier is called when production listTypeSpecifier is entered.
func (s *BasecqlListener) EnterListTypeSpecifier(ctx *ListTypeSpecifierContext) {}

// ExitListTypeSpecifier is called when production listTypeSpecifier is exited.
func (s *BasecqlListener) ExitListTypeSpecifier(ctx *ListTypeSpecifierContext) {}

// EnterIntervalTypeSpecifier is called when production intervalTypeSpecifier is entered.
func (s *BasecqlListener) EnterIntervalTypeSpecifier(ctx *IntervalTypeSpecifierContext) {}

// ExitIntervalTypeSpecifier is called when production intervalTypeSpecifier is exited.
func (s *BasecqlListener) ExitIntervalTypeSpecifier(ctx *IntervalTypeSpecifierContext) {}

// EnterTupleTypeSpecifier is called when production tupleTypeSpecifier is entered.
func (s *BasecqlListener) EnterTupleTypeSpecifier(ctx *TupleTypeSpecifierContext) {}

// ExitTupleTypeSpecifier is called when production tupleTypeSpecifier is exited.
func (s *BasecqlListener) ExitTupleTypeSpecifier(ctx *TupleTypeSpecifierContext) {}

// EnterTupleElementDefinition is called when production tupleElementDefinition is entered.
func (s *BasecqlListener) EnterTupleElementDefinition(ctx *TupleElementDefinitionContext) {}

// ExitTupleElementDefinition is called when production tupleElementDefinition is exited.
func (s *BasecqlListener) ExitTupleElementDefinition(ctx *TupleElementDefinitionContext) {}

// EnterChoiceTypeSpecifier is called when production choiceTypeSpecifier is entered.
func (s *BasecqlListener) EnterChoiceTypeSpecifier(ctx *ChoiceTypeSpecifierContext) {}

// ExitChoiceTypeSpecifier is called when production choiceTypeSpecifier is exited.
func (s *BasecqlListener) ExitChoiceTypeSpecifier(ctx *ChoiceTypeSpecifierContext) {}

// EnterStatement is called when production statement is entered.
func (s *BasecqlListener) EnterStatement(ctx *StatementContext) {}

// ExitStatement is called when production statement is exited.
func (s *BasecqlListener) ExitStatement(ctx *StatementContext) {}

// EnterExpressionDefinition is called when production expressionDefinition is entered.
func (s *BasecqlListener) EnterExpressionDefinition(ctx *ExpressionDefinitionContext) {}

// ExitExpressionDefinition is called when production expressionDefinition is exited.
func (s *BasecqlListener) ExitExpressionDefinition(ctx *ExpressionDefinitionContext) {}

// EnterContextDefinition is called when production contextDefinition is entered.
func (s *BasecqlListener) EnterContextDefinition(ctx *ContextDefinitionContext) {}

// ExitContextDefinition is called when production contextDefinition is exited.
func (s *BasecqlListener) ExitContextDefinition(ctx *ContextDefinitionContext) {}

// EnterFluentModifier is called when production fluentModifier is entered.
func (s *BasecqlListener) EnterFluentModifier(ctx *FluentModifierContext) {}

// ExitFluentModifier is called when production fluentModifier is exited.
func (s *BasecqlListener) ExitFluentModifier(ctx *FluentModifierContext) {}

// EnterFunctionDefinition is called when production functionDefinition is entered.
func (s *BasecqlListener) EnterFunctionDefinition(ctx *FunctionDefinitionContext) {}

// ExitFunctionDefinition is called when production functionDefinition is exited.
func (s *BasecqlListener) ExitFunctionDefinition(ctx *FunctionDefinitionContext) {}

// EnterOperandDefinition is called when production operandDefinition is entered.
func (s *BasecqlListener) EnterOperandDefinition(ctx *OperandDefinitionContext) {}

// ExitOperandDefinition is called when production operandDefinition is exited.
func (s *BasecqlListener) ExitOperandDefinition(ctx *OperandDefinitionContext) {}

// EnterFunctionBody is called when production functionBody is entered.
func (s *BasecqlListener) EnterFunctionBody(ctx *FunctionBodyContext) {}

// ExitFunctionBody is called when production functionBody is exited.
func (s *BasecqlListener) ExitFunctionBody(ctx *FunctionBodyContext) {}

// EnterQuerySource is called when production querySource is entered.
func (s *BasecqlListener) EnterQuerySource(ctx *QuerySourceContext) {}

// ExitQuerySource is called when production querySource is exited.
func (s *BasecqlListener) ExitQuerySource(ctx *QuerySourceContext) {}

// EnterAliasedQuerySource is called when production aliasedQuerySource is entered.
func (s *BasecqlListener) EnterAliasedQuerySource(ctx *AliasedQuerySourceContext) {}

// ExitAliasedQuerySource is called when production aliasedQuerySource is exited.
func (s *BasecqlListener) ExitAliasedQuerySource(ctx *AliasedQuerySourceContext) {}

// EnterAlias is called when production alias is entered.
func (s *BasecqlListener) EnterAlias(ctx *AliasContext) {}

// ExitAlias is called when production alias is exited.
func (s *BasecqlListener) ExitAlias(ctx *AliasContext) {}

// EnterQueryInclusionClause is called when production queryInclusionClause is entered.
func (s *BasecqlListener) EnterQueryInclusionClause(ctx *QueryInclusionClauseContext) {}

// ExitQueryInclusionClause is called when production queryInclusionClause is exited.
func (s *BasecqlListener) ExitQueryInclusionClause(ctx *QueryInclusionClauseContext) {}

// EnterWithClause is called when production withClause is entered.
func (s *BasecqlListener) EnterWithClause(ctx *WithClauseContext) {}

// ExitWithClause is called when production withClause is exited.
func (s *BasecqlListener) ExitWithClause(ctx *WithClauseContext) {}

// EnterWithoutClause is called when production withoutClause is entered.
func (s *BasecqlListener) EnterWithoutClause(ctx *WithoutClauseContext) {}

// ExitWithoutClause is called when production withoutClause is exited.
func (s *BasecqlListener) ExitWithoutClause(ctx *WithoutClauseContext) {}

// EnterRetrieve is called when production retrieve is entered.
func (s *BasecqlListener) EnterRetrieve(ctx *RetrieveContext) {}

// ExitRetrieve is called when production retrieve is exited.
func (s *BasecqlListener) ExitRetrieve(ctx *RetrieveContext) {}

// EnterContextIdentifier is called when production contextIdentifier is entered.
func (s *BasecqlListener) EnterContextIdentifier(ctx *ContextIdentifierContext) {}

// ExitContextIdentifier is called when production contextIdentifier is exited.
func (s *BasecqlListener) ExitContextIdentifier(ctx *ContextIdentifierContext) {}

// EnterCodePath is called when production codePath is entered.
func (s *BasecqlListener) EnterCodePath(ctx *CodePathContext) {}

// ExitCodePath is called when production codePath is exited.
func (s *BasecqlListener) ExitCodePath(ctx *CodePathContext) {}

// EnterCodeComparator is called when production codeComparator is entered.
func (s *BasecqlListener) EnterCodeComparator(ctx *CodeComparatorContext) {}

// ExitCodeComparator is called when production codeComparator is exited.
func (s *BasecqlListener) ExitCodeComparator(ctx *CodeComparatorContext) {}

// EnterTerminology is called when production terminology is entered.
func (s *BasecqlListener) EnterTerminology(ctx *TerminologyContext) {}

// ExitTerminology is called when production terminology is exited.
func (s *BasecqlListener) ExitTerminology(ctx *TerminologyContext) {}

// EnterQualifier is called when production qualifier is entered.
func (s *BasecqlListener) EnterQualifier(ctx *QualifierContext) {}

// ExitQualifier is called when production qualifier is exited.
func (s *BasecqlListener) ExitQualifier(ctx *QualifierContext) {}

// EnterQuery is called when production query is entered.
func (s *BasecqlListener) EnterQuery(ctx *QueryContext) {}

// ExitQuery is called when production query is exited.
func (s *BasecqlListener) ExitQuery(ctx *QueryContext) {}

// EnterSourceClause is called when production sourceClause is entered.
func (s *BasecqlListener) EnterSourceClause(ctx *SourceClauseContext) {}

// ExitSourceClause is called when production sourceClause is exited.
func (s *BasecqlListener) ExitSourceClause(ctx *SourceClauseContext) {}

// EnterLetClause is called when production letClause is entered.
func (s *BasecqlListener) EnterLetClause(ctx *LetClauseContext) {}

// ExitLetClause is called when production letClause is exited.
func (s *BasecqlListener) ExitLetClause(ctx *LetClauseContext) {}

// EnterLetClauseItem is called when production letClauseItem is entered.
func (s *BasecqlListener) EnterLetClauseItem(ctx *LetClauseItemContext) {}

// ExitLetClauseItem is called when production letClauseItem is exited.
func (s *BasecqlListener) ExitLetClauseItem(ctx *LetClauseItemContext) {}

// EnterWhereClause is called when production whereClause is entered.
func (s *BasecqlListener) EnterWhereClause(ctx *WhereClauseContext) {}

// ExitWhereClause is called when production whereClause is exited.
func (s *BasecqlListener) ExitWhereClause(ctx *WhereClauseContext) {}

// EnterReturnClause is called when production returnClause is entered.
func (s *BasecqlListener) EnterReturnClause(ctx *ReturnClauseContext) {}

// ExitReturnClause is called when production returnClause is exited.
func (s *BasecqlListener) ExitReturnClause(ctx *ReturnClauseContext) {}

// EnterAggregateClause is called when production aggregateClause is entered.
func (s *BasecqlListener) EnterAggregateClause(ctx *AggregateClauseContext) {}

// ExitAggregateClause is called when production aggregateClause is exited.
func (s *BasecqlListener) ExitAggregateClause(ctx *AggregateClauseContext) {}

// EnterStartingClause is called when production startingClause is entered.
func (s *BasecqlListener) EnterStartingClause(ctx *StartingClauseContext) {}

// ExitStartingClause is called when production startingClause is exited.
func (s *BasecqlListener) ExitStartingClause(ctx *StartingClauseContext) {}

// EnterSortClause is called when production sortClause is entered.
func (s *BasecqlListener) EnterSortClause(ctx *SortClauseContext) {}

// ExitSortClause is called when production sortClause is exited.
func (s *BasecqlListener) ExitSortClause(ctx *SortClauseContext) {}

// EnterSortDirection is called when production sortDirection is entered.
func (s *BasecqlListener) EnterSortDirection(ctx *SortDirectionContext) {}

// ExitSortDirection is called when production sortDirection is exited.
func (s *BasecqlListener) ExitSortDirection(ctx *SortDirectionContext) {}

// EnterSortByItem is called when production sortByItem is entered.
func (s *BasecqlListener) EnterSortByItem(ctx *SortByItemContext) {}

// ExitSortByItem is called when production sortByItem is exited.
func (s *BasecqlListener) ExitSortByItem(ctx *SortByItemContext) {}

// EnterQualifiedIdentifier is called when production qualifiedIdentifier is entered.
func (s *BasecqlListener) EnterQualifiedIdentifier(ctx *QualifiedIdentifierContext) {}

// ExitQualifiedIdentifier is called when production qualifiedIdentifier is exited.
func (s *BasecqlListener) ExitQualifiedIdentifier(ctx *QualifiedIdentifierContext) {}

// EnterQualifiedIdentifierExpression is called when production qualifiedIdentifierExpression is entered.
func (s *BasecqlListener) EnterQualifiedIdentifierExpression(ctx *QualifiedIdentifierExpressionContext) {
}

// ExitQualifiedIdentifierExpression is called when production qualifiedIdentifierExpression is exited.
func (s *BasecqlListener) ExitQualifiedIdentifierExpression(ctx *QualifiedIdentifierExpressionContext) {
}

// EnterQualifierExpression is called when production qualifierExpression is entered.
func (s *BasecqlListener) EnterQualifierExpression(ctx *QualifierExpressionContext) {}

// ExitQualifierExpression is called when production qualifierExpression is exited.
func (s *BasecqlListener) ExitQualifierExpression(ctx *QualifierExpressionContext) {}

// EnterSimplePathIndexer is called when production simplePathIndexer is entered.
func (s *BasecqlListener) EnterSimplePathIndexer(ctx *SimplePathIndexerContext) {}

// ExitSimplePathIndexer is called when production simplePathIndexer is exited.
func (s *BasecqlListener) ExitSimplePathIndexer(ctx *SimplePathIndexerContext) {}

// EnterSimplePathQualifiedIdentifier is called when production simplePathQualifiedIdentifier is entered.
func (s *BasecqlListener) EnterSimplePathQualifiedIdentifier(ctx *SimplePathQualifiedIdentifierContext) {
}

// ExitSimplePathQualifiedIdentifier is called when production simplePathQualifiedIdentifier is exited.
func (s *BasecqlListener) ExitSimplePathQualifiedIdentifier(ctx *SimplePathQualifiedIdentifierContext) {
}

// EnterSimplePathReferentialIdentifier is called when production simplePathReferentialIdentifier is entered.
func (s *BasecqlListener) EnterSimplePathReferentialIdentifier(ctx *SimplePathReferentialIdentifierContext) {
}

// ExitSimplePathReferentialIdentifier is called when production simplePathReferentialIdentifier is exited.
func (s *BasecqlListener) ExitSimplePathReferentialIdentifier(ctx *SimplePathReferentialIdentifierContext) {
}

// EnterSimpleStringLiteral is called when production simpleStringLiteral is entered.
func (s *BasecqlListener) EnterSimpleStringLiteral(ctx *SimpleStringLiteralContext) {}

// ExitSimpleStringLiteral is called when production simpleStringLiteral is exited.
func (s *BasecqlListener) ExitSimpleStringLiteral(ctx *SimpleStringLiteralContext) {}

// EnterSimpleNumberLiteral is called when production simpleNumberLiteral is entered.
func (s *BasecqlListener) EnterSimpleNumberLiteral(ctx *SimpleNumberLiteralContext) {}

// ExitSimpleNumberLiteral is called when production simpleNumberLiteral is exited.
func (s *BasecqlListener) ExitSimpleNumberLiteral(ctx *SimpleNumberLiteralContext) {}

// EnterDurationBetweenExpression is called when production durationBetweenExpression is entered.
func (s *BasecqlListener) EnterDurationBetweenExpression(ctx *DurationBetweenExpressionContext) {}

// ExitDurationBetweenExpression is called when production durationBetweenExpression is exited.
func (s *BasecqlListener) ExitDurationBetweenExpression(ctx *DurationBetweenExpressionContext) {}

// EnterInFixSetExpression is called when production inFixSetExpression is entered.
func (s *BasecqlListener) EnterInFixSetExpression(ctx *InFixSetExpressionContext) {}

// ExitInFixSetExpression is called when production inFixSetExpression is exited.
func (s *BasecqlListener) ExitInFixSetExpression(ctx *InFixSetExpressionContext) {}

// EnterRetrieveExpression is called when production retrieveExpression is entered.
func (s *BasecqlListener) EnterRetrieveExpression(ctx *RetrieveExpressionContext) {}

// ExitRetrieveExpression is called when production retrieveExpression is exited.
func (s *BasecqlListener) ExitRetrieveExpression(ctx *RetrieveExpressionContext) {}

// EnterTimingExpression is called when production timingExpression is entered.
func (s *BasecqlListener) EnterTimingExpression(ctx *TimingExpressionContext) {}

// ExitTimingExpression is called when production timingExpression is exited.
func (s *BasecqlListener) ExitTimingExpression(ctx *TimingExpressionContext) {}

// EnterQueryExpression is called when production queryExpression is entered.
func (s *BasecqlListener) EnterQueryExpression(ctx *QueryExpressionContext) {}

// ExitQueryExpression is called when production queryExpression is exited.
func (s *BasecqlListener) ExitQueryExpression(ctx *QueryExpressionContext) {}

// EnterNotExpression is called when production notExpression is entered.
func (s *BasecqlListener) EnterNotExpression(ctx *NotExpressionContext) {}

// ExitNotExpression is called when production notExpression is exited.
func (s *BasecqlListener) ExitNotExpression(ctx *NotExpressionContext) {}

// EnterBooleanExpression is called when production booleanExpression is entered.
func (s *BasecqlListener) EnterBooleanExpression(ctx *BooleanExpressionContext) {}

// ExitBooleanExpression is called when production booleanExpression is exited.
func (s *BasecqlListener) ExitBooleanExpression(ctx *BooleanExpressionContext) {}

// EnterOrExpression is called when production orExpression is entered.
func (s *BasecqlListener) EnterOrExpression(ctx *OrExpressionContext) {}

// ExitOrExpression is called when production orExpression is exited.
func (s *BasecqlListener) ExitOrExpression(ctx *OrExpressionContext) {}

// EnterCastExpression is called when production castExpression is entered.
func (s *BasecqlListener) EnterCastExpression(ctx *CastExpressionContext) {}

// ExitCastExpression is called when production castExpression is exited.
func (s *BasecqlListener) ExitCastExpression(ctx *CastExpressionContext) {}

// EnterAndExpression is called when production andExpression is entered.
func (s *BasecqlListener) EnterAndExpression(ctx *AndExpressionContext) {}

// ExitAndExpression is called when production andExpression is exited.
func (s *BasecqlListener) ExitAndExpression(ctx *AndExpressionContext) {}

// EnterBetweenExpression is called when production betweenExpression is entered.
func (s *BasecqlListener) EnterBetweenExpression(ctx *BetweenExpressionContext) {}

// ExitBetweenExpression is called when production betweenExpression is exited.
func (s *BasecqlListener) ExitBetweenExpression(ctx *BetweenExpressionContext) {}

// EnterMembershipExpression is called when production membershipExpression is entered.
func (s *BasecqlListener) EnterMembershipExpression(ctx *MembershipExpressionContext) {}

// ExitMembershipExpression is called when production membershipExpression is exited.
func (s *BasecqlListener) ExitMembershipExpression(ctx *MembershipExpressionContext) {}

// EnterDifferenceBetweenExpression is called when production differenceBetweenExpression is entered.
func (s *BasecqlListener) EnterDifferenceBetweenExpression(ctx *DifferenceBetweenExpressionContext) {}

// ExitDifferenceBetweenExpression is called when production differenceBetweenExpression is exited.
func (s *BasecqlListener) ExitDifferenceBetweenExpression(ctx *DifferenceBetweenExpressionContext) {}

// EnterInequalityExpression is called when production inequalityExpression is entered.
func (s *BasecqlListener) EnterInequalityExpression(ctx *InequalityExpressionContext) {}

// ExitInequalityExpression is called when production inequalityExpression is exited.
func (s *BasecqlListener) ExitInequalityExpression(ctx *InequalityExpressionContext) {}

// EnterEqualityExpression is called when production equalityExpression is entered.
func (s *BasecqlListener) EnterEqualityExpression(ctx *EqualityExpressionContext) {}

// ExitEqualityExpression is called when production equalityExpression is exited.
func (s *BasecqlListener) ExitEqualityExpression(ctx *EqualityExpressionContext) {}

// EnterExistenceExpression is called when production existenceExpression is entered.
func (s *BasecqlListener) EnterExistenceExpression(ctx *ExistenceExpressionContext) {}

// ExitExistenceExpression is called when production existenceExpression is exited.
func (s *BasecqlListener) ExitExistenceExpression(ctx *ExistenceExpressionContext) {}

// EnterImpliesExpression is called when production impliesExpression is entered.
func (s *BasecqlListener) EnterImpliesExpression(ctx *ImpliesExpressionContext) {}

// ExitImpliesExpression is called when production impliesExpression is exited.
func (s *BasecqlListener) ExitImpliesExpression(ctx *ImpliesExpressionContext) {}

// EnterTermExpression is called when production termExpression is entered.
func (s *BasecqlListener) EnterTermExpression(ctx *TermExpressionContext) {}

// ExitTermExpression is called when production termExpression is exited.
func (s *BasecqlListener) ExitTermExpression(ctx *TermExpressionContext) {}

// EnterTypeExpression is called when production typeExpression is entered.
func (s *BasecqlListener) EnterTypeExpression(ctx *TypeExpressionContext) {}

// ExitTypeExpression is called when production typeExpression is exited.
func (s *BasecqlListener) ExitTypeExpression(ctx *TypeExpressionContext) {}

// EnterDateTimePrecision is called when production dateTimePrecision is entered.
func (s *BasecqlListener) EnterDateTimePrecision(ctx *DateTimePrecisionContext) {}

// ExitDateTimePrecision is called when production dateTimePrecision is exited.
func (s *BasecqlListener) ExitDateTimePrecision(ctx *DateTimePrecisionContext) {}

// EnterDateTimeComponent is called when production dateTimeComponent is entered.
func (s *BasecqlListener) EnterDateTimeComponent(ctx *DateTimeComponentContext) {}

// ExitDateTimeComponent is called when production dateTimeComponent is exited.
func (s *BasecqlListener) ExitDateTimeComponent(ctx *DateTimeComponentContext) {}

// EnterPluralDateTimePrecision is called when production pluralDateTimePrecision is entered.
func (s *BasecqlListener) EnterPluralDateTimePrecision(ctx *PluralDateTimePrecisionContext) {}

// ExitPluralDateTimePrecision is called when production pluralDateTimePrecision is exited.
func (s *BasecqlListener) ExitPluralDateTimePrecision(ctx *PluralDateTimePrecisionContext) {}

// EnterAdditionExpressionTerm is called when production additionExpressionTerm is entered.
func (s *BasecqlListener) EnterAdditionExpressionTerm(ctx *AdditionExpressionTermContext) {}

// ExitAdditionExpressionTerm is called when production additionExpressionTerm is exited.
func (s *BasecqlListener) ExitAdditionExpressionTerm(ctx *AdditionExpressionTermContext) {}

// EnterIndexedExpressionTerm is called when production indexedExpressionTerm is entered.
func (s *BasecqlListener) EnterIndexedExpressionTerm(ctx *IndexedExpressionTermContext) {}

// ExitIndexedExpressionTerm is called when production indexedExpressionTerm is exited.
func (s *BasecqlListener) ExitIndexedExpressionTerm(ctx *IndexedExpressionTermContext) {}

// EnterWidthExpressionTerm is called when production widthExpressionTerm is entered.
func (s *BasecqlListener) EnterWidthExpressionTerm(ctx *WidthExpressionTermContext) {}

// ExitWidthExpressionTerm is called when production widthExpressionTerm is exited.
func (s *BasecqlListener) ExitWidthExpressionTerm(ctx *WidthExpressionTermContext) {}

// EnterSetAggregateExpressionTerm is called when production setAggregateExpressionTerm is entered.
func (s *BasecqlListener) EnterSetAggregateExpressionTerm(ctx *SetAggregateExpressionTermContext) {}

// ExitSetAggregateExpressionTerm is called when production setAggregateExpressionTerm is exited.
func (s *BasecqlListener) ExitSetAggregateExpressionTerm(ctx *SetAggregateExpressionTermContext) {}

// EnterTimeUnitExpressionTerm is called when production timeUnitExpressionTerm is entered.
func (s *BasecqlListener) EnterTimeUnitExpressionTerm(ctx *TimeUnitExpressionTermContext) {}

// ExitTimeUnitExpressionTerm is called when production timeUnitExpressionTerm is exited.
func (s *BasecqlListener) ExitTimeUnitExpressionTerm(ctx *TimeUnitExpressionTermContext) {}

// EnterIfThenElseExpressionTerm is called when production ifThenElseExpressionTerm is entered.
func (s *BasecqlListener) EnterIfThenElseExpressionTerm(ctx *IfThenElseExpressionTermContext) {}

// ExitIfThenElseExpressionTerm is called when production ifThenElseExpressionTerm is exited.
func (s *BasecqlListener) ExitIfThenElseExpressionTerm(ctx *IfThenElseExpressionTermContext) {}

// EnterTimeBoundaryExpressionTerm is called when production timeBoundaryExpressionTerm is entered.
func (s *BasecqlListener) EnterTimeBoundaryExpressionTerm(ctx *TimeBoundaryExpressionTermContext) {}

// ExitTimeBoundaryExpressionTerm is called when production timeBoundaryExpressionTerm is exited.
func (s *BasecqlListener) ExitTimeBoundaryExpressionTerm(ctx *TimeBoundaryExpressionTermContext) {}

// EnterElementExtractorExpressionTerm is called when production elementExtractorExpressionTerm is entered.
func (s *BasecqlListener) EnterElementExtractorExpressionTerm(ctx *ElementExtractorExpressionTermContext) {
}

// ExitElementExtractorExpressionTerm is called when production elementExtractorExpressionTerm is exited.
func (s *BasecqlListener) ExitElementExtractorExpressionTerm(ctx *ElementExtractorExpressionTermContext) {
}

// EnterConversionExpressionTerm is called when production conversionExpressionTerm is entered.
func (s *BasecqlListener) EnterConversionExpressionTerm(ctx *ConversionExpressionTermContext) {}

// ExitConversionExpressionTerm is called when production conversionExpressionTerm is exited.
func (s *BasecqlListener) ExitConversionExpressionTerm(ctx *ConversionExpressionTermContext) {}

// EnterTypeExtentExpressionTerm is called when production typeExtentExpressionTerm is entered.
func (s *BasecqlListener) EnterTypeExtentExpressionTerm(ctx *TypeExtentExpressionTermContext) {}

// ExitTypeExtentExpressionTerm is called when production typeExtentExpressionTerm is exited.
func (s *BasecqlListener) ExitTypeExtentExpressionTerm(ctx *TypeExtentExpressionTermContext) {}

// EnterPredecessorExpressionTerm is called when production predecessorExpressionTerm is entered.
func (s *BasecqlListener) EnterPredecessorExpressionTerm(ctx *PredecessorExpressionTermContext) {}

// ExitPredecessorExpressionTerm is called when production predecessorExpressionTerm is exited.
func (s *BasecqlListener) ExitPredecessorExpressionTerm(ctx *PredecessorExpressionTermContext) {}

// EnterPointExtractorExpressionTerm is called when production pointExtractorExpressionTerm is entered.
func (s *BasecqlListener) EnterPointExtractorExpressionTerm(ctx *PointExtractorExpressionTermContext) {
}

// ExitPointExtractorExpressionTerm is called when production pointExtractorExpressionTerm is exited.
func (s *BasecqlListener) ExitPointExtractorExpressionTerm(ctx *PointExtractorExpressionTermContext) {
}

// EnterMultiplicationExpressionTerm is called when production multiplicationExpressionTerm is entered.
func (s *BasecqlListener) EnterMultiplicationExpressionTerm(ctx *MultiplicationExpressionTermContext) {
}

// ExitMultiplicationExpressionTerm is called when production multiplicationExpressionTerm is exited.
func (s *BasecqlListener) ExitMultiplicationExpressionTerm(ctx *MultiplicationExpressionTermContext) {
}

// EnterAggregateExpressionTerm is called when production aggregateExpressionTerm is entered.
func (s *BasecqlListener) EnterAggregateExpressionTerm(ctx *AggregateExpressionTermContext) {}

// ExitAggregateExpressionTerm is called when production aggregateExpressionTerm is exited.
func (s *BasecqlListener) ExitAggregateExpressionTerm(ctx *AggregateExpressionTermContext) {}

// EnterDurationExpressionTerm is called when production durationExpressionTerm is entered.
func (s *BasecqlListener) EnterDurationExpressionTerm(ctx *DurationExpressionTermContext) {}

// ExitDurationExpressionTerm is called when production durationExpressionTerm is exited.
func (s *BasecqlListener) ExitDurationExpressionTerm(ctx *DurationExpressionTermContext) {}

// EnterDifferenceExpressionTerm is called when production differenceExpressionTerm is entered.
func (s *BasecqlListener) EnterDifferenceExpressionTerm(ctx *DifferenceExpressionTermContext) {}

// ExitDifferenceExpressionTerm is called when production differenceExpressionTerm is exited.
func (s *BasecqlListener) ExitDifferenceExpressionTerm(ctx *DifferenceExpressionTermContext) {}

// EnterCaseExpressionTerm is called when production caseExpressionTerm is entered.
func (s *BasecqlListener) EnterCaseExpressionTerm(ctx *CaseExpressionTermContext) {}

// ExitCaseExpressionTerm is called when production caseExpressionTerm is exited.
func (s *BasecqlListener) ExitCaseExpressionTerm(ctx *CaseExpressionTermContext) {}

// EnterPowerExpressionTerm is called when production powerExpressionTerm is entered.
func (s *BasecqlListener) EnterPowerExpressionTerm(ctx *PowerExpressionTermContext) {}

// ExitPowerExpressionTerm is called when production powerExpressionTerm is exited.
func (s *BasecqlListener) ExitPowerExpressionTerm(ctx *PowerExpressionTermContext) {}

// EnterSuccessorExpressionTerm is called when production successorExpressionTerm is entered.
func (s *BasecqlListener) EnterSuccessorExpressionTerm(ctx *SuccessorExpressionTermContext) {}

// ExitSuccessorExpressionTerm is called when production successorExpressionTerm is exited.
func (s *BasecqlListener) ExitSuccessorExpressionTerm(ctx *SuccessorExpressionTermContext) {}

// EnterPolarityExpressionTerm is called when production polarityExpressionTerm is entered.
func (s *BasecqlListener) EnterPolarityExpressionTerm(ctx *PolarityExpressionTermContext) {}

// ExitPolarityExpressionTerm is called when production polarityExpressionTerm is exited.
func (s *BasecqlListener) ExitPolarityExpressionTerm(ctx *PolarityExpressionTermContext) {}

// EnterTermExpressionTerm is called when production termExpressionTerm is entered.
func (s *BasecqlListener) EnterTermExpressionTerm(ctx *TermExpressionTermContext) {}

// ExitTermExpressionTerm is called when production termExpressionTerm is exited.
func (s *BasecqlListener) ExitTermExpressionTerm(ctx *TermExpressionTermContext) {}

// EnterInvocationExpressionTerm is called when production invocationExpressionTerm is entered.
func (s *BasecqlListener) EnterInvocationExpressionTerm(ctx *InvocationExpressionTermContext) {}

// ExitInvocationExpressionTerm is called when production invocationExpressionTerm is exited.
func (s *BasecqlListener) ExitInvocationExpressionTerm(ctx *InvocationExpressionTermContext) {}

// EnterCaseExpressionItem is called when production caseExpressionItem is entered.
func (s *BasecqlListener) EnterCaseExpressionItem(ctx *CaseExpressionItemContext) {}

// ExitCaseExpressionItem is called when production caseExpressionItem is exited.
func (s *BasecqlListener) ExitCaseExpressionItem(ctx *CaseExpressionItemContext) {}

// EnterDateTimePrecisionSpecifier is called when production dateTimePrecisionSpecifier is entered.
func (s *BasecqlListener) EnterDateTimePrecisionSpecifier(ctx *DateTimePrecisionSpecifierContext) {}

// ExitDateTimePrecisionSpecifier is called when production dateTimePrecisionSpecifier is exited.
func (s *BasecqlListener) ExitDateTimePrecisionSpecifier(ctx *DateTimePrecisionSpecifierContext) {}

// EnterRelativeQualifier is called when production relativeQualifier is entered.
func (s *BasecqlListener) EnterRelativeQualifier(ctx *RelativeQualifierContext) {}

// ExitRelativeQualifier is called when production relativeQualifier is exited.
func (s *BasecqlListener) ExitRelativeQualifier(ctx *RelativeQualifierContext) {}

// EnterOffsetRelativeQualifier is called when production offsetRelativeQualifier is entered.
func (s *BasecqlListener) EnterOffsetRelativeQualifier(ctx *OffsetRelativeQualifierContext) {}

// ExitOffsetRelativeQualifier is called when production offsetRelativeQualifier is exited.
func (s *BasecqlListener) ExitOffsetRelativeQualifier(ctx *OffsetRelativeQualifierContext) {}

// EnterExclusiveRelativeQualifier is called when production exclusiveRelativeQualifier is entered.
func (s *BasecqlListener) EnterExclusiveRelativeQualifier(ctx *ExclusiveRelativeQualifierContext) {}

// ExitExclusiveRelativeQualifier is called when production exclusiveRelativeQualifier is exited.
func (s *BasecqlListener) ExitExclusiveRelativeQualifier(ctx *ExclusiveRelativeQualifierContext) {}

// EnterQuantityOffset is called when production quantityOffset is entered.
func (s *BasecqlListener) EnterQuantityOffset(ctx *QuantityOffsetContext) {}

// ExitQuantityOffset is called when production quantityOffset is exited.
func (s *BasecqlListener) ExitQuantityOffset(ctx *QuantityOffsetContext) {}

// EnterTemporalRelationship is called when production temporalRelationship is entered.
func (s *BasecqlListener) EnterTemporalRelationship(ctx *TemporalRelationshipContext) {}

// ExitTemporalRelationship is called when production temporalRelationship is exited.
func (s *BasecqlListener) ExitTemporalRelationship(ctx *TemporalRelationshipContext) {}

// EnterConcurrentWithIntervalOperatorPhrase is called when production concurrentWithIntervalOperatorPhrase is entered.
func (s *BasecqlListener) EnterConcurrentWithIntervalOperatorPhrase(ctx *ConcurrentWithIntervalOperatorPhraseContext) {
}

// ExitConcurrentWithIntervalOperatorPhrase is called when production concurrentWithIntervalOperatorPhrase is exited.
func (s *BasecqlListener) ExitConcurrentWithIntervalOperatorPhrase(ctx *ConcurrentWithIntervalOperatorPhraseContext) {
}

// EnterIncludesIntervalOperatorPhrase is called when production includesIntervalOperatorPhrase is entered.
func (s *BasecqlListener) EnterIncludesIntervalOperatorPhrase(ctx *IncludesIntervalOperatorPhraseContext) {
}

// ExitIncludesIntervalOperatorPhrase is called when production includesIntervalOperatorPhrase is exited.
func (s *BasecqlListener) ExitIncludesIntervalOperatorPhrase(ctx *IncludesIntervalOperatorPhraseContext) {
}

// EnterIncludedInIntervalOperatorPhrase is called when production includedInIntervalOperatorPhrase is entered.
func (s *BasecqlListener) EnterIncludedInIntervalOperatorPhrase(ctx *IncludedInIntervalOperatorPhraseContext) {
}

// ExitIncludedInIntervalOperatorPhrase is called when production includedInIntervalOperatorPhrase is exited.
func (s *BasecqlListener) ExitIncludedInIntervalOperatorPhrase(ctx *IncludedInIntervalOperatorPhraseContext) {
}

// EnterBeforeOrAfterIntervalOperatorPhrase is called when production beforeOrAfterIntervalOperatorPhrase is entered.
func (s *BasecqlListener) EnterBeforeOrAfterIntervalOperatorPhrase(ctx *BeforeOrAfterIntervalOperatorPhraseContext) {
}

// ExitBeforeOrAfterIntervalOperatorPhrase is called when production beforeOrAfterIntervalOperatorPhrase is exited.
func (s *BasecqlListener) ExitBeforeOrAfterIntervalOperatorPhrase(ctx *BeforeOrAfterIntervalOperatorPhraseContext) {
}

// EnterWithinIntervalOperatorPhrase is called when production withinIntervalOperatorPhrase is entered.
func (s *BasecqlListener) EnterWithinIntervalOperatorPhrase(ctx *WithinIntervalOperatorPhraseContext) {
}

// ExitWithinIntervalOperatorPhrase is called when production withinIntervalOperatorPhrase is exited.
func (s *BasecqlListener) ExitWithinIntervalOperatorPhrase(ctx *WithinIntervalOperatorPhraseContext) {
}

// EnterMeetsIntervalOperatorPhrase is called when production meetsIntervalOperatorPhrase is entered.
func (s *BasecqlListener) EnterMeetsIntervalOperatorPhrase(ctx *MeetsIntervalOperatorPhraseContext) {}

// ExitMeetsIntervalOperatorPhrase is called when production meetsIntervalOperatorPhrase is exited.
func (s *BasecqlListener) ExitMeetsIntervalOperatorPhrase(ctx *MeetsIntervalOperatorPhraseContext) {}

// EnterOverlapsIntervalOperatorPhrase is called when production overlapsIntervalOperatorPhrase is entered.
func (s *BasecqlListener) EnterOverlapsIntervalOperatorPhrase(ctx *OverlapsIntervalOperatorPhraseContext) {
}

// ExitOverlapsIntervalOperatorPhrase is called when production overlapsIntervalOperatorPhrase is exited.
func (s *BasecqlListener) ExitOverlapsIntervalOperatorPhrase(ctx *OverlapsIntervalOperatorPhraseContext) {
}

// EnterStartsIntervalOperatorPhrase is called when production startsIntervalOperatorPhrase is entered.
func (s *BasecqlListener) EnterStartsIntervalOperatorPhrase(ctx *StartsIntervalOperatorPhraseContext) {
}

// ExitStartsIntervalOperatorPhrase is called when production startsIntervalOperatorPhrase is exited.
func (s *BasecqlListener) ExitStartsIntervalOperatorPhrase(ctx *StartsIntervalOperatorPhraseContext) {
}

// EnterEndsIntervalOperatorPhrase is called when production endsIntervalOperatorPhrase is entered.
func (s *BasecqlListener) EnterEndsIntervalOperatorPhrase(ctx *EndsIntervalOperatorPhraseContext) {}

// ExitEndsIntervalOperatorPhrase is called when production endsIntervalOperatorPhrase is exited.
func (s *BasecqlListener) ExitEndsIntervalOperatorPhrase(ctx *EndsIntervalOperatorPhraseContext) {}

// EnterInvocationTerm is called when production invocationTerm is entered.
func (s *BasecqlListener) EnterInvocationTerm(ctx *InvocationTermContext) {}

// ExitInvocationTerm is called when production invocationTerm is exited.
func (s *BasecqlListener) ExitInvocationTerm(ctx *InvocationTermContext) {}

// EnterLiteralTerm is called when production literalTerm is entered.
func (s *BasecqlListener) EnterLiteralTerm(ctx *LiteralTermContext) {}

// ExitLiteralTerm is called when production literalTerm is exited.
func (s *BasecqlListener) ExitLiteralTerm(ctx *LiteralTermContext) {}

// EnterExternalConstantTerm is called when production externalConstantTerm is entered.
func (s *BasecqlListener) EnterExternalConstantTerm(ctx *ExternalConstantTermContext) {}

// ExitExternalConstantTerm is called when production externalConstantTerm is exited.
func (s *BasecqlListener) ExitExternalConstantTerm(ctx *ExternalConstantTermContext) {}

// EnterIntervalSelectorTerm is called when production intervalSelectorTerm is entered.
func (s *BasecqlListener) EnterIntervalSelectorTerm(ctx *IntervalSelectorTermContext) {}

// ExitIntervalSelectorTerm is called when production intervalSelectorTerm is exited.
func (s *BasecqlListener) ExitIntervalSelectorTerm(ctx *IntervalSelectorTermContext) {}

// EnterTupleSelectorTerm is called when production tupleSelectorTerm is entered.
func (s *BasecqlListener) EnterTupleSelectorTerm(ctx *TupleSelectorTermContext) {}

// ExitTupleSelectorTerm is called when production tupleSelectorTerm is exited.
func (s *BasecqlListener) ExitTupleSelectorTerm(ctx *TupleSelectorTermContext) {}

// EnterInstanceSelectorTerm is called when production instanceSelectorTerm is entered.
func (s *BasecqlListener) EnterInstanceSelectorTerm(ctx *InstanceSelectorTermContext) {}

// ExitInstanceSelectorTerm is called when production instanceSelectorTerm is exited.
func (s *BasecqlListener) ExitInstanceSelectorTerm(ctx *InstanceSelectorTermContext) {}

// EnterListSelectorTerm is called when production listSelectorTerm is entered.
func (s *BasecqlListener) EnterListSelectorTerm(ctx *ListSelectorTermContext) {}

// ExitListSelectorTerm is called when production listSelectorTerm is exited.
func (s *BasecqlListener) ExitListSelectorTerm(ctx *ListSelectorTermContext) {}

// EnterCodeSelectorTerm is called when production codeSelectorTerm is entered.
func (s *BasecqlListener) EnterCodeSelectorTerm(ctx *CodeSelectorTermContext) {}

// ExitCodeSelectorTerm is called when production codeSelectorTerm is exited.
func (s *BasecqlListener) ExitCodeSelectorTerm(ctx *CodeSelectorTermContext) {}

// EnterConceptSelectorTerm is called when production conceptSelectorTerm is entered.
func (s *BasecqlListener) EnterConceptSelectorTerm(ctx *ConceptSelectorTermContext) {}

// ExitConceptSelectorTerm is called when production conceptSelectorTerm is exited.
func (s *BasecqlListener) ExitConceptSelectorTerm(ctx *ConceptSelectorTermContext) {}

// EnterParenthesizedTerm is called when production parenthesizedTerm is entered.
func (s *BasecqlListener) EnterParenthesizedTerm(ctx *ParenthesizedTermContext) {}

// ExitParenthesizedTerm is called when production parenthesizedTerm is exited.
func (s *BasecqlListener) ExitParenthesizedTerm(ctx *ParenthesizedTermContext) {}

// EnterQualifiedMemberInvocation is called when production qualifiedMemberInvocation is entered.
func (s *BasecqlListener) EnterQualifiedMemberInvocation(ctx *QualifiedMemberInvocationContext) {}

// ExitQualifiedMemberInvocation is called when production qualifiedMemberInvocation is exited.
func (s *BasecqlListener) ExitQualifiedMemberInvocation(ctx *QualifiedMemberInvocationContext) {}

// EnterQualifiedFunctionInvocation is called when production qualifiedFunctionInvocation is entered.
func (s *BasecqlListener) EnterQualifiedFunctionInvocation(ctx *QualifiedFunctionInvocationContext) {}

// ExitQualifiedFunctionInvocation is called when production qualifiedFunctionInvocation is exited.
func (s *BasecqlListener) ExitQualifiedFunctionInvocation(ctx *QualifiedFunctionInvocationContext) {}

// EnterQualifiedFunction is called when production qualifiedFunction is entered.
func (s *BasecqlListener) EnterQualifiedFunction(ctx *QualifiedFunctionContext) {}

// ExitQualifiedFunction is called when production qualifiedFunction is exited.
func (s *BasecqlListener) ExitQualifiedFunction(ctx *QualifiedFunctionContext) {}

// EnterMemberInvocation is called when production memberInvocation is entered.
func (s *BasecqlListener) EnterMemberInvocation(ctx *MemberInvocationContext) {}

// ExitMemberInvocation is called when production memberInvocation is exited.
func (s *BasecqlListener) ExitMemberInvocation(ctx *MemberInvocationContext) {}

// EnterFunctionInvocation is called when production functionInvocation is entered.
func (s *BasecqlListener) EnterFunctionInvocation(ctx *FunctionInvocationContext) {}

// ExitFunctionInvocation is called when production functionInvocation is exited.
func (s *BasecqlListener) ExitFunctionInvocation(ctx *FunctionInvocationContext) {}

// EnterThisInvocation is called when production thisInvocation is entered.
func (s *BasecqlListener) EnterThisInvocation(ctx *ThisInvocationContext) {}

// ExitThisInvocation is called when production thisInvocation is exited.
func (s *BasecqlListener) ExitThisInvocation(ctx *ThisInvocationContext) {}

// EnterIndexInvocation is called when production indexInvocation is entered.
func (s *BasecqlListener) EnterIndexInvocation(ctx *IndexInvocationContext) {}

// ExitIndexInvocation is called when production indexInvocation is exited.
func (s *BasecqlListener) ExitIndexInvocation(ctx *IndexInvocationContext) {}

// EnterTotalInvocation is called when production totalInvocation is entered.
func (s *BasecqlListener) EnterTotalInvocation(ctx *TotalInvocationContext) {}

// ExitTotalInvocation is called when production totalInvocation is exited.
func (s *BasecqlListener) ExitTotalInvocation(ctx *TotalInvocationContext) {}

// EnterFunction is called when production function is entered.
func (s *BasecqlListener) EnterFunction(ctx *FunctionContext) {}

// ExitFunction is called when production function is exited.
func (s *BasecqlListener) ExitFunction(ctx *FunctionContext) {}

// EnterRatio is called when production ratio is entered.
func (s *BasecqlListener) EnterRatio(ctx *RatioContext) {}

// ExitRatio is called when production ratio is exited.
func (s *BasecqlListener) ExitRatio(ctx *RatioContext) {}

// EnterBooleanLiteral is called when production booleanLiteral is entered.
func (s *BasecqlListener) EnterBooleanLiteral(ctx *BooleanLiteralContext) {}

// ExitBooleanLiteral is called when production booleanLiteral is exited.
func (s *BasecqlListener) ExitBooleanLiteral(ctx *BooleanLiteralContext) {}

// EnterNullLiteral is called when production nullLiteral is entered.
func (s *BasecqlListener) EnterNullLiteral(ctx *NullLiteralContext) {}

// ExitNullLiteral is called when production nullLiteral is exited.
func (s *BasecqlListener) ExitNullLiteral(ctx *NullLiteralContext) {}

// EnterStringLiteral is called when production stringLiteral is entered.
func (s *BasecqlListener) EnterStringLiteral(ctx *StringLiteralContext) {}

// ExitStringLiteral is called when production stringLiteral is exited.
func (s *BasecqlListener) ExitStringLiteral(ctx *StringLiteralContext) {}

// EnterNumberLiteral is called when production numberLiteral is entered.
func (s *BasecqlListener) EnterNumberLiteral(ctx *NumberLiteralContext) {}

// ExitNumberLiteral is called when production numberLiteral is exited.
func (s *BasecqlListener) ExitNumberLiteral(ctx *NumberLiteralContext) {}

// EnterLongNumberLiteral is called when production longNumberLiteral is entered.
func (s *BasecqlListener) EnterLongNumberLiteral(ctx *LongNumberLiteralContext) {}

// ExitLongNumberLiteral is called when production longNumberLiteral is exited.
func (s *BasecqlListener) ExitLongNumberLiteral(ctx *LongNumberLiteralContext) {}

// EnterDateTimeLiteral is called when production dateTimeLiteral is entered.
func (s *BasecqlListener) EnterDateTimeLiteral(ctx *DateTimeLiteralContext) {}

// ExitDateTimeLiteral is called when production dateTimeLiteral is exited.
func (s *BasecqlListener) ExitDateTimeLiteral(ctx *DateTimeLiteralContext) {}

// EnterDateLiteral is called when production dateLiteral is entered.
func (s *BasecqlListener) EnterDateLiteral(ctx *DateLiteralContext) {}

// ExitDateLiteral is called when production dateLiteral is exited.
func (s *BasecqlListener) ExitDateLiteral(ctx *DateLiteralContext) {}

// EnterTimeLiteral is called when production timeLiteral is entered.
func (s *BasecqlListener) EnterTimeLiteral(ctx *TimeLiteralContext) {}

// ExitTimeLiteral is called when production timeLiteral is exited.
func (s *BasecqlListener) ExitTimeLiteral(ctx *TimeLiteralContext) {}

// EnterQuantityLiteral is called when production quantityLiteral is entered.
func (s *BasecqlListener) EnterQuantityLiteral(ctx *QuantityLiteralContext) {}

// ExitQuantityLiteral is called when production quantityLiteral is exited.
func (s *BasecqlListener) ExitQuantityLiteral(ctx *QuantityLiteralContext) {}

// EnterRatioLiteral is called when production ratioLiteral is entered.
func (s *BasecqlListener) EnterRatioLiteral(ctx *RatioLiteralContext) {}

// ExitRatioLiteral is called when production ratioLiteral is exited.
func (s *BasecqlListener) ExitRatioLiteral(ctx *RatioLiteralContext) {}

// EnterExternalConstant is called when production externalConstant is entered.
func (s *BasecqlListener) EnterExternalConstant(ctx *ExternalConstantContext) {}

// ExitExternalConstant is called when production externalConstant is exited.
func (s *BasecqlListener) ExitExternalConstant(ctx *ExternalConstantContext) {}

// EnterIntervalSelector is called when production intervalSelector is entered.
func (s *BasecqlListener) EnterIntervalSelector(ctx *IntervalSelectorContext) {}

// ExitIntervalSelector is called when production intervalSelector is exited.
func (s *BasecqlListener) ExitIntervalSelector(ctx *IntervalSelectorContext) {}

// EnterTupleSelector is called when production tupleSelector is entered.
func (s *BasecqlListener) EnterTupleSelector(ctx *TupleSelectorContext) {}

// ExitTupleSelector is called when production tupleSelector is exited.
func (s *BasecqlListener) ExitTupleSelector(ctx *TupleSelectorContext) {}

// EnterTupleElementSelector is called when production tupleElementSelector is entered.
func (s *BasecqlListener) EnterTupleElementSelector(ctx *TupleElementSelectorContext) {}

// ExitTupleElementSelector is called when production tupleElementSelector is exited.
func (s *BasecqlListener) ExitTupleElementSelector(ctx *TupleElementSelectorContext) {}

// EnterInstanceSelector is called when production instanceSelector is entered.
func (s *BasecqlListener) EnterInstanceSelector(ctx *InstanceSelectorContext) {}

// ExitInstanceSelector is called when production instanceSelector is exited.
func (s *BasecqlListener) ExitInstanceSelector(ctx *InstanceSelectorContext) {}

// EnterInstanceElementSelector is called when production instanceElementSelector is entered.
func (s *BasecqlListener) EnterInstanceElementSelector(ctx *InstanceElementSelectorContext) {}

// ExitInstanceElementSelector is called when production instanceElementSelector is exited.
func (s *BasecqlListener) ExitInstanceElementSelector(ctx *InstanceElementSelectorContext) {}

// EnterListSelector is called when production listSelector is entered.
func (s *BasecqlListener) EnterListSelector(ctx *ListSelectorContext) {}

// ExitListSelector is called when production listSelector is exited.
func (s *BasecqlListener) ExitListSelector(ctx *ListSelectorContext) {}

// EnterDisplayClause is called when production displayClause is entered.
func (s *BasecqlListener) EnterDisplayClause(ctx *DisplayClauseContext) {}

// ExitDisplayClause is called when production displayClause is exited.
func (s *BasecqlListener) ExitDisplayClause(ctx *DisplayClauseContext) {}

// EnterCodeSelector is called when production codeSelector is entered.
func (s *BasecqlListener) EnterCodeSelector(ctx *CodeSelectorContext) {}

// ExitCodeSelector is called when production codeSelector is exited.
func (s *BasecqlListener) ExitCodeSelector(ctx *CodeSelectorContext) {}

// EnterConceptSelector is called when production conceptSelector is entered.
func (s *BasecqlListener) EnterConceptSelector(ctx *ConceptSelectorContext) {}

// ExitConceptSelector is called when production conceptSelector is exited.
func (s *BasecqlListener) ExitConceptSelector(ctx *ConceptSelectorContext) {}

// EnterKeyword is called when production keyword is entered.
func (s *BasecqlListener) EnterKeyword(ctx *KeywordContext) {}

// ExitKeyword is called when production keyword is exited.
func (s *BasecqlListener) ExitKeyword(ctx *KeywordContext) {}

// EnterReservedWord is called when production reservedWord is entered.
func (s *BasecqlListener) EnterReservedWord(ctx *ReservedWordContext) {}

// ExitReservedWord is called when production reservedWord is exited.
func (s *BasecqlListener) ExitReservedWord(ctx *ReservedWordContext) {}

// EnterKeywordIdentifier is called when production keywordIdentifier is entered.
func (s *BasecqlListener) EnterKeywordIdentifier(ctx *KeywordIdentifierContext) {}

// ExitKeywordIdentifier is called when production keywordIdentifier is exited.
func (s *BasecqlListener) ExitKeywordIdentifier(ctx *KeywordIdentifierContext) {}

// EnterObsoleteIdentifier is called when production obsoleteIdentifier is entered.
func (s *BasecqlListener) EnterObsoleteIdentifier(ctx *ObsoleteIdentifierContext) {}

// ExitObsoleteIdentifier is called when production obsoleteIdentifier is exited.
func (s *BasecqlListener) ExitObsoleteIdentifier(ctx *ObsoleteIdentifierContext) {}

// EnterFunctionIdentifier is called when production functionIdentifier is entered.
func (s *BasecqlListener) EnterFunctionIdentifier(ctx *FunctionIdentifierContext) {}

// ExitFunctionIdentifier is called when production functionIdentifier is exited.
func (s *BasecqlListener) ExitFunctionIdentifier(ctx *FunctionIdentifierContext) {}

// EnterTypeNameIdentifier is called when production typeNameIdentifier is entered.
func (s *BasecqlListener) EnterTypeNameIdentifier(ctx *TypeNameIdentifierContext) {}

// ExitTypeNameIdentifier is called when production typeNameIdentifier is exited.
func (s *BasecqlListener) ExitTypeNameIdentifier(ctx *TypeNameIdentifierContext) {}

// EnterReferentialIdentifier is called when production referentialIdentifier is entered.
func (s *BasecqlListener) EnterReferentialIdentifier(ctx *ReferentialIdentifierContext) {}

// ExitReferentialIdentifier is called when production referentialIdentifier is exited.
func (s *BasecqlListener) ExitReferentialIdentifier(ctx *ReferentialIdentifierContext) {}

// EnterReferentialOrTypeNameIdentifier is called when production referentialOrTypeNameIdentifier is entered.
func (s *BasecqlListener) EnterReferentialOrTypeNameIdentifier(ctx *ReferentialOrTypeNameIdentifierContext) {
}

// ExitReferentialOrTypeNameIdentifier is called when production referentialOrTypeNameIdentifier is exited.
func (s *BasecqlListener) ExitReferentialOrTypeNameIdentifier(ctx *ReferentialOrTypeNameIdentifierContext) {
}

// EnterIdentifierOrFunctionIdentifier is called when production identifierOrFunctionIdentifier is entered.
func (s *BasecqlListener) EnterIdentifierOrFunctionIdentifier(ctx *IdentifierOrFunctionIdentifierContext) {
}

// ExitIdentifierOrFunctionIdentifier is called when production identifierOrFunctionIdentifier is exited.
func (s *BasecqlListener) ExitIdentifierOrFunctionIdentifier(ctx *IdentifierOrFunctionIdentifierContext) {
}

// EnterIdentifier is called when production identifier is entered.
func (s *BasecqlListener) EnterIdentifier(ctx *IdentifierContext) {}

// ExitIdentifier is called when production identifier is exited.
func (s *BasecqlListener) ExitIdentifier(ctx *IdentifierContext) {}

// EnterParamList is called when production paramList is entered.
func (s *BasecqlListener) EnterParamList(ctx *ParamListContext) {}

// ExitParamList is called when production paramList is exited.
func (s *BasecqlListener) ExitParamList(ctx *ParamListContext) {}

// EnterQuantity is called when production quantity is entered.
func (s *BasecqlListener) EnterQuantity(ctx *QuantityContext) {}

// ExitQuantity is called when production quantity is exited.
func (s *BasecqlListener) ExitQuantity(ctx *QuantityContext) {}

// EnterUnit is called when production unit is entered.
func (s *BasecqlListener) EnterUnit(ctx *UnitContext) {}

// ExitUnit is called when production unit is exited.
func (s *BasecqlListener) ExitUnit(ctx *UnitContext) {}
