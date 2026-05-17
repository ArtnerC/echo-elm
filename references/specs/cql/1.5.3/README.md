# CQL and ELM 1.5.3 implementation notes

## Source of truth

- Specification: <https://cql.hl7.org/>
- Translation semantics: <https://cql.hl7.org/06-translationsemantics.html>
- ELM physical representation: <https://cql.hl7.org/07-physicalrepresentation.html>
- Reference implementations: <https://cql.hl7.org/10-c-referenceimplementations.html>

The current published Clinical Quality Language (CQL) specification is **1.5.3**. It is Release 1, mixed normative/trial-use, based on FHIR R4, and is the stable baseline for this workspace. CQL is an ANSI normative standard; trial-use content is explicitly identified by the specification.

## What a translator must produce

CQL is the author-facing language; ELM is the machine-facing logical model. Every valid CQL statement has a semantically equivalent ELM representation. A translator should preserve:

- library identity and version,
- `using`, `include`, terminology, parameter, expression, and function declarations,
- source/locator annotations when requested,
- inferred result types when requested,
- translator messages with stable severity and source spans,
- exact ELM semantics for nulls, intervals, lists, quantities, terminology, query constructs, and data-model retrieves.

## Core CQL-to-ELM mapping

| CQL element | ELM representation |
| --- | --- |
| `library` | `Library` |
| `using` | `UsingDef` |
| `include` | `IncludeDef` |
| `codesystem` | `CodeSystemDef` |
| `valueset` | `ValueSetDef` |
| `parameter` | `ParameterDef` |
| `define` | `ExpressionDef` |
| `function` | `FunctionDef` |
| named type | `NamedTypeSpecifier` |
| interval type | `IntervalTypeSpecifier` |
| list type | `ListTypeSpecifier` |
| tuple type | `TupleTypeSpecifier` |
| choice type | `ChoiceTypeSpecifier` |
| `null`, booleans, numerics, quantities, ratios, strings, date/time | corresponding ELM literal/operator forms |
| query | `Query`, `AliasedQuerySource`, `LetClause`, `With`, `Without`, `ReturnClause`, `AggregateClause`, sort elements |

The translation-semantics chapter defines operator-by-operator mappings. Do not invent mappings where the specification already defines a direct equivalent.

## Physical ELM

ELM physical representation is defined by XSDs, and the XSDs are the source of truth:

- `expression.xsd`
- `clinicalexpression.xsd`
- `library.xsd`

Supported media types:

- `text/cql`
- `text/cql-identifier`
- `text/cql-expression`
- `application/elm+xml`
- `application/elm+json`

ELM namespace is `urn:hl7-org:elm:r1`; CQL namespace is `urn:hl7-org:cql:r1`.

JSON ELM follows the XML structure: XML elements and attributes become JSON attributes; type-disambiguation uses a `type` attribute; XML namespaces in type names use `{namespace}Name`; mixed XML content is not supported.

## ModelInfo requirements

A translator needs resolvers for:

- library names and versions,
- data model names and versions,
- model structures and type information,
- contexts and context relationships,
- primary code paths and retrievability,
- patient birthdate expression support for patient-age functions,
- profile labels/identifiers when using FHIR profile-informed or derived model info.

The CQL specification describes expected behavior; concrete modelinfo artifacts are implementation artifacts, not normative parts of CQL itself.

## Translation options that affect semantics

Treat these as part of compatibility:

- `DisableListPromotion`
- `DisableListDemotion`
- `DisableListTraversal`
- `DisableMethodInvocation`
- `RequireFromKeyword`
- interval promotion/demotion support when enabled by tooling
- signature level for function overloads
- compatibility level

If ELM is reused in a runtime, the runtime must know the options used to produce it.

## Implementation acceptance criteria

A compatible translator should:

1. Parse CQL 1.5.3 grammar and produce ELM XML and JSON.
2. Resolve libraries, models, value sets, code systems, codes, units, and functions deterministically.
3. Report diagnostics with severity, message, source range, and cause.
4. Preserve semantic-equivalence between CQL and ELM.
5. Support FHIR R4 modelinfo and FHIRHelpers 4.0.1.
6. Support QDM 5.6 modelinfo when QDM measures are in scope.
7. Emit translator options and version metadata where artifacts are packaged.
8. Validate generated ELM against the ELM schema or generated model classes.

