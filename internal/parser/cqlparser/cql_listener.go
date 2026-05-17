// Code generated from cql.g4 by ANTLR 4.13.2. DO NOT EDIT.

package cqlparser // cql
import "github.com/antlr4-go/antlr/v4"

// cqlListener is a complete listener for a parse tree produced by cqlParser.
type cqlListener interface {
	antlr.ParseTreeListener

	// EnterDefinition is called when entering the definition production.
	EnterDefinition(c *DefinitionContext)

	// EnterLibrary is called when entering the library production.
	EnterLibrary(c *LibraryContext)

	// EnterLibraryDefinition is called when entering the libraryDefinition production.
	EnterLibraryDefinition(c *LibraryDefinitionContext)

	// EnterUsingDefinition is called when entering the usingDefinition production.
	EnterUsingDefinition(c *UsingDefinitionContext)

	// EnterIncludeDefinition is called when entering the includeDefinition production.
	EnterIncludeDefinition(c *IncludeDefinitionContext)

	// EnterLocalIdentifier is called when entering the localIdentifier production.
	EnterLocalIdentifier(c *LocalIdentifierContext)

	// EnterAccessModifier is called when entering the accessModifier production.
	EnterAccessModifier(c *AccessModifierContext)

	// EnterParameterDefinition is called when entering the parameterDefinition production.
	EnterParameterDefinition(c *ParameterDefinitionContext)

	// EnterCodesystemDefinition is called when entering the codesystemDefinition production.
	EnterCodesystemDefinition(c *CodesystemDefinitionContext)

	// EnterValuesetDefinition is called when entering the valuesetDefinition production.
	EnterValuesetDefinition(c *ValuesetDefinitionContext)

	// EnterCodesystems is called when entering the codesystems production.
	EnterCodesystems(c *CodesystemsContext)

	// EnterCodesystemIdentifier is called when entering the codesystemIdentifier production.
	EnterCodesystemIdentifier(c *CodesystemIdentifierContext)

	// EnterLibraryIdentifier is called when entering the libraryIdentifier production.
	EnterLibraryIdentifier(c *LibraryIdentifierContext)

	// EnterCodeDefinition is called when entering the codeDefinition production.
	EnterCodeDefinition(c *CodeDefinitionContext)

	// EnterConceptDefinition is called when entering the conceptDefinition production.
	EnterConceptDefinition(c *ConceptDefinitionContext)

	// EnterCodeIdentifier is called when entering the codeIdentifier production.
	EnterCodeIdentifier(c *CodeIdentifierContext)

	// EnterCodesystemId is called when entering the codesystemId production.
	EnterCodesystemId(c *CodesystemIdContext)

	// EnterValuesetId is called when entering the valuesetId production.
	EnterValuesetId(c *ValuesetIdContext)

	// EnterVersionSpecifier is called when entering the versionSpecifier production.
	EnterVersionSpecifier(c *VersionSpecifierContext)

	// EnterCodeId is called when entering the codeId production.
	EnterCodeId(c *CodeIdContext)

	// EnterTypeSpecifier is called when entering the typeSpecifier production.
	EnterTypeSpecifier(c *TypeSpecifierContext)

	// EnterNamedTypeSpecifier is called when entering the namedTypeSpecifier production.
	EnterNamedTypeSpecifier(c *NamedTypeSpecifierContext)

	// EnterModelIdentifier is called when entering the modelIdentifier production.
	EnterModelIdentifier(c *ModelIdentifierContext)

	// EnterListTypeSpecifier is called when entering the listTypeSpecifier production.
	EnterListTypeSpecifier(c *ListTypeSpecifierContext)

	// EnterIntervalTypeSpecifier is called when entering the intervalTypeSpecifier production.
	EnterIntervalTypeSpecifier(c *IntervalTypeSpecifierContext)

	// EnterTupleTypeSpecifier is called when entering the tupleTypeSpecifier production.
	EnterTupleTypeSpecifier(c *TupleTypeSpecifierContext)

	// EnterTupleElementDefinition is called when entering the tupleElementDefinition production.
	EnterTupleElementDefinition(c *TupleElementDefinitionContext)

	// EnterChoiceTypeSpecifier is called when entering the choiceTypeSpecifier production.
	EnterChoiceTypeSpecifier(c *ChoiceTypeSpecifierContext)

	// EnterStatement is called when entering the statement production.
	EnterStatement(c *StatementContext)

	// EnterExpressionDefinition is called when entering the expressionDefinition production.
	EnterExpressionDefinition(c *ExpressionDefinitionContext)

	// EnterContextDefinition is called when entering the contextDefinition production.
	EnterContextDefinition(c *ContextDefinitionContext)

	// EnterFluentModifier is called when entering the fluentModifier production.
	EnterFluentModifier(c *FluentModifierContext)

	// EnterFunctionDefinition is called when entering the functionDefinition production.
	EnterFunctionDefinition(c *FunctionDefinitionContext)

	// EnterOperandDefinition is called when entering the operandDefinition production.
	EnterOperandDefinition(c *OperandDefinitionContext)

	// EnterFunctionBody is called when entering the functionBody production.
	EnterFunctionBody(c *FunctionBodyContext)

	// EnterQuerySource is called when entering the querySource production.
	EnterQuerySource(c *QuerySourceContext)

	// EnterAliasedQuerySource is called when entering the aliasedQuerySource production.
	EnterAliasedQuerySource(c *AliasedQuerySourceContext)

	// EnterAlias is called when entering the alias production.
	EnterAlias(c *AliasContext)

	// EnterQueryInclusionClause is called when entering the queryInclusionClause production.
	EnterQueryInclusionClause(c *QueryInclusionClauseContext)

	// EnterWithClause is called when entering the withClause production.
	EnterWithClause(c *WithClauseContext)

	// EnterWithoutClause is called when entering the withoutClause production.
	EnterWithoutClause(c *WithoutClauseContext)

	// EnterRetrieve is called when entering the retrieve production.
	EnterRetrieve(c *RetrieveContext)

	// EnterContextIdentifier is called when entering the contextIdentifier production.
	EnterContextIdentifier(c *ContextIdentifierContext)

	// EnterCodePath is called when entering the codePath production.
	EnterCodePath(c *CodePathContext)

	// EnterCodeComparator is called when entering the codeComparator production.
	EnterCodeComparator(c *CodeComparatorContext)

	// EnterTerminology is called when entering the terminology production.
	EnterTerminology(c *TerminologyContext)

	// EnterQualifier is called when entering the qualifier production.
	EnterQualifier(c *QualifierContext)

	// EnterQuery is called when entering the query production.
	EnterQuery(c *QueryContext)

	// EnterSourceClause is called when entering the sourceClause production.
	EnterSourceClause(c *SourceClauseContext)

	// EnterLetClause is called when entering the letClause production.
	EnterLetClause(c *LetClauseContext)

	// EnterLetClauseItem is called when entering the letClauseItem production.
	EnterLetClauseItem(c *LetClauseItemContext)

	// EnterWhereClause is called when entering the whereClause production.
	EnterWhereClause(c *WhereClauseContext)

	// EnterReturnClause is called when entering the returnClause production.
	EnterReturnClause(c *ReturnClauseContext)

	// EnterAggregateClause is called when entering the aggregateClause production.
	EnterAggregateClause(c *AggregateClauseContext)

	// EnterStartingClause is called when entering the startingClause production.
	EnterStartingClause(c *StartingClauseContext)

	// EnterSortClause is called when entering the sortClause production.
	EnterSortClause(c *SortClauseContext)

	// EnterSortDirection is called when entering the sortDirection production.
	EnterSortDirection(c *SortDirectionContext)

	// EnterSortByItem is called when entering the sortByItem production.
	EnterSortByItem(c *SortByItemContext)

	// EnterQualifiedIdentifier is called when entering the qualifiedIdentifier production.
	EnterQualifiedIdentifier(c *QualifiedIdentifierContext)

	// EnterQualifiedIdentifierExpression is called when entering the qualifiedIdentifierExpression production.
	EnterQualifiedIdentifierExpression(c *QualifiedIdentifierExpressionContext)

	// EnterQualifierExpression is called when entering the qualifierExpression production.
	EnterQualifierExpression(c *QualifierExpressionContext)

	// EnterSimplePathIndexer is called when entering the simplePathIndexer production.
	EnterSimplePathIndexer(c *SimplePathIndexerContext)

	// EnterSimplePathQualifiedIdentifier is called when entering the simplePathQualifiedIdentifier production.
	EnterSimplePathQualifiedIdentifier(c *SimplePathQualifiedIdentifierContext)

	// EnterSimplePathReferentialIdentifier is called when entering the simplePathReferentialIdentifier production.
	EnterSimplePathReferentialIdentifier(c *SimplePathReferentialIdentifierContext)

	// EnterSimpleStringLiteral is called when entering the simpleStringLiteral production.
	EnterSimpleStringLiteral(c *SimpleStringLiteralContext)

	// EnterSimpleNumberLiteral is called when entering the simpleNumberLiteral production.
	EnterSimpleNumberLiteral(c *SimpleNumberLiteralContext)

	// EnterDurationBetweenExpression is called when entering the durationBetweenExpression production.
	EnterDurationBetweenExpression(c *DurationBetweenExpressionContext)

	// EnterInFixSetExpression is called when entering the inFixSetExpression production.
	EnterInFixSetExpression(c *InFixSetExpressionContext)

	// EnterRetrieveExpression is called when entering the retrieveExpression production.
	EnterRetrieveExpression(c *RetrieveExpressionContext)

	// EnterTimingExpression is called when entering the timingExpression production.
	EnterTimingExpression(c *TimingExpressionContext)

	// EnterQueryExpression is called when entering the queryExpression production.
	EnterQueryExpression(c *QueryExpressionContext)

	// EnterNotExpression is called when entering the notExpression production.
	EnterNotExpression(c *NotExpressionContext)

	// EnterBooleanExpression is called when entering the booleanExpression production.
	EnterBooleanExpression(c *BooleanExpressionContext)

	// EnterOrExpression is called when entering the orExpression production.
	EnterOrExpression(c *OrExpressionContext)

	// EnterCastExpression is called when entering the castExpression production.
	EnterCastExpression(c *CastExpressionContext)

	// EnterAndExpression is called when entering the andExpression production.
	EnterAndExpression(c *AndExpressionContext)

	// EnterBetweenExpression is called when entering the betweenExpression production.
	EnterBetweenExpression(c *BetweenExpressionContext)

	// EnterMembershipExpression is called when entering the membershipExpression production.
	EnterMembershipExpression(c *MembershipExpressionContext)

	// EnterDifferenceBetweenExpression is called when entering the differenceBetweenExpression production.
	EnterDifferenceBetweenExpression(c *DifferenceBetweenExpressionContext)

	// EnterInequalityExpression is called when entering the inequalityExpression production.
	EnterInequalityExpression(c *InequalityExpressionContext)

	// EnterEqualityExpression is called when entering the equalityExpression production.
	EnterEqualityExpression(c *EqualityExpressionContext)

	// EnterExistenceExpression is called when entering the existenceExpression production.
	EnterExistenceExpression(c *ExistenceExpressionContext)

	// EnterImpliesExpression is called when entering the impliesExpression production.
	EnterImpliesExpression(c *ImpliesExpressionContext)

	// EnterTermExpression is called when entering the termExpression production.
	EnterTermExpression(c *TermExpressionContext)

	// EnterTypeExpression is called when entering the typeExpression production.
	EnterTypeExpression(c *TypeExpressionContext)

	// EnterDateTimePrecision is called when entering the dateTimePrecision production.
	EnterDateTimePrecision(c *DateTimePrecisionContext)

	// EnterDateTimeComponent is called when entering the dateTimeComponent production.
	EnterDateTimeComponent(c *DateTimeComponentContext)

	// EnterPluralDateTimePrecision is called when entering the pluralDateTimePrecision production.
	EnterPluralDateTimePrecision(c *PluralDateTimePrecisionContext)

	// EnterAdditionExpressionTerm is called when entering the additionExpressionTerm production.
	EnterAdditionExpressionTerm(c *AdditionExpressionTermContext)

	// EnterIndexedExpressionTerm is called when entering the indexedExpressionTerm production.
	EnterIndexedExpressionTerm(c *IndexedExpressionTermContext)

	// EnterWidthExpressionTerm is called when entering the widthExpressionTerm production.
	EnterWidthExpressionTerm(c *WidthExpressionTermContext)

	// EnterSetAggregateExpressionTerm is called when entering the setAggregateExpressionTerm production.
	EnterSetAggregateExpressionTerm(c *SetAggregateExpressionTermContext)

	// EnterTimeUnitExpressionTerm is called when entering the timeUnitExpressionTerm production.
	EnterTimeUnitExpressionTerm(c *TimeUnitExpressionTermContext)

	// EnterIfThenElseExpressionTerm is called when entering the ifThenElseExpressionTerm production.
	EnterIfThenElseExpressionTerm(c *IfThenElseExpressionTermContext)

	// EnterTimeBoundaryExpressionTerm is called when entering the timeBoundaryExpressionTerm production.
	EnterTimeBoundaryExpressionTerm(c *TimeBoundaryExpressionTermContext)

	// EnterElementExtractorExpressionTerm is called when entering the elementExtractorExpressionTerm production.
	EnterElementExtractorExpressionTerm(c *ElementExtractorExpressionTermContext)

	// EnterConversionExpressionTerm is called when entering the conversionExpressionTerm production.
	EnterConversionExpressionTerm(c *ConversionExpressionTermContext)

	// EnterTypeExtentExpressionTerm is called when entering the typeExtentExpressionTerm production.
	EnterTypeExtentExpressionTerm(c *TypeExtentExpressionTermContext)

	// EnterPredecessorExpressionTerm is called when entering the predecessorExpressionTerm production.
	EnterPredecessorExpressionTerm(c *PredecessorExpressionTermContext)

	// EnterPointExtractorExpressionTerm is called when entering the pointExtractorExpressionTerm production.
	EnterPointExtractorExpressionTerm(c *PointExtractorExpressionTermContext)

	// EnterMultiplicationExpressionTerm is called when entering the multiplicationExpressionTerm production.
	EnterMultiplicationExpressionTerm(c *MultiplicationExpressionTermContext)

	// EnterAggregateExpressionTerm is called when entering the aggregateExpressionTerm production.
	EnterAggregateExpressionTerm(c *AggregateExpressionTermContext)

	// EnterDurationExpressionTerm is called when entering the durationExpressionTerm production.
	EnterDurationExpressionTerm(c *DurationExpressionTermContext)

	// EnterDifferenceExpressionTerm is called when entering the differenceExpressionTerm production.
	EnterDifferenceExpressionTerm(c *DifferenceExpressionTermContext)

	// EnterCaseExpressionTerm is called when entering the caseExpressionTerm production.
	EnterCaseExpressionTerm(c *CaseExpressionTermContext)

	// EnterPowerExpressionTerm is called when entering the powerExpressionTerm production.
	EnterPowerExpressionTerm(c *PowerExpressionTermContext)

	// EnterSuccessorExpressionTerm is called when entering the successorExpressionTerm production.
	EnterSuccessorExpressionTerm(c *SuccessorExpressionTermContext)

	// EnterPolarityExpressionTerm is called when entering the polarityExpressionTerm production.
	EnterPolarityExpressionTerm(c *PolarityExpressionTermContext)

	// EnterTermExpressionTerm is called when entering the termExpressionTerm production.
	EnterTermExpressionTerm(c *TermExpressionTermContext)

	// EnterInvocationExpressionTerm is called when entering the invocationExpressionTerm production.
	EnterInvocationExpressionTerm(c *InvocationExpressionTermContext)

	// EnterCaseExpressionItem is called when entering the caseExpressionItem production.
	EnterCaseExpressionItem(c *CaseExpressionItemContext)

	// EnterDateTimePrecisionSpecifier is called when entering the dateTimePrecisionSpecifier production.
	EnterDateTimePrecisionSpecifier(c *DateTimePrecisionSpecifierContext)

	// EnterRelativeQualifier is called when entering the relativeQualifier production.
	EnterRelativeQualifier(c *RelativeQualifierContext)

	// EnterOffsetRelativeQualifier is called when entering the offsetRelativeQualifier production.
	EnterOffsetRelativeQualifier(c *OffsetRelativeQualifierContext)

	// EnterExclusiveRelativeQualifier is called when entering the exclusiveRelativeQualifier production.
	EnterExclusiveRelativeQualifier(c *ExclusiveRelativeQualifierContext)

	// EnterQuantityOffset is called when entering the quantityOffset production.
	EnterQuantityOffset(c *QuantityOffsetContext)

	// EnterTemporalRelationship is called when entering the temporalRelationship production.
	EnterTemporalRelationship(c *TemporalRelationshipContext)

	// EnterConcurrentWithIntervalOperatorPhrase is called when entering the concurrentWithIntervalOperatorPhrase production.
	EnterConcurrentWithIntervalOperatorPhrase(c *ConcurrentWithIntervalOperatorPhraseContext)

	// EnterIncludesIntervalOperatorPhrase is called when entering the includesIntervalOperatorPhrase production.
	EnterIncludesIntervalOperatorPhrase(c *IncludesIntervalOperatorPhraseContext)

	// EnterIncludedInIntervalOperatorPhrase is called when entering the includedInIntervalOperatorPhrase production.
	EnterIncludedInIntervalOperatorPhrase(c *IncludedInIntervalOperatorPhraseContext)

	// EnterBeforeOrAfterIntervalOperatorPhrase is called when entering the beforeOrAfterIntervalOperatorPhrase production.
	EnterBeforeOrAfterIntervalOperatorPhrase(c *BeforeOrAfterIntervalOperatorPhraseContext)

	// EnterWithinIntervalOperatorPhrase is called when entering the withinIntervalOperatorPhrase production.
	EnterWithinIntervalOperatorPhrase(c *WithinIntervalOperatorPhraseContext)

	// EnterMeetsIntervalOperatorPhrase is called when entering the meetsIntervalOperatorPhrase production.
	EnterMeetsIntervalOperatorPhrase(c *MeetsIntervalOperatorPhraseContext)

	// EnterOverlapsIntervalOperatorPhrase is called when entering the overlapsIntervalOperatorPhrase production.
	EnterOverlapsIntervalOperatorPhrase(c *OverlapsIntervalOperatorPhraseContext)

	// EnterStartsIntervalOperatorPhrase is called when entering the startsIntervalOperatorPhrase production.
	EnterStartsIntervalOperatorPhrase(c *StartsIntervalOperatorPhraseContext)

	// EnterEndsIntervalOperatorPhrase is called when entering the endsIntervalOperatorPhrase production.
	EnterEndsIntervalOperatorPhrase(c *EndsIntervalOperatorPhraseContext)

	// EnterInvocationTerm is called when entering the invocationTerm production.
	EnterInvocationTerm(c *InvocationTermContext)

	// EnterLiteralTerm is called when entering the literalTerm production.
	EnterLiteralTerm(c *LiteralTermContext)

	// EnterExternalConstantTerm is called when entering the externalConstantTerm production.
	EnterExternalConstantTerm(c *ExternalConstantTermContext)

	// EnterIntervalSelectorTerm is called when entering the intervalSelectorTerm production.
	EnterIntervalSelectorTerm(c *IntervalSelectorTermContext)

	// EnterTupleSelectorTerm is called when entering the tupleSelectorTerm production.
	EnterTupleSelectorTerm(c *TupleSelectorTermContext)

	// EnterInstanceSelectorTerm is called when entering the instanceSelectorTerm production.
	EnterInstanceSelectorTerm(c *InstanceSelectorTermContext)

	// EnterListSelectorTerm is called when entering the listSelectorTerm production.
	EnterListSelectorTerm(c *ListSelectorTermContext)

	// EnterCodeSelectorTerm is called when entering the codeSelectorTerm production.
	EnterCodeSelectorTerm(c *CodeSelectorTermContext)

	// EnterConceptSelectorTerm is called when entering the conceptSelectorTerm production.
	EnterConceptSelectorTerm(c *ConceptSelectorTermContext)

	// EnterParenthesizedTerm is called when entering the parenthesizedTerm production.
	EnterParenthesizedTerm(c *ParenthesizedTermContext)

	// EnterQualifiedMemberInvocation is called when entering the qualifiedMemberInvocation production.
	EnterQualifiedMemberInvocation(c *QualifiedMemberInvocationContext)

	// EnterQualifiedFunctionInvocation is called when entering the qualifiedFunctionInvocation production.
	EnterQualifiedFunctionInvocation(c *QualifiedFunctionInvocationContext)

	// EnterQualifiedFunction is called when entering the qualifiedFunction production.
	EnterQualifiedFunction(c *QualifiedFunctionContext)

	// EnterMemberInvocation is called when entering the memberInvocation production.
	EnterMemberInvocation(c *MemberInvocationContext)

	// EnterFunctionInvocation is called when entering the functionInvocation production.
	EnterFunctionInvocation(c *FunctionInvocationContext)

	// EnterThisInvocation is called when entering the thisInvocation production.
	EnterThisInvocation(c *ThisInvocationContext)

	// EnterIndexInvocation is called when entering the indexInvocation production.
	EnterIndexInvocation(c *IndexInvocationContext)

	// EnterTotalInvocation is called when entering the totalInvocation production.
	EnterTotalInvocation(c *TotalInvocationContext)

	// EnterFunction is called when entering the function production.
	EnterFunction(c *FunctionContext)

	// EnterRatio is called when entering the ratio production.
	EnterRatio(c *RatioContext)

	// EnterBooleanLiteral is called when entering the booleanLiteral production.
	EnterBooleanLiteral(c *BooleanLiteralContext)

	// EnterNullLiteral is called when entering the nullLiteral production.
	EnterNullLiteral(c *NullLiteralContext)

	// EnterStringLiteral is called when entering the stringLiteral production.
	EnterStringLiteral(c *StringLiteralContext)

	// EnterNumberLiteral is called when entering the numberLiteral production.
	EnterNumberLiteral(c *NumberLiteralContext)

	// EnterLongNumberLiteral is called when entering the longNumberLiteral production.
	EnterLongNumberLiteral(c *LongNumberLiteralContext)

	// EnterDateTimeLiteral is called when entering the dateTimeLiteral production.
	EnterDateTimeLiteral(c *DateTimeLiteralContext)

	// EnterDateLiteral is called when entering the dateLiteral production.
	EnterDateLiteral(c *DateLiteralContext)

	// EnterTimeLiteral is called when entering the timeLiteral production.
	EnterTimeLiteral(c *TimeLiteralContext)

	// EnterQuantityLiteral is called when entering the quantityLiteral production.
	EnterQuantityLiteral(c *QuantityLiteralContext)

	// EnterRatioLiteral is called when entering the ratioLiteral production.
	EnterRatioLiteral(c *RatioLiteralContext)

	// EnterExternalConstant is called when entering the externalConstant production.
	EnterExternalConstant(c *ExternalConstantContext)

	// EnterIntervalSelector is called when entering the intervalSelector production.
	EnterIntervalSelector(c *IntervalSelectorContext)

	// EnterTupleSelector is called when entering the tupleSelector production.
	EnterTupleSelector(c *TupleSelectorContext)

	// EnterTupleElementSelector is called when entering the tupleElementSelector production.
	EnterTupleElementSelector(c *TupleElementSelectorContext)

	// EnterInstanceSelector is called when entering the instanceSelector production.
	EnterInstanceSelector(c *InstanceSelectorContext)

	// EnterInstanceElementSelector is called when entering the instanceElementSelector production.
	EnterInstanceElementSelector(c *InstanceElementSelectorContext)

	// EnterListSelector is called when entering the listSelector production.
	EnterListSelector(c *ListSelectorContext)

	// EnterDisplayClause is called when entering the displayClause production.
	EnterDisplayClause(c *DisplayClauseContext)

	// EnterCodeSelector is called when entering the codeSelector production.
	EnterCodeSelector(c *CodeSelectorContext)

	// EnterConceptSelector is called when entering the conceptSelector production.
	EnterConceptSelector(c *ConceptSelectorContext)

	// EnterKeyword is called when entering the keyword production.
	EnterKeyword(c *KeywordContext)

	// EnterReservedWord is called when entering the reservedWord production.
	EnterReservedWord(c *ReservedWordContext)

	// EnterKeywordIdentifier is called when entering the keywordIdentifier production.
	EnterKeywordIdentifier(c *KeywordIdentifierContext)

	// EnterObsoleteIdentifier is called when entering the obsoleteIdentifier production.
	EnterObsoleteIdentifier(c *ObsoleteIdentifierContext)

	// EnterFunctionIdentifier is called when entering the functionIdentifier production.
	EnterFunctionIdentifier(c *FunctionIdentifierContext)

	// EnterTypeNameIdentifier is called when entering the typeNameIdentifier production.
	EnterTypeNameIdentifier(c *TypeNameIdentifierContext)

	// EnterReferentialIdentifier is called when entering the referentialIdentifier production.
	EnterReferentialIdentifier(c *ReferentialIdentifierContext)

	// EnterReferentialOrTypeNameIdentifier is called when entering the referentialOrTypeNameIdentifier production.
	EnterReferentialOrTypeNameIdentifier(c *ReferentialOrTypeNameIdentifierContext)

	// EnterIdentifierOrFunctionIdentifier is called when entering the identifierOrFunctionIdentifier production.
	EnterIdentifierOrFunctionIdentifier(c *IdentifierOrFunctionIdentifierContext)

	// EnterIdentifier is called when entering the identifier production.
	EnterIdentifier(c *IdentifierContext)

	// EnterParamList is called when entering the paramList production.
	EnterParamList(c *ParamListContext)

	// EnterQuantity is called when entering the quantity production.
	EnterQuantity(c *QuantityContext)

	// EnterUnit is called when entering the unit production.
	EnterUnit(c *UnitContext)

	// ExitDefinition is called when exiting the definition production.
	ExitDefinition(c *DefinitionContext)

	// ExitLibrary is called when exiting the library production.
	ExitLibrary(c *LibraryContext)

	// ExitLibraryDefinition is called when exiting the libraryDefinition production.
	ExitLibraryDefinition(c *LibraryDefinitionContext)

	// ExitUsingDefinition is called when exiting the usingDefinition production.
	ExitUsingDefinition(c *UsingDefinitionContext)

	// ExitIncludeDefinition is called when exiting the includeDefinition production.
	ExitIncludeDefinition(c *IncludeDefinitionContext)

	// ExitLocalIdentifier is called when exiting the localIdentifier production.
	ExitLocalIdentifier(c *LocalIdentifierContext)

	// ExitAccessModifier is called when exiting the accessModifier production.
	ExitAccessModifier(c *AccessModifierContext)

	// ExitParameterDefinition is called when exiting the parameterDefinition production.
	ExitParameterDefinition(c *ParameterDefinitionContext)

	// ExitCodesystemDefinition is called when exiting the codesystemDefinition production.
	ExitCodesystemDefinition(c *CodesystemDefinitionContext)

	// ExitValuesetDefinition is called when exiting the valuesetDefinition production.
	ExitValuesetDefinition(c *ValuesetDefinitionContext)

	// ExitCodesystems is called when exiting the codesystems production.
	ExitCodesystems(c *CodesystemsContext)

	// ExitCodesystemIdentifier is called when exiting the codesystemIdentifier production.
	ExitCodesystemIdentifier(c *CodesystemIdentifierContext)

	// ExitLibraryIdentifier is called when exiting the libraryIdentifier production.
	ExitLibraryIdentifier(c *LibraryIdentifierContext)

	// ExitCodeDefinition is called when exiting the codeDefinition production.
	ExitCodeDefinition(c *CodeDefinitionContext)

	// ExitConceptDefinition is called when exiting the conceptDefinition production.
	ExitConceptDefinition(c *ConceptDefinitionContext)

	// ExitCodeIdentifier is called when exiting the codeIdentifier production.
	ExitCodeIdentifier(c *CodeIdentifierContext)

	// ExitCodesystemId is called when exiting the codesystemId production.
	ExitCodesystemId(c *CodesystemIdContext)

	// ExitValuesetId is called when exiting the valuesetId production.
	ExitValuesetId(c *ValuesetIdContext)

	// ExitVersionSpecifier is called when exiting the versionSpecifier production.
	ExitVersionSpecifier(c *VersionSpecifierContext)

	// ExitCodeId is called when exiting the codeId production.
	ExitCodeId(c *CodeIdContext)

	// ExitTypeSpecifier is called when exiting the typeSpecifier production.
	ExitTypeSpecifier(c *TypeSpecifierContext)

	// ExitNamedTypeSpecifier is called when exiting the namedTypeSpecifier production.
	ExitNamedTypeSpecifier(c *NamedTypeSpecifierContext)

	// ExitModelIdentifier is called when exiting the modelIdentifier production.
	ExitModelIdentifier(c *ModelIdentifierContext)

	// ExitListTypeSpecifier is called when exiting the listTypeSpecifier production.
	ExitListTypeSpecifier(c *ListTypeSpecifierContext)

	// ExitIntervalTypeSpecifier is called when exiting the intervalTypeSpecifier production.
	ExitIntervalTypeSpecifier(c *IntervalTypeSpecifierContext)

	// ExitTupleTypeSpecifier is called when exiting the tupleTypeSpecifier production.
	ExitTupleTypeSpecifier(c *TupleTypeSpecifierContext)

	// ExitTupleElementDefinition is called when exiting the tupleElementDefinition production.
	ExitTupleElementDefinition(c *TupleElementDefinitionContext)

	// ExitChoiceTypeSpecifier is called when exiting the choiceTypeSpecifier production.
	ExitChoiceTypeSpecifier(c *ChoiceTypeSpecifierContext)

	// ExitStatement is called when exiting the statement production.
	ExitStatement(c *StatementContext)

	// ExitExpressionDefinition is called when exiting the expressionDefinition production.
	ExitExpressionDefinition(c *ExpressionDefinitionContext)

	// ExitContextDefinition is called when exiting the contextDefinition production.
	ExitContextDefinition(c *ContextDefinitionContext)

	// ExitFluentModifier is called when exiting the fluentModifier production.
	ExitFluentModifier(c *FluentModifierContext)

	// ExitFunctionDefinition is called when exiting the functionDefinition production.
	ExitFunctionDefinition(c *FunctionDefinitionContext)

	// ExitOperandDefinition is called when exiting the operandDefinition production.
	ExitOperandDefinition(c *OperandDefinitionContext)

	// ExitFunctionBody is called when exiting the functionBody production.
	ExitFunctionBody(c *FunctionBodyContext)

	// ExitQuerySource is called when exiting the querySource production.
	ExitQuerySource(c *QuerySourceContext)

	// ExitAliasedQuerySource is called when exiting the aliasedQuerySource production.
	ExitAliasedQuerySource(c *AliasedQuerySourceContext)

	// ExitAlias is called when exiting the alias production.
	ExitAlias(c *AliasContext)

	// ExitQueryInclusionClause is called when exiting the queryInclusionClause production.
	ExitQueryInclusionClause(c *QueryInclusionClauseContext)

	// ExitWithClause is called when exiting the withClause production.
	ExitWithClause(c *WithClauseContext)

	// ExitWithoutClause is called when exiting the withoutClause production.
	ExitWithoutClause(c *WithoutClauseContext)

	// ExitRetrieve is called when exiting the retrieve production.
	ExitRetrieve(c *RetrieveContext)

	// ExitContextIdentifier is called when exiting the contextIdentifier production.
	ExitContextIdentifier(c *ContextIdentifierContext)

	// ExitCodePath is called when exiting the codePath production.
	ExitCodePath(c *CodePathContext)

	// ExitCodeComparator is called when exiting the codeComparator production.
	ExitCodeComparator(c *CodeComparatorContext)

	// ExitTerminology is called when exiting the terminology production.
	ExitTerminology(c *TerminologyContext)

	// ExitQualifier is called when exiting the qualifier production.
	ExitQualifier(c *QualifierContext)

	// ExitQuery is called when exiting the query production.
	ExitQuery(c *QueryContext)

	// ExitSourceClause is called when exiting the sourceClause production.
	ExitSourceClause(c *SourceClauseContext)

	// ExitLetClause is called when exiting the letClause production.
	ExitLetClause(c *LetClauseContext)

	// ExitLetClauseItem is called when exiting the letClauseItem production.
	ExitLetClauseItem(c *LetClauseItemContext)

	// ExitWhereClause is called when exiting the whereClause production.
	ExitWhereClause(c *WhereClauseContext)

	// ExitReturnClause is called when exiting the returnClause production.
	ExitReturnClause(c *ReturnClauseContext)

	// ExitAggregateClause is called when exiting the aggregateClause production.
	ExitAggregateClause(c *AggregateClauseContext)

	// ExitStartingClause is called when exiting the startingClause production.
	ExitStartingClause(c *StartingClauseContext)

	// ExitSortClause is called when exiting the sortClause production.
	ExitSortClause(c *SortClauseContext)

	// ExitSortDirection is called when exiting the sortDirection production.
	ExitSortDirection(c *SortDirectionContext)

	// ExitSortByItem is called when exiting the sortByItem production.
	ExitSortByItem(c *SortByItemContext)

	// ExitQualifiedIdentifier is called when exiting the qualifiedIdentifier production.
	ExitQualifiedIdentifier(c *QualifiedIdentifierContext)

	// ExitQualifiedIdentifierExpression is called when exiting the qualifiedIdentifierExpression production.
	ExitQualifiedIdentifierExpression(c *QualifiedIdentifierExpressionContext)

	// ExitQualifierExpression is called when exiting the qualifierExpression production.
	ExitQualifierExpression(c *QualifierExpressionContext)

	// ExitSimplePathIndexer is called when exiting the simplePathIndexer production.
	ExitSimplePathIndexer(c *SimplePathIndexerContext)

	// ExitSimplePathQualifiedIdentifier is called when exiting the simplePathQualifiedIdentifier production.
	ExitSimplePathQualifiedIdentifier(c *SimplePathQualifiedIdentifierContext)

	// ExitSimplePathReferentialIdentifier is called when exiting the simplePathReferentialIdentifier production.
	ExitSimplePathReferentialIdentifier(c *SimplePathReferentialIdentifierContext)

	// ExitSimpleStringLiteral is called when exiting the simpleStringLiteral production.
	ExitSimpleStringLiteral(c *SimpleStringLiteralContext)

	// ExitSimpleNumberLiteral is called when exiting the simpleNumberLiteral production.
	ExitSimpleNumberLiteral(c *SimpleNumberLiteralContext)

	// ExitDurationBetweenExpression is called when exiting the durationBetweenExpression production.
	ExitDurationBetweenExpression(c *DurationBetweenExpressionContext)

	// ExitInFixSetExpression is called when exiting the inFixSetExpression production.
	ExitInFixSetExpression(c *InFixSetExpressionContext)

	// ExitRetrieveExpression is called when exiting the retrieveExpression production.
	ExitRetrieveExpression(c *RetrieveExpressionContext)

	// ExitTimingExpression is called when exiting the timingExpression production.
	ExitTimingExpression(c *TimingExpressionContext)

	// ExitQueryExpression is called when exiting the queryExpression production.
	ExitQueryExpression(c *QueryExpressionContext)

	// ExitNotExpression is called when exiting the notExpression production.
	ExitNotExpression(c *NotExpressionContext)

	// ExitBooleanExpression is called when exiting the booleanExpression production.
	ExitBooleanExpression(c *BooleanExpressionContext)

	// ExitOrExpression is called when exiting the orExpression production.
	ExitOrExpression(c *OrExpressionContext)

	// ExitCastExpression is called when exiting the castExpression production.
	ExitCastExpression(c *CastExpressionContext)

	// ExitAndExpression is called when exiting the andExpression production.
	ExitAndExpression(c *AndExpressionContext)

	// ExitBetweenExpression is called when exiting the betweenExpression production.
	ExitBetweenExpression(c *BetweenExpressionContext)

	// ExitMembershipExpression is called when exiting the membershipExpression production.
	ExitMembershipExpression(c *MembershipExpressionContext)

	// ExitDifferenceBetweenExpression is called when exiting the differenceBetweenExpression production.
	ExitDifferenceBetweenExpression(c *DifferenceBetweenExpressionContext)

	// ExitInequalityExpression is called when exiting the inequalityExpression production.
	ExitInequalityExpression(c *InequalityExpressionContext)

	// ExitEqualityExpression is called when exiting the equalityExpression production.
	ExitEqualityExpression(c *EqualityExpressionContext)

	// ExitExistenceExpression is called when exiting the existenceExpression production.
	ExitExistenceExpression(c *ExistenceExpressionContext)

	// ExitImpliesExpression is called when exiting the impliesExpression production.
	ExitImpliesExpression(c *ImpliesExpressionContext)

	// ExitTermExpression is called when exiting the termExpression production.
	ExitTermExpression(c *TermExpressionContext)

	// ExitTypeExpression is called when exiting the typeExpression production.
	ExitTypeExpression(c *TypeExpressionContext)

	// ExitDateTimePrecision is called when exiting the dateTimePrecision production.
	ExitDateTimePrecision(c *DateTimePrecisionContext)

	// ExitDateTimeComponent is called when exiting the dateTimeComponent production.
	ExitDateTimeComponent(c *DateTimeComponentContext)

	// ExitPluralDateTimePrecision is called when exiting the pluralDateTimePrecision production.
	ExitPluralDateTimePrecision(c *PluralDateTimePrecisionContext)

	// ExitAdditionExpressionTerm is called when exiting the additionExpressionTerm production.
	ExitAdditionExpressionTerm(c *AdditionExpressionTermContext)

	// ExitIndexedExpressionTerm is called when exiting the indexedExpressionTerm production.
	ExitIndexedExpressionTerm(c *IndexedExpressionTermContext)

	// ExitWidthExpressionTerm is called when exiting the widthExpressionTerm production.
	ExitWidthExpressionTerm(c *WidthExpressionTermContext)

	// ExitSetAggregateExpressionTerm is called when exiting the setAggregateExpressionTerm production.
	ExitSetAggregateExpressionTerm(c *SetAggregateExpressionTermContext)

	// ExitTimeUnitExpressionTerm is called when exiting the timeUnitExpressionTerm production.
	ExitTimeUnitExpressionTerm(c *TimeUnitExpressionTermContext)

	// ExitIfThenElseExpressionTerm is called when exiting the ifThenElseExpressionTerm production.
	ExitIfThenElseExpressionTerm(c *IfThenElseExpressionTermContext)

	// ExitTimeBoundaryExpressionTerm is called when exiting the timeBoundaryExpressionTerm production.
	ExitTimeBoundaryExpressionTerm(c *TimeBoundaryExpressionTermContext)

	// ExitElementExtractorExpressionTerm is called when exiting the elementExtractorExpressionTerm production.
	ExitElementExtractorExpressionTerm(c *ElementExtractorExpressionTermContext)

	// ExitConversionExpressionTerm is called when exiting the conversionExpressionTerm production.
	ExitConversionExpressionTerm(c *ConversionExpressionTermContext)

	// ExitTypeExtentExpressionTerm is called when exiting the typeExtentExpressionTerm production.
	ExitTypeExtentExpressionTerm(c *TypeExtentExpressionTermContext)

	// ExitPredecessorExpressionTerm is called when exiting the predecessorExpressionTerm production.
	ExitPredecessorExpressionTerm(c *PredecessorExpressionTermContext)

	// ExitPointExtractorExpressionTerm is called when exiting the pointExtractorExpressionTerm production.
	ExitPointExtractorExpressionTerm(c *PointExtractorExpressionTermContext)

	// ExitMultiplicationExpressionTerm is called when exiting the multiplicationExpressionTerm production.
	ExitMultiplicationExpressionTerm(c *MultiplicationExpressionTermContext)

	// ExitAggregateExpressionTerm is called when exiting the aggregateExpressionTerm production.
	ExitAggregateExpressionTerm(c *AggregateExpressionTermContext)

	// ExitDurationExpressionTerm is called when exiting the durationExpressionTerm production.
	ExitDurationExpressionTerm(c *DurationExpressionTermContext)

	// ExitDifferenceExpressionTerm is called when exiting the differenceExpressionTerm production.
	ExitDifferenceExpressionTerm(c *DifferenceExpressionTermContext)

	// ExitCaseExpressionTerm is called when exiting the caseExpressionTerm production.
	ExitCaseExpressionTerm(c *CaseExpressionTermContext)

	// ExitPowerExpressionTerm is called when exiting the powerExpressionTerm production.
	ExitPowerExpressionTerm(c *PowerExpressionTermContext)

	// ExitSuccessorExpressionTerm is called when exiting the successorExpressionTerm production.
	ExitSuccessorExpressionTerm(c *SuccessorExpressionTermContext)

	// ExitPolarityExpressionTerm is called when exiting the polarityExpressionTerm production.
	ExitPolarityExpressionTerm(c *PolarityExpressionTermContext)

	// ExitTermExpressionTerm is called when exiting the termExpressionTerm production.
	ExitTermExpressionTerm(c *TermExpressionTermContext)

	// ExitInvocationExpressionTerm is called when exiting the invocationExpressionTerm production.
	ExitInvocationExpressionTerm(c *InvocationExpressionTermContext)

	// ExitCaseExpressionItem is called when exiting the caseExpressionItem production.
	ExitCaseExpressionItem(c *CaseExpressionItemContext)

	// ExitDateTimePrecisionSpecifier is called when exiting the dateTimePrecisionSpecifier production.
	ExitDateTimePrecisionSpecifier(c *DateTimePrecisionSpecifierContext)

	// ExitRelativeQualifier is called when exiting the relativeQualifier production.
	ExitRelativeQualifier(c *RelativeQualifierContext)

	// ExitOffsetRelativeQualifier is called when exiting the offsetRelativeQualifier production.
	ExitOffsetRelativeQualifier(c *OffsetRelativeQualifierContext)

	// ExitExclusiveRelativeQualifier is called when exiting the exclusiveRelativeQualifier production.
	ExitExclusiveRelativeQualifier(c *ExclusiveRelativeQualifierContext)

	// ExitQuantityOffset is called when exiting the quantityOffset production.
	ExitQuantityOffset(c *QuantityOffsetContext)

	// ExitTemporalRelationship is called when exiting the temporalRelationship production.
	ExitTemporalRelationship(c *TemporalRelationshipContext)

	// ExitConcurrentWithIntervalOperatorPhrase is called when exiting the concurrentWithIntervalOperatorPhrase production.
	ExitConcurrentWithIntervalOperatorPhrase(c *ConcurrentWithIntervalOperatorPhraseContext)

	// ExitIncludesIntervalOperatorPhrase is called when exiting the includesIntervalOperatorPhrase production.
	ExitIncludesIntervalOperatorPhrase(c *IncludesIntervalOperatorPhraseContext)

	// ExitIncludedInIntervalOperatorPhrase is called when exiting the includedInIntervalOperatorPhrase production.
	ExitIncludedInIntervalOperatorPhrase(c *IncludedInIntervalOperatorPhraseContext)

	// ExitBeforeOrAfterIntervalOperatorPhrase is called when exiting the beforeOrAfterIntervalOperatorPhrase production.
	ExitBeforeOrAfterIntervalOperatorPhrase(c *BeforeOrAfterIntervalOperatorPhraseContext)

	// ExitWithinIntervalOperatorPhrase is called when exiting the withinIntervalOperatorPhrase production.
	ExitWithinIntervalOperatorPhrase(c *WithinIntervalOperatorPhraseContext)

	// ExitMeetsIntervalOperatorPhrase is called when exiting the meetsIntervalOperatorPhrase production.
	ExitMeetsIntervalOperatorPhrase(c *MeetsIntervalOperatorPhraseContext)

	// ExitOverlapsIntervalOperatorPhrase is called when exiting the overlapsIntervalOperatorPhrase production.
	ExitOverlapsIntervalOperatorPhrase(c *OverlapsIntervalOperatorPhraseContext)

	// ExitStartsIntervalOperatorPhrase is called when exiting the startsIntervalOperatorPhrase production.
	ExitStartsIntervalOperatorPhrase(c *StartsIntervalOperatorPhraseContext)

	// ExitEndsIntervalOperatorPhrase is called when exiting the endsIntervalOperatorPhrase production.
	ExitEndsIntervalOperatorPhrase(c *EndsIntervalOperatorPhraseContext)

	// ExitInvocationTerm is called when exiting the invocationTerm production.
	ExitInvocationTerm(c *InvocationTermContext)

	// ExitLiteralTerm is called when exiting the literalTerm production.
	ExitLiteralTerm(c *LiteralTermContext)

	// ExitExternalConstantTerm is called when exiting the externalConstantTerm production.
	ExitExternalConstantTerm(c *ExternalConstantTermContext)

	// ExitIntervalSelectorTerm is called when exiting the intervalSelectorTerm production.
	ExitIntervalSelectorTerm(c *IntervalSelectorTermContext)

	// ExitTupleSelectorTerm is called when exiting the tupleSelectorTerm production.
	ExitTupleSelectorTerm(c *TupleSelectorTermContext)

	// ExitInstanceSelectorTerm is called when exiting the instanceSelectorTerm production.
	ExitInstanceSelectorTerm(c *InstanceSelectorTermContext)

	// ExitListSelectorTerm is called when exiting the listSelectorTerm production.
	ExitListSelectorTerm(c *ListSelectorTermContext)

	// ExitCodeSelectorTerm is called when exiting the codeSelectorTerm production.
	ExitCodeSelectorTerm(c *CodeSelectorTermContext)

	// ExitConceptSelectorTerm is called when exiting the conceptSelectorTerm production.
	ExitConceptSelectorTerm(c *ConceptSelectorTermContext)

	// ExitParenthesizedTerm is called when exiting the parenthesizedTerm production.
	ExitParenthesizedTerm(c *ParenthesizedTermContext)

	// ExitQualifiedMemberInvocation is called when exiting the qualifiedMemberInvocation production.
	ExitQualifiedMemberInvocation(c *QualifiedMemberInvocationContext)

	// ExitQualifiedFunctionInvocation is called when exiting the qualifiedFunctionInvocation production.
	ExitQualifiedFunctionInvocation(c *QualifiedFunctionInvocationContext)

	// ExitQualifiedFunction is called when exiting the qualifiedFunction production.
	ExitQualifiedFunction(c *QualifiedFunctionContext)

	// ExitMemberInvocation is called when exiting the memberInvocation production.
	ExitMemberInvocation(c *MemberInvocationContext)

	// ExitFunctionInvocation is called when exiting the functionInvocation production.
	ExitFunctionInvocation(c *FunctionInvocationContext)

	// ExitThisInvocation is called when exiting the thisInvocation production.
	ExitThisInvocation(c *ThisInvocationContext)

	// ExitIndexInvocation is called when exiting the indexInvocation production.
	ExitIndexInvocation(c *IndexInvocationContext)

	// ExitTotalInvocation is called when exiting the totalInvocation production.
	ExitTotalInvocation(c *TotalInvocationContext)

	// ExitFunction is called when exiting the function production.
	ExitFunction(c *FunctionContext)

	// ExitRatio is called when exiting the ratio production.
	ExitRatio(c *RatioContext)

	// ExitBooleanLiteral is called when exiting the booleanLiteral production.
	ExitBooleanLiteral(c *BooleanLiteralContext)

	// ExitNullLiteral is called when exiting the nullLiteral production.
	ExitNullLiteral(c *NullLiteralContext)

	// ExitStringLiteral is called when exiting the stringLiteral production.
	ExitStringLiteral(c *StringLiteralContext)

	// ExitNumberLiteral is called when exiting the numberLiteral production.
	ExitNumberLiteral(c *NumberLiteralContext)

	// ExitLongNumberLiteral is called when exiting the longNumberLiteral production.
	ExitLongNumberLiteral(c *LongNumberLiteralContext)

	// ExitDateTimeLiteral is called when exiting the dateTimeLiteral production.
	ExitDateTimeLiteral(c *DateTimeLiteralContext)

	// ExitDateLiteral is called when exiting the dateLiteral production.
	ExitDateLiteral(c *DateLiteralContext)

	// ExitTimeLiteral is called when exiting the timeLiteral production.
	ExitTimeLiteral(c *TimeLiteralContext)

	// ExitQuantityLiteral is called when exiting the quantityLiteral production.
	ExitQuantityLiteral(c *QuantityLiteralContext)

	// ExitRatioLiteral is called when exiting the ratioLiteral production.
	ExitRatioLiteral(c *RatioLiteralContext)

	// ExitExternalConstant is called when exiting the externalConstant production.
	ExitExternalConstant(c *ExternalConstantContext)

	// ExitIntervalSelector is called when exiting the intervalSelector production.
	ExitIntervalSelector(c *IntervalSelectorContext)

	// ExitTupleSelector is called when exiting the tupleSelector production.
	ExitTupleSelector(c *TupleSelectorContext)

	// ExitTupleElementSelector is called when exiting the tupleElementSelector production.
	ExitTupleElementSelector(c *TupleElementSelectorContext)

	// ExitInstanceSelector is called when exiting the instanceSelector production.
	ExitInstanceSelector(c *InstanceSelectorContext)

	// ExitInstanceElementSelector is called when exiting the instanceElementSelector production.
	ExitInstanceElementSelector(c *InstanceElementSelectorContext)

	// ExitListSelector is called when exiting the listSelector production.
	ExitListSelector(c *ListSelectorContext)

	// ExitDisplayClause is called when exiting the displayClause production.
	ExitDisplayClause(c *DisplayClauseContext)

	// ExitCodeSelector is called when exiting the codeSelector production.
	ExitCodeSelector(c *CodeSelectorContext)

	// ExitConceptSelector is called when exiting the conceptSelector production.
	ExitConceptSelector(c *ConceptSelectorContext)

	// ExitKeyword is called when exiting the keyword production.
	ExitKeyword(c *KeywordContext)

	// ExitReservedWord is called when exiting the reservedWord production.
	ExitReservedWord(c *ReservedWordContext)

	// ExitKeywordIdentifier is called when exiting the keywordIdentifier production.
	ExitKeywordIdentifier(c *KeywordIdentifierContext)

	// ExitObsoleteIdentifier is called when exiting the obsoleteIdentifier production.
	ExitObsoleteIdentifier(c *ObsoleteIdentifierContext)

	// ExitFunctionIdentifier is called when exiting the functionIdentifier production.
	ExitFunctionIdentifier(c *FunctionIdentifierContext)

	// ExitTypeNameIdentifier is called when exiting the typeNameIdentifier production.
	ExitTypeNameIdentifier(c *TypeNameIdentifierContext)

	// ExitReferentialIdentifier is called when exiting the referentialIdentifier production.
	ExitReferentialIdentifier(c *ReferentialIdentifierContext)

	// ExitReferentialOrTypeNameIdentifier is called when exiting the referentialOrTypeNameIdentifier production.
	ExitReferentialOrTypeNameIdentifier(c *ReferentialOrTypeNameIdentifierContext)

	// ExitIdentifierOrFunctionIdentifier is called when exiting the identifierOrFunctionIdentifier production.
	ExitIdentifierOrFunctionIdentifier(c *IdentifierOrFunctionIdentifierContext)

	// ExitIdentifier is called when exiting the identifier production.
	ExitIdentifier(c *IdentifierContext)

	// ExitParamList is called when exiting the paramList production.
	ExitParamList(c *ParamListContext)

	// ExitQuantity is called when exiting the quantity production.
	ExitQuantity(c *QuantityContext)

	// ExitUnit is called when exiting the unit production.
	ExitUnit(c *UnitContext)
}
